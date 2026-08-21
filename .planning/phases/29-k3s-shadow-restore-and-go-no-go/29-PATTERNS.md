# Phase 29/30: k3s Shadow, Restore, Cutover, and Podman Retirement - Pattern Map

**Mapped:** 2026-07-12
**Files analyzed:** 15 proposed new/modified files
**Analogs found:** 15 / 15

## Scope and source hierarchy

- Phase 29 owns the real restore rehearsal, shadow deployment, shadow smoke, and formal go/no-go without changing public traffic (`29-CONTEXT.md:5-11,27-38`).
- Phase 30 owns Apache retarget, public smoke, rollback/soak, and only then Podman retirement (`30-CONTEXT.md:5-21,23-33`).
- Neither Phase 29 nor Phase 30 currently has a `RESEARCH.md`; use Phase 22's completed preparation package as the technical baseline, especially `22-RESEARCH.md:84-109` and `docs/K3S-MIGRATION.md`.
- The public `/v1/` path must remain Go-only; do not restore `model-detailed`. Preserve the public `/v1/models` root shape and embedding dimension gates from `scripts/k3s-router-smoke.sh:41-59`.
- Secrets remain apply-time inputs and must never appear in this file, manifests, evidence, or logs.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `k8s/router-ai-atius/router.yaml` | config | request-response + file-I/O | same file; `/home/ubuntu/GitHub/embeddings/k8s/tei-gte.yaml` | exact self-extension |
| `k8s/router-ai-atius/postgres.yaml` | config | CRUD + file-I/O | same file; `k8s/router-ai-atius/router.yaml` | exact self-extension |
| `k8s/router-ai-atius/redis.yaml` | config | request-response | same file; `/home/ubuntu/GitHub/embeddings/k8s/tei-gte.yaml` | exact self-extension |
| `scripts/k3s-router-preflight.sh` | utility/gate | batch + request-response | same file; `scripts/k3s-router-validate-manifests.sh` | exact self-extension |
| `scripts/k3s-router-validate-manifests.sh` | utility/gate | batch + transform | same file | exact self-extension |
| `scripts/k3s-router-restore-rehearsal.sh` (new) | utility | file-I/O + CRUD + batch | `scripts/k3s-router-backup.sh` + `scripts/k3s-router-apply-shadow.sh` | composite exact |
| `scripts/k3s-router-apply-shadow.sh` | utility | batch + request-response | same file; `/home/ubuntu/GitHub/embeddings/scripts/apply-tei.sh` | exact self-extension |
| `scripts/k3s-router-smoke.sh` | test/utility | request-response | same file | exact |
| `scripts/k3s-router-go-no-go.sh` (new) | utility/gate | batch + transform | `scripts/k3s-router-cutover-checklist.sh` | role-match |
| `scripts/k3s-router-cutover-checklist.sh` | utility/gate | batch + request-response | same file | exact self-extension |
| `scripts/k3s-router-rollback-check.sh` | utility/gate | request-response | same file; `scripts/podman-admin.sh` | exact self-extension |
| `scripts/k3s-router-podman-retire.sh` (new) | utility | event-driven + batch | `scripts/podman-admin.sh:631-642` + `scripts/podman-down.sh` | role-match; add stronger guards |
| `docs/K3S-MIGRATION.md` | config/runbook | batch | same file | exact |
| `docs/PODMAN.md` | config/runbook | event-driven | same file | exact |
| Phase 29/30 execution evidence (`29-*-SUMMARY.md`, `30-*-SUMMARY.md`) | test/evidence | batch + transform | `22-03-SUMMARY.md`, `22-04-SUMMARY.md` | exact |

## Pattern Assignments

### `k8s/router-ai-atius/router.yaml` (config, request-response + file-I/O)

**Responsibilities:** pin the router to the dedicated srv1 label with required node affinity, bind its `local-path` PVC intentionally, and expose a host-reachable shadow endpoint for Apache without introducing Ingress.

**Current labels/PVC pattern** (`router.yaml:1-16`):

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: router-ai-atius-data
  namespace: router-ai-atius
  labels:
    app.kubernetes.io/name: router-ai-atius
    app.kubernetes.io/part-of: atius-router
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

