package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	store "github.com/cyberkit-x/cyberpilot/internal/storage/sqlite"
)

func TestSessionLifecycleAndRecovery(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewSessionService(db)
	session, err := service.Create(context.Background(), CreateSessionInput{Objective: "Assess API authorization across two targets", Targets: []string{"one.test", "two.test"}, Goals: []string{"IDOR", "auth bypass"}})
	if err != nil {
		t.Fatal(err)
	}
	if session.Name == "" || len(session.Targets) != 2 || session.State != domain.SessionCreated {
		t.Fatalf("unexpected session %#v", session)
	}
	running, err := service.transition(context.Background(), session.ID, domain.SessionRunning, "started")
	if err != nil {
		t.Fatal(err)
	}
	if running.State != domain.SessionRunning {
		t.Fatal("session did not start")
	}
	if err := service.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	recovered, err := db.Session(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.State != domain.SessionNeedsInput || recovered.TerminalReason == "" {
		t.Fatalf("unexpected recovery %#v", recovered)
	}
	updated, err := service.UpdateInstructions(context.Background(), session.ID, "focus on object ownership")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Instructions != "focus on object ownership" {
		t.Fatalf("instructions = %q", updated.Instructions)
	}
	updated, err = service.UpdateScope(context.Background(), session.ID, []string{"api.example.test"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.ScopeConfirmed || len(updated.Targets) != 1 {
		t.Fatalf("scope = %#v", updated)
	}
	approval := domain.Approval{ID: domain.MustNewID(), ActionID: domain.MustNewID(), State: domain.ApprovalAllowed, Reason: "approved for one request"}
	if _, err := service.DecideApproval(context.Background(), session.ID, approval); err != nil {
		t.Fatal(err)
	}
	events, err := db.Events(context.Background(), session.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if events[len(events)-1].Type != "approval.decided" {
		t.Fatalf("last event = %q", events[len(events)-1].Type)
	}
	cancelled, err := service.Cancel(context.Background(), session.ID, "operator cancelled")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != domain.SessionCancelled {
		t.Fatal("session did not cancel")
	}
}
