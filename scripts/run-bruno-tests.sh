#!/usr/bin/env bash
#
# run-bruno-tests.sh — Execute Bruno API tests for router-ai-atius
#
# Usage:
#   ./scripts/run-bruno-tests.sh [collection] [args...]
#
set -euo pipefail

REPO_ROOT="$(dirname "$(dirname "$0")")"
BRUNO_CLI="${BRUNO_CLI:-/home/ubuntu/.nvm/versions/node/v24.13.1/bin/bru}"
COLLECTION_DIR="${REPO_ROOT}/integration/bruno-tests/atius-router-tests"

# Default to running entire collection
TARGET="${1:-.}"

# Build env vars from .env.local
ENV_ARGS=""
if [[ -f "${COLLECTION_DIR}/.env.local" ]]; then
    while IFS='=' read -r key value; do
        [[ "$key" =~ ^# ]] && continue
        [[ -z "$key" ]] && continue
        ENV_ARGS="${ENV_ARGS} --env-var ${key}=${value}"
    done < "${COLLECTION_DIR}/.env.local"
fi

echo "=== router-ai-atius Bruno Tests ==="
echo "Target: ${TARGET}"
echo "Bruno CLI: ${BRUNO_CLI}"
echo ""

cd "${COLLECTION_DIR}"

# Run tests with delay to avoid rate limiting
eval "${BRUNO_CLI} run --delay 500 ${ENV_ARGS} ${TARGET}"
