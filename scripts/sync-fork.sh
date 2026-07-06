#!/usr/bin/env bash
#
# sync-fork.sh — Automatically sync atius-ai-router with upstream NewAPI
#
# Features:
# - Merges upstream with "theirs" strategy (prefer upstream on conflict)
# - Protects and restores fork-specific files after merge
# - Auto-commits restored overrides
# - Auto-bumps fork version
# - Auto-pushes only to the fork remote if changes detected
#
# Usage:
#   ./scripts/sync-fork.sh [--dry-run] [--strategy ours|theirs]
#
# GitHub Actions: runs daily at 03:00 UTC via sync.yml
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
GUARD="$SCRIPT_DIR/fork-sync-guard.sh"

FORK_REPO="${FORK_REPO:-giovannimnz/router-ai-atius}"
FORK_REMOTE="${FORK_REMOTE:-origin}"
UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/QuantumNous/new-api.git}"
UPSTREAM_NAME="upstream"
BRANCH="${SYNC_BRANCH:-main}"
STRATEGY="${SYNC_STRATEGY:-theirs}"  # theirs = prefer upstream, ours = prefer fork
DRY_RUN="${DRY_RUN:-false}"
SILENT="${SILENT:-false}"

log() {
    [[ "$SILENT" == "true" ]] && return
    echo "[$(date '+%H:%M:%S')] $*"
}

# Protected paths — always restore these from the fork baseline after merge
PROTECTED_PATHS=(
    ".dockerignore"
    ".github/workflows"
    "docker-compose.yml"
    "integration/middleware/model_detailed.py"
    "integration/middleware/model_details.py"
    "integration/middleware/model_enrichment.py"
    "controller/model.go"
    "controller/model_list_test.go"
    "relay/common/relay_utils.go"
    "relay/common/relay_utils_test.go"
    "relay/embedding_handler.go"
    "relay/channel/codex"
    "service/codex_auth.go"
    "service/codex_device.go"
    "service/codex_models.go"
    "service/modelcatalog"
)

# Fork-specific files/dirs that exist only in this fork (never in upstream)
FORK_ONLY=(
    ".planning/"
    "agent-harness/"
    "integration/bruno-tests/"
    "scripts/run-bruno-tests.sh"
    ".github/workflows/sync.yml"
    ".github/workflows/release.yml"
)

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true; shift ;;
        --strategy) STRATEGY="$2"; shift 2 ;;
        --strategy=*) STRATEGY="${1#*=}"; shift ;;
        --silent) SILENT=true; shift ;;
        -h|--help)
            echo "Usage: $0 [--dry-run] [--strategy ours|theirs] [--silent]"
            echo ""
            echo "Options:"
            echo "  --dry-run    Preview without making changes"
            echo "  --strategy   Conflict resolution: 'theirs' (default) or 'ours'"
            echo "  --silent     Minimize output (for cron)"
            echo "  -h, --help   This help"
            exit 0
            ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

cd "$REPO_ROOT"

[[ "$SILENT" != "true" ]] && echo "=== atius-ai-router Fork Sync ==="
[[ "$SILENT" != "true" ]] && echo "Upstream:  $UPSTREAM_URL"
[[ "$SILENT" != "true" ]] && echo "Branch:    $BRANCH"
[[ "$SILENT" != "true" ]] && echo "Strategy:  $STRATEGY"
[[ "$SILENT" != "true" ]] && echo "Dry run:   $DRY_RUN"
[[ "$SILENT" != "true" ]] && echo ""

# ─── Step 1: Upstream remote ───────────────────────────────────────────────
if [[ "$DRY_RUN" == "true" ]]; then
    log "[1/8] Would configure fork/upstream remotes"
else
    log "[1/8] Configuring fork/upstream remotes"
    "$GUARD" configure-remotes
fi

