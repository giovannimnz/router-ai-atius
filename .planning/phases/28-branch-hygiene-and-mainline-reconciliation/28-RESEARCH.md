# Phase 28 Research - branch-hygiene-and-mainline-reconciliation

**Date:** 2026-07-08  
**Status:** Ready for planning

## Worktree inventory

Observed local worktrees:

- `/home/ubuntu/GitHub/containers/router-ai-atius` -> `feat/pt-native`
- `/home/ubuntu/GitHub/containers/router-ai-atius-main-exec` -> `main`
- `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream` -> `feat/phase21-pt-native-upstream`
- `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean` -> `feat/brazilian-portuguese-localization`
- `/home/ubuntu/GitHub/containers/router-ai-atius-sync-fix` -> `fix/sync-upstream-tag-fetch`

## Divergence findings

### `feat/pt-native`

- Ahead `339` / behind `66` vs `origin/main`
- Ahead `145` / behind `66` vs `upstream/main`
- Contains mixed content:
  - planning artifacts for Phases 21-27
  - runtime/catalog/provider work
  - docs/CI work
  - not a clean PT-native lane

Conclusion:

- integration/planning branch only
- not suitable as upstream PR base
- do not merge wholesale

### `feat/phase21-pt-native-upstream`

- Behind `11` vs `origin/main`
- Behind `0` vs `upstream/main`
- Local worktree still carries uncommitted PT-native implementation/handoff material
- This is the cleanest lane for the future upstream PR

Conclusion:

- canonical PT-native handoff branch candidate

### `feat/brazilian-portuguese-localization`

- Ahead `1` / behind `12` vs `upstream/main`
- Contains only PT-native implementation-style changes, but is not the designated GSD handoff lane

Conclusion:

- useful historical/reference branch
- redundant after canonical handoff branch is promoted

### `feat/pt-native-i18n-clean`

- Remote branch at `cd8cb89bb`
- Matches the PT-native-only shape and was recorded in Obsidian as a translation source

Conclusion:

- keep only if needed as historical translation source
- otherwise redundant once canonical handoff branch is preserved

### `main` local worktree

- Local `main` worktree is behind `origin/main`
- Not trustworthy as a current fork mainline

Conclusion:

- recreate from `origin/main`, do not repair in place

## Obsidian / planning evidence

- `2026-07-04-router-phase21-ptbr-native-replan.md` already states:
  - reuse PT strings by value only
  - no cherry-pick from polluted branches
  - `feat/pt-native-i18n-clean` is a source, not the final handoff lane
- Current GSD state:
  - Phase 21: executed locally, no incomplete plans
  - Phase 22: 4 plans, 0 summaries, not started
  - Phase 23: 1 plan, 0 summaries, not started

## Recommended branch policy

### Preserve

- `origin/main`
- one remote PT-native upstream handoff branch only

### Recreate

- local `main` worktree from `origin/main`

### Retire

- local `feat/pt-native`
- local `feat/brazilian-portuguese-localization`
- local `fix/sync-upstream-tag-fetch`
- local `feat/phase21-pt-native-upstream` after the remote handoff branch is safe, if the operator wants remote-only preservation

### Delete remotely after reconciliation

- `origin/feat/pt-native`
- likely `origin/feat/pt-native-i18n-clean` unless explicitly kept as archival reference

## Recommendation

Proceed with a four-wave cleanup:

1. backup and freeze
2. promote Phase 21 canonical remote lane
3. reconcile a clean branch into `origin/main`
4. clean local and remote stale lanes
