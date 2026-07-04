#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${1:-$ROOT_DIR/podman-compose.yml}"

if ! command -v podman >/dev/null 2>&1; then
  echo "podman not found in PATH" >&2
  exit 3
fi

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "compose file not found: $COMPOSE_FILE" >&2
  exit 2
fi

echo "[podman-validate] podman compose provider"
COMPOSE_PROVIDER="${PODMAN_COMPOSE_PROVIDER:-$(command -v podman-compose 2>/dev/null || true)}"
if [[ -z "$COMPOSE_PROVIDER" || ! -x "$COMPOSE_PROVIDER" ]]; then
  echo "podman-compose provider not found; set PODMAN_COMPOSE_PROVIDER" >&2
  exit 3
fi
PROVIDER_HELP="$("$COMPOSE_PROVIDER" --help 2>&1 || true)"
if ! grep -q -- '--podman-build-args' <<<"$PROVIDER_HELP" ||
   ! grep -q -- '--podman-run-args' <<<"$PROVIDER_HELP"; then
  echo "compose provider does not support podman build/run arg injection: $COMPOSE_PROVIDER" >&2
  exit 3
fi
PODMAN_COMPOSE_PROVIDER="$COMPOSE_PROVIDER" podman compose version >/dev/null

echo "[podman-validate] rendering compose config"
CONFIG="$(PODMAN_COMPOSE_PROVIDER="$COMPOSE_PROVIDER" podman compose -f "$COMPOSE_FILE" config)"

for service in new-api postgres redis; do
  if ! grep -Eq "^[[:space:]]{2}${service}:" <<<"$CONFIG"; then
    echo "missing service in rendered config: $service" >&2
    exit 2
  fi
done

echo "[podman-validate] checking cpu caps"
if [[ "$(grep -Ec '^[[:space:]]+cpus: 2(\.0)?$' <<<"$CONFIG")" -lt 3 ]]; then
  echo "rendered config is missing cpus: 2 for one or more services" >&2
  exit 2
fi

if [[ "$(grep -Ec '^[[:space:]]+cpuset: 0-1$' <<<"$CONFIG")" -lt 3 ]]; then
  echo "rendered config is missing cpuset: 0-1 for one or more services" >&2
  exit 2
fi

echo "[podman-validate] checking memory caps"
if [[ "$(grep -Eci '^[[:space:]]+mem_limit: 11987m$' <<<"$CONFIG")" -lt 3 ]]; then
  echo "rendered config is missing mem_limit: 11987M for one or more services" >&2
  exit 2
fi

if [[ "$(grep -Eci '^[[:space:]]+memswap_limit: 11987m$' <<<"$CONFIG")" -lt 3 ]]; then
  echo "rendered config is missing memswap_limit: 11987M for one or more services" >&2
  exit 2
fi

if grep -Eq "^[[:space:]]+ports:" <<<"$CONFIG" &&
   ! grep -Eq "3000/tcp|3001:3000|target: 3000" <<<"$CONFIG"; then
  echo "rendered config does not expose backend port 3000" >&2
  exit 2
fi

echo "[podman-validate] OK: $COMPOSE_FILE"
