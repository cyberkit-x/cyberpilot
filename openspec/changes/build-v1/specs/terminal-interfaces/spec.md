## Purpose

Define a consistent terminal-native experience for interactive multi-session operation and automation while using one persistent session model and bounded output behavior.

## ADDED Requirements

### Requirement: Default command opens the TUI
Running `cyberpilot` without a subcommand SHALL open the terminal interface when configuration is ready and SHALL direct an uninitialized user to `cyberpilot init`.

#### Scenario: Ready environment
- **WHEN** an initialized user runs `cyberpilot`
- **THEN** the TUI opens the sessions overview and attaches to the local background runtime

### Requirement: Sessions overview prioritizes intervention
The TUI SHALL display a separate NEEDS INPUT list above other sessions and SHALL support creating, opening, searching, and moving between sessions without exposing internal agent branches as top-level sessions.

#### Scenario: Approval becomes required
- **WHEN** a running session creates an unresolved approval request
- **THEN** that session appears in NEEDS INPUT with a concise reason while other sessions remain in OTHER SESSIONS

#### Scenario: Narrow terminal
- **WHEN** terminal width cannot fit all row fields
- **THEN** the TUI truncates names with `...`, progressively hides secondary fields, wraps only detail content, and never requires horizontal scrolling

### Requirement: Session view supports observation and intervention
The session view SHALL show the objective, targets, state, current hypothesis and action, recent evidence, findings, and pending input, and SHALL let the operator add instructions, decide approvals, update scope with confirmation, cancel, and return to the overview.

#### Scenario: Approve with limits
- **WHEN** the operator opens an approval request and supplies narrower limits
- **THEN** the interface records and applies the restricted approval and the session resumes eligible work

### Requirement: Non-interactive exec creates a normal session
The command `cyberpilot exec <prompt>` SHALL create the same persistent session type as the TUI, run it without interactive questions, stream progress to stderr, and write only the final selected output format to stdout.

#### Scenario: Prompt from standard input
- **WHEN** the user runs `cyberpilot exec -` and supplies a non-empty prompt on stdin
- **THEN** the system creates and executes a persistent session using that prompt without echoing secrets from input

#### Scenario: Detached execution
- **WHEN** the user passes `--detach`
- **THEN** the command returns the session identifier after durable creation and the background runtime continues execution

### Requirement: Stable exec outcomes
The `exec` command SHALL support human-readable and JSON output and SHALL return stable exit codes: `0` for completed with no verified findings, `1` for completed with verified findings, `2` for needs-input or blocked, `3` for configuration or runtime failure, and `4` for invalid input or scope.

#### Scenario: Verified finding in JSON mode
- **WHEN** a session completes with one or more verified findings and JSON output is selected
- **THEN** stdout contains one valid final JSON result, progress remains on stderr, and the process exits with code 1

#### Scenario: Human input required
- **WHEN** all viable work is blocked on operator input
- **THEN** the command identifies the persistent session on stderr or in the selected result, exits with code 2, and does not wait indefinitely

### Requirement: Terminal clients do not own task lifetime
Exiting or disconnecting from the TUI SHALL not pause, cancel, or terminate background sessions.

#### Scenario: User quits overview
- **WHEN** the operator quits the TUI while sessions are active
- **THEN** the client exits and eligible sessions continue under the local runtime
