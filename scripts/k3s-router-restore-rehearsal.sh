#!/usr/bin/env bash
# shellcheck disable=SC2024
set -euo pipefail
trap - INT TERM

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

mode=dry-run
backup_dir="${K3S_BACKUP_DIR:-}"
evidence_dir=""
cleanup_evidence=""
bootstrap_evidence=""
namespace=router-ai-atius
database=DBRouterAiAtius
database_user="admin"
original_args=("$@")
restore_started=false
restore_evidence=""
retry_no_go=false
replace_go=false
replace_go_target=false
retry_prior_evidence=""
retry_prior_sha256=""
retry_source_evidence=""
lineage_prior_status=""
tmp=""
active_pid=""
restore_lock_fd=""
target_state_file=""
cluster_uid=""

die() {
  if $restore_started; then mark_no_go; fi
  echo "restore rehearsal failed: $*" >&2
  exit 1
}

mark_no_go() {
  local tmp_file canonical_status canonical_path canonical_sha actual_sha
  trap - ERR
  if [ -n "$target_state_file" ] && [ -f "$target_state_file" ] && [ ! -L "$target_state_file" ] &&
     [ -n "$restore_evidence" ] && [ -f "$restore_evidence" ] && [ ! -L "$restore_evidence" ]; then
    canonical_status="$(jq -r '.status // empty' "$target_state_file" 2>/dev/null)"
    canonical_path="$(jq -r '.evidence_path // empty' "$target_state_file" 2>/dev/null)"
    canonical_sha="$(jq -r '.evidence_sha256 // empty' "$target_state_file" 2>/dev/null)"
    actual_sha="$(sha256sum "$restore_evidence" 2>/dev/null | awk '{print $1}')"
    if [ "$canonical_status" = go ] && [ "$canonical_path" = "$restore_evidence" ] &&
       [ "$canonical_sha" = "$actual_sha" ] && [ "$(jq -r '.status // empty' "$restore_evidence" 2>/dev/null)" = go ]; then
      return 0
    fi
  fi
  set +e
  if [ -n "$restore_evidence" ] && [ -f "$restore_evidence" ] && [ ! -L "$restore_evidence" ]; then
    tmp_file="${restore_evidence}.tmp"
    jq --arg completed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
      '.status = "no-go" | .restore_passed = false | .failure = "restore_failed" | .completed_at = $completed_at' \
      "$restore_evidence" > "$tmp_file" && chmod 600 "$tmp_file" && mv "$tmp_file" "$restore_evidence"
    write_target_state no-go || true
  fi
}

on_restore_error() {
  local rc=$?
  mark_no_go
  exit "$rc"
}

cleanup_tmp() {
  if [ -n "$tmp" ] && [ -d "$tmp" ]; then rm -rf "$tmp"; fi
}

on_exit() {
  local rc=$?
  cleanup_tmp
  if $restore_started && [ "$rc" -ne 0 ]; then mark_no_go; fi
}

on_signal() {
  local signal="$1"
  if [ -n "$active_pid" ]; then
    terminate_active_group "$active_pid" || true
  fi
  mark_no_go
  cleanup_tmp
  trap - "$signal"
  kill -s "$signal" "$$"
}

terminate_active_group() {
  local pgid="$1"
  kill -TERM "-$pgid" 2>/dev/null || true
  sleep 0.1 || true
  kill -KILL "-$pgid" 2>/dev/null || true
  for _ in $(seq 1 20); do
    if ! kill -0 "-$pgid" 2>/dev/null; then return 0; fi
    sleep 0.05
  done
  return 1
}

run_interruptible() {
  local rc
  set +e
  setsid --wait "$@" &
  active_pid=$!
  if [ -n "${PHASE29_ACTIVE_PID_FILE:-}" ]; then printf '%s\n' "$active_pid" > "$PHASE29_ACTIVE_PID_FILE"; fi
  wait "$active_pid"
  rc=$?
  active_pid=""
  set -e
  return "$rc"
}

run_interruptible_stdin() {
  local input="$1" rc
  shift
  if [ ! -f "$input" ] || [ -L "$input" ]; then die 'interruptible stdin must be a regular non-symlink file'; fi
  set +e
  setsid --wait "$@" < "$input" &
  active_pid=$!
  if [ -n "${PHASE29_ACTIVE_PID_FILE:-}" ]; then printf '%s\n' "$active_pid" > "$PHASE29_ACTIVE_PID_FILE"; fi
  wait "$active_pid"
  rc=$?
  active_pid=""
  set -e
  return "$rc"
}

acquire_restore_lock() {
  local lock_root="$HOME/.local/state/router-ai-atius/phase29" lock
  if [ -L "$HOME/.local" ] || [ -L "$HOME/.local/state" ] || [ -L "$HOME/.local/state/router-ai-atius" ] || [ -L "$lock_root" ]; then
    die 'restore lock path contains a symlink'
  fi
  install -d -m 0700 "$HOME/.local/state/router-ai-atius" "$lock_root"
  [ "$(stat -c %U:%a "$lock_root")" = "$(id -un):700" ] || die 'restore lock root owner/mode invalid'
  lock="$lock_root/restore-target.lock"
  target_state_file="$lock_root/restore-target-state.json"
  [ ! -L "$lock" ] || die 'restore lock must not be a symlink'
  [ ! -L "$target_state_file" ] || die 'restore target state must not be a symlink'
  umask 077
  exec {restore_lock_fd}> "$lock"
  chmod 600 "$lock"
  flock -n "$restore_lock_fd" || die 'another restore attempt holds the evidence lock'
}

write_target_state() {
  local status="$1" state_evidence="${2:-$restore_evidence}" state_tmp evidence_sha256
  [ -n "$target_state_file" ] || return 1
  [ -n "$cluster_uid" ] || return 1
  [ -f "$state_evidence" ] && [ ! -L "$state_evidence" ] || return 1
  if [ "${PHASE29_TEST_STATE_WRITE_FAIL:-}" = sha256 ]; then return 1; fi
  if ! evidence_sha256="$(sha256sum "$state_evidence" | awk '{print $1}')" ||
     ! [[ "$evidence_sha256" =~ ^[0-9a-f]{64}$ ]]; then
    return 1
  fi
  if ! state_tmp="$(mktemp "$(dirname "$target_state_file")/.restore-target-state.XXXXXX")"; then return 1; fi
  if ! chmod 600 "$state_tmp"; then rm -f "$state_tmp"; return 1; fi
  if [ "${PHASE29_TEST_STATE_WRITE_FAIL:-}" = jq ]; then rm -f "$state_tmp"; return 1; fi
  if ! jq -n --arg status "$status" --arg cluster_uid "$cluster_uid" \
    --arg evidence_path "$state_evidence" --arg evidence_sha256 "$evidence_sha256" \
    --arg updated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{schema_version:1,target:"router-ai-atius/DBRouterAiAtius@atius-srv-1",status:$status,cluster_uid:$cluster_uid,evidence_path:$evidence_path,evidence_sha256:$evidence_sha256,updated_at:$updated_at}' \
    > "$state_tmp"; then
    rm -f "$state_tmp"
    return 1
  fi
  if ! jq -e --arg status "$status" --arg cluster_uid "$cluster_uid" --arg evidence_path "$state_evidence" \
    --arg evidence_sha256 "$evidence_sha256" \
    '.schema_version == 1 and .target == "router-ai-atius/DBRouterAiAtius@atius-srv-1" and .status == $status and .cluster_uid == $cluster_uid and .evidence_path == $evidence_path and .evidence_sha256 == $evidence_sha256' \
    "$state_tmp" >/dev/null; then
    rm -f "$state_tmp"
    return 1
  fi
  if [ "${PHASE29_TEST_STATE_WRITE_FAIL:-}" = rename ]; then rm -f "$state_tmp"; return 1; fi
  if ! mv -f "$state_tmp" "$target_state_file"; then rm -f "$state_tmp"; return 1; fi
  return 0
}

publish_restore_success() {
  local success_file="$1"
  trap '' INT TERM
  if ! mv "$success_file" "$restore_evidence"; then
    trap 'on_signal INT' INT
    trap 'on_signal TERM' TERM
    die 'failed to publish successful restore evidence'
  fi
  if ! write_target_state go; then
    trap 'on_signal INT' INT
    trap 'on_signal TERM' TERM
    die 'failed to persist canonical successful target state'
  fi
  if [ "${PHASE29_TEST_TERMINAL_SIGNAL:-0}" = 1 ]; then kill -TERM "$$"; fi
  restore_started=false
  trap - INT TERM
}

