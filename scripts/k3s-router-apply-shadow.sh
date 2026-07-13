#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
mode=dry-run
stage=""
cleanup=""
bootstrap=""
restore=""

die() {
  echo "shadow apply failed: $*" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --stage) stage="${2:?}"; shift ;;
    --cleanup-evidence) cleanup="${2:?}"; shift ;;
    --bootstrap-evidence) bootstrap="${2:?}"; shift ;;
    --restore-evidence) restore="${2:?}"; shift ;;
    --self-test)
      [ "${RUN_K3S_ROUTER_APPLY:-0}" != 1 ] || die 'unsafe test environment'
      echo 'apply self-test: PASS'
      exit 0
      ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

scripts/k3s-router-validate-manifests.sh
if [ "$mode" != live ]; then
  echo 'shadow apply dry-run: PASS; use --stage postgres or --stage runtime for live work'
  exit 0
fi

[ "$stage" = postgres ] || [ "$stage" = runtime ] || die '--stage must be postgres or runtime'
[ "${RUN_K3S_ROUTER_APPLY:-0}" = 1 ] || die '--live requires RUN_K3S_ROUTER_APPLY=1'
[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[ "${PHASE29_APPLY_CONFIRM:-}" = APPLY_CLUSTERIP_SHADOW_ONLY ] || die 'missing exact apply confirmation'
read -r quota period < /sys/fs/cgroup/cpu.max
if [ "$quota" = max ] || [ "$period" -le 0 ] || [ $((quota * 10)) -gt $((period * 8)) ]; then die "cpu.max exceeds 800m: $quota $period"; fi
if [ ! -f "$cleanup" ] || [ -L "$cleanup" ]; then die 'valid cleanup evidence required'; fi
if [ ! -f "$bootstrap" ] || [ -L "$bootstrap" ]; then die 'valid bootstrap evidence required'; fi

PHASE29_LIVE=1 PHASE29_REQUIRE_STABLE_SECONDS=300 scripts/k3s-router-preflight.sh --live \
  --require-cleanup-evidence "$cleanup" --require-bootstrap-evidence "$bootstrap"
PHASE29_LIVE=1 scripts/k3s-router-validate-manifests.sh --server

if [ "$stage" = postgres ]; then
  sudo -n k3s kubectl apply -f k8s/router-ai-atius/namespace.yaml >/dev/null
  sudo -n k3s kubectl apply -f k8s/router-ai-atius/configmap.yaml >/dev/null
  sudo -n k3s kubectl apply -f k8s/router-ai-atius/postgres.yaml >/dev/null
  sudo -n k3s kubectl -n router-ai-atius wait --for=jsonpath='{.status.phase}'=Bound pvc/router-ai-atius-postgres-data --timeout=15m
  sudo -n k3s kubectl -n router-ai-atius rollout status statefulset/router-ai-atius-postgres --timeout=20m
  sudo -n k3s kubectl -n router-ai-atius get endpoints router-ai-atius-postgres -o json | jq -e '.subsets | any(.addresses | length > 0)' >/dev/null
  echo 'shadow postgres stage: PASS'
  exit 0
fi

if [ ! -f "$restore" ] || [ -L "$restore" ]; then die 'runtime stage requires restore evidence'; fi
jq -e '.status == "go" and .restore_passed == true' "$restore" >/dev/null || die 'restore evidence is not green'
cluster_uid="$(sudo -n k3s kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')"
[ "$(jq -r '.cluster_uid' "$restore")" = "$cluster_uid" ] || die 'restore evidence belongs to another cluster'
generated="$(jq -r '.generated_at_epoch' "$restore")"; now="$(date +%s)"
if [ "$generated" -gt "$now" ] || [ $((now - generated)) -gt 3600 ]; then die 'restore evidence is stale or future-dated'; fi

sudo -n k3s kubectl apply -f k8s/router-ai-atius/redis.yaml >/dev/null
sudo -n k3s kubectl apply -f k8s/router-ai-atius/router.yaml >/dev/null
sudo -n k3s kubectl -n router-ai-atius wait --for=jsonpath='{.status.phase}'=Bound pvc/router-ai-atius-data --timeout=15m
sudo -n k3s kubectl -n router-ai-atius rollout status deployment/router-ai-atius-redis --timeout=15m
sudo -n k3s kubectl -n router-ai-atius rollout status deployment/router-ai-atius --timeout=30m
for service in router-ai-atius-redis router-ai-atius; do
  sudo -n k3s kubectl -n router-ai-atius get endpoints "$service" -o json | jq -e '.subsets | any(.addresses | length > 0)' >/dev/null
done
echo 'shadow runtime stage: PASS'
