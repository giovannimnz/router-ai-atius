---
phase: 24-router-db-catalog-recovery-and-canonical-host-db
plan: "02"
subsystem: db-recovery
tags:
  - recovery
  - postgres
  - pgbouncer
  - catalog
  - clianything
dependency_graph:
  requires:
    - 24-01-SUMMARY.md
    - 24-CONTEXT.md
    - 24-RESEARCH.md
    - 24-PATTERNS.md
    - 24-VALIDATION.md
  provides:
    - scripts/phase24-build-canonical-db.sh
    - scripts/phase24-catalog-transform.sql
    - docs/ROUTER-DB-RECOVERY.md
  affects:
    - .planning/STATE.md
    - Phase 24 Plan 03
    - Phase 24 Plan 04
tech_stack:
  added:
    - bash
    - PostgreSQL tooling
    - psql variables
  patterns:
    - dry-run-first mutation guard
    - transformed catalog restore
    - rollback holdback
key_files:
  created:
    - scripts/phase24-build-canonical-db.sh
    - scripts/phase24-catalog-transform.sql
    - .planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/24-02-SUMMARY.md
  modified:
    - docs/ROUTER-DB-RECOVERY.md
key_decisions:
  - "Candidate DB build stays dry-run by default and only mutates with explicit source/target confirmations."
  - "The transformed catalog SQL removes forbidden rows by pattern and injects the Codex credential only from a secure runtime variable."
  - "newapi stays intact as rollback holdback; this plan does not authorize destructive rename or drop."
patterns-established:
  - "Phase 24 DB mutation scripts must require explicit execute/confirm gates and show pg_restore -l before restore."
  - "Catalog recovery should restore only the allowed Codex/DeepSeek/MiniMax/TEI rows and preserve secret material outside git."
requirements-completed:
  - PHASE-24-CANONICAL-HOST-DB
  - PHASE-24-CATALOG-RESTORE
  - PHASE-24-CUTOVER-ROLLBACK
metrics:
  started_at: 2026-07-04T04:37:00Z
  completed_at: 2026-07-04T04:44:48Z
status: complete
---

# Phase 24 Plan 02: Candidate Canonical DB Build Summary

Guarded candidate DB build script, transformed catalog restore SQL, and explicit `newapi` rollback holdback for the Phase 24 canonical host DB cutover.

## Outcomes

- Added [phase24-build-canonical-db.sh](/home/ubuntu/GitHub/containers/router-ai-atius/scripts/phase24-build-canonical-db.sh) with `dry-run` default, `--execute`, `--confirm-source`, `--confirm-target`, optional `--replace-target`, optional catalog-apply step, `pg_restore -l` archive inspection, PgBouncer mapping check, and post-restore verification queries.
- Added [phase24-catalog-transform.sql](/home/ubuntu/GitHub/containers/router-ai-atius/scripts/phase24-catalog-transform.sql) to reconcile the candidate catalog toward the required final state: restore `OpenAI - Codex`, keep DeepSeek active, keep MiniMax restored-but-disabled, preserve `embedding-gte-v1`, and avoid replaying forbidden long-context/Codex-embedding rows.
- Extended [ROUTER-DB-RECOVERY.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/ROUTER-DB-RECOVERY.md) with the explicit rollback holdback language required by the plan.

## Files Created/Modified

- [phase24-build-canonical-db.sh](/home/ubuntu/GitHub/containers/router-ai-atius/scripts/phase24-build-canonical-db.sh) - guarded candidate DB build flow with dump, restore, PgBouncer check, and verification steps.
- [phase24-catalog-transform.sql](/home/ubuntu/GitHub/containers/router-ai-atius/scripts/phase24-catalog-transform.sql) - transformed catalog restore for the candidate DB with secure Codex credential injection.
- [ROUTER-DB-RECOVERY.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/ROUTER-DB-RECOVERY.md) - Phase 24 rollback holdback rule for `newapi`.

## Verification

`bash -n scripts/phase24-build-canonical-db.sh` -> passed.

`rg -n "newapi|DBRouterAiAtius|pg_restore -l|confirm|dry-run" scripts/phase24-build-canonical-db.sh` -> matched all required gates.

`rg -n "OpenAI - Codex|gpt-5.5|gpt-5.4|gpt-5.4-mini|gpt-5.3-codex-spark" scripts/phase24-catalog-transform.sql` -> matched all required allow-list rows.

`! rg -n "gpt-5.4-1m|gpt-5.5-1m|text-embedding-3-small|text-embedding-3-large" scripts/phase24-catalog-transform.sql` -> passed.

`rg -n "newapi permanece intacto para rollback|não fazer DROP nem rename destrutivo de newapi nesta fase" docs/ROUTER-DB-RECOVERY.md` -> matched both required phrases.

## Commits

- `743dce97` — `feat(24-02): add candidate db build script`
- `f96998fb` — `feat(24-02): add transformed catalog restore sql`
- `d6b3abd6` — `docs(24-02): document rollback holdback for newapi`

## Decisions Made

- Keep candidate DB mutation behind explicit `--execute` plus source/target confirmations instead of allowing a one-flag destructive path.
- Restore channel/catalog shape from the 2026-07-01 snapshots, but keep secret material outside git and inject the Codex credential only at execution time.
- Preserve `newapi` as rollback holdback until later Phase 24 gates validate cutover end-to-end.

## Deviations from Plan

None - plan executed as specified.

## Issues Encountered

- The local Graphify CLI in this checkout exposes `build/query/status` instead of the older `update` verb. Execution used `graphify build` plus focused file reads for the Phase 24 artifacts.

## Known Stubs

- [scripts/phase24-catalog-transform.sql](/home/ubuntu/GitHub/containers/router-ai-atius/scripts/phase24-catalog-transform.sql:5) keeps the placeholder `__SET_FROM_SECURE_SOURCE__` and aborts unless `codex_channel_key_json` is injected securely at execution time. This is intentional because the plan forbids embedding live credentials in git.

## Next Phase Readiness

- Plan 24-03 can now reconcile provider/channel state against a reviewable candidate-build path instead of inventing mutation steps ad hoc.
- Plan 24-04 can reuse the rollback wording and the `newapi` holdback rule during runtime cutover.

## Self-Check: PASSED

- Found `scripts/phase24-build-canonical-db.sh`.
- Found `scripts/phase24-catalog-transform.sql`.
- Found `docs/ROUTER-DB-RECOVERY.md`.
- Found `24-02-SUMMARY.md`.
- Verified commits `743dce97`, `f96998fb`, and `d6b3abd6` exist in git history.
