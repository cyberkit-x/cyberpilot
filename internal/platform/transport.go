package platform

import (
	"context"
	"net"
)

type Transport interface {
	Listen(string) (net.Listener, error)
	Dial(context.Context, string) (net.Conn, error)
}

func LocalTransport() Transport { return localTransport{} }
