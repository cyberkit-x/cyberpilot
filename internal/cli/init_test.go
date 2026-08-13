package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/configuration"
	"github.com/cyberkit-x/cyberpilot/internal/credentials"
	"github.com/cyberkit-x/cyberpilot/internal/model"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
)

type testConfigStore struct {
	config     configuration.Config
	configured bool
	saves      int
}

func (s *testConfigStore) Load(context.Context) (configuration.Config, error) {
	if !s.configured {
		return configuration.Config{}, configuration.ErrNotConfigured
	}
	return s.config, nil
}
func (s *testConfigStore) Save(_ context.Context, config configuration.Config) error {
	s.config, s.configured = config, true
	s.saves++
	return nil
}

type testCredentials struct{}

func (testCredentials) Put(context.Context, string, string) (string, error) {
	return "test:credential", nil
}
func (testCredentials) Get(context.Context, string) (string, error) { return "secret", nil }
func (testCredentials) Delete(context.Context, string) error        { return nil }

var _ credentials.Store = testCredentials{}

func TestInitInteractiveSuccess(t *testing.T) {
	store := &testConfigStore{}
	var order []string
	initializer := configuration.Initializer{Configs: store, Credentials: testCredentials{},
		ProbeModel: func(context.Context, configuration.ModelProfile, string) (model.CapabilityReport, error) {
			order = append(order, "model")
			return model.CapabilityReport{}, nil
		},
		ProbeRunner: func(context.Context, configuration.RunnerProfile) error { order = append(order, "runner"); return nil },
	}
	var output bytes.Buffer
	command := InitCommand{Input: strings.NewReader("http://127.0.0.1:8080/v1\ntest-model\nsecret\n2\n"), Output: &output, Configs: store, Initializer: initializer,
		Discover: func(context.Context) ([]runner.ProviderSummary, error) {
			return []runner.ProviderSummary{{Provider: "docker", Endpoint: "docker-default"}, {Provider: "podman", Endpoint: "podman-default", Rootless: true}}, nil
		},
	}
	if err := command.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "model,runner" || store.config.Runner.Provider != "podman" || !strings.Contains(output.String(), "CyberPilot is ready") {
		t.Fatalf("order=%v config=%#v output=%q", order, store.config, output.String())
	}
}

func TestInitRequiresConfirmationAndPreservesExisting(t *testing.T) {
	current := configuration.Config{Version: configuration.SchemaVersion, Model: configuration.ModelProfile{Provider: "openai-compatible", Model: "old", BaseURL: "http://old", CredentialRef: "test:old"}, Runner: configuration.RunnerProfile{Provider: "docker", Connection: "default"}}
	store := &testConfigStore{config: current, configured: true}
	var output bytes.Buffer
	command := InitCommand{Input: strings.NewReader("n\n"), Output: &output, Configs: store, Discover: func(context.Context) ([]runner.ProviderSummary, error) { return nil, errors.New("must not run") }}
	if err := command.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.saves != 0 || store.config != current || !strings.Contains(output.String(), "Configuration unchanged") {
		t.Fatalf("configuration changed: %#v", store.config)
	}
}
