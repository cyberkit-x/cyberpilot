package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

const ownerLabel = "io.cyberpilot.managed=true"

type Commander interface {
	Run(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) (int, error)
}
type OSCommander struct{}

func (OSCommander) Run(ctx context.Context, binary string, args, environment []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	command := exec.CommandContext(ctx, binary, args...)
	command.Env = environment
	command.Stdin, command.Stdout, command.Stderr = stdin, stdout, stderr
	err := command.Run()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return -1, err
}

type Adapter struct {
	Provider, Binary, Connection string
	Commander                    Commander
}

func New(provider, binary, connection string) *Adapter {
	return &Adapter{Provider: provider, Binary: binary, Connection: connection, Commander: OSCommander{}}
}
func (a *Adapter) Probe(ctx context.Context) error {
	_, err := a.run(ctx, nil, io.Discard, io.Discard, "info", "--format", "json")
	return err
}
func (a *Adapter) Create(ctx context.Context, spec runner.SandboxSpec) error {
	if spec.SessionID.Validate() != nil {
		return errors.New("invalid session id")
	}
	args := []string{"create", "--name", name(spec.SessionID), "--label", ownerLabel, "--label", "io.cyberpilot.session=" + string(spec.SessionID), "--user", "65532:65532", "--cap-drop", "ALL", "--read-only", "--security-opt", "no-new-privileges", "--network", "none"}
	if spec.MemoryBytes > 0 {
		args = append(args, "--memory", strconv.FormatInt(spec.MemoryBytes, 10))
	}
	if spec.ProcessLimit > 0 {
		args = append(args, "--pids-limit", strconv.Itoa(spec.ProcessLimit))
	}
	if spec.Workspace != "" {
		args = append(args, "--mount", "type=bind,src="+spec.Workspace+",dst=/workspace")
	}
	args = append(args, spec.Image, "hold")
	_, err := a.run(ctx, nil, io.Discard, io.Discard, args...)
	return err
}
func (a *Adapter) Start(ctx context.Context, id domain.ID) error {
	return a.owned(ctx, id, func() error { _, err := a.run(ctx, nil, io.Discard, io.Discard, "start", name(id)); return err })
}
func (a *Adapter) Exec(ctx context.Context, id domain.ID, command runner.Command, stdout, stderr io.Writer) (runner.Result, error) {
	started := time.Now().UTC()
	if err := a.ensureOwned(ctx, id); err != nil {
		return runner.Result{}, err
	}
	args := []string{"exec", "--workdir", command.Directory}
	for key, value := range command.Environment {
		if safeEnvName(key) {
			args = append(args, "--env", key+"="+value)
		}
	}
	args = append(args, name(id), command.Executable)
	args = append(args, command.Arguments...)
	actionCtx := ctx
	cancel := func() {}
	if command.Timeout > 0 {
		actionCtx, cancel = context.WithTimeout(ctx, command.Timeout)
	}
	defer cancel()
	exit, err := a.run(actionCtx, nil, stdout, stderr, args...)
	result := runner.Result{ExitCode: exit, StartedAt: started, FinishedAt: time.Now().UTC()}
	if errors.Is(actionCtx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
	} else if errors.Is(actionCtx.Err(), context.Canceled) {
		result.Cancelled = true
	}
	return result, err
}
func (a *Adapter) Stop(ctx context.Context, id domain.ID) error {
	return a.owned(ctx, id, func() error {
		_, err := a.run(ctx, nil, io.Discard, io.Discard, "stop", "--time", "5", name(id))
		return err
	})
}
func (a *Adapter) Remove(ctx context.Context, id domain.ID) error {
	return a.owned(ctx, id, func() error { _, err := a.run(ctx, nil, io.Discard, io.Discard, "rm", name(id)); return err })
}
func (a *Adapter) Inspect(ctx context.Context, id domain.ID) (runner.SandboxSpec, error) {
	if err := a.ensureOwned(ctx, id); err != nil {
		return runner.SandboxSpec{}, err
	}
	var output strings.Builder
	if _, err := a.run(ctx, nil, &output, io.Discard, "inspect", name(id), "--format", "{{json .Config.Labels}}"); err != nil {
		return runner.SandboxSpec{}, err
	}
	return runner.SandboxSpec{SessionID: id}, nil
}
func (a *Adapter) owned(ctx context.Context, id domain.ID, fn func() error) error {
	if err := a.ensureOwned(ctx, id); err != nil {
		return err
	}
	return fn()
}
func (a *Adapter) ensureOwned(ctx context.Context, id domain.ID) error {
	var output strings.Builder
	if _, err := a.run(ctx, nil, &output, io.Discard, "inspect", name(id), "--format", "{{json .Config.Labels}}"); err != nil {
		return err
	}
	var labels map[string]string
	if json.Unmarshal([]byte(strings.TrimSpace(output.String())), &labels) != nil || labels["io.cyberpilot.managed"] != "true" || labels["io.cyberpilot.session"] != string(id) {
		return errors.New("refusing to operate on a resource not owned by CyberPilot")
	}
	return nil
}
func (a *Adapter) run(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, args ...string) (int, error) {
	if a.Commander == nil {
		a.Commander = OSCommander{}
	}
	prefix := []string{}
	if a.Provider == "docker" && a.Connection != "" && a.Connection != "default" {
		prefix = []string{"--context", a.Connection}
	}
	if a.Provider == "podman" && a.Connection != "" && a.Connection != "default" {
		prefix = []string{"--connection", a.Connection}
	}
	var diagnostic strings.Builder
	destination := stderr
	if destination == nil || destination == io.Discard {
		destination = &diagnostic
	} else {
		destination = io.MultiWriter(destination, &diagnostic)
	}
	exit, err := a.Commander.Run(ctx, a.Binary, append(prefix, args...), sanitizedEnv(), stdin, stdout, destination)
	if err != nil {
		return exit, err
	}
	if exit != 0 {
		detail := strings.TrimSpace(diagnostic.String())
		if len(detail) > 1024 {
			detail = detail[:1024]
		}
		if detail != "" {
			return exit, fmt.Errorf("%s command failed with exit code %d: %s", a.Provider, exit, detail)
		}
		return exit, fmt.Errorf("%s command failed with exit code %d", a.Provider, exit)
	}
	return exit, nil
}
func name(id domain.ID) string { return "cyberpilot-" + string(id) }
func safeEnvName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
func sanitizedEnv() []string {
	allowed := []string{"PATH", "HOME", "USERPROFILE", "DOCKER_HOST", "DOCKER_CONTEXT", "CONTAINER_HOST", "XDG_RUNTIME_DIR"}
	var result []string
	for _, key := range allowed {
		if value, ok := os.LookupEnv(key); ok {
			result = append(result, key+"="+value)
		}
	}
	return result
}
