package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
	"github.com/cyberkit-x/cyberpilot/internal/service"
)

type ExecClient interface {
	Call(context.Context, string, any, any) error
	FollowEvents(context.Context, domain.ID, uint64, time.Duration, func(rpc.EventMessage) error) error
}

type ExecCommand struct {
	Input        io.Reader
	Output       io.Writer
	Error        io.Writer
	Client       ExecClient
	PollInterval time.Duration
}

type ExecResult struct {
	SessionID        domain.ID           `json:"session_id"`
	State            domain.SessionState `json:"state"`
	VerifiedFindings int                 `json:"verified_findings"`
	Reason           string              `json:"reason,omitempty"`
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

func (c ExecCommand) Run(ctx context.Context, args []string) error {
	options, err := ParseExecOptions(args, c.Input)
	if err != nil {
		return &ExitError{Code: 4, Err: err}
	}
	targets := extractTargets(options.Prompt)
	if len(targets) == 0 {
		return &ExitError{Code: 4, Err: errors.New("prompt must contain at least one explicit http:// or https:// target")}
	}
	input := service.CreateSessionInput{Objective: options.Prompt, Targets: targets, Goals: []string{"authorized Web/API vulnerability assessment"}}
	var session domain.Session
	if err := c.Client.Call(ctx, "session.create", input, &session); err != nil {
		return &ExitError{Code: 3, Err: err}
	}
	if options.Detach {
		return c.writeResult(ExecResult{SessionID: session.ID, State: session.State}, options.JSON)
	}
	fmt.Fprintf(c.Error, "Session %s created\n", session.ID)
	followCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	final := ExecResult{SessionID: session.ID, State: session.State}
	err = c.Client.FollowEvents(followCtx, session.ID, 0, c.PollInterval, func(message rpc.EventMessage) error {
		fmt.Fprintf(c.Error, "[%d] %s\n", message.Cursor, message.Event.Type)
		if message.Event.Type == "session.state-changed" {
			var change domain.SessionStateChangedPayload
			if json.Unmarshal(message.Event.Payload, &change) == nil {
				final.State, final.Reason = change.To, change.Reason
				if terminalState(change.To) {
					cancel()
				}
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return &ExitError{Code: 3, Err: err}
	}
	if final.State == domain.SessionCreated {
		return &ExitError{Code: 2, Err: fmt.Errorf("session %s remains pending", session.ID)}
	}
	var findings []json.RawMessage
	if err := c.Client.Call(ctx, "session.records", map[string]any{"id": session.ID, "kind": "finding"}, &findings); err != nil {
		return &ExitError{Code: 3, Err: err}
	}
	final.VerifiedFindings = len(findings)
	if err := c.writeResult(final, options.JSON); err != nil {
		return err
	}
	switch final.State {
	case domain.SessionNeedsInput, domain.SessionBlocked:
		return &ExitError{Code: 2, Err: errors.New("session requires operator input")}
	case domain.SessionFailed:
		return &ExitError{Code: 3, Err: errors.New("session failed")}
	default:
		if final.VerifiedFindings > 0 {
			return &ExitError{Code: 1, Err: errors.New("verified findings reported")}
		}
		return nil
	}
}

func (c ExecCommand) writeResult(result ExecResult, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(c.Output).Encode(result)
	}
	_, err := fmt.Fprintf(c.Output, "Session: %s\nState: %s\nVerified findings: %d\n", result.SessionID, result.State, result.VerifiedFindings)
	return err
}

func terminalState(state domain.SessionState) bool {
	switch state {
	case domain.SessionCompleted, domain.SessionFailed, domain.SessionCancelled, domain.SessionNeedsInput, domain.SessionBlocked:
		return true
	}
	return false
}

func extractTargets(prompt string) []string {
	seen := map[string]bool{}
	var result []string
	for _, field := range strings.Fields(prompt) {
		value := strings.Trim(field, ".,;()[]{}<>\"'")
		if (strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")) && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
