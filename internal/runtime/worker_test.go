package runtime

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/service"
	store "github.com/cyberkit-x/cyberpilot/internal/storage/sqlite"
)

func TestFindingCannotReferenceInventedEvidence(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sessions := service.NewSessionService(db)
	session, err := sessions.Create(context.Background(), service.CreateSessionInput{Objective: "Assess local API", Targets: []string{"http://127.0.0.1"}, Goals: []string{"authorization"}})
	if err != nil {
		t.Fatal(err)
	}
	observation := domain.Observation{ID: domain.MustNewID(), SessionID: session.ID, ActionID: domain.MustNewID(), Summary: "real evidence", ObservedAt: time.Now().UTC()}
	finding := domain.Finding{ID: domain.MustNewID(), SessionID: session.ID, Title: "invented", Target: session.Targets[0], Prerequisites: []string{"test identity"}, EvidenceIDs: []domain.ID{domain.MustNewID()}, ControlEvidence: []domain.ID{observation.ID}, Impact: "impact", Reproduction: []string{"step"}, Provenance: domain.Provenance{Tool: "http.request"}}
	err = (&Worker{Sessions: sessions}).recordFinding(context.Background(), session, []domain.Observation{observation}, domain.FindingProposal{Finding: finding, Signals: []string{"reproduction", "impact", "control"}, EvidenceOnly: true})
	if err == nil || !strings.Contains(err.Error(), "not recorded") {
		t.Fatalf("err=%v", err)
	}
}
