#!/usr/bin/env bash
#
# Pull the fork GHCR image and restart the managed Podman user unit.
#
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"
IMAGE="${IMAGE:-ghcr.io/giovannimnz/router-ai-atius}"
TAG="${1:-${TAG:-latest}}"
SERVICE="${SERVICE:-container-router-ai-atius.service}"
CONTAINER="${CONTAINER:-router-ai-atius}"
LOG="$REPO_ROOT/logs/auto-update.log"
FORCE_RESTART="${FORCE_RESTART:-false}"
ENV_FILE="${ENV_FILE:-/home/ubuntu/.config/router-ai-atius/.env}"

mkdir -p "$(dirname "$LOG")"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    log "ERROR: missing required command: $1"
    exit 1
  }
}

health_ok() {
  local url
  for url in \
    "http://127.0.0.1:3030/api/status" \
    "http://127.0.0.1:3000/api/status" \
    "http://127.0.0.1:3030/health" \
    "http://127.0.0.1:3000/health"; do
    if curl -fsS --max-time 3 "$url" >/dev/null 2>&1; then
      log "Health OK: $url"
      return 0
    fi
  done
  return 1
}

require_cmd podman
require_cmd systemctl
require_cmd curl

cd "$REPO_ROOT"

load_optional_env_var() {
  local key="$1"
  local line value
  [[ -f "$ENV_FILE" ]] || return 0
  line="$(grep -E "^${key}=" "$ENV_FILE" | tail -1 || true)"
  [[ -n "$line" ]] || return 0
  value="${line#*=}"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  export "$key=$value"
}

load_optional_env_var GHCR_USER
load_optional_env_var GHCR_TOKEN

if [[ -n "${GHCR_TOKEN:-}" ]]; then
  printf '%s' "$GHCR_TOKEN" | podman login ghcr.io --username "${GHCR_USER:-$USER}" --password-stdin >/dev/null 2>&1
  log "Authenticated to GHCR as ${GHCR_USER:-$USER}"
fi

current_image_id="$(podman inspect "$CONTAINER" --format '{{.Image}}' 2>/dev/null || true)"

log "Pulling ${IMAGE}:${TAG}"
podman pull "${IMAGE}:${TAG}"

if [[ "$TAG" != "latest" ]]; then
  log "Retagging ${IMAGE}:${TAG} as ${IMAGE}:latest for $SERVICE"
  podman tag "${IMAGE}:${TAG}" "${IMAGE}:latest"
fi

new_image_id="$(podman image inspect "${IMAGE}:latest" --format '{{.Id}}' 2>/dev/null || true)"
if [[ -n "$current_image_id" && -n "$new_image_id" && "$current_image_id" == "$new_image_id" && "$FORCE_RESTART" != "true" ]]; then
  log "Image unchanged; restart skipped. Set FORCE_RESTART=true to restart anyway."
  exit 0
fi

log "Restarting user unit: $SERVICE"
systemctl --user restart "$SERVICE"

for _ in {1..45}; do
  if systemctl --user is-active --quiet "$SERVICE" && health_ok; then
    podman inspect "$CONTAINER" --format 'Container={{.Name}} Image={{.ImageName}} Started={{.State.StartedAt}}' 2>/dev/null | tee -a "$LOG" || true
    exit 0
  fi
  sleep 2
done

log "ERROR: $SERVICE did not become healthy after restart"
systemctl --user --no-pager status "$SERVICE" | tail -40 | tee -a "$LOG" || true
exit 1
