//go:build !windows

package runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/cli"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/evidence/artifact"
	"github.com/cyberkit-x/cyberpilot/internal/model"
	"github.com/cyberkit-x/cyberpilot/internal/platform"
	"github.com/cyberkit-x/cyberpilot/internal/rpc"
	"github.com/cyberkit-x/cyberpilot/internal/runner"
	"github.com/cyberkit-x/cyberpilot/internal/runner/oci"
	localruntime "github.com/cyberkit-x/cyberpilot/internal/runtime"
	"github.com/cyberkit-x/cyberpilot/internal/service"
	"github.com/cyberkit-x/cyberpilot/internal/skills"
	store "github.com/cyberkit-x/cyberpilot/internal/storage/sqlite"
	"github.com/cyberkit-x/cyberpilot/internal/tui"
)

type acceptanceModel struct {
	mu       sync.Mutex
	turn     int
	sawSkill bool
}

func (*acceptanceModel) Probe(context.Context) (model.CapabilityReport, error) {
	return model.CapabilityReport{Model: "acceptance", ToolCalling: true, StructuredOutput: true}, nil
}

func (m *acceptanceModel) Turn(_ context.Context, request model.TurnRequest) (model.TurnResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	combined := ""
	for _, message := range request.Messages {
		combined += message.Content
	}
	if strings.Contains(combined, "Validate Object Authorization") {
		m.sawSkill = true
	}
	m.turn++
	var contextValue struct {
		Targets      []string             `json:"confirmed_targets"`
		Observations []domain.Observation `json:"recent_observations"`
	}
	for _, message := range request.Messages {
		if message.Role == "user" {
			_ = json.Unmarshal([]byte(message.Content), &contextValue)
		}
	}
	if len(contextValue.Targets) != 1 {
		return model.TurnResult{}, fmt.Errorf("confirmed target missing from model context")
	}
	sessionID := request.SessionID
	switch m.turn {
	case 1:
		return model.TurnResult{Proposals: []domain.ActionProposal{{ID: domain.MustNewID(), SessionID: sessionID, HypothesisID: domain.MustNewID(), Target: contextValue.Targets[0], Purpose: "capture owner object response", Capability: "http.request", Arguments: json.RawMessage(`{"method":"GET"}`), Risk: domain.Risk{Level: "low", TrafficClass: "single"}, ExpectedEvidence: []string{"protected object response"}, TimeoutSeconds: 10}}}, nil
	case 2:
		return model.TurnResult{Proposals: []domain.ActionProposal{{ID: domain.MustNewID(), SessionID: sessionID, HypothesisID: domain.MustNewID(), Target: contextValue.Targets[0], Purpose: "record deterministic control comparison in sandbox", Capability: "runner.exec", Arguments: json.RawMessage(`{"executable":"sh","arguments":["-c","printf 'control identity denied: 403'"],"directory":"/workspace"}`), Risk: domain.Risk{Level: "low", TrafficClass: "none"}, ExpectedEvidence: []string{"negative control"}, TimeoutSeconds: 10}}}, nil
	default:
		if len(contextValue.Observations) < 2 {
			return model.TurnResult{}, fmt.Errorf("observations were not replanned into context")
		}
		finding := domain.Finding{ID: domain.MustNewID(), SessionID: sessionID, Title: "Cross-identity object access", Target: contextValue.Targets[0], Prerequisites: []string{"two authorized test identities"}, EvidenceIDs: []domain.ID{contextValue.Observations[0].ID}, ControlEvidence: []domain.ID{contextValue.Observations[1].ID}, Impact: "another identity's protected object was disclosed", Reproduction: []string{"request the object as its owner", "request the same object with the control identity"}, Provenance: domain.Provenance{Model: "acceptance", SkillName: "validate-object-authorization", Tool: "http.request"}}
		return model.TurnResult{Findings: []domain.FindingProposal{{Finding: finding, Signals: []string{"reproduction", "impact", "control"}, EvidenceOnly: true}}, Complete: true, Reason: "authorization goal verified with positive and control evidence"}, nil
	}
}

