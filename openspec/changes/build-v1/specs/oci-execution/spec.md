## Purpose

Define portable, isolated task execution through local Docker or Podman while preserving task state, policy enforcement, observability, and host separation.

## ADDED Requirements

### Requirement: Provider-neutral OCI execution
The system SHALL provide equivalent sandbox lifecycle and command behavior through configured local Docker and rootless Podman providers.

#### Scenario: Run the same action on either provider
- **WHEN** equivalent Docker and Podman profiles execute the same supported shell action
- **THEN** both return normalized status, stdout, stderr, exit code, timing, cancellation state, and artifact references to the session runtime

### Requirement: One persistent sandbox per active session
The system SHALL create one isolated persistent sandbox for each active session, preserve its task workspace across actions and client exits, and remove compute resources only according to session retention policy.

#### Scenario: Script reused by a later action
- **WHEN** an allowed action writes a helper script into the session workspace and a later action uses it
- **THEN** the script remains available only within that session and its creation and use are represented in session evidence

### Requirement: Minimal V1 capabilities
The V1 sandbox SHALL provide a shell, curl, Python, and a writable task workspace and SHALL expose only capabilities explicitly registered with the session runtime.

#### Scenario: Tool not present
- **WHEN** an action requests an executable not included in the V1 environment
- **THEN** execution returns a structured missing-capability result without installing the tool or falling back to host execution

### Requirement: Container runtime isolation
The system SHALL NOT mount Docker sockets, Podman sockets, host credential stores, or unrestricted host filesystem paths into a task sandbox.

#### Scenario: Skill asks for container socket
- **WHEN** a skill or generated command attempts to access a container-runtime endpoint
- **THEN** the action is denied or fails inside the sandbox and the attempt is recorded as a policy-relevant event

### Requirement: Scoped network and resource control
The runner SHALL apply session network policy, concurrency, timeout, process, memory, and output limits before starting an action.

#### Scenario: Action exceeds timeout
- **WHEN** a process runs past its approved timeout
- **THEN** the runner terminates it, captures available output, marks the action timed out, and returns control to the runtime for replanning

#### Scenario: Target resolution changes
- **WHEN** a sandbox action resolves or follows a target outside the allowed network set
- **THEN** the network control blocks the connection and records enough context to explain the decision

### Requirement: No silent host fallback
The system SHALL fail closed when the configured OCI provider is unavailable and SHALL not execute a task action on the host unless a future explicit host-runner profile is configured and approved.

#### Scenario: Runtime stops during a hunt
- **WHEN** Docker or Podman becomes unavailable during an active session
- **THEN** affected actions fail with a runner-unavailable result and the session blocks or replans without host execution

