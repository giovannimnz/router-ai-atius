#!/bin/bash
#
# pull-and-restart.sh — Pull latest GHCR image and restart new-api container
# Run via cron: 0 * * * * /home/ubuntu/docker/Atius/router-ai-atius/scripts/pull-and-restart.sh
#
set -euo pipefail

REPO_DIR="/home/ubuntu/docker/Atius/router-ai-atius"
IMAGE="ghcr.io/giovannimnz/atius-ai-router"
LOG="$REPO_DIR/logs/auto-update.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG"
}

cd "$REPO_DIR"

# Source GHCR credentials from .env
if [[ -f "$REPO_DIR/.env" ]]; then
    set -a
    source "$REPO_DIR/.env"
    set +a
fi

# Login to GHCR if token is available
if [[ -n "${GHCR_TOKEN:-}" ]]; then
    echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin > /dev/null 2>&1
    log "Logged in to GHCR as $GHCR_USER"
fi

# Get current container version
CURRENT_VERSION=$(docker exec new-api /new-api --version 2>/dev/null || echo "unknown")
log "Current container version: $CURRENT_VERSION"

# Pull latest image
log "Pulling latest image from GHCR..."
if ! docker pull "${IMAGE}:latest" 2>&1 | tail -3; then
    log "ERROR: Failed to pull ${IMAGE}:latest"
    exit 1
fi

# Get new image tag/digest
NEW_DIGEST=$(docker inspect "${IMAGE}:latest" --format '{{index .RepoDigests 0}}' 2>/dev/null || echo "unknown")
log "New image digest: $NEW_DIGEST"

# Check if image changed (compare digests)
CURRENT_DIGEST=$(docker inspect "new-api" --format '{{index .RepoDigests 0}}' 2>/dev/null || echo "")
if [[ "$CURRENT_DIGEST" == "$NEW_DIGEST" ]]; then
    log "Image unchanged, skipping restart"
    exit 0
fi

log "Image updated, restarting new-api container..."

# Restart the container
docker compose -f "$REPO_DIR/docker-compose.yml" up -d new-api

# Wait for healthy
sleep 5
for i in {1..30}; do
    HEALTH=$(docker inspect --format='{{.State.Health.Status}}' new-api 2>/dev/null || echo "none")
    if [[ "$HEALTH" == "healthy" ]]; then
        break
    fi
    sleep 2
done

NEW_VERSION=$(docker exec new-api /new-api --version 2>/dev/null || echo "unknown")
log "Container restarted successfully. New version: $NEW_VERSION"

# Prune old images to save space
docker image prune -f >> /dev/null 2>&1 || true
