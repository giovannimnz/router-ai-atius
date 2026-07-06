#!/usr/bin/env bash
set -euo pipefail

workflow="${1:-.github/workflows/sync.yml}"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/.." && pwd)"
workflow_dir="$(dirname "$workflow")"
docker_publish_workflow="${workflow_dir}/docker-publish.yml"
docker_build_workflow="${workflow_dir}/docker-build.yml"
docker_alpha_workflow="${workflow_dir}/docker-image-alpha.yml"
docker_nightly_workflow="${workflow_dir}/docker-image-nightly.yml"
gitee_sync_workflow="${workflow_dir}/sync-to-gitee.yml"
next_version_script="$repo_root/scripts/next-fork-version.sh"
guard_script="$repo_root/scripts/fork-sync-guard.sh"
sync_fork_script="$repo_root/scripts/sync-fork.sh"
auto_sync_deploy_script="$repo_root/scripts/auto-sync-deploy.sh"
pull_restart_script="$repo_root/scripts/pull-and-restart.sh"
deploy_ghcr_script="$repo_root/scripts/deploy-ghcr.sh"
model_main="$repo_root/model/main.go"

if [[ ! -f "$workflow" ]]; then
  echo "workflow not found: $workflow" >&2
  exit 1
fi

if [[ ! -x "$guard_script" ]]; then
  echo "fork sync guard must exist and be executable: $guard_script" >&2
  exit 1
fi

grep -Eq 'FORK_REPO:[[:space:]]+giovannimnz/router-ai-atius' "$workflow" || {
  echo "sync workflow must pin the writable fork repository" >&2
  exit 1
}

grep -Eq 'scripts/fork-sync-guard\.sh ensure-actions-repo' "$workflow" || {
  echo "sync workflow must validate it is running in the fork repository" >&2
  exit 1
}

grep -Eq 'scripts/fork-sync-guard\.sh configure-remotes' "$workflow" || {
  echo "sync workflow must configure upstream through the fork sync guard" >&2
  exit 1
}

grep -Eq 'UPSTREAM_PUSH_URL="\$\{UPSTREAM_PUSH_URL:-DISABLED\}"' "$guard_script" || {
  echo "fork sync guard must default upstream push URL to DISABLED" >&2
  exit 1
}

grep -Eq 'git remote set-url --push "\$UPSTREAM_REMOTE" "\$UPSTREAM_PUSH_URL"' "$guard_script" || {
  echo "fork sync guard must disable the upstream push URL" >&2
  exit 1
}

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
  grep -Eq "scripts/fork-sync-guard\\.sh workflow-run ${release_workflow}" "$workflow" || {
    echo "sync workflow must dispatch ${release_workflow} against the fork repository after creating the version tag" >&2
    exit 1
  }
done

grep -Eq 'scripts/fork-sync-guard\.sh push origin main' "$workflow" || {
  echo "sync workflow must push main through the fork sync guard" >&2
  exit 1
}

grep -Eq 'scripts/fork-sync-guard\.sh push origin "v\$NEW_TAG"' "$workflow" || {
  echo "sync workflow must push tags through the fork sync guard" >&2
  exit 1
}

naked_workflow_runs="$(find "$repo_root/.github/workflows" "$repo_root/scripts" -type f \( -name '*.yml' -o -name '*.yaml' -o -name '*.sh' \) -print0 | xargs -0 grep -n 'gh workflow run' 2>/dev/null | grep -v 'fork-sync-guard.sh' | grep -v 'check-upstream-sync-workflow.sh' || true)"
if [[ -n "$naked_workflow_runs" ]]; then
  echo "workflow dispatches must go through scripts/fork-sync-guard.sh workflow-run:" >&2
  echo "$naked_workflow_runs" >&2
  exit 1
fi

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

