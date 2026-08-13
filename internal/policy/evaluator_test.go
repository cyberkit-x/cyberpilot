package policy

import (
	"context"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"strings"
	"testing"
)

func action(target, capability string) domain.ActionProposal {
	return domain.ActionProposal{ID: domain.MustNewID(), SessionID: domain.MustNewID(), HypothesisID: domain.MustNewID(), Target: target, Capability: capability, TimeoutSeconds: 10}
}
func TestPolicyAllowAskDeny(t *testing.T) {
	e := EvaluatorImpl{AllowCapabilities: map[string]bool{"http.request": true}}
	allowed := action("http://api.example.com/users", "http.request")
	decision, _ := e.Evaluate(context.Background(), allowed, Context{Scope: []string{"https://api.example.com"}})
	if decision.Decision != domain.PolicyDeny {
		t.Fatalf("scheme mismatch should deny: %#v", decision)
	}
	allowed.Target = "https://api.example.com/users"
	decision, _ = e.Evaluate(context.Background(), allowed, Context{Scope: []string{"https://api.example.com"}})
	if decision.Decision != domain.PolicyAllow {
		t.Fatalf("allow=%#v", decision)
	}
	ask := allowed
	ask.Risk.UsesCredentials = true
	decision, _ = e.Evaluate(context.Background(), ask, Context{Scope: []string{"https://api.example.com"}, NonInteractive: true})
	if decision.Decision != domain.PolicyAsk || !strings.Contains(strings.Join(decision.Basis, " "), "approval") {
		t.Fatalf("ask=%#v", decision)
	}
	deny := allowed
	deny.Target = "https://outside.example.com"
	decision, _ = e.Evaluate(context.Background(), deny, Context{Scope: []string{"https://api.example.com"}})
	if decision.Decision != domain.PolicyDeny {
		t.Fatalf("deny=%#v", decision)
	}
}
func TestPolicyCapabilityAndResolvedDestination(t *testing.T) {
	e := EvaluatorImpl{AllowCapabilities: map[string]bool{"http.request": true}}
	decision, _ := e.Evaluate(context.Background(), action("https://api.example.com", "shell.exec"), Context{Scope: []string{"https://api.example.com"}})
	if decision.Decision != domain.PolicyDeny {
		t.Fatal("unknown capability allowed")
	}
	if !resolvedIPAllowed("api.example.com", []byte{192, 0, 2, 1}, "192.0.2.0/24") {
		t.Fatal("resolved IP denied incorrectly")
	}
	if redirectAllowed("https://api.example.com", "https://outside.example.com", []string{"https://api.example.com"}) {
		t.Fatal("out-of-scope redirect allowed")
	}
}
