package skills

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type selectorFunc func(context.Context, []Candidate) (Selection, error)

func (f selectorFunc) Select(c context.Context, v []Candidate) (Selection, error) { return f(c, v) }
func TestSelectLazyBodyAndLinkedReference(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "jwt-auth", "Read [JWT variants](references/jwt.md) only when issuer validation is observed.")
	if err := os.MkdirAll(filepath.Join(root, "jwt-auth", "references"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "jwt-auth", "references", "jwt.md"), []byte("issuer controls"), 0600); err != nil {
		t.Fatal(err)
	}
	index := &Index{Sources: []Source{LocalSource{ID: "fixture", Root: root}}}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	candidates, _ := index.Search(context.Background(), Query{Objective: "JWT authentication issuer"})
	loaded, err := index.SelectAndLoad(context.Background(), candidates, selectorFunc(func(_ context.Context, got []Candidate) (Selection, error) {
		if len(got) > 8 {
			t.Fatal("unbounded")
		}
		return Selection{Name: "jwt-auth", Relevance: "JWT issuer was observed"}, nil
	}), []string{"references/jwt.md"}, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded.References["references/jwt.md"]) != "issuer controls" || len(loaded.Instructions) == 0 {
		t.Fatalf("loaded=%#v", loaded)
	}
}
func TestSelectionRejectsOutsideCandidateAndUnsafeResources(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "skill", "See [safe](references/safe.md).")
	if err := os.MkdirAll(filepath.Join(root, "skill", "references"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "skill", "references", "safe.md"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	index := &Index{Sources: []Source{LocalSource{ID: "fixture", Root: root}}}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	candidate := Candidate{Metadata: index.Documents()[0].Metadata}
	for _, test := range []struct {
		name      string
		selection Selection
		refs      []string
		budget    int
	}{{"outside", Selection{Name: "other", Relevance: "reason"}, nil, 1000}, {"no reason", Selection{Name: "skill"}, nil, 1000}, {"escape", Selection{Name: "skill", Relevance: "reason"}, []string{"../secret"}, 1000}, {"not linked", Selection{Name: "skill", Relevance: "reason"}, []string{"references/other.md"}, 1000}, {"budget", Selection{Name: "skill", Relevance: "reason"}, nil, 2}} {
		t.Run(test.name, func(t *testing.T) {
			_, err := index.SelectAndLoad(context.Background(), []Candidate{candidate}, selectorFunc(func(context.Context, []Candidate) (Selection, error) { return test.selection, nil }), test.refs, test.budget)
			if err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
func TestNoCandidatesDoesNotCallModel(t *testing.T) {
	_, err := (&Index{}).SelectAndLoad(context.Background(), nil, selectorFunc(func(context.Context, []Candidate) (Selection, error) {
		t.Fatal("selector called")
		return Selection{}, errors.New("called")
	}), nil, 0)
	if err == nil {
		t.Fatal("expected no-match")
	}
}
