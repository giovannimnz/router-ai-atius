#!/usr/bin/env bash
set -euo pipefail

raw="${1:-}"

if [[ -z "$raw" ]]; then
  raw="$(git describe --tags --always 2>/dev/null || echo dev)"
fi

version="${raw#v}"

sanitize_prerelease() {
  local value="$1"
  value="${value//[^0-9A-Za-z.-]/-}"
  value="${value#.}"
  value="${value%.}"
  while [[ "$value" == *..* ]]; do
    value="${value//../.}"
  done
  printf '%s' "${value:-dev}"
}

if [[ "$version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)\.(.+)$ ]]; then
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[3]}"
  rest="$(sanitize_prerelease "${BASH_REMATCH[4]}")"
  printf '%s\n' "${major}.${minor}.${patch}-patch.${rest}"
elif [[ "$version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)([-+].*)?$ ]]; then
  printf '%s\n' "$version"
elif [[ "$version" =~ ^([0-9]+)\.([0-9]+)$ ]]; then
  printf '%s\n' "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.0"
else
  rest="$(sanitize_prerelease "$version")"
  printf '%s\n' "0.0.0-${rest}"
fi
