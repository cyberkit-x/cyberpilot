## Purpose

Define how CyberPilot discovers, routes, progressively loads, and safely handles community security skills without treating collection size as verified capability.

## ADDED Requirements

### Requirement: Common SKILL.md compatibility
The system SHALL load a skill directory whose `SKILL.md` contains YAML frontmatter with a valid `name` and actionable `description`, followed by Markdown instructions.

#### Scenario: Valid minimal skill
- **WHEN** a configured source contains a uniquely named skill with valid metadata and readable instructions
- **THEN** the system indexes its name, description, source, version identity, license declaration, and trust status without loading the full body into every session

#### Scenario: Invalid skill
- **WHEN** metadata is missing, malformed, duplicated, path-unsafe, or inconsistent with the directory name
- **THEN** the system excludes the skill and reports a specific validation error

### Requirement: Contextual skill retrieval
The runtime SHALL retrieve skills from objective, target observations, current hypotheses, and declared applicability, and SHALL load only selected skill bodies and explicitly needed resources.

#### Scenario: JWT observation
- **WHEN** an API response or operator context establishes that JWT bearer tokens are in use
- **THEN** the runtime may retrieve a focused JWT or API-authentication skill without loading unrelated AD, mobile, forensics, or CTF skills

#### Scenario: No suitable skill
- **WHEN** no indexed skill is sufficiently relevant to the current hypothesis
- **THEN** the runtime continues with native model reasoning and available capabilities or records a knowledge gap; it does not select an unrelated skill merely to select one

### Requirement: Skill authority boundary
The system SHALL treat skill instructions as advisory reasoning input and SHALL never allow a skill to expand scope, grant approval, alter runtime policy, access host secrets, or bypass capability restrictions.

#### Scenario: Skill requests an out-of-scope target
- **WHEN** a loaded skill instructs the agent to probe an address outside the session scope
- **THEN** policy denies the proposed action and records the skill source associated with the rejected proposal

### Requirement: Script and dependency handling
The system SHALL treat scripts and installation commands bundled with third-party skills as untrusted and SHALL NOT execute or install them implicitly.

#### Scenario: Skill requires an unavailable tool
- **WHEN** a skill includes or recommends a package-install command for an unavailable program
- **THEN** the runtime records the requirement and uses a registered safe alternative or blocks the branch; it does not modify the host or task image automatically

#### Scenario: Approved bundled script
- **WHEN** policy permits execution of an inspected bundled script
- **THEN** the script runs only through the session runner with scope controls, timeout, captured output, and skill source identity recorded in the action evidence

### Requirement: Trust status is versioned evidence
The system SHALL present Compatible, Tested, and Maintained as version-specific registry status and SHALL not imply universal safety or effectiveness.

#### Scenario: Skill content changes
- **WHEN** a skill previously marked Tested changes content or executable resources
- **THEN** the changed version loses inherited test status until its applicable evaluation is rerun

