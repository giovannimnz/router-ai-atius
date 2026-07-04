#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GSD_TOOLS="${HOME}/.codex/gsd-core/bin/gsd-tools.cjs"
CLIANYTHING_BIN="${REPO_ROOT}/bin/clianything"
ROUTER_UNIT="${ROUTER_UNIT:-/home/ubuntu/.config/systemd/user/container-router-ai-atius.service}"
INCIDENT_DUMP="${INCIDENT_DUMP:-/home/ubuntu/.backups/router-ai-atius-incident-20260703T231027-0300/newapi-before.fix.dump}"
LEGACY_DUMP="${LEGACY_DUMP:-${REPO_ROOT}/data/pg_backup/newapi_backup_20260531_235230.dump}"
CATALOG_CHANNELS_SQL="${CATALOG_CHANNELS_SQL:-${REPO_ROOT}/backups/clianything/20260701_184735_channels.sql}"
CATALOG_MODELS_SQL="${CATALOG_MODELS_SQL:-${REPO_ROOT}/backups/clianything/20260701_184735_models.sql}"
CATALOG_ABILITIES_SQL="${CATALOG_ABILITIES_SQL:-${REPO_ROOT}/backups/clianything/20260701_184735_abilities.sql}"

section() {
  printf '\n### %s ###\n' "$1"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing command: %s\n' "$1" >&2
    exit 1
  }
}

require_file() {
  local path="$1"
  [ -f "$path" ] || {
    printf 'missing file: %s\n' "$path" >&2
    exit 1
  }
}

print_file_meta() {
  local path="$1"
  if [ -f "$path" ]; then
    ls -lh "$path"
  else
    printf 'missing file: %s\n' "$path"
  fi
}

require_cmd node
require_cmd rg
require_cmd psql
require_cmd pg_restore
require_cmd sudo
require_file "$GSD_TOOLS"
require_file "$CLIANYTHING_BIN"
require_file "$ROUTER_UNIT"

section "graphify status"
node "$GSD_TOOLS" graphify status

section "router status strict"
"$CLIANYTHING_BIN" status --strict

section "catalog counts"
"$CLIANYTHING_BIN" query --format table "select 'channels' as table_name, count(*)::bigint as total from channels
union all
select 'models' as table_name, count(*)::bigint as total from models
union all
select 'abilities' as table_name, count(*)::bigint as total from abilities
union all
select 'tokens' as table_name, count(*)::bigint as total from tokens
order by table_name;"

section "host database list"
sudo -u postgres psql -XtAc "select datname from pg_database where datistemplate = false order by datname;"

section "router unit db target"
rg -n "SQL_DSN=|EMBEDDING_GOVERNOR_" "$ROUTER_UNIT"

section "catalog snapshot files"
print_file_meta "$CATALOG_CHANNELS_SQL"
print_file_meta "$CATALOG_MODELS_SQL"
print_file_meta "$CATALOG_ABILITIES_SQL"

section "archive inventory via pg_restore -l"
for archive in "$INCIDENT_DUMP" "$LEGACY_DUMP"; do
  printf '\n-- %s --\n' "$archive"
  if [ -f "$archive" ]; then
    pg_restore -l "$archive" | sed -n '1,40p'
  else
    printf 'missing archive: %s\n' "$archive"
  fi
done
