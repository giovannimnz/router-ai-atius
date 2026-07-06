#!/usr/bin/env bash
set -euo pipefail

FORK_REPO="${FORK_REPO:-giovannimnz/router-ai-atius}"
UPSTREAM_REPO="${UPSTREAM_REPO:-QuantumNous/new-api}"
FORK_REMOTE="${FORK_REMOTE:-origin}"
UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
FORK_URL="${FORK_URL:-https://github.com/${FORK_REPO}.git}"
UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/${UPSTREAM_REPO}.git}"
UPSTREAM_PUSH_URL="${UPSTREAM_PUSH_URL:-DISABLED}"

die() {
  echo "fork-sync-guard: $*" >&2
  exit 1
}

ensure_actions_repo() {
  if [[ -n "${GITHUB_REPOSITORY:-}" && "$GITHUB_REPOSITORY" != "$FORK_REPO" ]]; then
    die "refusing to run in GitHub repository '$GITHUB_REPOSITORY'; expected '$FORK_REPO'"
  fi
}

ensure_local_fork_remote() {
  local origin_url
  origin_url="$(git remote get-url "$FORK_REMOTE" 2>/dev/null || true)"
  [[ -n "$origin_url" ]] || die "fork remote '$FORK_REMOTE' is missing"

  case "$origin_url" in
    *"$FORK_REPO"*|"$FORK_URL") ;;
    *) die "fork remote '$FORK_REMOTE' points to '$origin_url', not '$FORK_REPO'" ;;
  esac
}

configure_remotes() {
  ensure_actions_repo
  if git remote get-url "$UPSTREAM_REMOTE" >/dev/null 2>&1; then
    git remote set-url "$UPSTREAM_REMOTE" "$UPSTREAM_URL"
  else
    git remote add "$UPSTREAM_REMOTE" "$UPSTREAM_URL"
  fi
  git remote set-url --push "$UPSTREAM_REMOTE" "$UPSTREAM_PUSH_URL"
  git config "remote.${UPSTREAM_REMOTE}.tagOpt" --no-tags
  ensure_local_fork_remote
}

push_fork() {
  local remote="${1:-}"
  shift || true
  [[ "$remote" == "$FORK_REMOTE" ]] || die "refusing to push to '$remote'; only '$FORK_REMOTE' is allowed"
  ensure_actions_repo
  ensure_local_fork_remote
  git push "$remote" "$@"
}

workflow_run() {
  local workflow="${1:-}"
  shift || true
  [[ -n "$workflow" ]] || die "workflow name required"
  ensure_actions_repo
  gh workflow run "$workflow" --repo "$FORK_REPO" "$@"
}

case "${1:-}" in
  ensure-actions-repo)
    ensure_actions_repo
    ;;
  ensure-local-fork-remote)
    ensure_local_fork_remote
    ;;
  configure-remotes)
    configure_remotes
    ;;
  push)
    shift
    push_fork "$@"
    ;;
  workflow-run)
    shift
    workflow_run "$@"
    ;;
  *)
    cat >&2 <<'USAGE'
usage:
  scripts/fork-sync-guard.sh ensure-actions-repo
  scripts/fork-sync-guard.sh ensure-local-fork-remote
  scripts/fork-sync-guard.sh configure-remotes
  scripts/fork-sync-guard.sh push origin <refspec...>
  scripts/fork-sync-guard.sh workflow-run <workflow.yml> [gh workflow args...]
USAGE
    exit 2
    ;;
esac
