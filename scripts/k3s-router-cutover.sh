#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

mode=dry-run
stage=""
evidence_dir=""
output=""
apache_config=/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf
pgbouncer_config=/etc/pgbouncer/pgbouncer.ini
old_router_target=http://127.0.0.1:3000
public_health_url=https://router.atius.com.br/health

die() {
  echo "cutover failed: $*" >&2
  exit 1
}

require_tools() {
  local tool
  for tool in jq sha256sum cmp awk sed mktemp python3 sudo systemctl apache2ctl curl psql; do
    command -v "$tool" >/dev/null || die "required command missing: $tool"
  done
}

install_with_target_metadata() {
  local source="$1" target="$2" owner group mode
  owner="$(sudo -n stat -c '%u' "$target")"
  group="$(sudo -n stat -c '%g' "$target")"
  mode="$(sudo -n stat -c '%a' "$target")"
  sudo -n install -o "$owner" -g "$group" -m "$mode" "$source" "$target"
}

manifest_path() {
  printf '%s\n' "$evidence_dir/manifest.json"
}

manifest_status() {
  jq -r '.status' "$(manifest_path)"
}

load_manifest_contract() {
  [ -n "$evidence_dir" ] || die '--evidence-dir is required'
  [ -d "$evidence_dir" ] || die 'evidence directory missing'
  [ ! -L "$evidence_dir" ] || die 'evidence directory must not be a symlink'
  evidence_dir="$(realpath -e "$evidence_dir")"
  [ "$(stat -c '%U:%a' "$evidence_dir")" = "$(id -un):700" ] || die 'evidence directory owner/mode must be caller:700'
  [ -f "$(manifest_path)" ] || die 'manifest.json missing'
  case "$(manifest_status)" in
    READY|READY_WITH_PHASE29_OVERRIDE) ;;
    *) die 'manifest status is not READY' ;;
  esac
}

router_cluster_ip() {
  jq -r '.cluster.router_cluster_ip' "$(manifest_path)"
}

postgres_cluster_ip() {
  jq -r '.cluster.postgres_cluster_ip' "$(manifest_path)"
}

postgres_cluster_port() {
  jq -r '.cluster.postgres_cluster_port' "$(manifest_path)"
}

pgbouncer_current_mapping() {
  python3 - "$pgbouncer_config" <<'PY'
import json, sys
from pathlib import Path

cfg = Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
in_dbs = False
matches = []
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
        matches.append(stripped)
if len(matches) != 1:
    raise SystemExit(f"expected exactly one DBRouterAiAtius mapping, found {len(matches)}")
line = matches[0]
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
}

pgbouncer_candidate() {
  local source="$1" dest="$2" target_host="$3" target_port="$4"
  python3 - "$source" "$dest" "$target_host" "$target_port" <<'PY'
import sys
from pathlib import Path

source, dest, host, port = sys.argv[1:]
lines = Path(source).read_text(encoding="utf-8").splitlines()
in_dbs = False
matches = 0
out = []
for raw in lines:
    stripped = raw.strip()
    if stripped.startswith("[") and stripped.endswith("]"):
        in_dbs = stripped.lower() == "[databases]"
        out.append(raw)
        continue
    if in_dbs and stripped.startswith("DBRouterAiAtius"):
        matches += 1
        _, rhs = raw.split("=", 1)
        parts = []
        dbname_found = False
        for token in rhs.strip().split():
            if token.startswith("host="):
                parts.append(f"host={host}")
            elif token.startswith("port="):
                parts.append(f"port={port}")
            else:
                if token.startswith("dbname="):
                    dbname_found = True
                parts.append(token)
        if not dbname_found:
            parts.append("dbname=DBRouterAiAtius")
        raw = f"DBRouterAiAtius = {' '.join(parts)}"
    out.append(raw)
if matches != 1:
    raise SystemExit(f"expected exactly one DBRouterAiAtius mapping, found {matches}")
Path(dest).write_text("\n".join(out) + "\n", encoding="utf-8")
PY
}

