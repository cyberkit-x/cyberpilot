package evidence

import (
	"context"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"testing"
)

func completeFinding() domain.Finding {
	return domain.Finding{ID: domain.MustNewID(), SessionID: domain.MustNewID(), Title: "IDOR", State: domain.FindingLead, Target: "https://api.example.com/object/1", Prerequisites: []string{"two test identities"}, EvidenceIDs: []domain.ID{domain.MustNewID()}, ControlEvidence: []domain.ID{domain.MustNewID()}, Impact: "read another test identity object", Reproduction: []string{"request owner object", "repeat as control identity"}, Provenance: domain.Provenance{Tool: "http.request"}}
}
func TestBaselineFindingGate(t *testing.T) {
	finding := completeFinding()
	result, err := (BaselineValidator{}).Validate(context.Background(), finding)
	if err != nil || !result.Allowed || len(result.Reasons) != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
func TestIncompleteFindingRemainsLeadWithGaps(t *testing.T) {
	finding := domain.Finding{ID: domain.MustNewID(), SessionID: domain.MustNewID(), Title: "scanner match", State: domain.FindingLead}
	result, err := (BaselineValidator{}).Validate(context.Background(), finding)
	if err != nil || result.Allowed || len(result.Reasons) != 7 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	lead := RetainAsLead(finding, result)
	if lead.Reason == "" || lead.ID != finding.ID {
		t.Fatalf("lead=%#v", lead)
	}
}
