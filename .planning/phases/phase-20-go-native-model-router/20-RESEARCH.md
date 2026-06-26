# Phase 20: Go-Native Model Router Follow-up - Research

**Researched:** 2026-06-26 [VERIFIED: system date]
**Domain:** Go-native embedding governor for local TEI embeddings [VERIFIED: AGENTS.md + service/embeddinggovernor]
**Confidence:** HIGH for local code and operator state; MEDIUM for external TEI documentation because the GSD classifier returned MEDIUM for Context7. [VERIFIED: codebase grep] [VERIFIED: gsd classify-confidence]

## User Constraints

- This is a follow-up to Phase 20, not a new governor build. The planner must evolve `service/embeddinggovernor` and `relay/embedding_handler.go`. [VERIFIED: operator context]
- The existing Go-native path is canonical: client/GBrain/Obsidian -> router-ai-atius Go -> `service/embeddinggovernor` -> `Local TEI - GTE Embeddings` -> TEI. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:12]
- Do not reintroduce Python/model-detailed, a sidecar, or an extra container for local TEI embeddings. [VERIFIED: AGENTS.md:175]
- Graphify returned disabled in this checkout and `.planning/config.json` is absent, so planning must not rely on graph context. [VERIFIED: .planning/REQUIREMENTS.md:19]
- Current operator-provided production status says GBrain uses `https://router.atius.com.br/v1`, embeddings use `embedding-pt-v1`, `gbrain embed` sends `X-Embedding-Workload: batch` for embed commands, `GBRAIN_EMBED_CONCURRENCY=2`, provider sub-batch size is `4`, and the governed run had no embed errors at the reported checkpoint. [VERIFIED: operator context]
- Current operator-provided TEI status says pod ready is true, restarts are zero, CPU is near the 2 CPU limit during catch-up, memory is well below 12Gi, and a single `/health` curl timeout did not coincide with an embed error. [VERIFIED: operator context]
- Public model/brand protections from AGENTS.md remain in force; do not remove protected upstream identity or fork-specific routing guards. [VERIFIED: AGENTS.md:109]

## Project Constraints (from AGENTS.md)

- Backend work follows the Router -> Controller -> Service -> Model architecture; relay/provider logic lives under `relay/`, business logic under `service/`, DTOs under `dto/`, and shared helpers under `common/`. [VERIFIED: AGENTS.md:16]
- JSON marshal/unmarshal in business code must use `common.Marshal`, `common.Unmarshal`, `common.DecodeJson`, and related wrappers instead of direct `encoding/json` calls. [VERIFIED: AGENTS.md:57]
- Database changes must remain compatible with SQLite, MySQL >= 5.7.8, and PostgreSQL >= 9.6; this phase should avoid DB work unless a later plan explicitly needs persisted governor state. [VERIFIED: AGENTS.md:71]
- Optional scalar request DTO fields must preserve explicit zero values by using pointer types with `omitempty`. [VERIFIED: AGENTS.md:125]
- New or substantially rewritten Go backend tests must use `testify/require` for setup/fatal assertions and `testify/assert` for non-fatal assertions. [VERIFIED: AGENTS.md:147]
- Backend tests must protect behavior/contracts and must not be fake fuzz, stress, smoke, timing-only, duplicate, or implementation-detail coverage tests. [VERIFIED: AGENTS.md:147]
- The fork guard explicitly protects `relay/embedding_handler.go`, `service/embeddinggovernor/`, `docs/`, and `.planning/` from upstream sync regressions. [VERIFIED: AGENTS.md:178]
- The local TEI governor defaults are protected: `min=1`, `initial=2`, `max=3`, and `4` only for explicit/manual turbo windows. [VERIFIED: AGENTS.md:175]

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PHASE-20-PYTHON-MIDDLEWARE-REMOVAL | Middleware must not be required for `/v1/models`, route selection, provider queue/retry behavior, or MiniMax embeddings conversion. [VERIFIED: .planning/REQUIREMENTS.md:50] | Plan all governor changes inside Go `service/embeddinggovernor`, `relay/embedding_handler.go`, and existing Go DTO/relay paths. [VERIFIED: codebase grep] |
| PHASE-20-UPSTREAM-SYNC-GUARD | Fork-owned Go routing paths must be preserved during sync/merge. [VERIFIED: .planning/REQUIREMENTS.md:102] | Add tests and docs around `service/embeddinggovernor/` and `relay/embedding_handler.go` so future upstream sync cannot silently remove the Go-native governor. [VERIFIED: AGENTS.md:178] |
| PHASE-20-CLI-DOCS-RUNTIME-PARITY | Operators need CLI/docs parity and runtime restart docs must use user systemd services. [VERIFIED: .planning/REQUIREMENTS.md:93] | Update `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` with any new env vars, snapshot fields, and validation gates. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:153] |
| PHASE-20-SDK-SMOKES | Runtime validation must include SDK/smoke coverage and classify known upstream quota/rate-limit failures as upstream. [VERIFIED: .planning/REQUIREMENTS.md:113] | Use the existing embeddings smoke script for `embedding-pt-v1` and add targeted governor unit tests for adaptive behavior before production validation. [VERIFIED: scripts/smoke-embeddings.py] |

