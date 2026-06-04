#!/usr/bin/env bash
# podman-migrate-from-docker.sh
#
# One-shot: take a running Docker stack of router-ai-atius and lift it to
# Podman with no data loss. Tested for the SRV-1 (atius-router) deployment.
#
# Rebrand v2.11: container names, db name, and ports all updated.
#
# What it does:
#   1. Dumps the running router-ai-atius PostgreSQL DB to a SQL file.
#   2. Stops + removes the Docker stack (data/ and logs/ are NOT removed).
#   3. Builds the model-detailed image (Podman uses Buildah; we use
#      `podman build` to keep one toolchain).
#   4. Starts the Podman stack, pointing it at the same ./data and ./logs.
#   5. Restores the DB dump into the new Podman-managed Postgres.
#   6. Smoke-tests GET /api/status on the new v2.11 port (3030).
#
# Pre-conditions:
#   - Docker stack was up and serving (DB is alive).
#   - podman + podman-compose installed (4.x / 1.0+).
#   - .env populated with POSTGRES_PASSWORD and REDIS_PASSWORD.
#
# NOT a live cutover. The downtime window is roughly:
#   down:   5-30s
#   up:     30-120s (Podman pulls may be slower than dockerd's cache)
#   restore: 1-5 min for 100MB SQL dumps
#
# Use during a maintenance window.

set -euo pipefail
cd "$(dirname "$0")/.."

DUMP_DIR="/tmp/podman-migrate-$$"
DUMP_FILE="$DUMP_DIR/dump.sql"
mkdir -p "$DUMP_DIR"

# v2.11: source the DB name from .env (defaults to DBRouterAiAtius)
POSTGRES_DB="${POSTGRES_DB:-DBRouterAiAtius}"
POSTGRES_USER="${POSTGRES_USER:-admin}"

echo "[migrate] step 1/6: dump PostgreSQL data from Docker stack (db=$POSTGRES_DB)"
docker exec router-ai-atius-db pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-acl > "$DUMP_FILE" 2>/dev/null \
  || docker exec db-newapi pg_dump -U admin -d newapi --no-owner --no-acl > "$DUMP_FILE" 2>/dev/null \
  || { echo "ERROR: could not dump DB from any known container" >&2; exit 1; }
DUMP_SIZE=$(du -h "$DUMP_FILE" | cut -f1)
echo "[migrate]   dumped $DUMP_SIZE to $DUMP_FILE"

echo "[migrate] step 2/6: stop + remove Docker stack (keep volumes/data)"
docker compose down 2>&1 || true
# v2.11 names; fall back to old v1 names
docker rm -f router-ai-atius router-ai-atius-model-detailed router-ai-atius-db router-ai-atius-redis 2>/dev/null || true
docker rm -f new-api model-detailed db-newapi redis-newapi 2>/dev/null || true
docker network rm router-ai_newapi-internal 2>/dev/null || true
docker network rm atius-ai-router_internal 2>/dev/null || true

echo "[migrate] step 3/6: build the model-detailed image with Podman"
podman-compose -f podman-compose.yml build router-ai-atius-model-detailed

echo "[migrate] step 4/6: start the Podman stack"
podman-compose -f podman-compose.yml up -d

echo "[migrate] step 5/6: wait for postgres to be ready, then restore"
for i in $(seq 1 30); do
  if podman exec router-ai-atius-db pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" 2>/dev/null; then
    break
  fi
  sleep 2
done
cat "$DUMP_FILE" | podman exec -i router-ai-atius-db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" 2>&1 | tail -3
echo "[migrate]   restore complete"

echo "[migrate] step 6/6: smoke test (port 3030)"
healthy=0
for i in $(seq 1 30); do
  if curl -fs http://localhost:3030/api/status >/dev/null 2>&1; then
    echo "[migrate]   GET /api/status → OK"
    healthy=1
    break
  fi
  sleep 2
done
if [ "$healthy" -ne 1 ]; then
  echo "[migrate]   WARNING: /api/status did not return 200 within 60s" >&2
fi

rm -rf "$DUMP_DIR"
echo "[migrate] done. Podman stack is up at :3030 (api) and :3300 (middleware)."
echo "[migrate] next: ./scripts/podman-up.sh --logs  (or podman ps / podman logs router-ai-atius)"
