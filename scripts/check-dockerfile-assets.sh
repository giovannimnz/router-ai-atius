#!/usr/bin/env bash
set -euo pipefail

dockerfile="${1:-Dockerfile}"

if [[ ! -f "$dockerfile" ]]; then
  echo "Dockerfile not found: $dockerfile" >&2
  exit 1
fi

if ! grep -Eq 'COPY[[:space:]].*web/package\.json[[:space:]]+web/bun\.lock' "$dockerfile"; then
  echo "Dockerfile must install frontends from the web workspace lockfile" >&2
  exit 1
fi

if ! grep -Eq '^FROM oven/bun:1\.3\.14 AS builder' "$dockerfile" ||
   ! grep -Eq '^FROM oven/bun:1\.3\.14 AS builder-classic' "$dockerfile"; then
  echo "Dockerfile must use the same Bun version as CI release builds: oven/bun:1.3.14" >&2
  exit 1
fi

required_paths=(
  web/package.json
  web/bun.lock
  web/default/package.json
  web/classic/package.json
)

for path in "${required_paths[@]}"; do
  if [[ ! -f "$path" ]]; then
    echo "Dockerfile dependency is missing: $path" >&2
    exit 1
  fi
done

echo "dockerfile asset guard passed"
