package domain

import (
	"encoding/json"
	"time"
)

const EventSchemaVersion = 1

type Event struct {
	SchemaVersion int             `json:"schema_version"`
	ID            ID              `json:"id"`
	SessionID     ID              `json:"session_id"`
	Sequence      uint64          `json:"sequence"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

func NewEvent(sessionID ID, sequence uint64, eventType string, occurredAt time.Time, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	id, err := NewID()
	if err != nil {
		return Event{}, err
	}
	return Event{
		SchemaVersion: EventSchemaVersion,
		ID:            id,
		SessionID:     sessionID,
		Sequence:      sequence,
		Type:          eventType,
		OccurredAt:    Timestamp(occurredAt),
		Payload:       data,
	}, nil
}

type Session struct {
	ID             ID           `json:"id"`
	Name           string       `json:"name"`
	Objective      string       `json:"objective"`
	Targets        []string     `json:"targets"`
	Goals          []string     `json:"goals"`
	Constraints    []string     `json:"constraints,omitempty"`
	Instructions   string       `json:"instructions,omitempty"`
	ScopeConfirmed bool         `json:"scope_confirmed"`
	State          SessionState `json:"state"`
	TerminalReason string       `json:"terminal_reason,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type SessionCreatedPayload struct {
	Session Session `json:"session"`
}

type SessionStateChangedPayload struct {
	From   SessionState `json:"from"`
	To     SessionState `json:"to"`
	Reason string       `json:"reason,omitempty"`
}

type SessionInstructionsUpdatedPayload struct {
	Instructions string `json:"instructions"`
}

type SessionScopeUpdatedPayload struct {
	Targets   []string `json:"targets"`
	Confirmed bool     `json:"confirmed"`
}

type Hypothesis struct {
	ID                    ID              `json:"id"`
	SessionID             ID              `json:"session_id"`
	Claim                 string          `json:"claim"`
	GoalIDs               []string        `json:"goal_ids,omitempty"`
	AssetIDs              []string        `json:"asset_ids,omitempty"`
	SupportingEvidenceIDs []ID            `json:"supporting_evidence_ids,omitempty"`
	ContradictingIDs      []ID            `json:"contradicting_evidence_ids,omitempty"`
	Priority              int             `json:"priority"`
	State                 HypothesisState `json:"state"`
}

type Risk struct {
	Level               string `json:"level"`
	UsesCredentials     bool   `json:"uses_credentials"`
	ReadsSensitive      bool   `json:"reads_sensitive_data"`
	ChangesState        bool   `json:"changes_state"`
	AffectsAvailability bool   `json:"affects_availability"`
	TrafficClass        string `json:"traffic_class"`
}

type ActionProposal struct {
	ID               ID              `json:"id"`
	SessionID        ID              `json:"session_id"`
	HypothesisID     ID              `json:"hypothesis_id"`
	Target           string          `json:"target"`
	Purpose          string          `json:"purpose"`
	Capability       string          `json:"capability"`
	Arguments        json.RawMessage `json:"arguments"`
	Risk             Risk            `json:"risk"`
	ExpectedEvidence []string        `json:"expected_evidence"`
	SideEffects      []string        `json:"side_effects,omitempty"`
	TimeoutSeconds   int             `json:"timeout_seconds"`
}

type PolicyDecision string

const (
	PolicyAllow PolicyDecision = "allow"
	PolicyAsk   PolicyDecision = "ask"
	PolicyDeny  PolicyDecision = "deny"
)

type Decision struct {
	ID        ID             `json:"id"`
	ActionID  ID             `json:"action_id"`
	Decision  PolicyDecision `json:"decision"`
	Basis     []string       `json:"basis"`
	Limits    Limits         `json:"limits"`
	DecidedAt time.Time      `json:"decided_at"`
}

type Limits struct {
	MaxRequests int      `json:"max_requests,omitempty"`
	MaxDuration int      `json:"max_duration_seconds,omitempty"`
	Targets     []string `json:"targets,omitempty"`
}

type Approval struct {
	ID        ID            `json:"id"`
	ActionID  ID            `json:"action_id"`
	State     ApprovalState `json:"state"`
	Requested Limits        `json:"requested_limits"`
	Approved  Limits        `json:"approved_limits"`
	Reason    string        `json:"reason"`
	ExpiresAt *time.Time    `json:"expires_at,omitempty"`
}

type ArtifactRef struct {
	ID        ID     `json:"id"`
	SessionID ID     `json:"session_id"`
	SHA256    string `json:"sha256"`
	MediaType string `json:"media_type"`
	Size      int64  `json:"size"`
	Protected bool   `json:"protected"`
}

type Observation struct {
	ID          ID         `json:"id"`
	SessionID   ID         `json:"session_id"`
	ActionID    ID         `json:"action_id"`
	Summary     string     `json:"summary"`
	ArtifactIDs []ID       `json:"artifact_ids,omitempty"`
	Provenance  Provenance `json:"provenance"`
	ObservedAt  time.Time  `json:"observed_at"`
}

type Provenance struct {
	Model     string `json:"model,omitempty"`
	SkillName string `json:"skill_name,omitempty"`
	SkillHash string `json:"skill_hash,omitempty"`
	Tool      string `json:"tool,omitempty"`
}

type Finding struct {
	ID              ID           `json:"id"`
	SessionID       ID           `json:"session_id"`
	Title           string       `json:"title"`
	State           FindingState `json:"state"`
	Target          string       `json:"target"`
	Prerequisites   []string     `json:"prerequisites"`
	EvidenceIDs     []ID         `json:"evidence_ids"`
	ControlEvidence []ID         `json:"control_evidence_ids"`
	Impact          string       `json:"impact"`
	Reproduction    []string     `json:"reproduction"`
	Limitations     []string     `json:"limitations,omitempty"`
	Provenance      Provenance   `json:"provenance"`
}

// FindingProposal is the model's structured request to promote a lead. The
// runtime still applies the deterministic evidence gate before persistence.
type FindingProposal struct {
	Finding      Finding  `json:"finding"`
	Signals      []string `json:"signals"`
	EvidenceOnly bool     `json:"evidence_only"`
}
