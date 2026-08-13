package skills

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte("---\nname: " + name + "\ndescription: Use when an API needs focused authorization testing.\nlicense: MIT\n---\n" + body)
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestLocalRefreshAndContentChangeDropsTested(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "object-access", "first")
	index := &Index{Sources: []Source{LocalSource{ID: "fixture", Root: root}}, statuses: map[string]Metadata{"object-access": {Hash: "old", Status: Tested}}}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	docs := index.Documents()
	if len(docs) != 1 || docs[0].Metadata.Status != Compatible {
		t.Fatalf("docs=%#v", docs)
	}
	writeSkill(t, root, "object-access", "second")
	previous := index.statuses["object-access"]
	previous.Status = Tested
	index.statuses["object-access"] = previous
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if index.Documents()[0].Metadata.Status != Compatible {
		t.Fatal("changed content inherited Tested")
	}
}
func TestPinnedGitSource(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "test@example.invalid"}, {"config", "user.name", "Test"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, output)
		}
	}
	writeSkill(t, root, "jwt-auth", "body")
	cmd := exec.Command("git", "-C", root, "add", ".")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "-C", root, "commit", "-m", "fixture")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v %s", err, output)
	}
	output, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := string(output[:len(output)-1])
	index := &Index{Sources: []Source{GitSource{ID: "fixture", Checkout: root, Commit: commit}}}
	if err := index.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(index.Documents()) != 1 {
		t.Fatal("skill not loaded")
	}
	index.Sources = []Source{GitSource{ID: "fixture", Checkout: root, Commit: "deadbeef"}}
	if err := index.Refresh(context.Background()); err == nil {
		t.Fatal("unpinned checkout accepted")
	}
}
