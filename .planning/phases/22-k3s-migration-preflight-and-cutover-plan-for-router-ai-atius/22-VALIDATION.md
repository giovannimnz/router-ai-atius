---
phase: 22
phase_slug: k3s-migration-preflight-and-cutover-plan-for-router-ai-atius
created: 2026-06-29
status: planned
---

# Phase 22 Validation Strategy

## Validation Architecture

Phase 22 is infrastructure-sensitive. Validation must prove three different surfaces:

1. Current Podman production remains healthy and rollback-ready.
2. k3s manifests and shadow runtime work without touching public traffic.
3. Public cutover is reversible and preserves API/provider/embedding contracts.

## Required Gates

| Gate | Type | Evidence |
|---|---|---|
| Baseline | preflight | `bin/clianything status`, `bin/clianything providers --all`, `podman ps --filter pod=atius-ai-router` |
| Cluster | preflight | `sudo -n k3s kubectl get nodes -o wide`, `top nodes`, `get events`, `get storageclass,pv,pvc -A`, `get ingressclass,ingress -A` |
| Manifest | static/server | `sudo -n k3s kubectl apply --dry-run=server -f k8s/router-ai-atius/` |
| Backup | data | `pg_dump` artifact path recorded, restore rehearsal exits 0, rollback path documented |
| Shadow | runtime | `kubectl rollout status`, local `/api/status`, `/health`, `/v1/models` shape, embeddings dim 768 |
| Cutover | manual | Apache config backup, `apache2ctl configtest`, public smokes, rollback smokes |

## Blockers

- No production cutover while selected nodes have unhandled `DiskPressure`.
- No production cutover without a fresh DB backup and restore rehearsal.
- No production cutover if k3s path cannot pass `GET /v1/models` payload-shape checks.
- No production cutover if `embedding-gte-v1` smoke through k3s path does not return dimension `768`.
- No production cutover if rollback to Podman has not been rehearsed or documented.

## Secret Hygiene

Secrets are validated by presence and Kubernetes object references only. Do not write values to docs, manifests, logs, planning files, or final summaries.
