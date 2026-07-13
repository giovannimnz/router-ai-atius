#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

namespace=router-ai-atius
mode=dry-run
stage=""
cleanup=""
bootstrap=""
restore=""
evidence_dir=""
evidence_root="${PHASE29_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase29}"
restore_state="$HOME/.local/state/router-ai-atius/phase29/restore-target-state.json"

die() {
  echo "shadow apply failed: $*" >&2
  exit 1
}

kube() {
  sudo -n k3s kubectl "$@"
}

cpu_max_value() {
  local cgroup file
  cgroup="$(awk -F: '$1 == "0" {print $3}' /proc/self/cgroup)"
  file="/sys/fs/cgroup${cgroup}/cpu.max"
  [ -r "$file" ] || die "cpu.max unavailable for cgroup $cgroup"
  cat "$file"
}

quota_ok() {
  local value="$1" quota period
  read -r quota period <<< "$value"
  [[ "$quota" =~ ^[0-9]+$ ]] || die "cpu.max quota is malformed: $value"
  [[ "$period" =~ ^[0-9]+$ ]] || die "cpu.max period is malformed: $value"
  [ "$period" -gt 0 ] || die "cpu.max period is invalid: $value"
  [ $((quota * 10)) -le $((period * 8)) ] || die "cpu.max exceeds 800m: $value"
}

