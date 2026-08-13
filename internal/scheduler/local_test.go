package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestPlanningSerializedPerSessionAndParallelAcrossSessions(t *testing.T) {
	s := NewLocal(2)
	a, b := domain.MustNewID(), domain.MustNewID()
	var activeA, maxA, activeAll, maxAll atomic.Int32
	run := func(id domain.ID) {
		_ = s.WithPlanning(context.Background(), id, Budget{}, func(context.Context) (TurnUsage, bool, error) {
			aa := activeAll.Add(1)
			if aa > maxAll.Load() {
				maxAll.Store(aa)
			}
			if id == a {
				x := activeA.Add(1)
				if x > maxA.Load() {
					maxA.Store(x)
				}
			}
			time.Sleep(20 * time.Millisecond)
			if id == a {
				activeA.Add(-1)
			}
			activeAll.Add(-1)
			return TurnUsage{}, true, nil
		})
	}
	done := make(chan struct{}, 3)
	for _, id := range []domain.ID{a, a, b} {
		go func(id domain.ID) { run(id); done <- struct{}{} }(id)
	}
	for range 3 {
		<-done
	}
	if maxA.Load() != 1 || maxAll.Load() < 2 {
		t.Fatalf("maxA=%d maxAll=%d", maxA.Load(), maxAll.Load())
	}
}
func TestActionConcurrencyAndAllBudgets(t *testing.T) {
	s := NewLocal(1)
	id := domain.MustNewID()
	b := Budget{MaxActions: 1, MaxInputTokens: 2, MaxOutputTokens: 3, MaxCost: 1, MaxNoProgress: 2}
	if err := s.WithAction(context.Background(), id, b, func(context.Context) error { return nil }); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err=%v", err)
	}
	id2 := domain.MustNewID()
	if err := s.WithPlanning(context.Background(), id2, b, func(context.Context) (TurnUsage, bool, error) {
		return TurnUsage{InputTokens: 2, OutputTokens: 3, Cost: 1}, false, nil
	}); !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err=%v", err)
	}
}
func TestDeadlineBudget(t *testing.T) {
	s := NewLocal(1)
	err := s.WithAction(context.Background(), domain.MustNewID(), Budget{Deadline: time.Now().Add(-time.Second)}, func(context.Context) error { return nil })
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err=%v", err)
	}
}