</phase_requirements>

## Summary

The governor already exists and is wired into the Go relay before upstream embedding dispatch. `relay/embedding_handler.go` acquires a governor lease using only model, channel metadata, and `X-Embedding-Workload`, then finishes the lease with success/failure, upstream status, and upstream latency. [VERIFIED: relay/embedding_handler.go:75] The governor currently supports configured model scope, queue limits, interactive/batch separation, cooldown, EWMA latency, success-window scale-up, demand-based interactive scale-up, and idle scale-down. [VERIFIED: service/embeddinggovernor/governor.go:118] [VERIFIED: service/embeddinggovernor/governor.go:276]

The exact planning gap is not "create a governor"; it is "make the active governor more adaptive without making bad decisions from noisy signals." [VERIFIED: operator context] The safest first implementation is request-local signal enrichment: derive input count and coarse size from `dto.EmbeddingRequest.ParseInput()`, keep the existing header override, never pass input text into governor state, and split batch/interactive latency/failure accounting so catch-up latency cannot poison interactive behavior. [VERIFIED: dto/embedding.go:59] [VERIFIED: service/embeddinggovernor/governor.go:25]

Optional TEI health probing can be in scope only as a disabled-by-default, read-only background signal with hysteresis. [CITED: https://huggingface.github.io/text-embeddings-inference/] A single `/health` timeout must never force an immediate reduction, because the local operational history shows `/health` can lag during long CPU inference while embeddings continue successfully. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:337]

**Primary recommendation:** Plan a Go-only adaptive follow-up in two steps: first add request-local workload signals and split metrics; then add optional health/probe influence only as a consecutive-window guardrail, never as a per-request dependency. [VERIFIED: codebase grep] [VERIFIED: operator context]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Embedding request parsing | API / Relay | DTO | `EmbeddingHelper` owns relay dispatch and `dto.EmbeddingRequest.ParseInput()` already exposes input strings for count/size derivation. [VERIFIED: relay/embedding_handler.go:22] [VERIFIED: dto/embedding.go:59] |
| Governor concurrency/backpressure | API / Backend service | Relay | `service/embeddinggovernor` owns queue, leases, cooldown, scale decisions, and snapshots; relay only supplies metadata and finish events. [VERIFIED: service/embeddinggovernor/governor.go:84] |
| Batch vs interactive classification | API / Backend service | API / Relay | Current classification uses `X-Embedding-Workload` plus configured batch models; the follow-up should add request-local input count/size without passing text. [VERIFIED: service/embeddinggovernor/governor.go:337] |
| Optional TEI health probing | API / Backend service | External TEI / Kubernetes | Probe interpretation belongs near governor state, but TEI and Kubernetes remain external dependencies and must be optional. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:201] |
| Production validation | Operator tooling | API / Backend | Existing validation uses Go tests, `scripts/smoke-embeddings.py`, `bin/clianything`, curl, and operator TEI/Kubernetes checks. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:203] |

## Current State And Remaining Gaps

