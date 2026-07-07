#!/usr/bin/env bash
set -euo pipefail

current="${1#v}"
upstream="${2#v}"

if [[ -z "$upstream" ]]; then
  echo "upstream version is required" >&2
  exit 1
fi

next_suffix=1
prefix="${upstream}."

if [[ -n "$current" && "$current" == "$prefix"* ]]; then
  tail_part="${current#$prefix}"
  if [[ "$tail_part" =~ ^[0-9]+$ ]]; then
    next_suffix=$((tail_part + 1))
  fi
fi

printf '%s.%s\n' "$upstream" "$next_suffix"
