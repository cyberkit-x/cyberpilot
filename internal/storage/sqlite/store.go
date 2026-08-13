package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	_ "modernc.org/sqlite"
)

const currentSchema = 1

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.configureAndMigrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) configureAndMigrate(ctx context.Context) error {
	for _, pragma := range []string{"PRAGMA journal_mode=WAL", "PRAGMA foreign_keys=ON", "PRAGMA busy_timeout=5000", "PRAGMA synchronous=FULL"} {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("configure sqlite: %w", err)
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
		`INSERT INTO schema_version(version) SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM schema_version)`,
		`CREATE TABLE IF NOT EXISTS events (
            id TEXT PRIMARY KEY, session_id TEXT NOT NULL, sequence INTEGER NOT NULL,
            schema_version INTEGER NOT NULL, type TEXT NOT NULL, occurred_at TEXT NOT NULL, payload BLOB NOT NULL,
            UNIQUE(session_id, sequence))`,
		`CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY, name TEXT NOT NULL, objective TEXT NOT NULL,
            targets BLOB NOT NULL, goals BLOB NOT NULL, constraints BLOB NOT NULL,
            instructions TEXT NOT NULL DEFAULT '', scope_confirmed INTEGER NOT NULL DEFAULT 0,
            state TEXT NOT NULL, terminal_reason TEXT NOT NULL,
            created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS investigation_records (
            session_id TEXT NOT NULL, kind TEXT NOT NULL, id TEXT NOT NULL, data BLOB NOT NULL,
            PRIMARY KEY(session_id, kind, id),
            FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS events_by_session ON events(session_id, sequence)`,
		`UPDATE schema_version SET version = 1 WHERE version = 0`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&version); err != nil {
		return err
	}
	if version != currentSchema {
		return fmt.Errorf("unsupported database schema %d", version)
	}
	return tx.Commit()
}

func (s *Store) CreateSession(ctx context.Context, session domain.Session, event domain.Event) error {
	if event.SessionID != session.ID || event.Type != "session.created" {
		return fmt.Errorf("session creation requires matching session.created event")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := insertSession(ctx, tx, session); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Append(ctx context.Context, event domain.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertEvent(ctx, tx, event); err != nil {
		return err
	}
	if err := applyProjection(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit()
}

func insertEvent(ctx context.Context, tx *sql.Tx, event domain.Event) error {
	if event.SchemaVersion != domain.EventSchemaVersion || event.ID.Validate() != nil || event.SessionID.Validate() != nil || event.Sequence == 0 || event.Type == "" {
		return fmt.Errorf("invalid event envelope")
	}
	var expected uint64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM events WHERE session_id = ?`, event.SessionID).Scan(&expected); err != nil {
		return err
	}
	if event.Sequence != expected {
		return fmt.Errorf("event sequence %d, expected %d", event.Sequence, expected)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO events(id,session_id,sequence,schema_version,type,occurred_at,payload) VALUES(?,?,?,?,?,?,?)`,
		event.ID, event.SessionID, event.Sequence, event.SchemaVersion, event.Type, event.OccurredAt.UTC().Format(time.RFC3339Nano), []byte(event.Payload))
	return err
}

func insertSession(ctx context.Context, tx *sql.Tx, session domain.Session) error {
	targets, _ := json.Marshal(session.Targets)
	goals, _ := json.Marshal(session.Goals)
	constraints, _ := json.Marshal(session.Constraints)
	_, err := tx.ExecContext(ctx, `INSERT INTO sessions(id,name,objective,targets,goals,constraints,instructions,scope_confirmed,state,terminal_reason,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		session.ID, session.Name, session.Objective, targets, goals, constraints, session.Instructions, session.ScopeConfirmed, session.State, session.TerminalReason,
		session.CreatedAt.UTC().Format(time.RFC3339Nano), session.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func applyProjection(ctx context.Context, tx *sql.Tx, event domain.Event) error {
	switch event.Type {
	case "session.state-changed":
		var payload domain.SessionStateChangedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		var current domain.SessionState
		if err := tx.QueryRowContext(ctx, `SELECT state FROM sessions WHERE id = ?`, event.SessionID).Scan(&current); err != nil {
			return err
		}
		if current != payload.From {
			return fmt.Errorf("projected state %q does not match event from %q", current, payload.From)
		}
		if err := domain.ValidateSessionTransition(payload.From, payload.To); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE sessions SET state=?, terminal_reason=?, updated_at=? WHERE id=?`, payload.To, payload.Reason, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.SessionID)
		return err
	case "session.instructions-updated":
		var payload domain.SessionInstructionsUpdatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE sessions SET instructions=?, updated_at=? WHERE id=?`, payload.Instructions, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.SessionID)
		return err
	case "session.scope-updated":
		var payload domain.SessionScopeUpdatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		targets, err := json.Marshal(payload.Targets)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE sessions SET targets=?, scope_confirmed=?, updated_at=? WHERE id=?`, targets, payload.Confirmed, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.SessionID)
		return err
	default:
		return nil
	}
}

