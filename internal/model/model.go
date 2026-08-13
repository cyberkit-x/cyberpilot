package model

import (
	"context"
	"encoding/json"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type CapabilityReport struct {
	Model            string `json:"model"`
	ToolCalling      bool   `json:"tool_calling"`
	StructuredOutput bool   `json:"structured_output"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TurnRequest struct {
	SessionID  domain.ID       `json:"session_id"`
	Messages   []Message       `json:"messages"`
	Tools      []Tool          `json:"tools"`
	MaxTokens  int             `json:"max_tokens"`
	Stream     bool            `json:"stream,omitempty"`
	RepairFrom json.RawMessage `json:"repair_from,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

type Usage struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Cost         float64 `json:"cost,omitempty"`
}

type TurnResult struct {
	Text         string                   `json:"text,omitempty"`
	Proposals    []domain.ActionProposal  `json:"proposals,omitempty"`
	Findings     []domain.FindingProposal `json:"findings,omitempty"`
	Complete     bool                     `json:"complete,omitempty"`
	Reason       string                   `json:"reason,omitempty"`
	Usage        Usage                    `json:"usage"`
	FinishReason string                   `json:"finish_reason"`
}

type Provider interface {
	Probe(context.Context) (CapabilityReport, error)
	Turn(context.Context, TurnRequest) (TurnResult, error)
}
