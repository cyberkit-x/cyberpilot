package policy

import (
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestNonInteractiveRetainsPendingAndProgressesAllowed(t *testing.T) {
	result := EvaluateNonInteractive([]Branch{{Decision: domain.Decision{Decision: domain.PolicyAllow}}, {Decision: domain.Decision{Decision: domain.PolicyAsk}}, {Decision: domain.Decision{Decision: domain.PolicyDeny}}})
	if result.State != domain.SessionRunning || len(result.Allowed) != 1 || len(result.Pending) != 1 || len(result.Denied) != 1 {
		t.Fatalf("result=%#v", result)
	}
}
func TestNonInteractiveAllInputOrDenied(t *testing.T) {
	if got := EvaluateNonInteractive([]Branch{{Decision: domain.Decision{Decision: domain.PolicyAsk}}}); got.State != domain.SessionNeedsInput {
		t.Fatalf("got=%#v", got)
	}
	if got := EvaluateNonInteractive([]Branch{{Decision: domain.Decision{Decision: domain.PolicyDeny}}}); got.State != domain.SessionBlocked {
		t.Fatalf("got=%#v", got)
	}
}