cpu_max_value() {
  local cgroup file
  cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"
  file="/sys/fs/cgroup${cgroup}/cpu.max"
  [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"
  cat "$file"
}

quota_ok() {
  local quota period
  read -r quota period <<< "$1"
  [[ "$quota" =~ ^[0-9]+$ ]] && [[ "$period" =~ ^[0-9]+$ ]] && [ "$period" -gt 0 ] &&
    [ $((quota * 10)) -le $((period * 8)) ]
}

require_profile() {
  local cpu_max
  cpu_max="$(cpu_max_value)"
  if quota_ok "$cpu_max"; then return; fi
  [ "${PHASE29_PROFILED:-0}" != 1 ] || die "cpu.max exceeds 800m: $cpu_max"
  exec "$repo_root/scripts/podman-admin.sh" profile-run -- \
    env PHASE29_PROFILED=1 "$repo_root/scripts/k3s-router-restore-rehearsal.sh" "${original_args[@]}"
}

validate_dump_structure() {
  local dump="$1"
  if [ ! -f "$dump" ] || [ -L "$dump" ]; then die 'dump missing or symlinked'; fi
  [ "$(stat -c %s "$dump")" -gt 643 ] || die 'dump is empty or obsolete'
  grep -Fq 'PostgreSQL database dump' "$dump" || die 'dump header missing'
  for table in channels users tokens; do
    grep -Eq "^CREATE TABLE public\\.${table} " "$dump" || die "$table table definition missing"
  done
  grep -Fq 'PostgreSQL database dump complete' "$dump" || die 'dump completion marker missing'
}

normalize_schema_ddl() {
  local input="$1" output="$2"
  if [ ! -f "$input" ] || [ -L "$input" ]; then die 'schema inventory input is invalid'; fi
  sed -E '/^--/d; /^[[:space:]]*$/d; /^\\(un)?restrict /d' "$input" > "$output"
  [ -s "$output" ] || die 'normalized full schema DDL is empty'
}

normalize_database_inventory() {
  local schema_dump="$1" database_state="$2" role_settings="$3" large_objects="$4" output="$5" schema_output="$6"
  local schema_sha schema_lines schema_bytes large_object_sha large_object_count database_json settings_json
  if [ ! -f "$database_state" ] || [ -L "$database_state" ]; then die 'database-state inventory input is invalid'; fi
  if [ ! -f "$role_settings" ] || [ -L "$role_settings" ]; then die 'database role-setting inventory input is invalid'; fi
  if [ ! -f "$large_objects" ] || [ -L "$large_objects" ]; then die 'large-object inventory input is invalid'; fi
  jq -e 'type == "object" and .name == "DBRouterAiAtius" and (.owner | type == "string" and length > 0) and
    (.properties | type == "object") and (.acl | type == "array") and
    (.security_labels | type == "array") and has("comment") and (.tablespace | type == "string")' \
    "$database_state" >/dev/null || die 'database-state inventory is incomplete'
  awk -F '\t' 'NF != 3 || $1 == "" || $2 == "" || $3 !~ /^[0-9A-Fa-f]*$/ {exit 1}' "$role_settings" ||
    die 'pg_db_role_setting inventory is malformed'
  awk '/^[[:space:]]*$/ {next} !/^[0-9]+$/ {exit 1}' "$large_objects" || die 'large-object inventory contains a non-OID value'
  normalize_schema_ddl "$schema_dump" "$schema_output"
  schema_sha="$(sha256sum "$schema_output" | awk '{print $1}')"
  schema_lines="$(wc -l < "$schema_output")"
  schema_bytes="$(stat -c %s "$schema_output")"
  large_object_sha="$(sed '/^[[:space:]]*$/d' "$large_objects" | LC_ALL=C sort -n -u | sha256sum | awk '{print $1}')"
  large_object_count="$(sed '/^[[:space:]]*$/d' "$large_objects" | LC_ALL=C sort -n -u | wc -l)"
  database_json="$(jq -cS . "$database_state")"
  settings_json="$(while IFS=$'\t' read -r role name value_hex; do
    [ -n "$role" ] || continue
    jq -cn --arg role "$role" --arg name "$name" \
      --arg value_sha256 "$(printf '%s' "$value_hex" | sha256sum | awk '{print $1}')" \
      '{role:$role,name:$name,value_sha256:$value_sha256}'
  done < "$role_settings" | jq -csS 'sort_by(.role,.name,.value_sha256)')"
  jq -cS -n --argjson database "$database_json" --argjson role_settings "$settings_json" \
    --arg schema_sha256 "$schema_sha" --argjson schema_lines "$schema_lines" --argjson schema_bytes "$schema_bytes" \
    --arg large_object_oids_sha256 "$large_object_sha" --argjson large_object_count "$large_object_count" \
    '{schema_version:2,format:"phase29-database-inventory-v2",
      schema_ddl:{sha256:$schema_sha256,lines:$schema_lines,size_bytes:$schema_bytes,
        captures:{owners:true,acl:true,comments:true,security_labels:true}},
      database:$database,pg_db_role_setting:$role_settings,
      large_objects:{count:$large_object_count,oids_sha256:$large_object_oids_sha256}}' > "$output"
}

validate_database_inventory() {
  local inventory="$1" schema_ddl="$2" expected actual
  if [ ! -f "$inventory" ] || [ -L "$inventory" ]; then die 'database-wide inventory missing or symlinked'; fi
  if [ ! -f "$schema_ddl" ] || [ -L "$schema_ddl" ]; then die 'full schema DDL inventory missing or symlinked'; fi
  jq -e '
    .schema_version == 2 and .format == "phase29-database-inventory-v2" and
    (.schema_ddl.sha256 | test("^[0-9a-f]{64}$")) and .schema_ddl.lines > 0 and .schema_ddl.size_bytes > 0 and
    .schema_ddl.captures == {owners:true,acl:true,comments:true,security_labels:true} and
    .database.name == "DBRouterAiAtius" and (.database.owner | length > 0) and
    (.database.properties | type == "object") and (.database.acl | type == "array") and
    (.database.security_labels | type == "array") and (.database.tablespace | length > 0) and
    (.pg_db_role_setting | type == "array") and
    all(.pg_db_role_setting[]; (.role | length > 0) and (.name | length > 0) and
      (.value_sha256 | test("^[0-9a-f]{64}$"))) and
    (.large_objects.count | type == "number") and (.large_objects.oids_sha256 | test("^[0-9a-f]{64}$"))
  ' "$inventory" >/dev/null || die 'database-wide inventory v2 is incomplete or malformed'
  expected="$(jq -r '.schema_ddl.sha256' "$inventory")"
  actual="$(sha256sum "$schema_ddl" | awk '{print $1}')"
  [ "$actual" = "$expected" ] || die 'full schema DDL checksum differs from inventory'
  [ "$(stat -c %s "$schema_ddl")" -eq "$(jq -r '.schema_ddl.size_bytes' "$inventory")" ] ||
    die 'full schema DDL size differs from inventory'
}

validate_database_inventory_match() {
  validate_database_inventory "$1" "$2"
  validate_database_inventory "$3" "$4"
  cmp -s "$1" "$3" || die 'target database-wide object inventory differs from source backup'
  cmp -s "$2" "$4" || die 'target full schema DDL differs from source backup'
}

write_database_state_query() {
  local output="$1"
  cat > "$output" <<'SQL'
WITH selected AS (
  SELECT d.*, t.spcname AS tablespace_name
  FROM pg_database d JOIN pg_tablespace t ON t.oid = d.dattablespace
  WHERE d.datname = current_database()
), acl AS (
  SELECT COALESCE(jsonb_agg(value::text ORDER BY value::text), '[]'::jsonb) AS values
  FROM selected d LEFT JOIN LATERAL unnest(d.datacl) value ON true WHERE value IS NOT NULL
), labels AS (
  SELECT COALESCE(jsonb_agg(jsonb_build_object('provider',provider,'label',label) ORDER BY provider,label), '[]'::jsonb) AS values
  FROM pg_seclabel s JOIN selected d ON s.objoid=d.oid
  WHERE s.classoid='pg_database'::regclass AND s.objsubid=0
)
SELECT jsonb_build_object(
  'name', d.datname,
  'owner', pg_get_userbyid(d.datdba),
  'tablespace', d.tablespace_name,
  'properties', to_jsonb(d) - ARRAY['oid','datdba','dattablespace','datfrozenxid','datminmxid','datacl','datcollversion','tablespace_name'],
  'acl', acl.values,
  'comment', obj_description(d.oid,'pg_database'),
  'security_labels', labels.values
)::text
FROM selected d CROSS JOIN acl CROSS JOIN labels;
SQL
}

