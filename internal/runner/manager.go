package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type Capture struct {
	Stdout, Stderr                   []byte
	StdoutTruncated, StderrTruncated bool
	Result                           Result
}
type Manager struct {
	Provider      Provider
	WorkspaceRoot string
	mu            sync.Mutex
}

func (m *Manager) Ensure(ctx context.Context, spec SandboxSpec) error {
	if m.Provider == nil {
		return errors.New("runner provider is unavailable")
	}
	workspace := filepath.Join(m.WorkspaceRoot, string(spec.SessionID))
	if err := os.MkdirAll(workspace, 0700); err != nil {
		return err
	}
	spec.Workspace = workspace
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, err := m.Provider.Inspect(ctx, spec.SessionID); err == nil {
		if existing.SessionID != spec.SessionID {
			return errors.New("sandbox ownership mismatch")
		}
		return m.Provider.Start(ctx, spec.SessionID)
	}
	if err := m.Provider.Create(ctx, spec); err != nil {
		return err
	}
	return m.Provider.Start(ctx, spec.SessionID)
}
func (m *Manager) Execute(ctx context.Context, session domain.ID, command Command) (Capture, error) {
	limit := command.OutputLimit
	if limit <= 0 {
		limit = 1 << 20
	}
	stdout := &limitedWriter{remaining: limit}
	stderr := &limitedWriter{remaining: limit}
	result, err := m.Provider.Exec(ctx, session, command, stdout, stderr)
	return Capture{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), StdoutTruncated: stdout.truncated, StderrTruncated: stderr.truncated, Result: result}, err
}
func (m *Manager) Recover(ctx context.Context, session domain.ID) (SandboxSpec, error) {
	return m.Provider.Inspect(ctx, session)
}
func (m *Manager) Cleanup(ctx context.Context, session domain.ID, retain bool) error {
	if retain {
		return m.Provider.Stop(ctx, session)
	}
	if err := m.Provider.Stop(ctx, session); err != nil {
		return fmt.Errorf("stop sandbox: %w", err)
	}
	return m.Provider.Remove(ctx, session)
}

type limitedWriter struct {
	buffer    bytes.Buffer
	remaining int64
	truncated bool
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	original := len(data)
	if w.remaining <= 0 {
		w.truncated = true
		return original, nil
	}
	if int64(len(data)) > w.remaining {
		data = data[:w.remaining]
		w.truncated = true
	}
	_, _ = w.buffer.Write(data)
	w.remaining -= int64(len(data))
	return original, nil
}
func (w *limitedWriter) Bytes() []byte { return append([]byte(nil), w.buffer.Bytes()...) }

var _ io.Writer = (*limitedWriter)(nil)
