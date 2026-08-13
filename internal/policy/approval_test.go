package policy

import (
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestApprovalOnlyNarrowsLimits(t *testing.T) {
	requested := domain.Limits{MaxRequests: 10, MaxDuration: 60, Targets: []string{"https://api.example.com"}}
	action := domain.ActionProposal{}
	if _, err := RestrictApproval(action, requested, domain.Limits{MaxRequests: 11, MaxDuration: 60, Targets: requested.Targets}); err == nil {
		t.Fatal("broader request count approved")
	}
	if _, err := RestrictApproval(action, requested, domain.Limits{MaxRequests: 5, MaxDuration: 30, Targets: []string{"https://api.example.com"}}); err != nil {
		t.Fatal(err)
	}
}
func TestApprovalExpiryAndRevocation(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	record := ApprovalRecord{Approval: domain.Approval{State: domain.ApprovalAllowed, ExpiresAt: &expires}}
	if !record.Active(time.Now()) {
		t.Fatal("active approval not active")
	}
	if err := record.Revoke(time.Now()); err != nil || record.Active(time.Now()) {
		t.Fatalf("record=%#v err=%v", record, err)
	}
	expired := time.Now().Add(-time.Hour)
	record = ApprovalRecord{Approval: domain.Approval{State: domain.ApprovalAllowed, ExpiresAt: &expired}}
	if record.Active(time.Now()) {
		t.Fatal("expired approval active")
	}
}
