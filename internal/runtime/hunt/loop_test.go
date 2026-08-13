package hunt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/policy"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

type plannerFunc func(context.Context, Snapshot) (Plan, error)

func (f plannerFunc) Plan(c context.Context, s Snapshot) (Plan, error) { return f(c, s) }

type policyFunc func(context.Context, domain.ActionProposal, policy.Context) (domain.Decision, error)

func (f policyFunc) Evaluate(c context.Context, a domain.ActionProposal, p policy.Context) (domain.Decision, error) {
	return f(c, a, p)
}

type executorFunc func(context.Context, domain.ActionProposal, string) (runner.Result, error)

func (f executorFunc) Execute(c context.Context, a domain.ActionProposal, k string) (runner.Result, error) {
	return f(c, a, k)
}

type interpreterFunc func(context.Context, Snapshot, domain.ActionProposal, runner.Result) (Interpretation, error)

func (f interpreterFunc) Interpret(c context.Context, s Snapshot, a domain.ActionProposal, r runner.Result) (Interpretation, error) {
	return f(c, s, a, r)
}

type recorder struct{ states []domain.ActionState }

func (*recorder) RecordHypothesis(context.Context, domain.Hypothesis) error { return nil }
func (r *recorder) RecordAction(_ context.Context, _ domain.ActionProposal, s domain.ActionState, _ string) error {
	r.states = append(r.states, s)
	return nil
}
func (*recorder) RecordDecision(context.Context, domain.Decision) error       { return nil }
func (*recorder) RecordObservation(context.Context, domain.Observation) error { return nil }

func proposal() (domain.Hypothesis, domain.ActionProposal) {
	session, hypothesis, action := domain.MustNewID(), domain.MustNewID(), domain.MustNewID()
	h := domain.Hypothesis{ID: hypothesis, SessionID: session, Claim: "object access differs", State: domain.HypothesisProposed}
	a := domain.ActionProposal{ID: action, SessionID: session, HypothesisID: hypothesis, Target: "http://127.0.0.1", Purpose: "compare", Capability: "http.request", TimeoutSeconds: 1}
	return h, a
}

func TestLoopAgenticProgressAndCompletion(t *testing.T) {
	h, a := proposal()
	plans := 0
	executes := 0
	records := &recorder{}
	loop := Loop{Planner: plannerFunc(func(context.Context, Snapshot) (Plan, error) {
		plans++
		if plans > 1 {
			return Plan{Complete: true, Reason: "coverage complete"}, nil
		}
		return Plan{Hypothesis: h, Action: &a}, nil
	}), Policy: policyFunc(func(context.Context, domain.ActionProposal, policy.Context) (domain.Decision, error) {
		return domain.Decision{Decision: domain.PolicyAllow}, nil
	}), Executor: executorFunc(func(_ context.Context, _ domain.ActionProposal, key string) (runner.Result, error) {
		executes++
		if key == "" {
			t.Fatal("missing idempotency key")
		}
		return runner.Result{}, nil
	}), Interpreter: interpreterFunc(func(context.Context, Snapshot, domain.ActionProposal, runner.Result) (Interpretation, error) {
		return Interpretation{Hypothesis: domain.HypothesisSupported, Progress: true}, nil
	}), Recorder: records}
	outcome, err := loop.Run(context.Background(), Snapshot{Session: domain.Session{Targets: []string{"http://127.0.0.1"}}})
	if err != nil || outcome.State != domain.SessionCompleted || executes != 1 || plans != 2 {
		t.Fatalf("outcome=%#v err=%v executes=%d plans=%d", outcome, err, executes, plans)
	}
}

func TestLoopAskDenyRetryTimeoutAndUncertain(t *testing.T) {
	h, a := proposal()
	for _, test := range []struct {
		name     string
		decision domain.PolicyDecision
		result   runner.Result
		execErr  error
		want     domain.SessionState
		retries  int
	}{
		{name: "ask", decision: domain.PolicyAsk, want: domain.SessionNeedsInput},
		{name: "deny", decision: domain.PolicyDeny, want: domain.SessionBlocked},
		{name: "uncertain", decision: domain.PolicyAllow, result: runner.Result{Uncertain: true}, want: domain.SessionNeedsInput},
		{name: "timeout", decision: domain.PolicyAllow, result: runner.Result{TimedOut: true}, want: domain.SessionBlocked},
		{name: "retries", decision: domain.PolicyAllow, execErr: errors.New("temporary"), want: domain.SessionBlocked, retries: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			loop := Loop{Planner: plannerFunc(func(context.Context, Snapshot) (Plan, error) { return Plan{Hypothesis: h, Action: &a}, nil }), Policy: policyFunc(func(context.Context, domain.ActionProposal, policy.Context) (domain.Decision, error) {
				return domain.Decision{Decision: test.decision}, nil
			}), Executor: executorFunc(func(context.Context, domain.ActionProposal, string) (runner.Result, error) {
				calls++
				return test.result, test.execErr
			}), Interpreter: interpreterFunc(func(context.Context, Snapshot, domain.ActionProposal, runner.Result) (Interpretation, error) {
				return Interpretation{}, nil
			}), Recorder: &recorder{}, Config: Config{MaxNoProgress: 1, MaxActions: 2, MaxRetries: test.retries, ActionTimeout: time.Millisecond}}
			outcome, _ := loop.Run(context.Background(), Snapshot{})
			if outcome.State != test.want {
				t.Fatalf("outcome=%#v", outcome)
			}
			if test.execErr != nil && calls != test.retries+1 {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestLoopCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome, err := (Loop{Planner: plannerFunc(nil), Policy: policyFunc(nil), Executor: executorFunc(nil), Interpreter: interpreterFunc(nil), Recorder: &recorder{}}).Run(ctx, Snapshot{})
	if !errors.Is(err, context.Canceled) || outcome.State != domain.SessionCancelled {
		t.Fatalf("outcome=%#v err=%v", outcome, err)
	}
}
