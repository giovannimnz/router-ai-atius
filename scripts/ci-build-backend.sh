#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
version="${1:-$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null || echo dev)}"
go_bin="${GO_BIN:-/usr/local/go/bin/go}"

if [[ ! -x "$go_bin" ]]; then
  echo "go binary not found at $go_bin" >&2
  exit 1
fi

build_dir="$(mktemp -d)"
trap 'rm -rf "$build_dir"' EXIT

echo "building backend for ${version}"
cd "$repo_root"

"$go_bin" mod download
"$go_bin" build -buildvcs=false -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=${version}'" -o "$build_dir/new-api-${version}"

test -f "$build_dir/new-api-${version}"
echo "backend build verified:"
echo "  $build_dir/new-api-${version}"
