package hunt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/policy"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

type Snapshot struct {
	Session      domain.Session
	Hypotheses   []domain.Hypothesis
	Observations []domain.Observation
}

type Plan struct {
	Hypothesis domain.Hypothesis
	Action     *domain.ActionProposal
	Complete   bool
	Reason     string
}

type Interpretation struct {
	Observation domain.Observation
	Hypothesis  domain.HypothesisState
	Progress    bool
	Complete    bool
	Reason      string
}

type Planner interface {
	Plan(context.Context, Snapshot) (Plan, error)
}
type Executor interface {
	Execute(context.Context, domain.ActionProposal, string) (runner.Result, error)
}
type Interpreter interface {
	Interpret(context.Context, Snapshot, domain.ActionProposal, runner.Result) (Interpretation, error)
}
type Recorder interface {
	RecordHypothesis(context.Context, domain.Hypothesis) error
	RecordAction(context.Context, domain.ActionProposal, domain.ActionState, string) error
	RecordDecision(context.Context, domain.Decision) error
	RecordObservation(context.Context, domain.Observation) error
}

type Config struct {
	ActionTimeout time.Duration
	MaxRetries    int
	MaxActions    int
	MaxNoProgress int
}

type Outcome struct {
	State      domain.SessionState
	Reason     string
	Actions    int
	NoProgress int
	Pending    *domain.ActionProposal
}

type Loop struct {
	Planner     Planner
	Policy      policy.Evaluator
	Executor    Executor
	Interpreter Interpreter
	Recorder    Recorder
	Config      Config
}

