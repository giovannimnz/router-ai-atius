---
phase: 22-k3s-migration-preflight-and-cutover-plan-for-router-ai-atius
plan: "04"
type: summary
status: complete
completed_at: "2026-07-08T23:48:00-03:00"
---

# 22-04 Summary

## Deliverables

- `scripts/k3s-router-cutover-checklist.sh`
- `scripts/k3s-router-rollback-check.sh`
- `docs/K3S-MIGRATION.md`
- `docs/PODMAN.md`
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`

## Decision

Cutover was deferred.

## Why It Was Deferred

- `Metrics API` unavailable in cluster preflight.
- storage remains `local-path` only, `RWO`, reclaim `Delete`, no expansion.
- no `IngressClass` exists.
- no shadow deployment evidence was recorded yet.
- no public Apache retarget was executed in this run.

## Rollback State

- Podman remains active.
- Public traffic remains on the current Podman-backed Apache path.
- Rollback procedure is documented and validated syntactically via
  `scripts/k3s-router-rollback-check.sh`.

## Outcome

Phase 22 finished as a complete migration-preparation package:
preflight, manifests, backup path, shadow path, cutover checklist, and rollback
check all exist. Public migration remains intentionally gated behind future
shadow evidence and explicit operator approval.