write_database_role_settings_query() {
  local output="$1"
  cat > "$output" <<'SQL'
SELECT CASE s.setrole WHEN 0 THEN '*' ELSE r.rolname END,
       split_part(value,'=',1),
       encode(convert_to(value,'UTF8'),'hex')
FROM pg_db_role_setting s
LEFT JOIN pg_roles r ON r.oid=s.setrole
CROSS JOIN LATERAL unnest(s.setconfig) value
WHERE s.setdatabase=(SELECT oid FROM pg_database WHERE datname=current_database())
ORDER BY 1,2,3;
SQL
}

create_target_database_inventory() {
  local pod="$1" output="$2" schema_output="$3" schema_dump="$tmp/target-schema.sql"
  local database_state="$tmp/target-database-state.json" role_settings="$tmp/target-role-settings.tsv"
  local large_objects="$tmp/target-large-objects.tsv" database_query="$tmp/database-state.sql" role_settings_query="$tmp/database-role-settings.sql"
  run_interruptible sudo -n k3s kubectl -n "$namespace" exec "$pod" -- \
    pg_dump -U "$database_user" -d "$database" --schema-only \
    --no-subscriptions --quote-all-identifiers --restrict-key=phase29databaseinventory > "$schema_dump"
  write_database_state_query "$database_query"
  write_database_role_settings_query "$role_settings_query"
  run_interruptible_stdin "$database_query" sudo -n k3s kubectl -n "$namespace" exec -i "$pod" -- \
    psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" -At > "$database_state"
  run_interruptible_stdin "$role_settings_query" sudo -n k3s kubectl -n "$namespace" exec -i "$pod" -- \
    psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" -At -F $'\t' > "$role_settings"
  run_interruptible sudo -n k3s kubectl -n "$namespace" exec "$pod" -- \
    psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" -At \
    --command='SELECT oid::text FROM pg_largeobject_metadata ORDER BY oid' > "$large_objects"
  normalize_database_inventory "$schema_dump" "$database_state" "$role_settings" "$large_objects" "$output" "$schema_output"
  validate_database_inventory "$output" "$schema_output"
}

validate_backup() {
  local metadata dump checksum inventory schema_ddl expected actual now generated inventory_expected inventory_actual inventory_size
  if [ ! -d "$backup_dir" ] || [ -L "$backup_dir" ]; then die 'backup directory missing or symlinked'; fi
  backup_dir="$(realpath -e "$backup_dir")"
  metadata="$backup_dir/backup.json"
  dump="$backup_dir/db/DBRouterAiAtius.sql"
  checksum="$backup_dir/db/DBRouterAiAtius.sql.sha256"
  inventory="$backup_dir/db/DBRouterAiAtius.inventory"
  schema_ddl="$backup_dir/db/DBRouterAiAtius.schema.sql"
  if [ ! -f "$metadata" ] || [ -L "$metadata" ]; then die 'backup metadata missing or symlinked'; fi
  if [ ! -f "$checksum" ] || [ -L "$checksum" ]; then die 'backup checksum missing or symlinked'; fi
  validate_database_inventory "$inventory" "$schema_ddl"
  jq -e '
    .status == "go" and
    .source.kind == "host-postgresql" and .source.host == "127.0.0.1" and .source.port == 8745 and
    .source.server_addr == "127.0.0.1" and
    .source.database == "DBRouterAiAtius" and .source.user == "admin" and
    (.source.server_version_num | test("^17[0-9]{4}$")) and
    .source.data_directory == "/var/lib/postgresql/17/main" and
    .source.systemd_unit == "postgresql@17-main.service" and .source.backend_unit_matched == true and
    .pgbouncer_crosscheck.host == "10.11.1.11" and .pgbouncer_crosscheck.port == 6432 and
    .pgbouncer_crosscheck.matched == true and
    (.pg_dump_version | test("^17\\.[0-9]+([.][0-9]+)?$")) and
    .cpu.client_millicores > 0 and .cpu.client_millicores <= 400 and
    .cpu.postgres_millicores > 0 and .cpu.postgres_millicores <= 400 and
    .cpu.aggregate_millicores == (.cpu.client_millicores + .cpu.postgres_millicores) and
    .cpu.aggregate_millicores <= 800 and
    .cpu.postgres_quota_restored == true and
    (.cpu.postgres_quota_temporarily_applied | type == "boolean") and
    (.cpu.unit_state.before.fragment_path | type == "string" and startswith("/")) and
    (.cpu.unit_state.before.drop_in_paths | type == "array") and
    .cpu.unit_state.restored == .cpu.unit_state.before and
    (if .cpu.postgres_quota_temporarily_applied then
      .cpu.unit_state.applied.fragment_path == .cpu.unit_state.before.fragment_path and
      .cpu.unit_state.applied.cpu_quota_per_sec_usec != .cpu.unit_state.before.cpu_quota_per_sec_usec and
      ((.cpu.unit_state.before.drop_in_paths - .cpu.unit_state.applied.drop_in_paths) | length) == 0
     else .cpu.unit_state.applied == .cpu.unit_state.before end) and
    .dump.structurally_valid == true and .dump.size_bytes > 643 and
    .dump.subscriptions_included == false and
    .database_inventory.format == "phase29-database-inventory-v2" and
    .database_inventory.path == "db/DBRouterAiAtius.inventory" and
    .database_inventory.schema_ddl_path == "db/DBRouterAiAtius.schema.sql" and
    .database_inventory.source_backup_matched == true and
    .database_inventory.sanitized == true and
    (.database_inventory.sha256 | test("^[0-9a-f]{64}$")) and
    .database_inventory.size_bytes > 0 and
    .invariants.public_tables >= 34 and .invariants.channels > 0 and
    .invariants.users > 0 and .invariants.tokens > 0 and .invariants.subscriptions == 0
  ' "$metadata" >/dev/null || die 'backup metadata does not satisfy canonical PostgreSQL 17 gates'
  now="$(date +%s)"; generated="$(jq -r '.generated_at_epoch' "$metadata")"
  [[ "$generated" =~ ^[0-9]+$ ]] || die 'backup generated_at_epoch is not an integer'
  if [ "$generated" -gt "$now" ] || [ $((now - generated)) -gt 3600 ]; then die 'backup is stale or future-dated'; fi
  expected="$(jq -r '.dump.sha256' "$metadata")"
  [[ "$expected" =~ ^[0-9a-f]{64}$ ]] || die 'metadata checksum malformed'
  actual="$(sha256sum "$dump" | awk '{print $1}')"
  [ "$actual" = "$expected" ] || die 'dump checksum differs from metadata'
  (cd "$(dirname "$dump")" && sha256sum --check --status "$(basename "$checksum")") || die 'checksum file validation failed'
  validate_dump_structure "$dump"
  inventory_expected="$(jq -r '.database_inventory.sha256' "$metadata")"
  inventory_actual="$(sha256sum "$inventory" | awk '{print $1}')"
  [ "$inventory_actual" = "$inventory_expected" ] || die 'database-wide inventory checksum differs from metadata'
  inventory_size="$(stat -c %s "$inventory")"
  [ "$inventory_size" = "$(jq -r '.database_inventory.size_bytes' "$metadata")" ] || die 'database-wide inventory size differs from metadata'
  [ "$(sha256sum "$schema_ddl" | awk '{print $1}')" = "$(jq -r '.database_inventory.schema_ddl_sha256' "$metadata")" ] ||
    die 'full schema DDL checksum differs from backup metadata'
}

snapshot_value() {
  local file="$1" key="$2"
  awk -F '\t' -v key="$key" '$1 == key {print $2}' "$file"
}

validate_target_snapshot() {
  local file="$1" metadata="$backup_dir/backup.json" key expected actual
  [[ "$(snapshot_value "$file" server_version_num)" =~ ^17[0-9]{4}$ ]] || die 'target is not PostgreSQL 17'
  [ "$(snapshot_value "$file" database)" = DBRouterAiAtius ] || die 'target database mismatch'
  [ "$(snapshot_value "$file" user)" = admin ] || die 'target user mismatch'
  for key in public_tables channels users tokens; do
    expected="$(jq -r ".invariants.$key" "$metadata")"
    actual="$(snapshot_value "$file" "$key")"
    if [ -z "$actual" ] || [ "$actual" != "$expected" ]; then die "restored $key invariant mismatch"; fi
  done
}

