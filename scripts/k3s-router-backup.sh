#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

ts="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="backups/k3s-router-${ts}"
meta_dir="${backup_dir}/metadata"
db_dir="${backup_dir}/db"

mkdir -p "$meta_dir" "$db_dir"

echo "Backup dir: ${backup_dir}"

node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status > "${meta_dir}/graphify-status.json" || true
bin/clianything status > "${meta_dir}/clianything-status.txt"
bin/clianything providers --all > "${meta_dir}/providers.txt"
podman pod ps > "${meta_dir}/podman-pod-ps.txt"
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}' > "${meta_dir}/podman-ps.txt"

if bin/clianything backup channels > "${meta_dir}/backup-channels.txt" 2>&1; then
  :
else
  echo "WARN: clianything backup channels unsupported or failed" >> "${meta_dir}/backup-channels.txt"
fi

if bin/clianything backup tokens > "${meta_dir}/backup-tokens.txt" 2>&1; then
  :
else
  echo "WARN: clianything backup tokens unsupported or failed" >> "${meta_dir}/backup-tokens.txt"
fi

podman exec postgres pg_dump -U admin DBRouterAiAtius > "${db_dir}/DBRouterAiAtius.sql"

echo "backup path: ${backup_dir}"
