# CyberPilot

CyberPilot is an open-source, terminal-native agentic security operator for authorized Web and API vulnerability hunting. It keeps multiple persistent hunt sessions in one local runtime, retrieves focused community `SKILL.md` guidance as evidence changes, evaluates every proposed action with deterministic policy, runs commands in isolated Docker or rootless Podman sandboxes, and separates leads from evidence-verified findings.

> CyberPilot is for systems you own or are explicitly authorized to assess. Scope and approval controls are safety boundaries, not permission to test third-party systems.

## Current build-v1 status

The repository implements the local daemon, authenticated RPC, persistent sessions/events/artifacts, OpenAI-compatible model and OCI adapters, policy/network boundaries, dynamic Skill retrieval, evidence gates/exports, `init`, `config`, durable `exec`, and the English multi-session TUI. A created session is scheduled by the daemon through the model/Skill/policy/action/evidence loop and remains observable after the initiating client exits.

## Build

Requirements are Go 1.24+, Node/npm for strict OpenSpec validation, and Docker or rootless Podman for sandbox conformance.

```bash
make build
make check
make spec-check
```

The default Go binary is built with `CGO_ENABLED=0`. CI compiles Linux amd64/arm64, macOS amd64/arm64, and Windows amd64.

## Releases

Pushing a semantic version tag such as `v0.1.0` runs all tests, vulnerability and OpenSpec validation, Docker lifecycle and complete agent-path acceptance, and all five cross-builds before publishing a GitHub Release. The release contains:

- `cyberpilot_vX.Y.Z_linux_amd64.tar.gz`
- `cyberpilot_vX.Y.Z_linux_arm64.tar.gz`
- `cyberpilot_vX.Y.Z_darwin_amd64.tar.gz`
- `cyberpilot_vX.Y.Z_darwin_arm64.tar.gz`
- `cyberpilot_vX.Y.Z_windows_amd64.zip`
- `cyberpilot-release.spdx.json`
- `checksums.txt`

Every platform archive includes the binary, README, third-party license inventory, and pinned sandbox-image identity. GitHub build-provenance attestations cover all five archives, the SBOM, and checksums. Release publication is skipped on normal branch pushes and fails closed when any expected asset or prerequisite job is missing.

Maintainers create a release only from a reviewed `main` commit:

```bash
git tag -a v0.1.0 -m "CyberPilot v0.1.0"
git push origin v0.1.0
```

## Configure

Initialization configures exactly one OpenAI-compatible model and one local Docker or Podman runner. It probes typed tool calls and structured output before saving, then verifies a disposable OCI lifecycle using an image already present locally. Initialization never silently downloads host tools or images.

```bash
export CYBERPILOT_SANDBOX_IMAGE=cyberpilot-sandbox:v1
cyberpilot init
cyberpilot config
```

The API key is read without terminal echo and stored in macOS Keychain, Windows Credential Manager, or Linux Secret Service. Linux also supports explicit `env:VARIABLE_NAME` references. YAML contains only the opaque credential reference. `cyberpilot config` displays a redacted profile.

## Sessions

Running `cyberpilot` opens the TUI. The overview separates `NEEDS INPUT` from `OTHER SESSIONS`; quitting the TUI does not terminate background sessions.

Non-interactive execution accepts one explicit prompt source:

```bash
cyberpilot exec --detach --json \
  "Assess https://127.0.0.1:8443 for authorized API object-access issues"

printf '%s\n' "Assess https://127.0.0.1:8443" | cyberpilot exec -
cyberpilot exec --prompt-file objective.txt --max-actions 20 --timeout 30m
```

`exec` progress is written to stderr and the selected final format is written once to stdout. Exit codes are 0 completed/no finding, 1 completed/verified findings, 2 needs input or blocked, 3 configuration/runtime failure, and 4 invalid input or scope. `--detach` returns after durable session creation.

## Runtime and data

Platform configuration and data directories follow native per-user conventions. Unix directories are mode `0700`, local RPC sockets are `0600`, and Windows uses a per-user named pipe. SQLite WAL stores append-only events and current projections. Large evidence is stored by SHA-256 under the user data directory. Protected raw evidence remains local; outward model, log, terminal, and export boundaries receive redacted content.

One persistent non-root sandbox is allocated per session. The release image contains shell, curl, Python, CA roots, and a small helper. It is created read-only with dropped capabilities, resource limits, no runtime socket, no host credentials, and network disabled by default. Shell and Python actions operate on a dedicated bind-mounted workspace. Its per-user parent remains host-owner-only (`0700`); the isolated leaf is writable by the fixed non-root container UID so Linux bind mounts work without starting the sandbox as root. Target HTTP actions use the daemon's scoped network broker, which revalidates destination, DNS results, redirects, and request rates. Docker and Podman are accessed only from the host daemon through argument-array CLI adapters.

## Skills

CyberPilot accepts focused, licensed common-format `SKILL.md` directories from configured local sources or existing Git checkouts pinned to an exact commit. Imported repository content is read, never executed during refresh. Scripts and dependencies remain untrusted and require a separately approved sandbox action. See [docs/SKILL_STANDARD.md](docs/SKILL_STANDARD.md).

## Threat model and limits

Model output, target responses, skills, repositories, scripts, and tool output are untrusted. They cannot expand scope or grant approval. Policy returns `allow`, `ask`, or `deny` before execution and rechecks hostname, confirmed resolution, redirects, and rates through the scoped broker.

V1 covers Web/API investigation with shell, curl, and Python. It intentionally excludes browser automation, Kubernetes/remote runners, multi-user tenancy, mobile, Active Directory, binary exploitation, arbitrary host execution, implicit dependency installation, and multi-model orchestration. Missing browser capability is reported as a coverage gap rather than installed automatically.

## Provider checks

Docker lifecycle and complete agent-path acceptance run in CI. A release operator with rootless Podman runs the equivalent lifecycle and full-path checks:

```bash
export CYBERPILOT_CONFORMANCE_IMAGE=cyberpilot-sandbox:v1
scripts/check-rootless-podman.sh
```

Tests use local fixtures only. Automated test configuration must never contain public target addresses.
