## 1. Foundation and Contracts

- [ ] 1.1 Create the internal package boundaries for configuration, domain records, events, storage, RPC, models, skills, policy, runners, evidence, scheduler, CLI, and TUI without exposing provider types across boundaries.
- [ ] 1.2 Define versioned IDs, timestamps, session/hypothesis/action/approval/finding states, transition validators, and JSON event envelopes with unit tests for every valid and invalid transition.
- [ ] 1.3 Define typed model proposals, runner results, policy decisions, evidence records, RPC requests/responses, and stable public error codes; add schema round-trip and compatibility tests.
- [ ] 1.4 Add the V1 dependency set, license inventory, and CI checks for formatting, vet, race tests, vulnerability scanning, OpenSpec validation, and all five existing cross-build targets.

## 2. Persistent Runtime and Local RPC

- [ ] 2.1 Implement platform user config/data path resolution, owner-restricted directory creation, single-daemon locking, Unix socket transport, and Windows named-pipe transport behind one local RPC interface.
- [ ] 2.2 Implement pure-Go SQLite migrations, WAL configuration, transactional append-only events, normalized projections, event replay, and corruption/interrupted-migration tests.
- [ ] 2.3 Implement the content-addressed artifact store with SHA-256 integrity, atomic writes, metadata ownership, protected/raw versus redacted variants, and cross-session access-denial tests.
- [ ] 2.4 Implement daemon startup/readiness, authenticated per-user RPC, protocol-version negotiation, request errors, cursor-based event subscriptions, graceful shutdown, and auto-start from the client.
- [ ] 2.5 Implement session create/list/get/cancel, instruction updates, confirmed scope updates, approval decisions, recovery of non-terminal sessions, and uncertain interrupted-action handling over RPC.

## 3. Configuration and Credentials

- [ ] 3.1 Implement versioned YAML configuration with atomic validated updates and redacted `cyberpilot config` display for one active model and one active runner profile.
- [ ] 3.2 Implement native macOS Keychain and Windows Credential Manager storage plus Linux Secret Service or environment-reference fallback, with tests proving credentials never enter config, events, logs, or command arguments.
- [ ] 3.3 Implement Docker and Podman discovery, endpoint/privilege summaries, explicit multi-provider selection, disposable lifecycle probes, and actionable no-provider failures.
- [ ] 3.4 Implement interactive `cyberpilot init`, safe re-initialization confirmation, model/runner validation ordering, rollback to the last valid profile, and terminal integration tests.

## 4. Model Provider and Agent Loop

- [ ] 4.1 Implement the OpenAI-compatible capability probe for endpoint access, authentication, selected model, structured output, and typed tool calls using a deterministic fake provider in tests.
- [ ] 4.2 Implement normalized model turn requests, streaming/non-streaming response handling, usage and finish reasons, provider error classification, secret-safe logs, bounded output, and one schema-repair attempt.
- [ ] 4.3 Implement prompt/context assembly from objective, goals, scope, hypotheses, selected skills, recent observations, budgets, and artifact summaries without placing protected raw evidence into model context by default.
- [ ] 4.4 Implement the observe-reason-propose-policy-act-interpret-replan loop, explicit hypothesis transitions, non-progress detection, cancellation, deadlines, idempotency keys, bounded retries, and stopping conditions.
- [ ] 4.5 Implement per-session planning serialization, global action concurrency limits, elapsed/token/cost/action budgets, and honest budget-exhausted results.

## 5. Skill Discovery and Routing

- [ ] 5.1 Implement bounded `SKILL.md` frontmatter parsing, name/path/license validation, content hashing, provenance, trust status, duplicate resolution, and unsafe-path rejection.
- [ ] 5.2 Implement configured local and pinned-Git Skill sources with refresh that never executes repository content and never changes a Tested status across content hashes.
- [ ] 5.3 Implement deterministic lexical candidate retrieval from objectives, observations, hypotheses, and optional metadata; add positive, negative, unrelated-domain, and no-match routing tests.
- [ ] 5.4 Implement model selection from a bounded candidate set, lazy Skill-body loading, directly linked reference loading with size/path limits, and context-budget enforcement.
- [ ] 5.5 Register bundled scripts and dependency declarations as untrusted resources, prohibit implicit installation or execution, and preserve source/version identity when an approved script becomes an action.
- [ ] 5.6 Add a small licensed V1 Web/API Skill fixture set covering authorization/IDOR, JWT/API authentication, SSRF, file exposure/upload, and finding verification for integration and routing tests.

## 6. Scope and Policy Enforcement

- [ ] 6.1 Implement canonical target parsing for URL, hostname, port, IP, CIDR, and explicit wildcard forms with ambiguity errors and scope-confirmation tests.
- [ ] 6.2 Implement typed action risk and side-effect attributes plus deterministic allow/ask/deny evaluation for network, credential, traffic, data-access, state-change, command, and capability constraints.
- [ ] 6.3 Implement immutable policy decisions, informative approval requests, restricted approvals that can only narrow proposals, expiry/revocation, and full audit events.
- [ ] 6.4 Implement non-interactive policy behavior that progresses allowed branches, retains approval requests, never reads terminal input, and enters needs-input/blocked when no allowed work remains.
- [ ] 6.5 Implement the scoped network broker and sandbox proxy configuration with DNS, resolved-IP, redirect, proxy-route, rate, and destination revalidation against local test targets.
- [ ] 6.6 Add adversarial tests for prompt injection, Skill-requested scope expansion, redirects to out-of-scope addresses, DNS changes, command-generated network clients, and attempted policy bypass.