| Area | Current Behavior | Remaining Gap |
|------|------------------|---------------|
| Model scope | Defaults govern `embedding-pt-v1` and `embedding-pt-v1-batch`; other models bypass the governor. [VERIFIED: service/embeddinggovernor/governor.go:13] | Keep scope unchanged unless an operator adds explicit model names through env. [VERIFIED: AGENTS.md:175] |
| Request metadata | `Request` carries only `Model`, `ChannelID`, `ChannelName`, and `Workload`; the code comment forbids embedding input text in governor requests. [VERIFIED: service/embeddinggovernor/governor.go:25] | Add non-sensitive derived fields such as `InputCount`, coarse `InputChars`, and `ClientWorkload` while preserving the no-text invariant. [VERIFIED: dto/embedding.go:59] |
| Workload classification | Header values `batch`/`bulk` force batch; `interactive`/`realtime` force interactive; otherwise configured batch models decide. [VERIFIED: service/embeddinggovernor/governor.go:337] | Auto-classify unlabeled large input arrays as batch-like or heavy using configurable thresholds. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:98] |
| Latency accounting | A single EWMA is updated for all governed successful or failed requests after dispatch. [VERIFIED: service/embeddinggovernor/governor.go:298] | Separate interactive and batch EWMA so long batch/catch-up requests do not block healthy interactive scale decisions. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:100] |
| Failure classification | Relay currently calls `finishGovernor(false, status)` for any non-200 upstream response. [VERIFIED: relay/embedding_handler.go:105] | Differentiate client/validation errors from infrastructure pressure; `429`, `5xx`, timeout, and connection failures should affect adaptive backpressure, while ordinary `4xx` should not close the circuit. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:104] |
| Scale-down safety | Any failed finish sets concurrency to min and starts cooldown. [VERIFIED: service/embeddinggovernor/governor.go:300] | Add hysteresis for optional health-derived reductions; a single health timeout must be insufficient. [VERIFIED: operator context] |
| Snapshot | `CurrentSnapshot()` exists but no caller outside tests/code search currently exposes it. [VERIFIED: service/embeddinggovernor/governor.go:157] [VERIFIED: rg CurrentSnapshot] | Decide whether to expose an admin-only read endpoint or rely on logs/docs; never expose input text or secrets. [VERIFIED: service/embeddinggovernor/governor.go:25] |
| Sub-batching | GBrain currently enforces provider sub-batch size `4`; TEI is configured with `--max-client-batch-size 4`. [VERIFIED: operator context] [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:325] | Do not include upstream request splitting/recomposition in this follow-up unless explicitly expanded; it has ordering, usage, error, and quota accounting complexity. [VERIFIED: scripts/smoke-embeddings.py] |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library `context`, `sync`, `time`, `net/http` | Go module/runtime `go1.25.1` | Queue wait timeouts, condition variable coordination, clocks, optional health probes. [VERIFIED: go env] | Existing governor already uses these packages and Go docs define cancellation/Cond semantics. [VERIFIED: service/embeddinggovernor/governor.go:3] [CITED: https://pkg.go.dev/context@go1.25.3] [CITED: https://pkg.go.dev/sync@go1.25.3] |
| Existing `service/embeddinggovernor` package | In-repo | Lease, queue, cooldown, concurrency state, and snapshots. [VERIFIED: service/embeddinggovernor/governor.go:84] | It is the protected fork-owned owner for local TEI backpressure. [VERIFIED: AGENTS.md:175] |
| Existing `relay/embedding_handler.go` | In-repo | Converts request, applies param overrides, acquires governor, dispatches upstream, and finishes lease. [VERIFIED: relay/embedding_handler.go:22] | It is the only current integration point before TEI dispatch. [VERIFIED: relay/embedding_handler.go:75] |
| Existing `dto.EmbeddingRequest` | In-repo | Parses OpenAI-compatible embedding input as strings. [VERIFIED: dto/embedding.go:22] | It provides request-local input count/size without adding a parser or sidecar. [VERIFIED: dto/embedding.go:59] |

### Supporting

| Library / Tool | Version | Purpose | When to Use |
|----------------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.11.1 | Deterministic Go assertions. [VERIFIED: go list -m] | Required for new or rewritten backend tests by AGENTS.md. [VERIFIED: AGENTS.md:160] |
| `github.com/gin-gonic/gin` | v1.9.1 | Existing HTTP router/controller framework. [VERIFIED: go list -m] | Only needed if the plan exposes a guarded snapshot endpoint. [VERIFIED: go.mod] |
| `scripts/smoke-embeddings.py` | In-repo | Token-based runtime embedding smoke with redaction helpers and dimension assertions. [VERIFIED: scripts/smoke-embeddings.py] | Use as production validation for `embedding-pt-v1` dimension `768`. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:203] |
| `bin/clianything` | In-repo | Router operational CLI for status, providers, embeddings, models, logs, and API calls. [VERIFIED: bin/clianything --help] | Use for provider/catalog checks and sanitized log inspection. [VERIFIED: tools/clianything.py:887] |
| `curl` / `jq` | curl 8.5.0, jq 1.7 | Runtime HTTP and JSON validation. [VERIFIED: curl --version] [VERIFIED: jq --version] | Use for `/health`, `/v1/models`, and shape checks without adding dependencies. [VERIFIED: command availability audit] |
| `kubectl` | v1.35.5+k3s1 client | Optional TEI/Kubernetes operational checks. [VERIFIED: kubectl version --client] | Use only with authorized kubeconfig; current session cannot read `/etc/rancher/k3s/k3s.yaml`. [VERIFIED: kubectl version --client] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-router Go adaptive logic | Python/model-detailed queue or sidecar | Disallowed by user and AGENTS; reintroduces a retired runtime owner. [VERIFIED: AGENTS.md:175] |
| Request-local input count/size | TEI/Kubernetes metrics first | Metrics can be useful, but request-local signals are available synchronously and do not require new privileges. [VERIFIED: dto/embedding.go:59] |
| Optional HTTP health sampler | Kubernetes client embedded in router | Kubernetes client would add auth/config surface and can fail in this session due kubeconfig permissions; keep Kubernetes checks operational or management-service-owned. [VERIFIED: kubectl version --client] |
| Single EWMA | Separate interactive/batch EWMAs | Single EWMA is simpler but known to risk batch latency poisoning interactive scale decisions. [VERIFIED: service/embeddinggovernor/governor.go:298] [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:100] |

**Installation:**

```bash
# No new packages should be installed for this follow-up. [VERIFIED: go.mod]
```

**Version verification:**

```bash
/usr/local/go/bin/go env GOVERSION GOMOD
/usr/local/go/bin/go list -m github.com/stretchr/testify
/usr/local/go/bin/go list -m github.com/gin-gonic/gin
```

## Package Legitimacy Audit

This phase should not install external packages. [VERIFIED: research scope] The recommended implementation uses Go stdlib, existing project packages, and existing dependencies already present in `go.mod`. [VERIFIED: go.mod]

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| none | none | n/a | n/a | n/a | n/a | No package install planned. [VERIFIED: research scope] |

**Packages removed due to [SLOP] verdict:** none. [VERIFIED: no package install planned]
**Packages flagged as suspicious [SUS]:** none. [VERIFIED: no package install planned]

## Architecture Patterns

### System Architecture Diagram

