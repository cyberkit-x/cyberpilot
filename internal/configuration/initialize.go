package configuration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cyberkit-x/cyberpilot/internal/credentials"
	"github.com/cyberkit-x/cyberpilot/internal/model"
)

type Candidate struct {
	Model      ModelProfile
	Runner     RunnerProfile
	Credential string
}

type ModelProbe func(context.Context, ModelProfile, string) (model.CapabilityReport, error)
type RunnerProbe func(context.Context, RunnerProfile) error

type Initializer struct {
	Configs     Store
	Credentials credentials.Store
	ProbeModel  ModelProbe
	ProbeRunner RunnerProbe
}

// Initialize validates a complete candidate before making any persistent
// change. The credential is written immediately before the atomic config write
// and is removed if that final write fails.
func (i Initializer) Initialize(ctx context.Context, candidate Candidate) (Config, error) {
	if i.Configs == nil || i.Credentials == nil || i.ProbeModel == nil || i.ProbeRunner == nil {
		return Config{}, errors.New("initializer dependencies are incomplete")
	}
	if strings.TrimSpace(candidate.Credential) == "" {
		return Config{}, errors.New("model credential is required")
	}
	candidate.Model.CredentialRef = "pending:credential"
	config := Config{Version: SchemaVersion, Model: candidate.Model, Runner: candidate.Runner}
	if err := Validate(config); err != nil {
		return Config{}, err
	}
	if _, err := i.ProbeModel(ctx, candidate.Model, candidate.Credential); err != nil {
		return Config{}, fmt.Errorf("model validation failed: %w", err)
	}
	if err := i.ProbeRunner(ctx, candidate.Runner); err != nil {
		return Config{}, fmt.Errorf("runner validation failed: %w", err)
	}
	ref, err := i.Credentials.Put(ctx, "default-model", candidate.Credential)
	if err != nil {
		return Config{}, fmt.Errorf("protect model credential: %w", err)
	}
	config.Model.CredentialRef = ref
	if err := i.Configs.Save(ctx, config); err != nil {
		_ = i.Credentials.Delete(ctx, ref)
		return Config{}, fmt.Errorf("save configuration: %w", err)
	}
	return config, nil
}
