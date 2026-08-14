#!/usr/bin/env bash
set -euo pipefail

repo_root="${ATIUS_USER_QUOTA_REPO_ROOT:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)}"
patch_file="$repo_root/patches/atius-user-quota-unlimited.patch"
mode="${1:-audit}"

runtime_files=(
  "service/pre_consume_quota.go"
  "service/billing_session.go"
  "service/quota.go"
  "relay/mjproxy_handler.go"
)

die() {
  echo "atius-user-quota-guard: $*" >&2
  exit 1
}

audit_runtime() {
  local file matches
  local failed=0
  local pattern='GetUserQuota|ErrorCodeInsufficientUserQuota|quota_not_enough|user quota is not enough|userQuota[[:space:]]*<|userQuota[[:space:]]*-[^;]*<[[:space:]]*0'

  for file in "${runtime_files[@]}"; do
    [[ -f "$repo_root/$file" ]] || {
      echo "atius-user-quota-guard: missing runtime file: $file" >&2
      failed=1
      continue
    }
    matches="$(grep -nE "$pattern" "$repo_root/$file" || true)"
    if [[ -n "$matches" ]]; then
      echo "atius-user-quota-guard: forbidden local balance blocker in $file:" >&2
      echo "$matches" >&2
      failed=1
    fi
  done

  [[ "$failed" -eq 0 ]]
}

audit() {
  local failed=0
  if [[ ! -s "$patch_file" ]]; then
    echo "atius-user-quota-guard: canonical patch missing or empty: $patch_file" >&2
    failed=1
  fi
  if ! audit_runtime; then
    failed=1
  fi
  [[ "$failed" -eq 0 ]] || return 1
  echo "atius-user-quota-guard: audit OK"
}

repair() {
  if audit >/dev/null 2>&1; then
    echo "atius-user-quota-guard: repair not needed; invariant already satisfied"
    return 0
  fi

  [[ -s "$patch_file" ]] || die "cannot repair without canonical patch: $patch_file"

  local backup_dir file
  backup_dir="$(mktemp -d "${TMPDIR:-/tmp}/atius-user-quota-guard-backup.XXXXXX")"
  for file in "${runtime_files[@]}" types/error.go; do
    [[ -f "$repo_root/$file" ]] || continue
    mkdir -p "$backup_dir/$(dirname -- "$file")"
    cp -p "$repo_root/$file" "$backup_dir/$file"
  done
  echo "atius-user-quota-guard: backup=$backup_dir"

  if git -C "$repo_root" apply --check "$patch_file"; then
    git -C "$repo_root" apply "$patch_file"
  elif git -C "$repo_root" apply --reverse --check "$patch_file"; then
    echo "atius-user-quota-guard: canonical patch already applied"
  else
    die "canonical patch does not apply cleanly; source drift requires manual port (backup: $backup_dir)"
  fi

  audit || die "post-repair audit failed"
  echo "atius-user-quota-guard: repair complete"
}

case "$mode" in
  audit)
    audit || die "audit failed"
    ;;
  repair)
    repair
    ;;
  *)
    die "usage: scripts/atius-user-quota-guard.sh [audit|repair]"
    ;;
esac
