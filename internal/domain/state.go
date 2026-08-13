package domain

import "fmt"

type SessionState string

const (
	SessionCreated    SessionState = "created"
	SessionRunning    SessionState = "running"
	SessionNeedsInput SessionState = "needs-input"
	SessionCompleted  SessionState = "completed"
	SessionFailed     SessionState = "failed"
	SessionCancelled  SessionState = "cancelled"
	SessionBlocked    SessionState = "blocked"
)

type HypothesisState string

const (
	HypothesisProposed  HypothesisState = "proposed"
	HypothesisTesting   HypothesisState = "testing"
	HypothesisSupported HypothesisState = "supported"
	HypothesisRejected  HypothesisState = "rejected"
	HypothesisBlocked   HypothesisState = "blocked"
)

type ActionState string

const (
	ActionProposed  ActionState = "proposed"
	ActionApproved  ActionState = "approved"
	ActionDenied    ActionState = "denied"
	ActionRunning   ActionState = "running"
	ActionSucceeded ActionState = "succeeded"
	ActionFailed    ActionState = "failed"
	ActionCancelled ActionState = "cancelled"
	ActionTimedOut  ActionState = "timed-out"
	ActionUncertain ActionState = "uncertain"
)

type ApprovalState string

const (
	ApprovalPending ApprovalState = "pending"
	ApprovalAllowed ApprovalState = "allowed"
	ApprovalDenied  ApprovalState = "denied"
	ApprovalExpired ApprovalState = "expired"
	ApprovalRevoked ApprovalState = "revoked"
)

type FindingState string

const (
	FindingLead     FindingState = "lead"
	FindingVerified FindingState = "verified"
	FindingRejected FindingState = "rejected"
)

func ValidateSessionTransition(from, to SessionState) error {
	return validateTransition(string(from), string(to), map[SessionState]map[SessionState]bool{
		SessionCreated:    {SessionRunning: true, SessionCancelled: true, SessionFailed: true},
		SessionRunning:    {SessionNeedsInput: true, SessionCompleted: true, SessionFailed: true, SessionCancelled: true, SessionBlocked: true},
		SessionNeedsInput: {SessionRunning: true, SessionCancelled: true, SessionBlocked: true, SessionFailed: true},
	})
}

func ValidateHypothesisTransition(from, to HypothesisState) error {
	return validateTransition(string(from), string(to), map[HypothesisState]map[HypothesisState]bool{
		HypothesisProposed: {HypothesisTesting: true, HypothesisRejected: true, HypothesisBlocked: true},
		HypothesisTesting:  {HypothesisSupported: true, HypothesisRejected: true, HypothesisBlocked: true, HypothesisProposed: true},
		HypothesisBlocked:  {HypothesisProposed: true, HypothesisRejected: true},
	})
}

func ValidateActionTransition(from, to ActionState) error {
	return validateTransition(string(from), string(to), map[ActionState]map[ActionState]bool{
		ActionProposed: {ActionApproved: true, ActionDenied: true, ActionCancelled: true},
		ActionApproved: {ActionRunning: true, ActionCancelled: true},
		ActionRunning:  {ActionSucceeded: true, ActionFailed: true, ActionCancelled: true, ActionTimedOut: true, ActionUncertain: true},
	})
}

func ValidateApprovalTransition(from, to ApprovalState) error {
	return validateTransition(string(from), string(to), map[ApprovalState]map[ApprovalState]bool{
		ApprovalPending: {ApprovalAllowed: true, ApprovalDenied: true, ApprovalExpired: true, ApprovalRevoked: true},
		ApprovalAllowed: {ApprovalRevoked: true, ApprovalExpired: true},
	})
}

func ValidateFindingTransition(from, to FindingState) error {
	return validateTransition(string(from), string(to), map[FindingState]map[FindingState]bool{
		FindingLead: {FindingVerified: true, FindingRejected: true},
	})
}

func validateTransition[T ~string](from, to string, transitions map[T]map[T]bool) error {
	if from == to {
		return nil
	}
	if !transitions[T(from)][T(to)] {
		return fmt.Errorf("invalid transition %q -> %q", from, to)
	}
	return nil
}
