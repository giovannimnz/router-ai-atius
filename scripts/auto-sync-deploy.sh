#!/bin/bash
# Auto-sync-deploy: Sincroniza upstream + build local + restart container
# Run: manual ou via cron

set -e
CDIR="$(cd "$(dirname "$0")" && pwd)"
LOG="$CDIR/../logs/auto-sync-deploy.log"
IMG="ghcr.io/giovannimnz/router-ai-atius"

mkdir -p "$(dirname "$LOG")"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG"
}

cd "$CDIR/.."

log "=== INICIO AUTO-SYNC-DEPLOY ==="

# 1. Sync upstream
log "Sincronizando com upstream..."
git fetch upstream

# Get latest upstream tag
UPSTREAM_TAG=$(timeout 30 git ls-remote upstream --tags 2>/dev/null | awk -F'/' '{print $3}' | grep -E '^v[0-9]' | sort -V | tail -1)
UPSTREAM_CLEAN=$(echo "$UPSTREAM_TAG" | sed 's/^v//')
CURRENT_TAG=$(cat VERSION)

log "Upstream latest tag: $UPSTREAM_TAG | Current VERSION: $CURRENT_TAG"

if [ -n "$UPSTREAM_TAG" ] && [ "${UPSTREAM_CLEAN}.1" != "$CURRENT_TAG" ]; then
  log "Nova versao detectada: $UPSTREAM_TAG -> ${UPSTREAM_CLEAN}.1"

  # Merge upstream with theirs strategy
  git checkout main
  git fetch upstream
  git merge upstream/main -X theirs -m "chore: auto-merge upstream $(date)"

  # Update VERSION with .1 suffix
  echo "${UPSTREAM_CLEAN}.1" > VERSION
  git add VERSION
  git commit -m "chore: version bump to ${UPSTREAM_CLEAN}.1"

  # Push
  git push origin main

  # Tag
  git tag -f "v${UPSTREAM_CLEAN}.1"
  git push origin "v${UPSTREAM_CLEAN}.1"

  log "Push concluido: v${UPSTREAM_CLEAN}.1"
else
  log "Versao ja atualizada, pulando sync. ($CURRENT_TAG)"
fi

# 2. Build local (arm64)
log "Buildando imagem Docker (linux/arm64)..."
docker build --pull --platform linux/arm64 -t "$IMG:local" . 2>&1 | tail -5 >> "$LOG"

# Tag with version
VERSION_TAG=$(cat VERSION)
docker tag "$IMG:local" "$IMG:latest"
docker tag "$IMG:local" "$IMG:$VERSION_TAG"

log "Imagem buildada: $VERSION_TAG"

# 3. Restart container
log "Restartando container..."
cd "$CDIR/.."
docker compose stop new-api
docker compose rm -f new-api
docker compose create new-api
docker compose start new-api

# Wait for healthy
for i in $(seq 1 30); do
  HEALTH=$(docker inspect --format='{{.State.Health.Status}}' new-api 2>/dev/null || echo "starting")
  if [ "$HEALTH" = "healthy" ]; then
    log "Container healthy!"
    break
  fi
  log "Aguardando container... ($i/30) status=$HEALTH"
  sleep 2
done

# Verify
RESP=$(curl -s -o /dev/null -w "%{http_code}" "https://router.atius.com.br/v1/models" -H "Authorization: Bearer giovanniS23h3rm3s2026at1usr0ut3rk3yXYZ123456ABCD" 2>/dev/null || echo "000")
if [ "$RESP" = "200" ]; then
  log "SUCESSO! /v1/models retornou 200"
else
  log "ERRO! /v1/models retornou $RESP"
fi

log "=== FIM AUTO-SYNC-DEPLOY ==="
