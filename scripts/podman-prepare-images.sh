#!/usr/bin/env bash
# podman-prepare-images.sh
#
# Make sure the :latest images that podman-compose.yml and the quadlets
# reference actually exist on this host. Without this step,
# `./scripts/podman-up.sh` will fail with "image not known" because the
# compose file ships with :latest as the canonical tag.
#
# Strategy (first one that succeeds wins):
#   1. If :latest already exists, do nothing (idempotent).
#   2. Try to build from source (./Dockerfile present).
#   3. Try to pull :latest from GHCR.
#   4. Fail loud with a clear next-step.
#
# Pinning a specific version (optional):
#   ROUTER_AI_ATIUS_VERSION=v2.11.1-rebrand ./scripts/podman-prepare-images.sh
#   → pulls that tag and re-tags it as :latest (operator's choice).
#   Default behavior pulls :latest directly (no re-tag needed).
#
# Usage:
#   ./scripts/podman-prepare-images.sh                   # all images
#   ./scripts/podman-prepare-images.sh router-ai-atius  # just one
#
# Env (optional):
#   ROUTER_AI_ATIUS_VERSION   version tag to pull when (3) is taken;
#                            if set, pulled and re-tagged as :latest.
#                            Default: unset → pulls :latest directly.

set -euo pipefail
cd "$(dirname "$0")/.."

REGISTRY="ghcr.io/giovannimnz"
ROUTER_TAG="${ROUTER_AI_ATIUS_VERSION:-}"  # empty = use :latest
ROUTER_LOCAL="${REGISTRY}/router-ai-atius:latest"

# model-detailed is a build-only image (no registry).
MODEL_LOCAL="router-ai-atius-model-detailed:latest"

log() { echo "[prepare-images] $*"; }
err() { echo "[prepare-images] ERROR: $*" >&2; }

# --- preflight ----------------------------------------------------------
command -v podman >/dev/null || { err "podman not installed"; exit 1; }

# --- image: router-ai-atius:latest -------------------------------------
ensure_router_ai_atius() {
  if podman image exists "$ROUTER_LOCAL"; then
    log "OK   $ROUTER_LOCAL already present"
    podman image inspect "$ROUTER_LOCAL" --format '  → {{.Id}} ({{.Created}})'
    return 0
  fi

  log "$ROUTER_LOCAL missing — populating..."

  # Option A: build from source if a Dockerfile is at the repo root
  if [ -f Dockerfile ]; then
    log "  trying build from ./Dockerfile"
    if podman build -t "$ROUTER_LOCAL" . ; then
      log "  built OK"
      return 0
    fi
    err "  build failed; trying pull instead"
  fi

  # Option B: pull from GHCR. If ROUTER_AI_ATIUS_VERSION is set, pull
  # that pin and re-tag as :latest. Otherwise pull :latest directly.
  if [ -n "$ROUTER_TAG" ]; then
    REMOTE="${REGISTRY}/router-ai-atius:${ROUTER_TAG}"
    log "  pulling $REMOTE (will re-tag as :latest)"
    if podman pull "$REMOTE"; then
      podman tag "$REMOTE" "$ROUTER_LOCAL"
      log "  pulled and re-tagged as $ROUTER_LOCAL"
      return 0
    fi
    err "  pull of $REMOTE failed; trying :latest instead"
  fi

  log "  pulling $ROUTER_LOCAL"
  if podman pull "$ROUTER_LOCAL"; then
    log "  pulled OK"
    return 0
  fi

  err "could not populate $ROUTER_LOCAL"
  err "manual steps:"
  err "  podman build -t $ROUTER_LOCAL ."
  err "  OR"
  err "  podman pull $ROUTER_LOCAL"
  err "  OR pin a version:  ROUTER_AI_ATIUS_VERSION=v2.11.1-rebrand \\"
  err "       podman pull ${REGISTRY}/router-ai-atius:v2.11.1-rebrand && \\"
  err "       podman tag  ${REGISTRY}/router-ai-atius:v2.11.1-rebrand $ROUTER_LOCAL"
  exit 2
}

# --- image: router-ai-atius-model-detailed:latest ----------------------
# Always built locally — no upstream registry for this one.
ensure_model_detailed() {
  if podman image exists "$MODEL_LOCAL"; then
    log "OK   $MODEL_LOCAL already present"
    return 0
  fi

  log "$MODEL_LOCAL missing — building..."
  if [ -f integration/middleware/Dockerfile.fastapi ]; then
    podman build -t "$MODEL_LOCAL" -f integration/middleware/Dockerfile.fastapi integration/middleware
    log "  built OK"
  else
    err "integration/middleware/Dockerfile.fastapi not found"
    err "model-detailed must be built from local source (no registry copy)"
    exit 2
  fi
}

# --- selective vs full -------------------------------------------------
TARGETS=("$@")
if [ ${#TARGETS[@]} -eq 0 ]; then
  TARGETS=(router-ai-atius router-ai-atius-model-detailed)
fi

for t in "${TARGETS[@]}"; do
  case "$t" in
    router-ai-atius)                ensure_router_ai_atius ;;
    router-ai-atius-model-detailed) ensure_model_detailed ;;
    *)
      err "unknown image target: $t"
      err "valid: router-ai-atius, router-ai-atius-model-detailed"
      exit 1
      ;;
  esac
done

log ""
log "all :latest images present. next: ./scripts/podman-up.sh"
log ""
podman images --format "table {{.Repository}}:{{.Tag}}\t{{.CreatedSince}}\t{{.Size}}" \
  | grep -E "(router-ai-atius|REPOSITORY)" || true
