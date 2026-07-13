#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

mode=dry-run
evidence_root="${PHASE30_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase30}"
evidence_dir=""
phase29_dir="${PHASE29_EVIDENCE_DIR:-}"
prepare=false
output=""
tmp_root=""

apache_config=/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf
pgbouncer_config=/etc/pgbouncer/pgbouncer.ini
phase29_root="${PHASE29_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase29}"
override_literal=I_ACCEPT_PHASE29_RISK
database=DBRouterAiAtius
database_user="admin"
host_pg_host=127.0.0.1
host_pg_port=8745
host_pg_unit=postgresql@17-main
host_pg_data_dir=/var/lib/postgresql/17/main
canonical_dbrouter_tables=34
container_empty_tables=0
namespace=router-ai-atius
label_key=atius.com.br/router-ai-atius-node
label_value=true

die() {
  echo "cutover preflight failed: $*" >&2
  exit 1
}

kube() {
  sudo -n k3s kubectl "$@"
}

cpu_max_value() {
  local cgroup file
  cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"
  file="/sys/fs/cgroup${cgroup}/cpu.max"
  [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"
  cat "$file"
}

quota_ok() {
  local value="$1" quota period
  read -r quota period <<< "$value"
  [[ "$quota" =~ ^[0-9]+$ && "$period" =~ ^[0-9]+$ ]] || die "cpu.max malformed: $value"
  [ "$period" -gt 0 ] || die "cpu.max invalid period: $value"
  [ $((quota * 10)) -le $((period * 8)) ] || die "cpu.max exceeds 800m: $value"
}

latest_phase29_dir() {
  find "$phase29_root" -mindepth 1 -maxdepth 1 -type d -name 'run-*' | sort | tail -1
}

ensure_tools() {
  local tool
  for tool in jq sha256sum sudo awk curl mktemp install psql pg_dump systemctl stat cmp; do
    command -v "$tool" >/dev/null || die "required command missing: $tool"
  done
}

prepare_tmp_root() {
  tmp_root="$(mktemp -d /dev/shm/phase30-preflight.XXXXXX)"
  chmod 700 "$tmp_root"
  trap 'rm -rf "$tmp_root"' EXIT
}

regular_json() {
  local file="$1"
  [ -f "$file" ] && [ ! -L "$file" ] && jq -e 'type == "object"' "$file" >/dev/null 2>&1
}

require_file() {
  local file="$1" label="$2"
  if [ ! -f "$file" ] || [ -L "$file" ]; then
    die "$label missing or symlinked"
  fi
}

resolved_phase29_dir() {
  if [ -n "$phase29_dir" ]; then
    printf '%s\n' "$phase29_dir"
  else
    latest_phase29_dir
  fi
}

json_sha() {
  sha256sum "$1" | awk '{print $1}'
}

validate_phase29_chain() {
  local dir="$1"
  local apply="$dir/shadow-apply.json" smoke="$dir/smoke.json" decision
  decision="$(find "$dir" -maxdepth 1 -type f -name 'decision*.json' -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2-)"
  [ -n "$decision" ] || die 'phase 29 decision artifact not found'
  require_file "$apply" 'phase 29 shadow-apply'
  require_file "$smoke" 'phase 29 smoke'
  require_file "$decision" 'phase 29 decision'
  regular_json "$apply" || die 'phase 29 shadow-apply is invalid JSON'
  regular_json "$smoke" || die 'phase 29 smoke is invalid JSON'
  regular_json "$decision" || die 'phase 29 decision is invalid JSON'
  jq -e '.status == "go"' "$apply" >/dev/null || die 'phase 29 shadow-apply is not green'
  jq -e '.status == "go"' "$smoke" >/dev/null || die 'phase 29 smoke is not green'

  local decision_state failed_count
  decision_state="$(jq -r '.decision' "$decision")"
  failed_count="$(jq '(.failed_gates // []) | length' "$decision")"
  case "$decision_state" in
    go)
      [ "$failed_count" -eq 0 ] || die 'phase 29 GO has failed gates'
      ;;
    no-go)
      [ "${PHASE30_ALLOW_PHASE29_NO_GO:-}" = "$override_literal" ] ||
        die 'phase 29 is no-go; set PHASE30_ALLOW_PHASE29_NO_GO=I_ACCEPT_PHASE29_RISK to override'
      jq -e '.failed_gates == ["live-stability"]' "$decision" >/dev/null ||
        die 'phase 29 override is only allowed for live-stability'
      ;;
    *)
      die "unexpected phase 29 decision: $decision_state"
      ;;
  esac
  printf '%s\n' "$decision"
}

