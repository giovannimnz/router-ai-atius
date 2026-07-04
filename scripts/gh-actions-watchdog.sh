#!/usr/bin/env bash
set -euo pipefail

repo=""
run_id=""
tag=""
workflow=""
max_attempts="${GH_WATCHDOG_MAX_ATTEMPTS:-3}"
poll_seconds="${GH_WATCHDOG_POLL_SECONDS:-30}"

usage() {
  cat <<'EOF'
Usage:
  scripts/gh-actions-watchdog.sh --run-id <id> [--repo owner/name] [--max-attempts 3] [--poll-seconds 30]
  scripts/gh-actions-watchdog.sh --tag <tag> --workflow <workflow name|file> [--repo owner/name]

Watches a GitHub Actions run and reruns failed jobs until success or the attempt limit is reached.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      repo="$2"
      shift 2
      ;;
    --run-id)
      run_id="$2"
      shift 2
      ;;
    --tag)
      tag="$2"
      shift 2
      ;;
    --workflow)
      workflow="$2"
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
      if [[ -z "$run_id" && "$1" =~ ^[0-9]+$ ]]; then
        run_id="$1"
        shift
      else
        echo "unknown argument: $1" >&2
        usage >&2
        exit 2
      fi
      ;;
  esac
done

if [[ -z "$repo" ]]; then
  repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
fi

if [[ -z "$run_id" ]]; then
  if [[ -z "$tag" || -z "$workflow" ]]; then
    echo "--run-id or both --tag and --workflow are required" >&2
    usage >&2
    exit 2
  fi

  run_id="$(
    gh run list \
      --repo "$repo" \
      --workflow "$workflow" \
      --branch "$tag" \
      --limit 1 \
      --json databaseId \
      --jq '.[0].databaseId'
  )"
fi

if [[ -z "$run_id" || "$run_id" == "null" ]]; then
  echo "could not resolve a GitHub Actions run id" >&2
  exit 2
fi

print_summary() {
  gh run view "$run_id" --repo "$repo" \
    --json databaseId,url,status,conclusion,attempt,workflowName,headBranch,headSha,jobs \
    --jq '{
      id: .databaseId,
      url: .url,
      workflow: .workflowName,
      branch: .headBranch,
      sha: .headSha,
      status: .status,
      conclusion: .conclusion,
      attempt: .attempt,
      failed_jobs: [.jobs[]? | select(.conclusion == "failure") | {name, databaseId, conclusion, steps: [.steps[]? | select(.conclusion == "failure") | {name, number, conclusion}]}]
    }'
}

attempt=1
while (( attempt <= max_attempts )); do
  echo "watching GitHub Actions run ${run_id} (${repo}), attempt ${attempt}/${max_attempts}"

  if gh run watch "$run_id" --repo "$repo" --interval "$poll_seconds" --exit-status; then
    print_summary
    echo "GitHub Actions run succeeded"
    exit 0
  fi

  print_summary

  if (( attempt >= max_attempts )); then
    echo "GitHub Actions run failed after ${max_attempts} watched attempt(s)" >&2
    gh run view "$run_id" --repo "$repo" --log-failed || true
    exit 1
  fi

  echo "rerunning failed jobs for run ${run_id}"
  gh run rerun "$run_id" --repo "$repo" --failed
  attempt=$((attempt + 1))
  sleep "$poll_seconds"
done
