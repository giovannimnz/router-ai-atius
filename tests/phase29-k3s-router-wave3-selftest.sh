#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

expect_message() {
  local label="$1" message="$2"
  shift 2
  local output rc
  set +e
  output="$("$@" 2>&1)"
  rc=$?
  set -e
  [ "$rc" -ne 0 ] || fail "$label unexpectedly succeeded"
  grep -Fq -- "$message" <<< "$output" || fail "$label did not fail with the expected gate"
}

bash -n scripts/k3s-router-apply-shadow.sh scripts/k3s-router-smoke.sh
shellcheck -x scripts/k3s-router-apply-shadow.sh scripts/k3s-router-smoke.sh
grep -Fq 'export PHASE29_SNAPSHOT_LIBRARY=1' scripts/k3s-router-apply-shadow.sh ||
  fail 'snapshot library mode is not exported for the sourced collector'
if rg -n 'shellcheck disable=SC2034' scripts/k3s-router-apply-shadow.sh >/dev/null; then
  fail 'snapshot library mode suppresses SC2034 instead of exporting the variable'
fi

scripts/k3s-router-validate-manifests.sh
scripts/k3s-router-apply-shadow.sh --self-test
scripts/k3s-router-smoke.sh --self-test

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
chmod 700 "$tmp"
expect_message strict-required '--strict is required' \
  env K3S_ROUTER_BASE_URL=http://10.43.0.10:3000 ATIUS_ROUTER_TOKEN=test-token \
  scripts/k3s-router-smoke.sh --evidence-dir "$tmp"
expect_message base-url-required 'K3S_ROUTER_BASE_URL is required' \
  env ATIUS_ROUTER_TOKEN=test-token scripts/k3s-router-smoke.sh --strict --evidence-dir "$tmp"
expect_message token-required 'ATIUS_ROUTER_TOKEN is required' \
  env K3S_ROUTER_BASE_URL=http://10.43.0.10:3000 \
  scripts/k3s-router-smoke.sh --strict --evidence-dir "$tmp"
expect_message arbitrary-catalog-override 'unknown argument: --expected-models-config' \
  scripts/k3s-router-smoke.sh --expected-models-config "$tmp/attacker.json"

cp -a k8s/router-ai-atius "$tmp/manifests-two-containers"
python3 - "$tmp/manifests-two-containers/redis.yaml" <<'PY'
import copy
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
docs = list(yaml.safe_load_all(path.read_text(encoding="utf-8")))
deployment = next(doc for doc in docs if doc and doc.get("kind") == "Deployment")
extra = copy.deepcopy(deployment["spec"]["template"]["spec"]["containers"][0])
extra["name"] = "redis-sidecar-fixture"
extra["resources"]["requests"]["cpu"] = "100m"
extra["resources"]["limits"]["cpu"] = "100m"
deployment["spec"]["template"]["spec"]["containers"].append(extra)
path.write_text("---\n".join(yaml.safe_dump(doc, sort_keys=False) for doc in docs), encoding="utf-8")
PY
expect_message pod-container-total 'total pod CPU must have requests=limits and stay at or below 500m' \
  env PHASE29_MANIFEST_DIR="$tmp/manifests-two-containers" scripts/k3s-router-validate-manifests.sh

cp -a k8s/router-ai-atius "$tmp/manifests-init-container"
python3 - "$tmp/manifests-init-container/redis.yaml" <<'PY'
import copy
import pathlib
import sys
import yaml

path = pathlib.Path(sys.argv[1])
docs = list(yaml.safe_load_all(path.read_text(encoding="utf-8")))
deployment = next(doc for doc in docs if doc and doc.get("kind") == "Deployment")
init = copy.deepcopy(deployment["spec"]["template"]["spec"]["containers"][0])
init["name"] = "redis-init-fixture"
init["resources"]["requests"]["cpu"] = "100m"
init["resources"]["limits"]["cpu"] = "100m"
deployment["spec"]["template"]["spec"]["initContainers"] = [init]
path.write_text("---\n".join(yaml.safe_dump(doc, sort_keys=False) for doc in docs), encoding="utf-8")
PY
expect_message pod-init-total 'total pod CPU must have requests=limits and stay at or below 500m' \
  env PHASE29_MANIFEST_DIR="$tmp/manifests-init-container" scripts/k3s-router-validate-manifests.sh

if rg -n '\b(systemctl|apache2ctl|apachectl|a2en(site|mod)|a2dis(site|mod))\b|\bpodman[[:space:]]+(stop|start|restart|rm|kill|exec|run|compose)\b' \
  scripts/k3s-router-apply-shadow.sh scripts/k3s-router-smoke.sh >/dev/null; then
  fail 'Wave 3 scripts contain an Apache or Podman mutation command'
fi
if rg -n '(token|ATIUS_ROUTER_TOKEN).*jq|jq.*(token|ATIUS_ROUTER_TOKEN)' \
  scripts/k3s-router-smoke.sh >/dev/null; then
  fail 'smoke evidence path appears to serialize the token'
fi

grep -Fq 'restore evidence checksum differs from canonical state' scripts/k3s-router-apply-shadow.sh ||
  fail 'runtime apply lacks canonical restore checksum enforcement'