parse_shadow_contract() {
  local file="$1"
  jq -e '
    .status == "go" and
    .placement.node == "atius-srv-1" and
    .services.router.type == "ClusterIP" and
    .services.redis.type == "ClusterIP" and
    .services.router.endpoints_ready == true and
    .services.redis.endpoints_ready == true and
    .services.router.cluster_ip != null and
    .services.redis.cluster_ip != null and
    .workloads.router.container.resources.requests_cpu == "500m" and
    .workloads.redis.container.resources.requests_cpu == "500m" and
    .workloads.postgres.container.resources.requests_cpu == "500m"
  ' "$file" >/dev/null || die 'phase 29 shadow contract is incomplete'
}

vault_export() {
  local profile="$1"
  /home/ubuntu/.local/bin/atius-vault-env "$profile"
}

run_psql() {
  local host="$1" port="$2" db="$3" user="$4" sql="$5" output="$6"
  PGPASSFILE="$tmp_root/.pgpass" PGPASSWORD="${POSTGRES_PASSWORD:-}" psql \
    --no-psqlrc \
    --no-password \
    -h "$host" -p "$port" -U "$user" -d "$db" \
    -v ON_ERROR_STOP=1 \
    -Atqc "$sql" > "$output"
}

count_public_tables_sql() {
  cat <<'SQL'
SELECT count(*)::text
FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE';
SQL
}

query_host_data_directory_privileged() {
  sudo -n -u postgres psql \
    --no-psqlrc \
    --no-password \
    -d "$database" \
    -Atqc "select current_setting('data_directory')"
}

query_podman_empty_table_count() {
  podman exec postgres psql \
    --no-psqlrc \
    --no-password \
    -U admin \
    -d "$database" \
    -Atqc "$(count_public_tables_sql)"
}

