# Phase 22: k3s migration preflight and cutover plan - Context

**Gathered:** 2026-06-29
**Status:** Ready for planning
**Source:** `$gsd-plan-phase` request: "Analise os pre requisitos e afins ref. a passarmos o: router-ai-atius para o k3s"

<domain>
## Phase Boundary

This phase plans the migration of the live `router-ai-atius` runtime from the current Podman/user-systemd deployment into the existing local k3s cluster.

The first deliverable is not an immediate production cutover. The phase must produce a verified preflight, Kubernetes manifests or manifest templates, backup/restore rehearsal, shadow deployment validation, and a cutover/rollback runbook. Production traffic must remain on the current Podman path until the manual cutover checkpoint is explicitly approved after successful shadow validation.

The migration must preserve the current full-Go runtime contract:

- `GET /v1/models` remains Go-owned.
- Python/model-detailed remains out of the canonical `/v1/` path.
- `embedding-gte-v1` remains the only default public governed local embedding.
- The Go `service/embeddinggovernor/` and `relay/embedding_handler.go` remain the local TEI backpressure owner.
- Existing provider routing/customization and public payload guards remain protected fork behavior.

</domain>

<decisions>
## Implementation Decisions

### Runtime Strategy
- **D-01:** Treat Podman as the rollback source of truth until k3s shadow validation and cutover smoke tests pass. Do not stop or remove the Podman unit during preflight or manifest creation.
- **D-02:** Use a dedicated Kubernetes namespace named `router-ai-atius` for router stack resources. Do not colocate router resources into `ai-search`, which currently owns TEI.
- **D-03:** Keep Apache/Cloudflare as the initial public edge for this phase. There is no current IngressClass/Ingress in the cluster, so cutover should retarget Apache to a validated k3s Service/NodePort/ClusterIP path rather than introducing an ingress controller in the same phase.
- **D-04:** Use shadow deployment first on a non-public port/path. Public `https://router.atius.com.br` must not move until local and public smoke gates pass.

### Stateful Data
- **D-05:** Production database migration is high-risk because the cluster currently has only `local-path` RWO storage with `Delete` reclaim policy and no volume expansion. The plan must require a `pg_dump`/restore rehearsal and a rollback-ready backup before any DB cutover.
- **D-06:** Redis state can be recreated only if the router's current production semantics allow it. If Redis contains required runtime state at cutover time, include an explicit export/restore step; otherwise document it as ephemeral.
- **D-07:** Real DB passwords, Redis passwords, session secret, provider keys, OAuth credentials, and router tokens must become Kubernetes Secrets at apply time only. Do not commit secret values.

### Routing And Providers
- **D-08:** Preserve the current provider catalog and channel table before cutover with `bin/clianything` backups. Any channel base URL change, such as moving TEI from `http://10.1.1.4:3000` to `http://tei-gte.ai-search.svc.cluster.local`, requires backup, smoke, and rollback notes.
- **D-09:** Keep `EMBEDDING_GOVERNOR_*` explicit in k3s config so defaults cannot drift. Automatic daily concurrency remains `min=1`, `initial=2`, `max=3`; `4` remains manual/turbo only.
- **D-10:** Do not expose `pricing_source`, `pricing_estimated`, `pricing_version`, pagination fields, or non-`data` top-level fields from public `/v1/models`.

### Cluster Health Gates
- **D-11:** Current cluster health is not clean enough for blind cutover: `atius-srv-1` has `DiskPressure`, `atius-srv-2` has image filesystem pressure, and monitoring/Portainer pods are pending/evicted/crashing. These must be handled as preflight blockers or explicitly scoped away by node selection and rollback policy.
- **D-12:** The TEI deployment in `ai-search` is a dependency, not part of this router migration. Do not change TEI resources except to read status and validate router-to-TEI connectivity.

### the agent's Discretion
The executor may choose plain multi-document YAML, Kustomize-free manifests, or a small script wrapper if it keeps validation simple and does not add Helm or cluster-wide controllers. If a simpler first step is to run only the Go backend in k3s while keeping Postgres/Redis on Podman for shadow validation, that can be used as a rehearsal path, but the production cutover runbook must explicitly state whether the final target moves DB/Redis too.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Contracts
- `AGENTS.md` - fork guardrails, Go-owned `/v1/models`, protected channel/provider behavior, and secret handling.
- `.planning/STATE.md` - current runtime shape and Podman source of truth.
- `.planning/ROADMAP.md` - Phase 22 scope and plan list.
- `.planning/REQUIREMENTS.md` - Phase 20 runtime contracts that remain active migration constraints.

