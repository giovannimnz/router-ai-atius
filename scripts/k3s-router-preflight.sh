#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mode=dry-run; stable_seconds="${PHASE29_REQUIRE_STABLE_SECONDS:-300}"; cleanup=""; bootstrap=""
die() { echo "preflight failed: $*" >&2; exit 1; }
cpu_max_value() { local cgroup file; cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"; file="/sys/fs/cgroup${cgroup}/cpu.max"; [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"; cat "$file"; }
quota_ok() { local q p; read -r q p <<< "$1"; if ! [[ "$q" =~ ^[0-9]+$ ]] || ! [[ "$p" =~ ^[0-9]+$ ]]; then die "cpu.max is malformed: $1"; fi; if [ "$p" -le 0 ] || [ $((q * 10)) -gt $((p * 8)) ]; then die "cpu.max exceeds 800m: $1"; fi; }
minimum_free_bytes=34359738368
free_space_ok() { local free_bytes="$1"; [ "$free_bytes" -ge "$minimum_free_bytes" ] || die 'root filesystem has less than 32 GiB free'; }
evidence_cluster_ok() {
  local file="$1" expected_uid
  expected_uid="$(sudo -n k3s kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')"
  [ "$(jq -r '.cluster_uid' "$file")" = "$expected_uid" ] || die "$file belongs to another cluster"
}
evidence_fresh_ok() {
  local file="$1" now generated
  now="$(date +%s)"; generated="$(jq -r '.generated_at_epoch' "$file")"
  [[ "$generated" =~ ^[0-9]+$ ]] || die "$file generated_at_epoch is not an integer"
  if [ "$generated" -gt "$now" ] || [ $((now - generated)) -gt 3600 ]; then die "$file is stale or future-dated"; fi
}
live_state_ok() {
  local free_bytes node_json
  free_bytes="$(df -B1 --output=avail / | tail -1 | tr -d ' ')"
  [[ "$free_bytes" =~ ^[0-9]+$ ]] || die 'root filesystem free bytes are not numeric'
  free_space_ok "$free_bytes"
  node_json="$(sudo -n k3s kubectl get node atius-srv-1 -o json)"
  jq -e 'any(.status.conditions[]; .type == "DiskPressure" and .status == "False")' <<< "$node_json" >/dev/null || die 'DiskPressure is not False'
  jq -e 'all(.spec.taints[]?; .key != "node.kubernetes.io/disk-pressure")' <<< "$node_json" >/dev/null || die 'DiskPressure taint present'
}
while [ "$#" -gt 0 ]; do case "$1" in
  --live) mode=live;;
  --require-cleanup-evidence) cleanup="${2:?}"; shift;;
  --require-bootstrap-evidence) bootstrap="${2:?}"; shift;;
  --self-test) quota_ok '80000 100000'; quota_ok '40000 50000'; free_space_ok 34359738368; if (free_space_ok 34359738367) 2>/dev/null; then die 'free space below 32 GiB accepted'; fi; if (quota_ok '80001 100000') 2>/dev/null; then die 'quota above 800m accepted'; fi; if (quota_ok 'max 100000') 2>/dev/null; then die 'unbounded quota accepted'; fi; if (quota_ok 'now 100000') 2>/dev/null; then die 'non-numeric quota accepted'; fi; echo 'preflight self-test: PASS'; exit 0;;
  *) die "unknown argument: $1";; esac; shift; done
scripts/k3s-router-validate-manifests.sh
[ "$mode" = live ] || { echo 'preflight dry-run: PASS (no host/cluster command executed)'; exit 0; }
[ "${PHASE29_LIVE:-0}" = 1 ] || die '--live requires PHASE29_LIVE=1'
[ "$stable_seconds" -ge 300 ] || die 'stability window must be >=300s'
quota_ok "$(cpu_max_value)"
if [ -n "$cleanup" ]; then
  [ -f "$cleanup" ] || die 'cleanup evidence missing'
  [ ! -L "$cleanup" ] || die 'cleanup evidence cannot be a symlink'
  # Cleanup is historical reclaim evidence. Current safety is independently
  # proven by the five-minute space/DiskPressure window below.
  evidence_cluster_ok "$cleanup"
  jq -e '(.status == "pending-stability" or .status == "go") and .reclaimed_bytes >= 21474836480 and .free_percent >= 25' "$cleanup" >/dev/null || die 'cleanup evidence is not eligible for stability check'
  quota_ok "$(jq -r '.cpu_max' "$cleanup")"
fi
if [ -n "$bootstrap" ]; then
  [ -f "$bootstrap" ] || die 'bootstrap evidence missing'
  [ ! -L "$bootstrap" ] || die 'bootstrap evidence cannot be a symlink'
  evidence_cluster_ok "$bootstrap"
  evidence_fresh_ok "$bootstrap"
  jq -e '.status == "go" and .exclusive_node == "atius-srv-1" and .secret_keys == "POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET" and .digest_match == true' "$bootstrap" >/dev/null || die 'bootstrap evidence invalid'
  manifest_hash="$(sha256sum k8s/router-ai-atius/*.yaml | sha256sum | awk '{print $1}')"
  [ "$(jq -r '.manifest_sha256' "$bootstrap")" = "$manifest_hash" ] || die 'bootstrap evidence does not match manifests'
  nodes="$(sudo -n k3s kubectl get nodes -l atius.com.br/router-ai-atius-node=true -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')"
  [ "$nodes" = atius-srv-1 ] || die 'dedicated label is not exclusive'
  keys="$(sudo -n k3s kubectl -n router-ai-atius get secret router-ai-atius-secrets -o json | jq -r '.data | keys[]' | paste -sd, -)"
  [ "$keys" = 'POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET' ] || die 'live Secret key contract differs'
  image="$(python3 -c 'import yaml; d=list(yaml.safe_load_all(open("k8s/router-ai-atius/router.yaml"))); print(next(x for x in d if x and x.get("kind")=="Deployment")["spec"]["template"]["spec"]["containers"][0]["image"])')"
  sudo -n k3s ctr -n k8s.io images ls -q | grep -Fxq "$image" || die 'exact router image reference is absent from containerd'
fi
live_state_ok
end=$((SECONDS + stable_seconds))
while [ "$SECONDS" -lt "$end" ]; do
  live_state_ok
  remaining=$((end - SECONDS))
  [ "$remaining" -gt 0 ] || break
  if [ "$remaining" -gt 30 ]; then sleep 30; else sleep "$remaining"; fi
done
live_state_ok
if [ -n "$cleanup" ]; then
  tmp="${cleanup}.tmp"
  jq --argjson seconds "$stable_seconds" '.status = "go" | .stable_seconds = $seconds' "$cleanup" > "$tmp"
  chmod 600 "$tmp"
  mv "$tmp" "$cleanup"
fi
echo 'preflight live: PASS'