```text
OpenAI-compatible client / GBrain
        |
        v
POST /v1/embeddings
        |
        v
relay.EmbeddingHelper
        |
        +--> dto.EmbeddingRequest.ParseInput()
        |        |
        |        +--> derived metadata only: input_count, coarse_size, explicit_workload
        |
        v
service/embeddinggovernor.Acquire()
        |
        +--> governed model? no -> normal relay dispatch
        |
        +--> classify workload: header > input_count/size > configured batch model
        |
        +--> queue/cooldown/backpressure decision
        |        |
        |        +--> reject 429 before TEI if queue full/timeout
        |
        v
Existing provider adaptor / TEI upstream
        |
        v
Lease.Finish(success, status, latency, workload)
        |
        +--> update separate interactive/batch stats
        +--> reduce only on infrastructure pressure signals
        +--> optional TEI health sampler influences after consecutive bad windows
```

All boxes above are Go/router-side except TEI; no Python/model-detailed or sidecar is part of the proposed data path. [VERIFIED: AGENTS.md:175]

### Recommended Project Structure

```text
service/embeddinggovernor/
├── governor.go          # existing state machine, config, lease, snapshot [VERIFIED: codebase grep]
├── governor_test.go     # existing and new deterministic governor tests [VERIFIED: codebase grep]
relay/
├── embedding_handler.go # existing integration point, request-local signal extraction [VERIFIED: codebase grep]
dto/
├── embedding.go         # existing ParseInput source for non-sensitive counts [VERIFIED: codebase grep]
docs/
├── MANUAL-OPERACAO-ROUTER-AI-ATIUS.md # operator env/gates update [VERIFIED: docs grep]
scripts/
├── smoke-embeddings.py  # existing production smoke [VERIFIED: codebase grep]
```

### Pattern 1: Request-Local Signals First

**What:** Derive count and coarse size from `EmbeddingRequest.ParseInput()` before acquiring the governor, but pass only non-sensitive integers/enums to the governor. [VERIFIED: dto/embedding.go:59]  
**When to use:** Always for governed embedding models, because it works without TEI/Kubernetes privileges. [VERIFIED: service/embeddinggovernor/governor.go:326]  
**Example:**

```go
// Source: dto/embedding.go and service/embeddinggovernor/governor.go [VERIFIED: codebase grep]
inputs := embeddingReq.ParseInput()
signals := embeddinggovernor.Request{
    Model:       publicModelName,
    ChannelID:   c.GetInt("channel_id"),
    ChannelName: c.GetString("channel_name"),
    Workload:    c.GetHeader("X-Embedding-Workload"),
    // Planner should add metadata-only fields such as InputCount and TotalChars.
}
_ = inputs // count/size only; never store or log input text in governor state.
```

### Pattern 2: Separate Interactive And Batch Feedback

**What:** Keep batch and interactive counters/EWMAs separate while preserving a global concurrency cap. [VERIFIED: service/embeddinggovernor/governor.go:61]  
**When to use:** Use interactive EWMA for interactive scale-up decisions and batch EWMA for batch throttle decisions. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:100]  
**Example:**

```go
// Source: existing finish path plus recommended split metric fields. [VERIFIED: service/embeddinggovernor/governor.go:276]
if batch {
    g.batchLatencyEWMA = blendDuration(g.batchLatencyEWMA, latency)
} else {
    g.interactiveLatencyEWMA = blendDuration(g.interactiveLatencyEWMA, latency)
}
```

### Pattern 3: Consecutive-Window Probe Hysteresis

**What:** Optional health/probe samples should produce `healthy`, `degraded`, `unhealthy`, or `unknown`, and only sustained degraded/unhealthy windows should affect concurrency. [VERIFIED: operator context]  
**When to use:** Only if `EMBEDDING_GOVERNOR_TEI_HEALTH_URL` or equivalent is configured; disabled/misconfigured probing must degrade to `unknown`, not failure. [CITED: https://huggingface.github.io/text-embeddings-inference/]  
**Example:**

```go
// Source: TEI health docs + local operational warning about slow /health. [CITED: https://huggingface.github.io/text-embeddings-inference/] [VERIFIED: operator context]
if sample.Status != http.StatusOK || sample.Latency > cfg.HealthSlowAfter {
    g.badHealthWindows++
} else {
    g.badHealthWindows = 0
}
if g.badHealthWindows >= cfg.HealthWindowThreshold {
    g.reduceToMinWithCooldown(now)
}
```

### Anti-Patterns to Avoid

- **Recreating a Python/model-detailed queue:** This violates the fork guard and would create a second runtime owner for embeddings. [VERIFIED: AGENTS.md:175]
- **Letting one `/health` timeout close the circuit:** Local TEI can delay `/health` during long CPU inference while still completing embeddings. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:337]
- **Using batch latency as interactive health:** The current single EWMA cannot distinguish catch-up latency from interactive latency. [VERIFIED: service/embeddinggovernor/governor.go:298]
- **Logging or metric-labeling input text:** The governor request comment forbids input text, and relay currently has a debug log that can include the converted embedding body. [VERIFIED: service/embeddinggovernor/governor.go:25] [VERIFIED: relay/embedding_handler.go:64]
- **Adding Kubernetes credentials to the router:** Current `kubectl` exists but cannot read the k3s kubeconfig in this session; production Kubernetes checks should remain optional/operator-owned unless a separate secure management-service decision is made. [VERIFIED: kubectl version --client]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Provider queueing in another runtime | Python/model-detailed or sidecar queue | Existing Go `service/embeddinggovernor` | The fork guard requires local TEI embeddings to be governed inside the Go router. [VERIFIED: AGENTS.md:175] |
| Sensitive telemetry | Metrics/log labels containing request input, token, or raw body | Counts, durations, status classes, model/channel ids, and workload class | `Request` must never include embedding input text, and smoke helpers already scrub secrets. [VERIFIED: service/embeddinggovernor/governor.go:25] [VERIFIED: scripts/smoke-embeddings.py] |
| Per-request TEI health dependency | Synchronous `/health` curl before every embedding request | Optional background sampler with consecutive-window hysteresis | TEI `/health` can be slow during CPU inference, and per-request probes add load. [VERIFIED: operator context] |
| Kubernetes metrics client inside router | Embedded kubeconfig/client permissions | Optional operator/management checks or read-only HTTP health/Prometheus endpoint | Kubernetes access is external state and unavailable in this session without privileged kubeconfig. [VERIFIED: kubectl version --client] |
| Upstream request splitting in this phase | Custom response recomposer mixed into adaptive trigger work | Existing GBrain provider sub-batch size `4`; defer router sub-batching to a separate phase | Recomposition must preserve order, usage, quota, error semantics, and provider behavior. [VERIFIED: operator context] |

