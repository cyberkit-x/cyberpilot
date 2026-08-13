package model

import (
	"strings"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/scheduler"
	"github.com/cyberkit-x/cyberpilot/internal/skills"
)

func TestAssembleContextIncludesPlanningInputsAndWithholdsProtectedRaw(t *testing.T) {
	const secret = "protected-response-secret"
	session := domain.Session{ID: domain.MustNewID(), Objective: "Assess authorization", Targets: []string{"http://127.0.0.1"}, Goals: []string{"IDOR", "auth"}, Constraints: []string{"read-only"}, Instructions: "prioritize object access", ScopeConfirmed: true}
	observation := domain.Observation{ID: domain.MustNewID(), SessionID: session.ID, ActionID: domain.MustNewID(), Summary: "JWT observed", ObservedAt: time.Now().UTC()}
	messages, err := AssembleContext(ContextInput{Session: session, Observations: []domain.Observation{observation}, Skills: []skills.Metadata{{Name: "jwt-auth", Description: "Analyze JWT authentication", License: "Apache-2.0"}}, Artifacts: []ArtifactSummary{{Reference: "artifact:1", MediaType: "application/json", Protected: true, Summary: secret, Raw: []byte(secret)}}, Budget: scheduler.Budget{MaxActions: 10}, MaxBytes: 32 << 10})
	if err != nil {
		t.Fatal(err)
	}
	combined := messages[0].Content + messages[1].Content
	for _, expected := range []string{"Assess authorization", "IDOR", "JWT observed", "jwt-auth", "artifact:1", "withheld", "max_actions"} {
		if !strings.Contains(strings.ToLower(combined), strings.ToLower(expected)) {
			t.Fatalf("missing %q in %s", expected, combined)
		}
	}
	if strings.Contains(combined, secret) {
		t.Fatal("protected raw evidence entered model context")
	}
}

func TestAssembleContextBoundsOptionalHistory(t *testing.T) {
	session := domain.Session{Objective: "A", Targets: []string{"http://127.0.0.1"}, Goals: []string{"test"}}
	observations := make([]domain.Observation, 20)
	for index := range observations {
		observations[index] = domain.Observation{Summary: strings.Repeat("evidence ", 100), ObservedAt: time.Now().Add(time.Duration(index) * time.Second)}
	}
	messages, err := AssembleContext(ContextInput{Session: session, Observations: observations, MaxBytes: 1800})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages[1].Content) > 1800 {
		t.Fatalf("context length=%d", len(messages[1].Content))
	}
	if !strings.Contains(messages[1].Content, "evidence") {
		t.Fatal("all recent observations were removed")
	}
}
