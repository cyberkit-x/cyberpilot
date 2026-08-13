package policy

import "github.com/cyberkit-x/cyberpilot/internal/domain"

type Branch struct {
	Action   domain.ActionProposal
	Decision domain.Decision
}
type NonInteractiveResult struct {
	Allowed []Branch
	Pending []Branch
	Denied  []Branch
	State   domain.SessionState
}

func EvaluateNonInteractive(branches []Branch) NonInteractiveResult {
	result := NonInteractiveResult{State: domain.SessionRunning}
	for _, branch := range branches {
		switch branch.Decision.Decision {
		case domain.PolicyAllow:
			result.Allowed = append(result.Allowed, branch)
		case domain.PolicyAsk:
			result.Pending = append(result.Pending, branch)
		case domain.PolicyDeny:
			result.Denied = append(result.Denied, branch)
		}
	}
	if len(result.Allowed) == 0 {
		if len(result.Pending) > 0 {
			result.State = domain.SessionNeedsInput
		} else {
			result.State = domain.SessionBlocked
		}
	}
	return result
}
