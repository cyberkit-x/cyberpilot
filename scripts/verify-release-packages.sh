#!/bin/sh
set -eu

dist="${1:-dist}"
[ -d "$dist" ] || { echo "release directory does not exist: $dist" >&2; exit 1; }

version="${EXPECTED_VERSION:-}"
if [ -n "$version" ]; then
  if ! printf '%s\n' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "release tag must use semantic version form vMAJOR.MINOR.PATCH: $version" >&2
    exit 1
  fi
  prefix="cyberpilot_${version}_"
else
  prefix="cyberpilot_*_"
fi

require_one() {
  pattern="$1"
  label="$2"
  set -- "$dist"/$pattern
  [ "$#" -eq 1 ] && [ -f "$1" ] || { echo "expected exactly one package for $label" >&2; exit 1; }
}

require_one "${prefix}linux_amd64.tar.gz" "Linux amd64"
require_one "${prefix}linux_arm64.tar.gz" "Linux arm64"
require_one "${prefix}darwin_amd64.tar.gz" "macOS Intel"
require_one "${prefix}darwin_arm64.tar.gz" "macOS Apple Silicon"
require_one "${prefix}windows_amd64.zip" "Windows amd64"

[ -f "$dist/checksums.txt" ] || { echo "missing checksums" >&2; exit 1; }
[ -f "$dist/cyberpilot-release.spdx.json" ] || { echo "missing SBOM" >&2; exit 1; }

expected_lines=6
actual_lines="$(wc -l < "$dist/checksums.txt" | tr -d ' ')"
[ "$actual_lines" -eq "$expected_lines" ] || { echo "checksums must cover five packages and the SBOM" >&2; exit 1; }

(cd "$dist" && sha256sum -c checksums.txt)
