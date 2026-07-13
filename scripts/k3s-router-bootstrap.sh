#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mode=dry-run; cleanup=""; evidence_dir=""
evidence_root="${PHASE29_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase29}"
die() { echo "bootstrap failed: $*" >&2; exit 1; }
cpu_max_value() { local cgroup file; cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"; file="/sys/fs/cgroup${cgroup}/cpu.max"; [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"; cat "$file"; }
self_test() {
  [ "$(printf '%s\n' POSTGRES_PASSWORD REDIS_PASSWORD SESSION_SECRET | sort | paste -sd, -)" = 'POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET' ] || die 'secret key contract changed'
  echo 'bootstrap self-test: PASS'
}
while [ "$#" -gt 0 ]; do case "$1" in --live) mode=live;; --cleanup-evidence) cleanup="${2:?}"; shift;; --evidence-dir) evidence_dir="${2:?}"; shift;; --self-test) self_test; exit 0;; *) die "unknown argument: $1";; esac; shift; done
self_test >/dev/null
[ "$mode" = live ] || { echo 'bootstrap dry-run: label, Secret, image import and manifest remain unchanged'; exit 0; }
[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[ "${PHASE29_BOOTSTRAP_CONFIRM:-}" = LABEL_SECRET_IMAGE_AFTER_GREEN ] || die 'missing exact bootstrap confirmation'
read -r quota period <<< "$(cpu_max_value)"; if [ "$quota" = max ] || [ "$period" -le 0 ] || [ $((quota * 10)) -gt $((period * 8)) ]; then die "cpu.max exceeds 800m: $quota $period"; fi
[ -f "$cleanup" ] || die 'cleanup evidence missing'; grep -Eq '"status"[[:space:]]*:[[:space:]]*"go"' "$cleanup" || die 'cleanup evidence is not green'
[ -n "$evidence_dir" ] || die '--evidence-dir required'
case "$evidence_dir" in "$evidence_root"/run-[A-Za-z0-9._-]*) ;; *) die "invalid evidence directory" ;; esac
if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then die 'evidence directory missing or symlinked'; fi
[ "$(realpath -e "$evidence_dir")" = "$evidence_dir" ] || die 'evidence directory is not canonical'
[ "$(stat -c '%U:%a' "$evidence_dir")" = "$(id -un):700" ] || die 'invalid evidence directory owner/mode'
PHASE29_LIVE=1 PHASE29_REQUIRE_STABLE_SECONDS=300 scripts/k3s-router-preflight.sh --live --require-cleanup-evidence "$cleanup"

tmp="$(mktemp -d /dev/shm/phase29-bootstrap.XXXXXX)"; chmod 700 "$tmp"; trap 'rm -rf "$tmp"' EXIT INT TERM
env_file="$tmp/runtime.env"; umask 077; set +x
helper="$HOME/.local/bin/atius-vault-env"; [ -x "$helper" ] || die 'Vault helper unavailable'
# The trusted helper emits only shell-quoted export statements from Vault.
# shellcheck disable=SC1090
source <("$helper" router-ai-atius)
for key in POSTGRES_PASSWORD REDIS_PASSWORD SESSION_SECRET; do [ -n "${!key:-}" ] || die "Vault did not provide $key"; printf '%s=%s\n' "$key" "${!key}" >> "$env_file"; done
chmod 600 "$env_file"

sudo -n k3s kubectl apply -f k8s/router-ai-atius/namespace.yaml >/dev/null
sudo -n k3s kubectl label node atius-srv-1 atius.com.br/router-ai-atius-node=true --overwrite
nodes="$(sudo -n k3s kubectl get nodes -l atius.com.br/router-ai-atius-node=true -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')"
[ "$nodes" = atius-srv-1 ] || die "dedicated label is not exclusive: $nodes"
sudo -n k3s kubectl -n router-ai-atius create secret generic router-ai-atius-secrets --from-env-file="$env_file" --dry-run=client -o yaml | sudo -n k3s kubectl apply -f - >/dev/null
keys="$(sudo -n k3s kubectl -n router-ai-atius get secret router-ai-atius-secrets -o json | jq -r '.data | keys[]' | paste -sd, -)"
[ "$keys" = 'POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET' ] || die "unexpected Secret keys: $keys"
rm -f "$env_file"
unset POSTGRES_PASSWORD REDIS_PASSWORD SESSION_SECRET

