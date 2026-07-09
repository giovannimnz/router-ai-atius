---
phase: 22-k3s-migration-preflight-and-cutover-plan-for-router-ai-atius
plan: "01"
type: summary
status: complete
completed_at: "2026-07-08T23:45:00-03:00"
---

# 22-01 Summary

## Deliverables

- `docs/K3S-MIGRATION.md`
- `scripts/k3s-router-preflight.sh`

## Evidence Collected

- Podman runtime healthy via `bin/clianything status`.
- Provider baseline captured via `bin/clianything providers --all`.
- k3s nodes observed `Ready`.
- `Metrics API not available` on `kubectl top nodes`.
- Storage baseline confirms only `local-path`, `RWO`, `Delete`, no expansion.
- `IngressClass`/Ingress absent.

## Outcome

The migration contract now exists and is explicit about:

- Podman rollback staying active.
- namespace `router-ai-atius`.
- Apache remaining the initial edge.
- `model-detailed` staying out of `/v1/`.
- cluster blockers and no-go gates before any public cutover.
