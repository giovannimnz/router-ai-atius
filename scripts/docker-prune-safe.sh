#!/usr/bin/env bash
# docker-prune-safe.sh
#
# Libera espaço em disco de coisas que com certeza não são úteis:
#   - Dangling images (sem tag, <none>:<none>)               ~13.6GB típico
#   - Dangling build cache (camadas órfãs de docker build)   ~19.8GB típico
#   - Stopped containers (exited, dead)                      ~MB-GB
#
# NUNCA toca em:
#   - Dangling volumes (podem ter dados ativos —
#     hermes-data, plane-*, portainer_data, pm2web_*)
#   - Imagens nomeadas (mesmo que pareçam "não usadas" —
#     podem ser pull target de outros scripts/deploys)
#   - Containers rodando (Active)
#
# Idempotente: rodar várias vezes é seguro. Sem prune = noop silencioso.
#
# Uso:
#   ./scripts/docker-prune-safe.sh                # prune + report
#   ./scripts/docker-prune-safe.sh --dry-run      # mostra o que faria
#   ./scripts/docker-prune-safe.sh --json         # output em JSON
#   ./scripts/docker-prune-safe.sh --quiet        # só o resumo final
#   ./scripts/docker-prune-safe.sh --aggressive   # também remove imagens
#                                                  # nomeadas não usadas
#                                                  # (USE COM CUIDADO)
#
# Log:
#   - Human-readable: STDOUT
#   - Machine-readable (--json): JSON único
#   - Persistent: ~/logs/docker-prune-safe-YYYYMMDD-HHMMSS.log
#
# Exit codes:
#   0  — sucesso (incluindo se nada pra prune)
#   1  — erro genérico
#   2  — docker não está disponível
#
# Cron (semanal, domingo 04:00 — baixo tráfego):
#   0 4 * * 0 /home/ubuntu/docker/Atius/router-ai-atius/scripts/docker-prune-safe.sh --quiet >> ~/logs/docker-prune-cron.log 2>&1
#
# Refs:
#   61-Incidents/2026-06-04-srv1-podman-preflight.md (origem)
#   61-Incidents/2026-06-04-docker-prune-safe-automation.md (autodoc)

set -euo pipefail

# --- arg parsing --------------------------------------------------------
DRY_RUN=""
JSON_OUT=""
QUIET=""
AGGRESSIVE=""
for arg in "$@"; do
  case "$arg" in
    --dry-run)     DRY_RUN="1" ;;
    --json)        JSON_OUT="1" ;;
    --quiet)       QUIET="1" ;;
    --aggressive)  AGGRESSIVE="1" ;;
    -h|--help)
      sed -n '2,32p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *) echo "ERROR: unknown arg: $arg" >&2; exit 1 ;;
  esac
done

# --- helpers ------------------------------------------------------------
log()  { [ -z "$QUIET" ] && [ -z "$JSON_OUT" ] && echo "[$(date -u +%H:%M:%SZ)] $*"; }
err()  { echo "[$(date -u +%H:%M:%SZ)] ERROR: $*" >&2; }

# Disk usage before (bytes)
disk_used() { df -B1 / 2>/dev/null | awk 'NR==2 {print $3}'; }
disk_avail() { df -B1 / 2>/dev/null | awk 'NR==2 {print $4}'; }
disk_pct() { df / 2>/dev/null | awk 'NR==2 {gsub("%","",$5); print $5}'; }

# Docker total reclaimable bytes
docker_reclaimable_bytes() {
  docker system df --format "{{.Reclaimable}}" 2>/dev/null \
    | awk '{
        v=$1; u=$2;
        if (u=="GB") mult=1024*1024*1024;
        else if (u=="MB") mult=1024*1024;
        else if (u=="kB") mult=1024;
        else if (u=="B") mult=1;
        else mult=0;
        # strip non-digits from v (e.g. "(86%)")
        gsub(/[^0-9.]/, "", v);
        if (v == "") v=0;
        total += v * mult;
      }
      END { printf "%d\n", total }'
}

# --- preflight ----------------------------------------------------------
command -v docker >/dev/null 2>&1 || { err "docker not installed"; exit 2; }
docker info >/dev/null 2>&1 || { err "docker daemon not running"; exit 2; }

# --- snapshot before ----------------------------------------------------
BEFORE_USED=$(disk_used)
BEFORE_AVAIL=$(disk_avail)
BEFORE_PCT=$(disk_pct)
BEFORE_DOCKER_RECLAIM=$(docker_reclaimable_bytes)
BEFORE_DOCKER_IMAGES=$(docker images -q | wc -l)
BEFORE_DOCKER_CONTAINERS=$(docker ps -a -q | wc -l)
BEFORE_DOCKER_BUILDER_CACHE=$(docker builder du 2>/dev/null | tail -1 | awk '{print $1}')