Copy this structure, but add explicit `storageClassName: local-path`; implicit default selection is not sufficient for a planned single-node binding. Do not add a hand-written PV or `hostPath`: the existing local-path provisioner is the selected storage mechanism.

**Closest scheduling analog** (`/home/ubuntu/GitHub/embeddings/k8s/tei-gte.yaml:62-65`):

```yaml
spec:
  nodeSelector:
    kubernetes.io/hostname: horistic-srv
```

For this phase, use the same placement location under `template.spec`, but implement the locked requirement as `affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution` against a dedicated label (recommended contract: `atius.com.br/router-ai-atius-node: "true"`), not merely hostname. The cluster label itself is an operator precondition applied and verified by the preflight script; it is not declared by a workload manifest.

**Required affinity syntax analog** (`/home/ubuntu/GitHub/omni-srv-admin/modules/k3s-ha-portainer-oci/monitoring/loki/values.yaml:91-99`):

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: NotIn
              values:
                - horistic-srv
```

Copy the structure, changing the expression to `key: atius.com.br/router-ai-atius-node`, `operator: In`, `values: ["true"]`. Apply the identical affinity block to router, Postgres, and Redis so the stateful stack cannot split across nodes.

**Current Service pattern** (`router.yaml:100-117`):

```yaml
apiVersion: v1
kind: Service
metadata:
  name: router-ai-atius
  namespace: router-ai-atius
  annotations:
    router.atius.com.br/public-edge: "apache-shadow-only"
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: router-ai-atius
  ports:
    - name: http
      port: 3000
      targetPort: http
```

`ClusterIP` is not directly reachable by host Apache. Extend this Service to an explicit, reserved `NodePort` and keep it shadow-only until Phase 30. Do not use `hostNetwork`, `hostPort`, Ingress, or port `3000`, which belongs to Podman rollback. The exact NodePort must be selected after checking cluster allocations and then recorded in the runbook and Apache backup evidence.

### `k8s/router-ai-atius/postgres.yaml` (config, CRUD + file-I/O)

**Current stateful pattern** (`postgres.yaml:17-35,42-53,103-106`):

```yaml
kind: PersistentVolumeClaim
metadata:
  name: router-ai-atius-postgres-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
---
kind: StatefulSet
spec:
  serviceName: router-ai-atius-postgres
  replicas: 1
  template:
    spec:
      volumes:
        - name: postgres-data
          persistentVolumeClaim:
            claimName: router-ai-atius-postgres-data
```

Add `storageClassName: local-path` to the PVC and the same required dedicated-label affinity under `StatefulSet.spec.template.spec`. Preserve `replicas: 1`, RWO, probes, and the existing Secret references (`postgres.yaml:61-95`). The restore rehearsal must inspect the bound PV's `nodeAffinity` and prove it matches the selected srv1 node before importing data.

### `k8s/router-ai-atius/redis.yaml` (config, request-response)

**Current ephemeral shadow pattern** (`redis.yaml:35-49,69-81`):

```yaml
spec:
  containers:
    - name: redis
      env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: router-ai-atius-secrets
              key: REDIS_PASSWORD
      resources:
        requests:
          cpu: 250m
          memory: 256Mi
        limits:
          cpu: 250m
          memory: 512Mi
  volumes:
    - name: redis-tmp
      emptyDir: {}
```

Only add the shared required affinity block. Preserve ephemeral Redis for shadow unless Phase 29 evidence proves production semantics require export/restore; that decision is explicitly deferred by `k8s/router-ai-atius/README.md:50-55` and must be resolved before Phase 30 go.

### `scripts/k3s-router-preflight.sh` (utility/gate, batch + request-response)

**Shell convention** (`k3s-router-preflight.sh:1-17`):

```bash
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

