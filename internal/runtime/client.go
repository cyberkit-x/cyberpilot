package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
)

type Starter func(context.Context, platform.Paths) error

func StartProcess(_ context.Context, _ platform.Paths) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executable, "__daemon")
	command.Stdin, command.Stdout, command.Stderr = nil, nil, nil
	if err := command.Start(); err != nil {
		return fmt.Errorf("start CyberPilot daemon: %w", err)
	}
	return command.Process.Release()
}

func EnsureClient(ctx context.Context, paths platform.Paths, starter Starter) (*rpc.Client, error) {
	if starter == nil {
		starter = StartProcess
	}
	client, err := dialReady(ctx, paths)
	if err == nil {
		return client, nil
	}
	if err := starter(ctx, paths); err != nil {
		return nil, err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	var last error
	for {
		client, last = dialReady(ctx, paths)
		if last == nil {
			return client, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("daemon readiness: %w (last error: %v)", ctx.Err(), last)
		case <-ticker.C:
		}
	}
}

func dialReady(ctx context.Context, paths platform.Paths) (*rpc.Client, error) {
	token, err := ReadToken(paths)
	if err != nil {
		return nil, err
	}
	client, err := rpc.Dial(ctx, platform.LocalTransport(), paths.Endpoint, token)
	if err != nil {
		return nil, err
	}
	var ready struct {
		Ready    bool `json:"ready"`
		Protocol int  `json:"protocol"`
	}
	if err := client.Call(ctx, "system.ready", nil, &ready); err != nil {
		_ = client.Close()
		return nil, err
	}
	if !ready.Ready || ready.Protocol != rpc.ProtocolVersion {
		_ = client.Close()
		return nil, errors.New("daemon protocol negotiation failed")
	}
	return client, nil
}
