package sqlite

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestLinkedInvestigationRecordPersistenceAndQueries(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	session := domain.Session{ID: domain.MustNewID(), Name: "hunt", Objective: "test", Targets: []string{"http://127.0.0.1"}, Goals: []string{"authorization"}, State: domain.SessionCreated, CreatedAt: now, UpdatedAt: now}
	event, _ := domain.NewEvent(session.ID, 1, "session.created", now, domain.SessionCreatedPayload{Session: session})
	if err := store.CreateSession(ctx, session, event); err != nil {
		t.Fatal(err)
	}
	hypothesis := domain.Hypothesis{ID: domain.MustNewID(), SessionID: session.ID, Claim: "object access", State: domain.HypothesisTesting}
	observation := domain.Observation{ID: domain.MustNewID(), SessionID: session.ID, ActionID: domain.MustNewID(), Summary: "control denied"}
	lead := domain.Lead{ID: domain.MustNewID(), SessionID: session.ID, HypothesisID: hypothesis.ID, Title: "possible IDOR", EvidenceIDs: []domain.ID{observation.ID}}
	gap := domain.CoverageGap{ID: domain.MustNewID(), SessionID: session.ID, Goal: "admin path", Reason: "approval required", Blocked: true}
	values := []struct {
		kind  string
		id    domain.ID
		value any
	}{{"hypothesis", hypothesis.ID, hypothesis}, {"observation", observation.ID, observation}, {"lead", lead.ID, lead}, {"coverage-gap", gap.ID, gap}}
	for _, value := range values {
		if err := store.PutRecord(ctx, session.ID, value.kind, value.id, value.value); err != nil {
			t.Fatal(err)
		}
	}
	records, err := store.Records(ctx, session.ID, "lead")
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%v err=%v", records, err)
	}
	var got domain.Lead
	if err := json.Unmarshal(records[0], &got); err != nil || got.HypothesisID != hypothesis.ID || got.EvidenceIDs[0] != observation.ID {
		t.Fatalf("lead=%#v err=%v", got, err)
	}
}