log "BEFORE: disk ${BEFORE_PCT}% used, $(numfmt --to=iec --suffix=B "${BEFORE_AVAIL}" 2>/dev/null || echo "${BEFORE_AVAIL}B") available"
log "BEFORE: docker reclaimable=$(numfmt --to=iec --suffix=B "${BEFORE_DOCKER_RECLAIM}" 2>/dev/null || echo "${BEFORE_DOCKER_RECLAIM}B"), images=${BEFORE_DOCKER_IMAGES}, containers=${BEFORE_DOCKER_CONTAINERS}"

# --- snapshots per category --------------------------------------------
# Dangling images (no tag) — list IDs only, no tag
DANGLING_IMAGES=$(docker images -f "dangling=true" -q | wc -l)
# Stopped containers
STOPPED_CONTAINERS=$(docker ps -a -f "status=exited" -q | wc -l | tr -d ' ')
# Build cache size
BUILD_CACHE_SIZE_RAW=$(docker builder du 2>/dev/null | tail -1)
log "TARGETS: dangling_images=${DANGLING_IMAGES}, stopped_containers=${STOPPED_CONTAINERS}, build_cache=${BUILD_CACHE_SIZE_RAW:-unknown}"

# --- dry-run exit -------------------------------------------------------
if [ -n "$DRY_RUN" ]; then
  log "DRY RUN — not removing anything"
  log "Would run:"
  log "  docker image prune -f       (remove ${DANGLING_IMAGES} dangling images)"
  log "  docker container prune -f   (remove ${STOPPED_CONTAINERS} stopped containers)"
  log "  docker builder prune -f     (remove all build cache)"
  if [ -n "$AGGRESSIVE" ]; then
    log "  --aggressive: also docker image prune -a (removes named-but-unused images)"
  fi
  exit 0
fi

# --- the prunes ---------------------------------------------------------
log "step 1/3: docker image prune (dangling only)"
if [ -n "$AGGRESSIVE" ]; then
  log "  --aggressive: também remove imagens nomeadas não usadas"
  docker image prune -a -f 2>&1 | tail -5 || err "image prune -a failed"
else
  docker image prune -f 2>&1 | tail -3 || err "image prune failed"
fi

log "step 2/3: docker container prune (stopped only)"
docker container prune -f 2>&1 | tail -3 || err "container prune failed"

log "step 3/3: docker builder prune (all build cache)"
docker builder prune -f 2>&1 | tail -3 || err "builder prune failed"

# --- snapshot after -----------------------------------------------------
AFTER_USED=$(disk_used)
AFTER_AVAIL=$(disk_avail)
AFTER_PCT=$(disk_pct)
AFTER_DOCKER_RECLAIM=$(docker_reclaimable_bytes)
AFTER_DOCKER_IMAGES=$(docker images -q | wc -l)
AFTER_DOCKER_CONTAINERS=$(docker ps -a -q | wc -l)

BYTES_FREED=$((BEFORE_USED - AFTER_USED))
[ "$BYTES_FREED" -lt 0 ] && BYTES_FREED=0
IMAGES_REMOVED=$((BEFORE_DOCKER_IMAGES - AFTER_DOCKER_IMAGES))
[ "$IMAGES_REMOVED" -lt 0 ] && IMAGES_REMOVED=0
CONTAINERS_REMOVED=$((BEFORE_DOCKER_CONTAINERS - AFTER_DOCKER_CONTAINERS))
[ "$CONTAINERS_REMOVED" -lt 0 ] && CONTAINERS_REMOVED=0

# --- output -------------------------------------------------------------
TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
HOSTNAME_S=$(hostname -s 2>/dev/null || echo "unknown")

