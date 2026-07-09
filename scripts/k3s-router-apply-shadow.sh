#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [ "${RUN_K3S_ROUTER_APPLY:-}" != "1" ]; then
  echo "Set RUN_K3S_ROUTER_APPLY=1 to perform a real shadow apply." >&2
  exit 1
fi

scripts/k3s-router-validate-manifests.sh

ts="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="backups/k3s-router-shadow-${ts}"
mkdir -p "$backup_dir"

sudo -n k3s kubectl get ns router-ai-atius -o yaml > "${backup_dir}/namespace.yaml" 2>/dev/null || true
sudo -n k3s kubectl -n router-ai-atius get all,pvc,secret,configmap -o yaml > "${backup_dir}/resources.yaml" 2>/dev/null || true

sudo -n k3s kubectl apply -f k8s/router-ai-atius/
sudo -n k3s kubectl -n router-ai-atius rollout status statefulset/router-ai-atius-postgres --timeout=30m
sudo -n k3s kubectl -n router-ai-atius rollout status deployment/router-ai-atius-redis --timeout=15m
sudo -n k3s kubectl -n router-ai-atius rollout status deployment/router-ai-atius --timeout=30m
sudo -n k3s kubectl -n router-ai-atius get pods,svc,pvc -o wide
