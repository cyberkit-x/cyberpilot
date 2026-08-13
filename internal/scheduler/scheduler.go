package scheduler

import (
	"context"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Budget struct {
	Deadline        time.Time `json:"deadline,omitempty"`
	MaxActions      int       `json:"max_actions"`
	MaxInputTokens  int64     `json:"max_input_tokens"`
	MaxOutputTokens int64     `json:"max_output_tokens"`
	MaxCost         float64   `json:"max_cost"`
	MaxNoProgress   int       `json:"max_no_progress"`
}

type Scheduler interface {
	Start(context.Context, domain.ID) error
	Cancel(context.Context, domain.ID) error
}
