# Phase 28: branch-hygiene-and-mainline-reconciliation - Context

**Gathered:** 2026-07-08  
**Status:** Ready for planning  
**Source:** git branch/worktree audit, local planning state, Obsidian logs, and repo-local branch hygiene review

<domain>
## Phase Boundary

This phase is not about provider/runtime feature work. It is a repository hygiene and handoff phase.

Its purpose is to:

- preserve recoverability before any destructive cleanup
- promote one clean canonical PT-native upstream handoff branch
- reconcile what truly belongs on `origin/main`
- remove stale local/remote branch ambiguity after the safe state exists

It must not re-implement Phase 21 language work, Phase 22 k3s migration, or Phase 23 long-context validation.
</domain>

<decisions>
## Implementation Decisions

- **D-01 — `origin/main` is authoritative:** the local `main` worktree is stale and must not be trusted until recreated from `origin/main`.
- **D-02 — `feat/pt-native` is integration only:** it is not a clean upstream PR base and must not be used for PT-native upstream handoff.
- **D-03 — `feat/phase21-pt-native-upstream` is the clean lane:** this worktree/branch is the correct source for the final PT-native upstream handoff.
- **D-04 — One canonical remote PT branch only:** after promotion, there should be exactly one preserved remote PT-native handoff branch.
- **D-05 — Reconciliation to `main` must be selective:** create a clean branch from `origin/main` and port only what belongs there. Do not merge `feat/pt-native` wholesale.
- **D-06 — Backup before destruction:** no worktree/branch deletion happens before status snapshot, patch capture, and backup refs/tags are recorded.
- **D-07 — Phase 22 and 23 move out of the completed `v2.13`:** they belong to a future milestone and must not make the just-closed recovery milestone look open.
</decisions>

<canonical_refs>
## Canonical References

**Repo state**
- `docs/BRANCH-HYGIENE-PT-NATIVE.md`
- `.planning/STATE.md`
- `.planning/ROADMAP.md`
- `.planning/PROJECT.md`

**Phase 21 handoff artifacts**
- `.planning/phases/21-feat-pt-native-pr/21-UPSTREAM-HANDOFF.md`
- `.planning/phases/21-feat-pt-native-pr/21-05-SUMMARY.md`
- `.planning/phases/21-feat-pt-native-pr/21-REVIEWS.md`

**Recent closeout state to port selectively**
- `.planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/`
- `.planning/phases/25-embedding-governor-auto-workload-inference/`
- `.planning/phases/26-codex-dynamic-discovery-and-curated-catalog/`
- `.planning/phases/27-codex-official-docs-ci-and-release-alignment/`

**Cross-session notes**
- `/home/ubuntu/GitHub/obsidian-vault/AiSecondBrain/60-LOGS/2026-07-04-router-phase21-ptbr-native-replan.md`
- `/home/ubuntu/GitHub/obsidian-vault/AiSecondBrain/60-LOGS/2026-07-08-router-ai-atius-branch-hygiene.md`
</canonical_refs>

<specifics>
## Specific Ideas

- Record one reversible backup namespace per worktree before any deletion.
- Treat local worktrees as disposable after their authoritative state is either merged or preserved remotely.
- Prefer archiving obsolete remote branches with tags or refs only if they contain irreplaceable history that is not already preserved elsewhere.
- Keep final branch policy explicit in docs so this branch drift does not reappear.
</specifics>

<deferred>
## Deferred Ideas

- Executing Phase 22 k3s migration work
- Executing Phase 23 long-context validation work
- Opening the upstream PT-native PR itself before the clean branch is promoted and verified
</deferred>

---

*Phase: 28-branch-hygiene-and-mainline-reconciliation*  
*Context gathered: 2026-07-08*
