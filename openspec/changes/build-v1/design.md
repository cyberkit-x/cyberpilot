## Context

The repository currently contains a portable Go command, project standards, and a five-target release pipeline, but no application runtime. See `proposal.md` for product motivation and `specs/` for observable behavior.

V1 must run as a local terminal-native product on Linux, macOS, and Windows, use an external model service, control a local Docker or Podman engine, persist work after clients exit, and treat model output, target content, and community skills as untrusted. The binary remains compatible with `CGO_ENABLED=0`; target-facing actions run in OCI sandboxes rather than on the host.

## Goals / Non-Goals

**Goals:**

- Establish a small set of stable domain and provider interfaces around sessions, models, skills, policy, runners, and evidence.
- Make the event log and artifact store the recoverable source of truth for all CLI and TUI clients.
- Put deterministic validation and policy gates around every non-deterministic model decision.
- Ship one binary that can act as a client, background daemon, and internal worker entrypoint without publishing multiple executables.
- Keep V1 operable by one expert on one workstation and testable without public-network targets.

**Non-Goals:**

- A distributed control plane, multi-user tenancy, remote or Kubernetes scheduling, browser automation, or host-command fallback.
- Installing arbitrary toolchains from skills or dynamically rebuilding sandbox images during a hunt.
- A general attack graph database, vector database, model training pipeline, cloud registry, or public Skill marketplace.
- Proving that a model or Skill is universally effective; V1 provides evidence and evaluation hooks, not that claim.

## Decisions

### 1. One Go binary with client and daemon modes

`cyberpilot` owns public commands. Interactive commands connect to a per-user background daemon over a local authenticated transport; if it is absent, the client starts the same binary in an internal daemon mode and waits for readiness. Unix platforms use a Unix domain socket with owner-only permissions; Windows uses a named pipe restricted to the current user.

The daemon owns sessions, schedulers, model calls, OCI providers, and storage. TUI exit only closes a client. A per-user lock and daemon identity prevent multiple writers from opening the same store.

Alternatives considered:

- One foreground process per session is simpler but cannot provide persistent multi-session operation.
- Separate public `cyberpilotd` and `cyberpilot` binaries make packaging and service management more complex without helping V1 users.

### 2. SQLite event store plus content-addressed artifacts

Use a pure-Go SQLite driver to preserve `CGO_ENABLED=0`. SQLite runs in WAL mode and stores normalized current views plus an append-only session event table. Large request bodies, responses, command output, and files live in a per-user content-addressed artifact directory keyed by SHA-256; database records contain metadata, redaction state, ownership, and hashes.

State transitions append an event and update projections in one database transaction. Event payloads are versioned JSON. The runtime rebuilds projections from events in tests and during explicit repair, while normal reads use projections. Secrets are represented only by opaque credential references; raw credentials never enter events.

Alternatives considered:

- JSON files are easy to inspect but provide weak concurrency, transactions, queries, and recovery.
- An external database conflicts with the local-first single-binary product.
- Event-only reads would simplify authority but make common TUI queries unnecessarily expensive.

### 3. Explicit state machines and typed records

Sessions, hypotheses, actions, approvals, and findings use validated state transitions. IDs use UUIDv7 for sortable, globally unique identifiers. Timestamps are UTC. Core records include:

- Session: objective, editable name, targets, goals, constraints, budgets, state, terminal reason.
- Hypothesis: claim, related goals/assets, supporting and contradicting evidence, priority, state.
- Action proposal: target, purpose, hypothesis, capability, structured arguments, risk, side effects, expected evidence.
- Policy decision and approval: allow/ask/deny, decision basis, proposed and approved limits, operator identity.
- Observation and artifact: source action, normalized summary, protected raw material reference, provenance.
- Finding: lead or verified state, evidence links, prerequisites, impact, controls, reproduction, limitations.

Internal agent branches remain subordinate records, not user sessions.

### 4. Structured model protocol with one V1 adapter family

Define a model interface for capability probing and turn execution with structured messages, tool/action proposals, usage, finish reason, and provider error classification. V1 implements OpenAI-compatible HTTP semantics and treats named OpenAI-compatible endpoints as configuration variants; the interface does not expose provider-specific response types to the hunt runtime.

The model does not execute tools directly. It returns typed proposals, which are schema-validated, checked against capabilities and scope, decided by policy, then dispatched. Tool results return as bounded structured observations with artifact references. Invalid output is repaired once with schema feedback, then fails the turn explicitly.

Alternatives considered:

