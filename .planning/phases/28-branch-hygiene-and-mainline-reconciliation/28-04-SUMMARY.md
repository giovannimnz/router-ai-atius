---
phase: 28-branch-hygiene-and-mainline-reconciliation
plan: "04"
type: summary
status: complete
completed_at: "2026-07-08T18:08:00-03:00"
requirements:
  - PHASE-28-LOCAL-HYGIENE
  - PHASE-28-REMOTE-HYGIENE
  - PHASE-28-BRANCH-POLICY
---

# 28-04 Summary - local and remote hygiene

## Result

Phase 28 branch hygiene is complete.

Local final state:

- one worktree: `/home/ubuntu/GitHub/containers/router-ai-atius`
- one local branch: `main`
- `main` tracks `origin/main` at `505810fc9`

Remote final state:

- kept `origin/main`
- kept `origin/feat/phase21-pt-native-upstream`
- deleted `origin/feat/pt-native`
- deleted `origin/feat/pt-native-i18n-clean`

## Safety Backup

Before destructive cleanup, a final Wave 4 backup was created at:

- `/home/ubuntu/GitHub/containers/router-ai-atius-phase28-wave4-backup-20260708T210137Z`

The backup includes:

- per-worktree `git status`
- per-worktree binary diffs and cached diffs
- per-worktree untracked file lists
- tarballs for untracked files where present
- backup tags for the removed branch heads

## Removed Locally

- `/home/ubuntu/GitHub/containers/router-ai-atius-main-exec`
- `/home/ubuntu/GitHub/containers/router-ai-atius-mainline-reconcile`
- `/home/ubuntu/GitHub/containers/router-ai-atius-mainline-reconcile-clean`
- `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`
- `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean`
- `/home/ubuntu/GitHub/containers/router-ai-atius-sync-fix`

Deleted local branches:

- `feat/pt-native`
- `feat/brazilian-portuguese-localization`
- `feat/pt-native-i18n-clean`
- `feat/phase21-pt-native-upstream`
- `fix/sync-upstream-tag-fetch`
- `reconcile/v2.14-mainline`
- `reconcile/v2.14-mainline-clean`

## Verification

- `git status --short --branch`
- `git worktree list`
- `git branch -vv`
- `git branch -r`
- `git ls-remote --heads origin`
- `rg` policy check in `docs/BRANCH-HYGIENE-PT-NATIVE.md`

Final verified remote heads:

- `refs/heads/main` at `505810fc9`
- `refs/heads/feat/phase21-pt-native-upstream` at `7008eda67`