**Key insight:** The highest-value follow-up is not more concurrency; it is better classification of the work already flowing through Go. [VERIFIED: operator context] Request-local metadata is deterministic and cheap; TEI/Kubernetes health is useful only after debouncing and must not become required for normal request handling. [VERIFIED: dto/embedding.go:59] [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:366]

## Common Pitfalls

### Pitfall 1: Single Health Timeout Causes Bad Reduction

**What goes wrong:** The governor drops to `min=1` or enters cooldown after one slow health probe even though embedding requests still succeed. [VERIFIED: operator context]  
**Why it happens:** TEI `/health` can lag during long CPU inference, and local logs show previous liveness/readiness conclusions were contaminated by aggressive probe settings. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:194]  
**How to avoid:** Require multiple consecutive bad windows and combine health with request outcomes, restarts, and progress signals. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:366]  
**Warning signs:** `health=200` with high latency, no embed errors, pod ready, restarts zero, and GBrain `Embedded` still increasing. [VERIFIED: operator context]

### Pitfall 2: Batch Latency Poisons Interactive Latency

**What goes wrong:** Long catch-up requests raise the single EWMA and block interactive scale-up. [VERIFIED: service/embeddinggovernor/governor.go:298]  
**Why it happens:** Current EWMA is global even though batch and interactive requests have different slow thresholds. [VERIFIED: service/embeddinggovernor/governor.go:289]  
**How to avoid:** Maintain separate batch and interactive EWMA/slow counters and use the matching signal in `canIncreaseLocked`. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:100]  
**Warning signs:** Batch catch-up is running, interactive queue exists, no failures, but `CurrentConcurrency` refuses to return from `1` or `2`. [VERIFIED: codebase grep]

### Pitfall 3: Client 4xx Closes The Circuit

**What goes wrong:** A bad client request or validation error reduces all governed traffic to the minimum. [VERIFIED: relay/embedding_handler.go:105]  
**Why it happens:** Relay currently passes `success=false` for any non-200 upstream response before the governor can classify the status. [VERIFIED: relay/embedding_handler.go:105]  
**How to avoid:** Add a status classifier so only infrastructure pressure classes affect cooldown/reduction. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:104]  
**Warning signs:** `Failed` increases on ordinary `400`/`404` responses while TEI remains ready and healthy. [VERIFIED: service/embeddinggovernor/governor.go:300]

### Pitfall 4: Metrics Or Docs Leak Text/Secrets

**What goes wrong:** New snapshot fields, logs, or docs accidentally include embedding input, Authorization tokens, or provider secrets. [VERIFIED: AGENTS.md:164]  
**Why it happens:** Embedding request bodies contain raw user text, and relay currently has a debug log of the converted request body. [VERIFIED: relay/embedding_handler.go:64]  
**How to avoid:** Only expose counts, coarse sizes, status classes, durations, and model/channel identifiers; preserve smoke script redaction patterns. [VERIFIED: scripts/smoke-embeddings.py]  
**Warning signs:** A test fixture or docs snippet contains real input text, bearer tokens, OAuth JSON, or raw request bodies. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:292]

## Code Examples

Verified patterns from local code and official docs:

### Context Timeout Around Queue Waits

```go
// Source: existing governor and Go context docs. [VERIFIED: service/embeddinggovernor/governor.go:182] [CITED: https://pkg.go.dev/context@go1.25.3]
ctx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()
```

### Cond Wait Loop

```go
// Source: existing governor and Go sync docs. [VERIFIED: service/embeddinggovernor/governor.go:216] [CITED: https://pkg.go.dev/sync@go1.25.3]
for {
    if conditionIsReady() {
        break
    }
    cond.Wait()
}
```