func (loop Loop) Run(ctx context.Context, snapshot Snapshot) (Outcome, error) {
	if loop.Planner == nil || loop.Policy == nil || loop.Executor == nil || loop.Interpreter == nil || loop.Recorder == nil {
		return Outcome{}, errors.New("hunt loop dependencies are incomplete")
	}
	if loop.Config.MaxActions <= 0 {
		loop.Config.MaxActions = 50
	}
	if loop.Config.MaxNoProgress <= 0 {
		loop.Config.MaxNoProgress = 5
	}
	if loop.Config.ActionTimeout <= 0 {
		loop.Config.ActionTimeout = 30 * time.Second
	}
	outcome := Outcome{State: domain.SessionRunning}
	seen := map[string]runner.Result{}
	for outcome.Actions < loop.Config.MaxActions {
		if err := ctx.Err(); err != nil {
			return Outcome{State: domain.SessionCancelled, Reason: err.Error(), Actions: outcome.Actions}, err
		}
		plan, err := loop.Planner.Plan(ctx, snapshot)
		if err != nil {
			return Outcome{State: domain.SessionFailed, Reason: "planning failed", Actions: outcome.Actions}, err
		}
		if plan.Complete || plan.Action == nil {
			return Outcome{State: domain.SessionCompleted, Reason: plan.Reason, Actions: outcome.Actions, NoProgress: outcome.NoProgress}, nil
		}
		if plan.Hypothesis.State == "" {
			plan.Hypothesis.State = domain.HypothesisProposed
		}
		if err := loop.Recorder.RecordHypothesis(ctx, plan.Hypothesis); err != nil {
			return outcome, err
		}
		if err := domain.ValidateHypothesisTransition(plan.Hypothesis.State, domain.HypothesisTesting); err != nil {
			return outcome, err
		}
		plan.Hypothesis.State = domain.HypothesisTesting
		if err := loop.Recorder.RecordHypothesis(ctx, plan.Hypothesis); err != nil {
			return outcome, err
		}
		proposal := *plan.Action
		if err := loop.Recorder.RecordAction(ctx, proposal, domain.ActionProposed, ""); err != nil {
			return outcome, err
		}
		decision, err := loop.Policy.Evaluate(ctx, proposal, policy.Context{Scope: snapshot.Session.Targets, Constraints: snapshot.Session.Constraints})
		if err != nil {
			return Outcome{State: domain.SessionFailed, Reason: "policy failed", Actions: outcome.Actions}, err
		}
		if err := loop.Recorder.RecordDecision(ctx, decision); err != nil {
			return outcome, err
		}
		switch decision.Decision {
		case domain.PolicyDeny:
			_ = loop.Recorder.RecordAction(ctx, proposal, domain.ActionDenied, "policy denied")
			plan.Hypothesis.State = domain.HypothesisBlocked
			_ = loop.Recorder.RecordHypothesis(ctx, plan.Hypothesis)
			outcome.NoProgress++
			if outcome.NoProgress >= loop.Config.MaxNoProgress {
				return Outcome{State: domain.SessionBlocked, Reason: "no policy-allowed progress remains", Actions: outcome.Actions, NoProgress: outcome.NoProgress}, nil
			}
			continue
		case domain.PolicyAsk:
			return Outcome{State: domain.SessionNeedsInput, Reason: "operator approval required", Actions: outcome.Actions, Pending: &proposal}, nil
		case domain.PolicyAllow:
		default:
			return Outcome{State: domain.SessionFailed, Reason: "invalid policy decision", Actions: outcome.Actions}, fmt.Errorf("invalid policy decision %q", decision.Decision)
		}
		key := idempotencyKey(proposal)
		result, exists := seen[key]
		if !exists {
			_ = loop.Recorder.RecordAction(ctx, proposal, domain.ActionApproved, "")
			_ = loop.Recorder.RecordAction(ctx, proposal, domain.ActionRunning, "")
			var actionErr error
			for attempt := 0; attempt <= loop.Config.MaxRetries; attempt++ {
				actionCtx, cancel := context.WithTimeout(ctx, loop.Config.ActionTimeout)
				result, actionErr = loop.Executor.Execute(actionCtx, proposal, key)
				cancel()
				if actionErr == nil || result.Uncertain || result.Cancelled || result.TimedOut {
					break
				}
			}
			outcome.Actions++
			if actionErr != nil {
				_ = loop.Recorder.RecordAction(ctx, proposal, domain.ActionFailed, actionErr.Error())
				outcome.NoProgress++
				if outcome.NoProgress >= loop.Config.MaxNoProgress {
					return Outcome{State: domain.SessionBlocked, Reason: "non-progress limit reached", Actions: outcome.Actions, NoProgress: outcome.NoProgress}, nil
				}
				continue
			}
			seen[key] = result
			state := domain.ActionSucceeded
			switch {
			case result.Uncertain:
				state = domain.ActionUncertain
			case result.Cancelled:
				state = domain.ActionCancelled
			case result.TimedOut:
				state = domain.ActionTimedOut
			case result.ExitCode != 0:
				state = domain.ActionFailed
			}
			_ = loop.Recorder.RecordAction(ctx, proposal, state, "")
			if state == domain.ActionUncertain {
				return Outcome{State: domain.SessionNeedsInput, Reason: "action outcome is uncertain", Actions: outcome.Actions}, nil
			}
		}
		interpretation, err := loop.Interpreter.Interpret(ctx, snapshot, proposal, result)
		if err != nil {
			return Outcome{State: domain.SessionFailed, Reason: "interpretation failed", Actions: outcome.Actions}, err
		}
		if interpretation.Observation.ID.Validate() == nil {
			if err := loop.Recorder.RecordObservation(ctx, interpretation.Observation); err != nil {
				return outcome, err
			}
			snapshot.Observations = append(snapshot.Observations, interpretation.Observation)
		}
		if interpretation.Hypothesis != "" {
			if err := domain.ValidateHypothesisTransition(plan.Hypothesis.State, interpretation.Hypothesis); err != nil {
				return outcome, err
			}
			plan.Hypothesis.State = interpretation.Hypothesis
			if err := loop.Recorder.RecordHypothesis(ctx, plan.Hypothesis); err != nil {
				return outcome, err
			}
		}
		if interpretation.Progress {
			outcome.NoProgress = 0
		} else {
			outcome.NoProgress++
		}
		if interpretation.Complete {
			return Outcome{State: domain.SessionCompleted, Reason: interpretation.Reason, Actions: outcome.Actions, NoProgress: outcome.NoProgress}, nil
		}
		if outcome.NoProgress >= loop.Config.MaxNoProgress {
			return Outcome{State: domain.SessionBlocked, Reason: "non-progress limit reached", Actions: outcome.Actions, NoProgress: outcome.NoProgress}, nil
		}
	}
	return Outcome{State: domain.SessionCompleted, Reason: "action budget exhausted", Actions: outcome.Actions, NoProgress: outcome.NoProgress}, nil
}

func idempotencyKey(proposal domain.ActionProposal) string { return "action:" + string(proposal.ID) }
