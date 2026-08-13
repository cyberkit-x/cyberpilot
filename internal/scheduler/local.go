package scheduler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

var ErrBudgetExhausted = errors.New("budget exhausted")

type Usage struct {
	Actions      int
	InputTokens  int64
	OutputTokens int64
	Cost         float64
	NoProgress   int
}

type TurnUsage struct {
	InputTokens  int64
	OutputTokens int64
	Cost         float64
}

type Local struct {
	actions  chan struct{}
	mu       sync.Mutex
	planning map[domain.ID]*sync.Mutex
	usage    map[domain.ID]Usage
}

func NewLocal(maxConcurrentActions int) *Local {
	if maxConcurrentActions <= 0 {
		maxConcurrentActions = 1
	}
	return &Local{actions: make(chan struct{}, maxConcurrentActions), planning: map[domain.ID]*sync.Mutex{}, usage: map[domain.ID]Usage{}}
}

func (s *Local) WithPlanning(ctx context.Context, session domain.ID, budget Budget, fn func(context.Context) (TurnUsage, bool, error)) error {
	lock := s.sessionLock(session)
	lock.Lock()
	defer lock.Unlock()
	if err := s.check(session, budget); err != nil {
		return err
	}
	callCtx, cancel := budgetContext(ctx, budget)
	defer cancel()
	usage, progress, err := fn(callCtx)
	s.mu.Lock()
	current := s.usage[session]
	current.InputTokens += usage.InputTokens
	current.OutputTokens += usage.OutputTokens
	current.Cost += usage.Cost
	if progress {
		current.NoProgress = 0
	} else {
		current.NoProgress++
	}
	s.usage[session] = current
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return s.check(session, budget)
}

func (s *Local) WithAction(ctx context.Context, session domain.ID, budget Budget, fn func(context.Context) error) error {
	if err := s.check(session, budget); err != nil {
		return err
	}
	select {
	case s.actions <- struct{}{}:
		defer func() { <-s.actions }()
	case <-ctx.Done():
		return ctx.Err()
	}
	actionCtx, cancel := budgetContext(ctx, budget)
	defer cancel()
	err := fn(actionCtx)
	s.mu.Lock()
	u := s.usage[session]
	u.Actions++
	s.usage[session] = u
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return s.check(session, budget)
}

func (s *Local) Usage(session domain.ID) Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage[session]
}
func (s *Local) sessionLock(id domain.ID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.planning[id] == nil {
		s.planning[id] = &sync.Mutex{}
	}
	return s.planning[id]
}
func (s *Local) check(id domain.ID, b Budget) error {
	s.mu.Lock()
	u := s.usage[id]
	s.mu.Unlock()
	if !b.Deadline.IsZero() && time.Now().After(b.Deadline) || b.MaxActions > 0 && u.Actions >= b.MaxActions || b.MaxInputTokens > 0 && u.InputTokens >= b.MaxInputTokens || b.MaxOutputTokens > 0 && u.OutputTokens >= b.MaxOutputTokens || b.MaxCost > 0 && u.Cost >= b.MaxCost || b.MaxNoProgress > 0 && u.NoProgress >= b.MaxNoProgress {
		return ErrBudgetExhausted
	}
	return nil
}
func budgetContext(ctx context.Context, b Budget) (context.Context, context.CancelFunc) {
	if !b.Deadline.IsZero() {
		return context.WithDeadline(ctx, b.Deadline)
	}
	return context.WithCancel(ctx)
}