source_ref="$(podman inspect router-ai-atius --format '{{.Config.Image}}')"; [ -n "$source_ref" ] || die 'source image reference missing'
source_image="$(podman image inspect "$source_ref" --format '{{.Id}}')"; [ -n "$source_image" ] || die 'source image ID missing'
arch="$(podman image inspect "$source_ref" --format '{{.Architecture}}')"; [ "$arch" = arm64 ] || die "source image is $arch, not arm64"
digest="$(podman image inspect "$source_ref" --format '{{.Digest}}' | sed 's/^sha256://')"; [[ "$digest" =~ ^[0-9a-f]{64}$ ]] || die 'source digest unavailable'
immutable="ghcr.io/giovannimnz/router-ai-atius@sha256:$digest"; archive="$tmp/router.tar"
podman save -o "$archive" "$source_ref"; archive_sha="$(sha256sum "$archive" | awk '{print $1}')"; sudo -n k3s ctr -n k8s.io images import "$archive" >/dev/null
sed -i -E "s#image: ghcr.io/giovannimnz/router-ai-atius@sha256:[^[:space:]]+#image: $immutable#" k8s/router-ai-atius/router.yaml
sudo -n k3s ctr -n k8s.io images ls -q | grep -Fxq 'ghcr.io/giovannimnz/router-ai-atius:latest' || die 'source image reference not imported'
sudo -n k3s ctr -n k8s.io images tag --force 'ghcr.io/giovannimnz/router-ai-atius:latest' "$immutable" >/dev/null
sudo -n k3s ctr -n k8s.io images ls -q | grep -Fxq "$immutable" || die 'exact immutable reference not found in containerd'
cpu="$(cpu_max_value)"; read -r q p <<< "$cpu"; if [ "$q" = max ] || [ "$p" -le 0 ] || [ $((q * 10)) -gt $((p * 8)) ]; then die "cpu.max exceeds 800m: $cpu"; fi
manifest_hash="$(sha256sum k8s/router-ai-atius/*.yaml | sha256sum | awk '{print $1}')"
cluster_uid="$(sudo -n k3s kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')"
generated_at_epoch="$(date +%s)"
if [ -e "$evidence_dir/bootstrap.json" ] || [ -L "$evidence_dir/bootstrap.json" ]; then die 'bootstrap evidence already exists'; fi
(set -o noclobber; : > "$evidence_dir/bootstrap.json") 2>/dev/null || die 'cannot create bootstrap evidence safely'
jq -n --arg source_image_id "$source_image" --arg archive_sha256 "$archive_sha" \
  --arg manifest_digest "sha256:$digest" --arg image_ref "$immutable" --arg cpu_max "$cpu" \
  --arg manifest_sha256 "$manifest_hash" --arg cluster_uid "$cluster_uid" \
  --argjson generated_at_epoch "$generated_at_epoch" \
  '{status:"go",exclusive_node:"atius-srv-1",secret_keys:"POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET",source_image_id:$source_image_id,archive_sha256:$archive_sha256,manifest_digest:$manifest_digest,image_ref:$image_ref,digest_match:true,cpu_max:$cpu_max,manifest_sha256:$manifest_sha256,cluster_uid:$cluster_uid,generated_at_epoch:$generated_at_epoch}' \
  > "$evidence_dir/bootstrap.json"
chmod 600 "$evidence_dir/bootstrap.json"
echo 'bootstrap live: PASS'
