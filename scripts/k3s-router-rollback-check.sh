#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

mode=dry-run
evidence_dir=""
output=""
run_id=""
apache_config=/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf
apache_server_name=router.atius.com.br
unit_name=container-router-ai-atius.service
pod_name=atius-ai-router
local_url=http://127.0.0.1:3000

die() {
  echo "rollback check failed: $*" >&2
  exit 1
}

apache_selected_vhost_path() {
  awk -v wanted="$apache_server_name" '
    function emit(name, path) {
      if (name != wanted) return
      sub(/^\(/, "", path); sub(/:[0-9]+\)$/, "", path); print path
    }
    $1 == "port" && $2 == "443" && $3 == "namevhost" { current=$(NF); emit($4, current); next }
    $1 == "*:443" && $2 != "is" { current=$(NF); emit($2, current); next }
    $1 == "alias" && $2 == wanted { emit($2, current) }
  ' "$1"
}

expected_router_directives() {
  printf '%s\n' \
    $'rewriterule\t^/v1/models/?$\thttp://127.0.0.1:3000/v1/models\t[P,L,QSA]' \
    $'proxypassreverse\t/v1/models\thttp://127.0.0.1:3000/v1/models\t' \
    $'proxypass\t/v1/\thttp://127.0.0.1:3000/v1/\t' \
    $'proxypassreverse\t/v1/\thttp://127.0.0.1:3000/v1/\t' \
    $'proxypass\t/health\thttp://127.0.0.1:3000/api/status\t' \
    $'proxypassreverse\t/health\thttp://127.0.0.1:3000/api/status\t' \
    $'rewriterule\t^/login$\t/sign-in\t[PT,L]' \
    $'rewriterule\t^/logoff$\t/logout\t[PT,L]' \
    $'proxypass\t/login\thttp://127.0.0.1:3000/sign-in\t' \
    $'proxypassreverse\t/login\thttp://127.0.0.1:3000/sign-in\t' \
    $'proxypass\t/logoff\thttp://127.0.0.1:3000/logout\t' \
    $'proxypassreverse\t/logoff\thttp://127.0.0.1:3000/logout\t' \
    $'proxypass\t/api/\thttp://127.0.0.1:3000/api/\t' \
    $'proxypassreverse\t/api/\thttp://127.0.0.1:3000/api/\t' \
    $'proxypass\t/\thttp://127.0.0.1:3000/\t' \
    $'proxypassreverse\t/\thttp://127.0.0.1:3000/\t'
}

effective_vhost_router_directives() {
  awk -v wanted="$apache_server_name" '
    BEGIN { IGNORECASE=1 }
    function router_source(kind, source) {
      if (kind == "rewriterule") return source == "^/v1/models/?$" || source == "^/login$" || source == "^/logoff$"
      return source == "/v1/models" || source == "/v1/" || source == "/health" || source == "/login" || source == "/logoff" || source == "/api/" || source == "/"
    }
    function router_target(target) {
      return target ~ /^https?:\/\/[^/]*:3000(\/|$)/ ||
        target ~ /(10\.43\.|\.svc([.:/]|$)|k3s|nodeport)/
    }
    function proxyable_router_rewrite(target, options) {
      return options ~ /(^|,|\[)PT(,|\]|$)/ &&
        (target ~ /^\/(api|v1)(\/|$)/ || target == "/sign-in" || target == "/logout")
    }
    function append(kind, source, target, options, i) {
      options=""; for (i=4; i<=NF; i++) options=options (i == 4 ? "" : " ") $i
      routes=routes kind "\t" source "\t" target "\t" options "\n"
    }
    /^[[:space:]]*<VirtualHost[[:space:]][^>]*:443[^>]*>/ { inside=1; selected=0; routes=""; next }
    inside && /^[[:space:]]*<\/VirtualHost>/ { if (selected) printf "%s", routes; inside=0; next }
    !inside || /^[[:space:]]*#/ { next }
    tolower($1) == "servername" && tolower($2) == tolower(wanted) { selected=1; next }
    tolower($1) ~ /^(proxypass|proxypassreverse|rewriterule)$/ {
      kind=tolower($1)
      options=""; for (i=4; i<=NF; i++) options=options (i == 4 ? "" : " ") $i
      if (router_source(kind, $2) || router_target($3) || (kind == "rewriterule" && proxyable_router_rewrite($3, options))) append(kind, $2, $3)
    }
  ' "$1"
}

