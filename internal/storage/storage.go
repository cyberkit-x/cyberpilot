package storage

import (
	"context"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type EventStore interface {
	Append(context.Context, domain.Event) error
	Events(context.Context, domain.ID, uint64) ([]domain.Event, error)
	Replay(context.Context, domain.ID) (domain.Session, error)
}

type SessionStore interface {
	CreateSession(context.Context, domain.Session, domain.Event) error
	Session(context.Context, domain.ID) (domain.Session, error)
	ListSessions(context.Context) ([]domain.Session, error)
}