### Current Runtime
- `docs/PODMAN.md` - current Podman runbook, lifecycle, and validation gates.
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - operational manual for provider routing, embeddings, governor env, smokes, and recovery.
- `podman-compose.yml` - current dev stack service shape.
- `scripts/podman-validate.sh` - current Podman config gate.
- `makefile` - dev command conventions and Podman compose usage.

### Kubernetes Reference In This Environment
- `/home/ubuntu/GitHub/embeddings/k8s/tei-gte.yaml` - existing working k3s manifest style, namespace/PVC/probe/resource pattern.
- `/home/ubuntu/GitHub/embeddings/scripts/apply-tei.sh` - safe apply pattern with pre-apply backups and rollout status.

### Protected Runtime Code
- `controller/model.go` and `controller/model_list_test.go` - public `/v1/models` payload and ordering contract.
- `service/modelcatalog/` - Go model catalog construction.
- `relay/embedding_handler.go` and `service/embeddinggovernor/` - local TEI/governor path.
- `relay/channel/codex/`, `service/codex_*.go`, `relay/common/relay_utils.go` - provider routing/customization guards.
- `.dockerignore` - build-context guard for `/backups`, `/data`, `/logs`, `/runtime`.

</canonical_refs>

<code_context>
## Existing Code And Runtime Insights

### Live Runtime Snapshot
- Podman pod `atius-ai-router` is running with `router-ai-atius`, `postgres`, `redis`, and infra pause.
- User unit `container-router-ai-atius.service` owns production backend lifecycle.
- Backend is bound to host port `3000` and Apache routes public traffic to `127.0.0.1:3000`.
- Runtime bind paths are `data/` and `logs/` in this checkout.
- Active providers observed through `bin/clianything providers --all`: `MiniMax`, `DeepSeek`, `OpenAI - Codex`, and `Local TEI - GTE Embeddings`.
- `Local TEI - GTE Embeddings` currently points at `http://10.1.1.4:3000` and exposes `embedding-gte-v1`.

### k3s Snapshot
- Nodes: `atius-srv-1`, `atius-srv-2`, `atius-srv-3`, and `horistic-srv`, all `Ready`.
- `atius-srv-1` has `node.kubernetes.io/disk-pressure=:NoSchedule`.
- `atius-srv-2` reported image filesystem pressure at 86% used.
- `ai-search/tei-gte` exists and is currently the TEI dependency.
- `storageclass/local-path` is the only observed StorageClass. It is RWO, `Delete`, and does not allow volume expansion.
- No `IngressClass` or `Ingress` resources were observed.

### Migration-Relevant Gaps
- `bin/clianything` and docs assume Podman/Postgres container access for some operational paths. k3s migration must add or document a Kubernetes-compatible operational path before Podman is retired.
- Current Podman user unit has runtime env and bind mounts that must be translated to Kubernetes ConfigMap/Secret/PVC without committing secrets.
- Cluster storage/capacity issues must be resolved or explicitly avoided before stateful production cutover.

</code_context>

<specifics>
## Specific Ideas

- Prefer a no-controller, plain-manifest first implementation.
- Use an apply script modeled after `/home/ubuntu/GitHub/embeddings/scripts/apply-tei.sh`: capture current resource YAML, apply, wait rollout, print status.
- Use a dedicated `scripts/k3s-router-preflight.sh` to gather non-secret proof: Graphify status, Podman status, k3s nodes/taints, storage, ingress, current router health, and provider inventory.
- Add a `docs/K3S-MIGRATION.md` runbook with "go/no-go" gates before cutover.
- Treat actual cutover as a manual checkpoint, not an automatic execution task.

</specifics>

<deferred>
## Deferred Ideas

- Installing Traefik/Ingress, MetalLB, Longhorn, or another cluster-wide component is deferred unless preflight proves Apache/Service routing cannot meet the goal.
- Multi-replica router HA is deferred until DB/session/cache behavior is proven with the current single-replica migration.
- Moving TEI from `ai-search` or changing its CPU/memory envelope is out of scope.

</deferred>

---

*Phase: 22-k3s-migration-preflight-and-cutover-plan-for-router-ai-atius*
*Context gathered: 2026-06-29*
