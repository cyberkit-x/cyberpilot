package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
	"github.com/cyberkit-x/cyberpilot/internal/storage"
)

type Store interface {
	storage.EventStore
	storage.SessionStore
	PutRecord(context.Context, domain.ID, string, domain.ID, any) error
	Records(context.Context, domain.ID, string) ([]json.RawMessage, error)
}

type SessionService struct {
	store    Store
	mu       sync.Mutex
	onCreate func(context.Context, domain.Session)
}

func NewSessionService(store Store) *SessionService { return &SessionService{store: store} }

func (s *SessionService) OnCreate(callback func(context.Context, domain.Session)) {
	s.onCreate = callback
}

type CreateSessionInput struct {
	Objective   string   `json:"objective"`
	Name        string   `json:"name,omitempty"`
	Targets     []string `json:"targets"`
	Goals       []string `json:"goals"`
	Constraints []string `json:"constraints,omitempty"`
}

func (s *SessionService) Create(ctx context.Context, input CreateSessionInput) (domain.Session, error) {
	if strings.TrimSpace(input.Objective) == "" || len(input.Targets) == 0 || len(input.Goals) == 0 {
		return domain.Session{}, public(domain.ErrInvalidInput, "objective, targets, and goals are required")
	}
	now := domain.Timestamp(time.Now())
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = deriveName(input.Objective)
	}
	session := domain.Session{ID: domain.MustNewID(), Name: name, Objective: strings.TrimSpace(input.Objective), Targets: append([]string(nil), input.Targets...), Goals: append([]string(nil), input.Goals...), Constraints: append([]string(nil), input.Constraints...), State: domain.SessionCreated, CreatedAt: now, UpdatedAt: now}
	event, err := domain.NewEvent(session.ID, 1, "session.created", now, domain.SessionCreatedPayload{Session: session})
	if err != nil {
		return domain.Session{}, err
	}
	if err := s.store.CreateSession(ctx, session, event); err != nil {
		return domain.Session{}, err
	}
	if s.onCreate != nil {
		s.onCreate(context.WithoutCancel(ctx), session)
	}
	return session, nil
}

func deriveName(objective string) string {
	words := strings.Fields(objective)
	if len(words) > 6 {
		words = words[:6]
	}
	name := strings.Join(words, " ")
	if len(name) > 48 {
		name = name[:45] + "..."
	}
	return name
}

func (s *SessionService) transition(ctx context.Context, id domain.ID, to domain.SessionState, reason string) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, err := s.store.Session(ctx, id)
	if err != nil {
		return domain.Session{}, err
	}
	if err := domain.ValidateSessionTransition(session.State, to); err != nil {
		return domain.Session{}, err
	}
	events, err := s.store.Events(ctx, id, 0)
	if err != nil {
		return domain.Session{}, err
	}
	event, err := domain.NewEvent(id, uint64(len(events)+1), "session.state-changed", time.Now(), domain.SessionStateChangedPayload{From: session.State, To: to, Reason: reason})
	if err != nil {
		return domain.Session{}, err
	}
	if err := s.store.Append(ctx, event); err != nil {
		return domain.Session{}, err
	}
	return s.store.Session(ctx, id)
}

func (s *SessionService) Cancel(ctx context.Context, id domain.ID, reason string) (domain.Session, error) {
	return s.transition(ctx, id, domain.SessionCancelled, reason)
}

func (s *SessionService) Transition(ctx context.Context, id domain.ID, to domain.SessionState, reason string) (domain.Session, error) {
	return s.transition(ctx, id, to, reason)
}

func (s *SessionService) Get(ctx context.Context, id domain.ID) (domain.Session, error) {
	return s.store.Session(ctx, id)
}

func (s *SessionService) PutRecord(ctx context.Context, session domain.ID, kind string, id domain.ID, value any) error {
	if err := s.store.PutRecord(ctx, session, kind, id, value); err != nil {
		return err
	}
	_, err := s.append(ctx, session, kind+".recorded", value)
	return err
}

func (s *SessionService) append(ctx context.Context, id domain.ID, eventType string, payload any) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.store.Session(ctx, id); err != nil {
		return domain.Session{}, err
	}
	events, err := s.store.Events(ctx, id, 0)
	if err != nil {
		return domain.Session{}, err
	}
	event, err := domain.NewEvent(id, uint64(len(events)+1), eventType, time.Now(), payload)
	if err != nil {
		return domain.Session{}, err
	}
	if err := s.store.Append(ctx, event); err != nil {
		return domain.Session{}, err
	}
	return s.store.Session(ctx, id)
}

func (s *SessionService) UpdateInstructions(ctx context.Context, id domain.ID, instructions string) (domain.Session, error) {
	return s.append(ctx, id, "session.instructions-updated", domain.SessionInstructionsUpdatedPayload{Instructions: strings.TrimSpace(instructions)})
}