capture_db_topology() {
  local output="$1"
  local host_user host_tables host_version host_addr host_data_dir
  local pgbouncer_backend_host pgbouncer_backend_port pgbouncer_backend_db
  local router_pg_host router_pg_port k3s_tables podman_tables

  set +x
  # shellcheck disable=SC1090
  source <(vault_export router-ai-atius)
  host_user="${POSTGRES_USER:-$database_user}"
  : "${POSTGRES_PASSWORD:?missing POSTGRES_PASSWORD from Vault}"

  printf '%s:%s:%s:%s:%s\n' "$host_pg_host" "$host_pg_port" '*' "$host_user" "$POSTGRES_PASSWORD" > "$tmp_root/.pgpass"
  chmod 600 "$tmp_root/.pgpass"

  run_psql "$host_pg_host" "$host_pg_port" "$database" "$host_user" "$(count_public_tables_sql)" "$tmp_root/host_tables.txt"
  host_tables="$(tr -d '[:space:]' < "$tmp_root/host_tables.txt")"
  [ "$host_tables" = "$canonical_dbrouter_tables" ] || die "host DBRouterAiAtius table count is $host_tables, expected $canonical_dbrouter_tables"

  run_psql "$host_pg_host" "$host_pg_port" "$database" "$host_user" "select current_setting('server_version_num')" "$tmp_root/host_version.txt"
  host_version="$(tr -d '[:space:]' < "$tmp_root/host_version.txt")"
  [[ "$host_version" =~ ^17[0-9]{4}$ ]] || die "unexpected host PostgreSQL version_num: $host_version"

  run_psql "$host_pg_host" "$host_pg_port" "$database" "$host_user" "select inet_server_addr()::text" "$tmp_root/host_addr.txt"
  host_addr="$(tr -d '[:space:]' < "$tmp_root/host_addr.txt")"
  host_addr="${host_addr%/32}"
  [ "$host_addr" = "$host_pg_host" ] || die "host PostgreSQL inet_server_addr is $host_addr"

  query_host_data_directory_privileged > "$tmp_root/host_datadir.txt"
  host_data_dir="$(tr -d '[:space:]' < "$tmp_root/host_datadir.txt")"
  [ "$host_data_dir" = "$host_pg_data_dir" ] || die "host PostgreSQL data_directory is $host_data_dir"

  router_pg_host="$(kube -n "$namespace" get svc router-ai-atius-postgres -o jsonpath='{.spec.clusterIP}')"
  [[ "$router_pg_host" =~ ^[0-9.]+$ ]] || die 'router-ai-atius-postgres ClusterIP missing'
  router_pg_port="$(kube -n "$namespace" get svc router-ai-atius-postgres -o jsonpath='{.spec.ports[0].port}')"
  [ "$router_pg_port" = 5432 ] || die "unexpected router-ai-atius-postgres port: $router_pg_port"

  printf '%s:%s:%s:%s:%s\n' "$router_pg_host" "$router_pg_port" '*' "$host_user" "$POSTGRES_PASSWORD" > "$tmp_root/.pgpass"
  run_psql "$router_pg_host" "$router_pg_port" "$database" "$host_user" "$(count_public_tables_sql)" "$tmp_root/empty_tables.txt"
  k3s_tables="$(tr -d '[:space:]' < "$tmp_root/empty_tables.txt")"
  [[ "$k3s_tables" =~ ^[0-9]+$ ]] || die "k3s PostgreSQL table count is malformed: $k3s_tables"
  [ "$k3s_tables" -ge "$canonical_dbrouter_tables" ] || die "k3s PostgreSQL DBRouterAiAtius table count is $k3s_tables, expected at least $canonical_dbrouter_tables"

  podman_tables="$(query_podman_empty_table_count | tr -d '[:space:]')"
  [ "$podman_tables" = "$container_empty_tables" ] || die "Podman PostgreSQL DBRouterAiAtius table count is $podman_tables, expected $container_empty_tables"

  if ! sudo -n grep -Eq '^[[:space:]]*DBRouterAiAtius[[:space:]]*=' "$pgbouncer_config"; then
    die 'PgBouncer mapping for DBRouterAiAtius not found'
  fi
  python3 - "$pgbouncer_config" > "$tmp_root/pgbouncer-map.json" <<'PY'
import json, sys
from pathlib import Path

cfg = Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
in_dbs = False
line = ""
for raw in cfg:
    stripped = raw.strip()
    if not stripped or stripped.startswith(("#", ";")):
        continue
    if stripped.startswith("[") and stripped.endswith("]"):
        in_dbs = stripped.lower() == "[databases]"
        continue
    if not in_dbs:
        continue
    if stripped.startswith("DBRouterAiAtius"):
        line = stripped
        break
if not line:
    raise SystemExit("missing DBRouterAiAtius mapping")
_, rhs = line.split("=", 1)
parts = {}
for token in rhs.strip().split():
    if "=" in token:
        k, v = token.split("=", 1)
        parts[k] = v
print(json.dumps({
    "line": line,
    "host": parts.get("host", ""),
    "port": parts.get("port", ""),
    "dbname": parts.get("dbname", ""),
}, separators=(",", ":")))
PY
  pgbouncer_backend_host="$(jq -r '.host' "$tmp_root/pgbouncer-map.json")"
  pgbouncer_backend_port="$(jq -r '.port' "$tmp_root/pgbouncer-map.json")"
  pgbouncer_backend_db="$(jq -r '.dbname' "$tmp_root/pgbouncer-map.json")"
  [ "$pgbouncer_backend_host" = "$host_pg_host" ] || die "PgBouncer DBRouterAiAtius host is $pgbouncer_backend_host"
  [ "$pgbouncer_backend_port" = "$host_pg_port" ] || die "PgBouncer DBRouterAiAtius port is $pgbouncer_backend_port"
  [ -n "$pgbouncer_backend_db" ] || die 'PgBouncer DBRouterAiAtius dbname missing'

  jq -n \
    --arg host_pg_host "$host_pg_host" --argjson host_pg_port "$host_pg_port" \
    --arg host_pg_unit "$host_pg_unit" --arg host_pg_data_dir "$host_pg_data_dir" \
    --arg host_version "$host_version" --argjson host_tables "$host_tables" \
    --arg host_addr "$host_addr" \
    --arg router_pg_host "$router_pg_host" --argjson router_pg_port "$router_pg_port" \
    --argjson k3s_tables "$k3s_tables" \
    --argjson podman_tables "$podman_tables" \
    --arg pgbouncer_host "$pgbouncer_backend_host" --argjson pgbouncer_port "$pgbouncer_backend_port" \
    --arg pgbouncer_db "$pgbouncer_backend_db" \
    '{host_pg:{host:$host_pg_host,port:$host_pg_port,systemd_unit:$host_pg_unit,data_directory:$host_pg_data_dir,server_version_num:$host_version,inet_server_addr:$host_addr,public_tables:$host_tables},
      k3s_pg:{host:$router_pg_host,port:$router_pg_port,public_tables:$k3s_tables},
      podman_pg:{container:"postgres",public_tables:$podman_tables},
      pgbouncer:{backend_host:$pgbouncer_host,backend_port:$pgbouncer_port,backend_dbname:$pgbouncer_db}}' > "$output"
}

