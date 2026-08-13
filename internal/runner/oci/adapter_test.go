package oci

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

type call struct {
	binary    string
	args, env []string
}
type fakeCommander struct {
	calls   []call
	inspect string
	exit    int
}

func (f *fakeCommander) Run(_ context.Context, b string, args, env []string, _ io.Reader, stdout, stderr io.Writer) (int, error) {
	f.calls = append(f.calls, call{b, append([]string(nil), args...), append([]string(nil), env...)})
	if len(args) > 0 && contains(args, "inspect") {
		_, _ = io.WriteString(stdout, f.inspect)
	}
	return f.exit, nil
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func TestAdapterArgumentArraysLabelsAndOwnership(t *testing.T) {
	id := domain.MustNewID()
	commander := &fakeCommander{inspect: `{"io.cyberpilot.managed":"true","io.cyberpilot.session":"` + string(id) + `"}`}
	adapter := New("docker", "docker", "desktop-linux")
	adapter.Commander = commander
	spec := runner.SandboxSpec{SessionID: id, Image: "image@sha256:abc", Workspace: "/tmp/work", MemoryBytes: 1024, ProcessLimit: 32}
	if err := adapter.Create(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(commander.calls[0].args, " ")
	for _, required := range []string{"--context desktop-linux", "--label io.cyberpilot.managed=true", "--cap-drop ALL", "--read-only", "--network none", "image@sha256:abc"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing %q in %s", required, joined)
		}
	}
	if err := adapter.Start(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Exec(context.Background(), id, runner.Command{Executable: "curl", Arguments: []string{"--version"}, Directory: "/workspace", Environment: map[string]string{"SAFE": "value", "bad-name": "no"}}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	joined = strings.Join(commander.calls[len(commander.calls)-1].args, " ")
	if strings.Contains(joined, "bad-name") || !strings.Contains(joined, "SAFE=value") {
		t.Fatalf("unsafe env args=%s", joined)
	}
}
func TestAdapterRejectsForeignResource(t *testing.T) {
	id := domain.MustNewID()
	commander := &fakeCommander{inspect: `{"io.cyberpilot.managed":"false"}`}
	adapter := New("podman", "podman", "rootless")
	adapter.Commander = commander
	if err := adapter.Remove(context.Background(), id); err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("err=%v", err)
	}
	for _, call := range commander.calls {
		if contains(call.args, "rm") {
			t.Fatal("removed foreign resource")
		}
	}
}