apache_routes_are_podman() {
  local file="$1"
  [ -f "$file" ] || return 1
  cmp -s <(expected_router_directives | LC_ALL=C sort) \
    <(effective_vhost_router_directives "$file" | LC_ALL=C sort)
}

apache_has_k3s_target() {
  local file="$1" routes
  [ -f "$file" ] || return 1
  routes="$(effective_vhost_router_directives "$file")"
  awk -F '\t' '$3 ~ /(10\.43\.|\.svc([.:/]|$)|k3s|nodeport)/ { found=1 } END { exit !found }' <<< "$routes"
}

record_failure() {
  jq -cn --arg check "$1" --arg reason "$2" '{check:$check,reason:$reason}' >> "$failures_file"
}

write_evidence() {
  local status="$1" apache_sha="$2" unit_present="$3" unit_active="$4" pod_exists="$5" pod_running="$6"
  local containers_ready="$7" limits_valid="$8" health_ok="$9" clianything_ok="${10}"
  local syntax_ok="${11}" vhost_selection_ok="${12}" selected_vhost="${13}" routes_ok="${14}" k3s_present="${15}" generated_at generated_epoch tmp
  generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  generated_epoch="$(date +%s)"
  tmp="$(mktemp "${output}.tmp.XXXXXX")"
  chmod 600 "$tmp"
  jq -n --arg status "$status" --arg run_id "$run_id" --arg generated_at "$generated_at" \
    --argjson generated_at_epoch "$generated_epoch" --arg unit "$unit_name" --arg pod "$pod_name" \
    --arg local_url "$local_url" --arg apache_path "$apache_config" --arg apache_sha256 "$apache_sha" \
    --argjson unit_present "$unit_present" --argjson unit_active "$unit_active" \
    --argjson pod_exists "$pod_exists" --argjson pod_running "$pod_running" \
    --argjson containers_ready "$containers_ready" --argjson limits_valid "$limits_valid" \
    --argjson health_ok "$health_ok" --argjson clianything_ok "$clianything_ok" \
    --argjson syntax_ok "$syntax_ok" --argjson vhost_selection_ok "$vhost_selection_ok" --arg selected_vhost "$selected_vhost" \
    --argjson routes_ok "$routes_ok" --argjson k3s_present "$k3s_present" \
    --argjson failed_checks "$(jq -s '.' "$failures_file")" \
    '{schema_version:2,status:$status,run_id:$run_id,generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,
      podman:{unit:{name:$unit,present:$unit_present,required:false,active:$unit_active},pod:{name:$pod,exists:$pod_exists,running:$pod_running},
        containers_ready:$containers_ready,limits_valid:$limits_valid,local_url:$local_url,health_ok:$health_ok,
        clianything_backend:"podman",clianything_ok:$clianything_ok},
      apache:{config_path:$apache_path,config_sha256:$apache_sha256,syntax_ok:$syntax_ok,
        server_name:"router.atius.com.br",selected_vhost:$selected_vhost,vhost_selection_ok:$vhost_selection_ok,
        routes_to_podman:$routes_ok,k3s_target_present:$k3s_present},
      failed_checks:$failed_checks,read_only:true,mutations:{apache:false,podman:false,k3s:false}}' > "$tmp"
  mv "$tmp" "$output"
}

