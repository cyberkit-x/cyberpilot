//go:build !windows

package platform

import (
	"context"
	"fmt"
	"net"
	"os"
)

type localTransport struct{}

func (localTransport) Listen(endpoint string) (net.Listener, error) {
	// Darwin commonly limits sockaddr_un paths to 104 bytes; Linux typically
	// allows 108. Use the stricter bound so failure is deterministic.
	if len(endpoint) >= 104 {
		return nil, fmt.Errorf("local socket path is too long (%d bytes, max 103): %s", len(endpoint), endpoint)
	}
	if err := os.Remove(endpoint); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove stale local socket: %w", err)
	}
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", endpoint)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(endpoint, 0o600); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

func (localTransport) Dial(ctx context.Context, endpoint string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "unix", endpoint)
}
