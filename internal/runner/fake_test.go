package runner

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
)

func TestFakeRunnerLifecycleStreamingAndArtifacts(t *testing.T) {
	fake := NewFake()
	session := domain.MustNewID()
	artifact := domain.ArtifactRef{ID: domain.MustNewID(), SessionID: session}
	fake.Behavior = FakeBehavior{Stdout: "out", Stderr: "err", ExitCode: 7, Artifact: artifact}
	spec := SandboxSpec{SessionID: session, Image: "fixture"}
	if err := fake.Probe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := fake.Create(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	if err := fake.Start(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	result, err := fake.Exec(context.Background(), session, Command{}, &stdout, &stderr)
	if err != nil || stdout.String() != "out" || stderr.String() != "err" || result.ExitCode != 7 || len(result.Artifacts) != 1 {
		t.Fatalf("result=%#v stdout=%q stderr=%q err=%v", result, stdout.String(), stderr.String(), err)
	}
	if got, _ := fake.Inspect(context.Background(), session); got.SessionID != session {
		t.Fatalf("spec=%#v", got)
	}
	if err := fake.Stop(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if err := fake.Remove(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if _, err := fake.Inspect(context.Background(), session); err == nil {
		t.Fatal("removed sandbox found")
	}
}
func TestFakeRunnerTimeoutCancellationAndProviderFailure(t *testing.T) {
	fake := NewFake()
	fake.Available = false
	if err := fake.Probe(context.Background()); err == nil {
		t.Fatal("unavailable provider probed")
	}
	fake.Available = true
	session := domain.MustNewID()
	_ = fake.Create(context.Background(), SandboxSpec{SessionID: session})
	_ = fake.Start(context.Background(), session)
	fake.Behavior = FakeBehavior{Delay: time.Hour}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	result, err := fake.Exec(ctx, session, Command{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	if !errors.Is(err, context.DeadlineExceeded) || !result.TimedOut {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	result, err = fake.Exec(ctx, session, Command{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	if !errors.Is(err, context.Canceled) || !result.Cancelled {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
