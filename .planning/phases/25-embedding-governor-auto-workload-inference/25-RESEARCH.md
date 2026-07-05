# Phase 25: embedding-governor-auto-workload-inference - Research

**Researched:** 2026-07-05
**Domain:** Go-native embeddings relay, TEI governor workload inference, runtime smoke validation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

**Source:** `.planning/phases/25-embedding-governor-auto-workload-inference/25-CONTEXT.md` lines 15-96. `[VERIFIED: 25-CONTEXT.md]`

### Locked Decisions

## Implementation Decisions

### D-01 Governed model scope
- `embedding-gte-v1` must stay governed inside the Go router through `service/embeddinggovernor/` and `relay/embedding_handler.go`.
- `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` remains the default intended model scope.
- `EMBEDDING_GOVERNOR_BATCH_MODELS=` remains empty; do not introduce a public `embedding-gte-v1-batch` alias.
- `model != embedding-gte-v1` keeps current behavior and must not enter the governor unless explicitly configured in `EMBEDDING_GOVERNOR_MODELS`.

### D-02 Header override priority
- Explicit `X-Embedding-Workload` keeps priority over automatic inference.
- `batch` and existing `bulk` semantics classify as batch.
- `interactive` and `realtime` classify as interactive.
- Invalid or absent header falls back to router-side automatic inference when enabled.

### D-03 Automatic workload inference
- Add a testable helper in `service/embeddinggovernor/` for model scope and workload classification.
- Required helper surface should include an `IsGovernedModel("embedding-gte-v1")` equivalent and a `ClassifyWorkload(...)` equivalent.
- The helper must not retain or expose raw embedding text; use request metadata such as input count and character count.
- Without header, `input` array with at least 2 items must classify as `batch`.
- Without header, a single string classifies as `interactive` unless it crosses the configured character threshold.
- Add optional `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true` behavior, defaulting to the safe automatic behavior for governed models.
- Add or normalize `EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD=2`.

### D-04 TEI batch safety
- Preserve conservative batch concurrency (`EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=1`) and separate interactive feedback metrics.
- Preserve automatic concurrency bounds: min 1, initial 2, max 3; keep 4 reserved for explicit/manual turbo windows.
- Keep TEI max client batch size 4 as an execution invariant. If the current relay path can forward arrays larger than 4 to TEI without splitting, the plan must include a bounded sub-batch strategy or an explicit validation that the existing path already enforces it.

### D-05 Validation and smoke
- Unit tests must prove:
  - `model=embedding-gte-v1`, `input="texto"`, no header -> `interactive`.
  - `model=embedding-gte-v1`, `input=["a","b"]`, no header -> `batch`.
  - Header `batch` forces batch for a single string.
  - Header `interactive` forces interactive for a small array.
  - Unknown model does not enter the governor.
  - Batch larger than 4 respects the TEI sub-batch/cap contract.
- Relay tests must prove the classification happens before governor acquisition or that the request passed to the governor already carries the resolved workload.
- Live smoke after implementation must hit authenticated `/v1/embeddings` for `embedding-gte-v1` and verify dimensions `768`, without printing tokens.

### the agent's Discretion
- Exact helper names and signatures may vary if they stay testable, do not leak raw input, and keep existing project style.
- Exact docs placement may use `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`, `docs/CLIANYTHING.md`, or a narrower runbook section if the planner finds a better canonical doc.

### Deferred Ideas (OUT OF SCOPE)

## Deferred Ideas

- Do not implement semantic Graphify indexing in this phase.
- Do not change the public model catalog shape.
- Do not activate Codex/OpenAI `text-embedding-3-*` embeddings.
- Do not create a Python/model-detailed sidecar or any extra container.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PHASE-25-GOVERNED-MODEL-SCOPE | `embedding-gte-v1` must remain the single default public governed local embedding alias; `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1`; empty batch-model alias list; unknown models no-op. `[VERIFIED: .planning/REQUIREMENTS.md]` | Current code defaults `defaultModels` to `embedding-gte-v1`, `defaultBatchModels` to empty, and no-ops non-governed models. `[VERIFIED: service/embeddinggovernor/governor.go]` |
| PHASE-25-AUTO-WORKLOAD-INFERENCE | Unlabeled governed requests must classify by metadata with header priority, count threshold `2`, char threshold, and an enabled-by-default auto-workload control. `[VERIFIED: .planning/REQUIREMENTS.md]` | Current code already classifies by header, `InputCount`, and `InputChars`, but default count threshold is `4` and no `EMBEDDING_GOVERNOR_AUTO_WORKLOAD` exists. `[VERIFIED: service/embeddinggovernor/governor.go]` |
| PHASE-25-HEADER-OVERRIDE-COMPATIBILITY | `batch`/`bulk` force batch, `interactive`/`realtime` force interactive, invalid header falls back to inference, and docs must mark the header optional. `[VERIFIED: .planning/REQUIREMENTS.md]` | Current `isBatch` implements the four valid override values and falls through on unknown or absent header. `[VERIFIED: service/embeddinggovernor/governor.go]` |
| PHASE-25-TEI-BATCH-SAFETY | Batch concurrency remains conservative, accounting remains split, automatic max stays `3`, and input arrays over `4` must not overload TEI. `[VERIFIED: .planning/REQUIREMENTS.md]` | Current concurrency and split accounting exist; current relay/OpenAI adapter does not split or cap embedding input arrays before TEI. `[VERIFIED: service/embeddinggovernor/governor.go; relay/embedding_handler.go; relay/channel/openai/adaptor.go]` |
| PHASE-25-CLIENT-SMOKE-VALIDATION | Tests and runtime smokes must prove clients can omit `X-Embedding-Workload`; authenticated smoke must validate `embedding-gte-v1` dimension `768` without token leakage. `[VERIFIED: .planning/REQUIREMENTS.md]` | Current smoke script redacts secrets and omits the workload header, but its defaults are still `http://127.0.0.1:3001/v1`, `embo-01`, and dimension `1536`. `[VERIFIED: scripts/smoke-embeddings.py]` |
</phase_requirements>