func acceptancePaths(root string) platform.Paths {
	return platform.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), ConfigFile: filepath.Join(root, "config", "config.yaml"), DatabaseFile: filepath.Join(root, "data", "state.db"), ArtifactsDir: filepath.Join(root, "data", "artifacts"), RuntimeDir: filepath.Join(root, "data", "run"), LockFile: filepath.Join(root, "data", "run", "daemon.lock"), Endpoint: filepath.Join(root, "data", "run", "rpc.sock")}
}

func TestFullOCIExecEvidenceRestartAndTUIAcceptance(t *testing.T) {
	provider := os.Getenv("CYBERPILOT_ACCEPTANCE_PROVIDER")
	if provider != "docker" && provider != "podman" {
		t.Skip("set CYBERPILOT_ACCEPTANCE_PROVIDER=docker or podman")
	}
	image := os.Getenv("CYBERPILOT_CONFORMANCE_IMAGE")
	if image == "" {
		t.Fatal("CYBERPILOT_CONFORMANCE_IMAGE is required")
	}
	target := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"owner":"alice","secret":"fixture-private"}`))
	}))
	defer target.Close()
	root, err := os.MkdirTemp("/tmp", "cyberpilot-acceptance-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	paths := acceptancePaths(root)
	artifacts, err := artifact.Open(paths.ArtifactsDir)
	if err != nil {
		t.Fatal(err)
	}
	fakeModel := &acceptanceModel{}
	factory := func(sessions *service.SessionService, _ *store.Store) localruntime.SessionWorker {
		return &localruntime.Worker{Sessions: sessions, Model: fakeModel, Skills: &skills.Index{Sources: []skills.Source{skills.BundledSource{}}}, Runner: &runner.Manager{Provider: oci.New(provider, provider, "default"), WorkspaceRoot: filepath.Join(paths.DataDir, "workspaces")}, Artifacts: artifacts, Image: image, MaxActions: 5}
	}
	daemon, err := localruntime.NewDaemonWithWorker(paths, factory)
	if err != nil {
		t.Fatal(err)
	}
	serveCtx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- daemon.Serve(serveCtx) }()
	client := acceptanceClient(t, paths)
	defer client.Close()
	var stdout, stderr bytes.Buffer
	command := cli.ExecCommand{Input: &bytes.Buffer{}, Output: &stdout, Error: &stderr, Client: client, PollInterval: 10 * time.Millisecond}
	err = command.Run(context.Background(), []string{"--json", "Assess " + target.URL + " for authorized object access issues"})
	var exit *cli.ExitError
	if err == nil || !strings.Contains(err.Error(), "verified findings") || !asExit(err, &exit) || exit.Code != 1 {
		t.Fatalf("exec err=%v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	var result cli.ExecResult
	if json.Unmarshal(stdout.Bytes(), &result) != nil || result.State != domain.SessionCompleted || result.VerifiedFindings != 1 {
		t.Fatalf("result=%#v stdout=%s", result, stdout.String())
	}
	if !fakeModel.sawSkill {
		t.Fatal("dynamically selected Skill body never reached planning context")
	}
	stop()
	_ = client.Close()
	_ = daemon.Close()
	<-done

	restarted, err := localruntime.NewDaemon(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	restartCtx, restartStop := context.WithCancel(context.Background())
	defer restartStop()
	go restarted.Serve(restartCtx)
	restartClient := acceptanceClient(t, paths)
	defer restartClient.Close()
	view := tui.New(restartClient)
	if err := view.Refresh(context.Background()); err != nil || len(view.Sessions) != 1 {
		t.Fatalf("sessions=%v err=%v", view.Sessions, err)
	}
	view.Current = view.Sessions[0]
	view.Screen = tui.SessionDetail
	if err := view.SyncEvents(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(view.Findings) != 1 || !strings.Contains(view.View(), "Verified findings: 1") {
		t.Fatalf("TUI did not reconstruct finding: %s", view.View())
	}
}

func acceptanceClient(t *testing.T, paths platform.Paths) *rpc.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		client, err := localruntime.EnsureClient(ctx, paths, func(context.Context, platform.Paths) error { return nil })
		if err == nil {
			return client
		}
		select {
		case <-ctx.Done():
			t.Fatal(err)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func asExit(err error, target **cli.ExitError) bool {
	value, ok := err.(*cli.ExitError)
	if ok {
		*target = value
	}
	return ok
}