# ─── Step 2: Fetch upstream ────────────────────────────────────────────────
log "[2/8] Fetching upstream without tags..."
[[ "$DRY_RUN" == "true" ]] || git fetch "$UPSTREAM_NAME" --no-tags --prune "+refs/heads/${BRANCH}:refs/remotes/${UPSTREAM_NAME}/${BRANCH}"

# Check if there are new commits
UPSTREAM_SHA="$(git rev-parse "${UPSTREAM_NAME}/${BRANCH}" 2>/dev/null)"
LOCAL_SHA="$(git rev-parse "${BRANCH}" 2>/dev/null)"
COMMON_ANCESTOR="$(git merge-base "${BRANCH}" "${UPSTREAM_NAME}/${BRANCH}" 2>/dev/null || echo "")"

if [[ "$COMMON_ANCESTOR" == "$UPSTREAM_SHA" ]]; then
    log "Upstream is already in our history — nothing to sync."
    log "Nothing to push, sync complete."
    exit 0
fi

UPSTREAM_COMMITS_BEHIND="$(git log --oneline "${COMMON_ANCESTOR}..${UPSTREAM_NAME}/${BRANCH}" 2>/dev/null | wc -l)"
[[ "$SILENT" != "true" ]] && log "Upstream has $UPSTREAM_COMMITS_BEHIND new commit(s) behind us."

# ─── Step 3: Ensure on correct branch ─────────────────────────────────────
CURRENT_BRANCH="$(git branch --show-current)"
if [[ "$CURRENT_BRANCH" != "$BRANCH" ]]; then
    log "[3/8] Switching to branch: $BRANCH"
    [[ "$DRY_RUN" == "true" ]] || git checkout "$BRANCH"
else
    log "[3/8] Already on branch: $BRANCH"
fi

# ─── Step 4: Pull latest from origin ─────────────────────────────────────
log "[4/8] Pulling from $FORK_REMOTE..."
if [[ "$DRY_RUN" == "true" ]]; then
    log "  (skipped — dry run)"
else
    if ! git pull "$FORK_REMOTE" "$BRANCH" --rebase 2>/dev/null; then
        log "WARNING: Pull failed, trying merge..."
        git pull "$FORK_REMOTE" "$BRANCH" --no-rebase 2>/dev/null || true
    fi
fi

# ─── Step 5: Merge upstream ───────────────────────────────────────────────
log "[5/8] Merging upstream/${BRANCH}..."
if [[ "$DRY_RUN" == "true" ]]; then
    log "  (skipped — dry run)"
else
    BASE_HEAD="$(git rev-parse HEAD)"
    MERGE_FAILED=false
    MERGE_OUTPUT="$(git merge "${UPSTREAM_NAME}/${BRANCH}" --no-edit -X "$STRATEGY" 2>&1)" || MERGE_FAILED=true

    if [[ "$MERGE_FAILED" == "true" ]]; then
        log "WARNING: Merge had conflicts."
        log "Output: $MERGE_OUTPUT"
        log "Aborting merge and exiting..."
        git merge --abort 2>/dev/null || true
        exit 1
    fi

    [[ "$SILENT" != "true" ]] && [[ -n "$MERGE_OUTPUT" ]] && log "  $MERGE_OUTPUT"
fi

# ─── Step 6: Restore protected files ──────────────────────────────────────
log "[6/8] Restoring protected fork paths after merge..."

if [[ "$DRY_RUN" == "true" ]]; then
    log "  (skipped — dry run)"
else
    for file in "${PROTECTED_PATHS[@]}"; do
        log "  Restoring fork baseline: $file"
        git rm -r -f --ignore-unmatch -- "$file" >/dev/null 2>&1 || true
        rm -rf -- "$file"
        if git cat-file -e "${BASE_HEAD}:${file}" 2>/dev/null; then
            git checkout "$BASE_HEAD" -- "$file"
        fi
        git add -A -- "$file" 2>/dev/null || true
    done

    # Fork-only files: check they weren't deleted
    for file in "${FORK_ONLY[@]}"; do
        if [[ ! -e "$file" ]] && [[ "$(git ls-files "$file" 2>/dev/null)" == "$file" ]]; then
            log "  WARNING: Fork-only file deleted by merge: $file"
            log "  Restoring from git..."
            git checkout "$BASE_HEAD" -- "$file" 2>/dev/null || true
            git add -A -- "$file" 2>/dev/null || true
        fi
    done

    # Commit restored files
    if ! git diff --cached --quiet 2>/dev/null; then
        log "  Committing restored fork paths..."
        git commit -m "chore: restore fork overrides after upstream merge

