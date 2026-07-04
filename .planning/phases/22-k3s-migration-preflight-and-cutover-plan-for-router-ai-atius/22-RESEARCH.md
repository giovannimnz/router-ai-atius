# Phase 22: k3s migration preflight and cutover plan - Research

**Researched:** 2026-06-29
**Domain:** k3s migration of the live `router-ai-atius` Podman runtime
**Confidence:** HIGH for local repo/runtime/cluster evidence; MEDIUM for final production architecture until storage and cutover rehearsal pass.

## Summary

The migration is viable only as a staged cutover, not as a direct replacement. The current Podman runtime is healthy and is the known-good rollback path. The k3s cluster is present and has four ready nodes, but it currently shows operational blockers: `atius-srv-1` has `DiskPressure`, `atius-srv-2` has image filesystem pressure, and monitoring/Portainer workloads are pending, evicted, or crash-looping. There is also no IngressClass/Ingress installed, and storage is only `local-path` RWO with `Delete` reclaim policy.

The safest first phase is therefore:

1. Document a migration contract and preflight gate.
2. Add k3s manifests/templates and validation scripts without committing secrets.
3. Rehearse backup/restore and shadow deployment.
4. Cut over Apache only after smoke tests pass and rollback is proven.

## User And Fork Constraints

- Use PT-BR for user-facing planning and docs.
- Preserve the full-Go `/v1/` runtime. Do not reintroduce Python/model-detailed.
- Preserve Go-owned public `/v1/models` shape: root `{"data":[...]}` only, deterministic ordering, no internal pricing provenance.
- Preserve consolidated provider channels and Codex OAuth/shared credential behavior.
- Preserve local TEI embeddings through `embedding-gte-v1`, Go `service/embeddinggovernor/`, and `relay/embedding_handler.go`.
- Do not print or commit secrets.
- Do not disrupt Phase 21's clean PT-native upstream PR work in `feat/pt-native`.

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PHASE-22-K3S-PREFLIGHT | Migration cannot proceed until k3s node health, disk pressure, storage, routing edge, image architecture, and existing runtime state are inventoried. | Live cluster shows disk/image pressure and no ingress; preflight must catch this before cutover. |
| PHASE-22-RUNTIME-PARITY | k3s runtime must preserve Podman behavior for HTTP, `/v1/models`, providers, embeddings, logs, env, and CLI operations. | Current `bin/clianything status` and provider inventory are the baseline. |
| PHASE-22-STATEFUL-DATA | PostgreSQL/Redis/data/log migration requires backup, restore rehearsal, and a rollback path. | Current DB lives in Podman; k3s has only local-path RWO storage with Delete reclaim policy. |
| PHASE-22-CUTOVER-ROLLBACK | Public cutover must be manual, reversible, and validated by local + public smoke tests. | Apache currently owns the public edge; no k3s ingress exists. |

</phase_requirements>

## Current Runtime Findings

| Area | Evidence | Migration Meaning |
|---|---|---|
| Podman runtime | `container-router-ai-atius.service` active; pod `atius-ai-router` running `router-ai-atius`, `postgres`, `redis`, infra pause. | Keep Podman as rollback until final cutover passes. |
| Public edge | Current architecture routes Apache/Cloudflare to host backend on `127.0.0.1:3000`. | Cutover can initially be an Apache backend target change, not DNS/Cloudflare change. |
| Runtime env | User unit sets `SQL_DSN`, `REDIS_CONN_STRING`, `SESSION_SECRET`, `TRUST_PROXY`, logging, batch update, and governor envs. | Translate into ConfigMap/Secret/PVC. Never commit values. |
| Providers | Active providers: MiniMax, DeepSeek, OpenAI - Codex, Local TEI - GTE Embeddings. | Shadow deploy must use a copied/restored DB and preserve channel state. |
| Embeddings | `embedding-gte-v1` uses `Local TEI - GTE Embeddings` to `http://10.1.1.4:3000`. | Router pod must reach TEI directly or via `tei-gte.ai-search.svc.cluster.local`; any DB base URL change needs backup. |
| CLIAnything | Docs/CLI currently assume Podman exec for DB operations. | Add Kubernetes-compatible operational path or keep DB outside cluster until CLI is adapted. |

## k3s Findings

| Area | Evidence | Migration Meaning |
|---|---|---|
| Nodes | `atius-srv-1`, `atius-srv-2`, `atius-srv-3`, `horistic-srv` all Ready. | Cluster exists and can host a shadow deployment. |
| Disk pressure | `atius-srv-1` has `DiskPressure=True` and `NoSchedule` taint; recent events show evictions. | Do not schedule router DB or router pods there until disk is fixed. |
| Image pressure | Events show `atius-srv-2` image filesystem at 86% used. | Image cleanup/capacity is a preflight gate before pulling router images broadly. |
| Storage | Only `local-path` observed; RWO, `Delete`, no expansion. | Stateful DB in k3s is not HA and needs explicit backup/restore/retention guard. |
| Ingress | No IngressClass/Ingress resources observed. | Keep Apache edge for this phase unless a separate ingress phase is created. |
| TEI | `ai-search/tei-gte` exists; manifest pattern includes namespace, PVC, nodeSelector, `hostNetwork`, resource limits, and long probes. | Reuse the safe apply/rollout style, but do not mutate TEI in this phase. |

## Recommended Target Architecture For This Phase

