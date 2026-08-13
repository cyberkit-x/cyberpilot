package runner

import (
	"context"
	"io"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type SandboxSpec struct {
	SessionID      domain.ID     `json:"session_id"`
	Image          string        `json:"image"`
	Workspace      string        `json:"workspace"`
	MemoryBytes    int64         `json:"memory_bytes"`
	ProcessLimit   int           `json:"process_limit"`
	NetworkProfile string        `json:"network_profile"`
	Retention      time.Duration `json:"retention"`
}

type Command struct {
	Executable  string            `json:"executable"`
	Arguments   []string          `json:"arguments"`
	Directory   string            `json:"directory"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	OutputLimit int64             `json:"output_limit"`
}

type Result struct {
	ExitCode   int                  `json:"exit_code"`
	StartedAt  time.Time            `json:"started_at"`
	FinishedAt time.Time            `json:"finished_at"`
	TimedOut   bool                 `json:"timed_out"`
	Cancelled  bool                 `json:"cancelled"`
	Uncertain  bool                 `json:"uncertain"`
	Artifacts  []domain.ArtifactRef `json:"artifacts,omitempty"`
}

type Provider interface {
	Probe(context.Context) error
	Create(context.Context, SandboxSpec) error
	Start(context.Context, domain.ID) error
	Exec(context.Context, domain.ID, Command, io.Writer, io.Writer) (Result, error)
	Stop(context.Context, domain.ID) error
	Remove(context.Context, domain.ID) error
	Inspect(context.Context, domain.ID) (SandboxSpec, error)
}
