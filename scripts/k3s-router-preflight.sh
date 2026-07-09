#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

section() {
  printf '\n== %s ==\n' "$1"
}

run_or_warn() {
  local desc="$1"
  shift
  if ! "$@"; then
    printf 'WARN: %s failed\n' "$desc" >&2
    return 0
  fi
}

section "graphify status"
node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status

section "router status"
bin/clianything status

section "providers"
bin/clianything providers --all

section "podman pod ps"
podman pod ps

section "podman ps"
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'

section "k3s nodes"
sudo -n k3s kubectl get nodes -o wide

section "k3s top nodes"
run_or_warn "metrics API unavailable" sudo -n k3s kubectl top nodes

section "k3s storage"
sudo -n k3s kubectl get storageclass,pv,pvc -A -o wide

section "k3s ingress"
sudo -n k3s kubectl get ingressclass,ingress -A -o wide

section "k3s events"
sudo -n k3s kubectl get events -A --sort-by=.lastTimestamp | tail -120 || true
