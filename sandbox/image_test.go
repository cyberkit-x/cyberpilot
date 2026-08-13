package sandbox

import (
	"os"
	"strings"
	"testing"
)

func TestSandboxDefinitionSecurityContract(t *testing.T) {
	data, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	definition := string(data)
	for _, required := range []string{"@sha256:", "ca-certificates", "curl", "python3", "USER 65532:65532", "WORKDIR /workspace", "cyberpilot-runner"} {
		if !strings.Contains(definition, required) {
			t.Fatalf("Dockerfile missing %q", required)
		}
	}
	for _, prohibited := range []string{"docker.sock", "podman.sock", "USER root", "sudo"} {
		if strings.Contains(definition, prohibited) {
			t.Fatalf("Dockerfile contains prohibited %q", prohibited)
		}
	}
}
