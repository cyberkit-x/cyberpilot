#!/bin/sh
set -eu

dist="${1:-dist}"
for platform in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64; do
  found="$(find "$dist" -maxdepth 1 -type f -name "cyberpilot_*_${platform}.tar.gz" -o -name "cyberpilot_*_${platform}.zip" | head -1)"
  [ -n "$found" ] || { echo "missing package for $platform" >&2; exit 1; }
done
[ -f "$dist/checksums.txt" ] || { echo "missing checksums" >&2; exit 1; }
[ -f "$dist/cyberpilot-release.spdx.json" ] || { echo "missing SBOM" >&2; exit 1; }
