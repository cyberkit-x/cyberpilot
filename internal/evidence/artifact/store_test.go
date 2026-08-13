package artifact

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestPutOpenIsolationAndIntegrity(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	owner, other := domain.MustNewID(), domain.MustNewID()
	ref, err := store.Put(context.Background(), owner, "text/plain", true, []byte("secret evidence"))
	if err != nil {
		t.Fatal(err)
	}
	data, got, err := store.Open(context.Background(), owner, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "secret evidence" || got != ref {
		t.Fatalf("unexpected artifact %#v %q", got, data)
	}
	if _, _, err := store.Open(context.Background(), other, ref.ID); err == nil {
		t.Fatal("expected cross-session denial")
	}
	object := filepath.Join(store.root, "objects", ref.SHA256[:2], ref.SHA256)
	if err := os.WriteFile(object, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Open(context.Background(), owner, ref.ID); err == nil {
		t.Fatal("expected integrity failure")
	}
}

func TestMetadataSurvivesReopenAndDeduplicatesObjects(t *testing.T) {
	root := t.TempDir()
	sessionID := domain.MustNewID()
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.Put(context.Background(), sessionID, "text/plain", false, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Put(context.Background(), sessionID, "text/plain", false, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	if first.SHA256 != second.SHA256 || first.ID == second.ID {
		t.Fatal("expected shared object and distinct references")
	}
	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reopened.Open(context.Background(), sessionID, first.ID); err != nil {
		t.Fatal(err)
	}
}
