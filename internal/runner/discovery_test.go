package runner

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeProcess struct {
	paths   map[string]string
	results map[string]ProcessResult
	calls   [][]string
	failAt  string
}

func (f *fakeProcess) LookPath(name string) (string, error) {
	if path := f.paths[name]; path != "" {
		return path, nil
	}
	return "", errors.New("not found")
}
func (f *fakeProcess) Run(_ context.Context, executable string, args, environment []string) (ProcessResult, error) {
	call := append([]string{executable}, args...)
	f.calls = append(f.calls, call)
	key := strings.Join(call, " ")
	if strings.Contains(key, f.failAt) && f.failAt != "" {
		return ProcessResult{ExitCode: 1}, errors.New("injected failure")
	}
	return f.results[key], nil
}

func TestDiscoveryAndExplicitSelection(t *testing.T) {
	fake := &fakeProcess{paths: map[string]string{"docker": "/bin/docker", "podman": "/bin/podman"}, results: map[string]ProcessResult{
		"/bin/docker info --format json": {Stdout: `{"ServerVersion":"28.0","SecurityOptions":["name=rootless"]}`},
		"/bin/podman info --format json": {Stdout: `{"host":{"remoteSocket":{"path":"unix:///run/user/1000/podman.sock"},"security":{"rootless":true}},"version":{"Version":"5.5"}}`},
	}}
	summaries, err := (Discovery{Executor: fake}).Discover(context.Background())
	if err != nil || len(summaries) != 2 {
		t.Fatalf("summaries=%#v err=%v", summaries, err)
	}
	if _, err := SelectProvider(summaries, ""); err == nil {
		t.Fatal("expected explicit selection error")
	}
	selected, err := SelectProvider(summaries, "podman")
	if err != nil || !selected.Rootless || !strings.Contains(selected.Endpoint, "podman.sock") {
		t.Fatalf("selected=%#v err=%v", selected, err)
	}
}

func TestDiscoveryNoProviderIsActionable(t *testing.T) {
	_, err := (Discovery{Executor: &fakeProcess{paths: map[string]string{}}}).Discover(context.Background())
	if !errors.Is(err, ErrNoProvider) || !strings.Contains(err.Error(), "install") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestProbeLifecycleAndFailureCleanup(t *testing.T) {
	fake := &fakeProcess{results: map[string]ProcessResult{}}
	provider := ProviderSummary{Provider: "docker", Executable: "/bin/docker"}
	if err := ProbeLifecycle(context.Background(), fake, provider, "probe:test"); err != nil {
		t.Fatal(err)
	}
	var operations []string
	for _, call := range fake.calls {
		operations = append(operations, call[1])
	}
	if want := []string{"image", "create", "start", "exec", "stop", "rm"}; !reflect.DeepEqual(operations, want) {
		t.Fatalf("operations=%v want=%v", operations, want)
	}

	failing := &fakeProcess{results: map[string]ProcessResult{}, failAt: " exec "}
	if err := ProbeLifecycle(context.Background(), failing, provider, "probe:test"); err == nil {
		t.Fatal("expected probe failure")
	}
	last := failing.calls[len(failing.calls)-1]
	if len(last) < 3 || last[1] != "rm" || last[2] != "-f" {
		t.Fatalf("expected forced cleanup, got %v", last)
	}
}
