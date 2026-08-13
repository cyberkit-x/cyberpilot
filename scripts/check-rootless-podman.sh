#!/bin/sh
set -eu

command -v podman >/dev/null 2>&1 || { echo "rootless Podman is required" >&2; exit 1; }
rootless="$(podman info --format '{{.Host.Security.Rootless}}')"
[ "$rootless" = "true" ] || { echo "Podman must run rootless" >&2; exit 1; }
: "${CYBERPILOT_CONFORMANCE_IMAGE:?set CYBERPILOT_CONFORMANCE_IMAGE to the pinned local sandbox image}"
CYBERPILOT_CONFORMANCE_PROVIDER=podman go test ./internal/runner/oci -run '^TestProviderConformance$' -count=1 -v
CYBERPILOT_ACCEPTANCE_PROVIDER=podman go test ./internal/runtime -run '^TestFullOCIExecEvidenceRestartAndTUIAcceptance$' -count=1 -v
