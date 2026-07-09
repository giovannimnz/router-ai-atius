#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [ -z "${K3S_ROUTER_BASE_URL:-}" ]; then
  echo "K3S_ROUTER_BASE_URL is required" >&2
  exit 1
fi

base_url="${K3S_ROUTER_BASE_URL%/}"

health_status="$(curl -sS -o /tmp/k3s-router-health.out -w '%{http_code}' "${base_url}/api/status" || true)"
if [ "$health_status" != "200" ]; then
  health_status="$(curl -sS -o /tmp/k3s-router-health.out -w '%{http_code}' "${base_url}/health" || true)"
fi
if [ "$health_status" != "200" ]; then
  echo "health check failed: ${health_status}" >&2
  exit 1
fi

unauth_status="$(curl -sS -o /tmp/k3s-router-models-unauth.out -w '%{http_code}' "${base_url}/v1/models" || true)"
if [ "$unauth_status" != "401" ]; then
  echo "expected unauthenticated /v1/models to return 401, got ${unauth_status}" >&2
  exit 1
fi

if [ -z "${ATIUS_ROUTER_TOKEN:-}" ]; then
  echo "ATIUS_ROUTER_TOKEN not set; authenticated smoke skipped"
  exit 0
fi

auth_body="$(mktemp)"
trap 'rm -f "$auth_body" /tmp/k3s-router-health.out /tmp/k3s-router-models-unauth.out' EXIT
auth_status="$(curl -sS -o "$auth_body" -w '%{http_code}' -H "Authorization: Bearer ${ATIUS_ROUTER_TOKEN}" "${base_url}/v1/models" || true)"
if [ "$auth_status" != "200" ]; then
  echo "authenticated /v1/models failed: ${auth_status}" >&2
  exit 1
fi

python3 - "$auth_body" <<'PY'
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text())
if set(payload.keys()) != {"data"}:
    raise SystemExit(f"unexpected top-level keys: {sorted(payload.keys())}")
text = json.dumps(payload)
for forbidden in ("pricing_source", "pricing_estimated", "pricing_version"):
    if forbidden in text:
        raise SystemExit(f"forbidden field present: {forbidden}")
PY

ATIUS_ROUTER_BASE_URL="$base_url" \
ATIUS_ROUTER_TOKEN="${ATIUS_ROUTER_TOKEN}" \
ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 \
ATIUS_ROUTER_EXPECTED_DIMENSION=768 \
python3 scripts/smoke-embeddings.py
