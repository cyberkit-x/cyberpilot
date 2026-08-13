package policy

import (
	"errors"
	"fmt"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type ApprovalRecord struct {
	Approval  domain.Approval
	Action    domain.ActionProposal
	Decision  domain.Decision
	CreatedAt time.Time
	RevokedAt *time.Time
}

func RestrictApproval(action domain.ActionProposal, requested domain.Limits, approved domain.Limits) (domain.Limits, error) {
	if requested.MaxRequests > 0 && (approved.MaxRequests <= 0 || approved.MaxRequests > requested.MaxRequests) {
		return domain.Limits{}, errors.New("approval cannot broaden request count")
	}
	if requested.MaxDuration > 0 && (approved.MaxDuration <= 0 || approved.MaxDuration > requested.MaxDuration) {
		return domain.Limits{}, errors.New("approval cannot broaden duration")
	}
	if len(requested.Targets) > 0 {
		if len(approved.Targets) == 0 {
			return domain.Limits{}, errors.New("approval must retain a target restriction")
		}
		for _, target := range approved.Targets {
			found := false
			for _, allowed := range requested.Targets {
				if target == allowed {
					found = true
				}
			}
			if !found {
				return domain.Limits{}, fmt.Errorf("approval target %q is broader than request", target)
			}
		}
	}
	return approved, nil
}
func (r ApprovalRecord) Active(now time.Time) bool {
	if r.RevokedAt != nil || r.Approval.State != domain.ApprovalAllowed {
		return false
	}
	return r.Approval.ExpiresAt == nil || now.Before(*r.Approval.ExpiresAt)
}
func (r *ApprovalRecord) Revoke(now time.Time) error {
	if !r.Active(now) {
		return errors.New("approval is not active")
	}
	r.Approval.State = domain.ApprovalRevoked
	r.RevokedAt = &now
	return nil
}