validate_live_k3s() {
  local apply="$1" smoke="$2"
  local node_json workloads services endpoints
  node_json="$(kube get node atius-srv-1 -o json)"
  jq -e --arg key "$label_key" --arg value "$label_value" '
    .metadata.labels[$key] == $value and
    any(.status.conditions[]; .type == "Ready" and .status == "True") and
    any(.status.conditions[]; .type == "DiskPressure" and .status == "False") and
    all(.spec.taints[]?; .key != "node.kubernetes.io/disk-pressure")
  ' <<< "$node_json" >/dev/null || die 'node atius-srv-1 is not ready/green/labeled'

  workloads="$(kube -n "$namespace" get pods -o json)"
  jq -e '
    (.items | length) == 3 and all(.items[];
      .spec.nodeName == "atius-srv-1" and .status.phase == "Running" and
      (.status.containerStatuses | length) == 1 and all(.status.containerStatuses[]; .ready == true) and
      all(.spec.containers[]; .resources.requests.cpu == "500m" and .resources.limits.cpu == "500m"))
  ' <<< "$workloads" >/dev/null || die 'k3s workload placement/resources are invalid'

  services="$(kube -n "$namespace" get services -o json)"
  endpoints="$(kube -n "$namespace" get endpointslices.discovery.k8s.io -o json)"
  jq -e '
    any(.items[]; .metadata.name == "router-ai-atius" and .spec.type == "ClusterIP" and .spec.clusterIP != "None") and
    any(.items[]; .metadata.name == "router-ai-atius-postgres" and .spec.type == "ClusterIP" and .spec.clusterIP != "None")
  ' <<< "$services" >/dev/null || die 'required ClusterIP services are missing'
  jq -e '
    any(.items[]; .metadata.labels["kubernetes.io/service-name"] == "router-ai-atius" and any(.endpoints[]; .conditions.ready == true and .nodeName == "atius-srv-1")) and
    any(.items[]; .metadata.labels["kubernetes.io/service-name"] == "router-ai-atius-postgres" and any(.endpoints[]; .conditions.ready == true and .nodeName == "atius-srv-1"))
  ' <<< "$endpoints" >/dev/null || die 'required EndpointSlices are not ready on atius-srv-1'

  parse_shadow_contract "$apply"
  jq -e '.status == "go" and .checks.health_status == 200 and .checks.authenticated_models_status == 200' "$smoke" >/dev/null ||
    die 'phase 29 smoke contract is not reusable for cutover'
}

scan_sensitive_output() {
  local file="$1"
  [ -f "$file" ] || die "sensitive scan target missing: $file"
  ! rg -n --fixed-strings "${POSTGRES_PASSWORD:-__absent__}" "$file" >/dev/null 2>&1 || die "sensitive value leaked into $file"
}

prepare_evidence_dir() {
  [ -n "$evidence_dir" ] || evidence_dir="$evidence_root/run-$(date -u +%Y%m%dT%H%M%SZ)"
  install -d -m 700 "$evidence_root"
  if [ -e "$evidence_dir" ]; then
    if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then
      die 'invalid evidence directory path'
    fi
  else
    install -d -m 700 "$evidence_dir"
  fi
  evidence_dir="$(realpath -e "$evidence_dir")"
  [ "$(stat -c '%U:%a' "$evidence_dir")" = "$(id -un):700" ] || die 'evidence directory owner/mode must be caller:700'
}

