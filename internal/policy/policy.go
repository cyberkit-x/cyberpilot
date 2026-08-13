package policy

import (
	"context"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Context struct {
	Scope          []string          `json:"scope"`
	NonInteractive bool              `json:"non_interactive"`
	Constraints    []string          `json:"constraints,omitempty"`
	PriorApprovals []domain.Approval `json:"prior_approvals,omitempty"`
}

type Evaluator interface {
	Evaluate(context.Context, domain.ActionProposal, Context) (domain.Decision, error)
}