run_live() {
  local apache_sha="" status=go containers unit_load_state pod_state
  local unit_present=false unit_active=false pod_exists=false pod_running=false containers_ready=true
  local limits_valid=false health_ok=false clianything_ok=false syntax_ok=false vhost_selection_ok=false selected_vhost="" routes_ok=false k3s_present=false apache_dump
  [ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
  [[ "$run_id" =~ ^phase29-[A-Za-z0-9._-]+$ ]] || die '--run-id must be a generated Phase 29 run identifier'
  [ -n "$evidence_dir" ] || die '--evidence-dir is required'
  if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then die 'evidence directory must be a real directory'; fi
  evidence_dir="$(realpath -e "$evidence_dir")"
  [ "$(stat -c '%U:%a' "$evidence_dir")" = "$(id -un):700" ] || die 'evidence directory owner/mode must be caller:700'
  output="${output:-$evidence_dir/rollback-$run_id.json}"
  case "$output" in "$evidence_dir"/*) ;; *) die 'output must stay inside the evidence directory' ;; esac
  if [ -e "$output" ] || [ -L "$output" ]; then die 'fresh rollback evidence path already exists'; fi
  for command in jq sha256sum curl systemctl podman awk sudo; do command -v "$command" >/dev/null || die "required command missing: $command"; done
  failures_file="$(mktemp "$evidence_dir/.rollback-failures.XXXXXX")"
  trap 'rm -f "${failures_file:-}"' EXIT
  chmod 600 "$failures_file"

  unit_load_state="$(systemctl --user show "$unit_name" -p LoadState --value 2>/dev/null || true)"
  if [ -n "$unit_load_state" ] && [ "$unit_load_state" != not-found ]; then
    unit_present=true
    if systemctl --user is-active --quiet "$unit_name"; then unit_active=true
    else record_failure podman-unit 'optional user unit exists but is not active'; fi
  fi

  if podman pod exists "$pod_name"; then
    pod_exists=true
    pod_state="$(podman pod inspect "$pod_name" --format '{{.State}}' 2>/dev/null || true)"
    if [ "$pod_state" = Running ]; then pod_running=true
    else record_failure podman-pod 'Podman pod exists but is not Running'; fi
  else
    record_failure podman-pod 'Podman pod atius-ai-router does not exist'
  fi

  if ! containers="$(podman ps --filter "pod=$pod_name" --format '{{.Names}}\t{{.Status}}' 2>/dev/null)"; then
    containers_ready=false
    record_failure podman-containers 'pod container inventory failed'
  else
    local required
    for required in router-ai-atius postgres redis; do
      if ! awk -F '\t' -v wanted="$required" '$1 == wanted && $2 ~ /^Up / {found=1} END {exit !found}' <<< "$containers"; then
        containers_ready=false
        record_failure podman-containers "required container $required is not running"
      fi
    done
  fi

  if scripts/podman-admin.sh inspect-limits >/dev/null 2>&1; then limits_valid=true
  else record_failure podman-limits 'runtime CPU/memory limits are not valid'; fi
  if curl --fail --silent --show-error --connect-timeout 3 --max-time 15 "$local_url/api/status" -o /dev/null 2>/dev/null; then health_ok=true
  else record_failure podman-health 'local Podman endpoint is unavailable'; fi
  if bin/clianything status --backend podman >/dev/null 2>&1; then clianything_ok=true
  else record_failure podman-clianything 'CLIAnything status --backend podman failed'; fi

  if sudo -n apache2ctl configtest >/dev/null 2>&1; then syntax_ok=true
  else record_failure apache-syntax 'apache2ctl configtest failed'; fi
  apache_dump="$(sudo -n apache2ctl -S 2>&1 || true)"
  selected_vhost="$(apache_selected_vhost_path <(printf '%s\n' "$apache_dump"))"
  if [ "$selected_vhost" = "$apache_config" ]; then vhost_selection_ok=true
  else record_failure apache-vhost-selection 'router.atius.com.br:443 is not selected exactly from the canonical enabled vhost'; fi
  if [ ! -f "$apache_config" ]; then
    record_failure apache-config 'canonical enabled Apache vhost is unavailable'
  else
    apache_sha="$(sha256sum "$apache_config" | awk '{print $1}')"
    if apache_routes_are_podman "$apache_config"; then routes_ok=true
    else record_failure apache-routes 'canonical vhost routes do not all target 127.0.0.1:3000'; fi
    if apache_has_k3s_target "$apache_config"; then k3s_present=true; record_failure apache-k3s-target 'canonical vhost contains a k3s target'; fi
  fi

  [ ! -s "$failures_file" ] || status=no-go
  write_evidence "$status" "$apache_sha" "$unit_present" "$unit_active" "$pod_exists" "$pod_running" \
    "$containers_ready" "$limits_valid" "$health_ok" "$clianything_ok" "$syntax_ok" "$vhost_selection_ok" \
    "$selected_vhost" "$routes_ok" "$k3s_present"
  echo "rollback evidence: $output ($status, run_id=$run_id)"
  [ "$status" = go ]
}

self_test() {
  local tmp good bad dump selected
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  good="$tmp/good.conf"; bad="$tmp/bad.conf"
  {
    printf '%s\n' '<VirtualHost *:443>' 'ServerName router.atius.com.br'
    expected_router_directives | awk -F '\t' '{printf "%s %s %s%s%s\n", $1, $2, $3, ($4 == "" ? "" : " "), $4}'
    printf '%s\n' \
      'ProxyPass /docs/ http://127.0.0.1:3003/' \
      'ProxyPassReverse /docs/ http://127.0.0.1:3003/' \
      '</VirtualHost>'
  } > "$good"
  dump="$tmp/vhosts.txt"
  printf '%s\n' '*:443 is a NameVirtualHost' \
    ' port 443 namevhost router.atius.com.br (/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf:2)' > "$dump"
  selected="$(apache_selected_vhost_path "$dump")"
  [ "$selected" = "$apache_config" ] || die 'canonical apache2ctl -S fixture was rejected'
  sed 's#router.atius.com.br-le-ssl.conf#false-router.conf#' "$dump" > "$bad"
  [ "$(apache_selected_vhost_path "$bad")" != "$apache_config" ] || die 'false vhost fixture was accepted'
  apache_routes_are_podman "$good" || die 'valid Podman route fixture was rejected'
  if apache_has_k3s_target "$good"; then die 'valid Podman route fixture was classified as k3s'; fi
  sed 's#127.0.0.1:3000/v1/#10.43.0.20:3000/v1/#' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die 'ClusterIP Apache target was accepted'; fi
  apache_has_k3s_target "$bad" || die 'ClusterIP Apache target was not detected'
  sed '/proxypass \/health/d' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die 'missing health route was accepted'; fi
  sed '/proxypass \/v1\//p' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die 'duplicate route fixture was accepted'; fi
  sed '/<\/VirtualHost>/i proxypass /unexpected http://127.0.0.1:3000/unexpected' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die '/unexpected router route fixture was accepted'; fi
  sed '/<\/VirtualHost>/i proxypass /unexpected http://localhost:3000/unexpected' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die 'localhost:3000 router route fixture was accepted'; fi
  sed '/<\/VirtualHost>/i RewriteRule ^/unexpected$ /api/status [PT,L]' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die 'proxyable unexpected rewrite fixture was accepted'; fi
  sed 's#proxypass /health http://127.0.0.1:3000/api/status#proxypass /health http://127.0.0.1:3001/api/status#' "$good" > "$bad"
  if apache_routes_are_podman "$bad"; then die 'competing router target fixture was accepted'; fi

  output="$tmp/rollback.json"; failures_file="$tmp/failures.jsonl"; : > "$failures_file"; run_id=phase29-selftest
  record_failure podman-health fixture
  write_evidence no-go "$(sha256sum "$good" | awk '{print $1}')" false false true true true true false true true true "$apache_config" true false
  jq -e '.status == "no-go" and .podman.unit.present == false and .podman.unit.required == false and
    .podman.pod.exists == true and .podman.health_ok == false and .podman.clianything_backend == "podman" and
    .podman.clianything_ok == true and .apache.syntax_ok == true and .apache.vhost_selection_ok == true and
    .apache.routes_to_podman == true and
    (.failed_checks | length) == 1' "$output" >/dev/null || die 'independent rollback evidence fields collapsed into aggregate status'
  echo 'rollback check self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --output) output="${2:?}"; shift ;;
    --run-id) run_id="${2:?}"; shift ;;
    --self-test) self_test; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

if [ "$mode" = dry-run ]; then
  echo 'rollback check dry-run: no Podman, Apache, or k3s mutation performed'
  exit 0
fi
run_live