if [ -n "$JSON_OUT" ]; then
  cat <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "host": "${HOSTNAME_S}",
  "before": {
    "disk_used_bytes": ${BEFORE_USED},
    "disk_avail_bytes": ${BEFORE_AVAIL},
    "disk_pct": ${BEFORE_PCT},
    "docker_reclaimable_bytes": ${BEFORE_DOCKER_RECLAIM},
    "docker_images": ${BEFORE_DOCKER_IMAGES},
    "docker_containers": ${BEFORE_DOCKER_CONTAINERS}
  },
  "after": {
    "disk_used_bytes": ${AFTER_USED},
    "disk_avail_bytes": ${AFTER_AVAIL},
    "disk_pct": ${AFTER_PCT},
    "docker_reclaimable_bytes": ${AFTER_DOCKER_RECLAIM},
    "docker_images": ${AFTER_DOCKER_IMAGES},
    "docker_containers": ${AFTER_DOCKER_CONTAINERS}
  },
  "delta": {
    "bytes_freed": ${BYTES_FREED},
    "images_removed": ${IMAGES_REMOVED},
    "containers_removed": ${CONTAINERS_REMOVED},
    "dangling_images_targeted": ${DANGLING_IMAGES},
    "stopped_containers_targeted": ${STOPPED_CONTAINERS}
  }
}
EOF
else
  log "AFTER:  disk ${AFTER_PCT}% used, $(numfmt --to=iec --suffix=B "${AFTER_AVAIL}" 2>/dev/null || echo "${AFTER_AVAIL}B") available"
  log "AFTER:  docker reclaimable=$(numfmt --to=iec --suffix=B "${AFTER_DOCKER_RECLAIM}" 2>/dev/null || echo "${AFTER_DOCKER_RECLAIM}B"), images=${AFTER_DOCKER_IMAGES}, containers=${AFTER_DOCKER_CONTAINERS}"
  log ""
  log "=== SUMMARY ==="
  log "  bytes_freed:           $(numfmt --to=iec --suffix=B "${BYTES_FREED}" 2>/dev/null || echo "${BYTES_FREED}B") (${BYTES_FREED} B)"
  log "  images_removed:        ${IMAGES_REMOVED}"
  log "  containers_removed:    ${CONTAINERS_REMOVED}"
  log "  disk_pct:              ${BEFORE_PCT}% → ${AFTER_PCT}%"
  log "  docker_reclaimable:    $(numfmt --to=iec --suffix=B "${BEFORE_DOCKER_RECLAIM}" 2>/dev/null || echo "n/a") → $(numfmt --to=iec --suffix=B "${AFTER_DOCKER_RECLAIM}" 2>/dev/null || echo "n/a")"
fi

# --- persistent log -----------------------------------------------------
LOG_DIR="${HOME}/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="${LOG_DIR}/docker-prune-safe-$(date -u +%Y%m%d-%H%M%S).log"

{
  echo "# docker-prune-safe.sh run"
  echo "timestamp: ${TIMESTAMP}"
  echo "host:      ${HOSTNAME_S}"
  echo "trigger:   ${CRON_TAG:-manual}"
  echo ""
  echo "## Before"
  echo "disk_pct:           ${BEFORE_PCT}%"
  echo "disk_avail:         ${BEFORE_AVAIL} bytes ($(numfmt --to=iec --suffix=B "${BEFORE_AVAIL}" 2>/dev/null || echo "n/a"))"
  echo "docker_reclaimable: ${BEFORE_DOCKER_RECLAIM} bytes ($(numfmt --to=iec --suffix=B "${BEFORE_DOCKER_RECLAIM}" 2>/dev/null || echo "n/a"))"
  echo "images:             ${BEFORE_DOCKER_IMAGES}"
  echo "containers:         ${BEFORE_DOCKER_CONTAINERS}"
  echo "dangling_images:    ${DANGLING_IMAGES}"
  echo "stopped_containers: ${STOPPED_CONTAINERS}"
  echo ""
  echo "## After"
  echo "disk_pct:           ${AFTER_PCT}%"
  echo "disk_avail:         ${AFTER_AVAIL} bytes ($(numfmt --to=iec --suffix=B "${AFTER_AVAIL}" 2>/dev/null || echo "n/a"))"
  echo "docker_reclaimable: ${AFTER_DOCKER_RECLAIM} bytes ($(numfmt --to=iec --suffix=B "${AFTER_DOCKER_RECLAIM}" 2>/dev/null || echo "n/a"))"
  echo "images:             ${AFTER_DOCKER_IMAGES}"
  echo "containers:         ${AFTER_DOCKER_CONTAINERS}"
  echo ""
  echo "## Delta"
  echo "bytes_freed:        ${BYTES_FREED} ($(numfmt --to=iec --suffix=B "${BYTES_FREED}" 2>/dev/null || echo "n/a"))"
  echo "images_removed:     ${IMAGES_REMOVED}"
  echo "containers_removed: ${CONTAINERS_REMOVED}"
} > "$LOG_FILE"

[ -z "$QUIET" ] && [ -z "$JSON_OUT" ] && log "log: $LOG_FILE"
