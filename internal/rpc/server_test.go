package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestRPCProtocolAuthenticationAndErrors(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	server := NewServer("secret")
	server.Handle("echo", func(_ context.Context, payload json.RawMessage) (any, *domain.PublicError) {
		var value map[string]string
		if err := json.Unmarshal(payload, &value); err != nil {
			return nil, rpcError(domain.ErrInvalidInput, "bad payload")
		}
		return value, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go server.serveConn(ctx, serverConn)
	client := &Client{conn: clientConn, reader: bufio.NewReader(clientConn), token: "secret"}
	defer client.Close()
	callCtx, callCancel := context.WithTimeout(ctx, time.Second)
	defer callCancel()
	var output map[string]string
	if err := client.Call(callCtx, "echo", map[string]string{"hello": "world"}, &output); err != nil {
		t.Fatal(err)
	}
	if output["hello"] != "world" {
		t.Fatalf("output = %#v", output)
	}
	client.token = "wrong"
	if err := client.Call(callCtx, "echo", nil, &output); err == nil {
		t.Fatal("expected authentication error")
	}
}
