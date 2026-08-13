package evidence

import (
	"context"
	"errors"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Signal string

const (
	SignalHTTPStatus     Signal = "http-status"
	SignalScannerMatch   Signal = "scanner-match"
	SignalCodePattern    Signal = "code-pattern"
	SignalModelAssertion Signal = "model-assertion"
	SignalReproduction   Signal = "reproduction"
	SignalControl        Signal = "control"
	SignalImpact         Signal = "impact"
)

type PromotionProposal struct {
	Finding      domain.Finding `json:"finding"`
	Signals      []Signal       `json:"signals"`
	EvidenceOnly bool           `json:"evidence_only"`
}
type Verifier interface {
	Verify(context.Context, domain.Lead, []domain.Observation) (PromotionProposal, error)
}

func DecidePromotion(ctx context.Context, validator Validator, proposal PromotionProposal) (domain.Finding, PromotionResult, error) {
	if !proposal.EvidenceOnly {
		return proposal.Finding, PromotionResult{Reasons: []string{"promotion was not produced by an evidence-only verification turn"}}, nil
	}
	strong := map[Signal]bool{}
	for _, signal := range proposal.Signals {
		strong[signal] = true
	}
	if !strong[SignalReproduction] || !strong[SignalImpact] || !strong[SignalControl] {
		return proposal.Finding, PromotionResult{Reasons: []string{"reproduction, impact, and control signals are required"}}, nil
	}
	result, err := validator.Validate(ctx, proposal.Finding)
	if err != nil {
		return proposal.Finding, result, err
	}
	if result.Allowed {
		proposal.Finding.State = domain.FindingVerified
	} else {
		proposal.Finding.State = domain.FindingLead
	}
	return proposal.Finding, result, nil
}

var ErrVerificationUnavailable = errors.New("evidence verification unavailable")