### Metadata-Only Signal Extraction

```go
// Source: dto.ParseInput and governor Request no-text invariant. [VERIFIED: dto/embedding.go:59] [VERIFIED: service/embeddinggovernor/governor.go:25]
inputs := embeddingReq.ParseInput()
inputCount := len(inputs)
totalChars := 0
for _, input := range inputs {
    totalChars += len(input)
}
// Pass only inputCount and totalChars buckets; never pass the strings.
```

## State Of The Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Python/model-detailed owned parts of the `/v1` path and route enrichment. [VERIFIED: .planning/STATE.md] | Go owns `/v1/models`, provider routing, and local TEI governor path. [VERIFIED: .planning/phases/phase-20-go-native-model-router/20-VALIDATION.md] | Phase 20 final full-Go validation on 2026-06-18. [VERIFIED: .planning/phases/phase-20-go-native-model-router/20-VALIDATION.md] | Follow-up must extend Go, not revive Python. [VERIFIED: AGENTS.md:175] |
| TEI health probes were too aggressive and could kill or misclassify long CPU inference. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:194] | TEI probes use longer timeouts/thresholds operationally, and router defaults keep daily `initial=2`, `max=3`, `min=1`. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:195] | 2026-06-26 operational TEI/GBrain session. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:125] | Health is a noisy advisory signal, not a single-sample circuit breaker. [VERIFIED: operator context] |
| Header-only workload classification. [VERIFIED: service/embeddinggovernor/governor.go:337] | Recommended next approach adds request-local input count/size fallback while preserving header override. [VERIFIED: dto/embedding.go:59] | Proposed for this follow-up. [VERIFIED: research synthesis] | GBrain-like catch-up can be protected even when clients omit the workload header. [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md:98] |

**Deprecated/outdated:**

- Treating Graphify as fresh in this checkout is outdated; current phase docs say `.planning/config.json` is absent and Graphify is unavailable. [VERIFIED: .planning/REQUIREMENTS.md:19]
- Treating `max=4` as automatic daily governor behavior is outdated for the current TEI 2 CPU limit; `4` remains manual/turbo only. [VERIFIED: operator context] [VERIFIED: AGENTS.md:175]
- Reintroducing `model-detailed` for queue/retry is out of scope and violates the fork guard. [VERIFIED: AGENTS.md:175]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | If the planner includes an admin snapshot endpoint, existing auth/middleware patterns can guard it without a new dependency. [ASSUMED] | Current State And Remaining Gaps | A weak endpoint could expose operational state; planner should verify existing admin route patterns before exposing. |
| A2 | Router-side upstream sub-batching is outside this follow-up unless the user explicitly expands scope. [ASSUMED] | Don't Hand-Roll | If the user expects sub-batching now, the plan would under-scope a known gap. |
| A3 | Optional TEI health probing should be disabled by default and controlled by env/config. [ASSUMED] | Architecture Patterns | If operators want mandatory probing, the plan must add deployment config, failure policy, and permissions work. |

## Open Questions

1. **Should this phase expose a governor snapshot endpoint?**  
   - What we know: `CurrentSnapshot()` exists and returns non-sensitive counters/timestamps today. [VERIFIED: service/embeddinggovernor/governor.go:157]  
   - What's unclear: There is no current caller found by grep, and the desired operator surface is not specified. [VERIFIED: rg CurrentSnapshot]  
   - Recommendation: If needed, expose admin-only read access in a separate task with redaction tests; otherwise rely on logs and unit tests. [ASSUMED]

