## Purpose

Define evidence and finding states that separate useful hypotheses from reproducible vulnerabilities and make limits, provenance, and sensitive-data handling explicit.

## ADDED Requirements

### Requirement: Structured security records
The system SHALL represent hypotheses, observations, actions, artifacts, leads, verified findings, rejected hypotheses, blocked branches, and coverage gaps as distinct linked records.

#### Scenario: Scanner-like signal
- **WHEN** an HTTP response or tool output only suggests a possible vulnerability
- **THEN** the system records a lead linked to its source evidence and does not present it as a verified finding

### Requirement: Verified finding evidence gate
A finding SHALL be promoted to Verified only when evidence establishes the affected target, realistic prerequisites, attacker-controlled input or state, relevant action path, observable security impact, reproducible procedure, and meaningful control or false-positive exclusion.

#### Scenario: IDOR confirmation
- **WHEN** a test proves that one authorized test identity can access a protected object belonging to a different identity and an ownership-control request demonstrates the expected denial boundary
- **THEN** the system may promote the lead to a verified IDOR finding with both test and control evidence linked

#### Scenario: Status code without impact
- **WHEN** changing an object identifier returns HTTP 200 but does not prove another identity's protected data or state was accessed
- **THEN** the result remains a lead or rejected hypothesis and cannot be promoted to Verified

### Requirement: Reproducible evidence bundles
Each verified finding SHALL preserve the minimum requests, responses, commands, outputs, timestamps, target and session identifiers, model and skill provenance, prerequisites, reproduction steps, observed impact, and limitations needed for an authorized reviewer to reproduce it.

#### Scenario: Export finding
- **WHEN** an operator exports a verified finding
- **THEN** the exported record contains stable artifact references and integrity hashes while excluding unrelated session data

### Requirement: Secret and sensitive-data protection
The system SHALL redact configured credentials, authentication tokens, API keys, and unrelated sensitive target data from logs, terminal summaries, and exported evidence while retaining enough protected source material for authorized local review.

#### Scenario: Token appears in HTTP evidence
- **WHEN** a captured request contains an authentication token
- **THEN** user-facing and exported views mask the token, and the unredacted value is never sent back to the model solely for reporting

### Requirement: Honest coverage reporting
Session results SHALL report assessed targets and hypotheses, rejected leads, blocked branches, missing capabilities, and untested or uncertain surfaces without claiming exhaustive vulnerability absence.

#### Scenario: Browser-dependent surface remains
- **WHEN** a target requires browser execution that V1 does not provide
- **THEN** the final result records that surface as a coverage gap and does not state that the target has no vulnerabilities

