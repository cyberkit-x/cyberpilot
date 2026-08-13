package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type FakeBehavior struct {
	Stdout, Stderr string
	ExitCode       int
	Delay          time.Duration
	Error          error
	Artifact       domain.ArtifactRef
}
type fakeSandbox struct {
	spec    SandboxSpec
	started bool
}
type Fake struct {
	mu        sync.Mutex
	Available bool
	Behavior  FakeBehavior
	sandboxes map[domain.ID]fakeSandbox
}

func NewFake() *Fake { return &Fake{Available: true, sandboxes: map[domain.ID]fakeSandbox{}} }
func (f *Fake) Probe(context.Context) error {
	if !f.Available {
		return errors.New("fake provider unavailable")
	}
	return nil
}
func (f *Fake) Create(_ context.Context, spec SandboxSpec) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.sandboxes[spec.SessionID]; exists {
		return errors.New("sandbox already exists")
	}
	f.sandboxes[spec.SessionID] = fakeSandbox{spec: spec}
	return nil
}
func (f *Fake) Start(_ context.Context, id domain.ID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	box, ok := f.sandboxes[id]
	if !ok {
		return errors.New("sandbox not found")
	}
	box.started = true
	f.sandboxes[id] = box
	return nil
}
func (f *Fake) Exec(ctx context.Context, id domain.ID, command Command, stdout, stderr io.Writer) (Result, error) {
	f.mu.Lock()
	box, ok := f.sandboxes[id]
	behavior := f.Behavior
	f.mu.Unlock()
	if !ok || !box.started {
		return Result{}, errors.New("sandbox is not running")
	}
	started := time.Now().UTC()
	wait := behavior.Delay
	if command.Timeout > 0 && wait > command.Timeout {
		wait = command.Timeout
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Result{StartedAt: started, FinishedAt: time.Now().UTC(), Cancelled: !errors.Is(ctx.Err(), context.DeadlineExceeded), TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded)}, ctx.Err()
	case <-timer.C:
	}
	if behavior.Stdout != "" {
		_, _ = io.WriteString(stdout, behavior.Stdout)
	}
	if behavior.Stderr != "" {
		_, _ = io.WriteString(stderr, behavior.Stderr)
	}
	result := Result{ExitCode: behavior.ExitCode, StartedAt: started, FinishedAt: time.Now().UTC()}
	if behavior.Artifact.ID.Validate() == nil {
		result.Artifacts = []domain.ArtifactRef{behavior.Artifact}
	}
	return result, behavior.Error
}
func (f *Fake) Stop(_ context.Context, id domain.ID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	box, ok := f.sandboxes[id]
	if !ok {
		return errors.New("sandbox not found")
	}
	box.started = false
	f.sandboxes[id] = box
	return nil
}
func (f *Fake) Remove(_ context.Context, id domain.ID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sandboxes[id]; !ok {
		return errors.New("sandbox not found")
	}
	delete(f.sandboxes, id)
	return nil
}
func (f *Fake) Inspect(_ context.Context, id domain.ID) (SandboxSpec, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	box, ok := f.sandboxes[id]
	if !ok {
		return SandboxSpec{}, fmt.Errorf("sandbox not found")
	}
	return box.spec, nil
}