run_or_warn() {
  local desc="$1"
  shift
  if ! "$@"; then
    printf 'WARN: %s failed\n' "$desc" >&2
    return 0
  fi
}
```

Retain `set -euo pipefail`, relative repo-root resolution, and sectioned evidence. Do **not** use `run_or_warn` for node health, label, taint, storage, PVC, image filesystem, or Service-port gates.

**Current weak gate** (`k3s-router-preflight.sh:34-47`):

```bash
sudo -n k3s kubectl get nodes -o wide
run_or_warn "metrics API unavailable" sudo -n k3s kubectl top nodes
sudo -n k3s kubectl get storageclass,pv,pvc -A -o wide
sudo -n k3s kubectl get events -A --sort-by=.lastTimestamp | tail -120 || true
```

Replace observation-only behavior with a fail-closed gate that:

1. requires exactly one node carrying the dedicated label and requires its hostname to be `atius-srv-1`;
2. reads that node's `.status.conditions` and exits non-zero unless `Ready=True` and `DiskPressure=False`;
3. rejects a `node.kubernetes.io/disk-pressure` taint even if the condition output looks stale;
4. verifies `local-path` exists and records provisioner, binding mode, expansion, and reclaim policy;
5. verifies the intended NodePort is free;
6. treats missing/unparseable required evidence as no-go, not success.

`Metrics API not available` can remain a warning only if direct node conditions, taints, filesystem checks, and events provide complete gate evidence. Never add a DiskPressure toleration: that would bypass the fail-closed policy.

### `scripts/k3s-router-validate-manifests.sh` (utility/gate, batch + transform)

**Analog** (`k3s-router-validate-manifests.sh:14-25`):

```bash
sudo -n k3s kubectl apply --dry-run=server -f "$manifest_dir/namespace.yaml"
for file in "$manifest_dir"/*.yaml; do
  [ "$(basename "$file")" = "namespace.yaml" ] && continue
  sed '/^[[:space:]]*namespace: router-ai-atius$/d' "$file" | \
    sudo -n k3s kubectl apply --dry-run=server -n default -f -
done
```

Keep server-side dry-run, then add static assertions for all three workload templates: dedicated-label `requiredDuringSchedulingIgnoredDuringExecution`; both PVCs explicitly `local-path`; router Service has the selected non-conflicting exposure; no `Ingress`, `hostNetwork`, `hostPort`, DiskPressure toleration, or secret literal is introduced.

### `scripts/k3s-router-restore-rehearsal.sh` (new utility, file-I/O + CRUD + batch)

**Backup input analog** (`k3s-router-backup.sh:6-13,15-19,33-35`):

```bash
ts="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="backups/k3s-router-${ts}"
mkdir -p "$meta_dir" "$db_dir"
bin/clianything status > "${meta_dir}/clianything-status.txt"
podman ps --filter pod=atius-ai-router > "${meta_dir}/podman-ps.txt"
podman exec postgres pg_dump -U admin DBRouterAiAtius > "${db_dir}/DBRouterAiAtius.sql"
```

**Apply/rollout analog** (`k3s-router-apply-shadow.sh:13-24`):

```bash
ts="$(date -u +%Y%m%dT%H%M%SZ)"
backup_dir="backups/k3s-router-shadow-${ts}"
mkdir -p "$backup_dir"
sudo -n k3s kubectl -n router-ai-atius get all,pvc,secret,configmap -o yaml > "${backup_dir}/resources.yaml" 2>/dev/null || true
sudo -n k3s kubectl -n router-ai-atius rollout status statefulset/router-ai-atius-postgres --timeout=30m
sudo -n k3s kubectl -n router-ai-atius get pods,svc,pvc -o wide
```

The new script should require explicit opt-in and an existing backup directory, validate that the SQL dump is non-empty, resolve the Postgres pod, stream the dump through `psql -v ON_ERROR_STOP=1`, and write only sanitized evidence to a timestamped directory. Verify restored DB identity/count/invariants through a k3s-compatible command, not `bin/clianything`'s Podman default path. Never echo DSNs/passwords or copy Secret YAML into evidence. Re-running must either target a freshly recreated rehearsal database/PVC or fail safely; it must not merge blindly into an unknown DB.

### `scripts/k3s-router-apply-shadow.sh` (utility, batch + request-response)

**Safe apply analog** (`/home/ubuntu/GitHub/embeddings/scripts/apply-tei.sh:4-19`):

```bash
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TS="$(date -u +%Y%m%dT%H%M%SZ)"
BACKUP_DIR="$ROOT/backups/k8s-${TS}"
mkdir -p "$BACKUP_DIR"
sudo -n kubectl get ns ai-search -o yaml > "$BACKUP_DIR/ns-ai-search.yaml" 2>/dev/null || true
sudo -n kubectl apply -f "$MANIFEST"
sudo -n kubectl -n ai-search rollout status deployment/tei-gte --timeout=30m
sudo -n kubectl -n ai-search get deploy,svc,pvc,pod -o wide
```

Keep the existing opt-in at `k3s-router-apply-shadow.sh:6-9`, but call the new fail-closed preflight before validation/apply. Require the Secret to exist with the expected key names without reading values. After rollout, assert every pod's scheduled node is the dedicated srv1 node, both PVCs are `Bound`, and the Service endpoint/NodePort resolves. Apply remains shadow-only and must not touch Apache or Podman.

### `scripts/k3s-router-smoke.sh` (test/utility, request-response)

**HTTP and contract pattern** (`k3s-router-smoke.sh:13-25,33-59`):

```bash
health_status="$(curl -sS -o /tmp/k3s-router-health.out -w '%{http_code}' "${base_url}/api/status" || true)"
if [ "$health_status" != "200" ]; then
  exit 1
fi
unauth_status="$(curl -sS -o /tmp/k3s-router-models-unauth.out -w '%{http_code}' "${base_url}/v1/models" || true)"
if [ "$unauth_status" != "401" ]; then
  exit 1
fi
# authenticated payload must have only {data}, no pricing provenance
ATIUS_ROUTER_EXPECTED_DIMENSION=768 python3 scripts/smoke-embeddings.py
```

Preserve these API gates. For a Phase 29 go decision, missing `ATIUS_ROUTER_TOKEN` must be a failure, not the current successful skip at lines 28-31. Add a mode/flag only if the same script must retain a weaker diagnostic mode. Capture status codes and assertions without response payloads or tokens.

### `scripts/k3s-router-go-no-go.sh` (new utility/gate, batch + transform)

**Required-input analog** (`k3s-router-cutover-checklist.sh:6-18`):

```bash
required=(CURRENT_PUBLIC_URL K3S_ROUTER_BASE_URL K3S_BACKUP_DIR APACHE_VHOST_BACKUP_PATH)
for key in "${required[@]}"; do
  if [ -z "${!key:-}" ]; then
    echo "Missing required env: ${key}" >&2
    exit 1
  fi
done
```

Use the same explicit-input style for preflight evidence, restore evidence, rollout evidence, smoke evidence, PVC/node placement, Apache backup path, and rollback check. The script should generate a sanitized decision artifact with `decision: go|no-go`, failed gates, timestamps, commit/image identity, and Podman/public-edge state. Default to `no-go`; only all required gates passing may emit `go`. It must never edit Apache or stop Podman.

### `scripts/k3s-router-cutover-checklist.sh` (utility/gate, batch + request-response)

**Manual checkpoint pattern** (`k3s-router-cutover-checklist.sh:20-39`):

```bash
echo "1. Validate k3s target before any Apache edit:"
echo "   ATIUS_ROUTER_TOKEN=<token> K3S_ROUTER_BASE_URL=${K3S_ROUTER_BASE_URL} scripts/k3s-router-smoke.sh"
echo "2. Validate Apache syntax before reload:"
echo "   apache2ctl configtest"
echo "MANUAL CHECKPOINT: approve editing Apache only after the shadow smoke passes."
```

Extend, do not automate away, this checkpoint. Require the Phase 29 decision artifact to say `go`; verify the NodePort shadow target locally before Apache edit; require a timestamped vhost backup and checksum; print the exact old Podman target and new k3s target; run/require `apache2ctl configtest`; then run the same strict smoke against the public URL. Keep rollback immediate on any failure and keep Podman active throughout soak.

**Apache route contract** (`docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:49-71`):

```text
/api/ and /       -> 127.0.0.1:3000
/v1/              -> 127.0.0.1:3000 (Go backend)
/health            -> 127.0.0.1:3000/api/status
```

Retarget all Go-owned routes coherently to the selected k3s endpoint. Do not change docs routes or introduce `model-detailed`. The actual vhost is host-owned (`/etc/apache2/sites-available/router.atius.com.br-le-ssl.conf`) and is not versioned; `docs/PHASE-7-AUDIT.md:105-110` is the closest mutation/validation precedent.

### `scripts/k3s-router-rollback-check.sh` (utility/gate, request-response)

**Current non-mutating readiness pattern** (`k3s-router-rollback-check.sh:6-18`):

```bash
systemctl --user status container-router-ai-atius.service --no-pager
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
bin/clianything status
echo "  apache2ctl configtest"
echo "  systemctl reload apache2"
```

Preserve the read-only default. Add strict local Podman smoke on `127.0.0.1:3000`, validate the saved Apache target/checksum, and support an explicit rollback mode only if separately guarded. The rollback sequence remains: restore vhost backup, configtest, reload, verify Podman, strict public smoke (`docs/K3S-MIGRATION.md:180-196`).

### `scripts/k3s-router-podman-retire.sh` (new utility, event-driven + batch)

**Lifecycle analog** (`scripts/podman-admin.sh:631-642`):

```bash
cmd_prod_restart() {
  systemctl --user daemon-reload
  systemctl --user restart "${UNIT_NAME}"
  systemctl --user status "${UNIT_NAME}" --no-pager --lines=0
  cmd_inspect_limits
  run_profiled "$CLIANYTHING" status
}
```

**Destructive precedent** (`scripts/podman-down.sh:8-23`):

```bash
set -euo pipefail
VOLUMES=""
podman-compose -f podman-compose.yml down $VOLUMES
```

Do not copy `podman-down.sh`'s permissive behavior. Retirement needs a stronger state machine and two distinct stages:

1. **rollback holdback during soak:** unit remains enabled/available; no stop, disable, container removal, volume removal, prune, or backup deletion;
2. **retirement after accepted soak:** require explicit opt-in, Phase 29 go, Phase 30 public-smoke evidence, completed soak duration, operator approval, current backup path/checksum, and a final rollback rehearsal. Then stop/disable the router unit without deleting images, containers, volumes, env, quadlets, or backups. Physical cleanup is a later separately approved action.

Never invoke `podman system prune`, `podman volume rm`, `scripts/podman-down.sh --volumes`, or delete `data/`, logs, backups, user units, or secrets. `docs/PODMAN.md:110-115` records the prune data-loss precedent.

### `docs/K3S-MIGRATION.md` and `docs/PODMAN.md` (runbooks, batch/event-driven)

Use `docs/K3S-MIGRATION.md:37-61` for preflight interpretation, `96-155` for backup/restore/shadow, `157-196` for cutover/rollback, and `198-218` for go/no-go. Update concrete commands and evidence paths to match the implemented scripts; remove any statement that treats missing Metrics API alone as adequate evidence.

Use `docs/PODMAN.md:5-11` for source-of-truth wording and `117-135` for runtime verification. During Phase 29 and Phase 30 soak it must continue to say Podman is the rollback source. Change it to retired only after the retirement gate succeeds, while retaining recovery commands and preserved artifacts.

### Phase 29/30 summaries (test/evidence, batch + transform)

**Deferred-action evidence analog** (`22-03-SUMMARY.md:18-41`):

```markdown
## Executed Evidence
- Backup executed: `backups/k3s-router-...`

## Deferred Runtime Action
- Shadow apply was not executed.

## Outcome
Production cutover remains blocked until restore evidence and shadow smoke are recorded.
```

**Decision/rollback analog** (`22-04-SUMMARY.md:19-42`):

```markdown
## Decision
Cutover was deferred.

## Rollback State
- Podman remains active.
- Public traffic remains on the current Podman-backed Apache path.
```

Copy this evidence-first structure. Phase 29 must record each gate and `go`/`no-go`; Phase 30 must record old/new Apache targets (without secrets), config checksum, public smoke, soak start/end, rollback status, and whether Podman is holdback or retired. A no-go/rollback is a valid completed outcome when evidence is complete.

## Shared Patterns

### Fail-closed shell gates

**Source:** `scripts/k3s-router-apply-shadow.sh:1-11`, `scripts/k3s-router-cutover-checklist.sh:6-18`

Apply to all mutating scripts:

```bash
#!/usr/bin/env bash
set -euo pipefail

if [ "${EXPLICIT_OPT_IN:-}" != "1" ]; then
  echo "explicit opt-in required" >&2
  exit 1
fi
```

Unknown, missing, empty, or unparseable evidence is failure. Diagnostic warnings are allowed only for non-required observability.

### Backup before mutation

**Source:** `scripts/k3s-router-apply-shadow.sh:13-20`; `/home/ubuntu/GitHub/embeddings/scripts/apply-tei.sh:6-17`

Create timestamped backups before Kubernetes apply or Apache edit, print their paths, and preserve them through soak. Do not put Secret bodies in those backups.

### Runtime parity smoke

**Source:** `scripts/k3s-router-smoke.sh:13-59`

Apply to shadow, public cutover, rollback, and soak checkpoints: health 200; unauthenticated models 401; authenticated models 200 with only root `data`; forbidden pricing fields absent; `embedding-gte-v1` dimension 768.

### Stateful placement invariant

The dedicated label, required affinity, PVC `storageClassName: local-path`, bound PV node, and actual pod node must all agree. Validate the invariant before restore, after rollout, and before cutover. Do not solve `DiskPressure` with a toleration.

### Separation of phases

```text
Phase 29: preflight -> restore -> shadow -> strict smoke -> go/no-go
Phase 30: require go -> Apache backup/retarget -> public smoke -> soak
          -> rollback OR accepted permanence -> guarded Podman retirement
```

No Phase 29 file edits Apache. No cutover step retires Podman. No retirement step deletes rollback data.

## No Analog Found

| File/Concern | Role | Data Flow | Reason / Planner Direction |
|---|---|---|---|
| Dedicated custom srv1 node label convention | config | event-driven | Repo has hostname selectors but no custom ownership label. Use `atius.com.br/router-ai-atius-node=true`, document it, and verify exactly one matching node. |
| Exact Apache-to-k3s Service transport | config | request-response | Current repo only documents Podman `127.0.0.1:3000`; no versioned Apache vhost or NodePort example exists. Use explicit NodePort after collision check; do not invent Ingress. |
| Podman retirement after k3s soak | utility | event-driven | Existing scripts start/restart/down but have no soak-evidence state machine. Build a guarded stop/disable-only script; defer deletion to a later approved cleanup. |

## Planner guardrails

- Do not schedule a live label/apply/restore/Apache/systemd action as an automatic verification command; those are explicit operator checkpoints.
- Do not claim `local-path` is HA or portable. Its node binding is part of the restore/cutover evidence.
- Do not use a DiskPressure toleration or preferred affinity; placement is required and health is fail-closed.
- Do not expose the k3s Service on host port `3000`; Podman must remain independently reachable for rollback.
- Do not store tokens, Secret values, DSNs, response payloads, or Secret YAML in summaries/evidence.
- Preserve the full-Go public route, Go-owned model catalog, consolidated provider behavior, and governed `embedding-gte-v1` contract.

## Metadata

**Analog search scope:** `k8s/router-ai-atius/`, `scripts/k3s-router-*`, Podman scripts/units documentation, `docs/K3S-MIGRATION.md`, `docs/PODMAN.md`, Phase 22/29/30 planning artifacts, `/home/ubuntu/GitHub/embeddings`, targeted k3s patterns in `omni-srv-admin`.

**Files scanned:** 40+ focused files; 18 files read in full or targeted non-overlapping ranges.

**Pattern extraction date:** 2026-07-12
