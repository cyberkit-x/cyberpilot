//go:build !windows

package runtime

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
	"github.com/cyberkit-x/cyberpilot/internal/service"
)

func TestEnsureClientStartsAndWaitsForDaemon(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cpc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	paths := testPaths(root)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := 0
	var daemon *Daemon
	client, err := EnsureClient(ctx, paths, func(context.Context, platform.Paths) error {
		started++
		var err error
		daemon, err = NewDaemon(paths)
		if err != nil {
			return err
		}
		go daemon.Serve(ctx)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	defer daemon.Close()
	if started != 1 {
		t.Fatalf("started=%d", started)
	}
	second, err := EnsureClient(ctx, paths, func(context.Context, platform.Paths) error {
		t.Fatal("started second daemon")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = second.Close()
}

func TestCursorEventFollowingResumesWithoutDuplicates(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "cpe-")
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
	daemonCtx, stopDaemon := context.WithCancel(context.Background())
	defer stopDaemon()
	go daemon.Serve(daemonCtx)
	clientCtx, cancelClient := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelClient()
	client, err := EnsureClient(clientCtx, paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var session domain.Session
	if err := client.Call(clientCtx, "session.create", service.CreateSessionInput{Objective: "Inspect local API authorization", Targets: []string{"http://127.0.0.1"}, Goals: []string{"authorization"}}, &session); err != nil {
		t.Fatal(err)
	}

	followCtx, stopFollow := context.WithCancel(clientCtx)
	var first rpc.EventMessage
	err = client.FollowEvents(followCtx, session.ID, 0, time.Millisecond, func(message rpc.EventMessage) error {
		first = message
		stopFollow()
		return nil
	})
	if !errors.Is(err, context.Canceled) || first.Cursor != 1 || first.Event.Type != "session.created" {
		t.Fatalf("first=%#v err=%v", first, err)
	}

	var updated domain.Session
	if err := client.Call(clientCtx, "session.instructions.update", map[string]any{"id": session.ID, "instructions": "focus on object access"}, &updated); err != nil {
		t.Fatal(err)
	}
	resumeCtx, stopResume := context.WithCancel(clientCtx)
	var resumed []rpc.EventMessage
	err = client.FollowEvents(resumeCtx, session.ID, first.Cursor, time.Millisecond, func(message rpc.EventMessage) error {
		resumed = append(resumed, message)
		stopResume()
		return nil
	})
	if !errors.Is(err, context.Canceled) || len(resumed) != 1 || resumed[0].Cursor != 2 || resumed[0].Event.Type != "session.instructions-updated" {
		t.Fatalf("resumed=%#v err=%v", resumed, err)
	}
}
