package rpc

import (
	"encoding/json"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

const ProtocolVersion = 1

type Request struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Token   string          `json:"token"`
	Method  string          `json:"method"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Version int                 `json:"version"`
	ID      string              `json:"id"`
	Result  json.RawMessage     `json:"result,omitempty"`
	Error   *domain.PublicError `json:"error,omitempty"`
}

type EventMessage struct {
	Version int          `json:"version"`
	Cursor  uint64       `json:"cursor"`
	Event   domain.Event `json:"event"`
}
