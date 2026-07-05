#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
version="${1:-${VERSION:-}}"
go_bin="${GO_BIN:-$(command -v go || true)}"

if [[ -z "$go_bin" && -x /usr/local/go/bin/go ]]; then
  go_bin=/usr/local/go/bin/go
fi

if [[ -z "$go_bin" ]]; then
  echo "go binary not found in PATH or /usr/local/go/bin/go" >&2
  exit 1
fi

if [[ -z "$version" ]]; then
  version="$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

if [[ -z "${GOMAXPROCS:-}" ]]; then
  cores="$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2)"
  limit=$((cores / 2))
  if (( limit < 1 )); then
    limit=1
  fi
  export GOMAXPROCS="$limit"
fi

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

echo "building backend for ${version} (GOMAXPROCS=${GOMAXPROCS})"
cd "$repo_root"
"$go_bin" build -buildvcs=false -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=${version}" -o "$tmp/new-api" .

if [[ ! -x "$tmp/new-api" ]]; then
  echo "expected backend artifact missing: $tmp/new-api" >&2
  exit 1
fi

echo "backend build verified"