pgbouncer_only_changes_backend() {
  local before="$1" after="$2"
  python3 - "$before" "$after" <<'PY'
import json, sys
from pathlib import Path

def parse(path):
    lines = Path(path).read_text(encoding="utf-8").splitlines()
    in_dbs = False
    data = {}
    for raw in lines:
        stripped = raw.strip()
        if not stripped or stripped.startswith(("#", ";")):
            continue
        if stripped.startswith("[") and stripped.endswith("]"):
            in_dbs = stripped.lower() == "[databases]"
            continue
        if not in_dbs or "=" not in raw:
            continue
        name, rhs = raw.split("=", 1)
        name = name.strip()
        parts = {}
        for token in rhs.strip().split():
            if "=" in token:
                k, v = token.split("=", 1)
                parts[k] = v
        data[name] = parts
    return data

before = parse(sys.argv[1])
after = parse(sys.argv[2])
if before.keys() != after.keys():
    raise SystemExit("database key set changed")
for name, lhs in before.items():
    rhs = after[name]
    if name != "DBRouterAiAtius":
      if lhs != rhs:
        raise SystemExit("non-target database entry changed")
      continue
    for key in set(lhs) | set(rhs):
      if key in {"host", "port"}:
        continue
      if lhs.get(key) != rhs.get(key):
        raise SystemExit("DBRouterAiAtius changed outside host/port")
print("ok")
PY
}

apache_router_targets() {
  local file="$1"
  awk 'BEGIN{IGNORECASE=1}
    /^[[:space:]]*#/{next}
    tolower($1) ~ /^(proxypass|proxypassreverse|rewriterule)$/ {
      if ($0 ~ /127\.0\.0\.1:3000/ || $0 ~ /10\.43\./) print
    }' "$file"
}

apache_docs_lines() {
  local file="$1"
  awk 'BEGIN{IGNORECASE=1}
    /^[[:space:]]*#/{next}
    /127\.0\.0\.1:3003/ {print}' "$file"
}

apache_candidate() {
  local source="$1" dest="$2" cluster_ip="$3"
  python3 - "$source" "$dest" "$cluster_ip" <<'PY'
import re, sys
from pathlib import Path

source, dest, ip = sys.argv[1:]
text = Path(source).read_text(encoding="utf-8")
updated = text.replace("http://127.0.0.1:3000", f"http://{ip}:3000")
Path(dest).write_text(updated, encoding="utf-8")
PY
}

apache_only_changes_router_target() {
  local before="$1" after="$2"
  local before_docs after_docs
  before_docs="$(mktemp)"; after_docs="$(mktemp)"
  apache_docs_lines "$before" > "$before_docs"
  apache_docs_lines "$after" > "$after_docs"
  if ! cmp -s "$before_docs" "$after_docs"; then
    rm -f "$before_docs" "$after_docs"
    die '127.0.0.1:3003 region changed'
  fi
  rm -f "$before_docs" "$after_docs"
}

write_evidence() {
  local status="$1" component="$2" details_json="$3" destination="$4"
  local tmp generated_at generated_epoch
  tmp="$(mktemp "${destination}.tmp.XXXXXX")"
  generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  generated_epoch="$(date +%s)"
  jq -n \
    --arg status "$status" \
    --arg component "$component" \
    --arg generated_at "$generated_at" \
    --argjson generated_at_epoch "$generated_epoch" \
    --argjson manifest "$(cat "$(manifest_path)")" \
    --argjson details "$(cat "$details_json")" \
    '{schema_version:1,status:$status,component:$component,generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,
      manifest_status:$manifest.status,cluster:$manifest.cluster,details:$details,
      mutations:{apache:($component=="apache"),pgbouncer:($component=="pgbouncer"),podman:false,k3s:false}}' > "$tmp"
  mv "$tmp" "$destination"
  chmod 600 "$destination"
}

host_admin_scram_secret() {
  sudo -n -u postgres psql --no-psqlrc -d DBRouterAiAtius -Atqc \
    "select rolpassword from pg_authid where rolname='admin'"
}

k3s_admin_scram_fingerprint() {
  set +x
  # shellcheck disable=SC1090
  source <(/home/ubuntu/.local/bin/atius-vault-env router-ai-atius)
  PGPASSWORD="${POSTGRES_PASSWORD:-}" psql \
    --no-password \
    --no-psqlrc \
    -h "$(postgres_cluster_ip)" -p "$(postgres_cluster_port)" -U admin -d DBRouterAiAtius \
    -Atqc "select md5(coalesce((select rolpassword from pg_authid where rolname='admin'),''))"
}