clean_inventory_keys() {
  printf '%s\n' \
    database_acl_entries database_comments database_nondefault_properties database_owner_mismatch \
    database_role_settings database_security_labels default_acls event_triggers foreign_data_wrappers foreign_servers large_objects \
    non_system_schemas nondefault_extensions publications subscriptions \
    system_namespace_user_objects user_access_methods user_casts user_languages \
    user_mappings user_transforms
}

validate_clean_inventory_shape() {
  local file="$1" expected actual
  if [ ! -f "$file" ] || [ -L "$file" ]; then die 'pre-restore database inventory is invalid'; fi
  awk -F '\t' 'NF != 2 || $1 == "" || $2 !~ /^[0-9]+$/ {exit 1}' "$file" ||
    die 'pre-restore database inventory is malformed'
  expected="$(clean_inventory_keys | LC_ALL=C sort)"
  actual="$(cut -f1 "$file" | LC_ALL=C sort)"
  [ "$actual" = "$expected" ] || die 'pre-restore database inventory keys are incomplete or duplicated'
}

validate_integrally_clean_inventory() {
  local file="$1"
  validate_clean_inventory_shape "$file"
  awk -F '\t' '$2 != 0 {exit 1}' "$file" || die 'target contains database-wide objects before restore'
}

clean_inventory_json() {
  local file="$1"
  validate_clean_inventory_shape "$file"
  jq -Rn '[inputs | split("\t") | {(.[0]): (.[1] | tonumber)}] | add' < "$file"
}

record_pre_restore_inventory() {
  local inventory_json="$1" next
  next="$(mktemp "$evidence_dir/.restore.inventory.XXXXXX")"
  chmod 600 "$next"
  jq --argjson inventory "$inventory_json" '.target = (.target // {}) | .target.pre_restore_inventory = $inventory' \
    "$restore_evidence" > "$next"
  mv -f "$next" "$restore_evidence"
  write_target_state in-progress || die 'failed to persist pre-restore inventory evidence'
}

query_pre_restore_inventory() {
  local pod="$1" output="$2" query="$tmp/pre-restore-inventory.sql"
  cat > "$query" <<'SQL'
SELECT 'default_acls', count(*)::text FROM pg_default_acl
UNION ALL SELECT 'database_acl_entries', count(*)::text
  FROM pg_database d CROSS JOIN LATERAL unnest(d.datacl) acl WHERE d.datname=current_database()
UNION ALL SELECT 'database_comments', count(*)::text FROM pg_database d
  WHERE d.datname=current_database() AND obj_description(d.oid,'pg_database') IS NOT NULL
UNION ALL SELECT 'database_nondefault_properties', count(*)::text FROM pg_database d
  JOIN pg_tablespace t ON t.oid=d.dattablespace
  WHERE d.datname=current_database() AND (d.datistemplate OR NOT d.datallowconn OR d.datconnlimit <> -1 OR t.spcname <> 'pg_default')
UNION ALL SELECT 'database_owner_mismatch', count(*)::text FROM pg_database d
  WHERE d.datname=current_database() AND pg_get_userbyid(d.datdba) <> 'admin'
UNION ALL SELECT 'database_role_settings', count(*)::text FROM pg_db_role_setting s
  WHERE s.setdatabase=(SELECT oid FROM pg_database WHERE datname=current_database())
UNION ALL SELECT 'database_security_labels', count(*)::text FROM pg_seclabel s
  WHERE s.classoid='pg_database'::regclass AND s.objoid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND s.objsubid=0
UNION ALL SELECT 'event_triggers', count(*)::text FROM pg_event_trigger
UNION ALL SELECT 'foreign_data_wrappers', count(*)::text FROM pg_foreign_data_wrapper WHERE oid >= 16384
UNION ALL SELECT 'foreign_servers', count(*)::text FROM pg_foreign_server
UNION ALL SELECT 'large_objects', count(*)::text FROM pg_largeobject_metadata
UNION ALL SELECT 'non_system_schemas', count(*)::text FROM pg_namespace
  WHERE nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast', 'public')
    AND nspname !~ '^pg_temp_' AND nspname !~ '^pg_toast_temp_'
UNION ALL SELECT 'nondefault_extensions', count(*)::text FROM pg_extension WHERE extname <> 'plpgsql'
UNION ALL SELECT 'publications', count(*)::text FROM pg_publication
UNION ALL SELECT 'subscriptions', count(*)::text FROM pg_subscription
UNION ALL SELECT 'user_access_methods', count(*)::text FROM pg_am WHERE oid >= 16384
UNION ALL SELECT 'user_casts', count(*)::text FROM pg_cast WHERE oid >= 16384
UNION ALL SELECT 'user_languages', count(*)::text FROM pg_language WHERE oid >= 16384
UNION ALL SELECT 'user_mappings', count(*)::text FROM pg_user_mapping
UNION ALL SELECT 'user_transforms', count(*)::text FROM pg_transform WHERE oid >= 16384
UNION ALL SELECT 'system_namespace_user_objects', count(*)::text FROM (
  SELECT c.oid FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT p.oid FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT t.oid FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE t.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT o.oid FROM pg_operator o JOIN pg_namespace n ON n.oid=o.oprnamespace WHERE o.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT o.oid FROM pg_opclass o JOIN pg_namespace n ON n.oid=o.opcnamespace WHERE o.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT o.oid FROM pg_opfamily o JOIN pg_namespace n ON n.oid=o.opfnamespace WHERE o.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT c.oid FROM pg_collation c JOIN pg_namespace n ON n.oid=c.collnamespace WHERE c.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT c.oid FROM pg_conversion c JOIN pg_namespace n ON n.oid=c.connamespace WHERE c.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT c.oid FROM pg_ts_config c JOIN pg_namespace n ON n.oid=c.cfgnamespace WHERE c.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT d.oid FROM pg_ts_dict d JOIN pg_namespace n ON n.oid=d.dictnamespace WHERE d.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT p.oid FROM pg_ts_parser p JOIN pg_namespace n ON n.oid=p.prsnamespace WHERE p.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
  UNION ALL SELECT t.oid FROM pg_ts_template t JOIN pg_namespace n ON n.oid=t.tmplnamespace WHERE t.oid >= 16384 AND n.nspname IN ('pg_catalog','information_schema','pg_toast')
) system_user_objects;
SQL
  chmod 600 "$query"
  run_interruptible_stdin "$query" sudo -n k3s kubectl -n "$namespace" exec -i "$pod" -- \
    psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" -At -F $'\t' > "$output"
}

validate_pv_binding() {
  local pvc_json="$1" pv_json="$2" pvc_uid volume_name
  pvc_uid="$(jq -r '.metadata.uid' "$pvc_json")"
  volume_name="$(jq -r '.spec.volumeName' "$pvc_json")"
  if [ -z "$pvc_uid" ] || [ "$pvc_uid" = null ]; then die 'PVC UID missing'; fi
  if [ -z "$volume_name" ] || [ "$volume_name" = null ]; then die 'PVC is not bound to a PV'; fi
  [ "$(jq -r '.status.phase' "$pvc_json")" = Bound ] || die 'PVC is not Bound'
  [ "$(jq -r '.spec.storageClassName' "$pvc_json")" = local-path ] || die 'PVC is not local-path'
  [ "$(jq -r '.metadata.name' "$pv_json")" = "$volume_name" ] || die 'PV name differs from PVC volumeName'
  [ "$(jq -r '.spec.claimRef.uid' "$pv_json")" = "$pvc_uid" ] || die 'PV claimRef UID differs from PVC UID'
  [ "$(jq -r '.spec.claimRef.namespace' "$pv_json")" = "$namespace" ] || die 'PV claimRef namespace mismatch'
  jq -e '
    any(.spec.nodeAffinity.required.nodeSelectorTerms[]?.matchExpressions[]?;
      .key == "kubernetes.io/hostname" and .operator == "In" and (.values | index("atius-srv-1") != null))
  ' "$pv_json" >/dev/null || die 'PV is not pinned to atius-srv-1'
}

assert_runtime_absent() {
  local resources
  resources="$(sudo -n k3s kubectl -n "$namespace" get deployment,statefulset,daemonset,replicaset,replicationcontroller,job,cronjob,pod \
    -o json)"
  validate_runtime_inventory "$resources"
}

validate_runtime_inventory() {
  local resources="$1"
  jq -e '
    ([.items[] | select(.kind == "StatefulSet" and .metadata.name == "router-ai-atius-postgres")]) as $sets |
    if ($sets | length) == 0 then
      (.items | length) == 0
    else
      ($sets | length) == 1 and
      ($sets[0].metadata.uid | type == "string" and length > 0) and
      all(.items[];
        (.kind == "StatefulSet" and .metadata.name == "router-ai-atius-postgres") or
        (.kind == "Pod" and
         .metadata.labels["app.kubernetes.io/name"] == "router-ai-atius-postgres" and
         any(.metadata.ownerReferences[]?;
           .apiVersion == "apps/v1" and .kind == "StatefulSet" and
           .name == "router-ai-atius-postgres" and .uid == $sets[0].metadata.uid and
           .controller == true)))
    end
  ' <<< "$resources" >/dev/null || die 'non-PostgreSQL runtime exists during restore'
}

