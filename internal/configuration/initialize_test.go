package configuration

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/model"
)

type memoryConfigStore struct {
	config Config
	err    error
	saves  int
}

func (s *memoryConfigStore) Load(context.Context) (Config, error) { return s.config, s.err }
func (s *memoryConfigStore) Save(_ context.Context, config Config) error {
	s.saves++
	if s.err != nil {
		return s.err
	}
	s.config = config
	return nil
}

type memoryCredentialStore struct {
	puts, deletes int
	value         string
}

func (s *memoryCredentialStore) Put(_ context.Context, _, value string) (string, error) {
	s.puts++
	s.value = value
	return "native:default-model", nil
}
func (s *memoryCredentialStore) Get(context.Context, string) (string, error) { return s.value, nil }
func (s *memoryCredentialStore) Delete(context.Context, string) error        { s.deletes++; return nil }

func candidate() Candidate {
	config := validConfig()
	config.Model.CredentialRef = ""
	return Candidate{Model: config.Model, Runner: config.Runner, Credential: "top-secret"}
}

func TestInitializeValidationOrderingAndCommit(t *testing.T) {
	var order []string
	configs := &memoryConfigStore{}
	credentials := &memoryCredentialStore{}
	initializer := Initializer{Configs: configs, Credentials: credentials,
		ProbeModel: func(_ context.Context, _ ModelProfile, secret string) (model.CapabilityReport, error) {
			order = append(order, "model")
			if secret != "top-secret" {
				t.Fatal("wrong secret")
			}
			return model.CapabilityReport{ToolCalling: true, StructuredOutput: true}, nil
		},
		ProbeRunner: func(context.Context, RunnerProfile) error { order = append(order, "runner"); return nil },
	}
	got, err := initializer.Initialize(context.Background(), candidate())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"model", "runner"}) || configs.saves != 1 || credentials.puts != 1 || got.Model.CredentialRef != "native:default-model" {
		t.Fatalf("order=%v saves=%d puts=%d config=%#v", order, configs.saves, credentials.puts, got)
	}
}

func TestInitializeFailuresPreserveLastValidConfig(t *testing.T) {
	previous := validConfig()
	for _, test := range []struct {
		name                         string
		modelErr, runnerErr, saveErr error
		wantPuts, wantDeletes        int
	}{
		{name: "model", modelErr: errors.New("auth"), wantPuts: 0},
		{name: "runner", runnerErr: errors.New("unavailable"), wantPuts: 0},
		{name: "save", saveErr: errors.New("disk"), wantPuts: 1, wantDeletes: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			configs := &memoryConfigStore{config: previous}
			credentials := &memoryCredentialStore{}
			initializer := Initializer{Configs: configs, Credentials: credentials,
				ProbeModel: func(context.Context, ModelProfile, string) (model.CapabilityReport, error) {
					return model.CapabilityReport{}, test.modelErr
				},
				ProbeRunner: func(context.Context, RunnerProfile) error { return test.runnerErr },
			}
			configs.err = nil
			if test.saveErr != nil {
				configs.err = test.saveErr
			}
			_, err := initializer.Initialize(context.Background(), candidate())
			if err == nil {
				t.Fatal("expected failure")
			}
			if configs.config != previous || credentials.puts != test.wantPuts || credentials.deletes != test.wantDeletes {
				t.Fatalf("config changed or credential lifecycle wrong: %#v puts=%d deletes=%d", configs.config, credentials.puts, credentials.deletes)
			}
		})
	}
}
