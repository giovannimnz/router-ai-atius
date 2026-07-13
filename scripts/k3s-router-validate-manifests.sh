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


def cpu_millicores(value):
    if isinstance(value, int):
        return value * 1000
    if not isinstance(value, str):
        raise ValueError("CPU value must be a string or integer")
    if value.endswith("m") and value[:-1].isdigit():
        return int(value[:-1])
    if value.isdigit():
        return int(value) * 1000
    raise ValueError(f"unsupported CPU value: {value}")

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
    containers = pod.get("containers", [])
    init_containers = pod.get("initContainers", [])
    if not containers:
        raise SystemExit(f"{name} must define at least one regular container")
    request_total = 0
    limit_total = 0
    for container in [*containers, *init_containers]:
        image = container.get("image", "")
        if not re.fullmatch(r"[^@]+@sha256:[0-9a-f]{64}", image):
            raise SystemExit(f"{name}/{container.get('name')} image must use an exact digest")
        if any("hostPort" in port for port in container.get("ports", [])):
            raise SystemExit(f"{name} contains hostPort")
        resources = container.get("resources", {})
        request = resources.get("requests", {}).get("cpu")
        limit = resources.get("limits", {}).get("cpu")
        try:
            request_cpu = cpu_millicores(request)
            limit_cpu = cpu_millicores(limit)
        except ValueError as exc:
            raise SystemExit(f"{name}/{container.get('name')} {exc}") from exc
        if request_cpu != limit_cpu:
            raise SystemExit(f"{name}/{container.get('name')} CPU requests must equal limits")
        request_total += request_cpu
        limit_total += limit_cpu
    if request_total != limit_total or request_total > 500:
        raise SystemExit(
            f"{name} total pod CPU must have requests=limits and stay at or below 500m"
        )

for name in ("router-ai-atius", "router-ai-atius-redis"):
    if workloads[name]["spec"].get("strategy") != {"type": "Recreate"}:
        raise SystemExit(f"{name} must use Recreate to avoid an unschedulable surge pod")

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
approved_postgres = "docker.io/library/postgres@sha256:5530681ea5d3e2ed4ce396f9b5cb443efbac6baf2a8a19c0c0635e40ae7eadce"
if postgres_container.get("image") != approved_postgres:
    raise SystemExit("PostgreSQL image must use the approved PostgreSQL 17 arm64 digest")
if postgres_container.get("command") != ["/bin/bash", "-ec"]:
    raise SystemExit("PostgreSQL must generate the canonical pt_BR.UTF-8 locale before startup")
postgres_argv = "\n".join(postgres_container.get("args", []))
if "localedef -i pt_BR" not in postgres_argv or "exec docker-entrypoint.sh postgres" not in postgres_argv:
    raise SystemExit("PostgreSQL locale bootstrap or entrypoint contract is missing")
postgres_env = {item.get("name"): item for item in postgres_container.get("env", [])}
if postgres_env.get("LANG", {}).get("value") != "pt_BR.UTF-8":
    raise SystemExit("PostgreSQL LANG must stay pt_BR.UTF-8")
if postgres_env.get("POSTGRES_INITDB_ARGS", {}).get("value") != "--locale-provider=libc --locale=pt_BR.UTF-8":
    raise SystemExit("PostgreSQL initdb locale must stay pt_BR.UTF-8/libc")

redis = workloads["router-ai-atius-redis"]
redis_container = redis["spec"]["template"]["spec"]["containers"][0]
approved_redis = "docker.io/library/redis@sha256:084f4bcb3fedf990ba43d26774f58ed4697a2c044156544ac4717934ad1d57c8"
if redis_container.get("image") != approved_redis:
    raise SystemExit("Redis image must use the approved Redis 7 arm64 digest")
redis_env = {item.get("name"): item for item in redis_container.get("env", [])}
if set(redis_env) != {"REDISCLI_AUTH"}:
    raise SystemExit("Redis must expose only REDISCLI_AUTH from the Secret")
if redis_env["REDISCLI_AUTH"].get("valueFrom", {}).get("secretKeyRef", {}) != {
    "name": "router-ai-atius-secrets",
    "key": "REDIS_PASSWORD",
}:
    raise SystemExit("Redis REDISCLI_AUTH must come from the canonical Secret key")
argv = [*redis_container.get("command", []), *redis_container.get("args", [])]
if any("--requirepass" in item or "redis-cli -a" in item for item in argv):
    raise SystemExit("Redis password must not be passed in server or probe argv")
if "chmod 0600 /run/redis/redis.conf" not in "\n".join(argv):
    raise SystemExit("Redis must generate a mode 0600 config before startup")
for probe_name in ("readinessProbe", "livenessProbe"):
    probe = redis_container.get(probe_name, {}).get("exec", {}).get("command", [])
    if not any("redis-cli ping" in item for item in probe) or any(" -a" in item for item in probe):
        raise SystemExit(f"Redis {probe_name} must use REDISCLI_AUTH without -a")

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
