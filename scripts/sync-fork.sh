#!/usr/bin/env bash
#
# sync-fork.sh — Sync atius-ai-router with upstream NewAPI
#
# Usage:
#   ./scripts/sync-fork.sh [--strategy ours|theirs] [--dry-run] [--branch <branch>]
#
# Options:
#   --strategy   Conflict resolution: 'theirs' (prefer upstream, default) or 'ours' (prefer fork)
#   --branch     Branch to sync (default: main)
#   --dry-run    Show what would be done without making changes
#   -h, --help   Show this help message
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/QuantumNous/new-api.git}"
UPSTREAM_NAME="upstream"
BRANCH="${SYNC_BRANCH:-main}"
STRATEGY="${SYNC_STRATEGY:-theirs}"  # theirs = prefer upstream, ours = prefer fork
DRY_RUN=false

# Protected files — never overwrite from upstream
PROTECTED_PATTERNS=(
    "integration/middleware/model_detailed.py"
    ".planning/"
    "FORK_MIGRATION.md"
)

# Files to re-apply customizations after merge
RESTORE_PATTERNS=(
    "docker-compose.yml"
)

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --strategy) STRATEGY="$2"; shift 2 ;;
        --strategy=*) STRATEGY="${1#*=}"; shift ;;
        --branch) BRANCH="$2"; shift 2 ;;
        --branch=*) BRANCH="${1#*=}"; shift ;;
        --dry-run) DRY_RUN=true; shift ;;
        -h|--help)
            echo "Usage: $0 [--strategy ours|theirs] [--branch <branch>] [--dry-run]"
            echo ""
            echo "Options:"
            echo "  --strategy   Conflict resolution: 'theirs' (prefer upstream, default) or 'ours' (prefer fork)"
            echo "  --branch     Branch to sync (default: main)"
            echo "  --dry-run    Show what would be done without making changes"
            echo "  -h, --help   Show this help message"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

cd "$REPO_ROOT"

echo "=== atius-ai-router Fork Sync ==="
echo "Upstream:  $UPSTREAM_URL"
echo "Branch:    $BRANCH"
echo "Strategy:  $STRATEGY (conflict resolution)"
echo "Dry run:   $DRY_RUN"
echo ""

# Step 1: Ensure upstream remote exists
if ! git remote | grep -q "^${UPSTREAM_NAME}$"; then
    echo "[1/7] Adding upstream remote: $UPSTREAM_NAME -> $UPSTREAM_URL"
    [[ "$DRY_RUN" == true ]] || git remote add "$UPSTREAM_NAME" "$UPSTREAM_URL"
else
    CURRENT_URL="$(git remote get-url "$UPSTREAM_NAME")"
    echo "[1/7] Upstream remote already exists: $UPSTREAM_NAME -> $CURRENT_URL"
    [[ "$DRY_RUN" == true ]] || git remote set-url "$UPSTREAM_NAME" "$UPSTREAM_URL"
fi

# Step 2: Fetch upstream
echo "[2/7] Fetching from upstream..."
[[ "$DRY_RUN" == true ]] || git fetch "$UPSTREAM_NAME" --prune

# Step 3: Checkout target branch
CURRENT_BRANCH="$(git branch --show-current)"
if [[ "$CURRENT_BRANCH" != "$BRANCH" ]]; then
    echo "[3/7] Switching to branch: $BRANCH (currently on: $CURRENT_BRANCH)"
    [[ "$DRY_RUN" == true ]] || git checkout "$BRANCH"
else
    echo "[3/7] Already on branch: $BRANCH"
fi

# Step 4: Pull latest from origin
echo "[4/7] Pulling latest from origin..."
if [[ "$DRY_RUN" == true ]]; then
    echo "  (skipped - dry run)"
else
    git pull origin "$BRANCH" --rebase || {
        echo "ERROR: Failed to pull from origin. Resolve conflicts first."
        exit 1
    }
fi

# Step 5: Merge from upstream
echo "[5/7] Merging upstream/$BRANCH into $BRANCH..."
if [[ "$DRY_RUN" == true ]]; then
    echo "  (skipped - dry run)"
else
    MERGE_OUTPUT="$(git merge "$UPSTREAM_NAME/$BRANCH" --no-edit -X "$STRATEGY" 2>&1)" || {
        echo ""
        echo "WARNING: Merge had conflicts. Manual intervention needed."
        echo "Merge output: $MERGE_OUTPUT"
        echo ""
        echo "To continue manually:"
        echo "  1. Fix conflicts: git status"
        echo "  2. Stage fixes:    git add <files>"
        echo "  3. Complete merge: git commit"
        echo "  4. Push:           git push origin $BRANCH"
        echo ""
        echo "Or abort: git merge --abort"
        exit 1
    }
    echo "  $MERGE_OUTPUT"

    # Step 6: Restore protected files
    echo ""
    echo "[6/7] Restoring protected files..."
    for pattern in "${PROTECTED_PATTERNS[@]}"; do
        if [[ -e "$pattern" ]]; then
            if ! git diff --quiet HEAD -- "$pattern" 2>/dev/null; then
                echo "  Restoring protected: $pattern"
                git checkout HEAD -- "$pattern" 2>/dev/null || true
            else
                echo "  Protected (intact): $pattern"
            fi
        fi
    done

    # Restore docker-compose.yml if it was changed
    for pattern in "${RESTORE_PATTERNS[@]}"; do
        if [[ -e "$pattern" ]]; then
            if ! git diff --quiet HEAD -- "$pattern" 2>/dev/null; then
                echo "  Restoring: $pattern"
                git checkout HEAD -- "$pattern" 2>/dev/null || true
            fi
        fi
    done

    # Commit restored files if anything changed
    if ! git diff --cached --quiet 2>/dev/null && ! git diff --quiet 2>/dev/null; then
        echo "  All protected files intact, no changes needed."
    elif [[ "$(git status --porcelain)" != "" ]]; then
        echo "  Committing restored files..."
        git add -A
        git commit -m "chore: restore fork overrides after upstream merge" 2>/dev/null || true
    fi
fi

# Step 7: Version bump (if version-bump.sh exists)
echo "[7/7] Checking version..."
if [[ "$DRY_RUN" == true ]]; then
    echo "  (skipped - dry run)"
else
    if [[ -f "$SCRIPT_DIR/version-bump.sh" ]]; then
        CURRENT_VERSION="$(cat VERSION 2>/dev/null || echo "unknown")"
        echo "  Current version: $CURRENT_VERSION"
        $SCRIPT_DIR/version-bump.sh 2>&1 || echo "  WARNING: version bump failed"
    else
        echo "  version-bump.sh not found, skipping"
    fi
fi

# Step 8: Push to origin
if [[ "$DRY_RUN" == true ]]; then
    echo ""
    echo "[8/8] Push to origin (skipped - dry run)"
else
    echo "[8/8] Pushing to origin..."
    BEHIND="$(git rev-list --count "origin/$BRANCH..HEAD" 2>/dev/null || echo "0")"
    if [[ "$BEHIND" -eq 0 ]]; then
        echo "  Already up to date, nothing to push."
    else
        echo "  Pushing $BEHIND commit(s) to origin/$BRANCH..."
        git push origin "$BRANCH"
    fi
fi

echo ""
echo "=== Sync complete! ==="
if [[ "$DRY_RUN" == true ]]; then
    echo "(Dry run - no changes were made)"
fi
