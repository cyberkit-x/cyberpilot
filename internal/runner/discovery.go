package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const ProbeLabel = "io.cyberpilot.probe=true"

var ErrNoProvider = errors.New("no usable Docker or Podman provider found; install Docker Desktop/Engine or rootless Podman and ensure its CLI connection is available")

type ProviderSummary struct {
	Provider   string `json:"provider"`
	Executable string `json:"executable"`
	Connection string `json:"connection"`
	Endpoint   string `json:"endpoint"`
	Rootless   bool   `json:"rootless"`
	Version    string `json:"version"`
}

type ProcessResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type ProcessExecutor interface {
	LookPath(string) (string, error)
	Run(context.Context, string, []string, []string) (ProcessResult, error)
}

type OSProcessExecutor struct{}

func (OSProcessExecutor) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (OSProcessExecutor) Run(ctx context.Context, executable string, args, environment []string) (ProcessResult, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = environment
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	result := ProcessResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, fmt.Errorf("%s failed with exit code %d: %s", filepathBase(executable), result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	return result, err
}

type Discovery struct {
	Executor ProcessExecutor
	Timeout  time.Duration
}

func (d Discovery) Discover(ctx context.Context) ([]ProviderSummary, error) {
	if d.Executor == nil {
		d.Executor = OSProcessExecutor{}
	}
	if d.Timeout <= 0 {
		d.Timeout = 10 * time.Second
	}
	var summaries []ProviderSummary
	for _, provider := range []string{"docker", "podman"} {
		executable, err := d.Executor.LookPath(provider)
		if err != nil {
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, d.Timeout)
		result, err := d.Executor.Run(probeCtx, executable, []string{"info", "--format", "json"}, sanitizedEnvironment())
		cancel()
		if err != nil {
			continue
		}
		summary, err := parseProviderInfo(provider, executable, result.Stdout)
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}
	if len(summaries) == 0 {
		return nil, ErrNoProvider
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Provider < summaries[j].Provider })
	return summaries, nil
}

func SelectProvider(summaries []ProviderSummary, selected string) (ProviderSummary, error) {
	if len(summaries) == 0 {
		return ProviderSummary{}, ErrNoProvider
	}
	if selected == "" && len(summaries) > 1 {
		return ProviderSummary{}, errors.New("both Docker and Podman are usable; explicitly select one provider")
	}
	if selected == "" {
		return summaries[0], nil
	}
	for _, summary := range summaries {
		if summary.Provider == selected {
			return summary, nil
		}
	}
	return ProviderSummary{}, fmt.Errorf("selected provider %q is not usable", selected)
}

func parseProviderInfo(provider, executable, data string) (ProviderSummary, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return ProviderSummary{}, fmt.Errorf("parse %s info: %w", provider, err)
	}
	summary := ProviderSummary{Provider: provider, Executable: executable, Connection: "default"}
	if provider == "docker" {
		summary.Endpoint = os.Getenv("DOCKER_HOST")
		if summary.Endpoint == "" {
			summary.Endpoint = "default-context"
		}
		summary.Version = stringValue(raw, "ServerVersion")
		if security, ok := raw["SecurityOptions"].([]any); ok {
			for _, option := range security {
				if strings.Contains(strings.ToLower(fmt.Sprint(option)), "rootless") {
					summary.Rootless = true
				}
			}
		}
	} else {
		host, _ := raw["host"].(map[string]any)
		summary.Endpoint = stringValue(host, "remoteSocket", "path")
		summary.Version = stringValue(raw, "version", "Version")
		if security, ok := host["security"].(map[string]any); ok {
			summary.Rootless, _ = security["rootless"].(bool)
		}
		if summary.Endpoint == "" {
			summary.Endpoint = "default-connection"
		}
	}
	return summary, nil
}

func stringValue(value map[string]any, keys ...string) string {
	var current any = value
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	if current == nil {
		return ""
	}
	return fmt.Sprint(current)
}

func sanitizedEnvironment() []string {
	allowed := []string{"PATH", "HOME", "USERPROFILE", "DOCKER_HOST", "DOCKER_CONTEXT", "PODMAN_CONNECTIONS_CONF", "CONTAINER_HOST", "XDG_RUNTIME_DIR"}
	result := make([]string, 0, len(allowed))
	for _, name := range allowed {
		if value, ok := os.LookupEnv(name); ok {
			result = append(result, name+"="+value)
		}
	}
	return result
}

func filepathBase(path string) string {
	if index := strings.LastIndexAny(path, `/\\`); index >= 0 {
		return path[index+1:]
	}
	return path
}

// ProbeLifecycle verifies the complete disposable sandbox lifecycle. The image
// must already exist locally; initialization never silently pulls software.
func ProbeLifecycle(ctx context.Context, executor ProcessExecutor, provider ProviderSummary, image string) error {
	if strings.TrimSpace(image) == "" {
		return errors.New("probe image is required")
	}
	if executor == nil {
		executor = OSProcessExecutor{}
	}
	name := fmt.Sprintf("cyberpilot-probe-%d", time.Now().UTC().UnixNano())
	environment := sanitizedEnvironment()
	run := func(args ...string) error {
		_, err := executor.Run(ctx, provider.Executable, args, environment)
		return err
	}
	if err := run("image", "inspect", image); err != nil {
		return fmt.Errorf("probe image %q is unavailable locally: %w", image, err)
	}
	created := false
	defer func() {
		if created {
			_, _ = executor.Run(context.Background(), provider.Executable, []string{"rm", "-f", name}, environment)
		}
	}()
	// The sandbox image entrypoint owns the container lifecycle. Keep the probe
	// on the same entrypoint contract used by real sessions so init validates
	// the artifact users will actually run.
	if err := run("create", "--name", name, "--label", ProbeLabel, image, "hold"); err != nil {
		return fmt.Errorf("create probe sandbox: %w", err)
	}
	created = true
	if err := run("start", name); err != nil {
		return fmt.Errorf("start probe sandbox: %w", err)
	}
	if err := run("exec", name, "sh", "-c", "printf cyberpilot-ready"); err != nil {
		return fmt.Errorf("execute in probe sandbox: %w", err)
	}
	if err := run("stop", "--time", "2", name); err != nil {
		return fmt.Errorf("stop probe sandbox: %w", err)
	}
	if err := run("rm", name); err != nil {
		return fmt.Errorf("remove probe sandbox: %w", err)
	}
	created = false
	return nil
}
