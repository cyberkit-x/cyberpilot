package runner

import (
	"context"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"strings"
	"testing"
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
