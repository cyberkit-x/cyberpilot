## Purpose

Define deterministic authorization controls that constrain non-deterministic agent decisions before any target-facing or state-changing action is executed.

## ADDED Requirements

### Requirement: Explicit session scope
The system SHALL derive proposed targets and constraints from the objective, require operator confirmation before active testing begins, and persist the confirmed scope as the authoritative target boundary.

#### Scenario: Ambiguous wildcard scope
- **WHEN** an objective contains an ambiguous wildcard or relationship such as an organization name without owned domains
- **THEN** the system requests clarification and does not begin active target interaction

#### Scenario: Newly discovered asset
- **WHEN** a session discovers a related host that is not covered by confirmed scope
- **THEN** the system records it as out-of-scope context and does not interact with it unless the operator explicitly updates scope

### Requirement: Pre-execution policy decision
Every proposed external or state-changing action SHALL receive an `allow`, `ask`, or `deny` policy decision based on target scope, capability, authentication, traffic, data access, side effects, and configured operator constraints.

#### Scenario: Low-risk in-scope request
- **WHEN** an HTTP read targets a confirmed in-scope host, respects limits, and has no identified state-changing effect
- **THEN** policy may allow it and records the decision basis before execution

#### Scenario: Potential state change
- **WHEN** an action may create, update, delete, brute-force, exploit, access sensitive records, or materially increase traffic
- **THEN** policy returns ask or deny according to configured constraints and execution does not begin without an allow decision

#### Scenario: Out-of-scope action
- **WHEN** any proposed network action targets an address outside confirmed scope
- **THEN** policy denies it even if the model, skill, redirect, DNS response, proxy, or tool recommends the action

### Requirement: Approval contains decision context
An approval request SHALL state the intended action, why it is needed, target, expected evidence, risk, expected side effects, limits, and alternatives so the operator can make an informed decision.

#### Scenario: Restricted approval
- **WHEN** an operator approves an action with a lower request count or narrower target than proposed
- **THEN** only the restricted action becomes executable and any expansion requires a new decision

### Requirement: Non-interactive policy never prompts
An `exec` session running without attachment SHALL automatically execute allowed actions, skip denied actions, and transition approval-required branches to blocked or needs-input without waiting on terminal input.

#### Scenario: Other work remains
- **WHEN** one branch requires approval while another allowed branch can progress
- **THEN** the non-interactive session continues the allowed branch and retains the approval request for later TUI intervention

#### Scenario: All work requires input
- **WHEN** every viable branch requires operator input
- **THEN** the session transitions to needs-input or blocked, returns the corresponding stable exit outcome, and remains available for later attachment

### Requirement: Redirect and resolution enforcement
Scope SHALL be re-evaluated after DNS resolution, redirects, proxy routing, and tool-generated target expansion rather than only against the original input string.

#### Scenario: Redirect leaves scope
- **WHEN** an allowed URL redirects to a host or resolved address outside confirmed scope
- **THEN** the runtime prevents the follow-up request and records the boundary transition

