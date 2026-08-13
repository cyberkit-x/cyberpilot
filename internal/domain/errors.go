package domain

import "fmt"

type ErrorCode string

const (
	ErrInvalidInput      ErrorCode = "invalid_input"
	ErrNotInitialized    ErrorCode = "not_initialized"
	ErrAuthentication    ErrorCode = "authentication_failed"
	ErrCapabilityMissing ErrorCode = "capability_missing"
	ErrPolicyDenied      ErrorCode = "policy_denied"
	ErrNeedsInput        ErrorCode = "needs_input"
	ErrRunnerUnavailable ErrorCode = "runner_unavailable"
	ErrConflict          ErrorCode = "conflict"
	ErrNotFound          ErrorCode = "not_found"
	ErrInternal          ErrorCode = "internal"
)

type PublicError struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func (e *PublicError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
