#!/bin/sh
set -eu

case "${1:-}" in
  hold)
    trap 'exit 0' TERM INT
    while :; do sleep 3600 & wait "$!"; done
    ;;
  exec)
    shift
    exec "$@"
    ;;
  *)
    echo "usage: cyberpilot-runner hold | exec <command> [args...]" >&2
    exit 64
    ;;
esac