## Summary

Phase 25 is a narrow backend/runtime contract phase in the Go-native embeddings path, not a catalog redesign or provider split. `[VERIFIED: 25-CONTEXT.md; AGENTS.md]` The current code already routes `/v1/embeddings` through `relay/embedding_handler.go`, passes the public model alias before upstream model mapping, sends only numeric `InputCount` and `InputChars` plus the workload header to `service/embeddinggovernor`, and keeps raw embedding text out of governor state. `[VERIFIED: relay/embedding_handler.go; dto/embedding.go; service/embeddinggovernor/governor.go]`

The desired behavior is partially implemented. `[VERIFIED: service/embeddinggovernor/governor.go; service/embeddinggovernor/governor_test.go]` Header override priority already exists, invalid or missing header already falls through to metadata thresholds, and separate batch/interactive accounting already exists. `[VERIFIED: service/embeddinggovernor/governor.go]` The gaps are specific: default input-count threshold is `4` instead of required `2`, there is no explicit `EMBEDDING_GOVERNOR_AUTO_WORKLOAD` config flag, the classifier surface is unexported/private, relay tests only cover a header-present case, the smoke script defaults are stale, and the current relay path does not enforce TEI's client batch cap of `4`. `[VERIFIED: service/embeddinggovernor/governor.go; relay/embedding_handler_test.go; scripts/smoke-embeddings.py; relay/channel/openai/adaptor.go]`

**Primary recommendation:** Implement Phase 25 as three executable plan slices: governor classifier contract, relay/cap safety tests and enforcement, then docs/smoke updates. `[VERIFIED: 25-CONTEXT.md; .planning/REQUIREMENTS.md]`

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Governed model scoping for `embedding-gte-v1` | API / Backend | Service | `relay/embedding_handler.go` calls `embeddinggovernor.Acquire`, and `service/embeddinggovernor` owns model applicability. `[VERIFIED: relay/embedding_handler.go; service/embeddinggovernor/governor.go]` |
| Workload inference | API / Backend | DTO | `dto.EmbeddingRequest.GetInputStats()` derives numeric metadata; `service/embeddinggovernor` classifies header/count/chars. `[VERIFIED: dto/embedding.go; service/embeddinggovernor/governor.go]` |
| Header override compatibility | API / Backend | Browser / Client | Clients may send `X-Embedding-Workload`, but router-side classification remains authoritative and bounded. `[VERIFIED: 25-CONTEXT.md; service/embeddinggovernor/governor.go]` |
| TEI backpressure and queues | API / Backend | External TEI service | The governor controls queueing, lease acquisition, batch concurrency, cooldown, and feedback before upstream dispatch. `[VERIFIED: service/embeddinggovernor/governor.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| Public model catalog shape | API / Backend | Database / Storage | Phase 25 must not change `/v1/models` shape or add `*-batch`; the catalog currently has one active `embedding-gte-v1` row. `[VERIFIED: AGENTS.md; clianything query]` |
| Authenticated smoke validation | Runtime / CLI | API / Backend | `scripts/smoke-embeddings.py` sends authenticated `/v1/embeddings` requests and validates vector shape. `[VERIFIED: scripts/smoke-embeddings.py]` |

## Project Constraints (from AGENTS.md)

- Backend work must follow the repo's layered Router -> Controller -> Service -> Model shape; this phase belongs in `relay/`, `dto/`, and `service/embeddinggovernor/`. `[VERIFIED: AGENTS.md]`
- Business code must use `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`, `common.DecodeJson`, and related wrappers instead of direct `encoding/json` marshal/unmarshal calls. `[VERIFIED: AGENTS.md]`
- Backend tests must use `github.com/stretchr/testify/require` for setup/fatal assertions and `github.com/stretchr/testify/assert` for non-fatal value checks. `[VERIFIED: AGENTS.md; Context7 /websites/pkg_go_dev_github_com_stretchr_testify]`
- Local TEI embeddings must remain governed inside the Go router through `service/embeddinggovernor/` and `relay/embedding_handler.go`; no Python/model-detailed owner, sidecar, or extra container is allowed. `[VERIFIED: AGENTS.md]`
- The only default public governed local embedding model is `embedding-gte-v1`; batch selection is internal and must not create a public `*-batch` alias. `[VERIFIED: AGENTS.md]`
- Daily governor concurrency must keep fallback `min=1`, start at `initial=2`, cap automatic scale at `max=3`, and reserve `4` for explicit/manual turbo windows. `[VERIFIED: AGENTS.md; service/embeddinggovernor/governor.go]`
- Runtime directories `/backups`, `/data`, `/logs`, and `/runtime` must stay out of build context. `[VERIFIED: AGENTS.md]`
- Protected project and organization identifiers must not be removed, renamed, or replaced. `[VERIFIED: AGENTS.md]`

## Current Behavior Inventory

| Area | Current Behavior | Gap / Planning Implication |
|------|------------------|----------------------------|
| Model scope | `defaultModels = "embedding-gte-v1"` and `defaultBatchModels = ""`; `Acquire` returns no lease/reject when `applies(model)` is false. `[VERIFIED: service/embeddinggovernor/governor.go]` | Add explicit tests/helper surface for `embedding-gte-v1` governed and unknown model no-op. `[VERIFIED: 25-CONTEXT.md]` |
| Runtime catalog | Read-only CLI shows channel `9`, name `TEI - GTE Embeddings`, type `1`, status `1`, models `embedding-gte-v1`, and one enabled `embedding-gte-v1` model row. `[VERIFIED: clianything embeddings; clianything query]` | Docs still contain older wording `Local TEI - GTE Embeddings` in places; planner should update docs carefully without changing runtime rows. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; clianything embeddings]` |
| Header override | `batch`/`bulk` return batch; `interactive`/`realtime` return interactive; other values fall through. `[VERIFIED: service/embeddinggovernor/governor.go]` | Keep this priority unchanged and add no-header/invalid-header relay coverage. `[VERIFIED: .planning/REQUIREMENTS.md]` |
| Auto metadata inference | Current classifier treats `InputCount >= BatchInputCountThreshold` or `InputChars >= BatchInputCharsThreshold` as batch. `[VERIFIED: service/embeddinggovernor/governor.go]` | Change/normalize default count threshold from `4` to `2` and add `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true`. `[VERIFIED: .planning/REQUIREMENTS.md; service/embeddinggovernor/governor.go]` |
| Relay metadata | `EmbeddingHelper` reads `embeddingReq.GetInputStats()` before model mapping and passes `Model`, `ChannelID`, `ChannelName`, `Workload`, `InputCount`, and `InputChars` to `Acquire`. `[VERIFIED: relay/embedding_handler.go]` | Existing relay test captures only header-present batch; add no-header string/array cases. `[VERIFIED: relay/embedding_handler_test.go; 25-CONTEXT.md]` |
| Input stats | `GetInputStats()` uses `ParseInput()` and returns count plus UTF-8 rune count; tests prove no raw text is present in rendered stats. `[VERIFIED: dto/embedding.go; dto/embedding_test.go]` | Reuse this helper; do not pass raw input into the governor. `[VERIFIED: AGENTS.md; 25-CONTEXT.md]` |
| TEI batch cap | `relay/channel/openai.Adaptor.ConvertEmbeddingRequest` returns the embedding request unchanged, and `GetAndValidateEmbeddingRequest` only checks `input` is present. `[VERIFIED: relay/channel/openai/adaptor.go; relay/helper/valid_request.go]` | Current code can forward arrays over `4`; planner must add bounded sub-batching or a fail-closed cap for governed TEI requests. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; relay/channel/openai/adaptor.go]` |
| Smoke script | Script omits `X-Embedding-Workload`, redacts token-like values, exits `2` when `ATIUS_ROUTER_TOKEN` is missing, and validates vector dimension. `[VERIFIED: scripts/smoke-embeddings.py; env -u ATIUS_ROUTER_TOKEN smoke run]` | Defaults are stale for Phase 25: base `3001`, model `embo-01`, expected dim `1536`, no array-mode smoke. `[VERIFIED: scripts/smoke-embeddings.py]` |
| Local service reachability | `GET http://127.0.0.1:3000/v1/models` returned `401` without token, which is expected for unauthenticated catalog access. `[VERIFIED: curl probe; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` | Authenticated live `/v1/embeddings` smoke still requires an ephemeral token; token is not present in this shell. `[VERIFIED: environment audit]` |

