//go:build !windows

package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
)

func testPaths(root string) platform.Paths {
	return platform.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), ConfigFile: filepath.Join(root, "config", "config.yaml"), DatabaseFile: filepath.Join(root, "data", "state.db"), ArtifactsDir: filepath.Join(root, "data", "artifacts"), RuntimeDir: filepath.Join(root, "data", "run"), LockFile: filepath.Join(root, "data", "run", "daemon.lock"), Endpoint: filepath.Join(root, "data", "run", "rpc.sock")}
}

func TestDaemonReadinessAndSingleOwner(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cpd-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	paths := testPaths(root)
	daemon, err := NewDaemon(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()
	if _, err := NewDaemon(paths); err == nil {
		t.Fatal("expected second daemon to fail")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- daemon.Serve(ctx) }()
	token, err := ReadToken(paths)
	if err != nil {
		t.Fatal(err)
	}
	callCtx, callCancel := context.WithTimeout(ctx, time.Second)
	defer callCancel()
	client, err := rpc.Dial(callCtx, platform.LocalTransport(), paths.Endpoint, token)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var ready map[string]any
	if err := client.Call(callCtx, "system.ready", nil, &ready); err != nil {
		t.Fatal(err)
	}
	if value, ok := ready["ready"].(bool); !ok || !value {
		t.Fatalf("ready = %#v", ready)
	}
	cancel()
	_ = client.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("daemon did not stop")
	}
}
