#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

mode=dry-run
evidence_dir=""
output=""
run_id=""
rollback_file=""
identity_file=""
evidence_root="${PHASE29_EVIDENCE_ROOT:-$HOME/.local/state/router-ai-atius/phase29}"
fresh_seconds="${PHASE29_EVIDENCE_MAX_AGE_SECONDS:-3600}"
cleanup_max_age="${PHASE29_CLEANUP_MAX_AGE_SECONDS:-604800}"
stable_seconds="${PHASE29_REQUIRE_STABLE_SECONDS:-300}"

die() {
  echo "go/no-go failed: $*" >&2
  exit 1
}

quota_ok() {
  local value="$1" quota period
  read -r quota period <<< "$value"
  [[ "$quota" =~ ^[0-9]+$ && "$period" =~ ^[0-9]+$ ]] || return 1
  [ "$period" -gt 0 ] && [ $((quota * 10)) -le $((period * 8)) ]
}

manifest_hash() {
  sha256sum k8s/router-ai-atius/*.yaml | sha256sum | awk '{print $1}'
}

regular_json() {
  local file="$1"
  [ -f "$file" ] && [ ! -L "$file" ] && jq -e 'type == "object"' "$file" >/dev/null 2>&1
}

fresh_file() {
  local file="$1" now="$2" max_age="$3"
  jq -e --argjson now "$now" --argjson max "$max_age" '
    (.generated_at_epoch | type == "number") and .generated_at_epoch <= $now and
    ($now - .generated_at_epoch) <= $max
  ' "$file" >/dev/null
}

record_gate() {
  local name="$1" status="$2" reason="$3"
  jq -n --arg name "$name" --arg status "$status" --arg reason "$reason" \
    '{name:$name,status:$status,reason:$reason}' >> "$gates_file"
}

pass_gate() { record_gate "$1" pass "$2"; }
fail_gate() { record_gate "$1" fail "$2"; }

validate_evidence() {
  local cluster_uid="$1" now="$2" current_manifest="$3"
  local cleanup="$evidence_dir/cleanup.json" bootstrap="$evidence_dir/bootstrap.json"
  local backup="$evidence_dir/backup.json" restore="$evidence_dir/restore.json"
  local apply="$evidence_dir/shadow-apply.json" smoke="$evidence_dir/smoke.json"
  local rollback="$rollback_file" file

  for file in "$cleanup" "$bootstrap" "$backup" "$restore" "$apply" "$smoke" "$rollback"; do
    if regular_json "$file"; then pass_gate "artifact:$(basename "$file")" 'regular JSON artifact present';
    else fail_gate "artifact:$(basename "$file")" 'artifact missing, symlinked, or invalid JSON'; fi
  done

  if regular_json "$cleanup" && fresh_file "$cleanup" "$now" "$cleanup_max_age" &&
    jq -e --arg cluster "$cluster_uid" '
      .status == "go" and .cluster_uid == $cluster and .reclaimed_bytes >= 21474836480 and
      .free_percent >= 25 and .stable_seconds >= 300
    ' "$cleanup" >/dev/null && quota_ok "$(jq -r '.cpu_max' "$cleanup")"; then
    pass_gate cleanup 'cleanup is cluster-bound, checksummed by the decision, and meets recovery/stability gates'
  else fail_gate cleanup 'cleanup is stale, foreign, incomplete, or below disk/CPU gates'; fi

  if regular_json "$bootstrap" && fresh_file "$bootstrap" "$now" "$fresh_seconds" &&
    jq -e --arg cluster "$cluster_uid" --arg manifest "$current_manifest" '
      .status == "go" and .cluster_uid == $cluster and .exclusive_node == "atius-srv-1" and
      .secret_keys == "POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET" and
      .manifest_sha256 == $manifest and .digest_match == true and
      (.manifest_digest | test("^sha256:[0-9a-f]{64}$")) and
      (.image_ref | test("@sha256:[0-9a-f]{64}$"))
    ' "$bootstrap" >/dev/null && quota_ok "$(jq -r '.cpu_max' "$bootstrap")"; then
    pass_gate bootstrap 'bootstrap is fresh, cluster/manifest-bound, exclusive, and immutable'
  else fail_gate bootstrap 'bootstrap freshness, identity, Secret-key, digest, label, or CPU gate failed'; fi

  if regular_json "$backup" && fresh_file "$backup" "$now" "$fresh_seconds" &&
    jq -e '
      .status == "go" and .source.kind == "host-postgresql" and .source.host == "127.0.0.1" and
      .source.port == 8745 and (.source.server_version_num | test("^17[0-9]{4}$")) and
      (.source.systemd_unit == "postgresql@17-main" or .source.systemd_unit == "postgresql@17-main.service") and
      .source.backend_unit_matched == true and .pgbouncer_crosscheck.matched == true and
      (.pg_dump_version | test("^17\\.[0-9]+([.][0-9]+)?$")) and
      .dump.size_bytes > 0 and (.dump.sha256 | test("^[0-9a-f]{64}$")) and
      .dump.structurally_valid == true and .cpu.aggregate_millicores <= 800 and
      .cpu.postgres_quota_restored == true and
      .database_inventory.format == "phase29-database-inventory-v2" and
      (.database_inventory.sha256 | test("^[0-9a-f]{64}$")) and
      (.database_inventory.schema_ddl_sha256 | test("^[0-9a-f]{64}$"))
    ' "$backup" >/dev/null && quota_ok "$(jq -r '.cpu_max' "$backup")"; then
    pass_gate backup 'fresh canonical PostgreSQL 17 backup and PgBouncer cross-check are green'
  else fail_gate backup 'backup origin, freshness, checksum, PostgreSQL major, structure, or quota gate failed'; fi

  if regular_json "$restore" && fresh_file "$restore" "$now" "$fresh_seconds" &&
    jq -e --arg cluster "$cluster_uid" '
      .status == "go" and .restore_passed == true and .cluster_uid == $cluster and
      .backup.source == "host-postgresql-17" and .target.node == "atius-srv-1" and
      .target.database == "DBRouterAiAtius" and .target.clean_before_restore == true and
      (.target.server_version_num | test("^17[0-9]{4}$")) and (.pvs | length) >= 1 and
      .database_inventory.format == "phase29-database-inventory-v2" and
      .database_inventory.source_backup_target_matched == true and .database_inventory.matched == true and
      .database_inventory.source_sha256 == .database_inventory.target_sha256 and
      .database_inventory.source_schema_ddl_sha256 == .database_inventory.target_schema_ddl_sha256 and
      all(.pvs[]; .reclaim_policy == "Retain" and .claim_uid_matched == true) and
      .runtime_stage.redis_applied == false and .runtime_stage.router_applied == false
    ' "$restore" >/dev/null && quota_ok "$(jq -r '.cpu_max' "$restore")" &&
    [ "$(jq -r '.backup.sha256' "$restore")" = "$(jq -r '.dump.sha256' "$backup" 2>/dev/null)" ]; then
    pass_gate restore 'fresh atomic restore is clean, PostgreSQL 17, backup-bound, and Retain-protected'
  else fail_gate restore 'restore freshness, cluster/backup identity, target, Retain, or CPU gate failed'; fi

  if regular_json "$apply" && fresh_file "$apply" "$now" "$fresh_seconds" &&
    jq -e --arg cluster "$cluster_uid" --arg manifest "$current_manifest" '
      .status == "go" and .cluster_uid == $cluster and .inputs.manifest_sha256 == $manifest and
      .image.exact == true and (.image.digest | test("^sha256:[0-9a-f]{64}$")) and
      all([.images.router,.images.redis,.images.postgres][]; . as $image |
        $image.exact == true and ($image.reference | test("@sha256:[0-9a-f]{64}$")) and
        ($image.digest | test("^sha256:[0-9a-f]{64}$")) and ($image.runtime_image_id | endswith($image.digest))) and
      all([.workloads.router,.workloads.redis,.workloads.postgres][];
        (.controller.uid | length) > 0 and (.pod.uid | length) > 0 and (.pod.ip | length) > 0 and
        (.container.image_ref | test("@sha256:[0-9a-f]{64}$")) and
        (.container.image_id | test("sha256:[0-9a-f]{64}$")) and
        .container.resources.requests_cpu == "500m" and .container.resources.limits_cpu == "500m") and
      .placement.node == "atius-srv-1" and .placement.postgres_ready == true and
      .placement.redis_ready_before_router == true and .placement.router_ready == true and
      .placement.cpu_per_pod == "500m" and (.pvs | length) >= 2 and
      all(.pvs[]; .reclaim_policy == "Retain" and .claim_uid_matched == true) and
      .services.redis.type == "ClusterIP" and .services.redis.endpoints_ready == true and
      .services.router.type == "ClusterIP" and .services.router.endpoints_ready == true and
      (.services.router.cluster_ip | test("^[0-9a-fA-F:.]+$")) and
      .mutations.apache == false and .mutations.podman == false
    ' "$apply" >/dev/null && quota_ok "$(jq -r '.cpu_max' "$apply")" &&
    [ "$(jq -r '.inputs.restore_sha256' "$apply")" = "$(sha256sum "$restore" | awk '{print $1}')" ] &&
    [ "$(jq -r '.inputs.bootstrap_sha256' "$apply")" = "$(sha256sum "$bootstrap" | awk '{print $1}')" ] &&
    [ "$(jq -r '.image.digest' "$apply")" = "$(jq -r '.manifest_digest' "$bootstrap")" ]; then
    pass_gate shadow-apply 'fresh apply is checksum-bound, srv1-only, immutable, Retain, and ClusterIP-only'
  else fail_gate shadow-apply 'apply freshness, checksum chain, placement, image, PV, Service, or CPU gate failed'; fi

  if regular_json "$smoke" && fresh_file "$smoke" "$now" "$fresh_seconds" &&
    jq -e --arg cluster "$cluster_uid" '
      .status == "go" and .cluster_uid == $cluster and .transport.type == "ClusterIP" and
      .transport.endpoints_ready == true and .checks.health_status == 200 and
      .checks.unauthorized_models_status == 401 and .checks.authenticated_models_status == 200 and
      .checks.root_data_only == true and .checks.internal_fields_absent == true and
      .checks.expected_models_present == true and .checks.embedding_model == "embedding-gte-v1" and
      .checks.embedding_dimension == 768
    ' "$smoke" >/dev/null && quota_ok "$(jq -r '.cpu_max' "$smoke")" &&
    [ "$(jq -r '.inputs.shadow_apply_sha256' "$smoke")" = "$(sha256sum "$apply" | awk '{print $1}')" ] &&
    [ "$(jq -r '.inputs.restore_sha256' "$smoke")" = "$(sha256sum "$restore" | awk '{print $1}')" ] &&
    [ "$(jq -r '.inputs.image_digest' "$smoke")" = "$(jq -r '.image.digest' "$apply")" ] &&
    [ "$(jq -r '.transport.cluster_ip' "$smoke")" = "$(jq -r '.services.router.cluster_ip' "$apply")" ]; then
    pass_gate smoke 'strict authenticated smoke is fresh and checksum/image/ClusterIP-bound'
  else fail_gate smoke 'smoke freshness, auth/API contract, checksum chain, embedding, or transport gate failed'; fi

  if regular_json "$rollback" && fresh_file "$rollback" "$now" "$fresh_seconds" &&
    jq -e --arg run_id "$run_id" '
      .schema_version == 2 and .status == "go" and .run_id == $run_id and
      (.podman.unit.required == false) and
      ((.podman.unit.present == false) or .podman.unit.active == true) and
      .podman.pod.exists == true and .podman.pod.running == true and .podman.containers_ready == true and
      .podman.limits_valid == true and .podman.health_ok == true and .podman.clianything_ok == true and
      .podman.clianything_backend == "podman" and
      .apache.syntax_ok == true and .apache.vhost_selection_ok == true and
      .apache.selected_vhost == "/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf" and
      .apache.routes_to_podman == true and
      .apache.config_path == "/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf" and
      .apache.k3s_target_present == false and .read_only == true and
      .mutations.apache == false and .mutations.podman == false and .mutations.k3s == false
    ' "$rollback" >/dev/null; then
    pass_gate rollback 'Podman rollback and Apache edge remain healthy and read-only'
  else fail_gate rollback 'Podman runtime or Apache edge rollback proof is stale or not green'; fi

  if regular_json "$bootstrap" && regular_json "$restore" && regular_json "$apply" && regular_json "$smoke" &&
    [ "$(jq -r '.cluster_uid' "$bootstrap")" = "$cluster_uid" ] &&
    [ "$(jq -r '.cluster_uid' "$restore")" = "$cluster_uid" ] &&
    [ "$(jq -r '.cluster_uid' "$apply")" = "$cluster_uid" ] &&
    [ "$(jq -r '.cluster_uid' "$smoke")" = "$cluster_uid" ]; then
    pass_gate cluster-identity 'all cluster-bound artifacts identify the current cluster UID'
  else fail_gate cluster-identity 'artifact cluster UIDs are missing or contradictory'; fi
}

validate_and_write_identity_map() {
  local pods="$1" controllers="$2" services="$3" endpoints="$4" pvcs="$5" pvs="$6"
  local apply="$7" smoke="$8" manifest="$9" destination="${10}"
  python3 - "$pods" "$controllers" "$services" "$endpoints" "$pvcs" "$pvs" \
    "$apply" "$smoke" "$manifest" "$destination" <<'PY'
import hashlib
import json
import pathlib
import re
import sys

pod_path, controller_path, service_path, endpoint_path, pvc_path, pv_path, apply_path, smoke_path, manifest_hash, output_path = map(pathlib.Path, sys.argv[1:])
manifest_hash = str(manifest_hash)

def load(path):
    return json.loads(path.read_text(encoding="utf-8"))

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

pods = load(pod_path).get("items", [])
controllers = load(controller_path).get("items", [])
services = load(service_path).get("items", [])
slices = load(endpoint_path).get("items", [])
pvcs = load(pvc_path).get("items", [])
pvs = load(pv_path).get("items", [])
apply = load(apply_path)
smoke = load(smoke_path)

if apply.get("inputs", {}).get("manifest_sha256") != manifest_hash:
    raise SystemExit("apply evidence is not bound to current manifests")
if smoke.get("inputs", {}).get("shadow_apply_sha256") != sha(apply_path):
    raise SystemExit("smoke evidence is not bound to shadow apply")
if smoke.get("workloads") != apply.get("workloads") or smoke.get("images") != apply.get("images"):
    raise SystemExit("smoke workload/image evidence differs from shadow apply")
apply_runtime = apply.get("runtime_snapshot", {}).get("map", {})
smoke_pre_runtime = smoke.get("runtime_snapshots", {}).get("pre", {}).get("map", {})
smoke_runtime = smoke.get("runtime_snapshots", {}).get("post", {}).get("map", {})

service_specs = {"redis": ("router-ai-atius-redis", "router-ai-atius-redis"), "router": ("router-ai-atius", "router-ai-atius"), "postgres": ("router-ai-atius-postgres", "router-ai-atius-postgres")}

def snapshot_endpoint_slices(snapshot, source):
    raw = snapshot.get("endpoint_slices")
    if not isinstance(raw, dict) or set(raw) != set(service_specs):
        raise SystemExit(f"{source} EndpointSlice service set is not exact")
    normalized = {}
    for key, (service_name, _) in service_specs.items():
        rows = raw.get(key)
        if not isinstance(rows, list):
            raise SystemExit(f"{source} EndpointSlice list is invalid for {service_name}")
        normalized[key] = []
        for row in rows:
            if not isinstance(row, dict) or row.get("service_name", service_name) != service_name:
                raise SystemExit(f"{source} EndpointSlice serviceName mismatch for {service_name}")
            item = dict(row)
            item["service_name"] = service_name
            normalized[key].append(item)
        normalized[key].sort(key=lambda item: item.get("name", ""))
    return normalized

apply_endpoint_slices = snapshot_endpoint_slices(apply_runtime, "apply")
smoke_pre_endpoint_slices = snapshot_endpoint_slices(smoke_pre_runtime, "smoke pre")
smoke_post_endpoint_slices = snapshot_endpoint_slices(smoke_runtime, "smoke post")
if apply_endpoint_slices != smoke_pre_endpoint_slices or apply_endpoint_slices != smoke_post_endpoint_slices:
    raise SystemExit("EndpointSlices differ between shadow apply and smoke snapshots")

specs = {
    "postgres": ("router-ai-atius-postgres", "StatefulSet", "router-ai-atius-postgres"),
    "redis": ("router-ai-atius-redis", "Deployment", "router-ai-atius-redis"),
    "router": ("router-ai-atius", "Deployment", "router-ai-atius"),
}
workloads = {}
expected_top = {(kind, name) for _, (_, kind, name) in specs.items()}
top = {(item.get("kind"), item.get("metadata", {}).get("name")) for item in controllers if item.get("kind") != "ReplicaSet"}
replica_sets = [item for item in controllers if item.get("kind") == "ReplicaSet"]
if top != expected_top or len(replica_sets) != 2 or len(controllers) != 5:
    raise SystemExit("namespace controller set is not exact")
if len(pods) != 3 or {p.get("metadata", {}).get("labels", {}).get("app.kubernetes.io/name") for p in pods} != {item[0] for item in specs.values()}:
    raise SystemExit("namespace pod set is not exact")
for key, (app, kind, controller_name) in specs.items():
    matches = [p for p in pods if p.get("metadata", {}).get("labels", {}).get("app.kubernetes.io/name") == app]
    if len(matches) != 1:
        raise SystemExit(f"{app} must have exactly one pod")
    pod = matches[0]
    containers = pod.get("spec", {}).get("containers", [])
    statuses = pod.get("status", {}).get("containerStatuses", [])
    if len(containers) != 1 or len(statuses) != 1 or pod.get("spec", {}).get("nodeName") != "atius-srv-1" or not statuses[0].get("ready"):
        raise SystemExit(f"{app} pod shape/readiness/placement mismatch")
    controller_matches = [c for c in controllers if c.get("kind") == kind and c.get("metadata", {}).get("name") == controller_name]
    if len(controller_matches) != 1:
        raise SystemExit(f"{app} controller identity is not unique")
    controller = controller_matches[0]
    owners = [o for o in pod.get("metadata", {}).get("ownerReferences", []) if o.get("controller") is True]
    if len(owners) != 1:
        raise SystemExit(f"{app} pod owner is not unique")
    owner = owners[0]
    if kind == "Deployment":
        rs_matches = [item for item in replica_sets if item.get("metadata", {}).get("name") == owner.get("name") and item.get("metadata", {}).get("uid") == owner.get("uid")]
        if owner.get("kind") != "ReplicaSet" or len(rs_matches) != 1:
            raise SystemExit(f"{app} pod is not owned by the exact ReplicaSet UID")
        root = [item for item in rs_matches[0].get("metadata", {}).get("ownerReferences", []) if item.get("controller") is True]
        if len(root) != 1 or root[0].get("kind") != "Deployment" or root[0].get("name") != controller_name or root[0].get("uid") != controller.get("metadata", {}).get("uid"):
            raise SystemExit(f"{app} ReplicaSet is not owned by the exact Deployment UID")
    elif owner.get("kind") != "StatefulSet" or owner.get("name") != controller_name or owner.get("uid") != controller.get("metadata", {}).get("uid"):
        raise SystemExit(f"{app} pod is not owned by the exact StatefulSet UID")
    image_ref = containers[0].get("image", "")
    image_id = statuses[0].get("imageID", "")
    digest_match = re.search(r"@?(sha256:[0-9a-f]{64})$", image_ref)
    if not digest_match or not image_id.endswith(digest_match.group(1)):
        raise SystemExit(f"{app} image reference/imageID mismatch")
    resources = containers[0].get("resources", {})
    if resources.get("requests", {}).get("cpu") != "500m" or resources.get("limits", {}).get("cpu") != "500m":
        raise SystemExit(f"{app} CPU contract mismatch")
    actual = {
        "app": app,
        "controller": {"kind": kind, "name": controller_name, "uid": controller.get("metadata", {}).get("uid")},
        "pod_owner": {"name": owners[0].get("name"), "uid": owners[0].get("uid")},
        "pod": {"name": pod.get("metadata", {}).get("name"), "uid": pod.get("metadata", {}).get("uid"), "ip": pod.get("status", {}).get("podIP")},
        "container": {"name": containers[0].get("name"), "image_ref": image_ref, "image_id": image_id,
                      "resources": {"requests_cpu": "500m", "limits_cpu": "500m"}},
    }
    if actual != apply.get("workloads", {}).get(key):
        raise SystemExit(f"{app} live identity differs from shadow apply")
    snapshot = smoke_runtime.get("workloads", {}).get(key, {})
    if (snapshot.get("controller", {}).get("kind"), snapshot.get("controller", {}).get("name"), snapshot.get("controller", {}).get("uid")) != (kind, controller_name, controller.get("metadata", {}).get("uid")) or \
       (snapshot.get("pod_owner", {}).get("name"), snapshot.get("pod_owner", {}).get("uid")) != (owner.get("name"), owner.get("uid")) or \
       (snapshot.get("pod", {}).get("name"), snapshot.get("pod", {}).get("uid"), snapshot.get("pod", {}).get("ip"), snapshot.get("pod", {}).get("node")) != (actual["pod"]["name"], actual["pod"]["uid"], actual["pod"]["ip"], "atius-srv-1") or \
       (snapshot.get("container", {}).get("name"), snapshot.get("container", {}).get("image_ref"), snapshot.get("container", {}).get("image_id"), snapshot.get("container", {}).get("restart_count")) != (actual["container"]["name"], image_ref, image_id, statuses[0].get("restartCount")):
        raise SystemExit(f"{app} live identity differs semantically from smoke snapshot")
    image_evidence = apply.get("images", {}).get(key, {})
    if image_evidence.get("reference") != image_ref or image_evidence.get("runtime_image_id") != image_id or image_evidence.get("digest") != digest_match.group(1) or image_evidence.get("exact") is not True:
        raise SystemExit(f"{app} image evidence differs from live identity")
    workloads[key] = actual

service_map = {}
live_endpoint_slices = {}
if {s.get("metadata", {}).get("name") for s in services} != {v[0] for v in service_specs.values()}:
    raise SystemExit("namespace Service set is not exact")
for key, (name, app) in service_specs.items():
    svc = next(s for s in services if s.get("metadata", {}).get("name") == name)
    spec = svc.get("spec", {})
    if spec.get("type") != "ClusterIP" or spec.get("selector") != {"app.kubernetes.io/name": app} or not spec.get("clusterIP") or spec.get("clusterIP") == "None":
        raise SystemExit(f"{name} Service selector/type mismatch")
    if any("nodePort" in p for p in spec.get("ports", [])):
        raise SystemExit(f"{name} Service exposes nodePort")
    service_snapshot = smoke_runtime.get("services", {}).get(key, {})
    if (service_snapshot.get("name"), service_snapshot.get("uid"), service_snapshot.get("type"), service_snapshot.get("cluster_ip"), service_snapshot.get("selector")) != (name, svc.get("metadata", {}).get("uid"), "ClusterIP", spec.get("clusterIP"), spec.get("selector")):
        raise SystemExit(f"{name} Service differs semantically from smoke snapshot")
    endpoint_rows = [e for sl in slices if sl.get("metadata", {}).get("labels", {}).get("kubernetes.io/service-name") == name for e in sl.get("endpoints", [])]
    service_slices = [sl for sl in slices if sl.get("metadata", {}).get("labels", {}).get("kubernetes.io/service-name") == name]
    pod = workloads[key]["pod"]
    if len(service_slices) != 1 or len(endpoint_rows) != 1:
        raise SystemExit(f"{name} must have exactly one EndpointSlice and endpoint")
    endpoint = endpoint_rows[0]
    target = endpoint.get("targetRef", {})
    if endpoint.get("conditions", {}).get("ready") is not True or endpoint.get("nodeName") != "atius-srv-1" or endpoint.get("addresses") != [pod["ip"]] or target.get("uid") != pod["uid"] or target.get("name") != pod["name"]:
        raise SystemExit(f"{name} EndpointSlice does not bind the approved pod UID/IP")
    if key in ("router", "redis") and apply.get("services", {}).get(key, {}).get("cluster_ip") != spec.get("clusterIP"):
        raise SystemExit(f"{name} Service differs from shadow apply")
    live_endpoint_slices[key] = []
    for endpoint_slice in service_slices:
        metadata = endpoint_slice.get("metadata", {})
        if not metadata.get("name") or not metadata.get("uid") or not metadata.get("resourceVersion"):
            raise SystemExit(f"{name} EndpointSlice identity is incomplete")
        endpoint_view = []
        for row in endpoint_slice.get("endpoints", []):
            ref = row.get("targetRef", {})
            endpoint_view.append({
                "addresses": sorted(row.get("addresses", [])),
                "conditions": row.get("conditions", {}),
                "node": row.get("nodeName"),
                "target_ref": {"kind": ref.get("kind"), "name": ref.get("name"), "namespace": ref.get("namespace"), "uid": ref.get("uid")},
            })
        endpoint_view.sort(key=lambda row: (row["target_ref"].get("uid") or "", row["addresses"]))
        live_endpoint_slices[key].append({
            "name": metadata.get("name"), "uid": metadata.get("uid"),
            "resource_version": metadata.get("resourceVersion"), "service_name": name,
            "address_type": endpoint_slice.get("addressType"),
            "ports": sorted(endpoint_slice.get("ports", []), key=lambda row: (row.get("name") or "", row.get("port") or 0)),
            "endpoints": endpoint_view,
        })
    live_endpoint_slices[key].sort(key=lambda item: item.get("name", ""))
    service_map[key] = {"name": name, "uid": svc.get("metadata", {}).get("uid"), "type": "ClusterIP",
                        "cluster_ip": spec.get("clusterIP"), "selector": spec.get("selector"),
                        "endpoint": {"pod_uid": pod["uid"], "pod_ip": pod["ip"], "ready": True}}

if len(slices) != 3 or {item.get("metadata", {}).get("labels", {}).get("kubernetes.io/service-name") for item in slices} != {item[0] for item in service_specs.values()}:
    raise SystemExit("namespace EndpointSlice set is not exact")
if live_endpoint_slices != apply_endpoint_slices:
    raise SystemExit("live EndpointSlices differ from shadow apply/smoke snapshots")

expected_claims = {"router-ai-atius-postgres-data", "router-ai-atius-data"}
if {p.get("metadata", {}).get("name") for p in pvcs} != expected_claims:
    raise SystemExit("namespace PVC set is not exact")
storage = []
for pvc in pvcs:
    name = pvc.get("metadata", {}).get("name")
    uid = pvc.get("metadata", {}).get("uid")
    pv_name = pvc.get("spec", {}).get("volumeName")
    matches = [p for p in pvs if p.get("metadata", {}).get("name") == pv_name]
    if pvc.get("status", {}).get("phase") != "Bound" or len(matches) != 1:
        raise SystemExit(f"{name} PVC/PV binding mismatch")
    pv = matches[0]
    claim = pv.get("spec", {}).get("claimRef", {})
    if claim.get("uid") != uid or claim.get("name") != name or claim.get("namespace") != "router-ai-atius" or pv.get("spec", {}).get("persistentVolumeReclaimPolicy") != "Retain":
        raise SystemExit(f"{name} claim UID/Retain mismatch")
    key = "router" if name == "router-ai-atius-data" else "postgres"
    snapshot_storage = smoke.get("storage", {}).get(key, {})
    snapshot_pvc = snapshot_storage.get("pvc", {})
    snapshot_pv = snapshot_storage.get("pv", {})
    if snapshot_storage.get("binding_verified") is not True or \
       (snapshot_pvc.get("name"), snapshot_pvc.get("uid"), snapshot_pvc.get("phase"), snapshot_pvc.get("volume_name")) != (name, uid, "Bound", pv_name) or \
       (snapshot_pv.get("name"), snapshot_pv.get("uid"), snapshot_pv.get("phase"), snapshot_pv.get("reclaim_policy")) != (pv_name, pv.get("metadata", {}).get("uid"), "Bound", "Retain") or \
       snapshot_pv.get("claim_ref") != {"namespace": "router-ai-atius", "name": name, "uid": uid}:
        raise SystemExit(f"{name} live PVC/PV differs semantically from smoke snapshot")
    storage.append({"pvc": name, "pvc_uid": uid, "pv": pv_name, "reclaim_policy": "Retain", "claim_uid_matched": True})
storage.sort(key=lambda row: row["pvc"])
apply_storage = sorted(apply.get("pvs", []), key=lambda row: row.get("pvc", ""))
if storage != apply_storage:
    raise SystemExit("live PVC/PV map differs from shadow apply")
namespace_pvs = [item for item in pvs if item.get("spec", {}).get("claimRef", {}).get("namespace") == "router-ai-atius"]
if len(namespace_pvs) != 2 or {item.get("metadata", {}).get("name") for item in namespace_pvs} != {item["pv"] for item in storage}:
    raise SystemExit("namespace PV binding set is not exact")

result = {"schema_version": 1, "manifest_sha256": manifest_hash, "shadow_apply_sha256": sha(apply_path),
          "smoke_sha256": sha(smoke_path), "workloads": workloads, "services": service_map,
          "endpoint_slices": live_endpoint_slices, "storage": storage}
pathlib.Path(output_path).write_text(json.dumps(result, sort_keys=True, separators=(",", ":")), encoding="utf-8")
PY
}

live_cluster_checks() {
  local cluster_uid="$1" deadline node_json free_bytes total_bytes free_percent
  local workloads controllers services endpoints pvcs pvs claims ingress identity_tmp
  deadline=$((SECONDS + stable_seconds))
  while :; do
    node_json="$(sudo -n k3s kubectl get node atius-srv-1 -o json 2>/dev/null)" || { fail_gate live-stability 'cannot read atius-srv-1'; break; }
    free_bytes="$(df -B1 --output=avail / | tail -1 | tr -d ' ')"
    total_bytes="$(df -B1 --output=size / | tail -1 | tr -d ' ')"
    free_percent=$((free_bytes * 100 / total_bytes))
    if ! jq -e '
      any(.status.conditions[]; .type == "DiskPressure" and .status == "False") and
      all(.spec.taints // []; .key != "node.kubernetes.io/disk-pressure")
    ' <<< "$node_json" >/dev/null || [ "$free_bytes" -lt 34359738368 ] || [ "$free_percent" -lt 25 ]; then
      fail_gate live-stability 'DiskPressure, taint, <32GiB free, or <25% free observed'; break
    fi
    if [ "$SECONDS" -ge "$deadline" ]; then pass_gate live-stability 'current node stayed green for at least five minutes'; break; fi
    sleep 10
  done

  if [ "$(sudo -n k3s kubectl get nodes -l atius.com.br/router-ai-atius-node=true -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null)" = atius-srv-1 ]; then
    pass_gate live-label 'dedicated label is exclusive to atius-srv-1'
  else fail_gate live-label 'dedicated label is absent or not exclusive'; fi

  if [ "$(sudo -n k3s kubectl -n router-ai-atius get secret router-ai-atius-secrets -o json 2>/dev/null | jq -r '.data | keys | sort | join(",")')" = 'POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET' ]; then
    pass_gate live-secret-keys 'live Secret has exactly the required key names'
  else fail_gate live-secret-keys 'live Secret key-name contract failed'; fi

  workloads="$(sudo -n k3s kubectl -n router-ai-atius get pods -o json 2>/dev/null || printf '{"items":[]}')"
  if jq -e '
    (.items | length) == 3 and all(.items[];
      .spec.nodeName == "atius-srv-1" and .status.phase == "Running" and
      (.status.containerStatuses | length) == 1 and all(.status.containerStatuses[]; .ready == true) and
      all(.spec.containers[]; .resources.requests.cpu == "500m" and .resources.limits.cpu == "500m" and
        ((.ports // []) | all(.[]; has("hostPort") | not))))
  ' <<< "$workloads" >/dev/null; then pass_gate live-placement 'all three Ready pods are srv1-only at 500m with no hostPort'
  else fail_gate live-placement 'pod count, readiness, placement, CPU, or hostPort gate failed'; fi

  if jq -e '
    all(.items[]; all(.spec.containers[]; .image | test("@sha256:[0-9a-f]{64}$")))
  ' <<< "$workloads" >/dev/null && jq -e '
    all(.items[]; (.status.containerStatuses | length) == (.spec.containers | length) and
      all(.status.containerStatuses[]; (.imageID | test("sha256:[0-9a-f]{64}$"))))
  ' <<< "$workloads" >/dev/null; then
    pass_gate live-images 'all live workload references and runtime image IDs are immutable digests'
  else fail_gate live-images 'a live workload uses a mutable image reference or invalid runtime image ID'; fi

  services="$(sudo -n k3s kubectl -n router-ai-atius get services -o json 2>/dev/null || printf '{"items":[]}')"
  endpoints="$(sudo -n k3s kubectl -n router-ai-atius get endpointslices.discovery.k8s.io -o json 2>/dev/null || printf '{"items":[]}')"
  ingress="$(sudo -n k3s kubectl -n router-ai-atius get ingress -o json 2>/dev/null || printf '{"items":[]}')"
  if jq -e '
    (.items | length) >= 2 and all(.items[];
      .spec.type == "ClusterIP" and (.spec.clusterIP | length) > 0 and .spec.clusterIP != "None" and
      all(.spec.ports[]; has("nodePort") | not))
  ' <<< "$services" >/dev/null && jq -e '
    (.items | length) >= 2 and all(.items[];
      (.metadata.labels["kubernetes.io/service-name"] | length) > 0 and
      any(.endpoints[]; .conditions.ready == true and .nodeName == "atius-srv-1" and (.addresses | length) > 0))
  ' <<< "$endpoints" >/dev/null && jq -e '.items | length == 0' <<< "$ingress" >/dev/null; then
    pass_gate live-clusterip 'namespace exposes only ClusterIP Services with Ready srv1 endpoints and no Ingress'
  else fail_gate live-clusterip 'NodePort, non-ClusterIP Service, missing Ready EndpointSlice, or Ingress is present'; fi

  pvcs="$(sudo -n k3s kubectl -n router-ai-atius get pvc -o json 2>/dev/null || printf '{"items":[]}')"
  pvs="$(sudo -n k3s kubectl get pv -o json 2>/dev/null || printf '{"items":[]}')"
  claims="$(jq -c '[.items[] | {uid:.metadata.uid,pv:.spec.volumeName,phase:.status.phase}]' <<< "$pvcs")"
  if jq -e --argjson claims "$claims" '
    ($claims | length) >= 2 and all($claims[]; .phase == "Bound" and (.uid | length) > 0 and (.pv | length) > 0) and
    all($claims[] as $claim; any(.items[];
      .metadata.name == $claim.pv and .spec.persistentVolumeReclaimPolicy == "Retain" and
      .spec.claimRef.uid == $claim.uid))
  ' <<< "$pvs" >/dev/null; then pass_gate live-pv-retain 'every namespace PVC is bound to its claim UID with PV Retain'
  else fail_gate live-pv-retain 'one or more PVC/PV claim UID or Retain checks failed'; fi

  controllers="$(sudo -n k3s kubectl -n router-ai-atius get deployment,statefulset,replicaset -o json 2>/dev/null || printf '{"items":[]}')"
  identity_tmp="$(mktemp -d "$evidence_dir/.identity-inputs.XXXXXX")"
  printf '%s' "$workloads" > "$identity_tmp/pods.json"
  printf '%s' "$controllers" > "$identity_tmp/controllers.json"
  printf '%s' "$services" > "$identity_tmp/services.json"
  printf '%s' "$endpoints" > "$identity_tmp/endpoints.json"
  printf '%s' "$pvcs" > "$identity_tmp/pvcs.json"
  printf '%s' "$pvs" > "$identity_tmp/pvs.json"
  if validate_and_write_identity_map "$identity_tmp/pods.json" "$identity_tmp/controllers.json" \
    "$identity_tmp/services.json" "$identity_tmp/endpoints.json" "$identity_tmp/pvcs.json" "$identity_tmp/pvs.json" \
    "$evidence_dir/shadow-apply.json" "$evidence_dir/smoke.json" "$(manifest_hash)" "$identity_file"; then
    chmod 600 "$identity_file"
    pass_gate live-identity-map 'PostgreSQL/Redis/router pod, image, Service, EndpointSlice, PVC and PV identities match apply/smoke/manifests'
  else
    fail_gate live-identity-map 'live identity map differs from apply/smoke/manifests'
  fi
  rm -rf "$identity_tmp"

  if bin/clianything status --backend k3s >/dev/null 2>&1; then pass_gate live-clianything 'CLIAnything k3s status is operational'
  else fail_gate live-clianything 'CLIAnything k3s status failed'; fi

  [ -n "$cluster_uid" ] || fail_gate live-cluster-uid 'current cluster UID is empty'
}

write_decision() {
  local cluster_uid="$1" now="$2" current_manifest="$3" decision failed_json gates_json tmp commit image_digest identity_json identity_sha
  gates_json="$(jq -s '.' "$gates_file")"
  failed_json="$(jq '[.[] | select(.status == "fail") | .name]' <<< "$gates_json")"
  decision=no-go
  [ "$(jq 'length' <<< "$failed_json")" -eq 0 ] && decision=go
  commit="$(git rev-parse HEAD)"
  image_digest="$(jq -r '.images.router.digest // .image.digest // empty' "$evidence_dir/shadow-apply.json" 2>/dev/null || true)"
  if regular_json "$identity_file"; then
    identity_json="$(cat "$identity_file")"
    identity_sha="$(sha256sum "$identity_file" | awk '{print $1}')"
  else
    identity_json=null
    identity_sha=""
  fi
  tmp="$(mktemp "${output}.tmp.XXXXXX")"
  chmod 600 "$tmp"
  jq -n --arg decision "$decision" --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson generated_at_epoch "$now" --arg run_id "$run_id" --arg cluster_uid "$cluster_uid" --arg commit "$commit" \
    --arg manifest_sha256 "$current_manifest" --arg image_digest "$image_digest" \
    --argjson rollback "$(if regular_json "$rollback_file"; then cat "$rollback_file"; else printf null; fi)" \
    --argjson live_identity_map "$identity_json" --arg live_identity_sha256 "$identity_sha" \
    --argjson gates "$gates_json" --argjson failed_gates "$failed_json" \
    --arg cleanup_sha256 "$(sha256sum "$evidence_dir/cleanup.json" 2>/dev/null | awk '{print $1}')" \
    --arg bootstrap_sha256 "$(sha256sum "$evidence_dir/bootstrap.json" 2>/dev/null | awk '{print $1}')" \
    --arg backup_sha256 "$(sha256sum "$evidence_dir/backup.json" 2>/dev/null | awk '{print $1}')" \
    --arg restore_sha256 "$(sha256sum "$evidence_dir/restore.json" 2>/dev/null | awk '{print $1}')" \
    --arg apply_sha256 "$(sha256sum "$evidence_dir/shadow-apply.json" 2>/dev/null | awk '{print $1}')" \
    --arg smoke_sha256 "$(sha256sum "$evidence_dir/smoke.json" 2>/dev/null | awk '{print $1}')" \
    --arg rollback_sha256 "$(sha256sum "$rollback_file" 2>/dev/null | awk '{print $1}')" \
    '{schema_version:2,decision:$decision,phase30_authorized:($decision == "go"),run_id:$run_id,generated_at:$generated_at,
      generated_at_epoch:$generated_at_epoch,cluster_uid:$cluster_uid,commit:$commit,manifest_sha256:$manifest_sha256,
      image_digest:$image_digest,evidence_checksums:{cleanup:$cleanup_sha256,bootstrap:$bootstrap_sha256,backup:$backup_sha256,
      restore:$restore_sha256,shadow_apply:$apply_sha256,smoke:$smoke_sha256,rollback:$rollback_sha256},gates:$gates,
      failed_gates:$failed_gates,rollback:$rollback,live_identity_map:$live_identity_map,
      live_identity_map_sha256:$live_identity_sha256,
      public_edge_changed:false,mutations:{apache:false,podman:false,manifests:false,k3s:false}}' > "$tmp"
  mv "$tmp" "$output"
  echo "Phase 29 decision: $decision ($output)"
}

verify_new_decision() {
  local file="$1" now key path expected actual
  regular_json "$file" || die 'decision artifact is not regular valid JSON'
  now="$(date +%s)"
  jq -e --argjson now "$now" --arg run_id "$run_id" '
    .schema_version == 2 and .run_id == $run_id and (.decision == "go" or .decision == "no-go") and
    (.generated_at_epoch | type == "number") and .generated_at_epoch <= $now and
    ($now - .generated_at_epoch) <= 3600 and (.cluster_uid | length) > 0 and
    (.commit | test("^[0-9a-f]{40}$")) and (.manifest_sha256 | test("^[0-9a-f]{64}$")) and
    (.gates | type == "array" and length > 0) and (.failed_gates | type == "array") and
    .rollback.run_id == $run_id and .rollback.read_only == true and
    .public_edge_changed == false and
    .mutations.apache == false and .mutations.podman == false and .mutations.manifests == false and
    ((.decision == "go" and .phase30_authorized == true and .rollback.status == "go" and
      (.live_identity_map | type == "object") and (.live_identity_map_sha256 | test("^[0-9a-f]{64}$")) and
      (.failed_gates | length) == 0 and all(.gates[]; .status == "pass")) or
     (.decision == "no-go" and .phase30_authorized == false and (.failed_gates | length) > 0))
  ' "$file" >/dev/null || die 'decision artifact schema or fail-closed invariant is invalid'
  for key in cleanup bootstrap backup restore shadow_apply smoke; do
    case "$key" in
      shadow_apply) path="$evidence_dir/shadow-apply.json" ;;
      *) path="$evidence_dir/${key}.json" ;;
    esac
    expected="$(jq -r ".evidence_checksums.$key" "$file")"
    actual="$(sha256sum "$path" 2>/dev/null | awk '{print $1}')"
    [ "$expected" = "$actual" ] || die "decision checksum mismatch for $key"
  done
  [ "$(jq -r '.evidence_checksums.rollback' "$file")" = "$(sha256sum "$rollback_file" | awk '{print $1}')" ] ||
    die 'decision checksum mismatch for fresh rollback'
  if [ "$(jq -r '.decision' "$file")" = go ]; then
    [ "$(jq -r '.live_identity_map_sha256' "$file")" = "$(sha256sum "$identity_file" | awk '{print $1}')" ] ||
      die 'decision live identity checksum mismatch'
  fi
  if jq -r '.. | strings' "$file" | rg -i '(authorization:|bearer |postgres(ql)?://|password=|session_secret=|redis_password=)' >/dev/null; then
    die 'decision artifact contains a secret-bearing value'
  fi
  echo "decision verification: PASS ($(jq -r '.decision' "$file"))"
}

self_test() {
  local tmp now cluster manifest sha
  tmp="$(mktemp -d)"; chmod 700 "$tmp"
  trap 'rm -rf "$tmp"' RETURN
  evidence_dir="$tmp"; now="$(date +%s)"; cluster=fixture-cluster; run_id=phase29-selftest
  rollback_file="$tmp/rollback-$run_id.json"; identity_file="$tmp/live-identity.json"
  printf '%s\n' '{"schema_version":1,"fixture":true}' > "$identity_file"
  manifest="$(printf fixture-manifests | sha256sum | awk '{print $1}')"
  jq -n --arg c "$cluster" --argjson n "$now" '{status:"go",cluster_uid:$c,generated_at_epoch:$n,reclaimed_bytes:21474836480,free_percent:25,stable_seconds:300,cpu_max:"80000 100000"}' > "$tmp/cleanup.json"
  jq -n --arg c "$cluster" --argjson n "$now" --arg m "$manifest" '{status:"go",cluster_uid:$c,generated_at_epoch:$n,exclusive_node:"atius-srv-1",secret_keys:"POSTGRES_PASSWORD,REDIS_PASSWORD,SESSION_SECRET",manifest_sha256:$m,digest_match:true,manifest_digest:("sha256:"+("a"*64)),image_ref:("example@sha256:"+("a"*64)),cpu_max:"80000 100000"}' > "$tmp/bootstrap.json"
  jq -n --argjson n "$now" '{status:"go",generated_at_epoch:$n,source:{kind:"host-postgresql",host:"127.0.0.1",port:8745,server_version_num:"170010",systemd_unit:"postgresql@17-main",backend_unit_matched:true},pgbouncer_crosscheck:{matched:true},pg_dump_version:"17.10",cpu_max:"80000 100000",cpu:{aggregate_millicores:800,postgres_quota_restored:true},dump:{size_bytes:1024,sha256:("b"*64),structurally_valid:true},database_inventory:{format:"phase29-database-inventory-v2",sha256:("c"*64),schema_ddl_sha256:("d"*64)}}' > "$tmp/backup.json"
  jq -n --arg c "$cluster" --argjson n "$now" '{status:"go",restore_passed:true,cluster_uid:$c,generated_at_epoch:$n,backup:{sha256:("b"*64),source:"host-postgresql-17"},target:{node:"atius-srv-1",database:"DBRouterAiAtius",clean_before_restore:true,server_version_num:"170010"},database_inventory:{format:"phase29-database-inventory-v2",source_sha256:("c"*64),target_sha256:("c"*64),source_schema_ddl_sha256:("d"*64),target_schema_ddl_sha256:("d"*64),source_backup_target_matched:true,matched:true},pvs:[{reclaim_policy:"Retain",claim_uid_matched:true}],cpu_max:"80000 100000",runtime_stage:{redis_applied:false,router_applied:false}}' > "$tmp/restore.json"
  sha="$(sha256sum "$tmp/restore.json" | awk '{print $1}')"
  jq -n --arg c "$cluster" --argjson n "$now" --arg m "$manifest" --arg r "$sha" --arg b "$(sha256sum "$tmp/bootstrap.json" | awk '{print $1}')" '
    def workload($app;$kind;$controller;$pod;$puid;$ip;$container;$image):
      {app:$app,controller:{kind:$kind,name:$controller,uid:("controller-"+$app)},
       pod_owner:(if $kind == "StatefulSet" then {name:$controller,uid:("controller-"+$app)} elif $app == "router-ai-atius-redis" then {name:"redis-rs",uid:"redis-rs-uid"} else {name:"router-rs",uid:"router-rs-uid"} end),
       pod:{name:$pod,uid:$puid,ip:$ip},container:{name:$container,image_ref:$image,image_id:$image,
       resources:{requests_cpu:"500m",limits_cpu:"500m"}}};
    ("example@sha256:"+("a"*64)) as $image | ("sha256:"+("a"*64)) as $digest |
    {status:"go",cluster_uid:$c,generated_at_epoch:$n,cpu_max:"80000 100000",inputs:{manifest_sha256:$m,restore_sha256:$r,bootstrap_sha256:$b},
     image:{exact:true,digest:$digest},images:{router:{reference:$image,digest:$digest,runtime_image_id:$image,exact:true},redis:{reference:$image,digest:$digest,runtime_image_id:$image,exact:true},postgres:{reference:$image,digest:$digest,runtime_image_id:$image,exact:true}},
     workloads:{router:workload("router-ai-atius";"Deployment";"router-ai-atius";"router-pod";"router-uid";"10.42.0.12";"router-ai-atius";$image),redis:workload("router-ai-atius-redis";"Deployment";"router-ai-atius-redis";"redis-pod";"redis-uid";"10.42.0.11";"redis";$image),postgres:workload("router-ai-atius-postgres";"StatefulSet";"router-ai-atius-postgres";"postgres-pod";"postgres-uid";"10.42.0.10";"postgres";$image)},
     placement:{node:"atius-srv-1",postgres_ready:true,redis_ready_before_router:true,router_ready:true,cpu_per_pod:"500m"},
     pvs:[{pvc:"router-ai-atius-data",pvc_uid:"router-claim",pv:"router-pv",reclaim_policy:"Retain",claim_uid_matched:true},{pvc:"router-ai-atius-postgres-data",pvc_uid:"postgres-claim",pv:"postgres-pv",reclaim_policy:"Retain",claim_uid_matched:true}],
     services:{redis:{type:"ClusterIP",cluster_ip:"10.43.0.11",endpoints_ready:true},router:{type:"ClusterIP",cluster_ip:"10.43.0.12",endpoints_ready:true}},mutations:{apache:false,podman:false}}' > "$tmp/shadow-apply.json"
  jq -n --arg c "$cluster" --argjson n "$now" --arg a "$(sha256sum "$tmp/shadow-apply.json" | awk '{print $1}')" --arg r "$sha" '{status:"go",cluster_uid:$c,generated_at_epoch:$n,cpu_max:"80000 100000",transport:{type:"ClusterIP",cluster_ip:"10.43.0.12",endpoints_ready:true},inputs:{shadow_apply_sha256:$a,restore_sha256:$r,image_digest:("sha256:"+("a"*64))},checks:{health_status:200,unauthorized_models_status:401,authenticated_models_status:200,root_data_only:true,internal_fields_absent:true,expected_models_present:true,embedding_model:"embedding-gte-v1",embedding_dimension:768}}' > "$tmp/smoke.json"
  jq -n --argjson n "$now" --arg run "$run_id" '{schema_version:2,status:"go",run_id:$run,generated_at_epoch:$n,podman:{unit:{present:false,required:false,active:false},pod:{exists:true,running:true},containers_ready:true,limits_valid:true,health_ok:true,clianything_backend:"podman",clianything_ok:true},apache:{config_path:"/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf",selected_vhost:"/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf",vhost_selection_ok:true,syntax_ok:true,routes_to_podman:true,k3s_target_present:false},read_only:true,mutations:{apache:false,podman:false,k3s:false}}' > "$rollback_file"

  gates_file="$tmp/gates-go.jsonl"; : > "$gates_file"
  validate_evidence "$cluster" "$now" "$manifest"
  [ "$(jq -s '[.[] | select(.status == "fail")] | length' "$gates_file")" -eq 0 ] || die 'complete fixture did not produce all green gates'
  output="$tmp/decision-go.json"
  write_decision "$cluster" "$now" "$manifest" >/dev/null
  verify_new_decision "$output" >/dev/null
  [ "$(jq -r '.decision' "$output")" = go ] || die 'complete fixture did not emit GO'

  jq -n --arg image "example@sha256:$(printf 'a%.0s' {1..64})" '
    {items:[
      {metadata:{name:"postgres-pod",uid:"postgres-uid",labels:{"app.kubernetes.io/name":"router-ai-atius-postgres"},ownerReferences:[{controller:true,kind:"StatefulSet",name:"router-ai-atius-postgres",uid:"controller-router-ai-atius-postgres"}]},spec:{nodeName:"atius-srv-1",containers:[{name:"postgres",image:$image,resources:{requests:{cpu:"500m"},limits:{cpu:"500m"}}}]},status:{podIP:"10.42.0.10",containerStatuses:[{ready:true,restartCount:0,imageID:$image}]}},
      {metadata:{name:"redis-pod",uid:"redis-uid",labels:{"app.kubernetes.io/name":"router-ai-atius-redis"},ownerReferences:[{controller:true,kind:"ReplicaSet",name:"redis-rs",uid:"redis-rs-uid"}]},spec:{nodeName:"atius-srv-1",containers:[{name:"redis",image:$image,resources:{requests:{cpu:"500m"},limits:{cpu:"500m"}}}]},status:{podIP:"10.42.0.11",containerStatuses:[{ready:true,restartCount:0,imageID:$image}]}},
      {metadata:{name:"router-pod",uid:"router-uid",labels:{"app.kubernetes.io/name":"router-ai-atius"},ownerReferences:[{controller:true,kind:"ReplicaSet",name:"router-rs",uid:"router-rs-uid"}]},spec:{nodeName:"atius-srv-1",containers:[{name:"router-ai-atius",image:$image,resources:{requests:{cpu:"500m"},limits:{cpu:"500m"}}}]},status:{podIP:"10.42.0.12",containerStatuses:[{ready:true,restartCount:0,imageID:$image}]}}
    ]}' > "$tmp/pods.json"
  jq -n '{items:[{kind:"StatefulSet",metadata:{name:"router-ai-atius-postgres",uid:"controller-router-ai-atius-postgres"}},{kind:"Deployment",metadata:{name:"router-ai-atius-redis",uid:"controller-router-ai-atius-redis"}},{kind:"Deployment",metadata:{name:"router-ai-atius",uid:"controller-router-ai-atius"}},{kind:"ReplicaSet",metadata:{name:"redis-rs",uid:"redis-rs-uid",ownerReferences:[{controller:true,kind:"Deployment",name:"router-ai-atius-redis",uid:"controller-router-ai-atius-redis"}]}},{kind:"ReplicaSet",metadata:{name:"router-rs",uid:"router-rs-uid",ownerReferences:[{controller:true,kind:"Deployment",name:"router-ai-atius",uid:"controller-router-ai-atius"}]}}]}' > "$tmp/controllers.json"
  jq -n '{items:[{metadata:{name:"router-ai-atius-redis",uid:"redis-service"},spec:{type:"ClusterIP",clusterIP:"10.43.0.11",selector:{"app.kubernetes.io/name":"router-ai-atius-redis"},ports:[{port:6379}]}},{metadata:{name:"router-ai-atius",uid:"router-service"},spec:{type:"ClusterIP",clusterIP:"10.43.0.12",selector:{"app.kubernetes.io/name":"router-ai-atius"},ports:[{port:3000}]}},{metadata:{name:"router-ai-atius-postgres",uid:"postgres-service"},spec:{type:"ClusterIP",clusterIP:"10.43.0.10",selector:{"app.kubernetes.io/name":"router-ai-atius-postgres"},ports:[{port:5432}]}}]}' > "$tmp/services.json"
  jq -n '{items:[
    {metadata:{name:"redis-slice",uid:"redis-slice-uid",resourceVersion:"101",labels:{"kubernetes.io/service-name":"router-ai-atius-redis"}},addressType:"IPv4",ports:[{name:"redis",port:6379,protocol:"TCP"}],endpoints:[{conditions:{ready:true},nodeName:"atius-srv-1",addresses:["10.42.0.11"],targetRef:{kind:"Pod",namespace:"router-ai-atius",name:"redis-pod",uid:"redis-uid"}}]},
    {metadata:{name:"router-slice",uid:"router-slice-uid",resourceVersion:"102",labels:{"kubernetes.io/service-name":"router-ai-atius"}},addressType:"IPv4",ports:[{name:"http",port:3000,protocol:"TCP"}],endpoints:[{conditions:{ready:true},nodeName:"atius-srv-1",addresses:["10.42.0.12"],targetRef:{kind:"Pod",namespace:"router-ai-atius",name:"router-pod",uid:"router-uid"}}]},
    {metadata:{name:"postgres-slice",uid:"postgres-slice-uid",resourceVersion:"103",labels:{"kubernetes.io/service-name":"router-ai-atius-postgres"}},addressType:"IPv4",ports:[{name:"postgres",port:5432,protocol:"TCP"}],endpoints:[{conditions:{ready:true},nodeName:"atius-srv-1",addresses:["10.42.0.10"],targetRef:{kind:"Pod",namespace:"router-ai-atius",name:"postgres-pod",uid:"postgres-uid"}}]}
  ]}' > "$tmp/endpoints.json"
  jq -n '{items:[{metadata:{name:"router-ai-atius-postgres-data",uid:"postgres-claim"},spec:{volumeName:"postgres-pv"},status:{phase:"Bound"}},{metadata:{name:"router-ai-atius-data",uid:"router-claim"},spec:{volumeName:"router-pv"},status:{phase:"Bound"}}]}' > "$tmp/pvcs.json"
  jq -n '{items:[{metadata:{name:"postgres-pv",uid:"postgres-pv-uid"},spec:{persistentVolumeReclaimPolicy:"Retain",claimRef:{name:"router-ai-atius-postgres-data",namespace:"router-ai-atius",uid:"postgres-claim"}},status:{phase:"Bound"}},{metadata:{name:"router-pv",uid:"router-pv-uid"},spec:{persistentVolumeReclaimPolicy:"Retain",claimRef:{name:"router-ai-atius-data",namespace:"router-ai-atius",uid:"router-claim"}},status:{phase:"Bound"}}]}' > "$tmp/pvs.json"
  jq -n --slurpfile endpoints "$tmp/endpoints.json" --argjson apply "$(cat "$tmp/shadow-apply.json")" '
    def slice_view($service):
      [$endpoints[0].items[] | select(.metadata.labels["kubernetes.io/service-name"] == $service) |
       {name:.metadata.name,uid:.metadata.uid,resource_version:.metadata.resourceVersion,address_type:.addressType,
        ports:(.ports | sort_by(.name,.port)),endpoints:[.endpoints[] | {addresses:(.addresses|sort),conditions:.conditions,node:.nodeName,
        target_ref:{kind:.targetRef.kind,name:.targetRef.name,namespace:.targetRef.namespace,uid:.targetRef.uid}}]}];
    {workloads:($apply.workloads | with_entries(.value.pod += {node:"atius-srv-1"} | .value.container += {restart_count:0})),
     services:{router:{name:"router-ai-atius",uid:"router-service",type:"ClusterIP",cluster_ip:"10.43.0.12",selector:{"app.kubernetes.io/name":"router-ai-atius"}},redis:{name:"router-ai-atius-redis",uid:"redis-service",type:"ClusterIP",cluster_ip:"10.43.0.11",selector:{"app.kubernetes.io/name":"router-ai-atius-redis"}},postgres:{name:"router-ai-atius-postgres",uid:"postgres-service",type:"ClusterIP",cluster_ip:"10.43.0.10",selector:{"app.kubernetes.io/name":"router-ai-atius-postgres"}}},
     endpoint_slices:{router:slice_view("router-ai-atius"),redis:slice_view("router-ai-atius-redis"),postgres:slice_view("router-ai-atius-postgres")}}
  ' > "$tmp/runtime-map.json"
  jq --argjson runtime "$(cat "$tmp/runtime-map.json")" '.runtime_snapshot={map:$runtime}' "$tmp/shadow-apply.json" > "$tmp/shadow-apply.rich"
  mv "$tmp/shadow-apply.rich" "$tmp/shadow-apply.json"
  jq --arg apply_sha "$(sha256sum "$tmp/shadow-apply.json" | awk '{print $1}')" '.inputs.shadow_apply_sha256=$apply_sha' "$tmp/smoke.json" > "$tmp/smoke.bound"
  mv "$tmp/smoke.bound" "$tmp/smoke.json"
  jq --argjson apply "$(cat "$tmp/shadow-apply.json")" --argjson runtime "$(cat "$tmp/runtime-map.json")" '
    .images=$apply.images | .workloads=$apply.workloads |
    .storage={router:{pvc:{name:"router-ai-atius-data",uid:"router-claim",phase:"Bound",volume_name:"router-pv"},pv:{name:"router-pv",uid:"router-pv-uid",phase:"Bound",reclaim_policy:"Retain",claim_ref:{namespace:"router-ai-atius",name:"router-ai-atius-data",uid:"router-claim"}},binding_verified:true},postgres:{pvc:{name:"router-ai-atius-postgres-data",uid:"postgres-claim",phase:"Bound",volume_name:"postgres-pv"},pv:{name:"postgres-pv",uid:"postgres-pv-uid",phase:"Bound",reclaim_policy:"Retain",claim_ref:{namespace:"router-ai-atius",name:"router-ai-atius-postgres-data",uid:"postgres-claim"}},binding_verified:true}} |
    .runtime_snapshots={equal:true,pre:{map:$runtime},post:{map:$runtime}}
  ' "$tmp/smoke.json" > "$tmp/smoke.rich"; mv "$tmp/smoke.rich" "$tmp/smoke.json"
  validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/exact-map.json"
  jq '(.items[] | select(.metadata.name == "router-slice") | .metadata) += {name:"router-slice-replacement",uid:"router-slice-replacement-uid",resourceVersion:"999"}' "$tmp/endpoints.json" > "$tmp/replaced-endpoints.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/replaced-endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'replacement EndpointSlice with the same endpoint was accepted'
  fi
  jq '.items += [{kind:"Deployment",metadata:{name:"extra",uid:"extra-controller"}}]' "$tmp/controllers.json" > "$tmp/extra-controllers.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/extra-controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'extra controller was accepted by exact identity map'
  fi
  jq '.items[3].metadata.ownerReferences[0].uid="wrong-deployment"' "$tmp/controllers.json" > "$tmp/wrong-rs.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/wrong-rs.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'broken Pod to ReplicaSet to Deployment UID chain was accepted'
  fi
  jq '.items += [.items[0] | .metadata.name="extra-slice" | .metadata.uid="extra-slice-uid"]' "$tmp/endpoints.json" > "$tmp/extra-endpoints.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/extra-endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'extra EndpointSlice was accepted by exact identity map'
  fi
  jq '.workloads.router.pod.uid="smoke-divergence"' "$tmp/smoke.json" > "$tmp/smoke-workload-divergence.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke-workload-divergence.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'smoke workload divergence was accepted'
  fi
  jq '.images.router.digest=("sha256:"+("f"*64))' "$tmp/smoke.json" > "$tmp/smoke-image-divergence.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke-image-divergence.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'smoke image divergence was accepted'
  fi
  jq '.storage.router.pvc.uid="smoke-divergence"' "$tmp/smoke.json" > "$tmp/smoke-storage-divergence.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke-storage-divergence.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'smoke storage divergence was accepted'
  fi
  jq '.items += [{metadata:{name:"extra-pvc",uid:"extra-claim"},spec:{volumeName:"extra-pv"},status:{phase:"Bound"}}]' "$tmp/pvcs.json" > "$tmp/extra-pvcs.json"
  if validate_and_write_identity_map "$tmp/pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/extra-pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'extra PVC was accepted by exact identity map'
  fi
  jq '.items[2].metadata.uid="replacement-router"' "$tmp/pods.json" > "$tmp/replaced-pods.json"
  if validate_and_write_identity_map "$tmp/replaced-pods.json" "$tmp/controllers.json" "$tmp/services.json" "$tmp/endpoints.json" \
    "$tmp/pvcs.json" "$tmp/pvs.json" "$tmp/shadow-apply.json" "$tmp/smoke.json" "$manifest" "$tmp/bad-map.json" 2>/dev/null; then
    die 'replacement router pod was accepted by exact identity map'
  fi

  jq '.status="no-go"' "$tmp/restore.json" > "$tmp/restore.bad"; mv "$tmp/restore.bad" "$tmp/restore.json"
  gates_file="$tmp/gates-no-go.jsonl"; : > "$gates_file"
  validate_evidence "$cluster" "$now" "$manifest"
  jq -s -e 'any(.[]; .name == "restore" and .status == "fail") and any(.[]; .name == "shadow-apply" and .status == "fail")' "$gates_file" >/dev/null ||
    die 'single broken chain input did not fail closed'
  output="$tmp/decision-no-go.json"
  write_decision "$cluster" "$now" "$manifest" >/dev/null
  verify_new_decision "$output" >/dev/null
  jq -e '.decision == "no-go" and .phase30_authorized == false and (.failed_gates | length) > 0' "$output" >/dev/null ||
    die 'broken fixture did not emit a valid NO-GO'

  jq '.status="go"' "$tmp/restore.json" > "$tmp/restore.good"; mv "$tmp/restore.good" "$tmp/restore.json"
  printf '\n' >> "$tmp/bootstrap.json"
  gates_file="$tmp/gates-tamper.jsonl"; : > "$gates_file"
  validate_evidence "$cluster" "$now" "$manifest"
  jq -s -e 'any(.[]; .name == "shadow-apply" and .status == "fail")' "$gates_file" >/dev/null ||
    die 'tampered checksum chain was accepted'

  jq --argjson stale "$((now - fresh_seconds - 1))" '.generated_at_epoch=$stale' "$tmp/smoke.json" > "$tmp/smoke.stale"
  mv "$tmp/smoke.stale" "$tmp/smoke.json"
  gates_file="$tmp/gates-stale.jsonl"; : > "$gates_file"
  validate_evidence "$cluster" "$now" "$manifest"
  jq -s -e 'any(.[]; .name == "smoke" and .status == "fail")' "$gates_file" >/dev/null ||
    die 'stale smoke evidence was accepted'

  rm "$rollback_file"
  gates_file="$tmp/gates-missing.jsonl"; : > "$gates_file"
  validate_evidence "$cluster" "$now" "$manifest"
  jq -s -e 'any(.[]; (.name | startswith("artifact:rollback-")) and .status == "fail")' "$gates_file" >/dev/null ||
    die 'missing rollback evidence was accepted'
  echo 'go/no-go self-test: PASS'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --live) mode=live ;;
    --evidence-dir) evidence_dir="${2:?}"; shift ;;
    --output) output="${2:?}"; shift ;;
    --verify-existing) die '--verify-existing was removed: rerun --live to recompute artifacts, live identities, and a fresh rollback' ;;
    --self-test) self_test; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
  shift
done

if [ "$mode" = dry-run ]; then echo 'go/no-go dry-run: default decision is no-go; no runtime mutation performed'; exit 0; fi

[ "${PHASE29_EXECUTE:-0}" = 1 ] || die '--live requires PHASE29_EXECUTE=1'
[[ "$fresh_seconds" =~ ^[0-9]+$ && "$cleanup_max_age" =~ ^[0-9]+$ && "$stable_seconds" =~ ^[0-9]+$ ]] || die 'freshness/stability values must be integers'
[ "$stable_seconds" -ge 300 ] || die 'live stability window must be at least 300 seconds'
[ -n "$evidence_dir" ] || die '--evidence-dir is required'
case "$evidence_dir" in "$evidence_root"/run-[A-Za-z0-9._-]*) ;; *) die 'evidence directory is outside the canonical Phase 29 run root' ;; esac
if [ ! -d "$evidence_dir" ] || [ -L "$evidence_dir" ]; then
  die 'evidence directory must be a real directory'
fi
evidence_dir="$(realpath -e "$evidence_dir")"
[ "$(stat -c '%U:%a' "$evidence_dir")" = "$(id -un):700" ] || die 'evidence directory owner/mode must be caller:700'
output="${output:-$evidence_dir/decision.json}"
case "$output" in "$evidence_dir"/*) ;; *) die 'output must stay inside the evidence directory' ;; esac
if [ -e "$output" ] || [ -L "$output" ]; then die 'decision artifact already exists'; fi
for command in jq sha256sum sudo k3s python3; do command -v "$command" >/dev/null || die "required command missing: $command"; done

run_id="phase29-$(date -u +%Y%m%dT%H%M%SZ)-$(printf '%s:%s:%s' "$$" "$RANDOM" "$(date +%s%N)" | sha256sum | cut -c1-16)"
rollback_file="$evidence_dir/rollback-$run_id.json"
identity_file="$evidence_dir/live-identity-$run_id.json"
if ! PHASE29_EXECUTE=1 scripts/k3s-router-rollback-check.sh --live --evidence-dir "$evidence_dir" \
  --run-id "$run_id" --output "$rollback_file"; then
  : # A valid fresh no-go rollback artifact is consumed by the aggregate decision.
fi
cluster_uid="$(sudo -n k3s kubectl get namespace kube-system -o jsonpath='{.metadata.uid}')"
now="$(date +%s)"
current_manifest="$(manifest_hash)"
gates_file="$(mktemp "$evidence_dir/.decision-gates.XXXXXX")"
trap 'rm -f "${gates_file:-}"' EXIT
chmod 600 "$gates_file"
validate_evidence "$cluster_uid" "$now" "$current_manifest"
live_cluster_checks "$cluster_uid"
write_decision "$cluster_uid" "$now" "$current_manifest"
verify_new_decision "$output"
