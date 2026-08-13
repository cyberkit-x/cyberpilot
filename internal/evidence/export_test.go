package evidence

import (
	"encoding/json"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"strings"
	"testing"
	"time"
)

func TestSessionResultExportIntegrityAndCoverage(t *testing.T) {
	finding := completeFinding()
	finding.State = domain.FindingVerified
	result := SessionResult{Session: domain.Session{ID: finding.SessionID, Objective: "API key=secret-value", Targets: []string{"https://api.example.com"}}, AssessedScope: []string{"https://api.example.com"}, Findings: []domain.Finding{finding}, CoverageGaps: []domain.CoverageGap{{ID: domain.MustNewID(), SessionID: finding.SessionID, Goal: "admin", Reason: "blocked"}}, Artifacts: []domain.ArtifactRef{{ID: domain.MustNewID(), SessionID: finding.SessionID, SHA256: strings.Repeat("a", 64)}}, Limitations: []string{"no browser"}, ExportedAt: time.Now().UTC()}
	export, err := ExportResult(result, NewRedactor())
	if err != nil || len(export.SHA256) != 64 || strings.Contains(string(export.Data), "secret-value") {
		t.Fatalf("export=%#v err=%v", export, err)
	}
	var decoded SessionResult
	if err := json.Unmarshal(export.Data, &decoded); err != nil || len(decoded.Findings) != 1 || len(decoded.CoverageGaps) != 1 || decoded.Findings[0].Impact == "" || len(decoded.Findings[0].Reproduction) == 0 {
		t.Fatalf("decoded=%#v err=%v", decoded, err)
	}
}
