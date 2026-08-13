//go:build windows

package platform

import (
	"context"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

type localTransport struct{}

func (localTransport) Listen(endpoint string) (net.Listener, error) {
	return winio.ListenPipe(endpoint, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;OW)",
		MessageMode:        false,
		InputBufferSize:    64 * 1024,
		OutputBufferSize:   64 * 1024,
	})
}

func (localTransport) Dial(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}

var _ = time.Second
