package oci

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

func TestProviderConformance(t *testing.T) {
	provider := os.Getenv("CYBERPILOT_CONFORMANCE_PROVIDER")
	image := os.Getenv("CYBERPILOT_CONFORMANCE_IMAGE")
	if provider == "" || image == "" {
		t.Skip("set conformance provider and image")
	}
	if provider != "docker" && provider != "podman" {
		t.Fatal("unsupported conformance provider")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adapter := New(provider, provider, "default")
	if err := adapter.Probe(ctx); err != nil {
		t.Fatal(err)
	}
	session := domain.MustNewID()
	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(workspace, 0o777); err != nil {
		t.Fatal(err)
	}
	spec := runner.SandboxSpec{SessionID: session, Image: image, Workspace: workspace, MemoryBytes: 256 << 20, ProcessLimit: 64, NetworkProfile: "none"}
	if err := adapter.Create(ctx, spec); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = adapter.Stop(context.Background(), session)
		_ = adapter.Remove(context.Background(), session)
	})
	if err := adapter.Start(ctx, session); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	result, err := adapter.Exec(ctx, session, runner.Command{Executable: "sh", Arguments: []string{"-c", "printf conformance > /workspace/result && cat /workspace/result"}, Directory: "/workspace", Timeout: 10 * time.Second}, &stdout, &bytes.Buffer{})
	if err != nil || result.ExitCode != 0 || stdout.String() != "conformance" {
		t.Fatalf("result=%#v stdout=%q err=%v", result, stdout.String(), err)
	}
	if err := adapter.Stop(ctx, session); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Inspect(ctx, session); err != nil {
		t.Fatal("sandbox cannot be recovered after stop")
	}
	if err := adapter.Remove(ctx, session); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(workspace, "result")); err != nil || strings.TrimSpace(string(data)) != "conformance" {
		t.Fatalf("workspace data=%q err=%v", data, err)
	}
}