```text
Cloudflare
  |
Apache on current edge host
  |
  |-- current rollback: 127.0.0.1:3000 -> Podman router-ai-atius
  |
  `-- cutover candidate after approval:
        k3s Service for router-ai-atius
          |
          +-- Deployment/router-ai-atius (1 replica initially)
          +-- Postgres/Redis strategy selected after backup rehearsal
          +-- Optional PVCs for /data and logs if the app still needs file persistence
          |
          `-- TEI dependency via ai-search/tei-gte or 10.1.1.4:3000
```

Initial replica count should be `1`. Multi-replica router is not a free win unless `SESSION_SECRET`, Redis, DB migrations, background tasks, and provider rate limits are verified for concurrent pods.

## Pre-Requisites

### Must Fix Or Explicitly Gate Before Production Cutover

- Clear `DiskPressure` on `atius-srv-1`, or enforce node selectors/affinity so no router production resource schedules there.
- Investigate image filesystem pressure on `atius-srv-2` before pulling/building router images across nodes.
- Decide DB placement:
  - keep Postgres on Podman/external for first k3s backend shadow, or
  - move Postgres to k3s StatefulSet with backup/restore and local-path limitations documented.
- Create a current `pg_dump` backup and prove restore into an isolated target.
- Decide Redis treatment: ephemeral reset or export/restore.
- Create Kubernetes Secrets from existing runtime values without writing secrets to git.
- Decide router-to-TEI endpoint: existing `http://10.1.1.4:3000` versus cluster DNS `http://tei-gte.ai-search.svc.cluster.local`.
- Add k3s-compatible operational commands for `bin/clianything` or document a supported bridge path while DB remains in transition.
- Validate image architecture on arm64 nodes and image pull access from k3s/containerd.
- Back up Apache vhost before any proxy target change.

### Should Do Before Production Cutover

- Add `docs/K3S-MIGRATION.md` with go/no-go, apply, smoke, cutover, and rollback steps.
- Add `scripts/k3s-router-preflight.sh` for non-secret cluster/runtime checks.
- Add `scripts/k3s-router-apply.sh` modeled after the TEI apply script.
- Add `scripts/k3s-router-smoke.sh` for local shadow smoke, public smoke, `/v1/models`, and embeddings dimension checks.
- Keep `scripts/podman-validate.sh` as rollback validation.
- Create a shadow Service that does not conflict with host `3000`.

## Validation Architecture

| Gate | Command / Evidence | Pass Criteria |
|---|---|---|
| Graphify | `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status` | `stale=false` and `commit_stale=false` before planning/execution; rebuild after artifacts change. |
| Podman baseline | `bin/clianything status` and `bin/clianything providers --all` | Backend, DB, `/v1/models` baseline OK; provider matrix captured. |
| k3s baseline | `sudo -n k3s kubectl get nodes -o wide`, `top nodes`, `get events` | No unhandled DiskPressure/image pressure for selected nodes. |
| Storage | `sudo -n k3s kubectl get storageclass,pv,pvc -A` | DB migration decision acknowledges `local-path` RWO/Delete/no expansion. |
| Manifest dry-run | `sudo -n k3s kubectl apply --dry-run=server -f k8s/router-ai-atius/` | All manifests pass server-side validation. |
| Shadow deploy | `sudo -n k3s kubectl -n router-ai-atius rollout status deploy/router-ai-atius` | Shadow pod Ready without touching public Apache. |
| Local API | `curl http://<shadow-endpoint>/api/status` and `/health` | HTTP 200 healthy backend. |
| Public shape | Authenticated `/v1/models` against shadow endpoint | Root payload has `data`; no internal pricing fields; ordering preserved. |
| Embeddings | `scripts/smoke-embeddings.py` with `embedding-gte-v1`, expected dim `768` | Smoke passes through k3s router path; queue/governor behavior intact. |
| Cutover | Apache configtest + public smokes | Public `https://router.atius.com.br/health` 200; unauth `/v1/models` 401 expected; auth `/v1/models` 200. |
| Rollback | Restore Apache target to Podman + `systemctl --user restart container-router-ai-atius.service` if needed | Podman path passes `bin/clianything status`. |

## Risks And Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| DB data loss during StatefulSet migration | High | `pg_dump` backup, restore rehearsal, no cutover until restore verified, preserve Podman DB until post-cutover soak passes. |
| Cluster storage not HA | High | State explicitly that `local-path` is single-node persistence; do not claim HA; use node affinity and backups. |
| DiskPressure causes eviction | High | Fix disk pressure or avoid tainted node; preflight blocks production cutover. |
| Secrets leak into git/planning | High | Commit only Secret templates and commands; never values. |
| `/v1/models` regression | High | Keep Go-owned code path and run focused tests/smokes. |
| TEI endpoint drift breaks embeddings | Medium | Validate both current IP and service DNS; back up channel table before base URL changes. |
| CLIAnything loses DB access after DB moves | Medium | Add k3s-aware operational mode or document supported port-forward/pod exec before retiring Podman DB. |
| Apache cutover fails | Medium | Back up vhost, `apache2ctl configtest`, keep Podman unit active, rollback target to `127.0.0.1:3000`. |

## Research Conclusion

Proceed with Phase 22 as a migration-preparation phase. Do not attempt immediate production cutover until the phase has produced:

- a written k3s migration runbook;
- preflight scripts;
- non-secret manifests/templates;
- backup/restore rehearsal;
- shadow deployment validation;
- cutover/rollback gates.

The current cluster can host a shadow deployment, but the observed disk/storage/ingress state means production migration needs explicit gating.
