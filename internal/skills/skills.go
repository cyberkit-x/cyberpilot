package skills

import (
	"context"
	"embed"
	"io/fs"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

//go:embed bundled/*/SKILL.md
var bundledFS embed.FS

type BundledSource struct{}

func (BundledSource) Name() string                      { return "bundled:v1" }
func (BundledSource) FS(context.Context) (fs.FS, error) { return fs.Sub(bundledFS, "bundled") }

type TrustStatus string

const (
	Compatible TrustStatus = "compatible"
	Tested     TrustStatus = "tested"
	Maintained TrustStatus = "maintained"
)

type Metadata struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	License     string      `json:"license"`
	Source      string      `json:"source"`
	Hash        string      `json:"hash"`
	Status      TrustStatus `json:"status"`
	Domains     []string    `json:"domains,omitempty"`
	Intents     []string    `json:"intents,omitempty"`
	Requires    []string    `json:"requires,omitempty"`
}

type Query struct {
	Objective    string               `json:"objective"`
	Observations []domain.Observation `json:"observations,omitempty"`
	Hypotheses   []domain.Hypothesis  `json:"hypotheses,omitempty"`
	Limit        int                  `json:"limit"`
}

type Candidate struct {
	Metadata Metadata `json:"metadata"`
	Score    float64  `json:"score"`
	Reason   string   `json:"reason"`
}

type Registry interface {
	Refresh(context.Context) error
	Search(context.Context, Query) ([]Candidate, error)
	Load(context.Context, string, string) ([]byte, error)
}

type Source interface {
	Name() string
	FS(context.Context) (fs.FS, error)
}
