# CyberPilot Skill Acceptance Standard

## Purpose

A CyberPilot skill should help the agent choose a better next action in a recognizable security situation and determine whether the resulting hypothesis is supported or rejected.

A skill is practitioner guidance, not an authorization grant, executable workflow engine, tool installer, vulnerability finding, or claim of guaranteed effectiveness.

## Minimum structure

```text
<skill-name>/
└── SKILL.md
```

`SKILL.md` must contain YAML frontmatter with:

```yaml
---
name: validate-object-authorization
description: >-
  Use when an authenticated application exposes object identifiers and the
  agent needs to determine whether access is restricted to the owning identity.
license: MIT
---
```

Requirements:

- `name` uses lowercase letters, digits, and hyphens, is shorter than 64 characters, and matches the directory name;
- `description` states both the observable trigger and the intended investigation;
- the applicable license is explicit; repository-level licensing is acceptable when it clearly covers the skill;
- upstream source and attribution are retained when the content is adapted.

Optional directories are:

```text
references/   # detailed knowledge loaded only when relevant
scripts/      # deterministic reusable programs, never implicitly trusted
assets/       # files used as inputs or outputs, not prompt context
```

Keep references one level from `SKILL.md`. Do not add a README, changelog, installation guide, or duplicated quick-reference file inside an individual skill.

## Acceptance requirements

All six requirements are mandatory.

### 1. Focused and routable

The skill addresses a bounded situation such as JWT claim trust, IDOR validation, OAuth redirect handling, subdomain discovery, or PHP archive extraction.

The `name` and `description` alone must be sufficient for first-pass retrieval. Broad identities such as “security expert” and claims such as “complete penetration testing” are not accepted.

### 2. Actionable

The body identifies useful observations, plausible hypotheses, and safe next experiments. Pure definitions, framework mappings, payload lists, and general educational articles are reference material rather than standalone skills.

The skill may recommend strategies and alternatives, but must not require a universal fixed sequence when the correct action depends on evidence.

### 3. Falsifiable and evidence-aware

The skill explains:

- what is only a lead;
- what evidence supports promotion to a finding;
- what controls or counterexamples can reject the hypothesis;
- which prerequisites and threat-model assumptions must be reported.

An HTTP status code, scanner match, source-code pattern, or model assertion alone is not sufficient proof of a vulnerability.

### 4. Safe under policy control

The skill identifies actions that may alter state, affect availability, access sensitive data, increase traffic, use credentials, or cross a trust boundary.

It must not instruct the agent to evade CyberPilot policy, expand target scope, conceal actions, install host software, mount runtime sockets, or perform destructive verification by default. Skill instructions never override `allow`, `ask`, or `deny` decisions made by the runtime.

### 5. Portable

Prefer capability-level instructions such as making an HTTP request or decoding a token over assuming one branded tool. If a specific program is required, declare the requirement and provide a fallback or a clear blocked outcome.

Do not assume a particular model provider, host operating system, container provider, VPN, browser, or private service unless the skill is explicitly scoped to it.

### 6. Legally redistributable and attributable

The skill must have a clear license compatible with distribution. It must not contain real secrets, customer data, undisclosed vulnerability details, malicious binaries, or unattributed copied material.

Content with no explicit license can be listed as an external discovery source but cannot enter the CyberPilot registry or repository.

## Recommended body

The headings are recommendations, not a forced workflow:

```markdown
# Validate Object Authorization

## Use when
Signals that should trigger this skill.

## Do not use when
Similar-looking situations outside its scope.

## Investigate
Important observations, hypotheses, and pivots.

## Validate
Safe experiments and meaningful controls.

## Evidence required
The minimum proof for a verified finding.

## Safety
Actions requiring approval or prohibited by default.
```

Keep the primary file concise and normally below 500 lines. Move detailed framework variations, examples, payload families, or background knowledge to directly linked references and state exactly when each reference is useful.

## Scripts and dependencies

Scripts are untrusted code even when included in a trusted repository. A skill may declare a dependency but must not silently install it.

The following patterns require explicit runtime handling and must never be automatically executed from imported instructions:

```text
sudo ...
apt/brew/pip/npm install ...
curl ... | sh
docker/podman ...
access to container-runtime sockets
host filesystem or credential-store access
```

CyberPilot executes approved scripts only inside the selected runner, applies task scope and network policy, captures outputs, and records the script version in evidence.

## Registry status

Acceptance and effectiveness are separate:

- **Compatible:** structure, license, and baseline safety checks pass. Effectiveness is unverified.
- **Tested:** reproducible evaluation demonstrates correct retrieval and useful behavior on at least one authorized target, including a negative or control case where practical.
- **Maintained:** CyberPilot maintainers actively review the skill and its evaluation coverage across supported releases.

Status applies to a specific skill version. Popularity, repository stars, number of payloads, and author reputation do not determine status.

## Grounds for rejection

A proposed skill is rejected or returned for revision when it:

- is primarily a generic article, payload dump, link list, or marketing claim;
- duplicates an existing skill without a materially better scope or method;
- cannot be selected reliably from its metadata;
- reports findings without reachability, authorization, control, or impact evidence;
- assumes that a target must be vulnerable, as many CTF guides do;
- embeds a long fixed workflow that should be a task type or runtime feature;
- attempts to weaken policy or execute destructive actions by default;
- automatically installs dependencies or modifies the host;
- has ambiguous provenance or no redistributable license;
- claims measured effectiveness without reproducible evaluation artifacts.

## Evaluation expectations

Automated evaluation is not required for an initial **Compatible** submission, but promotion requires evidence. A useful evaluation records:

- the authorized, reproducible target and version;
- the prompt and relevant session conditions;
- whether the skill was retrieved in positive and unrelated scenarios;
- actions proposed and executed;
- expected and observed evidence;
- whether the outcome was a verified finding, rejected hypothesis, or blocked path;
- model, skill, and runtime versions needed to reproduce the result.

All automated evaluations must use local fixtures or targets explicitly designed and authorized for testing. They must not scan public systems.
