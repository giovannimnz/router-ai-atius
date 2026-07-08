#!/usr/bin/env bash
set -euo pipefail

SOURCE_DB="${SOURCE_DB:-newapi}"
TARGET_DB="${TARGET_DB:-DBRouterAiAtius}"
PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-8745}"
PGDATABASE="${PGDATABASE:-postgres}"
PG_SUPERUSER="${PG_SUPERUSER:-postgres}"
PGBouncer_INI="${PGBOUNCER_INI:-/etc/pgbouncer/pgbouncer.ini}"
BACKUP_ROOT="${BACKUP_ROOT:-/home/ubuntu/.backups/router-ai-atius-phase24}"
TIMESTAMP="${TIMESTAMP:-$(date +%Y%m%d_%H%M%S)}"
DUMP_FILE="${DUMP_FILE:-${BACKUP_ROOT}/${TIMESTAMP}-${SOURCE_DB}.dump}"
TOC_FILE="${TOC_FILE:-${DUMP_FILE}.toc}"
CATALOG_SQL="${CATALOG_SQL:-scripts/phase24-catalog-transform.sql}"
EXECUTE=0
REPLACE_TARGET=0
APPLY_CATALOG=0
CONFIRM_SOURCE=""
CONFIRM_TARGET=""

usage() {
  cat <<'EOF'
Usage:
  scripts/phase24-build-canonical-db.sh [options]

Default mode is dry-run. No DB mutation happens unless --execute is provided
and both confirmation flags match the expected source/target.

Options:
  --execute                     Run mutating commands.
  --replace-target              Drop/recreate the target DB if it already exists.
  --apply-catalog-transform     Apply scripts/phase24-catalog-transform.sql after restore.
  --confirm-source NAME         Must match SOURCE_DB when --execute is used.
  --confirm-target NAME         Must match TARGET_DB when --execute is used.
  --source-db NAME              Source DB. Default: newapi
  --target-db NAME              Target DB. Default: DBRouterAiAtius
  --backup-root PATH            Backup directory root.
  --dump-file PATH              Custom pg_dump archive path.
  --pgbouncer-ini PATH          PgBouncer ini to verify mappings.
  --catalog-sql PATH            Catalog transform SQL file.
  -h, --help                    Show this help.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --execute)
      EXECUTE=1
      ;;
    --replace-target)
      REPLACE_TARGET=1
      ;;
    --apply-catalog-transform)
      APPLY_CATALOG=1
      ;;
    --confirm-source)
      CONFIRM_SOURCE="${2:-}"
      shift
      ;;
    --confirm-target)
      CONFIRM_TARGET="${2:-}"
      shift
      ;;
    --source-db)
      SOURCE_DB="${2:-}"
      shift
      ;;
    --target-db)
      TARGET_DB="${2:-}"
      shift
      ;;
    --backup-root)
      BACKUP_ROOT="${2:-}"
      shift
      ;;
    --dump-file)
      DUMP_FILE="${2:-}"
      shift
      ;;
    --pgbouncer-ini)
      PGBouncer_INI="${2:-}"
      shift
      ;;
    --catalog-sql)
      CATALOG_SQL="${2:-}"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

TOC_FILE="${TOC_FILE:-${DUMP_FILE}.toc}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

run_sql() {
  sudo -u "$PG_SUPERUSER" psql \
    -h "$PGHOST" \
    -p "$PGPORT" \
    -d "$PGDATABASE" \
    -v ON_ERROR_STOP=1 \
    -X \
    -q \
    -c "$1"
}

run_cmd() {
  if [[ "$EXECUTE" -eq 1 ]]; then
    echo "+ $*"
    "$@"
  else
    echo "DRY-RUN: $*"
  fi
}

ensure_execute_guard() {
  if [[ "$EXECUTE" -eq 0 ]]; then
    return
  fi
  if [[ "$CONFIRM_SOURCE" != "$SOURCE_DB" ]]; then
    echo "Refusing to mutate: --confirm-source must equal ${SOURCE_DB}" >&2
    exit 1
  fi
  if [[ "$CONFIRM_TARGET" != "$TARGET_DB" ]]; then
    echo "Refusing to mutate: --confirm-target must equal ${TARGET_DB}" >&2
    exit 1
  fi
}