- Provider SDKs speed initial integration but leak provider types and complicate portable builds.
- Free-form command text is flexible but cannot support reliable policy, audit, or evidence linkage.
- Multiple planner/operator/verifier models are deferred to avoid cost, synchronization, and debugging complexity.

### 5. Skill index uses deterministic metadata retrieval before model selection

Configured local and pinned Git sources are scanned for `SKILL.md`. The loader parses bounded YAML frontmatter, validates paths and licenses, records content hashes and provenance, and indexes `name`, `description`, optional domains/intents/requirements, and trust status. Duplicate names require explicit precedence; otherwise they are disabled.

Retrieval uses lexical scoring over the objective, observations, and hypotheses plus declared metadata. The model chooses from a small scored candidate set and must state relevance before the body is loaded. Directly linked reference files are loaded on demand within size limits; references cannot escape the skill directory. Scripts are registered as untrusted resources, never auto-executed.

V1 does not require embeddings or a vector database. This keeps retrieval reproducible, inspectable, and dependency-light while the corpus and evaluation set are still small.

### 6. Deterministic policy surrounds every action

The policy evaluator receives a normalized action proposal, confirmed scope, DNS/redirect context, credentials requested, traffic limits, risk and side-effect attributes, session mode, and prior approvals. It produces an immutable allow/ask/deny decision before dispatch. Approval can only narrow or exactly authorize a proposal; broader actions require a new proposal.

Scope uses canonical URL/host/port/CIDR matchers. Network actions resolve through a daemon-controlled network broker that rechecks destination IPs and redirects and proxies approved requests into the sandbox. The sandbox receives no general unrestricted network path in the supported V1 configuration. This design is required to enforce post-resolution boundaries rather than relying only on command-string inspection.

Shell and Python actions are still high-variance. V1 permits only commands launched through the runner with a declared network mode, workspace root, timeout, output limit, and policy decision. Commands that can create arbitrary network clients use the scoped network proxy or are denied when enforceable routing is unavailable.

Alternatives considered:

- Prompt-only policy cannot resist injection or model error.
- Parsing shell text alone cannot reliably understand redirects, DNS changes, or program behavior.
- A fully unrestricted task network would contradict the scope contract.

### 7. OCI runner via provider CLI adapters

Implement one provider-neutral Runner interface and Docker and Podman adapters that invoke their installed CLI with argument arrays, explicit timeouts, sanitized environments, and normalized errors. CLI adapters reuse native Docker contexts and Podman connections and avoid importing two large client stacks or binding to unstable socket details.

Each session receives one named container, one workspace volume/directory, a read-only base image pinned by digest for releases, a non-root user, dropped Linux capabilities, resource limits, no runtime socket, and no host secret mounts. The image contains shell, curl, Python, CA roots, and a small runner helper. Commands are executed with `docker|podman exec`; outputs stream to bounded artifact capture. Cancellation terminates the process and, if necessary, recreates the sandbox while preserving the workspace.

Provider conformance tests run against Docker and Podman when available; unit tests use a fake Runner. Windows requires Docker Desktop or Podman machine with Linux containers.

Alternatives considered:

- Direct Engine APIs are more efficient but double adapter complexity and certificate/connection handling in V1.
- One ephemeral container per command loses useful task state and adds startup cost.
- Host execution is omitted because it weakens isolation and produces platform-specific behavior.

### 8. Scheduler is bounded and event driven

The daemon schedules at most one model-planning turn per session and a configurable global number of actions. Independent sessions can progress concurrently. Within a session, multiple ready actions may run concurrently only when policy and capability declarations mark them independent and resource limits permit it; V1 may initially execute them serially while preserving the scheduler contract.

Every model call and action has a context cancellation path, deadline, attempt count, idempotency key, and terminal result. Potentially side-effecting interrupted actions become `uncertain`, never automatic retry candidates. Read-only transient failures use bounded exponential backoff with jitter.

Budgets cover elapsed time, model tokens/cost when reported, actions, and non-progress loops. Reaching a budget produces an honest partial result.

### 9. Evidence promotion is deterministic, assisted by the model

The model may propose a lead or promotion, but a finding validator requires the structured evidence links defined in the spec. Class-specific Skills can add evidence prompts; they cannot weaken the baseline. Promotion runs checks for target, prerequisites, controllability, action path, observed impact, reproduction, and a control or explicit reason a control is infeasible.

The same model may assist V1 verification, but it receives a fresh verification prompt built from evidence rather than its original conclusion. The result is still gated by deterministic required fields and remains a lead when the gate is incomplete. A separate verifier model is deferred.

Redaction occurs before model context, terminal rendering, and export. Raw protected evidence stays local with restrictive permissions and explicit access paths.

