#!/usr/bin/env bash
set -euo pipefail

workflow="${1:-.github/workflows/sync.yml}"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
workflow_dir="$(dirname "$workflow")"
docker_publish_workflow="${workflow_dir}/docker-publish.yml"
next_version_script="$repo_root/scripts/next-fork-version.sh"

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

grep -Eq 'actions: write' "$workflow" || {
  echo "sync workflow must allow workflow_dispatch calls with actions: write" >&2
  exit 1
}

grep -Eq 'actions/checkout@v5' "$workflow" || {
  echo "sync workflow must use actions/checkout@v5 to avoid Node 20 deprecation warnings" >&2
  exit 1
}

for release_workflow in release.yml docker-build.yml electron-build.yml; do
  grep -Eq "gh workflow run ${release_workflow}" "$workflow" || {
    echo "sync workflow must dispatch ${release_workflow} after creating the version tag" >&2
    exit 1
  }
done

grep -Eq 'resolve_conflicts_with_side theirs' "$workflow" || {
  echo "sync workflow must use the per-path theirs resolver when strategy=theirs" >&2
  exit 1
}

grep -Eq 'git rm -f --ignore-unmatch' "$workflow" || {
  echo "sync workflow must remove files deleted by the selected merge side" >&2
  exit 1
}

grep -Eq 'git rm -r -f --ignore-unmatch -- "\$path"' "$workflow" || {
  echo "sync workflow must remove protected paths before restoring the fork baseline" >&2
  exit 1
}

grep -Eq 'clear_stale_index_lock' "$workflow" || {
  echo "sync workflow must clear stale index locks after failed merges" >&2
  exit 1
}

grep -Eq 'restore_upstream_paths' "$workflow" || {
  echo "sync workflow must restore upstream-owned paths after upstream merges" >&2
  exit 1
}

grep -Eq 'web/default' "$workflow" || {
  echo "sync workflow must keep web/default upstream-owned for strategy=theirs" >&2
  exit 1
}

grep -Eq 'Verify frontend release build' "$workflow" || {
  echo "sync workflow must verify frontend release build before pushing sync tags" >&2
  exit 1
}

grep -Eq 'oven-sh/setup-bun@v2' "$workflow" || {
  echo "sync workflow must install Bun before frontend verification" >&2
  exit 1
}

grep -Eq 'bun-version:[[:space:]]+1\.3\.14' "$workflow" || {
  echo "sync workflow must use Bun 1.3.14 for frontend verification" >&2
  exit 1
}

grep -Eq 'scripts/ci-build-frontends.sh "v\$NEW_TAG"' "$workflow" || {
  echo "sync workflow must run scripts/ci-build-frontends.sh for the new tag before push" >&2
  exit 1
}

grep -Eq 'scripts/next-fork-version\.sh "\$CURRENT" "\$UPSTREAM_VER"' "$workflow" || {
  echo "sync workflow must calculate fork suffix through scripts/next-fork-version.sh" >&2
  exit 1
}

grep -Eq 'Verify backend release build' "$workflow" || {
  echo "sync workflow must verify backend release build before pushing sync tags" >&2
  exit 1
}

grep -Eq 'scripts/ci-build-backend.sh "v\$NEW_TAG"' "$workflow" || {
  echo "sync workflow must run scripts/ci-build-backend.sh for the new tag before push" >&2
  exit 1
}

grep -Eq 'mapfile -d .*conflict_paths' "$workflow" || {
  echo "sync workflow must collect conflict paths before mutating the index" >&2
  exit 1
}

if grep -Eq 'git commit -m "Resolve conflicts:' "$workflow"; then
  echo "sync workflow must restore protected paths before completing the merge commit" >&2
  exit 1
fi

if [[ -f "$docker_publish_workflow" ]]; then
  grep -Fq 'workflows: ["Sync Upstream + Release"]' "$docker_publish_workflow" || {
    echo "docker-publish workflow_run must reference Sync Upstream + Release" >&2
    exit 1
  }
fi

if [[ ! -x "$next_version_script" ]]; then
  echo "next fork version script must exist and be executable: $next_version_script" >&2
  exit 1
fi

if [[ "$("$next_version_script" 1.0.0-rc.16.5 1.0.0-rc.16)" != "1.0.0-rc.16.6" ]]; then
  echo "next fork version script must increment rc suffixes without resetting to .1" >&2
  exit 1
fi

if [[ "$("$next_version_script" 1.0.0-rc.15.9 1.0.0-rc.16)" != "1.0.0-rc.16.1" ]]; then
  echo "next fork version script must reset suffix when upstream base changes" >&2
  exit 1
fi

echo "upstream sync workflow guard passed"
