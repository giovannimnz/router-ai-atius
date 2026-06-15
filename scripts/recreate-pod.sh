#!/bin/bash
# Recreate router-ai-stack pod + 4 user containers from compose.
# Run on SRV-1 only.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "=== Tear down ==="
podman-compose -f podman-compose.yml down 2>&1 || true
podman pod rm -f router-ai-stack 2>&1 || true

echo "=== Bring up ==="
podman-compose -f podman-compose.yml up -d

echo "=== Verify ==="
sleep 5
podman pod inspect router-ai-stack --format 'State={{.State}} Containers={{len .Containers}}' 2>&1
podman ps --filter pod=router-ai-stack --format 'table {{.Names}} {{.Status}} {{.Image}}' 2>&1
