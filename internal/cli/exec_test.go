package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
	"github.com/cyberkit-x/cyberpilot/internal/service"
)

type execClientFake struct {
	session domain.Session
	final   domain.SessionState
	calls   int
}

func (f *execClientFake) Call(_ context.Context, method string, input, output any) error {
	f.calls++
	if method == "session.records" {
		*(output.(*[]json.RawMessage)) = nil
		return nil
	}
	if method != "session.create" {
		return errors.New("unexpected method: " + method)
	}
	create := input.(service.CreateSessionInput)
	if len(create.Targets) != 1 {
		return errors.New("target missing")
	}
	f.session = domain.Session{ID: domain.MustNewID(), Objective: create.Objective, Targets: create.Targets, Goals: create.Goals, State: domain.SessionCreated}
	*(output.(*domain.Session)) = f.session
	return nil
}
func (f *execClientFake) FollowEvents(ctx context.Context, _ domain.ID, _ uint64, _ time.Duration, emit func(rpc.EventMessage) error) error {
	created, _ := domain.NewEvent(f.session.ID, 1, "session.created", time.Now(), domain.SessionCreatedPayload{Session: f.session})
	if err := emit(rpc.EventMessage{Cursor: 1, Event: created}); err != nil {
		return err
	}
	changed, _ := domain.NewEvent(f.session.ID, 2, "session.state-changed", time.Now(), domain.SessionStateChangedPayload{From: domain.SessionRunning, To: f.final, Reason: "done"})
	if err := emit(rpc.EventMessage{Cursor: 2, Event: changed}); err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}
func TestExecExactlyOneJSONResultAndProgressOnStderr(t *testing.T) {
	client := &execClientFake{final: domain.SessionCompleted}
	var stdout, stderr bytes.Buffer
	command := ExecCommand{Input: &bytes.Buffer{}, Output: &stdout, Error: &stderr, Client: client, PollInterval: time.Millisecond}
	if err := command.Run(context.Background(), []string{"--json", "Assess https://fixture.local/api"}); err != nil {
		t.Fatal(err)
	}
	var result ExecResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.State != domain.SessionCompleted {
		t.Fatalf("stdout=%q result=%#v err=%v", stdout.String(), result, err)
	}
	if bytes.Count(stdout.Bytes(), []byte("\n")) != 1 || !bytes.Contains(stderr.Bytes(), []byte("session.state-changed")) {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
func TestExecDetachAndInvalidScope(t *testing.T) {
	client := &execClientFake{}
	var stdout bytes.Buffer
	command := ExecCommand{Input: &bytes.Buffer{}, Output: &stdout, Error: &bytes.Buffer{}, Client: client}
	if err := command.Run(context.Background(), []string{"--json", "--detach", "Assess https://fixture.local"}); err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 {
		t.Fatal("session not durably created")
	}
	err := command.Run(context.Background(), []string{"no explicit target"})
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 4 {
		t.Fatalf("err=%v", err)
	}
}
func TestExecBlockedExitCode(t *testing.T) {
	client := &execClientFake{final: domain.SessionNeedsInput}
	command := ExecCommand{Input: &bytes.Buffer{}, Output: &bytes.Buffer{}, Error: &bytes.Buffer{}, Client: client}
	err := command.Run(context.Background(), []string{"Assess https://fixture.local"})
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 2 {
		t.Fatalf("err=%v", err)
	}
}
