# Phase 29 Context

## Objective

Finish the manual runtime work intentionally left out of Phase 22:

- create/apply real Kubernetes Secrets outside git
- run restore rehearsal against the k3s target
- apply the `router-ai-atius` shadow stack
- run shadow smoke
- record go/no-go

## Known Preconditions

- Phase 22 artifacts exist and passed static/dry-run validation.
- Backup already captured: `backups/k3s-router-20260709T024419Z`.
- Podman remains the production source of truth.
- Public edge still points to Podman-backed Apache.

## Known Blockers To Re-check

- `Metrics API not available`
- only `local-path` storageclass
- no `IngressClass`
- real secrets still not created in namespace `router-ai-atius`

## Success Definition

Either:

- shadow deployment is up and passes smoke with evidence, producing a real
  go/no-go for public cutover

or:

- the run records an explicit no-go with concrete blockers and rollback state

without touching public traffic.