2. **Should optional TEI health probing ship now or wait?**  
   - What we know: TEI documents `/health`, Prometheus, and OpenTelemetry support, and local ops recommend 30s health timeout. [CITED: https://huggingface.github.io/text-embeddings-inference/] [CITED: https://github.com/huggingface/text-embeddings-inference/blob/main/README.md] [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:349]  
   - What's unclear: The router deployment config for a TEI health URL and whether the router should ever make Kubernetes-aware decisions are not specified. [ASSUMED]  
   - Recommendation: Include only a disabled-by-default HTTP health sampler if the planner can keep it read-only and hysteresis-based; otherwise defer probing and implement request-local signals first. [VERIFIED: operator context]

3. **Should router-side sub-batching be in this phase?**  
   - What we know: GBrain provider sub-batch size is `4`, and TEI max client batch size is `4`. [VERIFIED: operator context] [VERIFIED: /home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md:325]  
   - What's unclear: Router-side splitting would need response recomposition, index preservation, usage aggregation, and partial failure policy. [VERIFIED: scripts/smoke-embeddings.py]  
   - Recommendation: Treat sub-batching as a separate future phase unless the user explicitly says it is part of this follow-up. [ASSUMED]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go runtime | Unit tests/build | Yes via `/usr/local/go/bin/go`; `go` is not on default PATH. [VERIFIED: go version] | go1.25.1 linux/arm64 [VERIFIED: /usr/local/go/bin/go version] | Use absolute `/usr/local/go/bin/go` in plan commands. [VERIFIED: command availability audit] |
| Python 3 | Smoke script and CLI tests | Yes [VERIFIED: python3 --version] | 3.12.3 [VERIFIED: python3 --version] | None needed. [VERIFIED: command availability audit] |
| Bun | Frontend/i18n not expected in this phase | Yes [VERIFIED: bun --version] | 1.3.14 [VERIFIED: bun --version] | Not required unless docs/UI scope changes. [VERIFIED: research scope] |
| `curl` | Runtime HTTP validation | Yes [VERIFIED: command -v curl] | 8.5.0 [VERIFIED: curl --version] | Use `bin/clianything api` where appropriate. [VERIFIED: bin/clianything --help] |
| `jq` | JSON response assertions | Yes [VERIFIED: command -v jq] | 1.7 [VERIFIED: jq --version] | Python JSON parsing in smoke scripts. [VERIFIED: scripts/smoke-embeddings.py] |
| `kubectl` | Optional TEI/Kubernetes checks | Client exists but kubeconfig read failed in this session. [VERIFIED: kubectl version --client] | v1.35.5+k3s1 client [VERIFIED: kubectl version --client] | Treat Kubernetes checks as operator-gated or run from authorized context. [VERIFIED: kubectl version --client] |
| `bin/clianything` | Router operational checks | Yes [VERIFIED: bin/clianything --help] | In-repo script [VERIFIED: bin/clianything --help] | Direct curl/SQL only if CLI is unavailable. [VERIFIED: tools/clianything.py] |
| Public router `/health` | Basic public runtime liveness | HTTP 200 in this session; body maps to router API status/info, not TEI health. [VERIFIED: curl https://router.atius.com.br/health] | n/a | Use TEI direct `/health` or Kubernetes checks for TEI-specific health. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:46] |

**Missing dependencies with no fallback:**

- None for code-level research and unit-test planning. [VERIFIED: environment audit]

**Missing dependencies with fallback:**

- Kubernetes access is present as a client but blocked by kubeconfig permissions in this session; planner should make TEI/Kubernetes validation an operator-gated production step. [VERIFIED: kubectl version --client]

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` plus `github.com/stretchr/testify` v1.11.1. [VERIFIED: go list -m] |
| Config file | `go.mod`; no separate Go test config found. [VERIFIED: rg --files] |
| Quick run command | `/usr/local/go/bin/go test ./service/embeddinggovernor ./relay -count=1` [VERIFIED: command run 2026-06-26] |
| Full focused suite command | `/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1` [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:203] |
| Runtime smoke command | `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-pt-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 ATIUS_ROUTER_TOKEN=... python3 scripts/smoke-embeddings.py` [VERIFIED: scripts/smoke-embeddings.py] |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| GOV-AUTO-BATCH | Missing workload header with input array count above threshold is classified as batch/heavy without storing text. [VERIFIED: dto/embedding.go:59] | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor -run 'Test.*Batch.*Input|Test.*Workload' -count=1` [ASSUMED test name] | Existing file yes; new test needed. [VERIFIED: service/embeddinggovernor/governor_test.go] |
| GOV-SPLIT-EWMA | Batch latency does not block interactive scale-up when interactive latency is healthy. [VERIFIED: service/embeddinggovernor/governor.go:298] | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor -run 'Test.*Batch.*Interactive.*Latency' -count=1` [ASSUMED test name] | Existing file yes; new test needed. [VERIFIED: service/embeddinggovernor/governor_test.go] |
| GOV-STATUS-CLASS | Ordinary client `4xx` does not reduce concurrency; `429`, `5xx`, timeout, and connection errors do. [VERIFIED: relay/embedding_handler.go:105] | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor -run 'Test.*Status.*Class' -count=1` [ASSUMED test name] | Existing file yes; new test needed. [VERIFIED: service/embeddinggovernor/governor_test.go] |
| GOV-HEALTH-HYSTERESIS | One slow/failed health sample does not reduce concurrency; consecutive bad windows can reduce. [VERIFIED: operator context] | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor -run 'Test.*Health.*Hysteresis' -count=1` [ASSUMED test name] | Existing file yes; new test needed if probing included. [VERIFIED: service/embeddinggovernor/governor_test.go] |
| GOV-NO-TEXT | Snapshot/loggable governor state contains no embedding input text. [VERIFIED: service/embeddinggovernor/governor.go:25] | unit/static | `/usr/local/go/bin/go test ./service/embeddinggovernor -run 'Test.*No.*Input.*Text|Test.*Snapshot' -count=1` [ASSUMED test name] | Existing file yes; new test needed. [VERIFIED: service/embeddinggovernor/governor_test.go] |
| RUNTIME-EMBED | `POST /v1/embeddings` with `embedding-pt-v1` returns vectors dimension `768`. [VERIFIED: operator context] | production smoke | `ATIUS_ROUTER_TOKEN=... ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-pt-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py` [VERIFIED: scripts/smoke-embeddings.py] | Yes. [VERIFIED: scripts/smoke-embeddings.py] |

### Sampling Rate

- **Per task commit:** `/usr/local/go/bin/go test ./service/embeddinggovernor ./relay -count=1` [VERIFIED: command run 2026-06-26]
- **Per wave merge:** `/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1` [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:203]
- **Phase gate:** Go focused suite green, smoke embeddings with dimension `768`, `GET /v1/models` includes `embedding-pt-v1`, `model-detailed` remains out of the path, and optional TEI checks show ready/restarts stable if operator access is available. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:135]

### Wave 0 Gaps

- [ ] Add deterministic unit tests in `service/embeddinggovernor/governor_test.go` for input-count classification, split EWMAs, status classification, and optional health hysteresis. [VERIFIED: service/embeddinggovernor/governor_test.go]
- [ ] Add or adjust tests around relay request metadata if the implementation changes `embeddinggovernor.Request` construction in `relay/embedding_handler.go`. [VERIFIED: relay/embedding_handler.go:75]
- [ ] Update `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` with any new env vars and production validation commands. [VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:153]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no direct auth change | Reuse existing router/admin auth if a snapshot endpoint is added. [ASSUMED] |
| V3 Session Management | no | This phase does not change sessions. [VERIFIED: research scope] |
| V4 Access Control | yes if exposing snapshot | Admin-only/read-only route or no endpoint. [ASSUMED] |
| V5 Input Validation | yes | Derive only count/size/workload enum from `EmbeddingRequest`; never trust headers as sole signal. [VERIFIED: dto/embedding.go:22] |
| V6 Cryptography | no | No new cryptographic behavior planned. [VERIFIED: research scope] |
| V7 Error Handling and Logging | yes | Keep request text, Authorization, OAuth files, and provider secrets out of metrics/docs/logs. [VERIFIED: AGENTS.md:164] |
| V14 Configuration | yes | Any optional TEI probe must be env/config gated and safe when unset. [ASSUMED] |

### Known Threat Patterns for Go Embedding Governor

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Embedding text leaked via metrics/logs/docs | Information Disclosure | Store/log only metadata counts, durations, statuses, workload class, and model/channel ids. [VERIFIED: service/embeddinggovernor/governor.go:25] |
| Forged workload header starves interactive traffic | Denial of Service | Treat header as a hint plus enforce global cap, batch cap, queue limits, and request-local classification. [VERIFIED: service/embeddinggovernor/governor.go:337] |
| Health probe manipulation causes excessive downscale | Denial of Service | Disable by default; require consecutive bad windows; combine with request outcomes. [VERIFIED: operator context] |
| Snapshot endpoint leaks operational details | Information Disclosure | Prefer no endpoint unless needed; if added, admin-only and no raw request bodies. [ASSUMED] |

## Sources

### Primary (HIGH confidence)

- `AGENTS.md` - project architecture, test quality, fork guards, local TEI governor constraints. [VERIFIED: AGENTS.md]
- `.planning/REQUIREMENTS.md` - Phase 20 validated requirements, Graphify disabled note, no Python/model-detailed path. [VERIFIED: .planning/REQUIREMENTS.md]
- `service/embeddinggovernor/governor.go` - existing governor config, queue, lease, cooldown, EWMA, snapshot, and scaling behavior. [VERIFIED: codebase grep]
- `service/embeddinggovernor/governor_test.go` - existing deterministic unit test patterns with `testify`. [VERIFIED: codebase grep]
- `relay/embedding_handler.go` - current acquire/finish integration before upstream dispatch. [VERIFIED: codebase grep]
- `dto/embedding.go` - current input parsing and request DTO behavior. [VERIFIED: codebase grep]
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - current operational contract and validation commands. [VERIFIED: repo docs]
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-router-embedding-governor-go-native.md` - local operational derivation and next-gap list. [VERIFIED: local Obsidian log]
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/2026-06-26-gbrain-obsidian-tei-atius-router-service-management.md` - TEI/GBrain operational status, probe risk, and proactive trigger candidates. [VERIFIED: local Obsidian log]

### Secondary (MEDIUM confidence)

- Context7 `/websites/pkg_go_dev_go1_25_3` - Go context and sync documentation digests cached by GSD research-store. [CITED: https://pkg.go.dev/context@go1.25.3] [CITED: https://pkg.go.dev/sync@go1.25.3]
- Context7 `/huggingface/text-embeddings-inference` - TEI health and observability docs digested through GSD research-store. [CITED: https://huggingface.github.io/text-embeddings-inference/] [CITED: https://github.com/huggingface/text-embeddings-inference/blob/main/README.md]

### Tertiary (LOW confidence)

- None used for recommendations. [VERIFIED: research log]

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - based on `go.mod`, `go env`, and existing code paths. [VERIFIED: go.mod] [VERIFIED: go env]
- Architecture: HIGH - based on AGENTS.md, current governor code, relay integration, and Phase 20 artifacts. [VERIFIED: AGENTS.md] [VERIFIED: codebase grep]
- Pitfalls: HIGH for local operational pitfalls from logs and code; MEDIUM for TEI official health/metrics details via Context7. [VERIFIED: local Obsidian log] [CITED: https://github.com/huggingface/text-embeddings-inference/blob/main/README.md]

**Research date:** 2026-06-26 [VERIFIED: system date]  
**Valid until:** 2026-07-03 for runtime/TEI threshold details; 2026-07-26 for codebase architecture if no upstream sync lands. [ASSUMED]