prepare_restore_slot() {
  local prior_status prior_epoch prior_path prior_sha state_cluster state_target
  restore_evidence="$evidence_dir/restore.json"
  if [ -L "$restore_evidence" ]; then die 'restore evidence must not be a symlink'; fi
  [ -n "$target_state_file" ] || die 'restore target state path was not initialized'
  if [ ! -e "$target_state_file" ]; then
    $retry_no_go && die 'retry requested but no canonical target state exists'
    [ ! -e "$restore_evidence" ] || die 'untracked restore evidence already exists'
    return
  fi
  [ -f "$target_state_file" ] || die 'canonical target state is not a regular file'
  [ "$(stat -c %U:%a "$target_state_file")" = "$(id -un):600" ] || die 'canonical target state owner/mode invalid'
  prior_status="$(jq -r '.status // empty' "$target_state_file")"
  prior_path="$(jq -r '.evidence_path // empty' "$target_state_file")"
  prior_sha="$(jq -r '.evidence_sha256 // empty' "$target_state_file")"
  state_cluster="$(jq -r '.cluster_uid // empty' "$target_state_file")"
  state_target="$(jq -r '.target // empty' "$target_state_file")"
  [ "$state_target" = 'router-ai-atius/DBRouterAiAtius@atius-srv-1' ] || die 'canonical target identity mismatch'
  [ "$state_cluster" = "$cluster_uid" ] || die 'canonical target state belongs to another cluster'
  [[ "$prior_sha" =~ ^[0-9a-f]{64}$ ]] || die 'canonical target evidence checksum malformed'
  if [ ! -f "$prior_path" ] || [ -L "$prior_path" ]; then die 'canonical prior evidence missing or symlinked'; fi
  [ "$(sha256sum "$prior_path" | awk '{print $1}')" = "$prior_sha" ] || die 'canonical prior evidence checksum mismatch'
  case "$prior_status" in
    go)
      $replace_go || die 'target already has a successful restore; use --replace-go for an explicit fresh rehearsal'
      [ "${PHASE29_RESTORE_REPLACE_CONFIRM:-}" = REPLACE_STALE_SHADOW_RESTORE ] ||
        die 'missing exact successful-restore replacement confirmation'
      replace_go_target=true
      ;;
    in-progress) die 'target has an unresolved in-progress restore; manual reconciliation required' ;;
    no-go) ;;
    *) die 'canonical target state has an invalid status' ;;
  esac
  if [ "$prior_status" = no-go ]; then
    $retry_no_go || die 'canonical no-go requires explicit --retry-no-go'
    [ "${PHASE29_RESTORE_REPLACE_CONFIRM:-}" = REPLACE_STALE_SHADOW_RESTORE ] ||
      die 'missing exact no-go target reset confirmation'
    replace_go_target=true
  elif $retry_no_go; then
    die '--retry-no-go cannot replace a successful restore'
  fi
  if [ -e "$restore_evidence" ] && [ "$restore_evidence" != "$prior_path" ]; then
    die 'new evidence path already contains an unrelated restore record'
  fi
  prior_epoch="$(jq -r '.generated_at_epoch // empty' "$prior_path")"
  [[ "$prior_epoch" =~ ^[0-9]+$ ]] || die 'no-go restore evidence has no valid generation epoch'
  retry_prior_sha256="$prior_sha"
  retry_source_evidence="$prior_path"
  retry_prior_evidence="pending:$prior_epoch"
  lineage_prior_status="$prior_status"
}

create_restore_evidence() {
  local cluster_uid="$1" started_at generated_at_epoch next archive prior_epoch archive_status
  started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  generated_at_epoch="$(date +%s)"
  next="$(mktemp "$evidence_dir/.restore.next.XXXXXX")"
  chmod 600 "$next"
  if [[ "$retry_prior_evidence" == pending:* ]]; then
    prior_epoch="${retry_prior_evidence#pending:}"
    archive_status="${lineage_prior_status:-no-go}"
    archive="$(mktemp "$evidence_dir/restore.$archive_status.$prior_epoch.XXXXXX.json")"
    chmod 600 "$archive"
    cp --preserve=mode,timestamps "$retry_source_evidence" "$archive"
    [ "$(sha256sum "$archive" | awk '{print $1}')" = "$retry_prior_sha256" ] || die 'retry archive checksum mismatch'
    retry_prior_evidence="$(basename "$archive")"
    write_target_state "$archive_status" "$archive" || die 'failed to preserve canonical restore lineage'
  else
    retry_prior_evidence=""
  fi
  jq -n --arg cluster_uid "$cluster_uid" --arg started_at "$started_at" \
    --arg retry_of "$retry_prior_evidence" --arg retry_sha256 "$retry_prior_sha256" --argjson generated_at_epoch "$generated_at_epoch" \
    '{status:"in-progress",restore_passed:false,cluster_uid:$cluster_uid,started_at:$started_at,generated_at_epoch:$generated_at_epoch,retry_of:(if $retry_of == "" then null else $retry_of end),retry_sha256:(if $retry_sha256 == "" then null else $retry_sha256 end)}' \
    > "$next"
  mv -f "$next" "$restore_evidence"
  write_target_state in-progress || die 'failed to persist canonical in-progress target state'
}

signal_self_test() {
  evidence_dir="${PHASE29_SIGNAL_TEST_DIR:?PHASE29_SIGNAL_TEST_DIR required}"
  if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then die 'signal self-test directory invalid'; fi
  cluster_uid='signal-test-cluster'
  acquire_restore_lock
  prepare_restore_slot
  create_restore_evidence "$cluster_uid"
  restore_started=true
  trap on_restore_error ERR
  trap on_exit EXIT
  trap 'on_signal INT' INT
  trap 'on_signal TERM' TERM
  echo ready > "$evidence_dir/ready"
  if [ "${PHASE29_SIGNAL_STUBBORN:-0}" = 1 ]; then
    run_interruptible bash -c 'trap "" INT TERM; sleep 60'
  else
    run_interruptible sleep 60
  fi
}