func (s *SessionService) UpdateScope(ctx context.Context, id domain.ID, targets []string, confirmed bool) (domain.Session, error) {
	if len(targets) == 0 {
		return domain.Session{}, public(domain.ErrInvalidInput, "scope requires at least one target")
	}
	return s.append(ctx, id, "session.scope-updated", domain.SessionScopeUpdatedPayload{Targets: append([]string(nil), targets...), Confirmed: confirmed})
}

func (s *SessionService) DecideApproval(ctx context.Context, sessionID domain.ID, approval domain.Approval) (domain.Session, error) {
	if approval.ID.Validate() != nil || approval.ActionID.Validate() != nil {
		return domain.Session{}, public(domain.ErrInvalidInput, "invalid approval identifiers")
	}
	if approval.State != domain.ApprovalAllowed && approval.State != domain.ApprovalDenied && approval.State != domain.ApprovalRevoked {
		return domain.Session{}, public(domain.ErrInvalidInput, "invalid approval decision")
	}
	return s.append(ctx, sessionID, "approval.decided", approval)
}

func (s *SessionService) Recover(ctx context.Context) error {
	sessions, err := s.store.ListSessions(ctx)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.State == domain.SessionRunning {
			if _, err := s.transition(ctx, session.ID, domain.SessionNeedsInput, "runtime restarted; interrupted actions require review"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SessionService) Register(server *rpc.Server) {
	server.Handle("system.ready", func(context.Context, json.RawMessage) (any, *domain.PublicError) {
		return map[string]any{"ready": true, "protocol": rpc.ProtocolVersion}, nil
	})
	server.Handle("session.create", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input CreateSessionInput
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, public(domain.ErrInvalidInput, "invalid create request")
		}
		value, err := s.Create(ctx, input)
		return rpcResult(value, err)
	})
	server.Handle("session.list", func(ctx context.Context, _ json.RawMessage) (any, *domain.PublicError) {
		value, err := s.store.ListSessions(ctx)
		return rpcResult(value, err)
	})
	server.Handle("session.get", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			ID domain.ID `json:"id"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.ID.Validate() != nil {
			return nil, public(domain.ErrInvalidInput, "invalid session id")
		}
		value, err := s.store.Session(ctx, input.ID)
		return rpcResult(value, err)
	})
	server.Handle("session.cancel", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			ID     domain.ID `json:"id"`
			Reason string    `json:"reason"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.ID.Validate() != nil {
			return nil, public(domain.ErrInvalidInput, "invalid cancel request")
		}
		value, err := s.Cancel(ctx, input.ID, input.Reason)
		return rpcResult(value, err)
	})
	server.Handle("session.instructions.update", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			ID           domain.ID `json:"id"`
			Instructions string    `json:"instructions"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.ID.Validate() != nil {
			return nil, public(domain.ErrInvalidInput, "invalid instruction update")
		}
		value, err := s.UpdateInstructions(ctx, input.ID, input.Instructions)
		return rpcResult(value, err)
	})
	server.Handle("session.scope.update", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			ID        domain.ID `json:"id"`
			Targets   []string  `json:"targets"`
			Confirmed bool      `json:"confirmed"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.ID.Validate() != nil {
			return nil, public(domain.ErrInvalidInput, "invalid scope update")
		}
		value, err := s.UpdateScope(ctx, input.ID, input.Targets, input.Confirmed)
		return rpcResult(value, err)
	})
	server.Handle("session.events", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			ID    domain.ID `json:"id"`
			After uint64    `json:"after"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.ID.Validate() != nil {
			return nil, public(domain.ErrInvalidInput, "invalid event cursor")
		}
		value, err := s.store.Events(ctx, input.ID, input.After)
		return rpcResult(value, err)
	})
	server.Handle("session.records", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			ID   domain.ID `json:"id"`
			Kind string    `json:"kind"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.ID.Validate() != nil || strings.TrimSpace(input.Kind) == "" {
			return nil, public(domain.ErrInvalidInput, "invalid record query")
		}
		value, err := s.store.Records(ctx, input.ID, input.Kind)
		return rpcResult(value, err)
	})
	server.Handle("approval.decide", func(ctx context.Context, raw json.RawMessage) (any, *domain.PublicError) {
		var input struct {
			SessionID domain.ID       `json:"session_id"`
			Approval  domain.Approval `json:"approval"`
		}
		if err := json.Unmarshal(raw, &input); err != nil || input.SessionID.Validate() != nil {
			return nil, public(domain.ErrInvalidInput, "invalid approval decision")
		}
		value, err := s.DecideApproval(ctx, input.SessionID, input.Approval)
		return rpcResult(value, err)
	})
}

func rpcResult(value any, err error) (any, *domain.PublicError) {
	if err == nil {
		return value, nil
	}
	var publicErr *domain.PublicError
	if errors.As(err, &publicErr) {
		return nil, publicErr
	}
	return nil, public(domain.ErrInternal, fmt.Sprintf("operation failed: %v", err))
}

func public(code domain.ErrorCode, message string) *domain.PublicError {
	return &domain.PublicError{Code: code, Message: message}
}
