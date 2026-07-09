#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

manifest_dir="k8s/router-ai-atius"
namespace="router-ai-atius"

if [ ! -d "$manifest_dir" ]; then
  echo "manifest directory missing: $manifest_dir" >&2
  exit 1
fi

sudo -n k3s kubectl apply --dry-run=server -f "$manifest_dir/namespace.yaml"

for file in "$manifest_dir"/*.yaml; do
  [ "$(basename "$file")" = "namespace.yaml" ] && continue
  # Server dry-run requires the target namespace to exist. While this phase is
  # still pre-apply, validate schema/field correctness in a harmless namespace
  # and keep the real namespace contract enforced by the namespace manifest.
  sed '/^[[:space:]]*namespace: router-ai-atius$/d' "$file" | \
    sudo -n k3s kubectl apply --dry-run=server -n default -f -
done

echo "dry-run only: no resources were created and no secret values were validated"
