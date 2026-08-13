package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/platform"
)

type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	token  string
	mu     sync.Mutex
}

func Dial(ctx context.Context, transport platform.Transport, endpoint, token string) (*Client, error) {
	conn, err := transport.Dial(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, reader: bufio.NewReader(conn), token: token}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Call(ctx context.Context, method string, input, output any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request := Request{Version: ProtocolVersion, ID: string(domain.MustNewID()), Token: c.token, Method: method, Payload: payload}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetDeadline(deadline)
		defer func() { _ = c.conn.SetDeadline(time.Time{}) }()
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return err
	}
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	var response Response
	if err := json.Unmarshal(line, &response); err != nil {
		return err
	}
	if response.ID != request.ID || response.Version != ProtocolVersion {
		return fmt.Errorf("RPC response does not match request")
	}
	if response.Error != nil {
		return response.Error
	}
	if output == nil {
		return nil
	}
	return json.Unmarshal(response.Result, output)
}

// FollowEvents delivers events after a durable cursor. Callers can reconnect
// and invoke it again with the last emitted cursor without duplicating events.
func (c *Client) FollowEvents(ctx context.Context, sessionID domain.ID, after uint64, interval time.Duration, emit func(EventMessage) error) error {
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		var events []domain.Event
		if err := c.Call(ctx, "session.events", map[string]any{"id": sessionID, "after": after}, &events); err != nil {
			return err
		}
		for _, event := range events {
			message := EventMessage{Version: ProtocolVersion, Cursor: event.Sequence, Event: event}
			if err := emit(message); err != nil {
				return err
			}
			after = event.Sequence
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
