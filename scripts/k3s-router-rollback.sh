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

die() {
  echo "rollback failed: $*" >&2
  exit 1
}

install_with_target_metadata() {
  local source="$1" target="$2" owner group mode
  owner="$(sudo -n stat -c '%u' "$target")"
  group="$(sudo -n stat -c '%g' "$target")"
  mode="$(sudo -n stat -c '%a' "$target")"
  sudo -n install -o "$owner" -g "$group" -m "$mode" "$source" "$target"
}

write_evidence() {
  local status="$1" component="$2" details="$3" destination="$4"
  local tmp
  tmp="$(mktemp "${destination}.tmp.XXXXXX")"
  jq -n --arg status "$status" --arg component "$component" --argjson details "$(cat "$details")" \
    '{schema_version:1,status:$status,component:$component,details:$details,mutations:{apache:($component=="apache"),pgbouncer:($component=="pgbouncer"),podman:false,k3s:false}}' > "$tmp"
  mv "$tmp" "$destination"
  chmod 600 "$destination"
}

rollback_pgbouncer() {
  local backup="$evidence_dir/pgbouncer.ini" out="${output:-$evidence_dir/rollback-pgbouncer.json}" details
  [ -f "$backup" ] || die 'pgbouncer backup missing in evidence dir'
  details="$(mktemp)"
  jq -n '{restored_from_backup:true,target:"127.0.0.1:8745"}' > "$details"
  if [ "$mode" = live ]; then
    [ "${PHASE30_EXECUTE:-0}" = 1 ] || die '--live requires PHASE30_EXECUTE=1'
    install_with_target_metadata "$backup" "$pgbouncer_config"
    sudo -n systemctl reload pgbouncer
    write_evidence rolled-back pgbouncer "$details" "$out"
  else
    write_evidence dry-run pgbouncer "$details" "$out"
  fi
  rm -f "$details"
  echo "rollback pgbouncer: $out"
}

rollback_apache() {
  local backup="$evidence_dir/apache.vhost.conf" out="${output:-$evidence_dir/rollback-apache.json}" details
  [ -f "$backup" ] || die 'apache backup missing in evidence dir'
  details="$(mktemp)"
  jq -n '{restored_from_backup:true,target:"http://127.0.0.1:3000"}' > "$details"
  if [ "$mode" = live ]; then
    [ "${PHASE30_EXECUTE:-0}" = 1 ] || die '--live requires PHASE30_EXECUTE=1'
    install_with_target_metadata "$backup" "$apache_config"
    sudo -n apache2ctl configtest >/dev/null
    sudo -n systemctl reload apache2
    write_evidence rolled-back apache "$details" "$out"
  else
    write_evidence dry-run apache "$details" "$out"
  fi
  rm -f "$details"
  echo "rollback apache: $out"
}

self_test_pgbouncer() {
  local tmp evidence out
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  evidence="$tmp/evidence"; mkdir -p "$evidence"
  printf '%s\n' '[databases]' 'DBRouterAiAtius = host=127.0.0.1 port=8745 dbname=DBRouterAiAtius' > "$evidence/pgbouncer.ini"
  evidence_dir="$evidence"; output="$tmp/out.json"; rollback_pgbouncer
  jq -e '.status == "dry-run" and .component == "pgbouncer" and .details.target == "127.0.0.1:8745"' "$output" >/dev/null || die 'pgbouncer rollback self-test failed'
  echo 'rollback pgbouncer self-test: PASS'
}

self_test_apache() {
  local tmp evidence out
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  evidence="$tmp/evidence"; mkdir -p "$evidence"
  printf '%s\n' '<VirtualHost *:443>' 'ProxyPass / http://127.0.0.1:3000/' '</VirtualHost>' > "$evidence/apache.vhost.conf"
  evidence_dir="$evidence"; output="$tmp/out.json"; rollback_apache
  jq -e '.status == "dry-run" and .component == "apache" and .details.target == "http://127.0.0.1:3000"' "$output" >/dev/null || die 'apache rollback self-test failed'
  echo 'rollback apache self-test: PASS'
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
  pgbouncer) rollback_pgbouncer ;;
  apache) rollback_apache ;;
  *) die '--stage must be pgbouncer or apache' ;;
esac
