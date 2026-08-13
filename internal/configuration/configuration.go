package configuration

import (
	"context"

	"github.com/cyberkit-x/cyberpilot/internal/model"
)

const SchemaVersion = 1

type ModelProfile struct {
	Provider      string `yaml:"provider" json:"provider"`
	Model         string `yaml:"model" json:"model"`
	BaseURL       string `yaml:"base_url" json:"base_url"`
	CredentialRef string `yaml:"credential" json:"credential"`
}

type RunnerProfile struct {
	Provider   string `yaml:"provider" json:"provider"`
	Connection string `yaml:"connection" json:"connection"`
	Rootless   bool   `yaml:"rootless" json:"rootless"`
}

type Config struct {
	Version int           `yaml:"version" json:"version"`
	Model   ModelProfile  `yaml:"model" json:"model"`
	Runner  RunnerProfile `yaml:"runner" json:"runner"`
}

type Store interface {
	Load(context.Context) (Config, error)
	Save(context.Context, Config) error
}

type Validator interface {
	ValidateModel(context.Context, ModelProfile) (model.CapabilityReport, error)
	ValidateRunner(context.Context, RunnerProfile) error
}