func (s *Store) Events(ctx context.Context, sessionID domain.ID, after uint64) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,sequence,schema_version,type,occurred_at,payload FROM events WHERE session_id=? AND sequence>? ORDER BY sequence`, sessionID, after)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var events []domain.Event
	for rows.Next() {
		var event domain.Event
		var occurred string
		var payload []byte
		event.SessionID = sessionID
		if err := rows.Scan(&event.ID, &event.Sequence, &event.SchemaVersion, &event.Type, &occurred, &payload); err != nil {
			return nil, err
		}
		event.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
		if err != nil {
			return nil, err
		}
		event.Payload = payload
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) Session(ctx context.Context, id domain.ID) (domain.Session, error) {
	return scanSession(s.db.QueryRowContext(ctx, `SELECT id,name,objective,targets,goals,constraints,instructions,scope_confirmed,state,terminal_reason,created_at,updated_at FROM sessions WHERE id=?`, id))
}

func (s *Store) ListSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,objective,targets,goals,constraints,instructions,scope_confirmed,state,terminal_reason,created_at,updated_at FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var sessions []domain.Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanSession(row scanner) (domain.Session, error) {
	var session domain.Session
	var targets, goals, constraints []byte
	var created, updated string
	if err := row.Scan(&session.ID, &session.Name, &session.Objective, &targets, &goals, &constraints, &session.Instructions, &session.ScopeConfirmed, &session.State, &session.TerminalReason, &created, &updated); err != nil {
		return domain.Session{}, err
	}
	if err := json.Unmarshal(targets, &session.Targets); err != nil {
		return domain.Session{}, err
	}
	if err := json.Unmarshal(goals, &session.Goals); err != nil {
		return domain.Session{}, err
	}
	if err := json.Unmarshal(constraints, &session.Constraints); err != nil {
		return domain.Session{}, err
	}
	var err error
	if session.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return domain.Session{}, err
	}
	if session.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return domain.Session{}, err
	}
	return session, nil
}

func (s *Store) Replay(ctx context.Context, sessionID domain.ID) (domain.Session, error) {
	events, err := s.Events(ctx, sessionID, 0)
	if err != nil {
		return domain.Session{}, err
	}
	if len(events) == 0 {
		return domain.Session{}, sql.ErrNoRows
	}
	var session domain.Session
	for _, event := range events {
		switch event.Type {
		case "session.created":
			var payload domain.SessionCreatedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return domain.Session{}, err
			}
			session = payload.Session
		case "session.state-changed":
			var payload domain.SessionStateChangedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return domain.Session{}, err
			}
			if session.State != payload.From {
				return domain.Session{}, fmt.Errorf("replay state mismatch")
			}
			if err := domain.ValidateSessionTransition(payload.From, payload.To); err != nil {
				return domain.Session{}, err
			}
			session.State, session.TerminalReason, session.UpdatedAt = payload.To, payload.Reason, event.OccurredAt
		case "session.instructions-updated":
			var payload domain.SessionInstructionsUpdatedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return domain.Session{}, err
			}
			session.Instructions, session.UpdatedAt = payload.Instructions, event.OccurredAt
		case "session.scope-updated":
			var payload domain.SessionScopeUpdatedPayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return domain.Session{}, err
			}
			session.Targets, session.ScopeConfirmed, session.UpdatedAt = payload.Targets, payload.Confirmed, event.OccurredAt
		}
	}
	if session.ID == "" {
		return domain.Session{}, errors.New("session creation event missing")
	}
	return session, nil
}
