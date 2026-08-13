package evidence

import (
	"context"
	"strings"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type BaselineValidator struct{}

func (BaselineValidator) Validate(_ context.Context, finding domain.Finding) (PromotionResult, error) {
	var reasons []string
	if strings.TrimSpace(finding.Target) == "" {
		reasons = append(reasons, "target is missing")
	}
	if len(finding.Prerequisites) == 0 {
		reasons = append(reasons, "prerequisites are missing")
	}
	if len(finding.EvidenceIDs) == 0 {
		reasons = append(reasons, "controllability and action-path evidence is missing")
	}
	if strings.TrimSpace(finding.Impact) == "" {
		reasons = append(reasons, "observed impact is missing")
	}
	if len(finding.Reproduction) == 0 {
		reasons = append(reasons, "reproduction steps are missing")
	}
	if finding.Provenance.Model == "" && finding.Provenance.SkillName == "" && finding.Provenance.Tool == "" {
		reasons = append(reasons, "evidence provenance is missing")
	}
	if len(finding.ControlEvidence) == 0 {
		reasons = append(reasons, "negative or control evidence is missing")
	}
	return PromotionResult{Allowed: len(reasons) == 0, Reasons: reasons}, nil
}
func RetainAsLead(finding domain.Finding, result PromotionResult) domain.Lead {
	return domain.Lead{ID: finding.ID, SessionID: finding.SessionID, Title: finding.Title, Reason: strings.Join(result.Reasons, "; "), EvidenceIDs: append([]domain.ID(nil), finding.EvidenceIDs...)}
}