build_manifest() {
  local phase29_decision="$1" phase29_apply="$2" phase29_smoke="$3" db_topology="$4" status="$5"
  local output_file="$6" generated_at generated_epoch
  generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  generated_epoch="$(date +%s)"
  jq -n \
    --arg status "$status" \
    --arg generated_at "$generated_at" \
    --argjson generated_at_epoch "$generated_epoch" \
    --arg cpu_max "$(cpu_max_value)" \
    --arg phase29_decision_sha256 "$(json_sha "$phase29_decision")" \
    --arg phase29_apply_sha256 "$(json_sha "$phase29_apply")" \
    --arg phase29_smoke_sha256 "$(json_sha "$phase29_smoke")" \
    --argjson phase29_decision "$(cat "$phase29_decision")" \
    --argjson phase29_apply "$(cat "$phase29_apply")" \
    --argjson phase29_smoke "$(cat "$phase29_smoke")" \
    --argjson db_topology "$(cat "$db_topology")" \
    '{
      schema_version: 1,
      status: $status,
      generated_at: $generated_at,
      generated_at_epoch: $generated_at_epoch,
      cpu_max: $cpu_max,
      phase29: {
        decision_sha256: $phase29_decision_sha256,
        shadow_apply_sha256: $phase29_apply_sha256,
        smoke_sha256: $phase29_smoke_sha256,
        decision: $phase29_decision.decision,
        phase30_authorized: $phase29_decision.phase30_authorized,
        override_live_stability_only: ($phase29_decision.decision == "no-go")
      },
      cluster: {
        router_cluster_ip: $phase29_apply.services.router.cluster_ip,
        redis_cluster_ip: $phase29_apply.services.redis.cluster_ip,
        postgres_cluster_ip: $db_topology.k3s_pg.host,
        postgres_cluster_port: $db_topology.k3s_pg.port
      },
      db_topology: $db_topology,
      read_only: true,
      mutations: {apache:false,pgbouncer:false,podman:false,k3s:false}
    }' > "$output_file"
}

write_checksums() {
  local out="$1"
  (
    cd "$evidence_dir"
    sha256sum "$(basename "$out")" apache.vhost.conf pgbouncer.ini source-dbrouter.sql db-topology.json > SHA256SUMS
  )
}

run_prepare() {
  local dir decision apply smoke decision_file apply_file smoke_file db_topology manifest
  ensure_tools
  quota_ok "$(cpu_max_value)"
  prepare_tmp_root
  prepare_evidence_dir

  dir="$(resolved_phase29_dir)"
  [ -n "$dir" ] || die 'phase 29 evidence directory not found'
  dir="$(realpath -e "$dir")"
  decision_file="$(validate_phase29_chain "$dir")"
  apply_file="$dir/shadow-apply.json"
  smoke_file="$dir/smoke.json"
  validate_live_k3s "$apply_file" "$smoke_file"

  capture_db_topology "$tmp_root/db-topology.json"
  cp --preserve=mode,ownership,timestamps "$apache_config" "$evidence_dir/apache.vhost.conf"
  cp --preserve=mode,ownership,timestamps "$pgbouncer_config" "$evidence_dir/pgbouncer.ini"

  set +x
  # shellcheck disable=SC1090
  source <(vault_export router-ai-atius)
  POSTGRES_USER="${POSTGRES_USER:-$database_user}"
  printf '%s:%s:%s:%s:%s\n' "$host_pg_host" "$host_pg_port" '*' "${POSTGRES_USER:?missing POSTGRES_USER from Vault}" "${POSTGRES_PASSWORD:?missing POSTGRES_PASSWORD from Vault}" > "$tmp_root/.pgpass"
  chmod 600 "$tmp_root/.pgpass"
  PGPASSFILE="$tmp_root/.pgpass" PGPASSWORD="${POSTGRES_PASSWORD}" pg_dump \
    --no-password \
    --no-owner \
    --no-privileges \
    -h "$host_pg_host" -p "$host_pg_port" -U "$POSTGRES_USER" -d "$database" \
    -f "$evidence_dir/source-dbrouter.sql"
  [ -s "$evidence_dir/source-dbrouter.sql" ] || die 'source DB dump is empty'

  cp "$tmp_root/db-topology.json" "$evidence_dir/db-topology.json"

  manifest="$evidence_dir/manifest.json"
  build_manifest "$decision_file" "$apply_file" "$smoke_file" "$tmp_root/db-topology.json" \
    "$(jq -r '.decision' "$decision_file" | awk '{if ($1=="go") print "READY"; else print "READY_WITH_PHASE29_OVERRIDE"}')" \
    "$manifest"
  chmod 600 "$manifest"
  write_checksums "$manifest"
  scan_sensitive_output "$manifest"
  scan_sensitive_output "$evidence_dir/db-topology.json"
  output="${output:-$manifest}"
  echo "cutover preflight: $(jq -r '.status' "$manifest")"
  echo "evidence dir: $evidence_dir"
}

