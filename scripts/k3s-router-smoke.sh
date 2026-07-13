#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

namespace=router-ai-atius
strict=false
capture_baseline=false
evidence_dir=""
evidence_root="${PHASE29_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase29}"
baseline_url=https://router.atius.com.br
restore_state="$HOME/.local/state/router-ai-atius/phase29/restore-target-state.json"

die() {
  echo "shadow smoke failed: $*" >&2
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
  if [ "$period" -le 0 ] || [ $((quota * 10)) -gt $((period * 8)) ]; then
    die "cpu.max exceeds 800m: $value"
  fi
}

require_regular_json() {
  local file="$1" label="$2"
  if [ ! -f "$file" ] || [ -L "$file" ]; then
    die "$label must be a regular non-symlink file"
  fi
  jq -e . "$file" >/dev/null || die "$label is not valid JSON"
}

require_evidence_directory() {
  [ -n "$evidence_dir" ] || die '--evidence-dir is required'
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

write_catalog_baseline() {
  local body="$1" output="$2" captured_at captured_epoch tmp_file
  if [ -e "$output" ] || [ -L "$output" ]; then die 'catalog-baseline.json already exists'; fi
  captured_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  captured_epoch="$(date +%s)"
  tmp_file="$(mktemp "$evidence_dir/.catalog-baseline.XXXXXX")"
  chmod 600 "$tmp_file"
  python3 - "$body" "$tmp_file" "$captured_at" "$captured_epoch" <<'PY'
import hashlib
import json
import pathlib
import re
import sys

body_path, output_path = map(pathlib.Path, sys.argv[1:3])
captured_at, captured_epoch = sys.argv[3], int(sys.argv[4])
payload = json.loads(body_path.read_text(encoding="utf-8"))
if not isinstance(payload, dict) or set(payload) != {"data"} or not isinstance(payload["data"], list):
    raise SystemExit("pre-shadow /v1/models must contain only a data array")
models = []
for row in payload["data"]:
    if not isinstance(row, dict) or not isinstance(row.get("id"), str) or not row["id"]:
        raise SystemExit("pre-shadow catalog contains an invalid model id")
    models.append(row["id"])
if not models or len(models) != len(set(models)) or "embedding-gte-v1" not in models:
    raise SystemExit("pre-shadow catalog must be unique and include embedding-gte-v1")
catalog_sha = hashlib.sha256(("\n".join(models) + "\n").encode()).hexdigest()
output_path.write_text(json.dumps({
    "schema_version": 1,
    "kind": "authenticated-pre-shadow-model-catalog",
    "captured_at": captured_at,
    "captured_at_epoch": captured_epoch,
    "models": models,
    "catalog_sha256": catalog_sha,
}, separators=(",", ":")), encoding="utf-8")
PY
  mv "$tmp_file" "$output"
}

validate_catalog_baseline() {
  local file="$1"
  require_regular_json "$file" catalog-baseline.json
  [ "$(stat -c %U:%a "$file")" = "$(id -un):600" ] ||
    die 'catalog-baseline.json owner/mode must be caller:600'
  python3 - "$file" <<'PY'
import hashlib
import json
import pathlib
import sys

doc = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
models = doc.get("models")
if set(doc) != {"schema_version", "kind", "captured_at", "captured_at_epoch", "models", "catalog_sha256"}:
    raise SystemExit("catalog baseline contains unexpected fields")
if doc["schema_version"] != 1 or doc["kind"] != "authenticated-pre-shadow-model-catalog":
    raise SystemExit("catalog baseline has the wrong schema/kind")
if not isinstance(doc["captured_at_epoch"], int):
    raise SystemExit("catalog baseline capture timestamp is invalid")
if not isinstance(models, list) or not models or len(models) != len(set(models)):
    raise SystemExit("catalog baseline models are absent or duplicated")
if not all(isinstance(item, str) and item and item.strip() == item for item in models):
    raise SystemExit("catalog baseline contains an invalid model id")
if "embedding-gte-v1" not in models:
    raise SystemExit("catalog baseline must include embedding-gte-v1")
actual = hashlib.sha256(("\n".join(models) + "\n").encode()).hexdigest()
if doc["catalog_sha256"] != actual:
    raise SystemExit("catalog baseline model checksum mismatch")
PY
}

validate_restore_chain() {
  local restore_file="$1" state_file="$2" apply_file="$3" expected_cluster="$4" now="$5"
  local state_path state_sha restore_sha apply_restore_sha generated
  require_regular_json "$state_file" 'canonical restore state'
  [ "$(stat -c %U:%a "$state_file")" = "$(id -un):600" ] ||
    die 'canonical restore state owner/mode must be caller:600'
  jq -e --arg cluster "$expected_cluster" '
    .schema_version == 1 and .target == "router-ai-atius/DBRouterAiAtius@atius-srv-1" and
    .status == "go" and .cluster_uid == $cluster and
    (.evidence_sha256 | test("^[0-9a-f]{64}$"))
  ' "$state_file" >/dev/null || die 'canonical restore state is not GO for this cluster and target'
  jq -e --arg cluster "$expected_cluster" '
    .status == "go" and .restore_passed == true and .cluster_uid == $cluster and
    .target.node == "atius-srv-1" and .target.database == "DBRouterAiAtius" and
    (.generated_at_epoch | type == "number")
  ' "$restore_file" >/dev/null || die 'restore evidence is not a complete cluster-bound GO'
  generated="$(jq -r '.generated_at_epoch' "$restore_file")"
  if [ "$generated" -gt "$now" ] || [ $((now - generated)) -gt 3600 ]; then
    die 'restore evidence is stale or future-dated'
  fi
  state_path="$(jq -r '.evidence_path' "$state_file")"
  if [ ! -f "$state_path" ] || [ -L "$state_path" ]; then
    die 'canonical restore state points to missing evidence'
  fi
  [ "$(realpath -e "$state_path")" = "$(realpath -e "$restore_file")" ] ||
    die 'canonical restore state points to different evidence'
  restore_sha="$(sha256sum "$restore_file" | awk '{print $1}')"
  state_sha="$(jq -r '.evidence_sha256' "$state_file")"
  [ "$restore_sha" = "$state_sha" ] || die 'restore evidence checksum differs from canonical state'
  apply_restore_sha="$(jq -r '.inputs.restore_sha256 // ""' "$apply_file")"
  [ "$restore_sha" = "$apply_restore_sha" ] || die 'shadow apply is bound to a different restore checksum'
}

validate_workload_json() {
  local file="$1" app="$2" expected_image="$3" expected_digest expected_runtime_digest
  [[ "$expected_image" =~ @sha256:[0-9a-f]{64}$ ]] || die "$app apply image is not digest pinned"
  expected_digest="${expected_image##*@}"
  expected_runtime_digest="${4:-$expected_digest}"
  jq -e --arg app "$app" --arg image "$expected_image" --arg digest "$expected_runtime_digest" '
    .items | length == 1 and
    .[0].metadata.labels["app.kubernetes.io/name"] == $app and
    (.[0].metadata.name | type == "string" and length > 0) and
    (.[0].metadata.uid | type == "string" and length > 0) and
    .[0].spec.nodeName == "atius-srv-1" and .[0].status.phase == "Running" and
    (.[0].status.podIP | type == "string" and length > 0) and
    ([.[0].status.conditions[]? | select(.type == "Ready" and .status == "True")] | length == 1) and
    ((.[0].spec.initContainers // []) | length == 0) and
    (.[0].spec.containers | length == 1) and
    .[0].spec.containers[0].image == $image and
    .[0].spec.containers[0].resources.requests.cpu == "500m" and
    .[0].spec.containers[0].resources.limits.cpu == "500m" and
    (.[0].status.containerStatuses | length == 1) and
    .[0].status.containerStatuses[0].ready == true and
    (.[0].status.containerStatuses[0].imageID | type == "string" and endswith($digest))
  ' "$file" >/dev/null || die "$app pod is not unique, Ready, immutable, and 500m on srv1"
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
  ' "$controller_file" >/dev/null || die "$app controller identity/resources differ from shadow apply"
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

validate_workload_chain() {
  local key="$1" identity_file="$2"
  jq -e --arg key "$key" --argjson live "$(cat "$identity_file")" '
    .images[$key].digest as $digest |
    (.images[$key].runtime_digest // $digest) as $runtime_digest |
    .workloads[$key] == $live and
    .images[$key].exact == true and
    (.images[$key].reference | test("@sha256:[0-9a-f]{64}$")) and
    .images[$key].digest == (.images[$key].reference | split("@")[-1]) and
    .images[$key].runtime_image_id == $live.container.image_id and
    .images[$key].reference == $live.container.image_ref and
    ($live.container.image_id | endswith($runtime_digest))
  ' "$apply_evidence" >/dev/null || die "$key live/apply workload or immutable image identity mismatch"
}

validate_service_json() {
  local file="$1"
  jq -e '
    .metadata.name == "router-ai-atius" and .spec.type == "ClusterIP" and
    .spec.selector == {"app.kubernetes.io/name":"router-ai-atius"} and
    (.spec.clusterIP | type == "string" and length > 0 and . != "None") and
    (.spec.clusterIPs | type == "array" and length == 1) and
    .spec.clusterIPs[0] == .spec.clusterIP and
    ((.spec.externalIPs // []) | length == 0) and
    (.spec.externalName // "" | length == 0) and
    (.spec.ports | length == 1) and .spec.ports[0].port == 3000 and
    (.spec.ports[0] | has("nodePort") | not)
  ' "$file" >/dev/null || die 'router Service is not strict ClusterIP on port 3000'
}

validate_endpointslice_json() {
  local file="$1" pod_file="$2" pod_name pod_uid pod_ip
  pod_name="$(jq -r '.items[0].metadata.name' "$pod_file")"
  pod_uid="$(jq -r '.items[0].metadata.uid' "$pod_file")"
  pod_ip="$(jq -r '.items[0].status.podIP' "$pod_file")"
  jq -e --arg pod_name "$pod_name" --arg pod_uid "$pod_uid" --arg pod_ip "$pod_ip" '
    .items | length >= 1 and
    all(.metadata.labels["kubernetes.io/service-name"] == "router-ai-atius") and
    ([.[].endpoints[]?] | length == 1) and
    all(.[].endpoints[]?;
      .conditions.ready == true and .nodeName == "atius-srv-1" and
      .addresses == [$pod_ip] and
      (.targetRef.apiVersion // "v1") == "v1" and .targetRef.kind == "Pod" and
      .targetRef.namespace == "router-ai-atius" and
      .targetRef.name == $pod_name and .targetRef.uid == $pod_uid)
  ' "$file" >/dev/null || die 'router EndpointSlice is not bound to the validated pod UID/IP'
}

capture_runtime_snapshot() {
  local output="$1" label="$2" snapshot_dir owner_name pv_name key service claim
  snapshot_dir="$tmp/runtime-$label"
  mkdir -m 700 "$snapshot_dir"

  kube -n "$namespace" get deployment router-ai-atius -o json > "$snapshot_dir/controller-router.json"
  kube -n "$namespace" get deployment router-ai-atius-redis -o json > "$snapshot_dir/controller-redis.json"
  kube -n "$namespace" get statefulset router-ai-atius-postgres -o json > "$snapshot_dir/controller-postgres.json"
  kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius -o json > "$snapshot_dir/pods-router.json"
  kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius-redis -o json > "$snapshot_dir/pods-redis.json"
  kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius-postgres -o json > "$snapshot_dir/pods-postgres.json"

  for key in router redis; do
    owner_name="$(jq -r '[.items[0].metadata.ownerReferences[]? | select(.controller == true)][0].name // ""' "$snapshot_dir/pods-$key.json")"
    [ -n "$owner_name" ] || die "$key snapshot pod has no controller owner"
    kube -n "$namespace" get replicaset "$owner_name" -o json > "$snapshot_dir/owner-$key.json"
  done
  cp "$snapshot_dir/controller-postgres.json" "$snapshot_dir/owner-postgres.json"

  for service in router-ai-atius router-ai-atius-redis router-ai-atius-postgres; do
    kube -n "$namespace" get service "$service" -o json > "$snapshot_dir/service-$service.json"
    kube -n "$namespace" get endpointslices.discovery.k8s.io \
      -l "kubernetes.io/service-name=$service" -o json > "$snapshot_dir/slices-$service.json"
  done

  for key in router postgres; do
    if [ "$key" = router ]; then
      claim=router-ai-atius-data
    else
      claim=router-ai-atius-postgres-data
    fi
    kube -n "$namespace" get pvc "$claim" -o json > "$snapshot_dir/pvc-$key.json"
    pv_name="$(jq -r '.spec.volumeName // ""' "$snapshot_dir/pvc-$key.json")"
    [ -n "$pv_name" ] || die "$claim snapshot has no bound PV name"
    kube get pv "$pv_name" -o json > "$snapshot_dir/pv-$key.json"
  done

  python3 - "$snapshot_dir" "$output" <<'PY'
import json
import pathlib
import sys

source, output = map(pathlib.Path, sys.argv[1:])
namespace = "router-ai-atius"


def load(name):
    return json.loads((source / name).read_text(encoding="utf-8"))


def required_text(value, label):
    if not isinstance(value, str) or not value:
        raise SystemExit(f"runtime snapshot missing {label}")
    return value


def metadata(obj, label):
    meta = obj.get("metadata", {})
    if meta.get("deletionTimestamp"):
        raise SystemExit(f"{label} is being deleted")
    return {
        "name": required_text(meta.get("name"), f"{label} name"),
        "uid": required_text(meta.get("uid"), f"{label} UID"),
        "resource_version": required_text(meta.get("resourceVersion"), f"{label} resourceVersion"),
    }


apps = {
    "router": ("Deployment", "router-ai-atius", "router-ai-atius"),
    "redis": ("Deployment", "router-ai-atius-redis", "redis"),
    "postgres": ("StatefulSet", "router-ai-atius-postgres", "postgres"),
}
workloads = {}
pod_identity = {}
for key, (kind, app, container_name) in apps.items():
    controller_obj = load(f"controller-{key}.json")
    pods = load(f"pods-{key}.json").get("items", [])
    if len(pods) != 1:
        raise SystemExit(f"{app} snapshot requires exactly one pod")
    pod = pods[0]
    controller = metadata(controller_obj, f"{app} controller")
    pod_meta = metadata(pod, f"{app} pod")
    controller["kind"] = kind
    generation = controller_obj.get("metadata", {}).get("generation")
    observed = controller_obj.get("status", {}).get("observedGeneration")
    if not isinstance(generation, int) or generation != observed:
        raise SystemExit(f"{app} controller generation is not fully observed")
    controller["generation"] = generation
    controller["observed_generation"] = observed
    status = controller_obj.get("status", {})
    if kind == "Deployment":
        if any(status.get(field, 0) != 1 for field in ("replicas", "updatedReplicas", "readyReplicas", "availableReplicas")):
            raise SystemExit(f"{app} Deployment is rolling out")
        if status.get("unavailableReplicas", 0) != 0:
            raise SystemExit(f"{app} Deployment has unavailable replicas")
        controller["revision"] = None
    else:
        if any(status.get(field, 0) != 1 for field in ("replicas", "currentReplicas", "updatedReplicas", "readyReplicas")):
            raise SystemExit(f"{app} StatefulSet is rolling out")
        current_revision = required_text(status.get("currentRevision"), f"{app} currentRevision")
        if current_revision != status.get("updateRevision"):
            raise SystemExit(f"{app} StatefulSet revisions differ")
        controller["revision"] = current_revision

    owners = [owner for owner in pod.get("metadata", {}).get("ownerReferences", []) if owner.get("controller") is True]
    if len(owners) != 1:
        raise SystemExit(f"{app} pod requires one controller owner")
    owner_ref = owners[0]
    owner_obj = load(f"owner-{key}.json")
    owner = metadata(owner_obj, f"{app} pod owner")
    owner["kind"] = required_text(owner_ref.get("kind"), f"{app} owner kind")
    if owner["name"] != owner_ref.get("name") or owner["uid"] != owner_ref.get("uid"):
        raise SystemExit(f"{app} pod owner UID/name changed during capture")
    if kind == "Deployment":
        root_owners = [item for item in owner_obj.get("metadata", {}).get("ownerReferences", []) if item.get("controller") is True]
        if len(root_owners) != 1 or root_owners[0].get("kind") != kind or root_owners[0].get("uid") != controller["uid"]:
            raise SystemExit(f"{app} ReplicaSet is not owned by the captured Deployment")
    elif owner["uid"] != controller["uid"]:
        raise SystemExit(f"{app} pod is not owned by the captured StatefulSet")

    if pod.get("status", {}).get("phase") != "Running" or pod.get("spec", {}).get("nodeName") != "atius-srv-1":
        raise SystemExit(f"{app} pod is not Running on atius-srv-1")
    conditions = pod.get("status", {}).get("conditions", [])
    if len([item for item in conditions if item.get("type") == "Ready" and item.get("status") == "True"]) != 1:
        raise SystemExit(f"{app} pod is not Ready")
    containers = pod.get("spec", {}).get("containers", [])
    statuses = pod.get("status", {}).get("containerStatuses", [])
    if (pod.get("spec", {}).get("initContainers") or len(containers) != 1 or len(statuses) != 1 or
            containers[0].get("name") != container_name):
        raise SystemExit(f"{app} snapshot container shape changed")
    container_status = statuses[0]
    restart_count = container_status.get("restartCount")
    if container_status.get("ready") is not True or not isinstance(restart_count, int):
        raise SystemExit(f"{app} container is not Ready or lacks restartCount")
    resources = containers[0].get("resources", {})
    if (resources.get("requests", {}).get("cpu"), resources.get("limits", {}).get("cpu")) != ("500m", "500m"):
        raise SystemExit(f"{app} container CPU contract differs")
    image_ref = required_text(containers[0].get("image"), f"{app} image ref")
    image_id = required_text(container_status.get("imageID"), f"{app} imageID")
    digest_match = re.search(r"@(sha256:[0-9a-f]{64})$", image_ref)
    runtime_match = re.search(r"(sha256:[0-9a-f]{64})$", image_id)
    if not digest_match or not runtime_match:
        raise SystemExit(f"{app} image reference or runtime imageID is not digest-addressed")
    pod_view = {
        **pod_meta,
        "ip": required_text(pod.get("status", {}).get("podIP"), f"{app} pod IP"),
        "node": "atius-srv-1",
    }
    container = {
        "name": container_name,
        "image_ref": image_ref,
        "image_id": image_id,
        "image_digest": digest_match.group(1),
        "runtime_digest": runtime_match.group(1),
        "restart_count": restart_count,
        "resources": {"requests_cpu": "500m", "limits_cpu": "500m"},
    }
    workloads[key] = {"controller": controller, "pod_owner": owner, "pod": pod_view, "container": container}
    pod_identity[app] = pod_view

service_names = {
    "router": "router-ai-atius",
    "redis": "router-ai-atius-redis",
    "postgres": "router-ai-atius-postgres",
}
services = {}
endpoint_slices = {}
for key, service_name in service_names.items():
    service_obj = load(f"service-{service_name}.json")
    service = metadata(service_obj, f"{service_name} Service")
    spec = service_obj.get("spec", {})
    if service["name"] != service_name or service_obj.get("metadata", {}).get("namespace") != namespace:
        raise SystemExit(f"{service_name} Service identity differs")
    if spec.get("type") != "ClusterIP" or spec.get("clusterIP") in (None, "", "None"):
        raise SystemExit(f"{service_name} Service is not ClusterIP")
    if spec.get("selector") != {"app.kubernetes.io/name": service_name}:
        raise SystemExit(f"{service_name} Service selector differs from the captured workload")
    if not spec.get("ports") or any("nodePort" in port for port in spec["ports"]):
        raise SystemExit(f"{service_name} Service ports are absent or externally exposed")
    service.update({
        "type": "ClusterIP",
        "cluster_ip": spec["clusterIP"],
        "cluster_ips": spec.get("clusterIPs", []),
        "selector": spec.get("selector", {}),
        "ports": sorted(
            [{
                "name": item.get("name"),
                "protocol": item.get("protocol", "TCP"),
                "port": item.get("port"),
                "target_port": item.get("targetPort"),
            } for item in spec.get("ports", [])],
            key=lambda item: (str(item["name"]), item["port"] or 0),
        ),
    })
    services[key] = service

    slices = []
    endpoint_count = 0
    for item in load(f"slices-{service_name}.json").get("items", []):
        slice_meta = metadata(item, f"{service_name} EndpointSlice")
        if item.get("metadata", {}).get("labels", {}).get("kubernetes.io/service-name") != service_name:
            raise SystemExit(f"{service_name} EndpointSlice label differs")
        endpoints = []
        for endpoint in item.get("endpoints", []):
            target = endpoint.get("targetRef", {})
            view = {
                "addresses": sorted(endpoint.get("addresses", [])),
                "conditions": {
                    "ready": endpoint.get("conditions", {}).get("ready"),
                    "serving": endpoint.get("conditions", {}).get("serving"),
                    "terminating": endpoint.get("conditions", {}).get("terminating"),
                },
                "node": endpoint.get("nodeName"),
                "target_ref": {
                    "kind": target.get("kind"),
                    "name": target.get("name"),
                    "namespace": target.get("namespace"),
                    "uid": target.get("uid"),
                },
            }
            expected_pod = pod_identity[service_name]
            if (view["conditions"]["ready"] is not True or view["node"] != "atius-srv-1" or
                    view["addresses"] != [expected_pod["ip"]] or view["target_ref"] != {
                        "kind": "Pod", "name": expected_pod["name"], "namespace": namespace, "uid": expected_pod["uid"]}):
                raise SystemExit(f"{service_name} EndpointSlice is not bound to the captured pod UID/IP")
            endpoints.append(view)
            endpoint_count += 1
        slices.append({
            **slice_meta,
            "address_type": item.get("addressType"),
            "ports": sorted(item.get("ports", []), key=lambda port: (str(port.get("name")), port.get("port") or 0)),
            "endpoints": sorted(endpoints, key=lambda endpoint: (endpoint["target_ref"]["uid"], endpoint["addresses"])),
        })
    if endpoint_count != 1:
        raise SystemExit(f"{service_name} requires exactly one ready endpoint")
    endpoint_slices[key] = sorted(slices, key=lambda item: item["name"])

storage = {}
for key, claim_name in {"router": "router-ai-atius-data", "postgres": "router-ai-atius-postgres-data"}.items():
    pvc_obj = load(f"pvc-{key}.json")
    pv_obj = load(f"pv-{key}.json")
    pvc = metadata(pvc_obj, f"{claim_name} PVC")
    pv = metadata(pv_obj, f"{claim_name} PV")
    volume_name = required_text(pvc_obj.get("spec", {}).get("volumeName"), f"{claim_name} volumeName")
    claim_ref = pv_obj.get("spec", {}).get("claimRef", {})
    if pvc["name"] != claim_name or pvc_obj.get("metadata", {}).get("namespace") != namespace:
        raise SystemExit(f"{claim_name} PVC identity differs")
    if pvc_obj.get("status", {}).get("phase") != "Bound" or pv_obj.get("status", {}).get("phase") != "Bound":
        raise SystemExit(f"{claim_name} PVC/PV is not Bound")
    if pv["name"] != volume_name:
        raise SystemExit(f"{claim_name} PVC volumeName differs from PV")
    if (claim_ref.get("namespace"), claim_ref.get("name"), claim_ref.get("uid")) != (namespace, claim_name, pvc["uid"]):
        raise SystemExit(f"{claim_name} PV claimRef is not bound by PVC UID")
    reclaim = pv_obj.get("spec", {}).get("persistentVolumeReclaimPolicy")
    if reclaim != "Retain":
        raise SystemExit(f"{claim_name} PV reclaim policy is not Retain")
    pvc.update({"phase": "Bound", "volume_name": volume_name})
    pv.update({
        "phase": "Bound",
        "reclaim_policy": reclaim,
        "claim_ref": {"namespace": namespace, "name": claim_name, "uid": pvc["uid"]},
    })
    storage[key] = {"pvc": pvc, "pv": pv, "binding_verified": True}

snapshot = {
    "schema_version": 1,
    "workloads": workloads,
    "services": services,
    "endpoint_slices": endpoint_slices,
    "storage": storage,
}
output.write_text(json.dumps(snapshot, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
PY
}

validate_snapshot_apply_chain() {
  local snapshot="$1" snapshot_sha
  snapshot_sha="$(sha256sum "$snapshot" | awk '{print $1}')"
  jq -e --arg sha "$snapshot_sha" --argjson live "$(cat "$snapshot")" '
    . as $apply |
    $apply.runtime_snapshot.sha256 == $sha and
    $apply.runtime_snapshot.map == $live and
    $apply.storage == $live.storage and
    all(["router", "postgres"][] as $key |
      $live.storage[$key] as $storage |
      any($apply.pvs[];
        .pvc == $storage.pvc.name and .pvc_uid == $storage.pvc.uid and
        .pv == $storage.pv.name and .reclaim_policy == "Retain" and .claim_uid_matched == true))
  ' "$apply_evidence" >/dev/null || die 'runtime snapshot is not exactly equal to the canonical shadow apply snapshot'
}

validate_runtime_stability() {
  local before="$1" after="$2"
  runtime_snapshot_pre_sha="$(sha256sum "$before" | awk '{print $1}')"
  runtime_snapshot_post_sha="$(sha256sum "$after" | awk '{print $1}')"
  if [ "$runtime_snapshot_pre_sha" != "$runtime_snapshot_post_sha" ] || ! cmp -s "$before" "$after"; then
    die 'runtime changed or rolled out during shadow smoke requests'
  fi
}

resolve_expected_models() {
  local output="$1" baseline_file="$2"
  validate_catalog_baseline "$baseline_file" || return 1
  jq -c '{source:.kind,models:.models,catalog_sha256:.catalog_sha256}' "$baseline_file" > "$output"
}

validate_models_payload() {
  local body="$1" expected="$2" summary="$3"
  python3 - "$body" "$expected" "$summary" <<'PY'
import hashlib
import json
import pathlib
import sys

body_path, expected_path, summary_path = map(pathlib.Path, sys.argv[1:])
payload = json.loads(body_path.read_text(encoding="utf-8"))
if not isinstance(payload, dict) or set(payload) != {"data"}:
    raise SystemExit("/v1/models root must contain only data")
rows = payload["data"]
if not isinstance(rows, list) or not rows:
    raise SystemExit("/v1/models data must be a non-empty array")

forbidden = {"pricing_source", "pricing_estimated", "pricing_version"}
def walk(value):
    if isinstance(value, dict):
        leaked = forbidden.intersection(value)
        if leaked:
            raise SystemExit(f"forbidden internal fields present: {sorted(leaked)}")
        for nested in value.values():
            walk(nested)
    elif isinstance(value, list):
        for nested in value:
            walk(nested)
walk(payload)

ids = []
for row in rows:
    if not isinstance(row, dict) or not isinstance(row.get("id"), str) or not row["id"]:
        raise SystemExit("/v1/models contains a row without a string id")
    ids.append(row["id"])
expected_doc = json.loads(expected_path.read_text(encoding="utf-8"))
expected = expected_doc["models"]
if len(ids) != len(set(ids)):
    raise SystemExit("/v1/models contains duplicate model ids")
if ids != expected:
    missing = sorted(set(expected).difference(ids))
    extra = sorted(set(ids).difference(expected))
    raise SystemExit(f"catalog is not the exact expected order; missing={missing}, extra={extra}")
expected_hash = hashlib.sha256(("\n".join(expected) + "\n").encode()).hexdigest()
actual_hash = hashlib.sha256(("\n".join(ids) + "\n").encode()).hexdigest()
summary_path.write_text(json.dumps({
    "expected_source": expected_doc["source"],
    "expected_count": len(expected),
    "expected_sha256": expected_hash,
    "catalog_count": len(ids),
    "catalog_sha256": actual_hash,
    "exact_order": True,
}, separators=(",", ":")), encoding="utf-8")
if expected_doc.get("catalog_sha256") != expected_hash:
    raise SystemExit("expected baseline checksum differs from its ordered model ids")
PY
}

curl_get() {
  local url="$1" output="$2"
  curl --silent --show-error --connect-timeout 5 --max-time 30 --output "$output" \
    --write-out '%{http_code}' "$url"
}

curl_get_authenticated() {
  local url="$1" output="$2" token="$3"
  printf 'Authorization: Bearer %s\n' "$token" | \
    curl --silent --show-error --connect-timeout 5 --max-time 30 --header @- \
      --output "$output" --write-out '%{http_code}' "$url"
}

write_smoke_evidence() {
  local output="$evidence_dir/smoke.json" tmp_file generated_at generated_epoch commit
  local apply_sha restore_sha baseline_sha cgroup image_digest apply_runtime_sha
  if [ -e "$output" ] || [ -L "$output" ]; then die 'smoke.json already exists'; fi
  generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  generated_epoch="$(date +%s)"
  commit="$(git rev-parse HEAD)"
  apply_sha="$(sha256sum "$apply_evidence" | awk '{print $1}')"
  restore_sha="$(sha256sum "$restore_evidence" | awk '{print $1}')"
  baseline_sha="$(sha256sum "$catalog_baseline" | awk '{print $1}')"
  cgroup="$(cpu_max_value)"
  image_digest="$(jq -r '.image.digest' "$apply_evidence")"
  apply_runtime_sha="$(jq -r '.runtime_snapshot.sha256' "$apply_evidence")"
  tmp_file="$(mktemp "$evidence_dir/.smoke.XXXXXX")"
  chmod 600 "$tmp_file"
  jq -n --arg generated_at "$generated_at" --argjson generated_at_epoch "$generated_epoch" \
    --arg cluster_uid "$cluster_uid" --arg commit "$commit" --arg cpu_max "$cgroup" \
    --arg cluster_ip "$cluster_ip" --arg image_digest "$image_digest" \
    --arg apply_sha256 "$apply_sha" --arg restore_sha256 "$restore_sha" --arg catalog_baseline_sha256 "$baseline_sha" \
    --arg apply_runtime_sha256 "$apply_runtime_sha" \
    --arg runtime_pre_sha256 "$runtime_snapshot_pre_sha" --arg runtime_post_sha256 "$runtime_snapshot_post_sha" \
    --argjson models "$(cat "$tmp/models-summary.json")" \
    --argjson images "$(jq '.images' "$apply_evidence")" \
    --argjson workloads "$(jq '.workloads' "$apply_evidence")" \
    --argjson apply_pvs "$(jq '.pvs' "$apply_evidence")" \
    --argjson runtime_pre "$(cat "$runtime_snapshot_pre")" \
    --argjson runtime_post "$(cat "$runtime_snapshot_post")" \
    '{schema_version:1,status:"go",generated_at:$generated_at,generated_at_epoch:$generated_at_epoch,
      cluster_uid:$cluster_uid,commit:$commit,cpu_max:$cpu_max,transport:{service:"router-ai-atius",type:"ClusterIP",cluster_ip:$cluster_ip,endpoints_ready:true},
      inputs:{shadow_apply_sha256:$apply_sha256,restore_sha256:$restore_sha256,image_digest:$image_digest,
        catalog_baseline_sha256:$catalog_baseline_sha256,apply_runtime_snapshot_sha256:$apply_runtime_sha256},images:$images,workloads:$workloads,
      runtime_snapshots:{equal:true,pre:{sha256:$runtime_pre_sha256,map:$runtime_pre},post:{sha256:$runtime_post_sha256,map:$runtime_post}},
      storage:$runtime_post.storage,
      storage_chain:{shadow_apply_sha256:$apply_sha256,apply_runtime_snapshot_sha256:$apply_runtime_sha256,
        matched:true,apply_pvs:$apply_pvs,apply_storage:$runtime_pre.storage,live:$runtime_post.storage},
      checks:{health_status:200,unauthorized_models_status:401,authenticated_models_status:200,root_data_only:true,internal_fields_absent:true,expected_models_present:true,embedding_model:"embedding-gte-v1",embedding_dimension:768},
      catalog:{expected_source:$models.expected_source,expected_count:$models.expected_count,expected_sha256:$models.expected_sha256,
        catalog_count:$models.catalog_count,catalog_sha256:$models.catalog_sha256,exact_order:$models.exact_order}}' > "$tmp_file"
  mv "$tmp_file" "$output"
}

self_test() {
  local test_dir expected restore_file state_file apply_file body summary service slices now cluster restore_sha baseline_sha
  local runtime_snapshot_pre runtime_snapshot_post runtime_snapshot_pre_sha runtime_snapshot_post_sha snapshot_sha
  test_dir="$(mktemp -d)"
  trap 'rm -rf "$test_dir"' RETURN
  evidence_dir="$test_dir"
  restore_file="$test_dir/restore.json"; state_file="$test_dir/state.json"; apply_file="$test_dir/apply.json"
  now="$(date +%s)"; cluster="cluster-test"
  jq -n --arg cluster "$cluster" --argjson now "$now" '{status:"go",restore_passed:true,cluster_uid:$cluster,generated_at_epoch:$now,target:{node:"atius-srv-1",database:"DBRouterAiAtius"}}' > "$restore_file"
  restore_sha="$(sha256sum "$restore_file" | awk '{print $1}')"
  jq -n --arg cluster "$cluster" --arg path "$restore_file" --arg sha "$restore_sha" '{schema_version:1,target:"router-ai-atius/DBRouterAiAtius@atius-srv-1",status:"go",cluster_uid:$cluster,evidence_path:$path,evidence_sha256:$sha}' > "$state_file"
  body="$test_dir/models.json"
  jq -n '{data:[{id:"gpt-test",object:"model"},{id:"embedding-gte-v1",object:"model"}]}' > "$body"
  write_catalog_baseline "$body" "$test_dir/catalog-baseline.json"
  validate_catalog_baseline "$test_dir/catalog-baseline.json"
  baseline_sha="$(sha256sum "$test_dir/catalog-baseline.json" | awk '{print $1}')"
  jq -n --arg sha "$restore_sha" --arg baseline_sha "$baseline_sha" '
    {inputs:{restore_sha256:$sha,catalog_baseline_sha256:$baseline_sha},
     images:{router:{reference:("example@sha256:"+("a"*64)),digest:("sha256:"+("a"*64)),runtime_digest:("sha256:"+("a"*64)),runtime_image_id:("example@sha256:"+("a"*64)),exact:true},
       redis:{reference:("redis@sha256:"+("b"*64)),digest:("sha256:"+("b"*64)),runtime_digest:("sha256:"+("b"*64)),runtime_image_id:("redis@sha256:"+("b"*64)),exact:true},
       postgres:{reference:("postgres@sha256:"+("c"*64)),digest:("sha256:"+("c"*64)),runtime_digest:("sha256:"+("c"*64)),runtime_image_id:("postgres@sha256:"+("c"*64)),exact:true}},
     workloads:{router:{app:"router-ai-atius",controller:{kind:"Deployment",name:"router-ai-atius",uid:"deployment-uid"},pod_owner:{name:"router-rs",uid:"rs-uid"},pod:{name:"router-abc",uid:"pod-uid",ip:"10.42.0.9"},container:{name:"router-ai-atius",image_ref:("example@sha256:"+("a"*64)),image_id:("example@sha256:"+("a"*64)),resources:{requests_cpu:"500m",limits_cpu:"500m"}}},
       redis:{app:"router-ai-atius-redis",controller:{kind:"Deployment",name:"router-ai-atius-redis",uid:"redis-deployment-uid"},pod_owner:{name:"redis-rs",uid:"redis-rs-uid"},pod:{name:"redis-abc",uid:"redis-pod-uid",ip:"10.42.0.10"},container:{name:"redis",image_ref:("redis@sha256:"+("b"*64)),image_id:("redis@sha256:"+("b"*64)),resources:{requests_cpu:"500m",limits_cpu:"500m"}}},
       postgres:{app:"router-ai-atius-postgres",controller:{kind:"StatefulSet",name:"router-ai-atius-postgres",uid:"postgres-sts-uid"},pod_owner:{name:"router-ai-atius-postgres",uid:"postgres-sts-uid"},pod:{name:"router-ai-atius-postgres-0",uid:"postgres-pod-uid",ip:"10.42.0.11"},container:{name:"postgres",image_ref:("postgres@sha256:"+("c"*64)),image_id:("postgres@sha256:"+("c"*64)),resources:{requests_cpu:"500m",limits_cpu:"500m"}}}},
     pvs:[
       {pvc:"router-ai-atius-data",pvc_uid:"router-pvc-uid",pv:"router-pv",reclaim_policy:"Retain",claim_uid_matched:true},
       {pvc:"router-ai-atius-postgres-data",pvc_uid:"postgres-pvc-uid",pv:"postgres-pv",reclaim_policy:"Retain",claim_uid_matched:true}],
     services:{router:{cluster_ip:"10.43.0.10"},redis:{cluster_ip:"10.43.0.11"}}}' > "$apply_file"
  chmod 600 "$restore_file" "$state_file" "$apply_file"
  apply_evidence="$apply_file"
  validate_restore_chain "$restore_file" "$state_file" "$apply_file" "$cluster" "$now"
  jq '.inputs.restore_sha256="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"' "$apply_file" > "$test_dir/mixed-apply.json"
  if (validate_restore_chain "$restore_file" "$state_file" "$test_dir/mixed-apply.json" "$cluster" "$now") 2>/dev/null; then die 'mixed restore/apply evidence was accepted'; fi
  expected="$test_dir/expected.json"
  resolve_expected_models "$expected" "$test_dir/catalog-baseline.json"
  summary="$test_dir/summary.json"
  validate_models_payload "$body" "$expected" "$summary"
  jq '.data[0].pricing_source="internal"' "$body" > "$test_dir/leak.json"
  if (validate_models_payload "$test_dir/leak.json" "$expected" "$summary") 2>/dev/null; then die 'internal field leak was accepted'; fi
  jq '.data=[{"id":"embedding-gte-v1"}]' "$body" > "$test_dir/missing.json"
  if (validate_models_payload "$test_dir/missing.json" "$expected" "$summary") 2>/dev/null; then die 'missing expected model was accepted'; fi
  jq '.data += [{"id":"unexpected"}]' "$body" > "$test_dir/extra.json"
  if (validate_models_payload "$test_dir/extra.json" "$expected" "$summary") 2>/dev/null; then die 'extra model was accepted'; fi
  jq '.data += [{"id":"embedding-gte-v1"}]' "$body" > "$test_dir/duplicate.json"
  if (validate_models_payload "$test_dir/duplicate.json" "$expected" "$summary") 2>/dev/null; then die 'duplicate model was accepted'; fi
  jq '.data |= reverse' "$body" > "$test_dir/misordered.json"
  if (validate_models_payload "$test_dir/misordered.json" "$expected" "$summary") 2>/dev/null; then die 'misordered catalog was accepted'; fi
  jq '.catalog_sha256=("0"*64)' "$test_dir/catalog-baseline.json" > "$test_dir/tampered-baseline.json"
  chmod 600 "$test_dir/tampered-baseline.json"
  if (resolve_expected_models "$expected" "$test_dir/tampered-baseline.json") 2>/dev/null; then die 'tampered catalog baseline was accepted'; fi
  service="$test_dir/service.json"
  jq -n '{metadata:{name:"router-ai-atius"},spec:{type:"ClusterIP",selector:{"app.kubernetes.io/name":"router-ai-atius"},clusterIP:"10.43.0.10",clusterIPs:["10.43.0.10"],ports:[{port:3000}]}}' > "$service"
  validate_service_json "$service"
  jq '.spec.type="NodePort"' "$service" > "$test_dir/nodeport.json"
  if (validate_service_json "$test_dir/nodeport.json") 2>/dev/null; then die 'NodePort was accepted'; fi
  jq -n '{items:[{metadata:{name:"router-abc",uid:"pod-uid",labels:{"app.kubernetes.io/name":"router-ai-atius"},ownerReferences:[{controller:true,kind:"ReplicaSet",name:"router-rs",uid:"rs-uid"}]},spec:{nodeName:"atius-srv-1",containers:[{name:"router-ai-atius",image:("example@sha256:"+("a"*64)),resources:{requests:{cpu:"500m"},limits:{cpu:"500m"}}}]},status:{phase:"Running",podIP:"10.42.0.9",conditions:[{type:"Ready",status:"True"}],containerStatuses:[{ready:true,imageID:("example@sha256:"+("a"*64))}]}}]}' > "$test_dir/pods.json"
  validate_workload_json "$test_dir/pods.json" router-ai-atius "example@sha256:$(printf 'a%.0s' {1..64})"
  jq '.workloads.router' "$apply_file" > "$test_dir/router-identity.json"
  validate_workload_chain router "$test_dir/router-identity.json"
  jq '.pod.uid="replacement-pod"' "$test_dir/router-identity.json" > "$test_dir/replaced-router.json"
  if (validate_workload_chain router "$test_dir/replaced-router.json") 2>/dev/null; then die 'replacement router pod was accepted'; fi
  jq '.images.redis.exact=false' "$apply_file" > "$test_dir/mutable-redis-apply.json"
  apply_evidence="$test_dir/mutable-redis-apply.json"
  jq '.workloads.redis' "$apply_file" > "$test_dir/redis-identity.json"
  if (validate_workload_chain redis "$test_dir/redis-identity.json") 2>/dev/null; then die 'mutable Redis apply image was accepted'; fi
  apply_evidence="$apply_file"
  jq '.workloads.postgres | .pod.uid="replacement-postgres"' "$apply_file" > "$test_dir/replaced-postgres.json"
  if (validate_workload_chain postgres "$test_dir/replaced-postgres.json") 2>/dev/null; then die 'replacement PostgreSQL pod was accepted'; fi
  slices="$test_dir/slices.json"
  jq -n '{items:[{metadata:{labels:{"kubernetes.io/service-name":"router-ai-atius"}},endpoints:[{addresses:["10.42.0.9"],conditions:{ready:true},nodeName:"atius-srv-1",targetRef:{kind:"Pod",namespace:"router-ai-atius",name:"router-abc",uid:"pod-uid"}}]}]}' > "$slices"
  validate_endpointslice_json "$slices" "$test_dir/pods.json"
  jq '.items[0].endpoints[0].nodeName="atius-srv-2"' "$slices" > "$test_dir/wrong-node.json"
  if (validate_endpointslice_json "$test_dir/wrong-node.json" "$test_dir/pods.json") 2>/dev/null; then die 'wrong EndpointSlice node was accepted'; fi
  jq '.items[0].endpoints[0].targetRef.uid="other-pod"' "$slices" > "$test_dir/wrong-pod.json"
  if (validate_endpointslice_json "$test_dir/wrong-pod.json" "$test_dir/pods.json") 2>/dev/null; then die 'wrong EndpointSlice pod was accepted'; fi
  runtime_snapshot_pre="$test_dir/runtime-pre.json"
  runtime_snapshot_post="$test_dir/runtime-post.json"
  jq -S -c --argjson apply "$(cat "$apply_file")" '
    def workload($key; $controller_rv; $owner_rv; $pod_rv):
      $apply.workloads[$key] as $item |
      {controller:($item.controller + {resource_version:$controller_rv,generation:1,observed_generation:1,revision:null}),
       pod_owner:($item.pod_owner + {kind:(if $key == "postgres" then "StatefulSet" else "ReplicaSet" end),resource_version:$owner_rv}),
       pod:($item.pod + {resource_version:$pod_rv,node:"atius-srv-1"}),
       container:{name:$item.container.name,image_ref:$item.container.image_ref,image_id:$item.container.image_id,
         image_digest:$apply.images[$key].digest,runtime_digest:$apply.images[$key].runtime_digest,
         restart_count:0,resources:$item.container.resources}};
    {schema_version:1,
     workloads:{router:workload("router";"101";"102";"103"),redis:workload("redis";"201";"202";"203"),postgres:workload("postgres";"301";"301";"303")},
     services:{router:{name:"router-ai-atius",uid:"router-service-uid",resource_version:"401",type:"ClusterIP",cluster_ip:"10.43.0.10"},
       redis:{name:"router-ai-atius-redis",uid:"redis-service-uid",resource_version:"402",type:"ClusterIP",cluster_ip:"10.43.0.11"},
       postgres:{name:"router-ai-atius-postgres",uid:"postgres-service-uid",resource_version:"403",type:"ClusterIP",cluster_ip:"10.43.0.12"}},
     endpoint_slices:{router:[{name:"router-slice",uid:"router-slice-uid",resource_version:"501"}],redis:[{name:"redis-slice",uid:"redis-slice-uid",resource_version:"502"}],postgres:[{name:"postgres-slice",uid:"postgres-slice-uid",resource_version:"503"}]},
     storage:{router:{pvc:{name:"router-ai-atius-data",uid:"router-pvc-uid",resource_version:"601",phase:"Bound",volume_name:"router-pv"},
       pv:{name:"router-pv",uid:"router-pv-uid",resource_version:"602",phase:"Bound",reclaim_policy:"Retain",claim_ref:{namespace:"router-ai-atius",name:"router-ai-atius-data",uid:"router-pvc-uid"}},binding_verified:true},
       postgres:{pvc:{name:"router-ai-atius-postgres-data",uid:"postgres-pvc-uid",resource_version:"603",phase:"Bound",volume_name:"postgres-pv"},
       pv:{name:"postgres-pv",uid:"postgres-pv-uid",resource_version:"604",phase:"Bound",reclaim_policy:"Retain",claim_ref:{namespace:"router-ai-atius",name:"router-ai-atius-postgres-data",uid:"postgres-pvc-uid"}},binding_verified:true}}}' \
    <<< '{}' > "$runtime_snapshot_pre"
  cp "$runtime_snapshot_pre" "$runtime_snapshot_post"
  snapshot_sha="$(sha256sum "$runtime_snapshot_pre" | awk '{print $1}')"
  jq --arg sha "$snapshot_sha" --argjson snapshot "$(cat "$runtime_snapshot_pre")" \
    '.runtime_snapshot={sha256:$sha,map:$snapshot} | .storage=$snapshot.storage' \
    "$apply_file" > "$test_dir/apply-with-runtime.json"
  mv "$test_dir/apply-with-runtime.json" "$apply_file"
  chmod 600 "$apply_file"
  apply_evidence="$apply_file"
  validate_snapshot_apply_chain "$runtime_snapshot_pre"
  validate_runtime_stability "$runtime_snapshot_pre" "$runtime_snapshot_post"
  jq -S -c '.workloads.router.container.restart_count=1' "$runtime_snapshot_pre" > "$test_dir/restarted-runtime.json"
  if (validate_runtime_stability "$runtime_snapshot_pre" "$test_dir/restarted-runtime.json") 2>/dev/null; then die 'runtime restart during smoke was accepted'; fi
  jq -S -c '.storage.router.pvc.uid="replacement-pvc"' "$runtime_snapshot_pre" > "$test_dir/replaced-pvc-runtime.json"
  if (validate_snapshot_apply_chain "$test_dir/replaced-pvc-runtime.json") 2>/dev/null; then die 'replacement PVC UID was accepted'; fi
  jq -S -c '.storage.postgres.pv.uid="replacement-pv"' "$runtime_snapshot_pre" > "$test_dir/replaced-pv-runtime.json"
  if (validate_snapshot_apply_chain "$test_dir/replaced-pv-runtime.json") 2>/dev/null; then die 'replacement PV UID matched shadow apply'; fi
  if (validate_runtime_stability "$runtime_snapshot_pre" "$test_dir/replaced-pv-runtime.json") 2>/dev/null; then die 'replacement PV UID during smoke was accepted'; fi
  jq -S -c '.storage.postgres.pv.reclaim_policy="Delete"' "$runtime_snapshot_pre" > "$test_dir/delete-pv-runtime.json"
  if (validate_snapshot_apply_chain "$test_dir/delete-pv-runtime.json") 2>/dev/null; then die 'non-Retain PV was accepted'; fi
  quota_ok '80000 100000'
  if (quota_ok 'max 100000') 2>/dev/null; then die 'unbounded quota was accepted'; fi
  echo 'smoke self-test: PASS'
}

if [ "${PHASE29_SNAPSHOT_LIBRARY:-0}" = 1 ]; then
  [ "${BASH_SOURCE[0]}" != "$0" ] || die 'snapshot library mode may only be sourced'
  return 0
fi

while [ "$#" -gt 0 ]; do
  case "$1" in
    --strict) strict=true ;;
    --capture-baseline) capture_baseline=true ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --self-test) self_test; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

[ -n "${ATIUS_ROUTER_TOKEN:-}" ] || die 'ATIUS_ROUTER_TOKEN is required; authenticated smoke cannot be skipped'
case "$ATIUS_ROUTER_TOKEN" in
  *$'\r'*|*$'\n'*) die 'ATIUS_ROUTER_TOKEN contains a forbidden line break' ;;
esac
if $capture_baseline; then
  $strict && die '--capture-baseline cannot be combined with --strict'
  require_evidence_directory
  if [ -e "$evidence_dir/shadow-apply.json" ] || [ -L "$evidence_dir/shadow-apply.json" ]; then
    die 'catalog baseline must be captured before shadow apply'
  fi
  if kube -n "$namespace" get deployment router-ai-atius >/dev/null 2>&1; then
    die 'catalog baseline capture requires the shadow router Deployment to be absent'
  fi
  if [ "$(kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius -o json | jq '.items | length')" -ne 0 ]; then
    die 'catalog baseline capture requires shadow router pods to be absent'
  fi
  tmp="$(mktemp -d /dev/shm/phase29-catalog-baseline.XXXXXX)"
  trap 'rm -rf "$tmp"' EXIT
  if ! auth_status="$(curl_get_authenticated "$baseline_url/v1/models" "$tmp/models.json" "$ATIUS_ROUTER_TOKEN")"; then
    die 'authenticated pre-shadow catalog request failed'
  fi
  [ "$auth_status" = 200 ] || die "authenticated pre-shadow /v1/models returned HTTP $auth_status"
  write_catalog_baseline "$tmp/models.json" "$evidence_dir/catalog-baseline.json"
  echo 'authenticated pre-shadow catalog baseline: PASS'
  exit 0
fi

$strict || die '--strict is required; partial smoke is forbidden'
[ -n "${K3S_ROUTER_BASE_URL:-}" ] || die 'K3S_ROUTER_BASE_URL is required'
require_evidence_directory
quota_ok "$(cpu_max_value)"

apply_evidence="$evidence_dir/shadow-apply.json"
restore_evidence="$evidence_dir/restore.json"
catalog_baseline="$evidence_dir/catalog-baseline.json"
require_regular_json "$apply_evidence" shadow-apply.json
require_regular_json "$restore_evidence" restore.json
validate_catalog_baseline "$catalog_baseline"
[ "$(stat -c %U:%a "$apply_evidence")" = "$(id -un):600" ] || die 'shadow-apply.json owner/mode must be caller:600'
[ "$(stat -c %U:%a "$restore_evidence")" = "$(id -un):600" ] || die 'restore.json owner/mode must be caller:600'

tmp="$(mktemp -d /dev/shm/phase29-shadow-smoke.XXXXXX)"
trap 'rm -rf "$tmp"' EXIT
kube -n "$namespace" get service router-ai-atius -o json > "$tmp/service.json"
kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius -o json > "$tmp/router-pods.json"
kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius-redis -o json > "$tmp/redis-pods.json"
kube -n "$namespace" get pods -l app.kubernetes.io/name=router-ai-atius-postgres -o json > "$tmp/postgres-pods.json"
kube -n "$namespace" get endpointslices.discovery.k8s.io \
  -l kubernetes.io/service-name=router-ai-atius -o json > "$tmp/endpointslices.json"
router_image="$(jq -r '.images.router.reference // ""' "$apply_evidence")"
redis_image="$(jq -r '.images.redis.reference // ""' "$apply_evidence")"
postgres_image="$(jq -r '.images.postgres.reference // ""' "$apply_evidence")"
router_runtime_digest="$(jq -r '.images.router.runtime_digest // .images.router.digest // ""' "$apply_evidence")"
redis_runtime_digest="$(jq -r '.images.redis.runtime_digest // .images.redis.digest // ""' "$apply_evidence")"
postgres_runtime_digest="$(jq -r '.images.postgres.runtime_digest // .images.postgres.digest // ""' "$apply_evidence")"
validate_workload_json "$tmp/router-pods.json" router-ai-atius "$router_image" "$router_runtime_digest"
validate_workload_json "$tmp/redis-pods.json" router-ai-atius-redis "$redis_image" "$redis_runtime_digest"
validate_workload_json "$tmp/postgres-pods.json" router-ai-atius-postgres "$postgres_image" "$postgres_runtime_digest"
workload_identity router-ai-atius deployment router-ai-atius "$router_image" "$tmp/router-pods.json" "$tmp/router-identity.json"
workload_identity router-ai-atius-redis deployment router-ai-atius-redis "$redis_image" "$tmp/redis-pods.json" "$tmp/redis-identity.json"
workload_identity router-ai-atius-postgres statefulset router-ai-atius-postgres "$postgres_image" "$tmp/postgres-pods.json" "$tmp/postgres-identity.json"
validate_workload_chain router "$tmp/router-identity.json"
validate_workload_chain redis "$tmp/redis-identity.json"
validate_workload_chain postgres "$tmp/postgres-identity.json"
validate_service_json "$tmp/service.json"
validate_endpointslice_json "$tmp/endpointslices.json" "$tmp/router-pods.json"
cluster_ip="$(jq -r '.spec.clusterIP' "$tmp/service.json")"
service_port="$(jq -r '.spec.ports[0].port' "$tmp/service.json")"
base_url="http://${cluster_ip}:${service_port}"
[ "${K3S_ROUTER_BASE_URL%/}" = "$base_url" ] ||
  die 'K3S_ROUTER_BASE_URL does not equal the live Service ClusterIP URL'
cluster_uid="$(kube get namespace kube-system -o jsonpath='{.metadata.uid}')"
now="$(date +%s)"
validate_restore_chain "$restore_evidence" "$restore_state" "$apply_evidence" "$cluster_uid" "$now"
baseline_sha="$(sha256sum "$catalog_baseline" | awk '{print $1}')"
jq -e --arg baseline_sha "$baseline_sha" --arg catalog_sha "$(jq -r '.catalog_sha256' "$catalog_baseline")" '
  .inputs.catalog_baseline_sha256 == $baseline_sha and .catalog.expected_sha256 == $catalog_sha
' "$apply_evidence" >/dev/null || die 'shadow apply is bound to a different catalog baseline checksum'
jq -e --arg cluster "$cluster_uid" --arg ip "$cluster_ip" --argjson now "$now" '
  .status == "go" and .cluster_uid == $cluster and .services.router.type == "ClusterIP" and
  .services.router.cluster_ip == $ip and .services.router.endpoints_ready == true and
  .placement.node == "atius-srv-1" and .placement.router_ready == true and
  .placement.redis_ready_before_router == true and .placement.cpu_per_pod == "500m" and
  .image.exact == true and (.generated_at_epoch | type == "number") and
  .generated_at_epoch <= $now and ($now - .generated_at_epoch) <= 3600 and
  .mutations.apache == false and .mutations.podman == false
' "$apply_evidence" >/dev/null || die 'shadow apply evidence is incomplete, stale, or belongs elsewhere'
resolve_expected_models "$tmp/expected-models.json" "$catalog_baseline"
runtime_snapshot_pre="$tmp/runtime-snapshot-pre.json"
runtime_snapshot_post="$tmp/runtime-snapshot-post.json"
capture_runtime_snapshot "$runtime_snapshot_pre" pre
validate_snapshot_apply_chain "$runtime_snapshot_pre"

if ! health_status="$(curl_get "$base_url/api/status" "$tmp/health.out")"; then
  die 'health request failed'
fi
[ "$health_status" = 200 ] || die "health check returned HTTP $health_status"
if ! unauth_status="$(curl_get "$base_url/v1/models" "$tmp/models-unauthorized.out")"; then
  die 'unauthenticated models request failed'
fi
[ "$unauth_status" = 401 ] || die "unauthenticated /v1/models returned HTTP $unauth_status"
if ! auth_status="$(curl_get_authenticated "$base_url/v1/models" "$tmp/models.json" "$ATIUS_ROUTER_TOKEN")"; then
  die 'authenticated models request failed'
fi
[ "$auth_status" = 200 ] || die "authenticated /v1/models returned HTTP $auth_status"
validate_models_payload "$tmp/models.json" "$tmp/expected-models.json" "$tmp/models-summary.json"

unset ATIUS_ROUTER_ACCEPT_UPSTREAM_ERROR ATIUS_ROUTER_EXPECT_CHANNEL_NAME
if ! ATIUS_ROUTER_BASE_URL="$base_url" \
  ATIUS_ROUTER_EMBEDDINGS_BASE_URL="$base_url/v1" \
  ATIUS_ROUTER_TOKEN="$ATIUS_ROUTER_TOKEN" \
  ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 \
  ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 \
  python3 scripts/smoke-embeddings.py > "$tmp/embedding.out"; then
  die 'embedding-gte-v1 smoke failed'
fi
grep -Eq 'dimension=768([[:space:]]|$)' "$tmp/embedding.out" ||
  die 'embedding smoke did not prove dimension 768'
capture_runtime_snapshot "$runtime_snapshot_post" post
validate_snapshot_apply_chain "$runtime_snapshot_post"
validate_runtime_stability "$runtime_snapshot_pre" "$runtime_snapshot_post"
quota_ok "$(cpu_max_value)"
write_smoke_evidence
echo 'shadow smoke strict: PASS'
