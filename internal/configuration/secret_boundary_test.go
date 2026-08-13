package configuration

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/model"
	"gopkg.in/yaml.v3"
)

// This is a cross-layer regression test for the credential boundary: the raw
// value is consumed only by the probe and credential store. Durable config,
// event payloads, rendered diagnostics, and references contain no secret.
func TestCredentialNeverEntersDurableOrRenderedRecords(t *testing.T) {
	const secret = "cyberpilot-secret-boundary-value"
	configs := &memoryConfigStore{}
	credentials := &memoryCredentialStore{}
	initializer := Initializer{Configs: configs, Credentials: credentials,
		ProbeModel: func(_ context.Context, _ ModelProfile, got string) (model.CapabilityReport, error) {
			if got != secret {
				t.Fatal("probe did not receive credential")
			}
			return model.CapabilityReport{}, nil
		},
		ProbeRunner: func(context.Context, RunnerProfile) error { return nil },
	}
	input := candidate()
	input.Credential = secret
	config, err := initializer.Initialize(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	yamlData, err := yamlMarshal(config)
	if err != nil {
		t.Fatal(err)
	}
	event, err := domain.NewEvent(domain.MustNewID(), 1, "configuration.updated", time.Now().UTC(), Redact(config))
	if err != nil {
		t.Fatal(err)
	}
	eventData, _ := json.Marshal(event)
	var diagnostics bytes.Buffer
	_, _ = diagnostics.WriteString("configuration updated: " + credentialSource(config.Model.CredentialRef))
	for name, data := range map[string][]byte{"yaml config": yamlData, "event": eventData, "diagnostics": diagnostics.Bytes(), "credential reference": []byte(config.Model.CredentialRef)} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("secret leaked into %s", name)
		}
	}
}

func yamlMarshal(value any) ([]byte, error) { return yaml.Marshal(value) }
