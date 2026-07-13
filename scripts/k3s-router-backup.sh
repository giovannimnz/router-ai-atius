#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

mode=dry-run
source_host=127.0.0.1
source_port=8745
pgbouncer_host=10.11.1.11
pgbouncer_port=6432
database=DBRouterAiAtius
database_user="admin"
evidence_dir=""
original_args=("$@")
postgres_unit=postgresql@17-main.service
postgres_quota_changed=false
postgres_quota_applied=false
postgres_cpu_max=""
postgres_cpu_max_before=""
postgres_millicores=""
postgres_quota_restored=false
postgres_quota_lock_fd=""
postgres_quota_lock_root="$HOME/.local/state/router-ai-atius/phase29"
postgres_unit_before='{}'
postgres_unit_applied='{}'
postgres_unit_restored='{}'
quota_test_state=""

die() {
  echo "backup failed: $*" >&2
  exit 1
}

cpu_max_value() {
  local cgroup file
  cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"
  file="/sys/fs/cgroup${cgroup}/cpu.max"
  [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"
  cat "$file"
}

quota_lte_millicores() {
  local value="$1" limit="$2" quota period
  read -r quota period <<< "$value"
  [[ "$quota" =~ ^[0-9]+$ ]] && [[ "$period" =~ ^[0-9]+$ ]] && [ "$period" -gt 0 ] || return 1
  [ $((quota * 1000)) -le $((period * limit)) ]
}

quota_millicores() {
  local quota period
  read -r quota period <<< "$1"
  [[ "$quota" =~ ^[0-9]+$ ]] && [[ "$period" =~ ^[0-9]+$ ]] && [ "$period" -gt 0 ] || return 1
  echo $(((quota * 1000 + period - 1) / period))
}

effective_cpu_max_for_pid() {
  local pid="$1" cgroup path value quota period best="" best_q=0 best_p=1
  if [ -n "$quota_test_state" ]; then
    cat "$quota_test_state/effective_cpu_max"
    return
  fi
  cgroup="$(awk -F: '$1 == "0" {print $3}' "/proc/$pid/cgroup")"
  [ -n "$cgroup" ] || die 'PostgreSQL cgroup is unavailable'
  path="/sys/fs/cgroup$cgroup"
  while :; do
    if [ -r "$path/cpu.max" ]; then
      value="$(cat "$path/cpu.max")"; read -r quota period <<< "$value"
      if [[ "$quota" =~ ^[0-9]+$ ]] && [[ "$period" =~ ^[0-9]+$ ]] && [ "$period" -gt 0 ]; then
        if [ -z "$best" ] || [ $((quota * best_p)) -lt $((best_q * period)) ]; then
          best="$value"; best_q="$quota"; best_p="$period"
        fi
      elif [ "$quota" != max ]; then
        die 'PostgreSQL cpu.max is malformed'
      fi
    fi
    [ "$path" = /sys/fs/cgroup ] && break
    path="$(dirname "$path")"
  done
  if [ -n "$best" ]; then echo "$best"; else echo 'max 100000'; fi
}

systemctl_value() {
  local property="$1"
  if [ -n "$quota_test_state" ]; then
    if [ "$property" = MainPID ]; then printf '%s\n' "$$"; else cat "$quota_test_state/$property"; fi
    return
  fi
  systemctl show "$postgres_unit" -p "$property" --value
}

normalized_drop_ins() {
  tr ' ' '\n' <<< "$1" | sed '/^$/d' | LC_ALL=C sort -u | paste -sd' ' -
}

capture_postgres_unit_state() {
  local fragment quota drop_ins drop_ins_json
  fragment="$(systemctl_value FragmentPath)"
  quota="$(systemctl_value CPUQuotaPerSecUSec)"
  drop_ins="$(normalized_drop_ins "$(systemctl_value DropInPaths)")"
  [ -n "$fragment" ] || die 'PostgreSQL FragmentPath is empty'
  [ -n "$quota" ] || die 'PostgreSQL CPUQuotaPerSecUSec is empty'
  drop_ins_json="$(printf '%s\n' "$drop_ins" | tr ' ' '\n' | jq -Rsc 'split("\n") | map(select(length > 0))')"
  jq -cS -n --arg fragment_path "$fragment" --arg cpu_quota_per_sec_usec "$quota" \
    --argjson drop_in_paths "$drop_ins_json" \
    '{fragment_path:$fragment_path,cpu_quota_per_sec_usec:$cpu_quota_per_sec_usec,drop_in_paths:$drop_in_paths}'
}

reject_preexisting_runtime_quota() {
  jq -e '
    all(.drop_in_paths[];
      ((startswith("/run/systemd/") and endswith("/50-CPUQuota.conf")) | not))
  ' <<< "$1" >/dev/null ||
    die 'pre-existing runtime 50-CPUQuota drop-in would be overwritten'
}

set_postgres_runtime_quota() {
  if [ -n "$quota_test_state" ]; then
    printf '%s\n' 'CPUQuota=40%' >> "$quota_test_state/set-property.calls"
    { cat "$quota_test_state/DropInPaths"; printf '%s\n' "/run/systemd/system.control/${postgres_unit}.d/50-CPUQuota.conf"; } |
      tr ' ' '\n' | sed '/^$/d' | LC_ALL=C sort -u | paste -sd' ' - > "$quota_test_state/DropInPaths.next"
    mv "$quota_test_state/DropInPaths.next" "$quota_test_state/DropInPaths"
    printf '400ms\n' > "$quota_test_state/CPUQuotaPerSecUSec"
    printf '40000 100000\n' > "$quota_test_state/effective_cpu_max"
  else
    sudo -n systemctl set-property --runtime "$postgres_unit" CPUQuota=40% >/dev/null
  fi
}

reset_postgres_runtime_quota() {
  if [ -n "$quota_test_state" ]; then
    printf '%s\n' 'CPUQuota=' >> "$quota_test_state/set-property.calls"
    tr ' ' '\n' < "$quota_test_state/DropInPaths" |
      awk -v own="/run/systemd/system.control/${postgres_unit}.d/50-CPUQuota.conf" '$0 != "" && $0 != own' |
      LC_ALL=C sort -u | paste -sd' ' - > "$quota_test_state/DropInPaths.next"
    mv "$quota_test_state/DropInPaths.next" "$quota_test_state/DropInPaths"
    cp "$quota_test_state/CPUQuotaPerSecUSec.before" "$quota_test_state/CPUQuotaPerSecUSec"
    cp "$quota_test_state/effective_cpu_max_before" "$quota_test_state/effective_cpu_max"
  else
    sudo -n systemctl set-property --runtime "$postgres_unit" CPUQuota= >/dev/null
  fi
}

acquire_postgres_quota_lock() {
  local lock
  if [ -L "$HOME/.local" ] || [ -L "$HOME/.local/state" ] || [ -L "$HOME/.local/state/router-ai-atius" ] || [ -L "$postgres_quota_lock_root" ]; then
    die 'PostgreSQL quota lock path contains a symlink'
  fi
  install -d -m 0700 "$HOME/.local/state/router-ai-atius" "$postgres_quota_lock_root"
  [ "$(stat -c %U:%a "$postgres_quota_lock_root")" = "$(id -un):700" ] || die 'PostgreSQL quota lock root owner/mode invalid'
  lock="$postgres_quota_lock_root/postgres-quota.lock"
  [ ! -L "$lock" ] || die 'PostgreSQL quota lock must not be a symlink'
  exec {postgres_quota_lock_fd}> "$lock"
  chmod 600 "$lock"
  flock -n "$postgres_quota_lock_fd" || die 'another backup attempt owns the PostgreSQL quota window'
}

release_postgres_quota_lock() {
  if [ -n "$postgres_quota_lock_fd" ]; then
    flock -u "$postgres_quota_lock_fd" || true
    exec {postgres_quota_lock_fd}>&-
    postgres_quota_lock_fd=""
  fi
}

release_postgres_quota() {
  local pid restored current expected_fragment
  if $postgres_quota_restored || [ "$postgres_unit_before" = '{}' ]; then
    release_postgres_quota_lock
    return 0
  fi
  if $postgres_quota_changed; then
    current="$(capture_postgres_unit_state)" || return 1
    [ "$current" = "$postgres_unit_applied" ] || return 1
    expected_fragment="$(jq -r '.fragment_path' <<< "$postgres_unit_before")"
    [ "$(jq -r '.fragment_path' <<< "$current")" = "$expected_fragment" ] || return 1
    reset_postgres_runtime_quota || return 1
    postgres_unit_restored="$(capture_postgres_unit_state)" || return 1
    [ "$postgres_unit_restored" = "$postgres_unit_before" ] || return 1
    pid="$(systemctl_value MainPID)"
    restored="$(effective_cpu_max_for_pid "$pid")"
    [ "$restored" = "$postgres_cpu_max_before" ] || return 1
    postgres_quota_changed=false
    postgres_quota_restored=true
  else
    postgres_unit_restored="$(capture_postgres_unit_state)" || return 1
    [ "$postgres_unit_restored" = "$postgres_unit_before" ] || return 1
    postgres_quota_restored=true
  fi
  release_postgres_quota_lock
}

configure_postgres_quota() {
  local pid before expected_fragment
  acquire_postgres_quota_lock
  pid="$(systemctl_value MainPID)"
  if ! [[ "$pid" =~ ^[1-9][0-9]*$ ]] || [ ! -r "/proc/$pid/cgroup" ]; then
    die 'PostgreSQL main PID is unavailable'
  fi
  before="$(effective_cpu_max_for_pid "$pid")"
  postgres_cpu_max_before="$before"
  postgres_unit_before="$(capture_postgres_unit_state)"
  reject_preexisting_runtime_quota "$postgres_unit_before"
  expected_fragment="$(jq -r '.fragment_path' <<< "$postgres_unit_before")"
  [ -n "$expected_fragment" ] || die 'PostgreSQL FragmentPath is unavailable'
  if ! quota_lte_millicores "$before" 400; then
    postgres_quota_changed=true
    if [ -n "$quota_test_state" ]; then
      cp "$quota_test_state/CPUQuotaPerSecUSec" "$quota_test_state/CPUQuotaPerSecUSec.before"
    fi
    set_postgres_runtime_quota
    postgres_quota_applied=true
    postgres_unit_applied="$(capture_postgres_unit_state)"
    jq -e --arg fragment "$expected_fragment" --argjson before "$postgres_unit_before" '
      .fragment_path == $fragment and .cpu_quota_per_sec_usec != $before.cpu_quota_per_sec_usec and
      (($before.drop_in_paths - .drop_in_paths) | length) == 0
    ' <<< "$postgres_unit_applied" >/dev/null || die 'runtime quota changed FragmentPath or hid an existing drop-in'
  else
    postgres_unit_applied="$postgres_unit_before"
  fi
  postgres_cpu_max="$(effective_cpu_max_for_pid "$pid")"
  quota_lte_millicores "$postgres_cpu_max" 400 || die "PostgreSQL backend exceeds 400m: $postgres_cpu_max"
  postgres_millicores="$(quota_millicores "$postgres_cpu_max")"
}

pg_dump_version_from_text() {
  sed -E 's/^pg_dump \(PostgreSQL\) ([^ )]+).*$/\1/' <<< "$1"
}

require_profile() {
  local cpu_max
  cpu_max="$(cpu_max_value)"
  if quota_lte_millicores "$cpu_max" 400; then
    return
  fi
  [ "${PHASE29_PROFILED:-0}" != 1 ] || die "backup client cpu.max exceeds 400m: $cpu_max"
  exec env PODMAN_ADMIN_PROFILE_CPU_QUOTA=40% "$repo_root/scripts/podman-admin.sh" profile-run -- \
    env PHASE29_PROFILED=1 "$repo_root/scripts/k3s-router-backup.sh" "${original_args[@]}"
}

validate_endpoint_contract() {
  [ "$source_host" = 127.0.0.1 ] || die 'canonical PostgreSQL host must be 127.0.0.1'
  [ "$source_port" = 8745 ] || die 'canonical PostgreSQL port must be 8745'
  [ "$pgbouncer_host" = 10.11.1.11 ] || die 'canonical PgBouncer host must be 10.11.1.11'
  [ "$pgbouncer_port" = 6432 ] || die 'canonical PgBouncer port must be 6432'
  [ "$database" = DBRouterAiAtius ] || die 'unexpected database identity'
  [ "$database_user" = admin ] || die 'unexpected database user'
}

query_snapshot() {
  local host="$1" port="$2" output="$3"
  PGPASSFILE="$PGPASSFILE" psql -X --set ON_ERROR_STOP=on \
    --host "$host" --port "$port" --username "$database_user" --dbname "$database" \
    --tuples-only --no-align --field-separator=$'\t' --output "$output" <<'SQL'
SELECT 'database', current_database()
UNION ALL SELECT 'user', current_user
UNION ALL SELECT 'server_version_num', current_setting('server_version_num')
UNION ALL SELECT 'server_version', current_setting('server_version')
UNION ALL SELECT 'server_addr', host(inet_server_addr())
UNION ALL SELECT 'table_count', count(*)::text FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
UNION ALL SELECT 'channels', count(*)::text FROM public.channels
UNION ALL SELECT 'users', count(*)::text FROM public.users
UNION ALL SELECT 'tokens', count(*)::text FROM public.tokens
UNION ALL SELECT 'subscriptions', count(*)::text FROM pg_subscription;
SQL
}

query_privileged_identity() {
  local output="$1"
  # Redirection intentionally stays in the caller-owned tmpfs directory.
  # shellcheck disable=SC2024
  sudo -n -u postgres psql -X --set ON_ERROR_STOP=on --port "$source_port" --dbname "$database" \
    --tuples-only --no-align --field-separator=$'\t' > "$output" <<'SQL'
SELECT 'database', current_database()
UNION ALL SELECT 'server_version_num', current_setting('server_version_num')
UNION ALL SELECT 'data_directory', current_setting('data_directory');
SQL
}

snapshot_value() {
  local file="$1" key="$2"
  awk -F '\t' -v key="$key" '$1 == key {print $2}' "$file"
}

validate_snapshots() {
  local direct="$1" pooled="$2" privileged="$3" key value other
  for key in database user server_version_num server_version server_addr table_count channels users tokens subscriptions; do
    value="$(snapshot_value "$direct" "$key")"
    other="$(snapshot_value "$pooled" "$key")"
    [ -n "$value" ] || die "direct source omitted $key"
    [ "$value" = "$other" ] || die "direct/PgBouncer mismatch for $key"
  done
  [ "$(snapshot_value "$direct" database)" = DBRouterAiAtius ] || die 'source database mismatch'
  [ "$(snapshot_value "$direct" user)" = admin ] || die 'source user mismatch'
  [[ "$(snapshot_value "$direct" server_version_num)" =~ ^17[0-9]{4}$ ]] || die 'source is not PostgreSQL 17'
  [ "$(snapshot_value "$privileged" database)" = "$(snapshot_value "$direct" database)" ] || die 'privileged source database mismatch'
  [ "$(snapshot_value "$privileged" server_version_num)" = "$(snapshot_value "$direct" server_version_num)" ] || die 'privileged source version mismatch'
  [ "$(snapshot_value "$privileged" data_directory)" = /var/lib/postgresql/17/main ] || die 'source data_directory is not the canonical PostgreSQL 17 cluster'
  [ "$(snapshot_value "$direct" server_addr)" = 127.0.0.1 ] || die 'direct source connection did not terminate on 127.0.0.1'
  [ "$(snapshot_value "$direct" table_count)" -ge 34 ] || die 'source has fewer than 34 public tables'
  for key in channels users tokens; do
    [ "$(snapshot_value "$direct" "$key")" -gt 0 ] || die "$key invariant is empty"
  done
  [ "$(snapshot_value "$direct" subscriptions)" -eq 0 ] || die 'source subscriptions are forbidden because conninfo is not backed up'
}

validate_listener_unit() {
  local main_pid listener_pids
  systemctl is-active --quiet "$postgres_unit" || die 'postgresql@17-main is not active'
  main_pid="$(systemctl show "$postgres_unit" -p MainPID --value)"
  [[ "$main_pid" =~ ^[1-9][0-9]*$ ]] || die 'postgresql@17-main MainPID is invalid'
  listener_pids="$(sudo -n ss -H -ltnp | awk '$4 ~ /:8745$/ {print}' | grep -Eo 'pid=[0-9]+' | cut -d= -f2 | sort -u | paste -sd, -)"
  [ "$listener_pids" = "$main_pid" ] || die '127.0.0.1:8745 listener is not owned exclusively by postgresql@17-main'
}

validate_dump() {
  local dump="$1" checksum_file="$2" size
  if [ ! -f "$dump" ] || [ -L "$dump" ]; then die 'dump missing or symlinked'; fi
  size="$(stat -c %s "$dump")"
  [ "$size" -gt 643 ] || die 'dump is empty or matches the obsolete 643-byte artifact'
  grep -Fq 'PostgreSQL database dump' "$dump" || die 'dump header missing'
  grep -Eq '^CREATE TABLE public\.channels ' "$dump" || die 'channels table definition missing'
  grep -Eq '^CREATE TABLE public\.users ' "$dump" || die 'users table definition missing'
  grep -Eq '^CREATE TABLE public\.tokens ' "$dump" || die 'tokens table definition missing'
  grep -Fq 'PostgreSQL database dump complete' "$dump" || die 'dump completion marker missing'
  if [ ! -f "$checksum_file" ] || [ -L "$checksum_file" ]; then die 'checksum missing or symlinked'; fi
  (cd "$(dirname "$dump")" && sha256sum --check --status "$(basename "$checksum_file")") ||
    die 'dump checksum validation failed'
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

write_database_state_query() {
  local output="$1"
  cat > "$output" <<'SQL'
WITH selected AS (
  SELECT d.*, t.spcname AS tablespace_name
  FROM pg_database d JOIN pg_tablespace t ON t.oid = d.dattablespace
  WHERE d.datname = current_database()
), acl AS (
  SELECT COALESCE(jsonb_agg(value ORDER BY value), '[]'::jsonb) AS values
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
  'properties', to_jsonb(d) - ARRAY['oid','datdba','dattablespace','datfrozenxid','datminmxid','datacl','tablespace_name'],
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

create_source_database_inventory() {
  local output="$1" schema_output="$2" schema_dump="$tmp/source-schema.sql" database_state="$tmp/source-database-state.json"
  local role_settings="$tmp/source-role-settings.tsv" large_objects="$tmp/source-large-objects.tsv"
  local database_query="$tmp/database-state.sql" role_settings_query="$tmp/database-role-settings.sql"
  PGPASSFILE="$PGPASSFILE" pg_dump \
    --host "$source_host" --port "$source_port" --username "$database_user" --dbname "$database" \
    --schema-only --no-subscriptions --quote-all-identifiers \
    --restrict-key=phase29databaseinventory --file "$schema_dump"
  write_database_state_query "$database_query"
  write_database_role_settings_query "$role_settings_query"
  PGPASSFILE="$PGPASSFILE" psql -X --set ON_ERROR_STOP=on \
    --host "$source_host" --port "$source_port" --username "$database_user" --dbname "$database" \
    --tuples-only --no-align --output "$database_state" --file "$database_query"
  PGPASSFILE="$PGPASSFILE" psql -X --set ON_ERROR_STOP=on \
    --host "$source_host" --port "$source_port" --username "$database_user" --dbname "$database" \
    --tuples-only --no-align --field-separator=$'\t' --output "$role_settings" --file "$role_settings_query"
  PGPASSFILE="$PGPASSFILE" psql -X --set ON_ERROR_STOP=on \
    --host "$source_host" --port "$source_port" --username "$database_user" --dbname "$database" \
    --tuples-only --no-align --output "$large_objects" --command='SELECT oid::text FROM pg_largeobject_metadata ORDER BY oid'
  normalize_database_inventory "$schema_dump" "$database_state" "$role_settings" "$large_objects" "$output" "$schema_output"
  validate_database_inventory "$output" "$schema_output"
}

reset_quota_test_stub() {
  local root="$1"
  release_postgres_quota_lock
  rm -rf "$root"
  mkdir -p "$root/state"
  quota_test_state="$root/state"
  postgres_quota_lock_root="$root/lock"
  printf '/usr/lib/systemd/system/postgresql@.service\n' > "$quota_test_state/FragmentPath"
  : > "$quota_test_state/DropInPaths"
  printf 'infinity\n' > "$quota_test_state/CPUQuotaPerSecUSec"
  printf 'max 100000\n' > "$quota_test_state/effective_cpu_max"
  cp "$quota_test_state/effective_cpu_max" "$quota_test_state/effective_cpu_max_before"
  : > "$quota_test_state/set-property.calls"
  postgres_quota_changed=false
  postgres_quota_applied=false
  postgres_quota_restored=false
  postgres_unit_before='{}'
  postgres_unit_applied='{}'
  postgres_unit_restored='{}'
}

quota_lock_self_test() {
  local ready="${PHASE29_QUOTA_TEST_DIR:?PHASE29_QUOTA_TEST_DIR required}/ready"
  acquire_postgres_quota_lock
  printf 'ready\n' > "$ready"
  sleep 60
}

self_test() {
  local tmp direct pooled privileged dump checksum schema_dump schema_ddl database_state role_settings large_objects inventory quota_root unrelated
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN
  direct="$tmp/direct.tsv"; pooled="$tmp/pooled.tsv"
  cat > "$direct" <<'EOF'
database	DBRouterAiAtius
user	admin
server_version_num	170010
server_version	17.10
data_directory	/var/lib/postgresql/17/main
server_addr	127.0.0.1
table_count	34
channels	4
users	2
tokens	3
subscriptions	0
EOF
  cp "$direct" "$pooled"
  privileged="$tmp/privileged.tsv"
  printf 'database\tDBRouterAiAtius\nserver_version_num\t170010\ndata_directory\t/var/lib/postgresql/17/main\n' > "$privileged"
  validate_endpoint_contract
  validate_snapshots "$direct" "$pooled" "$privileged"
  sed -i 's/server_version_num\t170010/server_version_num\t150010/' "$pooled"
  if (validate_snapshots "$direct" "$pooled" "$privileged") 2>/dev/null; then die 'version mismatch was accepted'; fi
  cp "$direct" "$pooled"
  sed -i 's/table_count\t34/table_count\t33/' "$direct"
  if (validate_snapshots "$direct" "$pooled" "$privileged") 2>/dev/null; then die 'undersized schema was accepted'; fi
  cp "$pooled" "$direct"
  dump="$tmp/DBRouterAiAtius.sql"; checksum="$tmp/DBRouterAiAtius.sql.sha256"
  {
    echo '-- PostgreSQL database dump'
    echo 'CREATE TABLE public.channels (id bigint);'
    echo 'CREATE TABLE public.users (id bigint);'
    echo 'CREATE TABLE public.tokens (id bigint);'
    printf '%0800d\n' 0
    echo '-- PostgreSQL database dump complete'
  } > "$dump"
  (cd "$tmp" && sha256sum "$(basename "$dump")" > "$(basename "$checksum")")
  validate_dump "$dump" "$checksum"
  printf 'tamper\n' >> "$dump"
  if (validate_dump "$dump" "$checksum") 2>/dev/null; then die 'tampered dump was accepted'; fi
  quota_lte_millicores '40000 100000' 400 || die '400m split quota rejected'
  if quota_lte_millicores '40001 100000' 400; then die 'quota above 400m accepted'; fi
  if quota_lte_millicores 'max 100000' 400; then die 'unbounded quota accepted'; fi
  [ $(( $(quota_millicores '40000 100000') + $(quota_millicores '40000 100000') )) -le 800 ] || die 'aggregate 800m rejected'
  [ "$(pg_dump_version_from_text 'pg_dump (PostgreSQL) 17.10 (Ubuntu 17.10-1.pgdg22.04+1)')" = 17.10 ] || die 'pg_dump version normalization failed'
  schema_dump="$tmp/schema.sql"; schema_ddl="$tmp/schema-normalized.sql"; database_state="$tmp/database-state.json"
  role_settings="$tmp/role-settings.tsv"; large_objects="$tmp/large-objects.tsv"; inventory="$tmp/inventory.json"
  printf '%s\n' '-- PostgreSQL database dump' '-- Name: channels; Type: TABLE' 'CREATE TABLE "public"."channels" ();' \
    'ALTER TABLE "public"."channels" OWNER TO "admin";' 'COMMENT ON TABLE "public"."channels" IS '\''catalog'\'';' \
    'GRANT SELECT ON TABLE "public"."channels" TO "admin";' '-- PostgreSQL database dump complete' > "$schema_dump"
  jq -cn '{name:"DBRouterAiAtius",owner:"admin",tablespace:"pg_default",properties:{datname:"DBRouterAiAtius",datallowconn:true},acl:[],comment:null,security_labels:[]}' > "$database_state"
  printf 'admin\tstatement_timeout\t313030306d73\n' > "$role_settings"
  printf '16384\n16385\n' > "$large_objects"
  normalize_database_inventory "$schema_dump" "$database_state" "$role_settings" "$large_objects" "$inventory" "$schema_ddl"
  validate_database_inventory "$inventory" "$schema_ddl"
  grep -Fq 'ALTER TABLE "public"."channels" OWNER TO "admin"' "$schema_ddl" || die 'full schema DDL lost owner state'
  grep -Fq 'GRANT SELECT ON TABLE "public"."channels" TO "admin"' "$schema_ddl" || die 'full schema DDL lost ACL state'
  jq -e '.schema_version == 2 and .database.owner == "admin" and (.pg_db_role_setting | length) == 1' "$inventory" >/dev/null ||
    die 'database-wide inventory v2 omitted database or role-setting state'
  cp "$inventory" "$tmp/inventory-before-change"
  printf 'CREATE VIEW "public"."changed" AS SELECT 1;\n' >> "$schema_dump"
  normalize_database_inventory "$schema_dump" "$database_state" "$role_settings" "$large_objects" "$inventory" "$schema_ddl"
  if cmp -s "$tmp/inventory-before-change" "$inventory"; then die 'schema object change did not alter normalized inventory'; fi

  quota_root="$tmp/quota-normal"
  reset_quota_test_stub "$quota_root"
  configure_postgres_quota
  release_postgres_quota
  [ "$postgres_unit_restored" = "$postgres_unit_before" ] || die 'stubbed unit state was not restored exactly'
  [ "$(sed -n '1p' "$quota_test_state/set-property.calls")" = 'CPUQuota=40%' ] || die 'runtime quota was not applied through set-property'
  [ "$(sed -n '2p' "$quota_test_state/set-property.calls")" = 'CPUQuota=' ] || die 'runtime quota was not reset through empty set-property'

  reset_quota_test_stub "$tmp/quota-persistent-override"
  unrelated=/etc/systemd/system/postgresql@17-main.service.d/90-unrelated.conf
  printf '%s\n' "$unrelated" > "$quota_test_state/DropInPaths"
  configure_postgres_quota
  release_postgres_quota
  [ "$(cat "$quota_test_state/DropInPaths")" = "$unrelated" ] || die 'persistent override was not preserved'

  reset_quota_test_stub "$tmp/quota-preexisting-runtime"
  unrelated="/run/systemd/system.control/${postgres_unit}.d/50-CPUQuota.conf"
  printf '%s\n' "$unrelated" > "$quota_test_state/DropInPaths"
  if (configure_postgres_quota) 2>/dev/null; then die 'pre-existing runtime quota was accepted'; fi
  [ ! -s "$quota_test_state/set-property.calls" ] || die 'pre-existing runtime quota was overwritten before rejection'
  [ "$(cat "$quota_test_state/DropInPaths")" = "$unrelated" ] || die 'pre-existing runtime quota changed during rejection'
  release_postgres_quota_lock

  reset_quota_test_stub "$tmp/quota-runtime-race"
  configure_postgres_quota
  unrelated=/run/systemd/system/postgresql@17-main.service.d/90-unrelated.conf
  printf '%s %s\n' "/run/systemd/system.control/${postgres_unit}.d/50-CPUQuota.conf" "$unrelated" > "$quota_test_state/DropInPaths"
  if release_postgres_quota 2>/dev/null; then die 'concurrent unrelated drop-in was accepted'; fi
  grep -Fq "$unrelated" "$quota_test_state/DropInPaths" || die 'quota reset discarded a concurrent unrelated drop-in'
  release_postgres_quota_lock

  reset_quota_test_stub "$tmp/quota-fragment-race"
  configure_postgres_quota
  printf '/etc/systemd/system/postgresql@17-main.service\n' > "$quota_test_state/FragmentPath"
  if release_postgres_quota 2>/dev/null; then die 'concurrent FragmentPath change was accepted'; fi
  [ "$(cat "$quota_test_state/FragmentPath")" = /etc/systemd/system/postgresql@17-main.service ] || die 'quota reset changed FragmentPath'
  release_postgres_quota_lock
  echo 'backup self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --source-host) source_host="${2:?}"; shift ;;
    --source-port) source_port="${2:?}"; shift ;;
    --pgbouncer-host) pgbouncer_host="${2:?}"; shift ;;
    --pgbouncer-port) pgbouncer_port="${2:?}"; shift ;;
    --database) database="${2:?}"; shift ;;
    --database-user) database_user="${2:?}"; shift ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --self-test-quota-lock) quota_lock_self_test; exit 0 ;;
    --self-test) self_test; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

