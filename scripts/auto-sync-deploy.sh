#!/usr/bin/env bash
#
# Dispatch the fork-only upstream sync, wait for the fork GHCR image, then
# deploy the latest image through the local Podman/systemd runtime.
#
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"
GUARD="$SCRIPT_DIR/fork-sync-guard.sh"
FORK_REPO="${FORK_REPO:-giovannimnz/router-ai-atius}"
SYNC_STRATEGY="${SYNC_STRATEGY:-theirs}"
WAIT_FOR_WORKFLOWS="${WAIT_FOR_WORKFLOWS:-true}"
DEPLOY_AFTER_GHCR="${DEPLOY_AFTER_GHCR:-true}"
DEPLOY_TAG="${DEPLOY_TAG:-latest}"
LOG="$REPO_ROOT/logs/auto-sync-deploy.log"

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

wait_for_workflow_run() {
  local workflow="$1"
  local started_at="$2"
  local timeout_seconds="$3"
  local deadline=$((SECONDS + timeout_seconds))
  local run_id=""

  while (( SECONDS < deadline )); do
    run_id="$(gh run list \
      --repo "$FORK_REPO" \
      --workflow "$workflow" \
      --limit 20 \
      --json databaseId,createdAt,event \
      --jq "map(select(.event == \"workflow_dispatch\" and .createdAt >= \"$started_at\")) | sort_by(.createdAt) | reverse | .[0].databaseId // empty")"

    if [[ -n "$run_id" ]]; then
      echo "$run_id"
      return 0
    fi

    sleep 10
  done

  return 1
}

cd "$REPO_ROOT"

require_cmd gh
require_cmd git

log "=== AUTO SYNC DEPLOY START ==="
log "Fork repo: $FORK_REPO"
log "Sync strategy: $SYNC_STRATEGY"

"$GUARD" configure-remotes

started_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
log "Dispatching Sync Upstream + Release on fork..."
"$GUARD" workflow-run sync.yml --ref main -f strategy="$SYNC_STRATEGY"

if [[ "$WAIT_FOR_WORKFLOWS" != "true" ]]; then
  log "WAIT_FOR_WORKFLOWS=false; dispatched sync and exiting."
  exit 0
fi

sync_run="$(wait_for_workflow_run sync.yml "$started_at" 180 || true)"
if [[ -z "$sync_run" ]]; then
  log "ERROR: could not find dispatched sync.yml run for $FORK_REPO"
  exit 1
fi

log "Watching sync run: https://github.com/$FORK_REPO/actions/runs/$sync_run"
gh run watch "$sync_run" --repo "$FORK_REPO" --exit-status

if [[ "$DEPLOY_AFTER_GHCR" != "true" ]]; then
  log "DEPLOY_AFTER_GHCR=false; sync succeeded and deploy is disabled."
  exit 0
fi

log "Waiting for docker-build.yml run dispatched by sync, if upstream changed..."
docker_run="$(wait_for_workflow_run docker-build.yml "$started_at" 900 || true)"
if [[ -z "$docker_run" ]]; then
  log "No docker-build.yml run found after sync. Upstream was probably already current; deploy skipped."
  exit 0
fi

log "Watching GHCR build run: https://github.com/$FORK_REPO/actions/runs/$docker_run"
gh run watch "$docker_run" --repo "$FORK_REPO" --exit-status

log "Deploying GHCR image tag: $DEPLOY_TAG"
"$SCRIPT_DIR/pull-and-restart.sh" "$DEPLOY_TAG"

log "=== AUTO SYNC DEPLOY COMPLETE ==="
