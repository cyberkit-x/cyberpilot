package skills

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func TestV1SkillFixturesAreLicensedValidAndRoutable(t *testing.T) {
	_, current, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(current), "..", "..", "testdata", "skills")
	index := &Index{Sources: []Source{LocalSource{ID: "v1-fixtures", Root: root}}}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if errors := index.Errors(); len(errors) != 0 {
		t.Fatalf("fixture errors=%v", errors)
	}
	if got := len(index.Documents()); got != 5 {
		t.Fatalf("loaded %d fixtures", got)
	}
	queries := map[string]string{"IDOR object owner authorization": "validate-object-authorization", "JWT bearer issuer authentication": "validate-jwt-api-auth", "server URL fetch SSRF redirect": "validate-scoped-ssrf", "file upload path exposure": "validate-file-boundaries", "promote lead evidence verification": "verify-security-finding"}
	for query, want := range queries {
		candidates, err := index.Search(context.Background(), Query{Objective: query})
		if err != nil || len(candidates) == 0 || candidates[0].Metadata.Name != want {
			t.Fatalf("query=%q candidates=%#v err=%v", query, candidates, err)
		}
	}
}
