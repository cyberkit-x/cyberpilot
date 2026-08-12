## Why

CyberPilot currently defines its product principles and release matrix but cannot yet run an authorized vulnerability-hunting task. The first release must prove the core product claim: a red-team operator can provide an objective and scope, then a persistent agent can select community skills, act safely in an isolated environment, and return evidence-backed results without a fixed scanning workflow.

## What Changes

- Add `cyberpilot init` to configure and validate one model profile and a local Docker or Podman runner without storing secrets in project files.
- Add `cyberpilot exec` to create and run a persistent, non-interactive Web/API hunt session from a prompt, with stable machine-readable outcomes and no hidden approval prompts.
- Add a terminal UI for creating, listing, switching, observing, and intervening in the same persistent sessions used by `exec`.
- Add an event-backed session runtime in which one hunt objective creates one session, a session may contain multiple targets and security goals, and execution survives client exit.
- Add an agentic control loop that observes evidence, retrieves focused community skills, proposes the next action, interprets results, and replans rather than following a fixed reconnaissance-to-exploitation pipeline.
- Add compatible `SKILL.md` discovery and progressive loading, initially from trusted local or configured Git sources, while treating bundled scripts as untrusted.
- Add deterministic scope and action policy with `allow`, `ask`, and `deny` decisions before tool execution.
- Add Docker and Podman OCI runners that provide a task workspace with shell, curl, Python, and evidence storage; container-runtime sockets are never exposed inside the task sandbox.
- Add structured hypotheses, observations, actions, artifacts, leads, verified findings, blocked branches, and coverage gaps with reproducible evidence requirements.
- Keep browser automation, Kubernetes, remote runners, team collaboration, mobile, Active Directory, binary exploitation, and multi-model orchestration outside V1.

## Capabilities

### New Capabilities

- `configuration`: First-run model credentials and Docker or Podman runner discovery, validation, storage, and later inspection or modification.
- `hunt-sessions`: Persistent session lifecycle, event history, multiple targets and goals, state recovery, cancellation, and terminal-client attachment.
- `agentic-hunt-runtime`: Evidence-driven hypothesis selection, skill retrieval, action proposal, result interpretation, replanning, and explicit stopping conditions.
- `skill-loading`: Compatible `SKILL.md` discovery, metadata routing, progressive resource loading, provenance, trust status, and script handling.
- `policy-and-scope`: Target scope enforcement, action risk classification, approval decisions, non-interactive behavior, and auditability.
- `oci-execution`: Provider-neutral task execution through local Docker or Podman with isolated workspace, resource lifecycle, cancellation, and artifact capture.
- `finding-evidence`: Structured observations and findings, promotion criteria, reproducible evidence bundles, redaction, and coverage reporting.
- `terminal-interfaces`: Shared CLI/TUI behavior for task creation, multi-session overview, intervention, non-interactive output, and stable exit codes.

### Modified Capabilities

None. This is the first behavioral specification for CyberPilot.

## Impact

- Introduces the initial Go application architecture and persistent local data model.
- Adds model-provider, skill-provider, policy, and runner interfaces while keeping their implementations replaceable.
- Adds local credential integration, OCI runtime access, task workspaces, and untrusted-content boundaries that require security-focused tests.
- Expands CI beyond compilation to cover command behavior, session recovery, policy allow/ask/deny paths, Docker and Podman contract tests, skill routing, and evidence promotion.
- Establishes the first user-facing CLI/TUI and machine-readable output contracts; compatibility must be preserved after the first tagged release.
