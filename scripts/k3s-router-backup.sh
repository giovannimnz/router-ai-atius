#!/usr/bin/env bash
set -euo pipefail
umask 077

cd "$(dirname "$0")/.."

ts="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="backups/k3s-router-${ts}"
meta_dir="${backup_dir}/metadata"
db_dir="${backup_dir}/db"

mkdir -p "$meta_dir" "$db_dir"

echo "Backup dir: ${backup_dir}"

node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status > "${meta_dir}/graphify-status.json" || true
podman pod ps > "${meta_dir}/podman-pod-ps.txt" || true
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}' > "${meta_dir}/podman-ps.txt" || true
sudo -n k3s kubectl -n router-ai-atius get all,pvc,configmap -o yaml > "${meta_dir}/k3s-resources.yaml" || true

source_mode="${ROUTER_BACKUP_SOURCE:-auto}"
if [ "$source_mode" = auto ]; then
  service_ip="$(sudo -n k3s kubectl -n router-ai-atius get service router-ai-atius -o jsonpath='{.spec.clusterIP}' 2>/dev/null || true)"
  if [ -n "$service_ip" ] && sudo -n grep -qF "http://${service_ip}:3000" /etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf; then
    source_mode=k3s
  else
    source_mode=podman
  fi
fi

case "$source_mode" in
  k3s)
    ready_replicas="$(sudo -n k3s kubectl -n router-ai-atius get deployment router-ai-atius -o jsonpath='{.status.readyReplicas}' 2>/dev/null || true)"
    if [ "${ready_replicas:-0}" -lt 1 ]; then
      echo "Runtime k3s nao esta Ready; backup abortado" >&2
      exit 1
    fi
    sudo -n k3s kubectl -n router-ai-atius exec statefulset/router-ai-atius-postgres -- \
      pg_dump -U admin DBRouterAiAtius > "${db_dir}/DBRouterAiAtius.sql"
    printf 'mode=k3s\nnamespace=router-ai-atius\ndatabase=DBRouterAiAtius\n' \
      > "${meta_dir}/database-source.txt"
    ;;
  podman)
    bin/clianything status > "${meta_dir}/clianything-status.txt"
    bin/clianything providers --all > "${meta_dir}/providers.txt"

    if ! bin/clianything backup channels > "${meta_dir}/backup-channels.txt" 2>&1; then
      echo "WARN: clianything backup channels unsupported or failed" >> "${meta_dir}/backup-channels.txt"
    fi
    if ! bin/clianything backup tokens > "${meta_dir}/backup-tokens.txt" 2>&1; then
      echo "WARN: clianything backup tokens unsupported or failed" >> "${meta_dir}/backup-tokens.txt"
    fi

    router_container="${ROUTER_CONTAINER_NAME:-router-ai-atius}"
    sql_dsn="$(podman exec "$router_container" printenv SQL_DSN)"
    mapfile -t db_connection < <(
      SQL_DSN="$sql_dsn" python3 - <<'PY'
import base64
import os
from urllib.parse import unquote, urlparse

dsn = urlparse(os.environ["SQL_DSN"])
if dsn.scheme not in {"postgres", "postgresql"} or not dsn.hostname or not dsn.path:
    raise SystemExit("SQL_DSN PostgreSQL invalido")

print(dsn.hostname)
print(dsn.port or 5432)
print(unquote(dsn.username or ""))
print(unquote(dsn.path.lstrip("/")))
print(base64.b64encode(unquote(dsn.password or "").encode()).decode())
PY
    )
    unset sql_dsn

    if [ "${#db_connection[@]}" -ne 5 ]; then
      echo "Nao foi possivel resolver o SQL_DSN ativo" >&2
      exit 1
    fi

    db_host="${db_connection[0]}"
    db_port="${db_connection[1]}"
    db_user="${db_connection[2]}"
    db_name="${db_connection[3]}"
    db_password="$(printf '%s' "${db_connection[4]}" | base64 --decode)"

    # A credencial entra somente no ambiente efemero do pg_dump 17 do host;
    # nunca aparece em argumentos, logs ou artefatos.
    PGPASSWORD="$db_password" pg_dump --no-password \
      -h "$db_host" -p "$db_port" -U "$db_user" "$db_name" > "${db_dir}/DBRouterAiAtius.sql"
    unset db_password db_connection
    printf 'mode=podman\nhost=%s\nport=%s\ndatabase=%s\n' "$db_host" "$db_port" "$db_name" \
      > "${meta_dir}/database-source.txt"
    ;;
  *)
    echo "ROUTER_BACKUP_SOURCE deve ser auto, k3s ou podman" >&2
    exit 1
    ;;
esac

dump_path="${db_dir}/DBRouterAiAtius.sql"
if [ ! -s "$dump_path" ] || ! grep -q '^CREATE TABLE ' "$dump_path" || ! grep -q 'PostgreSQL database dump complete' "$dump_path"; then
  echo "Dump canonico invalido ou sem tabelas: ${dump_path}" >&2
  exit 1
fi

printf 'source_mode=%s\ndump_bytes=%s\n' "$source_mode" "$(stat -c %s "$dump_path")" \
  > "${meta_dir}/backup-result.txt"

echo "backup path: ${backup_dir}"
