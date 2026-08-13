package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"time"
)

type SessionResult struct {
	SchemaVersion int                  `json:"schema_version"`
	Session       domain.Session       `json:"session"`
	AssessedScope []string             `json:"assessed_scope"`
	Findings      []domain.Finding     `json:"verified_findings"`
	Leads         []domain.Lead        `json:"leads"`
	Rejected      []domain.Hypothesis  `json:"rejected_hypotheses"`
	Blocked       []domain.Hypothesis  `json:"blocked_hypotheses"`
	CoverageGaps  []domain.CoverageGap `json:"coverage_gaps"`
	Artifacts     []domain.ArtifactRef `json:"artifacts"`
	Limitations   []string             `json:"limitations"`
	ExportedAt    time.Time            `json:"exported_at"`
}
type Export struct {
	SHA256 string `json:"sha256"`
	Data   []byte `json:"data"`
}

func ExportResult(result SessionResult, redactor Redactor) (Export, error) {
	result.SchemaVersion = 1
	raw, err := json.Marshal(result)
	if err != nil {
		return Export{}, err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return Export{}, err
	}
	redactJSONStrings(value, redactor)
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return Export{}, err
	}
	sum := sha256.Sum256(data)
	return Export{SHA256: hex.EncodeToString(sum[:]), Data: data}, nil
}

func redactJSONStrings(value any, redactor Redactor) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if text, ok := child.(string); ok {
				typed[key] = redactor.String(text)
			} else {
				redactJSONStrings(child, redactor)
			}
		}
	case []any:
		for index, child := range typed {
			if text, ok := child.(string); ok {
				typed[index] = redactor.String(text)
			} else {
				redactJSONStrings(child, redactor)
			}
		}
	}
}
