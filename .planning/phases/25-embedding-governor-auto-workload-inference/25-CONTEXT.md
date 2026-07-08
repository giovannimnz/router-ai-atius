# Phase 25: embedding-governor-auto-workload-inference - Context

**Gathered:** 2026-07-05
**Status:** Ready for planning
**Source:** User prompt plus Codex thread `019f2dc6-858a-79e1-a78d-495ee5631235`

<domain>
## Phase Boundary

This phase changes the Go-native local embeddings path so `embedding-gte-v1` remains the only default public governed embedding model, while the router automatically classifies unlabeled requests as `batch` or `interactive`.

The key behavior change is for clients that do not send `X-Embedding-Workload`: Graphify, GBrain, and future clients should not need to remember the header to get safe batch behavior.
</domain>

<decisions>
## Implementation Decisions

- **D-01 — Governed model scope:** `embedding-gte-v1` must stay governed inside the Go router through `service/embeddinggovernor/` and `relay/embedding_handler.go`. `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` remains the default intended model scope. `EMBEDDING_GOVERNOR_BATCH_MODELS=` remains empty; do not introduce a public `embedding-gte-v1-batch` alias. `model != embedding-gte-v1` keeps current behavior and must not enter the governor unless explicitly configured in `EMBEDDING_GOVERNOR_MODELS`.
- **D-02 — Header override priority:** Explicit `X-Embedding-Workload` keeps priority over automatic inference. `batch` and existing `bulk` semantics classify as batch. `interactive` and `realtime` classify as interactive. Invalid or absent header falls back to router-side automatic inference when enabled.
- **D-03 — Automatic workload inference:** Add a testable helper in `service/embeddinggovernor/` for model scope and workload classification. Required helper surface should include an `IsGovernedModel("embedding-gte-v1")` equivalent and a `ClassifyWorkload(...)` equivalent. The helper must not retain or expose raw embedding text; use request metadata such as input count and character count. Without header, `input` array with at least 2 items must classify as `batch`. Without header, a single string classifies as `interactive` unless it crosses the configured character threshold. Add optional `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true` behavior, defaulting to the safe automatic behavior for governed models. Add or normalize `EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD=2`.
- **D-04 — TEI batch safety:** Preserve conservative batch concurrency (`EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=1`) and separate interactive feedback metrics. Preserve automatic concurrency bounds: min 1, initial 2, max 3; keep 4 reserved for explicit/manual turbo windows. Keep TEI max client batch size 4 as an execution invariant. If the current relay path can forward arrays larger than 4 to TEI without splitting, the plan must include a bounded sub-batch strategy or an explicit validation that the existing path already enforces it.
- **D-05 — Validation and smoke:** Unit tests must prove `model=embedding-gte-v1`, `input="texto"`, no header -> `interactive`; `model=embedding-gte-v1`, `input=["a","b"]`, no header -> `batch`; header `batch` forces batch for a single string; header `interactive` forces interactive for a small array; unknown model does not enter the governor; and batch larger than 4 respects the TEI sub-batch/cap contract. Relay tests must prove the classification happens before governor acquisition or that the request passed to the governor already carries the resolved workload. Live smoke after implementation must hit authenticated `/v1/embeddings` for `embedding-gte-v1` and verify dimensions `768`, without printing tokens.
- **DX-01 — Agent discretion:** Exact helper names and signatures may vary if they stay testable, do not leak raw input, and keep existing project style. Exact docs placement may use `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`, `docs/CLIANYTHING.md`, or a narrower runbook section if the planner finds a better canonical doc.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Runtime and governor
- `AGENTS.md` - fork safety rules for Go-native routing and protected customizations.
- `relay/embedding_handler.go` - `/v1/embeddings` relay path and governor acquisition point.
- `service/embeddinggovernor/governor.go` - governor config, model scope, queueing, batch classification, metrics.
- `service/embeddinggovernor/governor_test.go` - current governor behavior tests and safe defaults.
- `relay/embedding_handler_test.go` - relay metadata test seam for captured governor requests.
- `dto/embedding.go` - embedding input parsing and safe metadata extraction.

### Operational contract
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - production governor env, smokes, and validation commands.
- `scripts/smoke-embeddings.py` - authenticated `/v1/embeddings` smoke behavior; requires token env and must not print secrets.
- `.planning/phases/phase-20-go-native-model-router/20-03-PLAN.md` - prior governor core plan.
- `.planning/phases/phase-20-go-native-model-router/20-04-SUMMARY.md` - prior relay/governor wiring result.
- `.planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/24-04-SUMMARY.md` - current runtime validation baseline.
</canonical_refs>

<specifics>
## Specific Ideas

- The router should become the source of truth for workload inference so Graphify/GBrain do not need to supply `X-Embedding-Workload`.
- Header override remains useful for operational steering and emergency forcing.
- The implementation should make the desired public contract explicit in code/tests/docs rather than relying on hidden threshold behavior.
</specifics>

<deferred>
## Deferred Ideas

- Do not implement semantic Graphify indexing in this phase.
- Do not change the public model catalog shape.
- Do not activate Codex/OpenAI `text-embedding-3-*` embeddings.
- Do not create a Python/model-detailed sidecar or any extra container.
</deferred>

---

*Phase: 25-embedding-governor-auto-workload-inference*
*Context gathered: 2026-07-05 via Codex plan-phase orchestration*