## Standard Stack

### Core

| Library / Tool | Version | Purpose | Why Standard |
|----------------|---------|---------|--------------|
| Go toolchain | `go1.25.1 linux/arm64` runtime; `go 1.25.1` in `go.mod`. `[VERIFIED: go version; go.mod]` | Backend build and tests. | Existing repo backend is Go and prior phase gates use `/usr/local/go/bin/go`. `[VERIFIED: AGENTS.md; 20-04-SUMMARY.md]` |
| Gin | `github.com/gin-gonic/gin v1.9.1`. `[VERIFIED: go.mod; Context7 /gin-gonic/gin]` | HTTP context and relay handler tests. | Existing relay tests use `gin.CreateTestContext`; docs support `httptest` and `GetHeader`. `[VERIFIED: relay/embedding_handler_test.go; Context7 /gin-gonic/gin]` |
| Testify | `github.com/stretchr/testify v1.11.1`. `[VERIFIED: go.mod; Context7 /websites/pkg_go_dev_github_com_stretchr_testify]` | Deterministic Go assertions. | AGENTS requires `require` for fatal setup and `assert` for value checks. `[VERIFIED: AGENTS.md; Context7 /websites/pkg_go_dev_github_com_stretchr_testify]` |
| Go stdlib `net/http` / `httptest` | Go `1.25.1`. `[VERIFIED: go version; Context7 /websites/pkg_go_dev_go1_25_3]` | Header semantics, recorder/request tests. | Header `Get` is case-insensitive and returns empty string when absent, matching the no-header classifier fallback. `[VERIFIED: Context7 /websites/pkg_go_dev_go1_25_3]` |
| `bin/clianything` | Present in repo; help command works. `[VERIFIED: environment audit]` | Read-only catalog/runtime inspection. | Project skill and docs prefer it because sensitive values are redacted by default. `[VERIFIED: router-ai-atius skill; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |

### Supporting

| Library / Tool | Version | Purpose | When to Use |
|----------------|---------|---------|-------------|
| Python | `Python 3.12.3`. `[VERIFIED: python3 --version]` | Run and compile `scripts/smoke-embeddings.py`. | Use for smoke script syntax checks and authenticated runtime smoke. `[VERIFIED: scripts/smoke-embeddings.py]` |
| Bun | `1.3.14`. `[VERIFIED: bun --version]` | Frontend package manager. | Not needed for Phase 25 unless docs/UI are unexpectedly touched. `[VERIFIED: AGENTS.md]` |
| Graphify | `graphify 0.8.39` last build, graph fresh at current commit. `[VERIFIED: graphify status]` | Mandatory GSD routing context. | Use before planning/execution and after modifying GSD/code artifacts. `[VERIFIED: AGENTS.md; graphify status]` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Existing Go governor | Python/model-detailed sidecar | Forbidden for this phase and fork; would reintroduce retired runtime ownership. `[VERIFIED: AGENTS.md; 25-CONTEXT.md]` |
| One public `embedding-gte-v1` model | Public `embedding-gte-v1-batch` alias | Forbidden; batch is internal workload class, not catalog surface. `[VERIFIED: AGENTS.md; .planning/REQUIREMENTS.md]` |
| Fail-closed cap for input arrays over `4` | Transparent sub-batching and response recomposition | Sub-batching can preserve large-array success but requires response ordering/usage merge work; fail-closed cap is smaller and safer unless product explicitly requires transparent arrays over `4`. `[VERIFIED: relay/channel/openai/adaptor.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |

**Installation:**

No new package installation is recommended for Phase 25. `[VERIFIED: go.mod; 25-CONTEXT.md]`

```bash
# No npm/go package install is required.
```

**Version verification performed:**

```bash
/usr/local/go/bin/go version
python3 --version
bun --version
rg -n 'github.com/gin-gonic/gin|github.com/stretchr/testify|go ' go.mod
```

## Package Legitimacy Audit

No external packages should be installed in this phase. `[VERIFIED: 25-CONTEXT.md; go.mod]`

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| none | n/a | n/a | n/a | n/a | n/a | No install planned. `[VERIFIED: 25-CONTEXT.md]` |

**Packages removed due to [SLOP] verdict:** none. `[VERIFIED: no package install planned]`
**Packages flagged as suspicious [SUS]:** none. `[VERIFIED: no package install planned]`

## Architecture Patterns

### System Architecture Diagram

```text
Client / SDK / Graphify / GBrain
        |
        | POST /v1/embeddings
        | optional X-Embedding-Workload
        v
Go relay: relay/embedding_handler.go
        |
        | parse dto.EmbeddingRequest
        | derive InputCount/InputChars
        | preserve public model alias
        v
service/embeddinggovernor
        |
        | Is governed model?
        +--> no: no lease, normal relay path
        |
        +--> yes:
              |
              | classify workload:
              | explicit header -> metadata count/chars -> batch model set
              v
        queue / lease / concurrency envelope
              |
              | success/failure/latency feedback
              v
OpenAI-compatible TEI upstream
        |
        v
Embedding response -> quota consume -> client
```

### Recommended Project Structure

```text
dto/
  embedding.go              # metadata-only input stats helper
  embedding_test.go         # input stats regression tests
relay/
  embedding_handler.go      # governor acquire point and TEI cap enforcement
  embedding_handler_test.go # captured governor request and cap tests
service/embeddinggovernor/
  governor.go               # config, model scope, classifier, queueing
  governor_test.go          # classifier and concurrency tests
docs/
  MANUAL-OPERACAO-ROUTER-AI-ATIUS.md # operator contract and env docs
scripts/
  smoke-embeddings.py       # authenticated smoke for no-header clients
```

### Pattern 1: Metadata-Only Relay Boundary

**What:** The relay derives `InputCount` and `InputChars` before upstream conversion and sends only numeric metadata plus model/header/channel metadata to the governor. `[VERIFIED: relay/embedding_handler.go; dto/embedding.go]`

**When to use:** Use this for every workload decision; do not pass raw input strings, JSON request bodies, tokens, or Authorization values into governor state or snapshots. `[VERIFIED: AGENTS.md; service/embeddinggovernor/governor_test.go]`

**Example:**

```go
// Source: relay/embedding_handler.go
inputStats := embeddingReq.GetInputStats()
lease, reject := acquireEmbeddingGovernor(c.Request.Context(), embeddinggovernor.Request{
    Model:      publicModelName,
    Workload:   c.GetHeader("X-Embedding-Workload"),
    InputCount: inputStats.InputCount,
    InputChars: inputStats.InputChars,
})
```

### Pattern 2: Header Priority Then Metadata

**What:** The current classifier checks explicit workload override values first, then metadata thresholds, then the configured batch-model set. `[VERIFIED: service/embeddinggovernor/governor.go]`

**When to use:** Preserve this exact priority while changing the default count threshold to `2` and adding an auto-workload guard flag. `[VERIFIED: .planning/REQUIREMENTS.md]`

**Example:**

```go
// Source: service/embeddinggovernor/governor.go
if workload == "batch" || workload == "bulk" {
    return true
}
if workload == "interactive" || workload == "realtime" {
    return false
}
```

### Pattern 3: Captured Governor Request Tests

**What:** `relay/embedding_handler_test.go` swaps the package-local `acquireEmbeddingGovernor` hook, captures the request, returns a synthetic reject, and asserts metadata without dispatching upstream. `[VERIFIED: relay/embedding_handler_test.go]`

**When to use:** Add no-header tests for single string and small array by reusing this hook. `[VERIFIED: relay/embedding_handler_test.go; 25-CONTEXT.md]`

### Anti-Patterns to Avoid

- **Public `*-batch` alias:** It violates the Phase 25 model scope and fork guardrails. `[VERIFIED: AGENTS.md; .planning/REQUIREMENTS.md]`
- **Raw input in governor state/logs:** It risks leaking embedding text and violates the metadata-only boundary. `[VERIFIED: service/embeddinggovernor/governor_test.go; AGENTS.md]`
- **Relying on clients to send the header:** The phase goal is router-side inference when `X-Embedding-Workload` is absent. `[VERIFIED: 25-CONTEXT.md]`
- **Assuming TEI enforces the max batch cap for the router:** Current Go validation/adaptor path does not cap arrays before dispatch. `[VERIFIED: relay/helper/valid_request.go; relay/channel/openai/adaptor.go]`
- **Using `go` from PATH in plans:** This shell does not have `go` on PATH; use `/usr/local/go/bin/go`. `[VERIFIED: environment audit]`

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Backpressure / queues | A new sidecar queue, Python middleware, or extra container | Existing `service/embeddinggovernor` lease/queue/cooldown logic | The fork requires Go-native ownership and current code already owns queueing. `[VERIFIED: AGENTS.md; service/embeddinggovernor/governor.go]` |
| Workload classifier | Client-side heuristics or a new public model alias | Router-side `ClassifyWorkload`/`isBatch` equivalent using header and metadata | Clients should omit the header during normal operation. `[VERIFIED: 25-CONTEXT.md; .planning/REQUIREMENTS.md]` |
| Secret/token handling in smoke | Printing env vars or Authorization headers | Existing `_scrub()` and token-missing exit `2` pattern | Smoke must validate live behavior without token leakage. `[VERIFIED: scripts/smoke-embeddings.py; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| JSON handling in business code | Direct `encoding/json` marshal/unmarshal | `common.Marshal` and existing reusable body helpers | AGENTS forbids direct business-code JSON marshal/unmarshal. `[VERIFIED: AGENTS.md]` |
| Generic embedding response merger | Broad multi-provider batching abstraction | Narrow TEI cap enforcement, or explicit separate task if transparent sub-batching is required | Current phase only needs safe governed local TEI behavior. `[VERIFIED: 25-CONTEXT.md; relay/channel/openai/adaptor.go]` |

**Key insight:** The hard part is not detecting batch versus interactive; the current governor already does that. `[VERIFIED: service/embeddinggovernor/governor.go]` The phase risk is making the desired contract explicit at the right boundary while preventing arrays over `4` from reaching TEI unchecked. `[VERIFIED: .planning/REQUIREMENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; relay/channel/openai/adaptor.go]`

## Common Pitfalls

### Pitfall 1: Thinking Phase 25 Starts From Zero

**What goes wrong:** Planner duplicates Phase 20 work instead of tightening the threshold/flag/test contract. `[VERIFIED: 20-03-PLAN.md; 20-04-SUMMARY.md]`

**Why it happens:** Phase 20 already added metadata-only fields, header priority, split feedback, and relay wiring. `[VERIFIED: 20-03-PLAN.md; 20-04-SUMMARY.md]`

**How to avoid:** Plan small deltas around `defaultBatchInputCountThreshold`, an auto-workload config field, helper/test surface, and docs/smoke. `[VERIFIED: service/embeddinggovernor/governor.go; .planning/REQUIREMENTS.md]`

**Warning signs:** A plan proposes new queue infrastructure or edits provider catalog rows. `[VERIFIED: AGENTS.md; 25-CONTEXT.md]`

### Pitfall 2: Leaving Threshold `4` In Docs Or Code

**What goes wrong:** Arrays of two or three texts stay interactive, contradicting Phase 25. `[VERIFIED: .planning/REQUIREMENTS.md; service/embeddinggovernor/governor.go]`

**Why it happens:** Current defaults and docs still say `InputCount >= 4`. `[VERIFIED: service/embeddinggovernor/governor.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

**How to avoid:** Change code default to `2`, normalize invalid env to `2`, and update docs from `InputCount >= 4` to `InputCount >= 2`. `[VERIFIED: .planning/REQUIREMENTS.md]`

**Warning signs:** `TestLoadConfigNormalizesWorkloadMetadataThresholds` still expects `defaultBatchInputCountThreshold` with value `4`. `[VERIFIED: service/embeddinggovernor/governor_test.go]`

### Pitfall 3: Treating Header Override As Trusted Capacity Bypass

**What goes wrong:** A client could force `interactive` for a large array and bypass batch accounting. `[VERIFIED: service/embeddinggovernor/governor.go]`

**Why it happens:** Current valid `interactive`/`realtime` headers override metadata thresholds. `[VERIFIED: service/embeddinggovernor/governor.go]`

**How to avoid:** Preserve header priority for compatibility, but enforce the TEI input cap independently of workload class. `[VERIFIED: .planning/REQUIREMENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

**Warning signs:** Tests only check classifier result and do not check arrays over `4`. `[VERIFIED: service/embeddinggovernor/governor_test.go]`

### Pitfall 4: Shipping A Smoke That Still Targets `embo-01`

**What goes wrong:** The smoke validates an inactive/historical provider instead of `embedding-gte-v1`. `[VERIFIED: scripts/smoke-embeddings.py; 24-04-SUMMARY.md]`

**Why it happens:** Script defaults are `DEFAULT_MODEL = "embo-01"` and `DEFAULT_EXPECTED_DIM = 1536`. `[VERIFIED: scripts/smoke-embeddings.py]`

**How to avoid:** Update defaults or add a Phase 25 documented command that sets `ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1` and `ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768`. `[VERIFIED: scripts/smoke-embeddings.py; .planning/REQUIREMENTS.md]`

**Warning signs:** Smoke output mentions `embo-01`, dimension `1536`, or port `3001` during Phase 25 validation. `[VERIFIED: scripts/smoke-embeddings.py]`

## Code Examples

Verified patterns from project and official docs:

### Numeric Input Stats

```go
// Source: dto/embedding.go
func (r *EmbeddingRequest) GetInputStats() EmbeddingInputStats {
    if r == nil {
        return EmbeddingInputStats{}
    }
    inputs := r.ParseInput()
    stats := EmbeddingInputStats{InputCount: len(inputs)}
    for _, input := range inputs {
        stats.InputChars += utf8.RuneCountInString(input)
    }
    return stats
}
```

### Existing Classifier Priority

```go
// Source: service/embeddinggovernor/governor.go
workload := strings.ToLower(strings.TrimSpace(req.Workload))
if workload == "batch" || workload == "bulk" {
    return true
}
if workload == "interactive" || workload == "realtime" {
    return false
}
```

### Existing Gin Relay Test Seam

```go
// Source: relay/embedding_handler_test.go
recorder := httptest.NewRecorder()
c, _ := gin.CreateTestContext(recorder)
c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
c.Request.Header.Set("X-Embedding-Workload", "batch")
```

### Testify Pattern

```go
// Source: Context7 pkg.go.dev/github.com/stretchr/testify
require.NotNil(t, err)
assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
```

## State of the Art

| Old Approach | Current / Required Approach | When Changed | Impact |
|--------------|-----------------------------|--------------|--------|
| Python/model-detailed or sidecar ownership for embeddings | Go router -> `service/embeddinggovernor` -> TEI | Phase 20 and fork guardrail | Phase 25 must stay in Go. `[VERIFIED: AGENTS.md; 20-04-SUMMARY.md]` |
| Public batch alias | One public `embedding-gte-v1`, internal workload class | Phase 20/24 and Phase 25 constraints | Do not expose `embedding-gte-v1-batch`. `[VERIFIED: AGENTS.md; 24-04-SUMMARY.md; .planning/REQUIREMENTS.md]` |
| Header required for batch behavior | Header optional; router infers by metadata | Phase 25 requirement | Graphify/GBrain clients should not need `X-Embedding-Workload`. `[VERIFIED: 25-CONTEXT.md]` |
| `InputCount >= 4` batch threshold | `InputCount >= 2` for governed local model path | Phase 25 requirement | Code, tests, docs, and smoke examples must change together. `[VERIFIED: .planning/REQUIREMENTS.md; service/embeddinggovernor/governor.go]` |
| Smoke default `embo-01` / `1536` | Smoke target `embedding-gte-v1` / `768` | Phase 24 runtime baseline | Existing script must be updated or invoked with env overrides. `[VERIFIED: scripts/smoke-embeddings.py; 24-04-SUMMARY.md]` |

**Deprecated/outdated:**

- `DEFAULT_MODEL = "embo-01"` in `scripts/smoke-embeddings.py` is outdated for Phase 25 validation. `[VERIFIED: scripts/smoke-embeddings.py; 24-04-SUMMARY.md]`
- Manual docs stating `InputCount >= 4` as classifier threshold are outdated for Phase 25. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; .planning/REQUIREMENTS.md]`
- Any public `*-batch` alias remains out of scope and forbidden. `[VERIFIED: AGENTS.md; 25-CONTEXT.md]`

## Recommended Plan Shape

| Slice | Files | Actions | Verification |
|-------|-------|---------|--------------|
| 25-01 Governor classifier contract | `service/embeddinggovernor/governor.go`, `service/embeddinggovernor/governor_test.go` | Add `AutoWorkload` config defaulting true, env `EMBEDDING_GOVERNOR_AUTO_WORKLOAD`, default count threshold `2`, and helper surface equivalent to `IsGovernedModel` / `ClassifyWorkload`. `[VERIFIED: .planning/REQUIREMENTS.md; service/embeddinggovernor/governor.go]` | `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(Test.*Workload|Test.*Governed|TestLoadConfig.*)$' -count=1` |
| 25-02 Relay no-header and TEI cap | `relay/embedding_handler.go`, `relay/embedding_handler_test.go`, optionally `dto/embedding.go` | Add captured-governor tests for no-header string and no-header array; enforce arrays over `4` for governed TEI via bounded sub-batching or fail-closed validation. `[VERIFIED: relay/embedding_handler.go; relay/embedding_handler_test.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` | `/usr/local/go/bin/go test ./dto ./relay ./service/embeddinggovernor -run '^(TestEmbedding|TestWorkload|Test.*Batch.*Cap)' -count=1` |
| 25-03 Docs and smoke | `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`, `scripts/smoke-embeddings.py`, possibly `tests/test_clianything.py` | Update threshold docs, mark header optional/override-only, update smoke defaults or add env-driven no-header/array mode for `embedding-gte-v1` dim `768`. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; scripts/smoke-embeddings.py]` | `python3 -m py_compile scripts/smoke-embeddings.py`; authenticated smoke with token. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| none | All factual claims above are tied to current code, planning artifacts, Context7 docs, Graphify/GBrain, or runtime probes from this session. | n/a | n/a |

## Open Questions (RESOLVED)

1. **Should arrays over `4` succeed transparently or fail closed?**
   - What we know: Current relay/adaptor path does not split or cap arrays before OpenAI-compatible TEI dispatch. `[VERIFIED: relay/helper/valid_request.go; relay/channel/openai/adaptor.go]`
   - Resolution: Phase 25 plans choose fail-closed validation for governed `embedding-gte-v1` requests with more than `4` input items. This satisfies the TEI safety requirement without inventing a response recomposition path that the current relay does not already have. `[VERIFIED: 25-02-PLAN.md; 25-PATTERNS.md]`
   - Deferred: Transparent sub-batching remains out of scope unless a later phase explicitly designs ordered response merge and usage accounting. `[VERIFIED: relay/channel/openai/adaptor.go; 25-02-PLAN.md]`

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| `/usr/local/go/bin/go` | Go unit/integration tests | yes | `go1.25.1 linux/arm64` | Use absolute path; `go` is not on PATH in this shell. `[VERIFIED: environment audit]` |
| Python 3 | Smoke script syntax/runtime | yes | `Python 3.12.3` | none needed. `[VERIFIED: python3 --version]` |
| `bin/clianything` | Catalog/runtime read-only checks | yes | CLI help works | Use read-only `query`/`embeddings`; avoid write modes unless a plan explicitly requires them. `[VERIFIED: environment audit; router-ai-atius skill]` |
| Local router `127.0.0.1:3000` | Unauthenticated reachability check | yes | returned `401` without token | Authenticated smoke still needs token. `[VERIFIED: curl probe]` |
| `ATIUS_ROUTER_TOKEN` | Authenticated `/v1/embeddings` smoke | no | n/a | Planner must require ephemeral export from secure source before live smoke. `[VERIFIED: environment audit; scripts/smoke-embeddings.py]` |
| Bun | Frontend scripts if unexpectedly needed | yes | `1.3.14` | Not expected for Phase 25. `[VERIFIED: bun --version; AGENTS.md]` |
| Graphify | Mandatory GSD context loop | yes | graph fresh, commit-stale false | Rebuild only if later edits make it stale. `[VERIFIED: graphify status]` |

**Missing dependencies with no fallback:**

- `ATIUS_ROUTER_TOKEN` for authenticated live smoke is not present in this shell. `[VERIFIED: environment audit]`

**Missing dependencies with fallback:**

- `go` on PATH is missing, but `/usr/local/go/bin/go` is available and should be used in plans. `[VERIFIED: environment audit]`

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` plus Testify `v1.11.1`; Python syntax check for smoke script. `[VERIFIED: go.mod; Context7 /websites/pkg_go_dev_github_com_stretchr_testify]` |
| Config file | `go.mod`; no separate Go test config found. `[VERIFIED: rg --files]` |
| Quick run command | `/usr/local/go/bin/go test ./service/embeddinggovernor ./dto ./relay -run '^(TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestLoadConfigNormalizesWorkloadMetadataThresholds|TestLoadConfigUsesDailySafeDefaults|TestEmbeddingInputStatsNilInput|TestEmbeddingInputStatsCountsStringAndSlices|TestEmbeddingHelperPassesGovernorRequestMetadata)$' -count=1` |
| Full suite command | `/usr/local/go/bin/go test ./service/embeddinggovernor ./dto ./relay -count=1` plus `python3 -m py_compile scripts/smoke-embeddings.py` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| PHASE-25-GOVERNED-MODEL-SCOPE | `embedding-gte-v1` governed, unknown model no-op, no public batch alias | unit + catalog smoke | `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestAcquireNoopsForNonGovernedModel|TestLoadConfigUsesDailySafeDefaults)$' -count=1`; `bin/clianything query ...` | yes for service tests; catalog command exists. `[VERIFIED: service/embeddinggovernor/governor_test.go; clianything query]` |
| PHASE-25-AUTO-WORKLOAD-INFERENCE | No-header string -> interactive; no-header array length `2` -> batch; char threshold -> batch | unit | add/update tests in `service/embeddinggovernor/governor_test.go` | partial; threshold currently `4`. `[VERIFIED: service/embeddinggovernor/governor_test.go]` |
| PHASE-25-HEADER-OVERRIDE-COMPATIBILITY | `batch`/`bulk` and `interactive`/`realtime` override metadata | unit + relay | `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^TestWorkloadHeaderOverridesMetadataClassification$' -count=1`; add relay no-header/override tests | partial. `[VERIFIED: service/embeddinggovernor/governor_test.go; relay/embedding_handler_test.go]` |
| PHASE-25-TEI-BATCH-SAFETY | Arrays over `4` do not reach TEI unchecked | unit/integration | add relay or dto/governor cap test; exact command after implementation | no; Wave 0 gap. `[VERIFIED: relay/channel/openai/adaptor.go; relay/helper/valid_request.go]` |
| PHASE-25-CLIENT-SMOKE-VALIDATION | Authenticated no-header smoke validates `embedding-gte-v1` dimension `768` | smoke/manual live | `ATIUS_ROUTER_TOKEN=... ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py` | script exists, defaults stale. `[VERIFIED: scripts/smoke-embeddings.py; 24-04-SUMMARY.md]` |

### Sampling Rate

- **Per task commit:** run the focused package command for changed files. `[VERIFIED: existing Phase 20 plan pattern]`
- **Per wave merge:** `/usr/local/go/bin/go test ./service/embeddinggovernor ./dto ./relay -count=1` and `python3 -m py_compile scripts/smoke-embeddings.py`. `[VERIFIED: tests executed in this research]`
- **Phase gate:** authenticated `/v1/embeddings` smoke with `embedding-gte-v1` and `768` dimensions, no token printed. `[VERIFIED: .planning/REQUIREMENTS.md; scripts/smoke-embeddings.py]`

### Wave 0 Gaps

- [ ] `service/embeddinggovernor/governor_test.go` needs explicit tests for threshold `2` and `EMBEDDING_GOVERNOR_AUTO_WORKLOAD`. `[VERIFIED: service/embeddinggovernor/governor_test.go; .planning/REQUIREMENTS.md]`
- [ ] `relay/embedding_handler_test.go` needs no-header single-string and no-header array capture tests. `[VERIFIED: relay/embedding_handler_test.go; 25-CONTEXT.md]`
- [ ] `relay/embedding_handler.go` or a narrow helper needs cap/sub-batch enforcement for governed TEI arrays over `4`. `[VERIFIED: relay/embedding_handler.go; relay/channel/openai/adaptor.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`
- [ ] `scripts/smoke-embeddings.py` needs Phase 25 defaults or documented env mode for `embedding-gte-v1`, `768`, local/public base URL, and optional array/no-header smoke. `[VERIFIED: scripts/smoke-embeddings.py; .planning/REQUIREMENTS.md]`

### Validation Already Run During Research

```bash
/usr/local/go/bin/go test ./service/embeddinggovernor ./dto ./relay -run '^(TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestLoadConfigNormalizesWorkloadMetadataThresholds|TestLoadConfigUsesDailySafeDefaults|TestEmbeddingInputStatsNilInput|TestEmbeddingInputStatsCountsStringAndSlices|TestEmbeddingHelperPassesGovernorRequestMetadata)$' -count=1
python3 -m py_compile scripts/smoke-embeddings.py
env -u ATIUS_ROUTER_TOKEN python3 scripts/smoke-embeddings.py
```

Results: the focused Go tests passed for `service/embeddinggovernor`, `dto`, and `relay`; Python compile passed; no-token smoke exited `2` as designed. `[VERIFIED: command execution]`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | Phase does not change auth; live smoke must use existing bearer-token path without printing secrets. `[VERIFIED: scripts/smoke-embeddings.py; .planning/REQUIREMENTS.md]` |
| V3 Session Management | no | Phase does not change sessions. `[VERIFIED: 25-CONTEXT.md]` |
| V4 Access Control | partial | Public catalog shape and active model exposure must not add `*-batch` or hidden internal fields. `[VERIFIED: AGENTS.md; .planning/REQUIREMENTS.md]` |
| V5 Input Validation | yes | Validate and classify untrusted request headers/input metadata; enforce TEI max client batch size `4`. `[VERIFIED: service/embeddinggovernor/governor.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| V6 Cryptography | no | Phase does not add crypto. `[VERIFIED: 25-CONTEXT.md]` |

### Known Threat Patterns for Go Embedding Governor

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Header tampering to force `interactive` for large payloads | Tampering / DoS | Header remains override for compatibility, but TEI cap enforcement must be independent of workload class. `[VERIFIED: service/embeddinggovernor/governor.go; .planning/REQUIREMENTS.md]` |
| Large array forwarded to TEI above max client batch size | DoS | Add bounded sub-batching or fail-closed cap before upstream dispatch. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; relay/channel/openai/adaptor.go]` |
| Raw embedding text or secrets in governor telemetry | Information Disclosure | Keep governor request/snapshot metadata-only and reuse existing no-raw-text tests. `[VERIFIED: service/embeddinggovernor/governor_test.go; dto/embedding_test.go]` |
| Client `4xx` errors reducing concurrency for other users | DoS | Keep existing pressure classifier that ignores ordinary client errors for adaptive downscale. `[VERIFIED: service/embeddinggovernor/governor.go; service/embeddinggovernor/governor_test.go]` |
| Smoke leaking bearer token | Information Disclosure | Use `_scrub()` and never print env values; missing token exits `2`. `[VERIFIED: scripts/smoke-embeddings.py]` |

## Sources

### Primary (HIGH confidence)

- `AGENTS.md` - fork guardrails, test quality, Go-native embeddings constraints. `[VERIFIED: file read]`
- `.planning/phases/25-embedding-governor-auto-workload-inference/25-CONTEXT.md` - locked decisions, discretion, deferred ideas. `[VERIFIED: file read]`
- `.planning/REQUIREMENTS.md` - Phase 25 requirement IDs and acceptance contract. `[VERIFIED: file read]`
- `service/embeddinggovernor/governor.go` and `service/embeddinggovernor/governor_test.go` - current governor config, classifier, queue/concurrency, tests. `[VERIFIED: file read; go test]`
- `relay/embedding_handler.go`, `relay/embedding_handler_test.go`, `dto/embedding.go`, `dto/embedding_test.go` - relay/governor metadata boundary. `[VERIFIED: file read; go test]`
- `scripts/smoke-embeddings.py` and `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - runtime smoke and operator docs. `[VERIFIED: file read; py_compile]`
- `bin/clianything embeddings` and read-only SQL queries - current local catalog state for `embedding-gte-v1`. `[VERIFIED: command execution]`

### Secondary (MEDIUM confidence)

- Context7 `/gin-gonic/gin` - Gin `httptest` and `GetHeader` docs. `[CITED: https://github.com/gin-gonic/gin/blob/master/docs/doc.md]`
- Context7 `/websites/pkg_go_dev_github_com_stretchr_testify` - Testify `assert`/`require` docs. `[CITED: https://pkg.go.dev/github.com/stretchr/testify]`
- Context7 `/websites/pkg_go_dev_go1_25_3` - Go stdlib `net/http.Header.Get` and `textproto.MIMEHeader.Get` docs. `[CITED: https://pkg.go.dev/net/http@go1.25.3]`
- GBrain query snippets - prior operational logs for governed `embedding-gte-v1` and Graphify/GBrain cutover. `[VERIFIED: gbrain query]`
- Codex memory `MEMORY.md` - prior router-ai-atius embedding governor consolidation summary. `[VERIFIED: memory search]`

### Tertiary (LOW confidence)

- None used for decisions. `[VERIFIED: research process]`

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - versions verified from `go.mod`, local tool probes, and Context7 docs. `[VERIFIED: go.mod; environment audit; Context7]`
- Architecture: HIGH - based on current code, Phase 20 summaries, Phase 24 summary, and locked Phase 25 context. `[VERIFIED: codebase; 20-04-SUMMARY.md; 24-04-SUMMARY.md; 25-CONTEXT.md]`
- Pitfalls: HIGH - each pitfall maps to a current code/doc gap or locked requirement. `[VERIFIED: service/embeddinggovernor/governor.go; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; scripts/smoke-embeddings.py; .planning/REQUIREMENTS.md]`
- External docs: MEDIUM - GSD confidence classifier returned `MEDIUM` for Context7 verified provider. `[VERIFIED: classify-confidence context7 --verified]`

**Research date:** 2026-07-05
**Valid until:** 2026-08-04 for code/planning conclusions; re-run runtime catalog and token-gated smoke before execution because production catalog/env can drift. `[VERIFIED: runtime probe; .planning/STATE.md]`