self_test() {
  local tmp dump checksum metadata pvc pv target evidence retry_evidence retry_archive state_before injected_failure terminal_success stdin_source stdin_output inventory inventory_copy inventory_sha unit_state clean contaminated contaminant inventory_schema inventory_schema_copy inventory_database_state inventory_role_settings inventory_large_objects schema_sha
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  backup_dir="$tmp/backup"; mkdir -p "$backup_dir/db"
  dump="$backup_dir/db/DBRouterAiAtius.sql"; checksum="$backup_dir/db/DBRouterAiAtius.sql.sha256"
  {
    echo '-- PostgreSQL database dump'
    echo 'CREATE TABLE public.channels (id bigint);'
    echo 'CREATE TABLE public.users (id bigint);'
    echo 'CREATE TABLE public.tokens (id bigint);'
    printf '%0800d\n' 0
    echo '-- PostgreSQL database dump complete'
  } > "$dump"
  (cd "$(dirname "$dump")" && sha256sum "$(basename "$dump")" > "$(basename "$checksum")")
  inventory="$backup_dir/db/DBRouterAiAtius.inventory"
  inventory_schema="$tmp/inventory-schema.sql"; inventory_schema_copy="$backup_dir/db/DBRouterAiAtius.schema.sql"
  inventory_database_state="$tmp/inventory-database-state.json"; inventory_role_settings="$tmp/inventory-role-settings.tsv"
  inventory_large_objects="$tmp/inventory-large-objects.tsv"
  printf '%s\n' 'CREATE TABLE "public"."channels" ();' 'ALTER TABLE "public"."channels" OWNER TO "admin";' \
    'COMMENT ON TABLE "public"."channels" IS '\''catalog'\'';' 'GRANT SELECT ON TABLE "public"."channels" TO "admin";' > "$inventory_schema"
  jq -cn '{name:"DBRouterAiAtius",owner:"admin",tablespace:"pg_default",properties:{datname:"DBRouterAiAtius",datallowconn:true},acl:[],comment:null,security_labels:[]}' > "$inventory_database_state"
  printf 'admin\tstatement_timeout\t313030306d73\n' > "$inventory_role_settings"
  : > "$inventory_large_objects"
  normalize_database_inventory "$inventory_schema" "$inventory_database_state" "$inventory_role_settings" \
    "$inventory_large_objects" "$inventory" "$inventory_schema_copy"
  inventory_sha="$(sha256sum "$inventory" | awk '{print $1}')"
  schema_sha="$(sha256sum "$inventory_schema_copy" | awk '{print $1}')"
  unit_state='{"fragment_path":"/usr/lib/systemd/system/postgresql@.service","cpu_quota_per_sec_usec":"infinity","drop_in_paths":[]}'
  metadata="$backup_dir/backup.json"
  jq -n --arg sha "$(sha256sum "$dump" | awk '{print $1}')" --arg inventory_sha "$inventory_sha" --arg schema_sha "$schema_sha" \
    --argjson inventory_size "$(stat -c %s "$inventory")" --argjson now "$(date +%s)" --argjson unit_state "$unit_state" \
    '{status:"go",generated_at:"2026-07-13T00:00:00Z",generated_at_epoch:$now,source:{kind:"host-postgresql",host:"127.0.0.1",port:8745,server_addr:"127.0.0.1",database:"DBRouterAiAtius",user:"admin",server_version_num:"170010",data_directory:"/var/lib/postgresql/17/main",systemd_unit:"postgresql@17-main.service",backend_unit_matched:true},pgbouncer_crosscheck:{host:"10.11.1.11",port:6432,matched:true},pg_dump_version:"17.10",cpu:{client_millicores:400,postgres_millicores:400,aggregate_millicores:800,postgres_quota_temporarily_applied:false,postgres_quota_restored:true,unit_state:{before:$unit_state,applied:$unit_state,restored:$unit_state}},dump:{structurally_valid:true,size_bytes:1000,sha256:$sha,subscriptions_included:false,owners_included:true,acl_included:true},database_inventory:{format:"phase29-database-inventory-v2",path:"db/DBRouterAiAtius.inventory",schema_ddl_path:"db/DBRouterAiAtius.schema.sql",schema_ddl_sha256:$schema_sha,size_bytes:$inventory_size,sha256:$inventory_sha,source_backup_matched:true,target_equality_required:true,sanitized:true},invariants:{public_tables:34,channels:4,users:2,tokens:3,subscriptions:0}}' > "$metadata"
  validate_backup
  inventory_copy="$tmp/target.inventory"
  cp "$inventory" "$inventory_copy"
  cp "$inventory_schema_copy" "$tmp/target.schema.sql"
  validate_database_inventory_match "$inventory" "$inventory_schema_copy" "$inventory_copy" "$tmp/target.schema.sql"
  jq '.database.owner="rogue"' "$inventory_copy" > "$tmp/changed.inventory"; mv "$tmp/changed.inventory" "$inventory_copy"
  if (validate_database_inventory_match "$inventory" "$inventory_schema_copy" "$inventory_copy" "$tmp/target.schema.sql") 2>/dev/null; then
    die 'database owner mismatch was accepted'
  fi
  cp "$inventory" "$inventory_copy"
  printf 'CREATE VIEW "public"."rogue" AS SELECT 1;\n' >> "$tmp/target.schema.sql"
  if (validate_database_inventory_match "$inventory" "$inventory_schema_copy" "$inventory_copy" "$tmp/target.schema.sql") 2>/dev/null; then
    die 'full schema DDL mismatch was accepted'
  fi
  cp "$inventory" "$tmp/inventory.good"
  printf 'tamper\n' >> "$inventory"
  if (validate_backup) 2>/dev/null; then die 'tampered database-wide backup inventory was accepted'; fi
  mv "$tmp/inventory.good" "$inventory"
  jq '.source.server_version_num = "150010"' "$metadata" > "$tmp/bad.json"; mv "$tmp/bad.json" "$metadata"
  if (validate_backup) 2>/dev/null; then die 'PostgreSQL 15 source metadata was accepted'; fi
  jq '.source.server_version_num = "170010"' "$metadata" > "$tmp/good.json"; mv "$tmp/good.json" "$metadata"
  pvc="$tmp/pvc.json"; pv="$tmp/pv.json"
  jq -n '{metadata:{name:"router-ai-atius-postgres-data",uid:"claim-uid"},spec:{volumeName:"pv-one",storageClassName:"local-path"},status:{phase:"Bound"}}' > "$pvc"
  jq -n '{metadata:{name:"pv-one"},spec:{claimRef:{uid:"claim-uid",namespace:"router-ai-atius"},nodeAffinity:{required:{nodeSelectorTerms:[{matchExpressions:[{key:"kubernetes.io/hostname",operator:"In",values:["atius-srv-1"]}]}]}}}}' > "$pv"
  validate_pv_binding "$pvc" "$pv"
  jq '.spec.claimRef.uid = "other"' "$pv" > "$tmp/bad-pv.json"
  if (validate_pv_binding "$pvc" "$tmp/bad-pv.json") 2>/dev/null; then die 'wrong PV claimRef UID was accepted'; fi
  target="$tmp/target.tsv"
  printf 'database\tDBRouterAiAtius\nuser\tadmin\nserver_version_num\t170010\npublic_tables\t34\nchannels\t4\nusers\t2\ntokens\t3\n' > "$target"
  validate_target_snapshot "$target"
  sed -i 's/channels\t4/channels\t5/' "$target"
  if (validate_target_snapshot "$target") 2>/dev/null; then die 'restored invariant mismatch was accepted'; fi
  clean="$tmp/clean-inventory.tsv"
  clean_inventory_keys | awk '{print $1 "\t0"}' > "$clean"
  validate_integrally_clean_inventory "$clean"
  jq -e '.non_system_schemas == 0 and .subscriptions == 0' <<< "$(clean_inventory_json "$clean")" >/dev/null ||
    die 'clean inventory JSON is incomplete'
  for contaminant in database_acl_entries database_comments database_nondefault_properties database_owner_mismatch \
    database_role_settings database_security_labels non_system_schemas large_objects publications event_triggers \
    subscriptions foreign_data_wrappers user_casts system_namespace_user_objects; do
    contaminated="$tmp/contaminated-$contaminant.tsv"
    awk -F '\t' -v key="$contaminant" 'BEGIN {OFS="\t"} {$2 = ($1 == key ? 1 : $2); print}' "$clean" > "$contaminated"
    if (validate_integrally_clean_inventory "$contaminated") 2>/dev/null; then die "$contaminant contaminant was accepted"; fi
  done
  validate_runtime_inventory '{"items":[]}'
  if (validate_runtime_inventory '{"items":[{"kind":"Deployment","metadata":{"name":"router-ai-atius"}}]}') 2>/dev/null; then
    die 'unlabeled router workload was accepted during restore'
  fi
  if (validate_runtime_inventory '{"items":[{"kind":"ReplicaSet","metadata":{"name":"rogue-router"}}]}') 2>/dev/null; then
    die 'rogue ReplicaSet was accepted during restore'
  fi
  if (validate_runtime_inventory '{"items":[{"kind":"ReplicationController","metadata":{"name":"rogue-redis"}}]}') 2>/dev/null; then
    die 'rogue ReplicationController was accepted during restore'
  fi
  if (validate_runtime_inventory '{"items":[{"kind":"StatefulSet","metadata":{"name":"router-ai-atius-postgres","uid":"ss-1"}},{"kind":"Pod","metadata":{"name":"rogue","labels":{"app.kubernetes.io/name":"router-ai-atius-postgres"}}}]}') 2>/dev/null; then
    die 'rogue PostgreSQL-labeled pod without ownerReference was accepted'
  fi
  validate_runtime_inventory '{"items":[{"kind":"StatefulSet","metadata":{"name":"router-ai-atius-postgres","uid":"ss-1"}},{"kind":"Pod","metadata":{"name":"router-ai-atius-postgres-0","labels":{"app.kubernetes.io/name":"router-ai-atius-postgres"},"ownerReferences":[{"apiVersion":"apps/v1","kind":"StatefulSet","name":"router-ai-atius-postgres","uid":"ss-1","controller":true}]}}]}'
  stdin_source="$tmp/stdin-source"; stdin_output="$tmp/stdin-output"
  printf 'phase29-stdin-sentinel\n' > "$stdin_source"
  # shellcheck disable=SC2016
  run_interruptible_stdin "$stdin_source" sh -c 'cat > "$1"' _ "$stdin_output"
  grep -Fxq phase29-stdin-sentinel "$stdin_output" || die 'interruptible runner discarded explicit stdin'
  HOME="$tmp/home"; export HOME
  evidence="$tmp/evidence"; mkdir -m 700 "$evidence"
  evidence_dir="$evidence"
  cluster_uid=cluster-test
  acquire_restore_lock
  jq -n --argjson epoch "$(date +%s)" '{status:"no-go",generated_at_epoch:$epoch}' > "$evidence/restore.json"
  chmod 600 "$evidence/restore.json"
  restore_evidence="$evidence/restore.json"
  write_target_state no-go
  state_before="$(sha256sum "$target_state_file" | awk '{print $1}')"
  for injected_failure in sha256 jq rename; do
    if PHASE29_TEST_STATE_WRITE_FAIL="$injected_failure" write_target_state no-go; then
      die "canonical state writer accepted injected $injected_failure failure"
    fi
    [ "$(sha256sum "$target_state_file" | awk '{print $1}')" = "$state_before" ] ||
      die "canonical state changed after injected $injected_failure failure"
    jq -e . "$target_state_file" >/dev/null || die 'canonical state became invalid after injected writer failure'
  done
  retry_no_go=false
  if (prepare_restore_slot) 2>/dev/null; then die 'existing no-go evidence was accepted without explicit retry'; fi
  retry_no_go=true
  PHASE29_RESTORE_REPLACE_CONFIRM=REPLACE_STALE_SHADOW_RESTORE; export PHASE29_RESTORE_REPLACE_CONFIRM
  retry_evidence="$tmp/retry-evidence"; mkdir -m 700 "$retry_evidence"
  evidence_dir="$retry_evidence"
  prepare_restore_slot
  create_restore_evidence cluster-test
  unset PHASE29_RESTORE_REPLACE_CONFIRM
  retry_archive="$retry_evidence/$retry_prior_evidence"
  if [ ! -f "$retry_archive" ] || [ "$(jq -r '.status' "$retry_evidence/restore.json")" != in-progress ]; then
    die 'no-go retry did not archive prior evidence'
  fi
  [ "$(jq -r '.retry_of' "$retry_evidence/restore.json")" = "$(basename "$retry_archive")" ] || die 'retry lineage missing'
  [ "$(jq -r '.retry_sha256' "$retry_evidence/restore.json")" = "$(sha256sum "$retry_archive" | awk '{print $1}')" ] || die 'retry checksum lineage missing'
  [ "$(jq -r '.evidence_path' "$target_state_file")" = "$retry_evidence/restore.json" ] || die 'canonical state did not move to retry evidence'
  [ "$(jq -r '.evidence_sha256' "$target_state_file")" = "$(sha256sum "$retry_evidence/restore.json" | awk '{print $1}')" ] || die 'canonical retry checksum mismatch'
  jq -n --argjson epoch "$(date +%s)" '{status:"go",generated_at_epoch:$epoch}' > "$retry_evidence/restore.json"
  restore_evidence="$retry_evidence/restore.json"
  write_target_state go
  restore_started=true
  mark_no_go
  [ "$(jq -r '.status' "$restore_evidence")" = go ] || die 'mark_no_go downgraded canonical go evidence'
  [ "$(jq -r '.status' "$target_state_file")" = go ] || die 'mark_no_go downgraded canonical go state'
  terminal_success="$tmp/terminal-success.json"
  cp "$restore_evidence" "$terminal_success"
  restore_started=true
  PHASE29_TEST_TERMINAL_SIGNAL=1 publish_restore_success "$terminal_success"
  [ "$(jq -r '.status' "$restore_evidence")" = go ] || die 'terminal signal downgraded go evidence'
  [ "$(jq -r '.status' "$target_state_file")" = go ] || die 'terminal signal downgraded go state'
  if (prepare_restore_slot) 2>/dev/null; then die 'successful restore evidence was accepted for retry'; fi
  jq '.generated_at_epoch = "now"' "$metadata" > "$tmp/bad-epoch.json"; mv "$tmp/bad-epoch.json" "$metadata"
  if (validate_backup) 2>/dev/null; then die 'non-numeric backup epoch was accepted'; fi
  quota_ok '80000 100000' || die '800m quota rejected'
  if quota_ok 'max 100000'; then die 'unbounded quota accepted'; fi
  if quota_ok 'now 100000'; then die 'non-numeric quota accepted'; fi
  echo 'restore rehearsal self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --backup-dir) backup_dir="${2:?}"; shift ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --cleanup-evidence) cleanup_evidence="${2:?}"; shift ;;
    --bootstrap-evidence) bootstrap_evidence="${2:?}"; shift ;;
    --retry-no-go) retry_no_go=true ;;
    --replace-go) replace_go=true ;;
    --self-test-signal) signal_self_test; exit 0 ;;
    --self-test) self_test; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

