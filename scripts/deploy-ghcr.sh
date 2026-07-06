#!/usr/bin/env bash
#
# Compatibility wrapper for the GHCR -> Podman/systemd deploy path.
#
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/pull-and-restart.sh" "${1:-latest}"