self_test() {
  local tmp_dir decision apply smoke topo manifest
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN
  decision="$tmp_dir/decision.json"
  apply="$tmp_dir/shadow-apply.json"
  smoke="$tmp_dir/smoke.json"
  topo="$tmp_dir/db-topology.json"
  jq -n '{decision:"go",phase30_authorized:true,failed_gates:[]}' > "$decision"
  jq -n '{status:"go",placement:{node:"atius-srv-1"},services:{router:{type:"ClusterIP",cluster_ip:"10.43.0.12",endpoints_ready:true},redis:{type:"ClusterIP",cluster_ip:"10.43.0.11",endpoints_ready:true}},workloads:{router:{container:{resources:{requests_cpu:"500m"}}},redis:{container:{resources:{requests_cpu:"500m"}}},postgres:{container:{resources:{requests_cpu:"500m"}}}}}' > "$apply"
  jq -n '{status:"go",checks:{health_status:200,authenticated_models_status:200}}' > "$smoke"
  jq -n --arg host "$host_pg_host" --argjson port "$host_pg_port" '{host_pg:{host:$host,port:$port,public_tables:34},k3s_pg:{host:"10.43.0.24",port:5432,public_tables:0},pgbouncer:{backend_host:$host,backend_port:$port,backend_dbname:"dbrouter"}}' > "$topo"
  manifest="$tmp_dir/manifest.json"
  build_manifest "$decision" "$apply" "$smoke" "$topo" READY "$manifest"
  jq -e '.status == "READY" and .phase29.decision == "go" and .cluster.router_cluster_ip == "10.43.0.12"' "$manifest" >/dev/null ||
    die 'READY manifest self-test failed'
  jq '.decision="no-go" | .phase30_authorized=false | .failed_gates=["live-stability"]' "$decision" > "$tmp_dir/no-go.json"
  build_manifest "$tmp_dir/no-go.json" "$apply" "$smoke" "$topo" READY_WITH_PHASE29_OVERRIDE "$manifest"
  jq -e '.status == "READY_WITH_PHASE29_OVERRIDE" and .phase29.override_live_stability_only == true' "$manifest" >/dev/null ||
    die 'override manifest self-test failed'
  echo 'cutover preflight self-test: PASS'
}

self_test_backup() {
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN
  evidence_dir="$tmp_dir/evidence"
  install -d -m 700 "$evidence_dir"
  printf 'vhost\n' > "$evidence_dir/apache.vhost.conf"
  printf 'pgbouncer\n' > "$evidence_dir/pgbouncer.ini"
  printf 'dump\n' > "$evidence_dir/source-dbrouter.sql"
  printf '{}\n' > "$evidence_dir/db-topology.json"
  printf '{}\n' > "$evidence_dir/manifest.json"
  write_checksums "$evidence_dir/manifest.json"
  [ -s "$evidence_dir/SHA256SUMS" ] || die 'SHA256SUMS backup fixture missing'
  echo 'cutover preflight backup self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --prepare) prepare=true ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --phase29-evidence-dir) phase29_dir="${2:?}"; shift ;;
    --output) output="${2:?}"; shift ;;
    --self-test) self_test; exit 0 ;;
    --self-test-backup) self_test_backup; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

if [ "$mode" != live ]; then
  echo 'cutover preflight dry-run: PASS; use --live --prepare for live capture'
  exit 0
fi

[ "${PHASE30_EXECUTE:-0}" = 1 ] || die '--live requires PHASE30_EXECUTE=1'
$prepare || die '--live requires --prepare'
run_prepare
