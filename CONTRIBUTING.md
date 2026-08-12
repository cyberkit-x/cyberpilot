# Contributing to CyberPilot

CyberPilot aims to turn reusable security practice into a safe, testable open-source agent runtime. Contributions are welcome from practitioners and engineers; a useful contribution does not need to be large.

## Good first contributions

- improve one focused attack skill;
- add a reproducible local evaluation case;
- improve evidence capture or false-positive handling;
- add tests for Docker or Podman compatibility;
- improve CLI/TUI clarity without changing the product contract;
- document a concrete runtime failure or coverage gap.

Please open an issue before implementing a broad architecture change, new execution backend, new model abstraction, or new top-level command. A small fix or focused skill can go directly to a pull request.

## Pull request requirements

Every pull request should state:

- the user or contributor problem it solves;
- what behavior changed;
- how it was tested;
- any safety, authorization, compatibility, or licensing impact;
- what remains intentionally out of scope.

Functional changes require relevant automated tests. Security testing must use targets owned by the contributor or intentionally vulnerable local fixtures. Do not include real credentials, customer data, undisclosed vulnerabilities, or evidence collected without authorization.

## Contributing a skill

Read [docs/SKILL_STANDARD.md](docs/SKILL_STANDARD.md). The smallest valid contribution is:

```text
skills/<skill-name>/
└── SKILL.md
```

A skill contribution must be focused, routable, actionable, evidence-aware, safe, attributable, and covered by an explicit compatible license. Scripts and references are optional; their presence does not make a skill higher quality.

Newly accepted skills begin as **Compatible**. Promotion to **Tested** or **Maintained** depends on reproducible evidence, not popularity or author identity.

## Third-party material

Linking to a repository does not grant permission to redistribute it. When adapting third-party work:

- verify the license before copying anything;
- preserve required copyright and attribution notices;
- identify the upstream source and meaningful modifications;
- avoid importing an entire collection when only one focused skill is needed.

Repositories without an explicit license may be referenced for discovery but must not be copied, modified, or bundled.

## Review priorities

Reviews prioritize, in order:

1. authorization and operator safety;
2. evidence integrity and false-positive resistance;
3. usefulness in real red-team work;
4. interoperability and maintainability;
5. breadth of features or content.

The project may decline technically impressive contributions that expand scope without strengthening the core hunt loop.

