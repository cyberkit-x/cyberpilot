# CyberPilot

CyberPilot is an open-source agentic security operator for authorized vulnerability hunting.

Users define the objective, targets, and boundaries. CyberPilot decides what to investigate next, selects relevant community skills, acts through a controlled runner, evaluates the evidence, and replans until the task reaches a meaningful stopping point.

```text
Objective + Scope
       ↓
Observe → Reason → Act → Verify → Replan
       ↓
Reproducible findings and explicit coverage gaps
```

CyberPilot is not a vulnerability scanner with an AI chat interface, a fixed penetration-testing workflow, or a collection of payloads. The execution path is dynamic; authorization and evidence requirements are deterministic.

## Product principles

- **Goal-driven:** users state security goals instead of selecting a scan template.
- **Agentic:** the model chooses and revises actions from current evidence.
- **Evidence-first:** a suspicious signal is a lead, not a verified vulnerability.
- **Human-governed:** scope, policy, and approval boundaries always outrank skills and model decisions.
- **Community-powered:** security practitioners contribute focused, reusable attack skills.
- **Open and portable:** model providers, skills, and execution backends must remain replaceable.
- **Terminal-native:** CLI/TUI is the primary interface for expert operators and automation.

## Initial product boundary

The first release will validate one complete loop for authorized Web/API testing:

```bash
cyberpilot init
cyberpilot exec "Assess https://example.com for authorization weaknesses"
cyberpilot
```

The initial runtime is expected to provide:

- natural-language task creation;
- persistent task sessions shared by CLI and TUI;
- non-interactive execution through `cyberpilot exec`;
- shell, curl, Python, and filesystem capabilities;
- a Docker or Podman sandbox selected during initialization;
- dynamic discovery and loading of focused `SKILL.md` files;
- explicit `Lead`, `Verified Finding`, and `Blocked` outcomes;
- evidence and an audit trail for every reportable finding.

## Build and release targets

CyberPilot is implemented as a Go command and distributed as a native, statically linked binary. Pull requests and changes to `main` build all supported targets; a `v*` tag publishes the same packages to GitHub Releases with SHA-256 checksums.

| Platform | Architecture | Release format |
|---|---|---|
| Linux | x86-64 (`amd64`) | `.tar.gz` |
| Linux | ARM64 (`arm64`) | `.tar.gz` |
| macOS | Intel (`amd64`) | `.tar.gz` |
| macOS | Apple Silicon (`arm64`) | `.tar.gz` |
| Windows | x86-64 (`amd64`) | `.zip` |

Build and test locally with:

```bash
make check
make build
./bin/cyberpilot version
```

Browser automation, Kubernetes runners, team collaboration, distributed execution, mobile, Active Directory, and binary exploitation are future capabilities, not requirements for the first usable release.

## Skills

A CyberPilot skill captures a focused piece of practitioner knowledge: when a security hypothesis is worth pursuing, what to inspect, how to test it safely, and what evidence is required before reporting it.

Skills guide reasoning; they do not override policy, directly control the host, or impose a universal workflow. A minimal skill is a directory containing one `SKILL.md` file. References and scripts are optional and loaded only when needed.

See [Skill acceptance standard](docs/SKILL_STANDARD.md) before proposing or importing a skill.

## Trust model

Third-party skills and scripts are untrusted input. CyberPilot must never silently expand scope, mount a container-runtime socket into a task sandbox, install host dependencies, or execute destructive actions because a skill requested it.

Skill status describes evidence about compatibility and quality:

- **Compatible:** readable by CyberPilot; effectiveness has not been established.
- **Tested:** exercised against a reproducible authorized target.
- **Maintained:** actively reviewed and tested by CyberPilot maintainers.

No label means that a skill is safe or effective in every environment.

## Contributing

CyberPilot welcomes focused contributions from red teamers, security researchers, tool authors, agent engineers, and platform engineers. Start with [CONTRIBUTING.md](CONTRIBUTING.md).

The project values small, testable improvements over large collections of unverified content. New functionality must preserve authorization boundaries, observable behavior, and provider independence.

## Safety and legal use

CyberPilot is intended only for systems you own or are explicitly authorized to test. Contributors and users are responsible for complying with applicable law and rules of engagement. Safety controls are part of the product contract, not optional prompt guidance.

## Status

CyberPilot is in the architecture and bootstrap stage. Interfaces described here establish project direction and may evolve before the first tagged release.
