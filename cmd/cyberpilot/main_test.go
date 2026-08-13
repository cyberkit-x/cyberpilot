//go:build !windows

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/cli"
	"github.com/cyberkit-x/cyberpilot/internal/configuration"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/runtime"
)

func TestRealDaemonDetachedExecAndRestart(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cpe2e-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HOME", root)
	paths, err := platform.ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	config := configuration.Config{Version: configuration.SchemaVersion, Model: configuration.ModelProfile{Provider: "openai-compatible", Model: "fixture", BaseURL: "http://127.0.0.1:1/v1", CredentialRef: "env:CYBERPILOT_TEST_KEY"}, Runner: configuration.RunnerProfile{Provider: "docker", Connection: "default"}}
	if err := (configuration.FileStore{Path: paths.ConfigFile}).Save(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	daemon, err := runtime.NewDaemon(paths)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- daemon.Serve(ctx) }()
	client, err := runtime.EnsureClient(context.Background(), paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	command := cli.ExecCommand{Input: &bytes.Buffer{}, Output: &stdout, Error: &bytes.Buffer{}, Client: client}
	if err := command.Run(context.Background(), []string{"--json", "--detach", "Assess http://127.0.0.1:18080/objects/1"}); err != nil {
		t.Fatal(err)
	}
	client.Close()
	cancel()
	daemon.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("daemon did not stop")
	}
	restarted, err := runtime.NewDaemon(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	restartCtx, stop := context.WithCancel(context.Background())
	defer stop()
	go restarted.Serve(restartCtx)
	client, err = runtime.EnsureClient(context.Background(), paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var sessions []domain.Session
	if err := client.Call(context.Background(), "session.list", nil, &sessions); err != nil || len(sessions) != 1 {
		t.Fatalf("sessions=%v err=%v stdout=%q", sessions, err, stdout.String())
	}
}

func TestUninitializedAndInvalidScope(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HOME", root)
	if err := run(context.Background(), nil); err == nil {
		t.Fatal("uninitialized TUI started")
	}
}
