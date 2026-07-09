#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

required=(
  CURRENT_PUBLIC_URL
  K3S_ROUTER_BASE_URL
  K3S_BACKUP_DIR
  APACHE_VHOST_BACKUP_PATH
)

for key in "${required[@]}"; do
  if [ -z "${!key:-}" ]; then
    echo "Missing required env: ${key}" >&2
    exit 1
  fi
done

cat <<EOF
Pre-cutover checklist
- current public url: ${CURRENT_PUBLIC_URL}
- k3s target url: ${K3S_ROUTER_BASE_URL}
- backup directory: ${K3S_BACKUP_DIR}
- Apache vhost backup path: ${APACHE_VHOST_BACKUP_PATH}
EOF

echo
echo "1. Validate k3s target before any Apache edit:"
echo "   ATIUS_ROUTER_TOKEN=<token> K3S_ROUTER_BASE_URL=${K3S_ROUTER_BASE_URL} scripts/k3s-router-smoke.sh"
echo
echo "2. Validate Apache syntax before reload:"
echo "   apache2ctl configtest"
echo
echo "3. After manual Apache reload, run public smoke:"
echo "   curl -fsS ${CURRENT_PUBLIC_URL}/health"
echo "   curl -sS -o /dev/null -w '%{http_code}\n' ${CURRENT_PUBLIC_URL}/v1/models"
echo
echo "MANUAL CHECKPOINT: approve editing Apache only after the shadow smoke passes."
