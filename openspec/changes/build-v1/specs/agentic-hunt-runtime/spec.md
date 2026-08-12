## Purpose

Define the evidence-driven autonomous hunt loop that chooses and revises investigations without presenting a fixed penetration-testing workflow as agentic behavior.

## ADDED Requirements

### Requirement: Evidence-driven planning loop
The system SHALL repeatedly observe session state, form or update security hypotheses, retrieve relevant skills, propose the next bounded action, interpret the result, and replan from new evidence.

#### Scenario: New evidence changes priority
- **WHEN** an action reveals a previously unknown authenticated API surface relevant to a session goal
- **THEN** the runtime records the observation, updates related hypotheses, and may prioritize an authentication or authorization investigation without waiting for a predefined phase to finish

#### Scenario: Failed experiment
- **WHEN** an experiment contradicts its hypothesis
- **THEN** the runtime records the contradiction, rejects or revises the hypothesis, and selects a different justified action instead of repeating the same experiment indefinitely

### Requirement: Explainable proposed actions
Before execution, every agent-proposed action SHALL identify its target, purpose, related hypothesis, expected evidence, requested capability, risk attributes, and side-effect expectation.

#### Scenario: Operator inspects current work
- **WHEN** an operator opens a running session
- **THEN** the interface can show the current hypothesis, why the latest action was selected, the action status, and the evidence expected from it

### Requirement: Bounded autonomy
The system SHALL limit planning by the session objective, target scope, available capabilities, policy, configured resource budget, and operator instructions, regardless of skill or model output.

#### Scenario: Model proposes an unavailable capability
- **WHEN** the model proposes browser interaction but the session only has shell, HTTP, Python, and filesystem capabilities
- **THEN** the runtime records the capability gap, seeks an alternative action when possible, and otherwise marks the affected branch blocked without inventing a result

#### Scenario: Repeated non-progress
- **WHEN** repeated actions produce no material new observation and reach the configured loop limit
- **THEN** the runtime stops that branch, records the non-progress reason, and replans or reports the coverage gap

### Requirement: Explicit stopping conditions
The system SHALL end autonomous work only when all goals have an explicit outcome, the configured budget is exhausted, the operator cancels, a fatal runtime error occurs, or remaining work is blocked by policy, missing input, missing capability, or unreachable targets.

#### Scenario: Budget exhaustion
- **WHEN** the configured action, token, time, or cost budget is exhausted
- **THEN** the session stops scheduling actions and reports accumulated findings, leads, rejected hypotheses, and remaining coverage without describing the hunt as fully complete