manifest_hash() {
  sha256sum k8s/router-ai-atius/*.yaml | sha256sum | awk '{print $1}'
}

manifest_image() {
  local workload="$1" file="$2"
  python3 - "$workload" "$file" <<'PY'
import sys
import yaml

workload, path = sys.argv[1:]
for document in yaml.safe_load_all(open(path, encoding="utf-8")):
    if (
        document
        and document.get("kind") in {"Deployment", "StatefulSet"}
        and document.get("metadata", {}).get("name") == workload
    ):
        containers = document["spec"]["template"]["spec"]["containers"]
        if len(containers) != 1:
            raise SystemExit(f"{workload} must have exactly one container")
        print(containers[0]["image"])
        break
else:
    raise SystemExit(f"workload not found: {workload}")
PY
}

apply_manifest_resource() {
  local kind="$1" name="$2" file="$3"
  python3 - "$kind" "$name" "$file" <<'PY' | kube apply -f - >/dev/null
import sys
import yaml

kind, name, path = sys.argv[1:]
matches = [
    document
    for document in yaml.safe_load_all(open(path, encoding="utf-8"))
    if document
    and document.get("kind") == kind
    and document.get("metadata", {}).get("name") == name
]
if len(matches) != 1:
    raise SystemExit(f"expected one {kind}/{name}, found {len(matches)}")
yaml.safe_dump(matches[0], sys.stdout, sort_keys=False)
PY
}

require_regular_evidence() {
  local file="$1" label="$2"
  if [ ! -f "$file" ] || [ -L "$file" ]; then
    die "$label evidence must be a regular non-symlink file"
  fi
  jq -e . "$file" >/dev/null || die "$label evidence is not valid JSON"
}

require_evidence_directory() {
  [ -n "$evidence_dir" ] || die '--evidence-dir is required for the runtime stage'
  case "$evidence_dir" in
    "$evidence_root"/run-[A-Za-z0-9._-]*) ;;
    *) die 'evidence directory is outside the canonical Phase 29 run root' ;;
  esac
  if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then
    die 'evidence directory must be a real directory'
  fi
  [ "$(realpath -e "$evidence_dir")" = "$evidence_dir" ] || die 'evidence directory must be canonical'
  [ "$(stat -c %U:%a "$evidence_dir")" = "$(id -un):700" ] ||
    die 'evidence directory must be owned by the caller with mode 700'
}

validate_restore_gate() {
  local file="$1" state="$2" expected_cluster="$3" now="$4"
  local generated state_path state_sha actual_sha
  require_regular_evidence "$file" restore
  require_regular_evidence "$state" 'canonical restore state'
  [ "$(stat -c %U:%a "$file")" = "$(id -un):600" ] || die 'restore evidence owner/mode must be caller:600'
  [ "$(stat -c %U:%a "$state")" = "$(id -un):600" ] || die 'canonical restore state owner/mode must be caller:600'
  jq -e --arg cluster "$expected_cluster" '
    .status == "go" and .restore_passed == true and .cluster_uid == $cluster and
    (.generated_at_epoch | type == "number") and
    .target.node == "atius-srv-1" and .target.database == "DBRouterAiAtius" and
    .target.clean_before_restore == true and
    (.target.server_version_num | test("^17[0-9]{4}$")) and
    (.pvs | type == "array" and length >= 1 and
      all(.reclaim_policy == "Retain" and .claim_uid_matched == true)) and
    .runtime_stage.redis_applied == false and .runtime_stage.router_applied == false
  ' "$file" >/dev/null || die 'restore evidence is not a complete GO for this runtime stage'
  generated="$(jq -r '.generated_at_epoch' "$file")"
  if [ "$generated" -gt "$now" ] || [ $((now - generated)) -gt 3600 ]; then
    die 'restore evidence is stale or future-dated'
  fi
  jq -e --arg cluster "$expected_cluster" '
    .schema_version == 1 and
    .target == "router-ai-atius/DBRouterAiAtius@atius-srv-1" and
    .status == "go" and .cluster_uid == $cluster and
    (.evidence_sha256 | test("^[0-9a-f]{64}$"))
  ' "$state" >/dev/null || die 'canonical restore state is not GO for this cluster and target'
  state_path="$(jq -r '.evidence_path' "$state")"
  if [ ! -f "$state_path" ] || [ -L "$state_path" ]; then
    die 'canonical restore state points to missing evidence'
  fi
  [ "$(realpath -e "$state_path")" = "$(realpath -e "$file")" ] ||
    die 'canonical restore state points to different evidence'
  state_sha="$(jq -r '.evidence_sha256' "$state")"
  actual_sha="$(sha256sum "$file" | awk '{print $1}')"
  [ "$actual_sha" = "$state_sha" ] || die 'restore evidence checksum differs from canonical state'
}

validate_bootstrap_gate() {
  local file="$1" expected_cluster="$2" expected_manifest="$3" now="$4"
  local generated
  require_regular_evidence "$file" bootstrap
  jq -e --arg cluster "$expected_cluster" --arg manifest "$expected_manifest" '
    .status == "go" and .cluster_uid == $cluster and .exclusive_node == "atius-srv-1" and
    .manifest_sha256 == $manifest and .digest_match == true and
    (.manifest_digest | test("^sha256:[0-9a-f]{64}$")) and
    (.image_ref | test("^ghcr.io/giovannimnz/router-ai-atius@sha256:[0-9a-f]{64}$")) and
    .image_ref == ("ghcr.io/giovannimnz/router-ai-atius@" + .manifest_digest)
  ' "$file" >/dev/null || die 'bootstrap evidence is not cluster/manifest/image bound'
  generated="$(jq -r '.generated_at_epoch' "$file")"
  [[ "$generated" =~ ^[0-9]+$ ]] || die 'bootstrap generated_at_epoch is malformed'
  if [ "$generated" -gt "$now" ] || [ $((now - generated)) -gt 3600 ]; then
    die 'bootstrap evidence is stale or future-dated'
  fi
}

validate_catalog_baseline() {
  local file="$1" now="$2"
  require_regular_evidence "$file" 'catalog baseline'
  [ "$(stat -c %U:%a "$file")" = "$(id -un):600" ] ||
    die 'catalog baseline owner/mode must be caller:600'
  python3 - "$file" "$now" <<'PY'
import hashlib
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
now = int(sys.argv[2])
doc = json.loads(path.read_text(encoding="utf-8"))
models = doc.get("models")
if doc.get("schema_version") != 1 or doc.get("kind") != "authenticated-pre-shadow-model-catalog":
    raise SystemExit("catalog baseline has the wrong schema/kind")
if not isinstance(doc.get("captured_at_epoch"), int):
    raise SystemExit("catalog baseline has no capture timestamp")
if doc["captured_at_epoch"] > now or now - doc["captured_at_epoch"] > 3600:
    raise SystemExit("catalog baseline is stale or future-dated")
if not isinstance(models, list) or not models:
    raise SystemExit("catalog baseline models are absent")
if not all(isinstance(item, str) and item and item.strip() == item for item in models):
    raise SystemExit("catalog baseline contains an invalid model id")
if len(models) != len(set(models)) or "embedding-gte-v1" not in models:
    raise SystemExit("catalog baseline must be unique and include embedding-gte-v1")
expected = hashlib.sha256(("\n".join(models) + "\n").encode()).hexdigest()
if doc.get("catalog_sha256") != expected:
    raise SystemExit("catalog baseline model checksum mismatch")
PY
}

validate_workload_json() {
  local file="$1" app="$2" expected_image="$3" expected_digest expected_runtime_digest
  [[ "$expected_image" =~ @sha256:[0-9a-f]{64}$ ]] || die "$app image is not digest pinned"
  expected_digest="${expected_image##*@}"
  expected_runtime_digest="${4:-$expected_digest}"
  [[ "$expected_runtime_digest" =~ ^sha256:[0-9a-f]{64}$ ]] || die "$app runtime digest is malformed"
  jq -e --arg app "$app" --arg image "$expected_image" --arg digest "$expected_runtime_digest" '
    .items | length == 1 and
    .[0].metadata.labels["app.kubernetes.io/name"] == $app and
    (.[0].metadata.name | type == "string" and length > 0) and
    (.[0].metadata.uid | type == "string" and length > 0) and
    .[0].spec.nodeName == "atius-srv-1" and
    .[0].status.phase == "Running" and
    (.[0].status.podIP | type == "string" and length > 0) and
    ((.[0].spec.initContainers // []) | length == 0) and
    (.[0].spec.containers | length == 1) and
    .[0].spec.containers[0].image == $image and
    .[0].spec.containers[0].resources.requests.cpu == "500m" and
    .[0].spec.containers[0].resources.limits.cpu == "500m" and
    (.[0].status.containerStatuses | length == 1) and
    .[0].status.containerStatuses[0].ready == true and
    (.[0].status.containerStatuses[0].imageID | type == "string" and endswith($digest))
  ' "$file" >/dev/null || die "$app pod is not unique, Ready on srv1, exact-image, and 500m"
}

validate_service_json() {
  local file="$1" expected_name="$2" expected_port="$3" expected_app="$4"
  jq -e --arg name "$expected_name" --arg app "$expected_app" --argjson port "$expected_port" '
    .metadata.name == $name and
    .spec.type == "ClusterIP" and
    .spec.selector == {"app.kubernetes.io/name":$app} and
    (.spec.clusterIP | type == "string" and length > 0 and . != "None") and
    (.spec.clusterIPs | type == "array" and length == 1) and
    .spec.clusterIPs[0] == .spec.clusterIP and
    ((.spec.externalIPs // []) | length == 0) and
    (.spec.externalName // "" | length == 0) and
    (.spec.ports | length == 1) and .spec.ports[0].port == $port and
    (.spec.ports[0] | has("nodePort") | not)
  ' "$file" >/dev/null || die "$expected_name Service is not strict ClusterIP"
}

endpointslice_matches_workload() {
  local file="$1" service="$2" pod_name="$3" pod_uid="$4" pod_ip="$5"
  jq -e --arg service "$service" --arg pod_name "$pod_name" --arg pod_uid "$pod_uid" --arg pod_ip "$pod_ip" '
    .items | length >= 1 and
    all(.metadata.labels["kubernetes.io/service-name"] == $service) and
    ([.[].endpoints[]?] | length == 1) and
    all(.[].endpoints[]?;
      .conditions.ready == true and .nodeName == "atius-srv-1" and
      .addresses == [$pod_ip] and
      (.targetRef.apiVersion // "v1") == "v1" and .targetRef.kind == "Pod" and
      .targetRef.namespace == "router-ai-atius" and
      .targetRef.name == $pod_name and .targetRef.uid == $pod_uid)
  ' "$file" >/dev/null
}

validate_endpointslice_json() {
  endpointslice_matches_workload "$@" ||
    die "$2 EndpointSlice is not bound to the validated pod UID/IP"
}

workload_snapshot() {
  local app="$1" expected_image="$2" output="$3" expected_runtime_digest="${4:-}"
  kube -n "$namespace" get pods -l "app.kubernetes.io/name=$app" -o json > "$output"
  validate_workload_json "$output" "$app" "$expected_image" "$expected_runtime_digest"
}

workload_identity() {
  local app="$1" kind="$2" name="$3" expected_image="$4" pod_file="$5" output="$6"
  local controller_file owner_file owner_name owner_uid controller_uid kind_title
  controller_file="$tmp/${app}-controller.json"
  owner_file="$tmp/${app}-owner.json"
  kube -n "$namespace" get "$kind" "$name" -o json > "$controller_file"
  controller_uid="$(jq -r '.metadata.uid' "$controller_file")"
  if [ -z "$controller_uid" ] || [ "$controller_uid" = null ]; then die "$app controller has no UID"; fi
  jq -e --arg app "$app" --arg image "$expected_image" '
    .spec.replicas == 1 and
    .spec.selector.matchLabels == {"app.kubernetes.io/name":$app} and
    (.spec.template.spec.initContainers // [] | length) == 0 and
    (.spec.template.spec.containers | length) == 1 and
    .spec.template.spec.containers[0].name != "" and
    .spec.template.spec.containers[0].image == $image and
    .spec.template.spec.containers[0].resources.requests.cpu == "500m" and
    .spec.template.spec.containers[0].resources.limits.cpu == "500m"
  ' "$controller_file" >/dev/null || die "$app controller identity/resources differ from the approved workload"
  owner_name="$(jq -r '[.items[0].metadata.ownerReferences[]? | select(.controller == true)][0].name // ""' "$pod_file")"
  owner_uid="$(jq -r '[.items[0].metadata.ownerReferences[]? | select(.controller == true)][0].uid // ""' "$pod_file")"
  if [ -z "$owner_name" ] || [ -z "$owner_uid" ]; then die "$app pod has no controller owner"; fi
  if [ "$kind" = deployment ]; then
    kind_title=Deployment
    kube -n "$namespace" get replicaset "$owner_name" -o json > "$owner_file"
    jq -e --arg uid "$owner_uid" --arg controller_uid "$controller_uid" '
      .metadata.uid == $uid and
      ([.metadata.ownerReferences[]? | select(.controller == true and .kind == "Deployment" and .uid == $controller_uid)] | length) == 1
    ' "$owner_file" >/dev/null || die "$app ReplicaSet is not owned by the validated Deployment UID"
  else
    kind_title=StatefulSet
    [ "$owner_uid" = "$controller_uid" ] || die "$app pod owner UID differs from the StatefulSet UID"
    jq -e '[.items[0].metadata.ownerReferences[]? | select(.controller == true and .kind == "StatefulSet")] | length == 1' \
      "$pod_file" >/dev/null || die "$app pod is not owned by a StatefulSet"
  fi
  jq -n --arg app "$app" --arg kind "$kind_title" --arg name "$name" \
    --arg controller_uid "$controller_uid" --arg owner_name "$owner_name" --arg owner_uid "$owner_uid" \
    --arg pod_name "$(jq -r '.items[0].metadata.name' "$pod_file")" \
    --arg pod_uid "$(jq -r '.items[0].metadata.uid' "$pod_file")" \
    --arg pod_ip "$(jq -r '.items[0].status.podIP' "$pod_file")" \
    --arg container_name "$(jq -r '.items[0].spec.containers[0].name' "$pod_file")" \
    --arg image_ref "$(jq -r '.items[0].spec.containers[0].image' "$pod_file")" \
    --arg image_id "$(jq -r '.items[0].status.containerStatuses[0].imageID' "$pod_file")" \
    '{app:$app,controller:{kind:$kind,name:$name,uid:$controller_uid},pod_owner:{name:$owner_name,uid:$owner_uid},
      pod:{name:$pod_name,uid:$pod_uid,ip:$pod_ip},container:{name:$container_name,image_ref:$image_ref,image_id:$image_id,
      resources:{requests_cpu:"500m",limits_cpu:"500m"}}}' > "$output"
}

service_snapshot() {
  local service="$1" port="$2" service_file="$3" slice_file="$4" workload_file="$5" app="$6"
  local pod_name pod_uid pod_ip expected_image expected_runtime_digest
  expected_image="$(jq -r '.items[0].spec.containers[0].image' "$workload_file")"
  expected_runtime_digest="$(jq -r '.items[0].status.containerStatuses[0].imageID | capture("(?<digest>sha256:[0-9a-f]{64})$").digest' "$workload_file")"
  for _ in $(seq 1 60); do
    workload_snapshot "$app" "$expected_image" "$workload_file" "$expected_runtime_digest"
    kube -n "$namespace" get service "$service" -o json > "$service_file"
    validate_service_json "$service_file" "$service" "$port" "$app"
    pod_name="$(jq -r '.items[0].metadata.name' "$workload_file")"
    pod_uid="$(jq -r '.items[0].metadata.uid' "$workload_file")"
    pod_ip="$(jq -r '.items[0].status.podIP' "$workload_file")"
    kube -n "$namespace" get endpointslices.discovery.k8s.io \
      -l "kubernetes.io/service-name=$service" -o json > "$slice_file"
    if endpointslice_matches_workload "$slice_file" "$service" "$pod_name" "$pod_uid" "$pod_ip"; then
      return 0
    fi
    sleep 1
  done
  die "$service EndpointSlice did not converge to the validated pod UID/IP within 60 seconds"
}

capture_apply_runtime_snapshot() (
  local output="$1" label="$2" snapshot_tmp="$3"
  export PHASE29_SNAPSHOT_LIBRARY=1
  # Keep one canonical collector for apply and smoke so exact equality is meaningful.
  # shellcheck source=/dev/null
  source "$repo_root/scripts/k3s-router-smoke.sh"
  # shellcheck disable=SC2030
  tmp="$snapshot_tmp"
  capture_runtime_snapshot "$output" "$label"
)

validate_apply_runtime_snapshot() {
  local snapshot="$1" postgres_binding="$2" router_binding="$3"
  local router_ref="$4" redis_ref="$5" postgres_ref="$6" router_runtime_digest="$7"
  jq -e --arg router_ref "$router_ref" --arg redis_ref "$redis_ref" --arg postgres_ref "$postgres_ref" \
    --arg router_runtime_digest "$router_runtime_digest" \
    --argjson bindings "[$postgres_binding,$router_binding]" '
    def image_matches($key; $ref; $runtime_digest):
      .workloads[$key].container.image_ref == $ref and
      .workloads[$key].container.image_digest == ($ref | split("@")[-1]) and
      .workloads[$key].container.runtime_digest == $runtime_digest and
      (.workloads[$key].container.image_id | endswith($runtime_digest));
    def binding_matches($key):
      .storage[$key] as $storage |
      $storage.binding_verified == true and
      $storage.pvc.phase == "Bound" and $storage.pv.phase == "Bound" and
      $storage.pv.reclaim_policy == "Retain" and
      any($bindings[];
        .pvc == $storage.pvc.name and .pvc_uid == $storage.pvc.uid and
        .pv == $storage.pv.name and .reclaim_policy == "Retain" and .claim_uid_matched == true);
    .schema_version == 1 and
    image_matches("router"; $router_ref; $router_runtime_digest) and
    image_matches("redis"; $redis_ref; ($redis_ref | split("@")[-1])) and
    image_matches("postgres"; $postgres_ref; ($postgres_ref | split("@")[-1])) and
    binding_matches("router") and binding_matches("postgres")
  ' "$snapshot" >/dev/null || die 'canonical runtime snapshot differs from approved image or PVC/PV binding identity'
}

runtime_snapshots_equal() {
  local before="$1" after="$2" before_sha after_sha
  before_sha="$(sha256sum "$before" | awk '{print $1}')"
  after_sha="$(sha256sum "$after" | awk '{print $1}')"
  [ "$before_sha" = "$after_sha" ] && cmp -s "$before" "$after"
}

patch_retain_by_claim_uid() {
  local claim="$1" tmp_dir="$2" pvc_file pv_list pv_name pvc_uid readback
  pvc_file="$tmp_dir/pvc-$claim.json"
  pv_list="$tmp_dir/pvs-$claim.json"
  kube -n "$namespace" get pvc "$claim" -o json > "$pvc_file"
  jq -e '.status.phase == "Bound" and (.metadata.uid | length > 0)' "$pvc_file" >/dev/null ||
    die "$claim is not Bound with a UID"
  pvc_uid="$(jq -r '.metadata.uid' "$pvc_file")"
  kube get pv -o json > "$pv_list"
  mapfile -t matches < <(jq -r --arg ns "$namespace" --arg claim "$claim" --arg uid "$pvc_uid" '
    .items[] | select(.spec.claimRef.namespace == $ns and .spec.claimRef.name == $claim and
      .spec.claimRef.uid == $uid) | .metadata.name
  ' "$pv_list")
  [ "${#matches[@]}" -eq 1 ] || die "$claim must resolve to exactly one PV by claim UID"
  pv_name="${matches[0]}"
  [ "$(jq -r '.spec.volumeName' "$pvc_file")" = "$pv_name" ] || die "$claim volumeName differs from UID-selected PV"
  kube patch pv "$pv_name" --type=merge -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}' >/dev/null
  readback="$(kube get pv "$pv_name" -o json)"
  jq -e --arg ns "$namespace" --arg claim "$claim" --arg uid "$pvc_uid" '
    .spec.persistentVolumeReclaimPolicy == "Retain" and
    .spec.claimRef.namespace == $ns and .spec.claimRef.name == $claim and .spec.claimRef.uid == $uid
  ' <<< "$readback" >/dev/null || die "$claim PV Retain/claim UID readback failed"
  jq -n --arg pvc "$claim" --arg pvc_uid "$pvc_uid" --arg pv "$pv_name" \
    '{pvc:$pvc,pvc_uid:$pvc_uid,pv:$pv,reclaim_policy:"Retain",claim_uid_matched:true}'
}

containerd_digest() {
  local image_ref="$1"
  sudo -n k3s ctr -n k8s.io images ls | awk -v ref="$image_ref" '$1 == ref {print $3}'
}

assert_containerd_image() {
  local image_ref="$1" label="$2" expected actual
  [[ "$image_ref" =~ @sha256:[0-9a-f]{64}$ ]] || die "$label image is not digest pinned"
  expected="${image_ref##*@}"
  actual="$(containerd_digest "$image_ref")"
  if [ "$(grep -c . <<< "$actual")" -ne 1 ] || [ "$actual" != "$expected" ]; then
    die "containerd exact $label image/digest readback failed"
  fi
}

apply_router_after_retain() {
  local tmp_dir="$1"
  apply_manifest_resource PersistentVolumeClaim router-ai-atius-data k8s/router-ai-atius/router.yaml
  kube -n "$namespace" annotate pvc/router-ai-atius-data \
    volume.kubernetes.io/selected-node=atius-srv-1 --overwrite >/dev/null
  kube -n "$namespace" wait --for=jsonpath='{.status.phase}'=Bound pvc/router-ai-atius-data --timeout=15m
  postgres_pv="$(patch_retain_by_claim_uid router-ai-atius-postgres-data "$tmp_dir")"
  router_pv="$(patch_retain_by_claim_uid router-ai-atius-data "$tmp_dir")"
  apply_manifest_resource Deployment router-ai-atius k8s/router-ai-atius/router.yaml
  apply_manifest_resource Service router-ai-atius k8s/router-ai-atius/router.yaml
  kube -n "$namespace" rollout status deployment/router-ai-atius --timeout=30m
}

write_apply_evidence() {
  local output="$evidence_dir/shadow-apply.json" tmp_file generated_at generated_epoch commit
  local restore_sha bootstrap_sha baseline_sha expected_catalog_sha router_image router_digest runtime_image_id cgroup
  local router_runtime_digest redis_image redis_digest redis_runtime_image_id redis_runtime_digest
  local postgres_image postgres_digest postgres_runtime_image_id postgres_runtime_digest
  local postgres_pv="$1" router_pv="$2" redis_service="$3" router_service="$4" apply_snapshot="$5"
  local runtime_snapshot_sha pre_publish_snapshot
  if [ -e "$output" ] || [ -L "$output" ]; then die 'shadow-apply.json already exists'; fi
  generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  generated_epoch="$(date +%s)"
  commit="$(git rev-parse HEAD)"
  restore_sha="$(sha256sum "$restore" | awk '{print $1}')"
  bootstrap_sha="$(sha256sum "$bootstrap" | awk '{print $1}')"
  baseline_sha="$(sha256sum "$baseline" | awk '{print $1}')"
  expected_catalog_sha="$(jq -r '.catalog_sha256' "$baseline")"
  router_image="$(jq -r '.image_ref' "$bootstrap")"
  router_digest="$(jq -r '.manifest_digest' "$bootstrap")"
  runtime_image_id="$(jq -r '.workloads.router.container.image_id' "$apply_snapshot")"
  router_runtime_digest="${runtime_image_id##*@}"
  redis_image="$(manifest_image router-ai-atius-redis k8s/router-ai-atius/redis.yaml)"
  redis_digest="${redis_image##*@}"
  redis_runtime_image_id="$(jq -r '.workloads.redis.container.image_id' "$apply_snapshot")"
  redis_runtime_digest="${redis_runtime_image_id##*@}"
  postgres_image="$(manifest_image router-ai-atius-postgres k8s/router-ai-atius/postgres.yaml)"
  postgres_digest="${postgres_image##*@}"
  postgres_runtime_image_id="$(jq -r '.workloads.postgres.container.image_id' "$apply_snapshot")"
  postgres_runtime_digest="${postgres_runtime_image_id##*@}"
  cgroup="$(cpu_max_value)"
  runtime_snapshot_sha="$(sha256sum "$apply_snapshot" | awk '{print $1}')"
  # shellcheck disable=SC2031
  pre_publish_snapshot="$tmp/runtime-snapshot-pre-publish.json"
  tmp_file="$(mktemp "$evidence_dir/.shadow-apply.XXXXXX")"
  chmod 600 "$tmp_file"
  jq -n --arg generated_at "$generated_at" --argjson generated_at_epoch "$generated_epoch" \
    --arg cluster_uid "$cluster_uid" --arg commit "$commit" --arg cgroup "$cgroup" \
    --arg restore_sha256 "$restore_sha" --arg bootstrap_sha256 "$bootstrap_sha" \
    --arg catalog_baseline_sha256 "$baseline_sha" --arg expected_catalog_sha256 "$expected_catalog_sha" \
    --arg manifest_sha256 "$current_manifest_hash" --arg image_ref "$router_image" \
    --arg image_digest "$router_digest" --arg runtime_image_id "$runtime_image_id" --arg router_runtime_digest "$router_runtime_digest" \
    --arg redis_image_ref "$redis_image" --arg redis_image_digest "$redis_digest" \
    --arg redis_runtime_image_id "$redis_runtime_image_id" --arg redis_runtime_digest "$redis_runtime_digest" \
    --arg postgres_image_ref "$postgres_image" --arg postgres_image_digest "$postgres_digest" \
    --arg postgres_runtime_image_id "$postgres_runtime_image_id" --arg postgres_runtime_digest "$postgres_runtime_digest" \
    --argjson pvs "[$postgres_pv,$router_pv]" \
    --arg runtime_snapshot_sha256 "$runtime_snapshot_sha" \
    --argjson runtime_snapshot "$(cat "$apply_snapshot")" \
    --arg redis_cluster_ip "$(jq -r '.spec.clusterIP' "$redis_service")" \
    --arg router_cluster_ip "$(jq -r '.spec.clusterIP' "$router_service")" \
    'def legacy_workload($key; $app):
      $runtime_snapshot.workloads[$key] as $item |
      {app:$app,
       controller:{kind:$item.controller.kind,name:$item.controller.name,uid:$item.controller.uid},
       pod_owner:{name:$item.pod_owner.name,uid:$item.pod_owner.uid},
       pod:{name:$item.pod.name,uid:$item.pod.uid,ip:$item.pod.ip},
       container:{name:$item.container.name,image_ref:$item.container.image_ref,image_id:$item.container.image_id,
         resources:$item.container.resources}};
    {schema_version:1,status:"go",generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,
      cluster_uid:$cluster_uid,commit:$commit,cpu_max:$cgroup,
      inputs:{restore_sha256:$restore_sha256,bootstrap_sha256:$bootstrap_sha256,manifest_sha256:$manifest_sha256,
        catalog_baseline_sha256:$catalog_baseline_sha256},
      catalog:{expected_sha256:$expected_catalog_sha256},
      image:{reference:$image_ref,digest:$image_digest,runtime_digest:$router_runtime_digest,runtime_image_id:$runtime_image_id,exact:true},
      images:{
        router:{reference:$image_ref,digest:$image_digest,runtime_digest:$router_runtime_digest,runtime_image_id:$runtime_image_id,exact:true},
        redis:{reference:$redis_image_ref,digest:$redis_image_digest,runtime_digest:$redis_runtime_digest,runtime_image_id:$redis_runtime_image_id,exact:true},
        postgres:{reference:$postgres_image_ref,digest:$postgres_image_digest,runtime_digest:$postgres_runtime_digest,runtime_image_id:$postgres_runtime_image_id,exact:true}},
      workloads:{router:legacy_workload("router";"router-ai-atius"),redis:legacy_workload("redis";"router-ai-atius-redis"),
        postgres:legacy_workload("postgres";"router-ai-atius-postgres")},
      placement:{node:"atius-srv-1",postgres_ready:true,redis_ready_before_router:true,router_ready:true,cpu_per_pod:"500m"},
      pvs:$pvs,services:{redis:{type:"ClusterIP",cluster_ip:$redis_cluster_ip,endpoints_ready:true},router:{type:"ClusterIP",cluster_ip:$router_cluster_ip,endpoints_ready:true}},
      runtime_snapshot:{sha256:$runtime_snapshot_sha256,pre_publish_sha256:$runtime_snapshot_sha256,exact:true,map:$runtime_snapshot},
      storage:$runtime_snapshot.storage,
      mutations:{apache:false,podman:false}}' > "$tmp_file"
  # shellcheck disable=SC2031
  capture_apply_runtime_snapshot "$pre_publish_snapshot" pre-publish "$tmp"
  if ! runtime_snapshots_equal "$apply_snapshot" "$pre_publish_snapshot"; then
    rm -f "$tmp_file"
    die 'runtime changed before shadow apply evidence publication; GO not published'
  fi
  mv "$tmp_file" "$output"
}

self_test() {
  local test_dir now cluster restore_file state_file workload service slices sha order expected_order
  local stable_snapshot recreated_pv_snapshot rollout_snapshot
  test_dir="$(mktemp -d)"
  trap 'rm -rf "$test_dir"' RETURN
  now="$(date +%s)"; cluster="cluster-test"
  restore_file="$test_dir/restore.json"; state_file="$test_dir/state.json"
  jq -n --arg cluster "$cluster" --argjson now "$now" '{status:"go",restore_passed:true,cluster_uid:$cluster,generated_at_epoch:$now,target:{node:"atius-srv-1",database:"DBRouterAiAtius",clean_before_restore:true,server_version_num:"170010"},pvs:[{reclaim_policy:"Retain",claim_uid_matched:true}],runtime_stage:{redis_applied:false,router_applied:false}}' > "$restore_file"
  sha="$(sha256sum "$restore_file" | awk '{print $1}')"
  jq -n --arg cluster "$cluster" --arg path "$restore_file" --arg sha "$sha" '{schema_version:1,target:"router-ai-atius/DBRouterAiAtius@atius-srv-1",status:"go",cluster_uid:$cluster,evidence_path:$path,evidence_sha256:$sha}' > "$state_file"
  chmod 600 "$restore_file" "$state_file"
  validate_restore_gate "$restore_file" "$state_file" "$cluster" "$now"
  jq '.restore_passed=false' "$restore_file" > "$test_dir/bad-restore.json"
  if (validate_restore_gate "$test_dir/bad-restore.json" "$state_file" "$cluster" "$now") 2>/dev/null; then die 'invalid restore GO was accepted'; fi
  printf '\n' >> "$restore_file"
  if (validate_restore_gate "$restore_file" "$state_file" "$cluster" "$now") 2>/dev/null; then die 'tampered restore checksum was accepted'; fi
  quota_ok '80000 100000'
  if (quota_ok '80001 100000') 2>/dev/null; then die 'quota above 800m was accepted'; fi
  [[ "$(manifest_image router-ai-atius k8s/router-ai-atius/router.yaml)" =~ @sha256:[0-9a-f]{64}$ ]] ||
    die 'router workload image extraction failed'
  [[ "$(manifest_image router-ai-atius-redis k8s/router-ai-atius/redis.yaml)" =~ ^docker.io/library/redis@sha256:[0-9a-f]{64}$ ]] ||
    die 'Redis workload image is not immutable'
  workload="$test_dir/workload.json"
  jq -n '{items:[{metadata:{name:"router-abc",uid:"pod-uid",labels:{"app.kubernetes.io/name":"router-ai-atius"}},spec:{nodeName:"atius-srv-1",containers:[{image:"example@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",resources:{requests:{cpu:"500m"},limits:{cpu:"500m"}}}]},status:{phase:"Running",podIP:"10.42.0.9",containerStatuses:[{ready:true,imageID:"example@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}}]}' > "$workload"
  validate_workload_json "$workload" router-ai-atius 'example@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
  jq '.items[0].spec.nodeName="atius-srv-2"' "$workload" > "$test_dir/bad-workload.json"
  if (validate_workload_json "$test_dir/bad-workload.json" router-ai-atius 'example@sha256:abc') 2>/dev/null; then die 'wrong placement was accepted'; fi
  service="$test_dir/service.json"
  jq -n '{metadata:{name:"router-ai-atius"},spec:{type:"ClusterIP",selector:{"app.kubernetes.io/name":"router-ai-atius"},clusterIP:"10.43.0.10",clusterIPs:["10.43.0.10"],ports:[{port:3000}]}}' > "$service"
  validate_service_json "$service" router-ai-atius 3000 router-ai-atius
  jq '.spec.ports[0].nodePort=32000' "$service" > "$test_dir/bad-service.json"
  if (validate_service_json "$test_dir/bad-service.json" router-ai-atius 3000) 2>/dev/null; then die 'NodePort was accepted'; fi
  slices="$test_dir/slices.json"
  jq -n '{items:[{metadata:{labels:{"kubernetes.io/service-name":"router-ai-atius"}},endpoints:[{addresses:["10.42.0.9"],conditions:{ready:true},nodeName:"atius-srv-1",targetRef:{kind:"Pod",namespace:"router-ai-atius",name:"router-abc",uid:"pod-uid"}}]}]}' > "$slices"
  validate_endpointslice_json "$slices" router-ai-atius router-abc pod-uid 10.42.0.9
  jq '.items[0].endpoints[0].conditions.ready=false' "$slices" > "$test_dir/bad-slices.json"
  if (validate_endpointslice_json "$test_dir/bad-slices.json" router-ai-atius router-abc pod-uid 10.42.0.9) 2>/dev/null; then die 'unready EndpointSlice was accepted'; fi
  jq '.items[0].endpoints[0].targetRef.uid="other-pod"' "$slices" > "$test_dir/wrong-pod-slices.json"
  if (validate_endpointslice_json "$test_dir/wrong-pod-slices.json" router-ai-atius router-abc pod-uid 10.42.0.9) 2>/dev/null; then die 'wrong pod EndpointSlice was accepted'; fi
  order="$test_dir/router-order.log"
  (
    kube() { printf 'kube %s\n' "$*" >> "$order"; }
    apply_manifest_resource() { printf 'apply %s/%s\n' "$1" "$2" >> "$order"; }
    patch_retain_by_claim_uid() { printf 'retain %s\n' "$1" >> "$order"; printf '{"pvc":"%s"}\n' "$1"; }
    apply_router_after_retain "$test_dir"
  )
  expected_order="$(printf '%s\n' \
    'apply PersistentVolumeClaim/router-ai-atius-data' \
    "kube -n $namespace annotate pvc/router-ai-atius-data volume.kubernetes.io/selected-node=atius-srv-1 --overwrite" \
    "kube -n $namespace wait --for=jsonpath={.status.phase}=Bound pvc/router-ai-atius-data --timeout=15m" \
    'retain router-ai-atius-postgres-data' \
    'retain router-ai-atius-data' \
    'apply Deployment/router-ai-atius' \
    'apply Service/router-ai-atius' \
    "kube -n $namespace rollout status deployment/router-ai-atius --timeout=30m")"
  [ "$(cat "$order")" = "$expected_order" ] || die 'router Deployment was not ordered after PV Retain readback'
  stable_snapshot="$test_dir/runtime-stable.json"
  jq -S -c -n '{schema_version:1,
    workloads:{router:{controller:{uid:"deployment-uid",resource_version:"101",generation:1},
      pod_owner:{uid:"rs-uid",resource_version:"102"},pod:{uid:"pod-uid",resource_version:"103"}}},
    storage:{postgres:{pvc:{name:"router-ai-atius-postgres-data",uid:"pvc-uid",resource_version:"201",volume_name:"pv-name"},
      pv:{name:"pv-name",uid:"pv-uid",resource_version:"202",claim_ref:{uid:"pvc-uid",name:"router-ai-atius-postgres-data",namespace:"router-ai-atius"},reclaim_policy:"Retain"}}}}' \
    > "$stable_snapshot"
  runtime_snapshots_equal "$stable_snapshot" "$stable_snapshot" || die 'equal canonical snapshots were rejected'
  recreated_pv_snapshot="$test_dir/runtime-recreated-pv.json"
  jq -S -c '.storage.postgres.pv.uid="replacement-pv-uid" | .storage.postgres.pv.resource_version="203"' \
    "$stable_snapshot" > "$recreated_pv_snapshot"
  if runtime_snapshots_equal "$stable_snapshot" "$recreated_pv_snapshot"; then
    die 'recreated PV between apply snapshots was accepted'
  fi
  rollout_snapshot="$test_dir/runtime-rollout.json"
  jq -S -c '.workloads.router.controller.generation=2 | .workloads.router.controller.resource_version="104" |
    .workloads.router.pod_owner.uid="replacement-rs-uid" | .workloads.router.pod.uid="replacement-pod-uid"' \
    "$stable_snapshot" > "$rollout_snapshot"
  if runtime_snapshots_equal "$stable_snapshot" "$rollout_snapshot"; then
    die 'rollout between apply snapshots was accepted'
  fi
  echo 'apply self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --stage) stage="${2:?}"; shift ;;
    --cleanup-evidence) cleanup="${2:?}"; shift ;;
    --bootstrap-evidence) bootstrap="${2:?}"; shift ;;
    --restore-evidence) restore="${2:?}"; shift ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --self-test)
      [ "${RUN_K3S_ROUTER_APPLY:-0}" != 1 ] || die 'unsafe test environment'
      self_test
      exit 0
      ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

scripts/k3s-router-validate-manifests.sh
if [ "$mode" != live ]; then
  echo 'shadow apply dry-run: PASS; use --stage postgres or runtime for live work'
  exit 0
fi

[ "${RUN_K3S_ROUTER_APPLY:-0}" = 1 ] || die '--live requires RUN_K3S_ROUTER_APPLY=1'
[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[ "${PHASE29_APPLY_CONFIRM:-}" = APPLY_CLUSTERIP_SHADOW_ONLY ] || die 'missing exact apply confirmation'
quota_ok "$(cpu_max_value)"

if [ -z "$stage" ] && [ -n "$restore" ]; then stage=runtime; fi
[ "$stage" = postgres ] || [ "$stage" = runtime ] || die '--stage must be postgres or runtime'

if [ "$stage" = runtime ]; then
  require_evidence_directory
  [ -n "$cleanup" ] || cleanup="$evidence_dir/cleanup.json"
  [ -n "$bootstrap" ] || bootstrap="$evidence_dir/bootstrap.json"
  [ -n "$restore" ] || restore="$evidence_dir/restore.json"
  baseline="$evidence_dir/catalog-baseline.json"
fi
if [ -z "$cleanup" ] || [ -z "$bootstrap" ]; then die 'cleanup and bootstrap evidence are required'; fi
require_regular_evidence "$cleanup" cleanup
require_regular_evidence "$bootstrap" bootstrap

PHASE29_LIVE=1 PHASE29_REQUIRE_STABLE_SECONDS=300 scripts/k3s-router-preflight.sh --live \
  --require-cleanup-evidence "$cleanup" --require-bootstrap-evidence "$bootstrap"
PHASE29_LIVE=1 scripts/k3s-router-validate-manifests.sh --server

if [ "$stage" = postgres ]; then
  kube apply -f k8s/router-ai-atius/namespace.yaml >/dev/null
  kube apply -f k8s/router-ai-atius/configmap.yaml >/dev/null
  kube apply -f k8s/router-ai-atius/postgres.yaml >/dev/null
  kube -n "$namespace" wait --for=jsonpath='{.status.phase}'=Bound pvc/router-ai-atius-postgres-data --timeout=15m
  kube -n "$namespace" rollout status statefulset/router-ai-atius-postgres --timeout=20m
  tmp="$(mktemp -d /dev/shm/phase29-postgres-apply.XXXXXX)"
  trap 'rm -rf "$tmp"' EXIT
  postgres_image="$(manifest_image router-ai-atius-postgres k8s/router-ai-atius/postgres.yaml)"
  workload_snapshot router-ai-atius-postgres "$postgres_image" "$tmp/postgres-pods.json"
  service_snapshot router-ai-atius-postgres 5432 "$tmp/postgres-service.json" "$tmp/postgres-slices.json" "$tmp/postgres-pods.json" router-ai-atius-postgres
  echo 'shadow postgres stage: PASS'
  exit 0
fi

cluster_uid="$(kube get namespace kube-system -o jsonpath='{.metadata.uid}')"
now="$(date +%s)"
current_manifest_hash="$(manifest_hash)"
[ "$(realpath -e "$restore")" = "$evidence_dir/restore.json" ] ||
  die 'runtime restore evidence must be the canonical restore.json in the evidence directory'
validate_restore_gate "$restore" "$restore_state" "$cluster_uid" "$now"
validate_bootstrap_gate "$bootstrap" "$cluster_uid" "$current_manifest_hash" "$now"
validate_catalog_baseline "$baseline" "$now"

router_image="$(manifest_image router-ai-atius k8s/router-ai-atius/router.yaml)"
redis_image="$(manifest_image router-ai-atius-redis k8s/router-ai-atius/redis.yaml)"
postgres_image="$(manifest_image router-ai-atius-postgres k8s/router-ai-atius/postgres.yaml)"
[ "$router_image" = "$(jq -r '.image_ref' "$bootstrap")" ] || die 'router manifest image differs from bootstrap evidence'
expected_runtime_digest="$(jq -r '.podman_digest' "$bootstrap")"
assert_containerd_image "$router_image" router
existing_router_deployments="$(kube -n "$namespace" get deployments -o json | jq '[.items[] | select(.metadata.name == "router-ai-atius")] | length')"
existing_router_pods="$(kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius -o json | jq '.items | length')"
if [ "$existing_router_deployments" -ne 0 ] || [ "$existing_router_pods" -ne 0 ]; then
  die 'router runtime already exists before the Redis-first stage'
fi

tmp="$(mktemp -d /dev/shm/phase29-runtime-apply.XXXXXX)"
trap 'rm -rf "$tmp"' EXIT
kube apply -f k8s/router-ai-atius/redis.yaml >/dev/null
kube -n "$namespace" rollout status deployment/router-ai-atius-redis --timeout=15m
workload_snapshot router-ai-atius-redis "$redis_image" "$tmp/redis-pods.json"
assert_containerd_image "$redis_image" redis
service_snapshot router-ai-atius-redis 6379 "$tmp/redis-service.json" "$tmp/redis-slices.json" "$tmp/redis-pods.json" router-ai-atius-redis

apply_router_after_retain "$tmp"

workload_snapshot router-ai-atius-postgres "$postgres_image" "$tmp/postgres-pods.json"
workload_snapshot router-ai-atius-redis "$redis_image" "$tmp/redis-pods.json"
workload_snapshot router-ai-atius "$router_image" "$tmp/router-pods.json" "$expected_runtime_digest"
assert_containerd_image "$postgres_image" postgres
runtime_image_id="$(jq -r '.items[0].status.containerStatuses[0].imageID' "$tmp/router-pods.json")"
case "$runtime_image_id" in
  *@sha256:*) [ "${runtime_image_id##*@}" = "$expected_runtime_digest" ] || die 'router runtime imageID reports a different content digest' ;;
  *) die 'router runtime imageID is not digest-addressed' ;;
esac
service_snapshot router-ai-atius-redis 6379 "$tmp/redis-service.json" "$tmp/redis-slices.json" "$tmp/redis-pods.json" router-ai-atius-redis
service_snapshot router-ai-atius 3000 "$tmp/router-service.json" "$tmp/router-slices.json" "$tmp/router-pods.json" router-ai-atius
workload_identity router-ai-atius-postgres statefulset router-ai-atius-postgres "$postgres_image" "$tmp/postgres-pods.json" "$tmp/postgres-identity.json"
workload_identity router-ai-atius-redis deployment router-ai-atius-redis "$redis_image" "$tmp/redis-pods.json" "$tmp/redis-identity.json"
workload_identity router-ai-atius deployment router-ai-atius "$router_image" "$tmp/router-pods.json" "$tmp/router-identity.json"
quota_ok "$(cpu_max_value)"

apply_runtime_snapshot="$tmp/runtime-snapshot-apply.json"
capture_apply_runtime_snapshot "$apply_runtime_snapshot" apply "$tmp"
validate_apply_runtime_snapshot "$apply_runtime_snapshot" "$postgres_pv" "$router_pv" \
  "$router_image" "$redis_image" "$postgres_image" "$expected_runtime_digest"
write_apply_evidence "$postgres_pv" "$router_pv" "$tmp/redis-service.json" "$tmp/router-service.json" "$apply_runtime_snapshot"
echo 'shadow runtime stage: PASS'
