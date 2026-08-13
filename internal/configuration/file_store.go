package configuration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrNotConfigured = errors.New("CyberPilot is not initialized")

type FileStore struct{ Path string }

func (s FileStore) Load(context.Context) (Config, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return Config{}, ErrNotConfigured
	}
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	if err := Validate(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (s FileStore) Save(_ context.Context, config Config) error {
	if err := Validate(config); err != nil {
		return err
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".config-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(s.Path)
	}
	return os.Rename(name, s.Path)
}

func Validate(config Config) error {
	if config.Version != SchemaVersion {
		return fmt.Errorf("unsupported configuration version %d", config.Version)
	}
	if strings.TrimSpace(config.Model.Provider) == "" || strings.TrimSpace(config.Model.Model) == "" || strings.TrimSpace(config.Model.BaseURL) == "" || strings.TrimSpace(config.Model.CredentialRef) == "" {
		return errors.New("model provider, model, base URL, and credential reference are required")
	}
	if config.Runner.Provider != "docker" && config.Runner.Provider != "podman" {
		return errors.New("runner provider must be docker or podman")
	}
	return nil
}

type RedactedConfig struct {
	Version int `json:"version" yaml:"version"`
	Model   struct {
		Provider   string `json:"provider" yaml:"provider"`
		Model      string `json:"model" yaml:"model"`
		BaseURL    string `json:"base_url" yaml:"base_url"`
		Credential string `json:"credential" yaml:"credential"`
	} `json:"model" yaml:"model"`
	Runner RunnerProfile `json:"runner" yaml:"runner"`
}

func Redact(config Config) RedactedConfig {
	var value RedactedConfig
	value.Version = config.Version
	value.Model.Provider, value.Model.Model, value.Model.BaseURL = config.Model.Provider, config.Model.Model, config.Model.BaseURL
	value.Model.Credential = credentialSource(config.Model.CredentialRef)
	value.Runner = config.Runner
	return value
}

func credentialSource(ref string) string {
	if index := strings.IndexByte(ref, ':'); index > 0 {
		return ref[:index] + ":***"
	}
	return "configured"
}