sync_admin_scram_secret_if_needed() {
  local host_secret host_fp k3s_fp sql
  host_secret="$(host_admin_scram_secret)"
  [[ "$host_secret" == SCRAM-SHA-256\$* ]] || die 'host admin role does not expose a SCRAM secret'
  host_fp="$(printf '%s' "$host_secret" | md5sum | awk '{print $1}')"
  k3s_fp="$(k3s_admin_scram_fingerprint | tr -d '[:space:]')"
  if [ "$host_fp" = "$k3s_fp" ]; then
    return 0
  fi
  sql="$(mktemp /dev/shm/phase30-admin-scram.XXXXXX.sql)"
  chmod 600 "$sql"
  printf "ALTER ROLE admin PASSWORD '%s';\n" "${host_secret//\'/\'\'}" > "$sql"
  set +x
  # shellcheck disable=SC1090
  source <(/home/ubuntu/.local/bin/atius-vault-env router-ai-atius)
  PGPASSWORD="${POSTGRES_PASSWORD:-}" psql \
    --no-password \
    --no-psqlrc \
    -h "$(postgres_cluster_ip)" -p "$(postgres_cluster_port)" -U admin -d DBRouterAiAtius \
    -v ON_ERROR_STOP=1 -f "$sql" >/dev/null
  rm -f "$sql"
  k3s_fp="$(k3s_admin_scram_fingerprint | tr -d '[:space:]')"
  [ "$host_fp" = "$k3s_fp" ] || die 'k3s admin SCRAM fingerprint still differs from host after sync'
}

verify_pgbouncer_backend() {
  local expected_tables actual sql
  set +x
  # shellcheck disable=SC1090
  source <(/home/ubuntu/.local/bin/atius-vault-env router-ai-atius)
  expected_tables="$(jq -r '.db_topology.k3s_pg.public_tables' "$(manifest_path)")"
  sql="select count(*) from information_schema.tables where table_schema='public' and table_type='BASE TABLE'"
  actual="$(PGPASSWORD="${POSTGRES_PASSWORD:-}" psql \
    --no-password \
    --no-psqlrc \
    -h 127.0.0.1 -p 6432 -U admin -d DBRouterAiAtius \
    -Atqc "$sql" | tr -d '[:space:]')"
  [ "$actual" = "$expected_tables" ] || die "PgBouncer backend public table count is $actual, expected $expected_tables"
}

run_pgbouncer_stage() {
  local tmp mapping current backup candidate details out
  load_manifest_contract
  require_tools
  tmp="$(mktemp -d /dev/shm/phase30-cutover-pgbouncer.XXXXXX)"
  current="$tmp/current.ini"
  backup="$evidence_dir/pgbouncer.ini"
  candidate="$tmp/candidate.ini"
  details="$tmp/details.json"
  out="${output:-$evidence_dir/cutover-pgbouncer.json}"
  cp --preserve=mode,ownership,timestamps "$pgbouncer_config" "$current"
  pgbouncer_candidate "$current" "$candidate" "$(postgres_cluster_ip)" "$(postgres_cluster_port)"
  pgbouncer_only_changes_backend "$current" "$candidate" >/dev/null
  mapping="$(pgbouncer_current_mapping)"
  jq -n --argjson before "$mapping" --arg target_host "$(postgres_cluster_ip)" --argjson target_port "$(postgres_cluster_port)" \
    '{before:$before,after:{host:$target_host,port:$target_port,dbname:"DBRouterAiAtius"},only_target_changed:true}' > "$details"

  if [ "$mode" != live ]; then
    write_evidence dry-run pgbouncer "$details" "$out"
    rm -rf "$tmp"
    echo "cutover dry-run stage pgbouncer: $out"
    return 0
  fi
  [ "${PHASE30_EXECUTE:-0}" = 1 ] || die '--live requires PHASE30_EXECUTE=1'
  sync_admin_scram_secret_if_needed
  sudo -n cp --preserve=mode,ownership,timestamps "$backup" "$tmp/original.ini"
  install_with_target_metadata "$candidate" "$pgbouncer_config"
  sudo -n systemctl reload pgbouncer
  verify_pgbouncer_backend
  write_evidence cutover-applied pgbouncer "$details" "$out"
  rm -rf "$tmp"
  echo "cutover pgbouncer: $out"
}

