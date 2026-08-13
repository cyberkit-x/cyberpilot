package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestManagerPersistentWorkspaceRecoveryAndBoundedCapture(t *testing.T) {
	fake := NewFake()
	session := domain.MustNewID()
	fake.Behavior = FakeBehavior{Stdout: strings.Repeat("o", 20), Stderr: strings.Repeat("e", 20)}
	manager := &Manager{Provider: fake, WorkspaceRoot: t.TempDir()}
	spec := SandboxSpec{SessionID: session, Image: "fixture"}
	if err := manager.Ensure(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	recovered, err := manager.Recover(context.Background(), session)
	if err != nil || recovered.Workspace == "" {
		t.Fatalf("recovered=%#v err=%v", recovered, err)
	}
	capture, err := manager.Execute(context.Background(), session, Command{OutputLimit: 5})
	if err != nil || string(capture.Stdout) != "ooooo" || !capture.StdoutTruncated || string(capture.Stderr) != "eeeee" || !capture.StderrTruncated {
		t.Fatalf("capture=%#v err=%v", capture, err)
	}
	if err := manager.Cleanup(context.Background(), session, true); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Recover(context.Background(), session); err != nil {
		t.Fatal("retained sandbox lost")
	}
}
func TestManagerFailsWithoutProvider(t *testing.T) {
	if err := (&Manager{}).Ensure(context.Background(), SandboxSpec{SessionID: domain.MustNewID()}); err == nil {
		t.Fatal("host fallback occurred")
	}
}

func TestManagerKeepsParentPrivateAndMakesLeafWritableForContainerUID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private-workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	id := domain.MustNewID()
	manager := &Manager{Provider: NewFake(), WorkspaceRoot: root}
	if err := manager.Ensure(context.Background(), SandboxSpec{SessionID: id, Image: "fixture"}); err != nil {
		t.Fatal(err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	leafInfo, err := os.Stat(filepath.Join(root, string(id)))
	if err != nil {
		t.Fatal(err)
	}
	if rootInfo.Mode().Perm() != 0o700 || leafInfo.Mode().Perm() != 0o777 {
		t.Fatalf("root=%o leaf=%o", rootInfo.Mode().Perm(), leafInfo.Mode().Perm())
	}
}
