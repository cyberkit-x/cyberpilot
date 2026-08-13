package domain

import "testing"

func TestStateTransitions(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
		check func() error
	}{
		{"session starts", true, func() error { return ValidateSessionTransition(SessionCreated, SessionRunning) }},
		{"session resumes", true, func() error { return ValidateSessionTransition(SessionNeedsInput, SessionRunning) }},
		{"terminal session stays terminal", true, func() error { return ValidateSessionTransition(SessionCompleted, SessionCompleted) }},
		{"terminal session cannot restart", false, func() error { return ValidateSessionTransition(SessionCompleted, SessionRunning) }},
		{"hypothesis tests", true, func() error { return ValidateHypothesisTransition(HypothesisProposed, HypothesisTesting) }},
		{"hypothesis supports", true, func() error { return ValidateHypothesisTransition(HypothesisTesting, HypothesisSupported) }},
		{"supported hypothesis cannot retest", false, func() error { return ValidateHypothesisTransition(HypothesisSupported, HypothesisTesting) }},
		{"action approved", true, func() error { return ValidateActionTransition(ActionProposed, ActionApproved) }},
		{"action succeeds", true, func() error { return ValidateActionTransition(ActionRunning, ActionSucceeded) }},
		{"denied action cannot run", false, func() error { return ValidateActionTransition(ActionDenied, ActionRunning) }},
		{"approval restricted", true, func() error { return ValidateApprovalTransition(ApprovalPending, ApprovalAllowed) }},
		{"expired approval cannot allow", false, func() error { return ValidateApprovalTransition(ApprovalExpired, ApprovalAllowed) }},
		{"lead verified", true, func() error { return ValidateFindingTransition(FindingLead, FindingVerified) }},
		{"rejected finding cannot verify", false, func() error { return ValidateFindingTransition(FindingRejected, FindingVerified) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.check()
			if test.valid && err != nil {
				t.Fatalf("expected valid transition: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected invalid transition")
			}
		})
	}
}
