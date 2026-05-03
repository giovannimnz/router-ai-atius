#!/usr/bin/env bash
#
# version-bump.sh — Bump fork version based on upstream changes
#
# Logic:
#   1. Fetch upstream latest version/tag
#   2. Read current version from VERSION file
#   3. If upstream base changed → reset suffix to .1
#   4. If upstream base same → increment suffix
#   5. Write new version to VERSION and create git tag
#
# Usage:
#   ./scripts/version-bump.sh          # auto-increment
#   ./scripts/version-bump.sh --check  # show what would happen, no write
#   -h, --help                        # show this help
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

UPSTREAM_NAME="upstream"
VERSION_FILE="$REPO_ROOT/VERSION"

# Parse arguments
CHECK_ONLY=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --check) CHECK_ONLY=true; shift ;;
        -h|--help)
            echo "Usage: $0 [--check]"
            echo ""
            echo "Options:"
            echo "  --check   Show what would happen without making changes"
            echo "  -h, --help  Show this help message"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

cd "$REPO_ROOT"

echo "=== version-bump.sh ==="

# Get current version
if [[ -f "$VERSION_FILE" ]]; then
    CURRENT_VERSION="$(cat "$VERSION_FILE" | tr -d '[:space:]')"
else
    CURRENT_VERSION="0.0.0.0"
fi

# Parse fork version: X.Y.Z.N format (strips 'v' prefix)
# If no suffix yet, treat as X.Y.Z.0
parse_fork_version() {
    local version="$1"
    version="${version#v}"  # Strip 'v' prefix if present
    if [[ "$version" =~ ^([0-9]+\.[0-9]+\.[0-9]+)\.([0-9]+)$ ]]; then
        echo "${BASH_REMATCH[1]} ${BASH_REMATCH[2]}"
    else
        echo "$version 0"
    fi
}

# Get upstream version from git tag (follows upstream's latest release, including pre-releases like -rc, -alpha, -beta)
get_upstream_version() {
    # Get latest tag from upstream (follows upstream exactly — rc, alpha, etc. are all valid releases)
    local upstream_tag
    upstream_tag="$(git ls-remote --tags "$UPSTREAM_NAME" 2>/dev/null | \
                    grep -v '\^{}' | \
                    awk '{print $2}' | \
                    sed 's|refs/tags/||' | \
                    grep -v '^$' | \
                    sort -V | \
                    tail -1)"

    if [[ -z "$upstream_tag" ]]; then
        # Fallback: read from upstream VERSION file
        upstream_tag="$(git show "$UPSTREAM_NAME/main:VERSION" 2>/dev/null | tr -d '[:space:]' || \
                       git show "$UPSTREAM_NAME/main:package.json" 2>/dev/null | \
                       grep '"version"' | \
                       sed 's/.*"version"\s*:\s*"\([^"]*\)".*/\1/' | \
                       tr -d '[:space:]')"
    fi

    # Strip 'v' prefix and refs/tags/ from tag name
    echo "${upstream_tag:-unknown}" | sed 's/^v//'
}

# Main logic
UPSTREAM_VERSION="$(get_upstream_version)"
read -r BASE CURRENT_SUFFIX <<< "$(parse_fork_version "$CURRENT_VERSION")"

if [[ "$UPSTREAM_VERSION" == "unknown" ]]; then
    echo "WARNING: Could not determine upstream version"
    echo "  Setting base to current version without upstream sync"
    UPSTREAM_VERSION="$BASE"
fi

echo "Current version:  $CURRENT_VERSION"
echo "Upstream version: $UPSTREAM_VERSION"
echo "Base:             $BASE"
echo "Suffix:           $CURRENT_SUFFIX"

# Determine new suffix
if [[ "$BASE" != "$UPSTREAM_VERSION" ]]; then
    # Upstream base changed — reset suffix to 1
    NEW_SUFFIX=1
    echo ""
    echo "Upstream base changed: $BASE -> $UPSTREAM_VERSION"
else
    # Same upstream base — increment suffix
    NEW_SUFFIX=$((CURRENT_SUFFIX + 1))
    echo ""
    echo "Upstream base unchanged: $UPSTREAM_VERSION"
fi

NEW_VERSION="${UPSTREAM_VERSION}.${NEW_SUFFIX}"
echo "New fork version: $NEW_VERSION"

if [[ "$CHECK_ONLY" == true ]]; then
    echo ""
    echo "(Dry run — no changes made)"
    exit 0
fi

# Write new version to VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"
echo "Updated VERSION to $NEW_VERSION"

# Create git tag
git tag -f "v$NEW_VERSION" 2>/dev/null || true
echo "Tagged: v$NEW_VERSION"

echo ""
echo "=== Version bump complete ==="