grep -Fq 'patch_retain_by_claim_uid router-ai-atius-data' scripts/k3s-router-apply-shadow.sh ||
  fail 'runtime apply lacks router claim UID to PV Retain handling'
grep -Fq 'apply_manifest_resource PersistentVolumeClaim router-ai-atius-data' scripts/k3s-router-apply-shadow.sh ||
  fail 'router PVC is not applied independently before the workload'
grep -Fq 'shadow apply is bound to a different restore checksum' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not bind restore evidence to shadow apply evidence'
grep -Fq 'catalog is not the exact expected order' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not enforce an exact ordered public catalog'
grep -Fq 'authenticated-pre-shadow-model-catalog' scripts/k3s-router-smoke.sh ||
  fail 'smoke lacks the authenticated pre-shadow baseline contract'
grep -Fq 'catalog_baseline_sha256' scripts/k3s-router-apply-shadow.sh ||
  fail 'apply does not record the canonical catalog baseline checksum'
grep -Fq 'shadow apply is bound to a different catalog baseline checksum' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not require the same catalog baseline checksum as apply'
grep -Fq 'targetRef.uid' scripts/k3s-router-apply-shadow.sh ||
  fail 'EndpointSlice validation is not bound to pod UID'
grep -Fq 'validate_workload_chain redis' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not preserve Redis immutable workload identity'
grep -Fq 'validate_workload_chain postgres' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not preserve PostgreSQL immutable workload identity'
grep -Fq 'runtime changed or rolled out during shadow smoke requests' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not fail closed on pre/post runtime drift'
grep -Fq 'runtime_snapshot_pre_sha' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not record the pre-request runtime snapshot hash'
grep -Fq 'runtime_snapshot_post_sha' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not record the post-request runtime snapshot hash'
grep -Fq "runtime_snapshot:{sha256:\$runtime_snapshot_sha256" scripts/k3s-router-apply-shadow.sh ||
  fail 'shadow apply evidence does not record the canonical runtime snapshot hash'
grep -Fq "capture_apply_runtime_snapshot \"\$pre_publish_snapshot\" pre-publish" scripts/k3s-router-apply-shadow.sh ||
  fail 'shadow apply does not recapture runtime immediately before publication'
grep -Fq 'runtime changed before shadow apply evidence publication; GO not published' scripts/k3s-router-apply-shadow.sh ||
  fail 'shadow apply does not fail closed on pre-publication runtime drift'
grep -Fq "\$apply.runtime_snapshot.map == \$live" scripts/k3s-router-smoke.sh ||
  fail 'smoke does not require exact equality with the canonical apply snapshot'
grep -Fq ".storage == \$live.storage" scripts/k3s-router-smoke.sh ||
  fail 'smoke does not require exact PVC/PV identity equality with shadow apply'
grep -Fq 'resource_version' scripts/k3s-router-smoke.sh ||
  fail 'runtime snapshots do not preserve Kubernetes resourceVersion'
grep -Fq 'restart_count' scripts/k3s-router-smoke.sh ||
  fail 'runtime snapshots do not preserve container restart counts'
grep -Fq 'PV reclaim policy is not Retain' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not revalidate PV Retain policy'
grep -Fq 'PV claimRef is not bound by PVC UID' scripts/k3s-router-smoke.sh ||
  fail 'smoke does not revalidate PVC UID to PV binding'
grep -Fq "storage:\$runtime_post.storage" scripts/k3s-router-smoke.sh ||
  fail 'smoke.json does not propagate the post-request PVC/PV identity map'
grep -Fq "apply_runtime_snapshot_sha256:\$apply_runtime_sha256" scripts/k3s-router-smoke.sh ||
  fail 'smoke.json does not explicitly chain live storage to shadow apply evidence'
grep -Fq 'recreated PV between apply snapshots was accepted' scripts/k3s-router-apply-shadow.sh ||
  fail 'apply self-test lacks a recreated-PV drift fixture'
grep -Fq 'rollout between apply snapshots was accepted' scripts/k3s-router-apply-shadow.sh ||
  fail 'apply self-test lacks a rollout drift fixture'
grep -Fq 'ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768' scripts/k3s-router-smoke.sh ||
  fail 'smoke uses the wrong embedding dimension environment contract'
if rg -n -- '--expected-models-config|PHASE29_EXPECTED_MODELS' scripts/k3s-router-smoke.sh >/dev/null; then
  fail 'smoke still permits an arbitrary expected-catalog override'
fi
if rg -n -- '--requirepass|redis-cli -a' k8s/router-ai-atius/redis.yaml >/dev/null; then
  fail 'Redis password can still reach process argv'
fi
grep -Fq 'chmod 0600 /run/redis/redis.conf' k8s/router-ai-atius/redis.yaml ||
  fail 'Redis does not generate a protected config file'
grep -Fq 'REDISCLI_AUTH' k8s/router-ai-atius/redis.yaml ||
  fail 'Redis probes do not use REDISCLI_AUTH'

echo 'phase29 Wave 3 shell self-tests: PASS'
