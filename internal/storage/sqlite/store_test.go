package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func createSession(t *testing.T, store *Store) domain.Session {
	t.Helper()
	now := domain.Timestamp(time.Now())
	session := domain.Session{ID: domain.MustNewID(), Name: "Auth assessment", Objective: "Assess authorization", Targets: []string{"example.test"}, Goals: []string{"find IDOR"}, State: domain.SessionCreated, CreatedAt: now, UpdatedAt: now}
	event, err := domain.NewEvent(session.ID, 1, "session.created", now, domain.SessionCreatedPayload{Session: session})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(context.Background(), session, event); err != nil {
		t.Fatal(err)
	}
	return session
}

func TestCreateAppendProjectionAndReplay(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	session := createSession(t, store)
	event, err := domain.NewEvent(session.ID, 2, "session.state-changed", time.Now(), domain.SessionStateChangedPayload{From: domain.SessionCreated, To: domain.SessionRunning})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	projected, err := store.Session(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := store.Replay(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if projected.State != domain.SessionRunning || !reflect.DeepEqual(projected, replayed) {
		t.Fatalf("projection/replay mismatch\nprojected=%#v\nreplayed=%#v", projected, replayed)
	}
}

func TestAppendRollsBackInvalidProjection(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	session := createSession(t, store)
	event, _ := domain.NewEvent(session.ID, 2, "session.state-changed", time.Now(), domain.SessionStateChangedPayload{From: domain.SessionCompleted, To: domain.SessionRunning})
	if err := store.Append(context.Background(), event); err == nil {
		t.Fatal("expected projection failure")
	}
	events, err := store.Events(context.Background(), session.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1 after rollback", len(events))
	}
}

func TestSequenceAndCorruptionFailures(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	session := createSession(t, store)
	event, _ := domain.NewEvent(session.ID, 3, "noop", time.Now(), map[string]any{})
	if err := store.Append(context.Background(), event); err == nil {
		t.Fatal("expected sequence gap failure")
	}
	store.Close()
	corrupt := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(corrupt, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(corrupt); err == nil {
		t.Fatal("expected corrupt database failure")
	}
}