### 10. Local RPC and terminal presentation

Define a versioned local RPC protocol over newline-delimited JSON messages: request ID, protocol version, method, typed payload, typed error, and event subscription cursor. Public methods cover configuration status, session create/list/get/cancel, instruction and scope updates, approval decisions, event subscription, artifact metadata, and result export.

The CLI uses the same RPC as the TUI. `exec` submits a session, subscribes to events, sends progress to stderr, and renders exactly one final human or JSON result to stdout. Detached mode returns after durable creation. The TUI is implemented with Bubble Tea and Lip Gloss, keeps all labels in English, uses NEEDS INPUT and OTHER SESSIONS lists, and applies width-aware field hiding and ellipsis.

Alternatives considered:

- In-process CLI calls would create two behavior paths and make background lifetime inconsistent.
- HTTP on localhost expands attack surface and port management; a user-scoped local transport is sufficient.

### 11. Configuration and credential storage

Non-secret configuration uses a versioned YAML file in the platform user-config directory. Runtime data uses the platform user-data directory. On macOS and Windows, credentials use the native Keychain/Credential Manager through a small credential interface. On Linux, Secret Service is used when available; otherwise initialization supports an environment-variable reference and clearly reports that no persisted secret backend is available. V1 does not write plaintext API keys to disk.

Configuration writes use temporary files, owner-only permissions where supported, fsync, and atomic rename. Updates are validated before active-profile replacement.

### 12. Testing and release gates

Unit tests cover state machines, event replay, scope matching, policy matrices, redaction, Skill validation/retrieval, evidence promotion, RPC schemas, and provider error normalization. Integration tests use a fake OpenAI-compatible server, intentionally vulnerable local HTTP fixtures, and disposable OCI sandboxes. Tests never contact public targets.

CI keeps the five existing binary targets. Host-independent tests run on every pull request. Docker conformance runs on Linux CI. Podman conformance runs where the GitHub runner supports rootless Podman and otherwise remains a documented required release check. Release packages include license, README, pinned sandbox-image identity, and checksums.

## Risks / Trade-offs

- **[Shell/Python can create network behavior that is hard to infer]** → Route supported target traffic through the scoped broker, deny unrestricted networking, record declared capabilities, and keep raw socket/high-privilege operations out of V1.
- **[A local container engine is a high-privilege control surface]** → Only the daemon connects to it; task containers never receive its socket, user input does not become raw engine arguments, and sandbox names/labels are scoped to CyberPilot.
- **[One daemon and SQLite can become a throughput limit]** → Bound concurrency and WAL transactions for V1; preserve provider and storage interfaces for later remote runners without adding distributed complexity now.
- **[OpenAI-compatible endpoints differ in tool-call behavior]** → Require a real initialization probe, normalize errors, version the model contract, and fail explicitly when capability guarantees are absent.
- **[Lexical Skill retrieval may miss semantically relevant skills]** → Use high-quality descriptions, optional domain metadata, candidate explanations, and evaluation data; introduce embeddings only after measured retrieval failures justify them.
- **[The same model can reinforce its own incorrect finding]** → Use a fresh evidence-only verification turn, deterministic promotion fields, negative controls, and retain Lead status when proof is incomplete.
- **[Protected raw evidence remains sensitive on disk]** → Apply restrictive ownership, content hashing, redaction at every outward boundary, configurable retention, and explicit deletion in a later lifecycle change.
- **[Windows named pipes, OCI machines, and Unix sockets differ]** → Isolate platform transport and provider discovery behind conformance-tested interfaces; keep domain behavior platform-neutral.
- **[The V1 scope is still substantial]** → Implement vertical milestones that each preserve the final contracts, starting with fake providers and one deterministic fixture before live model and OCI integrations.

## Migration Plan

1. Introduce versioned configuration and data directories without migrating the existing bootstrap binary, which has no user data.
2. Land domain records, storage, fake providers, and RPC behind commands that remain explicitly experimental.
3. Enable `init` and daemon startup after configuration, credential, and provider probes pass integration tests.
4. Enable `exec` against deterministic local fixtures, then approved live model endpoints, then Docker and Podman conformance.
5. Enable the TUI after the shared RPC and session behavior are stable.
6. Publish a prerelease and run the full release checklist on all five binary targets and supported OCI providers before `v0.1.0`.

Rollback before `v0.1.0` removes the prerelease binary and its per-user data directory after optional evidence export. After the first tagged release, schema upgrades must be forward-only, transactional, backed up before migration, and compatible with the previous released schema until an explicit breaking-change proposal says otherwise.
