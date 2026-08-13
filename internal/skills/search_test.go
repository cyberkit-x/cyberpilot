package skills

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func searchIndex() *Index {
	return &Index{documents: map[string]Document{
		"jwt-auth":         {Metadata: Metadata{Name: "jwt-auth", Description: "Use when bearer JWT tokens require authentication claim validation", Domains: []string{"api"}, Intents: []string{"authentication"}}},
		"object-access":    {Metadata: Metadata{Name: "object-access", Description: "Use when object identifiers require authorization owner control validation", Domains: []string{"web", "api"}, Intents: []string{"idor", "authorization"}}},
		"active-directory": {Metadata: Metadata{Name: "active-directory", Description: "Use for Windows domain Kerberos and LDAP investigation", Domains: []string{"ad"}}},
	}}
}

func TestLexicalSearchPositiveObservationAndHypothesis(t *testing.T) {
	index := searchIndex()
	session := domain.MustNewID()
	candidates, err := index.Search(context.Background(), Query{Objective: "Assess API access", Observations: []domain.Observation{{SessionID: session, Summary: "Bearer JWT token observed", ObservedAt: time.Now()}}, Hypotheses: []domain.Hypothesis{{Claim: "JWT issuer claim may be trusted incorrectly"}}})
	if err != nil || len(candidates) == 0 || candidates[0].Metadata.Name != "jwt-auth" || !strings.Contains(candidates[0].Reason, "jwt") {
		t.Fatalf("candidates=%#v err=%v", candidates, err)
	}
}

func TestLexicalSearchAuthorizationAndUnrelatedNegative(t *testing.T) {
	index := searchIndex()
	candidates, _ := index.Search(context.Background(), Query{Objective: "Validate IDOR authorization for object owner identifiers"})
	if len(candidates) == 0 || candidates[0].Metadata.Name != "object-access" {
		t.Fatalf("candidates=%#v", candidates)
	}
	candidates, _ = index.Search(context.Background(), Query{Objective: "Reverse an Android mobile binary for a CTF cryptography challenge"})
	if len(candidates) != 0 {
		t.Fatalf("unrelated candidates=%#v", candidates)
	}
}

func TestLexicalSearchNoMatchAndLimit(t *testing.T) {
	index := searchIndex()
	candidates, _ := index.Search(context.Background(), Query{Objective: "Investigate quantum lattice mathematics"})
	if len(candidates) != 0 {
		t.Fatalf("candidates=%#v", candidates)
	}
	candidates, _ = index.Search(context.Background(), Query{Objective: "API authentication authorization", Limit: 1})
	if len(candidates) != 1 {
		t.Fatalf("candidates=%#v", candidates)
	}
}
