package policy

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type EvaluatorImpl struct {
	NonInteractive    bool
	MaxRequests       int
	AllowCapabilities map[string]bool
}

func (e EvaluatorImpl) Evaluate(_ context.Context, action domain.ActionProposal, input Context) (domain.Decision, error) {
	decision := domain.Decision{ID: domain.MustNewID(), ActionID: action.ID, Decision: domain.PolicyAllow, DecidedAt: time.Now().UTC()}
	if action.ID.Validate() != nil {
		return decision, errors.New("invalid action id")
	}
	if !inScope(action.Target, input.Scope) {
		decision.Decision = domain.PolicyDeny
		decision.Basis = []string{"target is outside confirmed scope"}
		return decision, nil
	}
	if action.Capability == "" || e.AllowCapabilities != nil && !e.AllowCapabilities[action.Capability] {
		decision.Decision = domain.PolicyDeny
		decision.Basis = []string{"capability is unavailable or not allowed"}
		return decision, nil
	}
	risk := action.Risk
	if risk.UsesCredentials || risk.ReadsSensitive || risk.ChangesState || risk.AffectsAvailability {
		decision.Decision = domain.PolicyAsk
		decision.Basis = []string{"action requests sensitive or state-changing side effects"}
	}
	if input.NonInteractive && decision.Decision == domain.PolicyAsk {
		decision.Basis = append(decision.Basis, "non-interactive mode retains approval request")
	}
	if e.MaxRequests > 0 && risk.TrafficClass == "high" {
		decision.Decision = domain.PolicyAsk
		decision.Basis = append(decision.Basis, "traffic exceeds configured rate class")
	}
	return decision, nil
}

func inScope(candidate string, scopes []string) bool {
	target, err := ParseTarget(candidate)
	if err != nil {
		return false
	}
	for _, raw := range scopes {
		scope, err := ParseTarget(raw)
		if err != nil {
			continue
		}
		if target.Kind == TargetURL && scope.Kind == TargetURL {
			if target.Scheme == scope.Scheme && target.Host == scope.Host && (scope.Port == 0 || target.Port == scope.Port) && strings.HasPrefix(target.Canonical, scope.Canonical) {
				return true
			}
		}
		if target.Host == scope.Host && (scope.Port == 0 || target.Port == scope.Port) {
			return true
		}
		if scope.Kind == TargetWildcard && strings.HasSuffix(target.Host, "."+scope.Host) {
			return true
		}
		if scope.Kind == TargetCIDR && target.Kind == TargetIP {
			if address, err := netip.ParseAddr(target.Host); err == nil && scope.Prefix.Contains(address) {
				return true
			}
		}
	}
	return false
}

func resolvedIPAllowed(host string, ip net.IP, scope string) bool {
	target, err := ParseTarget(host)
	if err != nil {
		return false
	}
	resolved := netip.MustParseAddr(ip.String())
	allowed, err := ParseTarget(scope)
	if err != nil {
		return false
	}
	if allowed.Kind == TargetCIDR {
		return allowed.Prefix.Contains(resolved)
	}
	return target.Host == allowed.Host
}
func redirectAllowed(original, targetURL string, scopes []string) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.User != nil {
		return false
	}
	return inScope(original, scopes) && inScope(targetURL, scopes)
}
