# OCI provider prerequisites

CyberPilot keeps Docker and Podman behind the same provider-neutral Runner contract. Domain behavior, policy decisions, evidence, and session state must not vary by provider.

## Linux

- Docker Engine: the current user must be authorized to use the selected Docker context.
- Podman: use rootless Podman with a functioning user connection. Verify `podman info --format '{{.Host.Security.Rootless}}'` returns `true`.

## macOS

- Docker Desktop or OrbStack may expose a Docker context.
- Podman requires a running rootless Podman machine. Start and verify the machine before `cyberpilot init`.

## Windows

- Docker Desktop must use Linux containers, or
- Podman machine must be initialized and running with a usable connection.

Windows task sandboxes are Linux containers; CyberPilot does not silently fall back to host PowerShell or command execution.

## Release conformance

Docker conformance runs in CI. Rootless Podman must be checked on a release host with:

```bash
export CYBERPILOT_CONFORMANCE_IMAGE=cyberpilot-sandbox:v1
scripts/check-rootless-podman.sh
```

The check runs both provider lifecycle conformance and the full session/model/Skill/policy/action/evidence/restart/TUI acceptance path. It fails when Podman is absent, rootful, the pinned image is unavailable, or any lifecycle or product-path operation fails. A missing provider is never treated as a skipped successful release check.
