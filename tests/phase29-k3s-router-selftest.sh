#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "$0")/.." && pwd)"; cd "$repo_root"
fail() { echo "FAIL: $*" >&2; exit 1; }
expect_fail() { local d="$1"; shift; if "$@" >/dev/null 2>&1; then fail "$d unexpectedly succeeded"; fi; }
scripts/k3s-router-validate-manifests.sh
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
cp -a k8s/router-ai-atius "$tmp/manifests"
sed -i 's/type: ClusterIP/type: NodePort/' "$tmp/manifests/router.yaml"
expect_fail NodePort env PHASE29_MANIFEST_DIR="$tmp/manifests" scripts/k3s-router-validate-manifests.sh
cp k8s/router-ai-atius/router.yaml "$tmp/manifests/router.yaml"
sed -i '/storageClassName: local-path/d' "$tmp/manifests/postgres.yaml"
expect_fail missing-local-path env PHASE29_MANIFEST_DIR="$tmp/manifests" scripts/k3s-router-validate-manifests.sh
cp k8s/router-ai-atius/postgres.yaml "$tmp/manifests/postgres.yaml"
sed -i '0,/requiredDuringSchedulingIgnoredDuringExecution/s//preferredDuringSchedulingIgnoredDuringExecution/' "$tmp/manifests/redis.yaml"
expect_fail preferred-affinity env PHASE29_MANIFEST_DIR="$tmp/manifests" scripts/k3s-router-validate-manifests.sh
cp k8s/router-ai-atius/redis.yaml "$tmp/manifests/redis.yaml"
sed -i -E 's/@sha256:[0-9a-f]{64}/@sha256:abc/' "$tmp/manifests/router.yaml"
expect_fail invalid-digest env PHASE29_MANIFEST_DIR="$tmp/manifests" scripts/k3s-router-validate-manifests.sh
cp k8s/router-ai-atius/router.yaml "$tmp/manifests/router.yaml"
sed -i 's/imagePullPolicy: Never/imagePullPolicy: IfNotPresent/' "$tmp/manifests/router.yaml"
expect_fail pull-policy env PHASE29_MANIFEST_DIR="$tmp/manifests" scripts/k3s-router-validate-manifests.sh
cp k8s/router-ai-atius/router.yaml "$tmp/manifests/router.yaml"
sed -i '0,/namespace: router-ai-atius/s//namespace: default/' "$tmp/manifests/router.yaml"
expect_fail wrong-namespace env PHASE29_MANIFEST_DIR="$tmp/manifests" scripts/k3s-router-validate-manifests.sh
cp k8s/router-ai-atius/router.yaml "$tmp/manifests/router.yaml"
expect_fail cleanup-gate scripts/k3s-router-cleanup.sh --live --evidence-dir "$tmp"
expect_fail bootstrap-gate scripts/k3s-router-bootstrap.sh --live --cleanup-evidence "$tmp/cleanup.json" --evidence-dir "$tmp"
expect_fail apply-gate scripts/k3s-router-apply-shadow.sh --live --cleanup-evidence "$tmp/cleanup.json" --bootstrap-evidence "$tmp/bootstrap.json"
expect_fail cleanup-evidence-path env PHASE29_EXECUTE=1 PHASE29_CLEANUP_CONFIRM=DELETE_ONLY_LITERAL_ALLOWLIST scripts/k3s-router-cleanup.sh --live --evidence-dir "$tmp/outside-root"
scripts/k3s-router-cleanup.sh --self-test
scripts/k3s-router-preflight.sh --self-test
scripts/k3s-router-bootstrap.sh --self-test
scripts/k3s-router-apply-shadow.sh --self-test
echo "phase29 k3s router self-tests: PASS"