[ "$mode" = live ] || {
  echo 'restore rehearsal dry-run: PostgreSQL-only apply, PV Retain, clean target and fail-closed restore planned; no command executed'
  exit 0
}

[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[ "${PHASE29_RESTORE_CONFIRM:-}" = RESTORE_CANONICAL_BACKUP_TO_CLEAN_K3S_POSTGRES17 ] ||
  die 'missing exact restore confirmation'
[ -n "$backup_dir" ] || die '--backup-dir or K3S_BACKUP_DIR required'
[ -n "$cleanup_evidence" ] || die '--cleanup-evidence required'
[ -n "$bootstrap_evidence" ] || die '--bootstrap-evidence required'
require_profile
for command in flock jq setsid sha256sum; do command -v "$command" >/dev/null || die "required command missing: $command"; done
validate_backup

[ -n "$evidence_dir" ] || die '--evidence-dir required'
if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then die 'evidence directory missing or symlinked'; fi
evidence_dir="$(realpath -e "$evidence_dir")"
[ "$(stat -c %U:%a "$evidence_dir")" = "$(id -un):700" ] || die 'evidence directory must be owned by the caller with mode 700'
acquire_restore_lock
cluster_uid="$(sudo -n k3s kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')"
prepare_restore_slot
create_restore_evidence "$cluster_uid"
restore_started=true
trap on_restore_error ERR
trap on_exit EXIT
trap 'on_signal INT' INT
trap 'on_signal TERM' TERM

assert_runtime_absent
run_interruptible env RUN_K3S_ROUTER_APPLY=1 PHASE29_APPLY_CONFIRM=APPLY_CLUSTERIP_SHADOW_ONLY \
  scripts/k3s-router-apply-shadow.sh --live --stage postgres \
  --cleanup-evidence "$cleanup_evidence" --bootstrap-evidence "$bootstrap_evidence"
assert_runtime_absent

pod_json="$(sudo -n k3s kubectl -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius-postgres -o json)"
[ "$(jq '.items | length' <<< "$pod_json")" -eq 1 ] || die 'expected exactly one PostgreSQL pod'
pod="$(jq -r '.items[0].metadata.name' <<< "$pod_json")"
[ "$(jq -r '.items[0].spec.nodeName' <<< "$pod_json")" = atius-srv-1 ] || die 'PostgreSQL pod is outside atius-srv-1'
jq -e '.items[0].status.containerStatuses | length == 1 and all(.ready == true)' <<< "$pod_json" >/dev/null || die 'PostgreSQL pod is not Ready'
tmp="$(mktemp -d /dev/shm/phase29-restore.XXXXXX)"

if $replace_go_target; then
  run_interruptible sudo -n k3s kubectl -n "$namespace" exec "$pod" -- \
    dropdb --force -U "$database_user" "$database"
  run_interruptible sudo -n k3s kubectl -n "$namespace" exec "$pod" -- \
    createdb -U "$database_user" -O "$database_user" --template=template0 --locale=pt_BR.UTF-8 "$database"
  # shellcheck disable=SC2016
  run_interruptible sudo -n k3s kubectl -n "$namespace" exec "$pod" -- \
    psql -X --set ON_ERROR_STOP=on -U "$database_user" -d postgres -c \
    'DO $phase29$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '\''postgres'\'') THEN CREATE ROLE postgres NOLOGIN; END IF; END $phase29$;'
  helper="$HOME/.local/bin/atius-vault-env"
  [ -x "$helper" ] || die 'Vault helper unavailable for target role synchronization'
  set +x
  # shellcheck disable=SC1090
  source <("$helper" router-ai-atius)
  [ -n "${POSTGRES_PASSWORD:-}" ] || die 'Vault profile did not provide the database password'
  case "$POSTGRES_PASSWORD" in *$'\r'*|*$'\n'*) die 'database password contains a forbidden line break' ;; esac
  password_sql="$tmp/admin-password.sql"
  escaped_password="${POSTGRES_PASSWORD//\'/\'\'}"
  printf "ALTER ROLE %s PASSWORD '%s';\n" "$database_user" "$escaped_password" > "$password_sql"
  chmod 600 "$password_sql"
  unset POSTGRES_PASSWORD escaped_password
  run_interruptible_stdin "$password_sql" sudo -n k3s kubectl -n "$namespace" exec -i "$pod" -- \
    psql -X --set ON_ERROR_STOP=on -U "$database_user" -d postgres >/dev/null
  rm -f "$password_sql"
