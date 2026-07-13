#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
manifest_dir="${PHASE29_MANIFEST_DIR:-k8s/router-ai-atius}"
server=false

die() {
  echo "manifest validation failed: $*" >&2
  exit 1
}

case "${1:-}" in
  "") ;;
  --server) server=true ;;
  *) die "unknown argument: $1" ;;
esac

python3 - "$manifest_dir" <<'PY'
import pathlib
import re
import sys

import yaml

root = pathlib.Path(sys.argv[1])
docs = {}
for name in ("namespace", "configmap", "postgres", "redis", "router"):
    path = root / f"{name}.yaml"
    if not path.is_file():
        raise SystemExit(f"missing {path}")
    docs[name] = [doc for doc in yaml.safe_load_all(path.read_text()) if doc]

namespaces = [doc for doc in docs["namespace"] if doc.get("kind") == "Namespace"]
if len(namespaces) != 1 or namespaces[0].get("metadata", {}).get("name") != "router-ai-atius":
    raise SystemExit("namespace manifest must define only router-ai-atius")
for group in docs.values():
    for doc in group:
        if doc.get("kind") != "Namespace" and doc.get("metadata", {}).get("namespace") != "router-ai-atius":
            raise SystemExit(f"{doc.get('kind')}/{doc.get('metadata', {}).get('name')} has the wrong namespace")

workloads = {}
for group in docs.values():
    for doc in group:
        if doc.get("kind") in {"Deployment", "StatefulSet"}:
            workloads[doc["metadata"]["name"]] = doc

expected = {
    "router-ai-atius",
    "router-ai-atius-postgres",
    "router-ai-atius-redis",
}
if set(workloads) != expected:
    raise SystemExit(f"unexpected workloads: {sorted(workloads)}")

for name, doc in workloads.items():
    pod = doc["spec"]["template"]["spec"]
    terms = pod["affinity"]["nodeAffinity"]["requiredDuringSchedulingIgnoredDuringExecution"]["nodeSelectorTerms"]
    expressions = [item for term in terms for item in term.get("matchExpressions", [])]
    matches = [item for item in expressions if item.get("key") == "atius.com.br/router-ai-atius-node"]
    if matches != [{"key": "atius.com.br/router-ai-atius-node", "operator": "In", "values": ["true"]}]:
        raise SystemExit(f"{name} does not require the dedicated srv1 label")
    for forbidden in ("tolerations", "hostNetwork"):
        if forbidden in pod:
            raise SystemExit(f"{name} contains forbidden {forbidden}")
    if "preferredDuringSchedulingIgnoredDuringExecution" in pod.get("affinity", {}).get("nodeAffinity", {}):
        raise SystemExit(f"{name} contains preferred node affinity")
    for container in pod.get("containers", []):
        if any("hostPort" in port for port in container.get("ports", [])):
            raise SystemExit(f"{name} contains hostPort")
        resources = container.get("resources", {})
        if resources.get("requests", {}).get("cpu") != "500m" or resources.get("limits", {}).get("cpu") != "500m":
            raise SystemExit(f"{name} must request and limit exactly 500m CPU")

pvcs = {}
services = {}
for group in docs.values():
    for doc in group:
        kind = doc.get("kind")
        name = doc.get("metadata", {}).get("name")
        if kind == "PersistentVolumeClaim":
            pvcs[name] = doc
        elif kind == "Service":
            services[name] = doc

for name in ("router-ai-atius-data", "router-ai-atius-postgres-data"):
    if pvcs[name]["spec"].get("storageClassName") != "local-path":
        raise SystemExit(f"{name} is not explicitly local-path")

router_service = services["router-ai-atius"]
if router_service["spec"].get("type", "ClusterIP") != "ClusterIP":
    raise SystemExit("router Service must be ClusterIP")

router = workloads["router-ai-atius"]
container = router["spec"]["template"]["spec"]["containers"][0]
image = container.get("image", "")
if not re.fullmatch(r"ghcr\.io/giovannimnz/router-ai-atius@sha256:[0-9a-f]{64}", image):
    raise SystemExit("router image must use an exact 64-hex digest")
if container.get("imagePullPolicy") != "Never":
    raise SystemExit("router imagePullPolicy must be Never for the imported image")

postgres = workloads["router-ai-atius-postgres"]
postgres_container = postgres["spec"]["template"]["spec"]["containers"][0]
approved_postgres = "docker.io/library/postgres@sha256:b797483593b82cbea9a7ee41c88f324a90d10d9c2504d40e755d91c75456366d"
if postgres_container.get("image") != approved_postgres:
    raise SystemExit("PostgreSQL image must use the approved PostgreSQL 17 arm64 digest")

for group in docs.values():
    for doc in group:
        if doc.get("kind") == "Ingress":
            raise SystemExit("Ingress is forbidden for this shadow")
PY

if $server; then
  [ "${PHASE29_LIVE:-0}" = 1 ] || die '--server requires PHASE29_LIVE=1'
  sudo -n k3s kubectl get namespace router-ai-atius >/dev/null
  for file in "$manifest_dir"/*.yaml; do
    sudo -n k3s kubectl apply --dry-run=server -f "$file" >/dev/null
  done
fi

echo 'manifest validation: PASS'
