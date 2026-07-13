#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

mode=dry-run
evidence_dir=""
evidence_root="${PHASE29_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase29}"
resume=false
baseline_free_bytes=""

die() {
  echo "cleanup failed: $*" >&2
  exit 1
}

cpu_max_value() {
  local cgroup file
  cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"
  file="/sys/fs/cgroup${cgroup}/cpu.max"
  [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"
  cat "$file"
}

# Every entry was audited on atius-srv-1 as an unmounted, regenerable cache or
# an orphaned build directory with no process holding it open.
ALLOWLIST=(
  '/var/tmp/buildah1276219724|ubuntu|orphaned-build-dir|directory'
  '/var/tmp/buildah1503855588|ubuntu|orphaned-build-dir|directory'
  '/var/tmp/buildah1924244745|ubuntu|orphaned-build-dir|directory'
  '/var/tmp/buildah942305812|ubuntu|orphaned-build-dir|directory'
  '/var/tmp/buildah664809777|ubuntu|orphaned-build-dir|directory'
  '/var/tmp/container_images_storage2525074258|ubuntu|orphaned-build-dir|directory'
  '/var/tmp/container_images_storage780933148|ubuntu|orphaned-build-dir|directory'
  '/tmp/router-ai-atius-go-cache-phase32|ubuntu|regenerable-go-cache|directory'
  '/tmp/router-ai-atius-image-go-cache|ubuntu|regenerable-go-cache|directory'
  '/tmp/router-ai-atius-codex-catalog-gocache|ubuntu|regenerable-go-cache|directory'
  '/tmp/router-ai-atius-phase32-final-gocache|ubuntu|regenerable-go-cache|directory'
  '/tmp/router-ai-atius-phase32-gocache|ubuntu|regenerable-go-cache|directory'
  '/tmp/router-ai-atius-go-cache-32-01|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build3938716036|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build3994217846|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build224276378|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build3032808738|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build2261881659|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build2619055287|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build2588312504|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build3926667159|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build3657123886|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build3006439396|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build4119053554|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build2107211695|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build1996314442|ubuntu|regenerable-go-cache|directory'
  '/tmp/phase48-gocache|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build1392854785|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build2084778720|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build1058590525|ubuntu|regenerable-go-cache|directory'
  '/tmp/go-build250894402|ubuntu|regenerable-go-cache|directory'
  '/tmp/tmp.Zf7X48b0cu|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.KrGbln|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.TaOJgP|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.mtUoWt|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.QEHSfm|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.Tiy7Xn|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.UetrXn|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.b6rROh|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.ctaJN7|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.rASV0n|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.R8P7Vg|ubuntu|orphaned-build-dir|directory'
  '/tmp/org.chromium.Chromium.scoped_dir.4jDstO|ubuntu|orphaned-build-dir|directory'
  '/tmp/atius-talk-audit-chrome|ubuntu|orphaned-build-dir|directory'
  '/tmp/atius-talk-chrome-profile|ubuntu|orphaned-build-dir|directory'
  '/tmp/home-proxy-graphify.ozhohG|ubuntu|orphaned-build-dir|directory'
  '/tmp/tmp.yLFoTsq5DL|ubuntu|orphaned-build-dir|directory'
  '/home/ubuntu/.cache/router-ai-atius|ubuntu|regenerable-project-cache|directory'
  '/home/ubuntu/.cache/camoufox|ubuntu|regenerable-package-cache|directory'
  '/home/ubuntu/.cache/uv|ubuntu|regenerable-package-cache|directory'
  '/home/ubuntu/.cache/chrome-devtools-mcp|ubuntu|regenerable-package-cache|directory'
  '/home/ubuntu/.cache/typescript|ubuntu|regenerable-package-cache|directory'
  '/home/ubuntu/.npm/_cacache|ubuntu|regenerable-package-cache|directory'
  '/root/.npm/_cacache|root|regenerable-package-cache|directory'
  '/root/.cache/puppeteer|root|regenerable-package-cache|directory'
  '/root/.cache/pip|root|regenerable-package-cache|directory'
  '/root/.cache/node-gyp|root|regenerable-package-cache|directory'
  '/home/ubuntu/.bun/install/cache|ubuntu|regenerable-package-cache|directory'
)

forbidden() {
  case "$1" in
    *'*'*|*'?'*|*'['*|/var/lib/rancher/k3s|/var/lib/rancher/k3s/*|/home/ubuntu/.local/share/containers|/home/ubuntu/.local/share/containers/*|/var/backups|/var/backups/*|*/data|*/data/*|*/logs|*/logs/*|*secret*) return 0 ;;
    *) return 1 ;;
  esac
}

self_test() {
  local record path owner policy method
  for record in "${ALLOWLIST[@]}"; do
    IFS='|' read -r path owner policy method <<< "$record"
    [[ "$path" = /* ]] || die 'relative path in allowlist'
    ! forbidden "$path" || die "forbidden path $path"
    [[ "$owner" =~ ^(ubuntu|root)$ ]] || die "invalid owner for $path"
    [[ "$policy" =~ ^(orphaned-build-dir|regenerable-(go|project|package)-cache)$ ]] || die "invalid policy for $path"
    [ "$method" = directory ] || die "invalid method for $path"
  done
  forbidden '/tmp/*.log' || die 'glob accepted'
  forbidden '/var/lib/rancher/k3s/storage' || die 'k3s data accepted'
  lsof_result_ok 1 0 0
  if (lsof_result_ok 2 0 0) 2>/dev/null; then die 'lsof execution error accepted'; fi
  if (lsof_result_ok 1 0 1) 2>/dev/null; then die 'lsof stderr accepted'; fi
  echo 'cleanup self-test: PASS'
}

lsof_result_ok() {
  local rc="$1" stdout_size="$2" stderr_size="$3"
  [ "$stderr_size" -eq 0 ] || return 20
  if [ "$rc" -eq 1 ] && [ "$stdout_size" -eq 0 ]; then return 0; fi
  if [ "$rc" -eq 0 ] && [ "$stdout_size" -gt 0 ]; then return 10; fi
  return 20
}

assert_no_open_descendants() {
  local target="$1" out err rc
  out="$(mktemp /dev/shm/phase29-lsof-out.XXXXXX)"
  err="$(mktemp /dev/shm/phase29-lsof-err.XXXXXX)"
  set +e
  # Redirects intentionally stay unprivileged in the caller-owned tmpfs files.
  # shellcheck disable=SC2024
  sudo -n lsof -w +D "$target" >"$out" 2>"$err"
  rc=$?
  set -e
  if lsof_result_ok "$rc" "$(stat -c %s "$out")" "$(stat -c %s "$err")"; then
    rm -f "$out" "$err"
    return 0
  fi
  rm -f "$out" "$err"
  die "path has open descendants: $target"
}

prepare_evidence_dir() {
  local parent="$HOME/.local/state/router-ai-atius"
  if [ -L "$HOME/.local" ] || [ -L "$HOME/.local/state" ]; then die 'state path contains a symlink'; fi
  install -d -m 0700 "$parent"
  [ ! -L "$parent" ] || die 'evidence parent is a symlink'
  install -d -m 0700 "$evidence_root"
  [ ! -L "$evidence_root" ] || die 'evidence root is a symlink'
  [ "$(realpath -e "$evidence_root")" = "$evidence_root" ] || die 'evidence root is not canonical'
  case "$evidence_dir" in
    "$evidence_root"/run-[A-Za-z0-9._-]*) ;;
    *) die "evidence dir must be $evidence_root/run-<id>" ;;
  esac
  if [ ! -e "$evidence_dir" ]; then mkdir -m 0700 "$evidence_dir"; fi
  if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then die 'invalid evidence directory'; fi
  [ "$(realpath -e "$evidence_dir")" = "$evidence_dir" ] || die 'evidence directory is not canonical'
  [ "$(stat -c '%U:%a' "$evidence_dir")" = "$(id -un):700" ] || die 'invalid evidence directory owner/mode'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --resume) resume=true ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --baseline-free-bytes) baseline_free_bytes="${2:?}"; shift ;;
    --self-test) self_test; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

self_test >/dev/null
if [ "$mode" != live ]; then
  printf 'cleanup candidate: %s\n' "${ALLOWLIST[@]}"
  echo 'cleanup dry-run: no files removed'
  exit 0
fi

[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[ "${PHASE29_CLEANUP_CONFIRM:-}" = DELETE_ONLY_LITERAL_ALLOWLIST ] || die 'missing exact cleanup confirmation'
read -r quota period <<< "$(cpu_max_value)"
if [ "$quota" = max ] || [ "$period" -le 0 ] || [ $((quota * 10)) -gt $((period * 8)) ]; then
  die "cpu.max exceeds 800m: $quota $period"
fi
[ -n "$evidence_dir" ] || die '--evidence-dir required'
prepare_evidence_dir

attempt_before="$(df -B1 --output=avail / | tail -1 | tr -d ' ')"
if [ -z "$baseline_free_bytes" ]; then baseline_free_bytes="$attempt_before"; fi
[[ "$baseline_free_bytes" =~ ^[0-9]+$ ]] || die 'baseline free bytes must be an integer'
before="$baseline_free_bytes"
items="$evidence_dir/cleanup-items.jsonl"
if $resume; then
  if [ ! -f "$items" ] || [ -L "$items" ]; then die 'resume requires regular cleanup item evidence'; fi
  removed_sum="$(jq -s 'map(.bytes) | add // 0' "$items")"
else
  if [ -e "$items" ] || [ -L "$items" ]; then die 'cleanup item evidence already exists'; fi
  (set -o noclobber; : > "$items") 2>/dev/null || die 'cannot create cleanup item evidence safely'
  chmod 600 "$items"
  removed_sum=0
fi

for record in "${ALLOWLIST[@]}"; do
  IFS='|' read -r path owner policy method <<< "$record"
  if [ "$owner" = root ]; then
    sudo -n test -e "$path" || continue
    [ "$(sudo -n realpath -e "$path")" = "$path" ] || die "realpath mismatch $path"
    [ "$(sudo -n stat -c %U "$path")" = "$owner" ] || die "owner mismatch $path"
    sudo -n test -d "$path" || die "not a directory $path"
  else
    [ -e "$path" ] || continue
    [ "$(realpath -e "$path")" = "$path" ] || die "realpath mismatch $path"
    [ "$(stat -c %U "$path")" = "$owner" ] || die "owner mismatch $path"
    [ -d "$path" ] || die "not a directory $path"
  fi
  [ "$(findmnt -rn -o TARGET -T "$path" | tail -1)" = / ] || die "unexpected mount for $path"
  if findmnt -rn -o TARGET | grep -Fxq "$path"; then die "path is a mountpoint: $path"; fi
  assert_no_open_descendants "$path"
  if [ "$owner" = root ]; then bytes="$(sudo -n du -sb "$path" | awk '{print $1}')"; else bytes="$(du -sb "$path" | awk '{print $1}')"; fi
  assert_no_open_descendants "$path"
  if [ "$owner" = root ]; then
    sudo -n chmod -R u+w "$path"
    sudo -n rm -rf -- "$path"
  else
    chmod -R u+w "$path"
    rm -rf -- "$path"
  fi
  [ ! -e "$path" ] || die "path still exists after removal: $path"
  removed_sum=$((removed_sum + bytes))
  printf '{"path":"%s","owner":"%s","policy":"%s","bytes":%s}\n' "$path" "$owner" "$policy" "$bytes" >> "$items"
done

sync
after="$(df -B1 --output=avail / | tail -1 | tr -d ' ')"
reclaimed=$((after - before))
if [ "$removed_sum" -lt "$reclaimed" ]; then reclaimed="$removed_sum"; fi
free_percent=$((100 - $(df -P / | awk 'NR==2 {gsub(/%/,"",$5); print $5}')))
cpu="$(cpu_max_value)"
generated_at_epoch="$(date +%s)"
cluster_uid="$(sudo -n k3s kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')"
status=no-go
if [ "$reclaimed" -ge 21474836480 ] && [ "$free_percent" -ge 25 ]; then status=pending-stability; fi
if [ -e "$evidence_dir/cleanup.json" ] || [ -L "$evidence_dir/cleanup.json" ]; then die 'cleanup evidence already exists'; fi
(set -o noclobber; : > "$evidence_dir/cleanup.json") 2>/dev/null || die 'cannot create cleanup evidence safely'
jq -n \
  --arg status "$status" --arg cpu_max "$cpu" --arg cluster_uid "$cluster_uid" \
  --argjson reclaimed_bytes "$reclaimed" --argjson free_percent "$free_percent" \
  --argjson removed_sum_bytes "$removed_sum" --argjson generated_at_epoch "$generated_at_epoch" \
  --argjson before_free_bytes "$before" --argjson attempt_before_free_bytes "$attempt_before" --argjson after_free_bytes "$after" \
  '{status:$status,reclaimed_bytes:$reclaimed_bytes,removed_sum_bytes:$removed_sum_bytes,free_percent:$free_percent,stable_seconds:0,cpu_max:$cpu_max,cluster_uid:$cluster_uid,generated_at_epoch:$generated_at_epoch,before_free_bytes:$before_free_bytes,attempt_before_free_bytes:$attempt_before_free_bytes,after_free_bytes:$after_free_bytes}' \
  > "$evidence_dir/cleanup.json"
chmod 600 "$evidence_dir/cleanup.json"
[ "$status" = pending-stability ] || die 'recovery gates not met; no-go evidence written'
