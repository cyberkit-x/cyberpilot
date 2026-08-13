package configuration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() Config {
	return Config{Version: SchemaVersion, Model: ModelProfile{Provider: "openai-compatible", Model: "test-model", BaseURL: "http://127.0.0.1:8080/v1", CredentialRef: "env:CYBERPILOT_TEST_KEY"}, Runner: RunnerProfile{Provider: "docker", Connection: "default"}}
}

func TestFileStoreAtomicRoundTripAndRedaction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "config.yaml")
	store := FileStore{Path: path}
	config := validConfig()
	if err := store.Save(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != config {
		t.Fatalf("got %#v want %#v", got, config)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "actual-secret") {
		t.Fatal("secret appeared in config")
	}
	redacted := Redact(config)
	if redacted.Model.Credential != "env:***" {
		t.Fatalf("credential = %q", redacted.Model.Credential)
	}
}

func TestInvalidUpdatePreservesLastValidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store := FileStore{Path: path}
	good := validConfig()
	if err := store.Save(context.Background(), good); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.Runner.Provider = "host"
	if err := store.Save(context.Background(), bad); err == nil {
		t.Fatal("expected validation failure")
	}
	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != good {
		t.Fatal("valid config changed")
	}
}
