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

release_postgres_quota() {
  local pid restored
  if $postgres_quota_changed; then
    sudo -n systemctl set-property --runtime "$postgres_unit" CPUQuota=infinity >/dev/null
    pid="$(systemctl show "$postgres_unit" -p MainPID --value)"
    restored="$(effective_cpu_max_for_pid "$pid")"
    [ "$restored" = "$postgres_cpu_max_before" ] || return 1
    postgres_quota_changed=false
    postgres_quota_restored=true
  else
    postgres_quota_restored=true
  fi
}

configure_postgres_quota() {
  local pid before unit_quota
  pid="$(systemctl show "$postgres_unit" -p MainPID --value)"
  if ! [[ "$pid" =~ ^[1-9][0-9]*$ ]] || [ ! -r "/proc/$pid/cgroup" ]; then
    die 'PostgreSQL main PID is unavailable'
  fi
  before="$(effective_cpu_max_for_pid "$pid")"
  postgres_cpu_max_before="$before"
  if ! quota_lte_millicores "$before" 400; then
    unit_quota="$(systemctl show "$postgres_unit" -p CPUQuotaPerSecUSec --value)"
    [ "$unit_quota" = infinity ] || die 'PostgreSQL has an unknown noncompliant CPU quota; refusing temporary override'
    postgres_quota_changed=true
    sudo -n systemctl set-property --runtime "$postgres_unit" CPUQuota=40% CPUQuotaPeriodSec=100ms >/dev/null
    postgres_quota_applied=true
  fi
  postgres_cpu_max="$(effective_cpu_max_for_pid "$pid")"
  quota_lte_millicores "$postgres_cpu_max" 400 || die "PostgreSQL backend exceeds 400m: $postgres_cpu_max"
  postgres_millicores="$(quota_millicores "$postgres_cpu_max")"
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
UNION ALL SELECT 'data_directory', current_setting('data_directory')
UNION ALL SELECT 'server_addr', inet_server_addr()::text
UNION ALL SELECT 'table_count', count(*)::text FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
UNION ALL SELECT 'channels', count(*)::text FROM public.channels
UNION ALL SELECT 'users', count(*)::text FROM public.users
UNION ALL SELECT 'tokens', count(*)::text FROM public.tokens;
SQL
}

snapshot_value() {
  local file="$1" key="$2"
  awk -F '\t' -v key="$key" '$1 == key {print $2}' "$file"
}

