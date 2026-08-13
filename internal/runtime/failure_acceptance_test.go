package runtime_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/evidence"
	"github.com/cyberkit-x/cyberpilot/internal/model"
	modelprovider "github.com/cyberkit-x/cyberpilot/internal/model/openai"
	"github.com/cyberkit-x/cyberpilot/internal/policy"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

func TestFailureAcceptanceBoundaries(t *testing.T) {
	t.Run("model outage", func(t *testing.T) {
		_, err := (modelprovider.Provider{BaseURL: "http://127.0.0.1:1", Model: "fixture"}).Probe(context.Background())
		if err == nil {
			t.Fatal("outage accepted")
		}
	})
	t.Run("runtime loss", func(t *testing.T) {
		if err := (&runner.Manager{}).Ensure(context.Background(), runner.SandboxSpec{SessionID: domain.MustNewID()}); err == nil {
			t.Fatal("host fallback occurred")
		}
	})
	t.Run("scope escape", func(t *testing.T) {
		proposal := domain.ActionProposal{ID: domain.MustNewID(), Target: "https://outside.invalid", Capability: "http.request"}
		decision, _ := (policy.EvaluatorImpl{AllowCapabilities: map[string]bool{"http.request": true}}).Evaluate(context.Background(), proposal, policy.Context{Scope: []string{"https://fixture.local"}})
		if decision.Decision != domain.PolicyDeny {
			t.Fatal("escape allowed")
		}
	})
	t.Run("credential leakage", func(t *testing.T) {
		if bytes.Contains(evidence.NewRedactor().Bytes([]byte("API key=secret")), []byte("secret")) {
			t.Fatal("secret leaked")
		}
	})
	t.Run("missing browser capability", func(t *testing.T) {
		proposal := domain.ActionProposal{ID: domain.MustNewID(), Target: "https://fixture.local", Capability: "browser.control"}
		decision, _ := (policy.EvaluatorImpl{AllowCapabilities: map[string]bool{"http.request": true}}).Evaluate(context.Background(), proposal, policy.Context{Scope: []string{"https://fixture.local"}})
		if decision.Decision != domain.PolicyDeny {
			t.Fatal("missing browser capability allowed")
		}
	})
	t.Run("timeout cancellation", func(t *testing.T) {
		fake := runner.NewFake()
		id := domain.MustNewID()
		_ = fake.Create(context.Background(), runner.SandboxSpec{SessionID: id})
		_ = fake.Start(context.Background(), id)
		fake.Behavior.Delay = time.Hour
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		result, err := fake.Exec(ctx, id, runner.Command{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
		if !errors.Is(err, context.DeadlineExceeded) || !result.TimedOut {
			t.Fatalf("result=%#v err=%v", result, err)
		}
	})
	_ = model.TurnRequest{}
}