fi

target_version="$(sudo -n k3s kubectl -n "$namespace" exec "$pod" -- psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" -Atc "select current_setting('server_version_num')")"
[[ "$target_version" =~ ^17[0-9]{4}$ ]] || die 'k3s target is not PostgreSQL 17'
pre_restore_inventory="$tmp/pre-restore-inventory.tsv"
query_pre_restore_inventory "$pod" "$pre_restore_inventory"
pre_restore_inventory_json="$(clean_inventory_json "$pre_restore_inventory")"
record_pre_restore_inventory "$pre_restore_inventory_json"
validate_integrally_clean_inventory "$pre_restore_inventory"
clean_probe="$tmp/clean-probe.sql"
cat > "$clean_probe" <<'SQL'
BEGIN;
DROP SCHEMA public RESTRICT;
ROLLBACK;
SQL
chmod 600 "$clean_probe"
run_interruptible_stdin "$clean_probe" sudo -n k3s kubectl -n "$namespace" exec -i "$pod" -- \
  psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" >/dev/null

sudo -n k3s kubectl -n "$namespace" get pvc -o json > "$tmp/pvcs.json"
mapfile -t pvc_names < <(jq -r '.items[].metadata.name' "$tmp/pvcs.json")
if [ "${#pvc_names[@]}" -ne 1 ] || [ "${pvc_names[0]}" != router-ai-atius-postgres-data ]; then
  die 'PostgreSQL-only stage must contain exactly the PostgreSQL PVC'
fi
: > "$tmp/pv-evidence.jsonl"
for pvc_name in "${pvc_names[@]}"; do
  pvc_json="$tmp/pvc-$pvc_name.json"
  sudo -n k3s kubectl -n "$namespace" get pvc "$pvc_name" -o json > "$pvc_json"
  pv_name="$(jq -r '.spec.volumeName' "$pvc_json")"
  pv_json="$tmp/pv-$pv_name.json"
  sudo -n k3s kubectl get pv "$pv_name" -o json > "$pv_json"
  validate_pv_binding "$pvc_json" "$pv_json"
  sudo -n k3s kubectl patch pv "$pv_name" --type=merge -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}' >/dev/null
  readback="$(sudo -n k3s kubectl get pv "$pv_name" -o json)"
  [ "$(jq -r '.spec.persistentVolumeReclaimPolicy' <<< "$readback")" = Retain ] || die "PV $pv_name Retain readback failed"
  jq -n --arg pvc "$pvc_name" --arg pvc_uid "$(jq -r '.metadata.uid' "$pvc_json")" \
    --arg pv "$pv_name" --arg node atius-srv-1 --arg reclaim_policy Retain \
    '{pvc:$pvc,pvc_uid:$pvc_uid,pv:$pv,node:$node,reclaim_policy:$reclaim_policy,claim_uid_matched:true}' >> "$tmp/pv-evidence.jsonl"
done
pvs_json="$(jq -s '.' "$tmp/pv-evidence.jsonl")"

dump="$backup_dir/db/DBRouterAiAtius.sql"
validate_backup
if grep -Eq '^CREATE SCHEMA public;' "$dump"; then
  restore_input="$tmp/restore-with-schema-drop.sql"
  { printf 'DROP SCHEMA public RESTRICT;\n'; cat "$dump"; } > "$restore_input"
  chmod 600 "$restore_input"
else
  restore_input="$dump"
fi
run_interruptible_stdin "$restore_input" sudo -n k3s kubectl -n "$namespace" exec -i "$pod" -- \
  psql -X --set ON_ERROR_STOP=on --single-transaction -U "$database_user" -d "$database"

target_snapshot="$tmp/target.tsv"
run_interruptible sudo -n k3s kubectl -n "$namespace" exec "$pod" -- \
  psql -X --set ON_ERROR_STOP=on -U "$database_user" -d "$database" \
  -At -F $'\t' -c "SELECT 'database',current_database() UNION ALL SELECT 'user',current_user UNION ALL SELECT 'server_version_num',current_setting('server_version_num') UNION ALL SELECT 'public_tables',count(*)::text FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' UNION ALL SELECT 'channels',count(*)::text FROM public.channels UNION ALL SELECT 'users',count(*)::text FROM public.users UNION ALL SELECT 'tokens',count(*)::text FROM public.tokens" > "$target_snapshot"
validate_target_snapshot "$target_snapshot"
source_inventory="$backup_dir/db/DBRouterAiAtius.inventory"
source_schema_ddl="$backup_dir/db/DBRouterAiAtius.schema.sql"
target_inventory="$tmp/target.inventory"
target_schema_ddl="$tmp/target.schema.sql"
create_target_database_inventory "$pod" "$target_inventory" "$target_schema_ddl"
validate_database_inventory_match "$source_inventory" "$source_schema_ddl" "$target_inventory" "$target_schema_ddl"
source_inventory_sha256="$(sha256sum "$source_inventory" | awk '{print $1}')"
target_inventory_sha256="$(sha256sum "$target_inventory" | awk '{print $1}')"
source_schema_sha256="$(sha256sum "$source_schema_ddl" | awk '{print $1}')"
target_schema_sha256="$(sha256sum "$target_schema_ddl" | awk '{print $1}')"
[ "$source_inventory_sha256" = "$target_inventory_sha256" ] || die 'database-wide inventory checksum mismatch after restore'
[ "$source_schema_sha256" = "$target_schema_sha256" ] || die 'full schema DDL checksum mismatch after restore'

cpu_max="$(cpu_max_value)"; quota_ok "$cpu_max" || die "cpu.max exceeds 800m: $cpu_max"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"; generated_at_epoch="$(date +%s)"
backup_sha256="$(jq -r '.dump.sha256' "$backup_dir/backup.json")"
backup_generated_at="$(jq -r '.generated_at' "$backup_dir/backup.json")"
success_tmp="${restore_evidence}.success"
jq -n --arg generated_at "$generated_at" --argjson generated_at_epoch "$generated_at_epoch" \
  --arg cluster_uid "$cluster_uid" --arg backup_sha256 "$backup_sha256" \
  --arg backup_generated_at "$backup_generated_at" --arg cpu_max "$cpu_max" \
  --arg retry_of "$retry_prior_evidence" \
  --arg retry_sha256 "$retry_prior_sha256" \
  --arg source_inventory_sha256 "$source_inventory_sha256" --arg target_inventory_sha256 "$target_inventory_sha256" \
  --arg source_schema_sha256 "$source_schema_sha256" --arg target_schema_sha256 "$target_schema_sha256" \
  --arg pod "$pod" --arg target_server_version_num "$target_version" --argjson pvs "$pvs_json" \
  --argjson pre_restore_inventory "$pre_restore_inventory_json" \
  --argjson public_tables "$(snapshot_value "$target_snapshot" public_tables)" \
  --argjson channels "$(snapshot_value "$target_snapshot" channels)" \
  --argjson users "$(snapshot_value "$target_snapshot" users)" \
  --argjson tokens "$(snapshot_value "$target_snapshot" tokens)" \
  '{status:"go",restore_passed:true,generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,cluster_uid:$cluster_uid,retry_of:(if $retry_of == "" then null else $retry_of end),retry_sha256:(if $retry_sha256 == "" then null else $retry_sha256 end),backup:{sha256:$backup_sha256,generated_at:$backup_generated_at,source:"host-postgresql-17"},target:{pod:$pod,node:"atius-srv-1",database:"DBRouterAiAtius",server_version_num:$target_server_version_num,clean_before_restore:true,pre_restore_inventory:$pre_restore_inventory},database_inventory:{format:"phase29-database-inventory-v2",source_sha256:$source_inventory_sha256,target_sha256:$target_inventory_sha256,source_schema_ddl_sha256:$source_schema_sha256,target_schema_ddl_sha256:$target_schema_sha256,source_backup_target_matched:true,matched:true},pvs:$pvs,cpu_max:$cpu_max,invariants:{public_tables:$public_tables,channels:$channels,users:$users,tokens:$tokens},runtime_stage:{redis_applied:false,router_applied:false}}' > "$success_tmp"
chmod 600 "$success_tmp"
publish_restore_success "$success_tmp"
echo "restore evidence: $restore_evidence"
