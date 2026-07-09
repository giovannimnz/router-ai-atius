---
phase: 22-k3s-migration-preflight-and-cutover-plan-for-router-ai-atius
plan: "03"
type: summary
status: complete
completed_at: "2026-07-08T23:47:00-03:00"
---

# 22-03 Summary

## Deliverables

- `scripts/k3s-router-backup.sh`
- `scripts/k3s-router-apply-shadow.sh`
- `scripts/k3s-router-smoke.sh`
- `docs/K3S-MIGRATION.md` updated with backup and restore rehearsal gates

## Executed Evidence

- Backup executed:
  - `backups/k3s-router-20260709T024419Z`
- Backup contents captured:
  - sanitized runtime metadata
  - `clianything` status/providers snapshots
  - `DBRouterAiAtius.sql`

## Deferred Runtime Action

- Shadow apply was not executed.

Reason:

- cluster still has operational blockers captured in preflight
- secret material was intentionally kept out of git
- `RUN_K3S_ROUTER_APPLY=1` remains the deliberate opt-in guard

## Outcome

The backup/rehearsal path is now implemented and documented.
Production cutover remains blocked until restore evidence and shadow smoke are
recorded.