validate_snapshots() {
  local direct="$1" pooled="$2" key value other
  for key in database user server_version_num server_version data_directory server_addr table_count channels users tokens; do
    value="$(snapshot_value "$direct" "$key")"
    other="$(snapshot_value "$pooled" "$key")"
    [ -n "$value" ] || die "direct source omitted $key"
    [ "$value" = "$other" ] || die "direct/PgBouncer mismatch for $key"
  done
  [ "$(snapshot_value "$direct" database)" = DBRouterAiAtius ] || die 'source database mismatch'
  [ "$(snapshot_value "$direct" user)" = admin ] || die 'source user mismatch'
  [[ "$(snapshot_value "$direct" server_version_num)" =~ ^17[0-9]{4}$ ]] || die 'source is not PostgreSQL 17'
  [ "$(snapshot_value "$direct" data_directory)" = /var/lib/postgresql/17/main ] || die 'source data_directory is not the canonical PostgreSQL 17 cluster'
  [ "$(snapshot_value "$direct" server_addr)" = 127.0.0.1 ] || die 'direct source connection did not terminate on 127.0.0.1'
  [ "$(snapshot_value "$direct" table_count)" -ge 34 ] || die 'source has fewer than 34 public tables'
  for key in channels users tokens; do
    [ "$(snapshot_value "$direct" "$key")" -gt 0 ] || die "$key invariant is empty"
  done
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

self_test() {
  local tmp direct pooled dump checksum
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
EOF
  cp "$direct" "$pooled"
  validate_endpoint_contract
  validate_snapshots "$direct" "$pooled"
  sed -i 's/server_version_num\t170010/server_version_num\t150010/' "$pooled"
  if (validate_snapshots "$direct" "$pooled") 2>/dev/null; then die 'version mismatch was accepted'; fi
  cp "$direct" "$pooled"
  sed -i 's/table_count\t34/table_count\t33/' "$direct"
  if (validate_snapshots "$direct" "$pooled") 2>/dev/null; then die 'undersized schema was accepted'; fi
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

for command in jq pg_dump psql sha256sum ss systemctl; do
  command -v "$command" >/dev/null || die "required command missing: $command"
done
pg_dump_version="$(pg_dump --version | awk '{print $NF}')"
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
  local rc=$?
  set +e
  release_postgres_quota
  rm -rf "$tmp"
  if ! $success; then rm -rf "$backup_dir"; fi
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

direct="$tmp/direct.tsv"; pooled="$tmp/pooled.tsv"
query_snapshot "$source_host" "$source_port" "$direct"
query_snapshot "$pgbouncer_host" "$pgbouncer_port" "$pooled"
validate_snapshots "$direct" "$pooled"

dump="$backup_dir/db/DBRouterAiAtius.sql"
dump_error="$tmp/pg_dump.stderr"
if ! PGPASSFILE="$PGPASSFILE" pg_dump \
  --host "$source_host" --port "$source_port" --username "$database_user" --dbname "$database" \
  --format=plain --no-owner --no-privileges --file "$dump" 2>"$dump_error"; then
  die 'pg_dump failed (details intentionally suppressed)'
fi
checksum_file="$backup_dir/db/DBRouterAiAtius.sql.sha256"
(cd "$(dirname "$dump")" && sha256sum "$(basename "$dump")" > "$(basename "$checksum_file")")
validate_dump "$dump" "$checksum_file"

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
  --arg data_directory "$(snapshot_value "$direct" data_directory)" --arg postgres_unit "$postgres_unit" \
  --arg server_addr "$(snapshot_value "$direct" server_addr)" \
  --arg pg_dump_version "$pg_dump_version" --arg cpu_max "$cpu_max" \
  --arg postgres_cpu_max "$postgres_cpu_max" --argjson client_millicores "$client_millicores" \
  --argjson postgres_millicores "$postgres_millicores" --argjson aggregate_millicores "$aggregate_millicores" \
  --argjson postgres_quota_temporarily_applied "$postgres_quota_applied" --argjson postgres_quota_restored "$postgres_quota_restored" \
  --arg dump_sha256 "$dump_sha256" --argjson dump_size_bytes "$dump_size" \
  --argjson table_count "$(snapshot_value "$direct" table_count)" \
  --argjson channels "$(snapshot_value "$direct" channels)" \
  --argjson users "$(snapshot_value "$direct" users)" \
  --argjson tokens "$(snapshot_value "$direct" tokens)" \
  '{status:"go",generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,source:{kind:"host-postgresql",host:$source_host,port:$source_port,server_addr:$server_addr,database:$database,user:$database_user,server_version:$server_version,server_version_num:$server_version_num,data_directory:$data_directory,systemd_unit:$postgres_unit,backend_unit_matched:true},pgbouncer_crosscheck:{host:$pgbouncer_host,port:$pgbouncer_port,matched:true},pg_dump_version:$pg_dump_version,cpu_max:$cpu_max,cpu:{client_cpu_max:$cpu_max,postgres_cpu_max:$postgres_cpu_max,client_millicores:$client_millicores,postgres_millicores:$postgres_millicores,aggregate_millicores:$aggregate_millicores,postgres_quota_temporarily_applied:$postgres_quota_temporarily_applied,postgres_quota_restored:$postgres_quota_restored},dump:{path:"db/DBRouterAiAtius.sql",size_bytes:$dump_size_bytes,sha256:$dump_sha256,structurally_valid:true},invariants:{public_tables:$table_count,channels:$channels,users:$users,tokens:$tokens}}' \
  > "$backup_dir/backup.json"
chmod 600 "$backup_dir/backup.json" "$dump" "$checksum_file"
success=true
echo "K3S_BACKUP_DIR=$backup_dir"
