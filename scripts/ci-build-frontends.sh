#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
version="${1:-${VITE_REACT_APP_VERSION:-}}"

if [[ -z "$version" ]]; then
  version="$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

retry() {
  local attempts="${CI_FRONTEND_RETRIES:-3}"
  local delay="${CI_FRONTEND_RETRY_DELAY:-10}"
  local current=1

  until "$@"; do
    local status=$?
    if (( current >= attempts )); then
      echo "command failed after ${attempts} attempt(s): $*" >&2
      return "$status"
    fi

    echo "command failed with exit ${status}; retrying in ${delay}s (${current}/${attempts}): $*" >&2
    sleep "$delay"
    current=$((current + 1))
  done
}

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "expected build artifact missing: $path" >&2
    return 1
  fi
}

export VITE_REACT_APP_VERSION="$version"
export DISABLE_ESLINT_PLUGIN="${DISABLE_ESLINT_PLUGIN:-true}"

echo "building frontends for ${VITE_REACT_APP_VERSION}"
cd "$repo_root/web"

retry bun install --frozen-lockfile
rm -rf default/dist classic/dist "$repo_root/web/dist"

(
  cd default
  retry bun run build
)

(
  cd classic
  retry bun run build
)

cp -R "$repo_root/web/default/dist" "$repo_root/web/dist"

require_file "$repo_root/web/default/dist/index.html"
require_file "$repo_root/web/classic/dist/index.html"
require_file "$repo_root/web/dist/index.html"

echo "frontend build verified:"
echo "  web/default/dist/index.html"
echo "  web/classic/dist/index.html"
echo "  web/dist/index.html"