if [[ -f "$gitee_sync_workflow" ]]; then
  grep -Fq "GITEE_OWNER: 'giovannimnz'" "$gitee_sync_workflow" || {
    echo "sync-to-gitee workflow must target the fork owner by default" >&2
    exit 1
  }

  grep -Fq "GITEE_REPO: 'router-ai-atius'" "$gitee_sync_workflow" || {
    echo "sync-to-gitee workflow must target the fork repo by default" >&2
    exit 1
  }

  grep -Fq "vars.ENABLE_GITEE_SYNC == 'true'" "$gitee_sync_workflow" || {
    echo "sync-to-gitee workflow must be disabled unless explicitly enabled" >&2
    exit 1
  }
fi

for image_workflow in "$docker_build_workflow" "$docker_alpha_workflow" "$docker_nightly_workflow"; do
  [[ -f "$image_workflow" ]] || continue

  grep -Fq 'ghcr.io/${GITHUB_REPOSITORY,,}' "$image_workflow" || {
    echo "image workflow must publish to the fork GHCR repository: $image_workflow" >&2
    exit 1
  }

  if grep -Eq 'calciumion/new-api|dockerhub_enabled=true|DOCKERHUB_' "$image_workflow"; then
    echo "image workflow must not publish to upstream/DockerHub targets: $image_workflow" >&2
    exit 1
  fi
done

grep -Fq '"$GUARD" configure-remotes' "$sync_fork_script" || {
  echo "sync-fork.sh must configure remotes through the fork sync guard" >&2
  exit 1
}

grep -Fq 'git fetch "$UPSTREAM_NAME" --no-tags --prune' "$sync_fork_script" || {
  echo "sync-fork.sh must fetch upstream with --no-tags" >&2
  exit 1
}

grep -Fq '"$GUARD" push "$FORK_REMOTE"' "$sync_fork_script" || {
  echo "sync-fork.sh must push through the fork sync guard" >&2
  exit 1
}

for deploy_script in "$auto_sync_deploy_script" "$pull_restart_script" "$deploy_ghcr_script"; do
  [[ -f "$deploy_script" ]] || continue
  if grep -Eq 'docker build|docker compose|git merge upstream|git push upstream|Authorization: Bearer' "$deploy_script"; then
    echo "deploy automation must not perform local builds, local upstream merges, upstream pushes, or hardcoded bearer probes: $deploy_script" >&2
    exit 1
  fi
done

grep -Fq 'recover_stale_pod_storage' "$pull_restart_script" || {
  echo "pull-and-restart.sh must recover stale Podman pod storage once" >&2
  exit 1
}

grep -Fq 'recover_cached_plan' "$pull_restart_script" || {
  echo "pull-and-restart.sh must recover PostgreSQL cached-plan failures once" >&2
  exit 1
}

grep -Fq 'systemctl restart pgbouncer' "$pull_restart_script" || {
  echo "pull-and-restart.sh cached-plan recovery must restart PgBouncer once" >&2
  exit 1
}

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

grep -Eq 'RelayIdleConnTimeout' common/constants.go common/init.go || {
  echo "fork must preserve common.RelayIdleConnTimeout required by upstream protected fetch client" >&2
  exit 1
}

postgres_prepare_stmt="$(
  awk '
    /strings.HasPrefix\(dsn, "postgres:\/\// { in_pg=1 }
    in_pg && /PrepareStmt:/ { print $0 }
    in_pg && /return db, common.DatabaseTypePostgreSQL/ { in_pg=0 }
  ' "$model_main"
)"

if grep -Eq 'PrepareStmt:[[:space:]]+true' <<<"$postgres_prepare_stmt"; then
  echo "PostgreSQL must not use GORM PrepareStmt=true behind PgBouncer" >&2
  exit 1
fi

grep -Eq 'PrepareStmt:[[:space:]]+false' <<<"$postgres_prepare_stmt" || {
  echo "PostgreSQL must explicitly set GORM PrepareStmt=false behind PgBouncer" >&2
  exit 1
}

echo "upstream sync workflow guard passed"
