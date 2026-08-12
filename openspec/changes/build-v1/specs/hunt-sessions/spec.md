## Purpose

Define durable hunt sessions as the shared unit of work across interactive and automated clients, including recovery, lifecycle control, and multiple targets or goals.

## ADDED Requirements

### Requirement: One objective creates one session
The system SHALL create one independent persistent session for each submitted hunt objective, while allowing that session to contain multiple targets and multiple security goals.

#### Scenario: Multi-target objective
- **WHEN** a user submits one objective naming multiple authorized hosts and multiple security goals
- **THEN** the system creates one session, derives a concise editable objective name, and associates all recognized targets and goals with that session

#### Scenario: Separate objectives
- **WHEN** a user submits two different hunt objectives
- **THEN** the system creates two independently addressable sessions that can run concurrently

### Requirement: Durable event history
The system SHALL persist every accepted objective change, hypothesis transition, policy decision, action lifecycle event, observation, artifact reference, finding transition, user intervention, error, and terminal state in ordered session history.

#### Scenario: Client exits during execution
- **WHEN** the CLI or TUI disconnects while a session is running
- **THEN** the session continues under the local background runtime and later clients can reconstruct its current state from persisted history

#### Scenario: Runtime restart
- **WHEN** the background runtime restarts after an interruption
- **THEN** it reconstructs each non-terminal session, marks interrupted actions explicitly, and does not automatically repeat an action that may have side effects

### Requirement: Session lifecycle
The system SHALL expose created, running, needs-input, completed, failed, cancelled, and blocked session states and SHALL record the reason for every terminal or intervention state.

#### Scenario: Safe cancellation
- **WHEN** the operator cancels a running session
- **THEN** the system stops scheduling new actions, requests cancellation of active actions, preserves accumulated evidence, and records whether each active action stopped or remained uncertain

#### Scenario: Meaningful completion
- **WHEN** the runtime determines that goals have sufficient evidence, viable hypotheses are exhausted, or all remaining paths are blocked by scope or missing capability
- **THEN** it terminates or blocks the session with explicit goal results and coverage gaps rather than claiming universal completion

### Requirement: Concurrent session isolation
The system SHALL keep session prompts, targets, credentials, workspaces, artifacts, policies, and model context isolated while allowing multiple sessions to make progress concurrently within configured resource limits.

#### Scenario: Two sessions use different credentials
- **WHEN** two concurrent sessions use distinct target credentials
- **THEN** neither session can read or receive the other session's credential, workspace, or evidence references

