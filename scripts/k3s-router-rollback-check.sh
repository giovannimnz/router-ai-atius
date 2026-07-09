#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

systemctl --user status container-router-ai-atius.service --no-pager
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
bin/clianything status

echo
echo "If Apache was pointed to k3s, restore Apache vhost backup first, then run:"
echo "  apache2ctl configtest"
echo "  systemctl reload apache2"

if [ -n "${CURRENT_PUBLIC_URL:-}" ]; then
  echo "Optional public smoke after Apache restore:"
  echo "  curl -fsS ${CURRENT_PUBLIC_URL%/}/health"
fi
