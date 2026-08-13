package rpc

import (
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Handler func(context.Context, json.RawMessage) (any, *domain.PublicError)

type Server struct {
	token    string
	handlers map[string]Handler
	mu       sync.RWMutex
}

func NewServer(token string) *Server { return &Server{token: token, handlers: map[string]Handler{}} }

func (s *Server) Handle(method string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	go func() { <-ctx.Done(); _ = listener.Close() }()
	var connections sync.WaitGroup
	defer connections.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		connections.Add(1)
		go func() { defer connections.Done(); defer func() { _ = conn.Close() }(); s.serveConn(ctx, conn) }()
	}
}

func (s *Server) serveConn(ctx context.Context, stream io.ReadWriter) {
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(stream)
	for scanner.Scan() {
		var request Request
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			_ = encoder.Encode(Response{Version: ProtocolVersion, Error: rpcError(domain.ErrInvalidInput, "invalid RPC request")})
			continue
		}
		if err := encoder.Encode(s.dispatch(ctx, request)); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(ctx context.Context, request Request) Response {
	response := Response{Version: ProtocolVersion, ID: request.ID}
	if request.Version != ProtocolVersion {
		response.Error = rpcError(domain.ErrInvalidInput, fmt.Sprintf("unsupported RPC protocol %d", request.Version))
		return response
	}
	if subtle.ConstantTimeCompare([]byte(request.Token), []byte(s.token)) != 1 {
		response.Error = rpcError(domain.ErrAuthentication, "local RPC authentication failed")
		return response
	}
	s.mu.RLock()
	handler := s.handlers[request.Method]
	s.mu.RUnlock()
	if handler == nil {
		response.Error = rpcError(domain.ErrNotFound, "RPC method not found")
		return response
	}
	result, publicErr := handler(ctx, request.Payload)
	if publicErr != nil {
		response.Error = publicErr
		return response
	}
	data, err := json.Marshal(result)
	if err != nil {
		response.Error = rpcError(domain.ErrInternal, "cannot encode RPC result")
		return response
	}
	response.Result = data
	return response
}

func rpcError(code domain.ErrorCode, message string) *domain.PublicError {
	return &domain.PublicError{Code: code, Message: message}
}
