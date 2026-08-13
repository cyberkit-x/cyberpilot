package domain

import "time"

type ActionRecord struct {
	Proposal      ActionProposal `json:"proposal"`
	State         ActionState    `json:"state"`
	DecisionID    ID             `json:"decision_id,omitempty"`
	ResultSummary string         `json:"result_summary,omitempty"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
type Lead struct {
	ID           ID        `json:"id"`
	SessionID    ID        `json:"session_id"`
	HypothesisID ID        `json:"hypothesis_id"`
	Title        string    `json:"title"`
	Reason       string    `json:"reason"`
	EvidenceIDs  []ID      `json:"evidence_ids"`
	CreatedAt    time.Time `json:"created_at"`
}
type CoverageGap struct {
	ID        ID        `json:"id"`
	SessionID ID        `json:"session_id"`
	Goal      string    `json:"goal"`
	Reason    string    `json:"reason"`
	Blocked   bool      `json:"blocked"`
	CreatedAt time.Time `json:"created_at"`
}
