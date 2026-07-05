#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${ATIUS_ROUTER_PUBLIC_BASE_URL:-https://router.atius.com.br}"
LOCAL_DOCS_URL="${ATIUS_ROUTER_DOCS_LOCAL_BASE_URL:-http://127.0.0.1:3003}"
CURL_BIN="${CURL_BIN:-$(command -v curl || true)}"

if [[ -z "$CURL_BIN" && -x /usr/bin/curl ]]; then
  CURL_BIN="/usr/bin/curl"
fi

if [[ -z "$CURL_BIN" ]]; then
  echo "smoke-docs-links: FAIL: curl not found" >&2
  exit 1
fi

fail() {
  echo "smoke-docs-links: FAIL: $*" >&2
  exit 1
}

curl_get() {
  "$CURL_BIN" -fsSk "$@"
}

check_status_docs_link() {
  local payload
  payload="$(curl_get "$BASE_URL/api/status")" || fail "could not fetch $BASE_URL/api/status"
  DOCS_STATUS_PAYLOAD="$payload" python3 - "$BASE_URL" <<'PY'
import json
import os
import sys

base_url = sys.argv[1]
payload = json.loads(os.environ["DOCS_STATUS_PAYLOAD"])
data = payload.get("data", payload)
docs_link = str(data.get("docs_link", ""))
if not docs_link:
    raise SystemExit(f"docs_link missing from {base_url}/api/status")
if "docs.newapi.pro" in docs_link:
    raise SystemExit(f"docs_link still points to upstream docs: {docs_link}")
if docs_link not in {"/en/docs", "/pt/docs", "/zh/docs", "/ja/docs"}:
    raise SystemExit(f"docs_link must be an internal docs route, got: {docs_link}")
PY
}

check_public_docs_route() {
  local path="$1"
  local url="$BASE_URL$path"
  local redirects
  redirects="$("$CURL_BIN" -sSkIL "$url" | tr -d '\r')" || fail "could not inspect redirects for $url"
  if printf '%s\n' "$redirects" | grep -Eiq '^location: https://docs\.newapi\.pro'; then
    fail "$url redirects to docs.newapi.pro"
  fi

  local result http_code effective_url
  result="$("$CURL_BIN" -sSkL -o /dev/null -w '%{http_code} %{url_effective}' "$url")" ||
    fail "could not fetch $url"
  http_code="${result%% *}"
  effective_url="${result#* }"
  if [[ "$http_code" != "200" ]]; then
    fail "$url returned $http_code after redirects (effective: $effective_url)"
  fi
  if [[ "$effective_url" == https://docs.newapi.pro* ]]; then
    fail "$url ended on upstream docs: $effective_url"
  fi
}

check_public_openapi_route() {
  local path="$1"
  local payload
  payload="$(curl_get "$BASE_URL$path")" || fail "could not fetch $BASE_URL$path"
  OPENAPI_PAYLOAD="$payload" python3 - "$path" <<'PY'
import json
import os
import sys

path = sys.argv[1]
try:
    payload = json.loads(os.environ["OPENAPI_PAYLOAD"])
except json.JSONDecodeError as exc:
    raise SystemExit(f"{path} did not return JSON: {exc}") from exc

if not str(payload.get("openapi", "")).startswith("3."):
    raise SystemExit(f"{path} did not return an OpenAPI 3.x document")
if not payload.get("paths"):
    raise SystemExit(f"{path} returned an OpenAPI document without paths")
PY
}

check_local_docs_route() {
  local path="$1"
  local url="$LOCAL_DOCS_URL$path"
  local code
  code="$("$CURL_BIN" -sSL -o /dev/null -w '%{http_code}' "$url")" ||
    fail "could not fetch local docs route $url"
  if [[ "$code" != "200" ]]; then
    fail "$url returned $code"
  fi
}

check_status_docs_link
check_public_docs_route "/en/docs"
check_public_docs_route "/pt/docs"
check_public_docs_route "/pt/docs/guide/project-introduction"
check_public_docs_route "/pt/docs/guide/technical-architecture"
check_public_openapi_route "/docs.json"
check_public_openapi_route "/docs/openapi.json"
check_local_docs_route "/en/docs/"
check_local_docs_route "/pt/docs/"
check_local_docs_route "/pt/docs/guide/project-introduction"
check_local_docs_route "/pt/docs/guide/technical-architecture"

echo "smoke-docs-links: OK"