## 7. Docker and Podman Execution

- [ ] 7.1 Define and implement the provider-neutral Runner contract with a fake adapter covering sandbox lifecycle, exec streaming, normalized exits, timeouts, cancellation, artifacts, and provider failures.
- [ ] 7.2 Implement Docker CLI and Podman CLI adapters using argument arrays, sanitized environments, native contexts/connections, CyberPilot labels, and strict ownership of managed resources.
- [ ] 7.3 Build and pin the non-root V1 sandbox image with shell, curl, Python, CA roots, the runner helper, dropped capabilities, read-only base filesystem, workspace mount, and no runtime socket or host-secret mounts.
- [ ] 7.4 Implement one persistent sandbox per session, workspace recovery, bounded stdout/stderr artifact capture, timeout termination, uncertain-action handling, and cleanup/retention policy.
- [ ] 7.5 Enforce memory, process, output, concurrency, and network settings and fail closed when the configured provider or enforceable network path is unavailable.
- [ ] 7.6 Add Docker conformance tests to CI and a repeatable rootless Podman release check covering create/start/exec/cancel/recover/stop/remove and cross-session isolation.

## 8. Evidence and Findings

- [ ] 8.1 Implement linked hypotheses, observations, actions, artifacts, leads, verified findings, rejected hypotheses, blocked branches, and coverage-gap persistence and queries.
- [ ] 8.2 Implement baseline verified-finding gates for target, prerequisites, controllability, action path, impact, reproduction, provenance, and negative/control evidence, with incomplete cases retained as leads.
- [ ] 8.3 Implement an evidence-only verification turn, structured promotion proposal, deterministic gate decision, downgrade reasons, and tests that prevent HTTP status, scanner match, code pattern, or model assertion from proving a finding alone.
- [ ] 8.4 Implement credential/token/sensitive-data detection and redaction before model context, logs, terminal rendering, and export while preserving explicitly protected local raw artifacts.
- [ ] 8.5 Implement finding and session-result export with integrity hashes, stable artifact references, prerequisites, reproduction, impact, limitations, assessed scope, rejected hypotheses, and coverage gaps.

## 9. CLI and Non-Interactive Execution

- [ ] 9.1 Replace the bootstrap argument handling with a tested command tree for `init`, `config`, `exec`, `version`, default TUI launch, and hidden daemon mode while preserving embedded build information.
- [ ] 9.2 Implement prompt argument, stdin, prompt-file, JSON output, model/runner profile selection, budget, and `--detach` handling with input validation and secret-safe diagnostics.
- [ ] 9.3 Implement `exec` durable session creation, event following, stderr progress, exactly one stdout result, signals/cancellation, late TUI attachment, and exit codes 0 through 4 defined by the spec.
- [ ] 9.4 Add golden and end-to-end CLI tests for uninitialized use, successful no-finding/finding results, blocked approval, invalid scope, runner failure, JSON pipeline use, detach, and daemon restart.

## 10. TUI Experience

- [ ] 10.1 Implement the Bubble Tea application shell, daemon connection/reconnection, keyboard navigation, responsive width utilities, ellipsis, wrapped detail content, and no-horizontal-scroll snapshots.
- [ ] 10.2 Implement the Sessions Overview with summary counts, separate NEEDS INPUT and OTHER SESSIONS lists, objective-derived names, concise current activity, search, movement, create, open, and quit behavior.
- [ ] 10.3 Implement the Session view with objective, multiple targets/goals, state, current hypothesis/action, timeline, evidence, findings, coverage gaps, and live event updates.
- [ ] 10.4 Implement intervention flows for instructions, approval detail and restricted decisions, scope changes with confirmation, cancellation, and return to overview.
- [ ] 10.5 Add terminal-size, long-text, Unicode, reconnect, multiple-concurrent-session, and keyboard-flow tests with all visible TUI labels in English.

## 11. End-to-End Acceptance and Release

- [ ] 11.1 Create intentionally vulnerable local Web/API fixtures with positive and negative authorization, authentication, SSRF-like scoped redirect, and false-positive cases; prohibit public-target addresses in test configuration.
- [ ] 11.2 Prove the full Docker path from `init` through `exec`, dynamic Skill retrieval, policy decisions, action execution, evidence promotion, final output, daemon restart, and later TUI inspection.
- [ ] 11.3 Prove the equivalent rootless Podman path and document platform-specific prerequisites for Linux, macOS Podman machine, Windows, and Docker Desktop without changing domain behavior.
- [ ] 11.4 Add failure acceptance for model outage, invalid structured output, runtime loss, scope escape, timeout, cancellation, credential leakage checks, interrupted side-effect action, and missing browser capability.
- [ ] 11.5 Update README and operator documentation to match implemented commands, configuration locations, threat model, supported providers, data retention, known V1 coverage gaps, and authorized-use constraints.
- [ ] 11.6 Extend the release workflow to publish the pinned sandbox-image identity, licenses, SBOMs, signatures, checksums, and five binary packages; verify each package's `version` output and platform format.
- [ ] 11.7 Run `make check`, strict OpenSpec validation, all unit/integration/conformance tests, and the five-target cross-build; resolve every release-blocking failure before marking V1 apply complete.
