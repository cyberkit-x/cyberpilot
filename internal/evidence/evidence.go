package evidence

import (
	"context"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

type PromotionResult struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons"`
}

type Validator interface {
	Validate(context.Context, domain.Finding) (PromotionResult, error)
}

type Store interface {
	Put(context.Context, domain.ID, string, bool, []byte) (domain.ArtifactRef, error)
	Open(context.Context, domain.ID, domain.ID) ([]byte, domain.ArtifactRef, error)
}