Upstream: ${UPSTREAM_NAME}/${BRANCH} (${UPSTREAM_SHA:0:8})
Protected paths restored: ${PROTECTED_PATHS[*]}
Fork-only files verified: ${FORK_ONLY[*]}" --allow-empty 2>/dev/null || true
    else
        log "  Protected fork paths already match baseline."
    fi
fi

# ─── Step 7: Version bump ───────────────────────────────────────────────────
log "[7/8] Version bump..."
if [[ "$DRY_RUN" == "true" ]]; then
    log "  (skipped — dry run)"
else
    UPSTREAM_TAG="$(git ls-remote --tags --refs "$UPSTREAM_NAME" 'refs/tags/v*' 2>/dev/null | awk '{print $2}' | sed 's|refs/tags/||' | grep -v '^$' | sort -V | tail -1 | sed 's/^v//')"
    if [[ -z "$UPSTREAM_TAG" ]]; then
        log "WARNING: could not detect upstream version tag; skipping version bump"
    else
        CURRENT_VERSION="$(cat VERSION 2>/dev/null | tr -d '[:space:]' || echo "0.0.0.0")"
        NEW_VERSION="$("$SCRIPT_DIR/next-fork-version.sh" "$CURRENT_VERSION" "$UPSTREAM_TAG")"
        if [[ "$NEW_VERSION" != "$CURRENT_VERSION" ]]; then
            echo "$NEW_VERSION" > VERSION
            git add VERSION
            git commit -m "chore: bump fork version to $NEW_VERSION" 2>/dev/null || true
            git tag "v${NEW_VERSION}" 2>/dev/null || true
        fi
    fi

    # Check if version changed
    NEW_VERSION="$(cat VERSION 2>/dev/null || echo "")"
    if [[ -n "$NEW_VERSION" ]]; then
        [[ "$SILENT" != "true" ]] && log "  Version: $NEW_VERSION"
    fi
fi

# ─── Step 8: Push ─────────────────────────────────────────────────────────
log "[8/8] Pushing to origin..."

if [[ "$DRY_RUN" == "true" ]]; then
    BEHIND="$(git rev-list --count "origin/${BRANCH}..HEAD" 2>/dev/null || echo "0")"
    log "  Would push $BEHIND commit(s) to origin/${BRANCH}"
    log "  (skipped — dry run)"
else
    BEHIND="$(git rev-list --count "origin/${BRANCH}..HEAD" 2>/dev/null || echo "0")"
    if [[ "$BEHIND" -eq 0 ]]; then
        log "Already up to date, nothing to push."
    else
        log "Pushing $BEHIND commit(s) to origin/${BRANCH}..."
        if "$GUARD" push "$FORK_REMOTE" "$BRANCH" 2>&1; then
            log "Push successful."

            # Push new tag if exists
            NEW_TAG="$(cat VERSION 2>/dev/null || echo "")"
            if [[ -n "$NEW_TAG" ]]; then
                if git tag -l "v${NEW_TAG}" | grep -q "v${NEW_TAG}"; then
                    log "Pushing tag v${NEW_TAG}..."
                    "$GUARD" push "$FORK_REMOTE" "v${NEW_TAG}" 2>/dev/null || log "  (tag push skipped)"
                fi
            fi
        else
            log "WARNING: Push failed. May need force push."
        fi
    fi
fi

[[ "$SILENT" != "true" ]] && echo ""
[[ "$SILENT" != "true" ]] && echo "=== Sync complete ==="
