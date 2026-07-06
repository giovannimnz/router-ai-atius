#!/usr/bin/env bash
set -euo pipefail

current="${1:-}"
upstream="${2:-}"

if [[ -z "$current" || -z "$upstream" ]]; then
  echo "usage: $0 <current-version> <upstream-version>" >&2
  exit 2
fi

current="${current#v}"
upstream="${upstream#v}"

if [[ "$current" == "$upstream."* ]]; then
  suffix="${current#"$upstream."}"
  if [[ "$suffix" =~ ^[0-9]+$ ]]; then
    echo "${upstream}.$((suffix + 1))"
    exit 0
  fi
fi

echo "${upstream}.1"
