#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo=""
tag=""
max_attempts="${GH_WATCHDOG_MAX_ATTEMPTS:-3}"
poll_seconds="${GH_WATCHDOG_POLL_SECONDS:-30}"
deterministic_pattern="${GH_WATCHDOG_DETERMINISTIC_PATTERN:-Script not found|Username and password required|No such file or directory|Tag .* does not exist|permission denied|authentication required|frozen lockfile|Module not found|Cannot find module|undefined:|Build errors|failed to load config|script \"build\" exited|command not found|exit code 127}"

usage() {
  cat <<'EOF'
Usage:
  scripts/check-release-actions.sh <tag> [--repo owner/name] [--max-attempts 3] [--poll-seconds 30]

Checks every GitHub Actions run for a release tag and reruns failed jobs through
scripts/gh-actions-watchdog.sh until all runs succeed or a deterministic failure
requires a code/config fix.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      repo="$2"
      shift 2
      ;;
    --max-attempts)
      max_attempts="$2"
      shift 2
      ;;
    --poll-seconds)
      poll_seconds="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ -z "$tag" ]]; then
        tag="$1"
        shift
      else
        echo "unknown argument: $1" >&2
        usage >&2
        exit 2
      fi
      ;;
  esac
done

if [[ -z "$tag" ]]; then
  tag="$(git describe --tags --exact-match 2>/dev/null || true)"
fi

if [[ -z "$tag" ]]; then
  echo "release tag is required" >&2
  usage >&2
  exit 2
fi

if [[ -z "$repo" ]]; then
  repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
fi

print_release_summary() {
  gh release view "$tag" --repo "$repo" \
    --json tagName,url,assets \
    --jq '{tagName, url, asset_count: (.assets | length), assets: [.assets[].name]}'
}

has_deterministic_failure() {
  local run_id="$1"
  local log
  log="$(gh run view "$run_id" --repo "$repo" --log-failed 2>&1 || true)"
  grep -Eiq "$deterministic_pattern" <<<"$log"
}

attempt=1
while (( attempt <= max_attempts )); do
  echo "checking release actions for ${tag} (${repo}), pass ${attempt}/${max_attempts}"

  mapfile -t runs < <(
    gh run list \
      --repo "$repo" \
      --branch "$tag" \
      --limit 50 \
      --json databaseId,workflowName,status,conclusion,url \
      --jq '.[] | select(.status != "completed" or .conclusion != "success") | [.databaseId, .workflowName, .status, (.conclusion // ""), .url] | @tsv'
  )

  if [[ "${#runs[@]}" -eq 0 ]]; then
    print_release_summary
    echo "all release actions for ${tag} succeeded"
    exit 0
  fi

  printf '%s\n' "${runs[@]}"

  for row in "${runs[@]}"; do
    IFS=$'\t' read -r run_id workflow status conclusion url <<<"$row"
    echo "run ${run_id}: ${workflow} ${status}/${conclusion} ${url}"

    if [[ "$status" == "completed" && "$conclusion" == "failure" ]]; then
      if has_deterministic_failure "$run_id"; then
        echo "deterministic failure detected in run ${run_id}; apply code/config fix before retrying" >&2
        gh run view "$run_id" --repo "$repo" --log-failed || true
        exit 2
      fi
      "$script_dir/gh-actions-watchdog.sh" \
        --repo "$repo" \
        --run-id "$run_id" \
        --max-attempts 1 \
        --poll-seconds "$poll_seconds"
    else
      gh run watch "$run_id" --repo "$repo" --interval "$poll_seconds" --exit-status || true
    fi
  done

  attempt=$((attempt + 1))
  if (( attempt <= max_attempts )); then
    sleep "$poll_seconds"
  fi
done

echo "release actions for ${tag} did not converge after ${max_attempts} pass(es)" >&2
exit 1