validate_endpoint_contract
[ "$mode" = live ] || {
  echo 'backup dry-run: canonical host PostgreSQL 17 and PgBouncer checks planned; no command executed'
  exit 0
}

[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[ "${PHASE29_BACKUP_CONFIRM:-}" = BACKUP_CANONICAL_HOST_POSTGRES17 ] ||
  die 'missing exact canonical-host backup confirmation'
require_profile

for command in flock jq pg_dump psql sha256sum ss systemctl; do
  command -v "$command" >/dev/null || die "required command missing: $command"
done
pg_dump_version="$(pg_dump_version_from_text "$(pg_dump --version)")"
[[ "$pg_dump_version" =~ ^17\. ]] || die "pg_dump client must be version 17"

[ -n "$evidence_dir" ] || die '--evidence-dir required'
if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then die 'evidence directory missing or symlinked'; fi
evidence_dir="$(realpath -e "$evidence_dir")"
[ "$(stat -c %U:%a "$evidence_dir")" = "$(id -un):700" ] || die 'evidence directory must be owned by the caller with mode 700'

ts="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="$evidence_dir/backup-$ts"
umask 077
mkdir "$backup_dir"
mkdir "$backup_dir/db"
tmp="$(mktemp -d /dev/shm/phase29-backup.XXXXXX)"
success=false
cleanup() {
  local rc=$? release_rc=0
  set +e
  release_postgres_quota || release_rc=$?
  rm -rf "$tmp"
  if ! $success; then rm -rf "$backup_dir"; fi
  if [ "$rc" -eq 0 ] && [ "$release_rc" -ne 0 ]; then rc="$release_rc"; fi
  return "$rc"
}
on_backup_signal() { local signal="$1"; trap - "$signal"; kill -s "$signal" "$$"; }
trap cleanup EXIT
trap 'on_backup_signal INT' INT
trap 'on_backup_signal TERM' TERM

if [ -n "${PHASE29_PGPASSFILE:-}" ]; then
  if [ ! -f "$PHASE29_PGPASSFILE" ] || [ -L "$PHASE29_PGPASSFILE" ]; then die 'PHASE29_PGPASSFILE must be a regular non-symlink file'; fi
  PGPASSFILE="$(realpath -e "$PHASE29_PGPASSFILE")"
  [ "$(stat -c %a "$PGPASSFILE")" = 600 ] || die 'PHASE29_PGPASSFILE must have mode 600'
else
  helper="$HOME/.local/bin/atius-vault-env"
  [ -x "$helper" ] || die 'Vault helper unavailable and PHASE29_PGPASSFILE not set'
  set +x
  # shellcheck disable=SC1090
  source <("$helper" router-ai-atius)
  [ -n "${POSTGRES_PASSWORD:-}" ] || die 'Vault profile did not provide the database password'
  PGPASSFILE="$tmp/pgpass"
  printf '%s:%s:%s:%s:%s\n' "$source_host" "$source_port" "$database" "$database_user" "$POSTGRES_PASSWORD" > "$PGPASSFILE"
  printf '%s:%s:%s:%s:%s\n' "$pgbouncer_host" "$pgbouncer_port" "$database" "$database_user" "$POSTGRES_PASSWORD" >> "$PGPASSFILE"
  chmod 600 "$PGPASSFILE"
  unset POSTGRES_PASSWORD
fi
export PGPASSFILE
validate_listener_unit
configure_postgres_quota

direct="$tmp/direct.tsv"; pooled="$tmp/pooled.tsv"; privileged="$tmp/privileged.tsv"
query_snapshot "$source_host" "$source_port" "$direct"
query_snapshot "$pgbouncer_host" "$pgbouncer_port" "$pooled"
query_privileged_identity "$privileged"
validate_snapshots "$direct" "$pooled" "$privileged"

dump="$backup_dir/db/DBRouterAiAtius.sql"
dump_error="$tmp/pg_dump.stderr"
if ! PGPASSFILE="$PGPASSFILE" pg_dump \
  --host "$source_host" --port "$source_port" --username "$database_user" --dbname "$database" \
  --format=plain --no-subscriptions --file "$dump" 2>"$dump_error"; then
  die 'pg_dump failed (details intentionally suppressed)'
fi
checksum_file="$backup_dir/db/DBRouterAiAtius.sql.sha256"
(cd "$(dirname "$dump")" && sha256sum "$(basename "$dump")" > "$(basename "$checksum_file")")
validate_dump "$dump" "$checksum_file"
inventory="$backup_dir/db/DBRouterAiAtius.inventory"
schema_ddl="$backup_dir/db/DBRouterAiAtius.schema.sql"
create_source_database_inventory "$inventory" "$schema_ddl"
inventory_sha256="$(sha256sum "$inventory" | awk '{print $1}')"
inventory_size="$(stat -c %s "$inventory")"
schema_ddl_sha256="$(sha256sum "$schema_ddl" | awk '{print $1}')"

dump_sha256="$(awk '{print $1}' "$checksum_file")"
dump_size="$(stat -c %s "$dump")"
cpu_max="$(cpu_max_value)"
quota_lte_millicores "$cpu_max" 400 || die "backup client exceeds 400m: $cpu_max"
client_millicores="$(quota_millicores "$cpu_max")"
aggregate_millicores=$((client_millicores + postgres_millicores))
[ "$aggregate_millicores" -le 800 ] || die "aggregate backup CPU exceeds 800m: $aggregate_millicores"
release_postgres_quota
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
generated_at_epoch="$(date +%s)"
jq -n \
  --arg generated_at "$generated_at" --argjson generated_at_epoch "$generated_at_epoch" \
  --arg source_host "$source_host" --argjson source_port "$source_port" \
  --arg pgbouncer_host "$pgbouncer_host" --argjson pgbouncer_port "$pgbouncer_port" \
  --arg database "$(snapshot_value "$direct" database)" --arg database_user "$(snapshot_value "$direct" user)" \
  --arg server_version "$(snapshot_value "$direct" server_version)" \
  --arg server_version_num "$(snapshot_value "$direct" server_version_num)" \
  --arg data_directory "$(snapshot_value "$privileged" data_directory)" --arg postgres_unit "$postgres_unit" \
  --arg server_addr "$(snapshot_value "$direct" server_addr)" \
  --arg pg_dump_version "$pg_dump_version" --arg cpu_max "$cpu_max" \
  --arg postgres_cpu_max "$postgres_cpu_max" --argjson client_millicores "$client_millicores" \
  --argjson postgres_millicores "$postgres_millicores" --argjson aggregate_millicores "$aggregate_millicores" \
  --argjson postgres_quota_temporarily_applied "$postgres_quota_applied" --argjson postgres_quota_restored "$postgres_quota_restored" \
  --arg dump_sha256 "$dump_sha256" --argjson dump_size_bytes "$dump_size" \
  --arg inventory_sha256 "$inventory_sha256" --argjson inventory_size_bytes "$inventory_size" \
  --arg schema_ddl_sha256 "$schema_ddl_sha256" \
  --argjson postgres_unit_before "$postgres_unit_before" --argjson postgres_unit_applied "$postgres_unit_applied" \
  --argjson postgres_unit_restored "$postgres_unit_restored" \
  --argjson table_count "$(snapshot_value "$direct" table_count)" \
  --argjson channels "$(snapshot_value "$direct" channels)" \
  --argjson users "$(snapshot_value "$direct" users)" \
  --argjson tokens "$(snapshot_value "$direct" tokens)" \
  --argjson subscriptions "$(snapshot_value "$direct" subscriptions)" \
  '{status:"go",generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,source:{kind:"host-postgresql",host:$source_host,port:$source_port,server_addr:$server_addr,database:$database,user:$database_user,server_version:$server_version,server_version_num:$server_version_num,data_directory:$data_directory,systemd_unit:$postgres_unit,backend_unit_matched:true},pgbouncer_crosscheck:{host:$pgbouncer_host,port:$pgbouncer_port,matched:true},pg_dump_version:$pg_dump_version,cpu_max:$cpu_max,cpu:{client_cpu_max:$cpu_max,postgres_cpu_max:$postgres_cpu_max,client_millicores:$client_millicores,postgres_millicores:$postgres_millicores,aggregate_millicores:$aggregate_millicores,postgres_quota_temporarily_applied:$postgres_quota_temporarily_applied,postgres_quota_restored:$postgres_quota_restored,unit_state:{before:$postgres_unit_before,applied:$postgres_unit_applied,restored:$postgres_unit_restored}},dump:{path:"db/DBRouterAiAtius.sql",size_bytes:$dump_size_bytes,sha256:$dump_sha256,structurally_valid:true,subscriptions_included:false,owners_included:true,acl_included:true},database_inventory:{format:"phase29-database-inventory-v2",path:"db/DBRouterAiAtius.inventory",schema_ddl_path:"db/DBRouterAiAtius.schema.sql",schema_ddl_sha256:$schema_ddl_sha256,size_bytes:$inventory_size_bytes,sha256:$inventory_sha256,source_backup_matched:true,target_equality_required:true,sanitized:true},invariants:{public_tables:$table_count,channels:$channels,users:$users,tokens:$tokens,subscriptions:$subscriptions}}' \
  > "$backup_dir/backup.json"
chmod 600 "$backup_dir/backup.json" "$dump" "$checksum_file" "$inventory" "$schema_ddl"
success=true
echo "K3S_BACKUP_DIR=$backup_dir"