run_apache_stage() {
  local tmp current candidate details out
  load_manifest_contract
  require_tools
  tmp="$(mktemp -d /dev/shm/phase30-cutover-apache.XXXXXX)"
  current="$tmp/current.conf"
  candidate="$tmp/candidate.conf"
  details="$tmp/details.json"
  out="${output:-$evidence_dir/cutover-apache.json}"
  cp --preserve=mode,ownership,timestamps "$apache_config" "$current"
  apache_candidate "$current" "$candidate" "$(router_cluster_ip)"
  apache_only_changes_router_target "$current" "$candidate"
  jq -n --arg old "$old_router_target" --arg new "http://$(router_cluster_ip):3000" \
    '{from:$old,to:$new,docs_region_preserved:true}' > "$details"

  if [ "$mode" != live ]; then
    write_evidence dry-run apache "$details" "$out"
    rm -rf "$tmp"
    echo "cutover dry-run stage apache: $out"
    return 0
  fi
  [ "${PHASE30_EXECUTE:-0}" = 1 ] || die '--live requires PHASE30_EXECUTE=1'
  install_with_target_metadata "$candidate" "$apache_config"
  sudo -n apache2ctl configtest >/dev/null
  sudo -n systemctl reload apache2
  curl --fail --silent --show-error --connect-timeout 5 --max-time 20 "$public_health_url" -o /dev/null
  write_evidence cutover-applied apache "$details" "$out"
  rm -rf "$tmp"
  echo "cutover apache: $out"
}

self_test_pgbouncer() {
  local tmp before after
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  before="$tmp/before.ini"; after="$tmp/after.ini"
  cat > "$before" <<'EOF'
[databases]
DBRouterAiAtius = host=127.0.0.1 port=8745 dbname=DBRouterAiAtius
gbrain = host=127.0.0.1 port=6543 dbname=gbrain
EOF
  pgbouncer_candidate "$before" "$after" 10.43.179.157 5432
  pgbouncer_only_changes_backend "$before" "$after" >/dev/null
  python3 - "$after" <<'PY'
import json, sys
from pathlib import Path
line = [l.strip() for l in Path(sys.argv[1]).read_text().splitlines() if l.strip().startswith("DBRouterAiAtius")][0]
assert "host=10.43.179.157" in line and "port=5432" in line
PY
  echo 'cutover pgbouncer self-test: PASS'
}

self_test_apache() {
  local tmp before after
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  before="$tmp/before.conf"; after="$tmp/after.conf"
  cat > "$before" <<'EOF'
<VirtualHost *:443>
ServerName router.atius.com.br
RewriteRule ^/v1/models/?$ http://127.0.0.1:3000/v1/models [P,L,QSA]
ProxyPassReverse /v1/models http://127.0.0.1:3000/v1/models
ProxyPass /v1/ http://127.0.0.1:3000/v1/
ProxyPassReverse /v1/ http://127.0.0.1:3000/v1/
ProxyPass /health http://127.0.0.1:3000/api/status
ProxyPassReverse /health http://127.0.0.1:3000/api/status
ProxyPass /docs http://127.0.0.1:3003
ProxyPassReverse /docs http://127.0.0.1:3003
ProxyPass / http://127.0.0.1:3000/
ProxyPassReverse / http://127.0.0.1:3000/
</VirtualHost>
EOF
  apache_candidate "$before" "$after" 10.43.102.221
  apache_only_changes_router_target "$before" "$after"
  rg -n "10\\.43\\.102\\.221:3000" "$after" >/dev/null || die 'router target was not replaced'
  rg -n "127\\.0\\.0\\.1:3003" "$after" >/dev/null || die 'docs target changed unexpectedly'
  echo 'cutover apache self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --stage) stage="${2:?}"; shift ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --output) output="${2:?}"; shift ;;
    --self-test)
      case "${2:-}" in
        pgbouncer) self_test_pgbouncer ;;
        apache) self_test_apache ;;
        *) die '--self-test requires pgbouncer or apache' ;;
      esac
      exit 0
      ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

case "$stage" in
  pgbouncer) run_pgbouncer_stage ;;
  apache) run_apache_stage ;;
  *) die '--stage must be pgbouncer or apache' ;;
esac
