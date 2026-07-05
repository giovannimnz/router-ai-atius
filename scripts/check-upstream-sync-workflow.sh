#!/usr/bin/env bash
set -euo pipefail

workflow="${1:-.github/workflows/sync.yml}"

if [[ ! -f "$workflow" ]]; then
  echo "workflow not found: $workflow" >&2
  exit 1
fi

if grep -Eq 'git fetch upstream .*--tags|git fetch .*--tags .*upstream' "$workflow"; then
  echo "sync workflow must not fetch upstream tags into the fork namespace" >&2
  exit 1
fi

grep -Eq 'git fetch --no-tags --prune upstream' "$workflow" || {
  echo "sync workflow must fetch upstream branches with --no-tags" >&2
  exit 1
}

grep -Eq 'git ls-remote --tags --refs upstream' "$workflow" || {
  echo "sync workflow must detect upstream versions through ls-remote --tags --refs" >&2
  exit 1
}

grep -Eq 'resolve_conflicts_with_side theirs' "$workflow" || {
  echo "sync workflow must use the per-path theirs resolver when strategy=theirs" >&2
  exit 1
}

grep -Eq 'git rm -f --ignore-unmatch' "$workflow" || {
  echo "sync workflow must remove files deleted by the selected merge side" >&2
  exit 1
}

grep -Eq 'clear_stale_index_lock' "$workflow" || {
  echo "sync workflow must clear stale index locks after failed merges" >&2
  exit 1
}

grep -Eq 'mapfile -d .*conflict_paths' "$workflow" || {
  echo "sync workflow must collect conflict paths before mutating the index" >&2
  exit 1
}

echo "upstream sync workflow guard passed"