preflight() {
  require_cmd sudo
  require_cmd psql
  require_cmd pg_dump
  require_cmd pg_restore
  require_cmd createdb
  require_cmd dropdb
  require_cmd sha256sum
  require_cmd rg

  mkdir -p "$BACKUP_ROOT"

  echo "Phase 24 candidate DB build"
  echo "  source_db: ${SOURCE_DB}"
  echo "  target_db: ${TARGET_DB}"
  echo "  backup_root: ${BACKUP_ROOT}"
  echo "  dump_file: ${DUMP_FILE}"
  echo "  catalog_sql: ${CATALOG_SQL}"
  echo "  execute: ${EXECUTE}"

  echo
  echo "Preflight checks"
  run_sql "select datname from pg_database where datname in ('${SOURCE_DB}', '${TARGET_DB}') order by datname;"

  if [[ -f "$PGBouncer_INI" ]]; then
    rg -n "^[[:space:]]*(${SOURCE_DB}|${TARGET_DB})[[:space:]]*=" "$PGBouncer_INI" || true
  else
    echo "WARN: PgBouncer ini not found at ${PGBouncer_INI}" >&2
  fi
}

dump_source() {
  run_cmd sudo -u "$PG_SUPERUSER" pg_dump \
    -h "$PGHOST" \
    -p "$PGPORT" \
    -d "$SOURCE_DB" \
    -Fc \
    -f "$DUMP_FILE"

  if [[ "$EXECUTE" -eq 1 ]]; then
    echo "+ pg_restore -l ${DUMP_FILE} > ${TOC_FILE}"
    pg_restore -l "$DUMP_FILE" >"$TOC_FILE"
    sha256sum "$DUMP_FILE" "$TOC_FILE"
  else
    echo "DRY-RUN: pg_restore -l ${DUMP_FILE} > ${TOC_FILE}"
    echo "DRY-RUN: sha256sum ${DUMP_FILE} ${TOC_FILE}"
  fi
}

restore_target() {
  local target_exists
  target_exists="$(sudo -u "$PG_SUPERUSER" psql -h "$PGHOST" -p "$PGPORT" -d "$PGDATABASE" -Atqc "select 1 from pg_database where datname='${TARGET_DB}'")"

  if [[ "$target_exists" == "1" && "$REPLACE_TARGET" -eq 1 ]]; then
    run_cmd sudo -u "$PG_SUPERUSER" dropdb -h "$PGHOST" -p "$PGPORT" "$TARGET_DB"
    target_exists=""
  fi

  if [[ -n "$target_exists" ]]; then
    echo "Target DB ${TARGET_DB} already exists."
    if [[ "$REPLACE_TARGET" -eq 0 ]]; then
      echo "Refusing to continue without --replace-target." >&2
      exit 1
    fi
  fi

  run_cmd sudo -u "$PG_SUPERUSER" createdb -h "$PGHOST" -p "$PGPORT" "$TARGET_DB"

  if [[ "$EXECUTE" -eq 1 ]]; then
    echo "+ pg_restore -l ${DUMP_FILE}"
    pg_restore -l "$DUMP_FILE"
  else
    echo "DRY-RUN: pg_restore -l ${DUMP_FILE}"
  fi

  run_cmd sudo -u "$PG_SUPERUSER" pg_restore \
    -h "$PGHOST" \
    -p "$PGPORT" \
    -d "$TARGET_DB" \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    "$DUMP_FILE"

  if [[ "$APPLY_CATALOG" -eq 1 ]]; then
    run_cmd sudo -u "$PG_SUPERUSER" psql \
      -h "$PGHOST" \
      -p "$PGPORT" \
      -d "$TARGET_DB" \
      -v ON_ERROR_STOP=1 \
      -f "$CATALOG_SQL"
  fi
}

verify_target() {
  cat <<EOF
Verification queries to review after restore:
  select current_database();
  select datname from pg_database where datname in ('${SOURCE_DB}', '${TARGET_DB}') order by datname;
  select count(*) as channels from channels;
  select count(*) as models from models;
  select count(*) as abilities from abilities;
  select id, name, status, type, models from channels where id in (1,2,5,9) order by id;
EOF

  if [[ "$EXECUTE" -eq 1 ]]; then
    run_sql "select datname from pg_database where datname in ('${SOURCE_DB}', '${TARGET_DB}') order by datname;"
    run_sql "select 'channels' as table_name, count(*) from channels union all select 'models', count(*) from models union all select 'abilities', count(*) from abilities;"
    run_sql "select id, name, status, type, models from channels where id in (1,2,5,9) order by id;"
  else
    echo "DRY-RUN: psql verification queries against ${TARGET_DB}"
  fi
}

main() {
  ensure_execute_guard
  preflight
  dump_source
  restore_target
  verify_target

  echo
  echo "Review notes:"
  echo "  - ${SOURCE_DB} remains the rollback source until all Phase 24 gates pass."
  echo "  - PgBouncer must expose mappings for both ${SOURCE_DB} and ${TARGET_DB} during cutover rehearsal."
  echo "  - Apply ${CATALOG_SQL} only with a secure codex channel credential variable."
}

main "$@"
