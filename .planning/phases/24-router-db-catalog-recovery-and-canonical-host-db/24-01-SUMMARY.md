---
phase: 24-router-db-catalog-recovery-and-canonical-host-db
plan: "01"
subsystem: db-recovery
tags:
  - recovery
  - postgres
  - pgbouncer
  - clianything
dependency_graph:
  requires:
    - 24-CONTEXT.md
    - 24-RESEARCH.md
    - 24-PATTERNS.md
    - 24-VALIDATION.md
  provides:
    - docs/ROUTER-DB-RECOVERY.md
    - scripts/phase24-db-preflight.sh
  affects:
    - .planning/STATE.md
tech_stack:
  added:
    - bash
    - PostgreSQL tooling
  patterns:
    - read-only preflight
    - backup-before-mutation
    - live-data-preservation
key_files:
  created:
    - docs/ROUTER-DB-RECOVERY.md
    - scripts/phase24-db-preflight.sh
    - .planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/24-01-SUMMARY.md
  modified:
    - .planning/STATE.md
decisions:
  - Preserve live operational data in current newapi and use the 2026-07-01 catalog snapshot only for missing catalog rows.
  - Keep the preflight strictly read-only and prove backup candidates with pg_restore -l before any later mutation plan.
metrics:
  started_at: 2026-07-04T00:00:00-03:00
  completed_at: 2026-07-04T04:35:00-03:00
status: complete
---

# Phase 24 Plan 01: Router DB Recovery Contract Summary

Recovery contract for canonical host DB cutover with live-data preservation and read-only preflight evidence.

## Outcomes

- Added [ROUTER-DB-RECOVERY.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/ROUTER-DB-RECOVERY.md) in PT-BR with the required sections: `Estado atual`, `Fontes de restauracao`, `Transformacoes obrigatorias`, `Banco final canonico`, `Backups obrigatorios`, `Mutacao segura`, `Rollback`, and `Validacao final`.
- Added [phase24-db-preflight.sh](/home/ubuntu/GitHub/containers/router-ai-atius/scripts/phase24-db-preflight.sh) as a read-only preflight covering Graphify freshness, `bin/clianything status --strict`, catalog counts, host DB inventory, router unit DB target, snapshot file inventory, and `pg_restore -l` archive inspection.
- Merged the minimum Phase 24 execution note into `.planning/STATE.md` in the working tree so the repo records that Plan 24-01 is the active recovery focus and that Phase 21 remains parked.

## Verification

- `bash -n scripts/phase24-db-preflight.sh` -> passed.
- `rg -n "catalogo 2026-07-01|users, tokens e logs permanecem vindo do banco live|Rollback" docs/ROUTER-DB-RECOVERY.md` -> matched all required phrases/section.
- `! rg -n "INSERT |UPDATE |DELETE |ALTER DATABASE|systemctl --user restart|pg_restore -d" scripts/phase24-db-preflight.sh` -> passed.
- `rg -n "Phase 24|Phase 21 remains parked" .planning/STATE.md` -> passed.
- `bash scripts/phase24-db-preflight.sh` -> passed and confirmed:
  - Graphify fresh at commit `c92c0f6`.
  - `bin/clianything status --strict` healthy.
  - current counts: `channels=5`, `models=14`, `abilities=18`, `tokens=8`.
  - host DB list includes `newapi` and does not include `DBRouterAiAtius`.
  - active router unit still targets `10.1.1.1:6432/newapi`.
  - `pg_restore -l` succeeded for `newapi-before.fix.dump` and `data/pg_backup/newapi_backup_20260531_235230.dump`.

## Commits

- `878024cf` — `docs(24-01): add db recovery contract runbook`
- `5cf57669` — `chore(24-01): add read-only db preflight script`

## Deviations from Plan

### Auto-fixed Issues

None.

### Execution Notes

- `.planning/STATE.md` already had substantial pre-existing uncommitted changes unrelated to this plan. To preserve other in-flight work, only the minimal Phase 24 note was merged into the working tree and the file was not included in a task commit.

## Known Stubs

None.

## Self-Check: PASSED

- Found `docs/ROUTER-DB-RECOVERY.md`.
- Found `scripts/phase24-db-preflight.sh`.
- Found `24-01-SUMMARY.md`.
- Verified commit `878024cf` exists in git history.
- Verified commit `5cf57669` exists in git history.
