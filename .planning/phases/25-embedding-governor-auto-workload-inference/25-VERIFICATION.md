---
phase: 25-embedding-governor-auto-workload-inference
verified: 2026-07-05T14:04:46Z
status: passed
score: 7/7 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 25: embedding-governor-auto-workload-inference Verification Report

**Phase Goal:** Tornar `embedding-gte-v1` sempre governado no router e inferir automaticamente `batch` versus `interactive` quando o cliente nao enviar `X-Embedding-Workload`, preservando o header como override operacional, sem criar alias publico `*-batch` e mantendo o limite seguro do TEI.
**Verified:** 2026-07-05T14:04:46Z
**Status:** passed
**Re-verification:** Yes - after the initial verifier found a docs regression, the manual was restored and the Phase 25 doc gates passed.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | `embedding-gte-v1` is the default governed model and no public `*-batch` alias is added. | VERIFIED | `service/embeddinggovernor/governor.go` defaults `Models` to `embedding-gte-v1` and `BatchModels` to empty; `TestIsGovernedModelMatchesDefaultScope` and daily-default tests pass. |
| 2 | No-header governed requests infer workload from metadata only. | VERIFIED | `ClassifyWorkload` uses `InputCount >= 2` or `InputChars >= 12000` when auto-workload is enabled; governor request/snapshot fields remain aggregate metadata. |
| 3 | Explicit `X-Embedding-Workload` remains an operational override. | VERIFIED | Header classification is evaluated before metadata thresholds, and tests cover `batch`, `bulk`, `interactive`, and `realtime` behavior. |
| 4 | Relay passes public model alias plus input metadata to the governor. | VERIFIED | `relay/embedding_handler.go` uses the public model for governed scope/cap and sends `InputCount`/`InputChars`; relay metadata tests pass for header and no-header requests. |
| 5 | Governed TEI input arrays over 4 fail closed before governor acquisition/upstream dispatch. | VERIFIED | `maxGovernedTEIInputCount = 4` is enforced before conversion/acquire; relay cap test covers >4 and verifies `interactive` header cannot bypass it. |
| 6 | Smoke tooling defaults to the governed client contract and validates 768-dimension single/array responses without printing tokens. | VERIFIED | `fed6ea5d` fixed default `embedding-gte-v1` dimension resolution to 768. Python helper tests, no-token exit-2, authenticated single smoke, and authenticated array smoke all passed. |
| 7 | Operator docs describe the final Phase 25 client contract. | VERIFIED | `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` now documents optional/override-only `X-Embedding-Workload`, `InputCount >= 2`, auto-workload envs, no public batch alias, fail-closed arrays above 4, token-safe Vault usage, and array-mode smoke. |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `service/embeddinggovernor/governor.go` | Governed-model helpers, auto-workload config, threshold-2 classifier, safe defaults | VERIFIED | Implemented via `IsGovernedModel`, `AutoWorkload`, and `ClassifyWorkload`. |
| `service/embeddinggovernor/governor_test.go` | Service coverage for model scope, classifier, overrides, safe defaults, metadata privacy | VERIFIED | Focused governor tests pass. |
| `relay/embedding_handler.go` | Public model alias scope check, metadata-only governor request, fail-closed governed TEI cap of 4 | VERIFIED | Cap and governor metadata are wired before upstream dispatch. |
| `relay/embedding_handler_test.go` | Header/no-header metadata and >4 cap tests | VERIFIED | Focused relay tests pass through compiled test binary in this environment. |
| `scripts/smoke-embeddings.py` | Local Go router, `embedding-gte-v1`, default dimension 768, array mode, token redaction | VERIFIED | Defaults now resolve `embedding-gte-v1` to 768 without `ATIUS_ROUTER_EXPECT_EMBEDDING_DIM`. |
| `tests/test_clianything.py` | Smoke helper test coverage for payload shape, dimension defaults, redaction | VERIFIED | Focused unittest covers dimension defaults and redaction. |
| `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | Phase 25 envs, threshold, optional header semantics, cap, and validation commands | VERIFIED | Phase 25 `rg` doc gates passed after restoring the regressed block. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Smoke script syntax | `python3 -m py_compile scripts/smoke-embeddings.py` | exit 0 | PASS |
| Smoke helper unit coverage | `python3 -m unittest tests.test_clianything.Phase19ProviderRoutingTests.test_smoke_embeddings_helpers_cover_payload_shape_and_redaction -v` | 1 test passed | PASS |
| Missing-token safety | `env -u ATIUS_ROUTER_TOKEN python3 scripts/smoke-embeddings.py` | exit 2 before network with expected missing-token message | PASS |
| Authenticated single smoke | secure env via `/home/ubuntu/.local/bin/atius-vault-env router-ai-atius`, no dimension override, then `python3 scripts/smoke-embeddings.py` | `embedding-gte-v1`, dimension `768`, rows `1`, mode `single` | PASS |
| Authenticated array smoke | same secure env, `ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE=array`, no dimension override | `embedding-gte-v1`, dimension `768`, rows `2`, mode `array` | PASS |
| Phase 25 manual facts | `rg` checks from `25-03-PLAN.md` against `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | all positive checks passed; token-literal negative check passed | PASS |
| Graphify freshness | `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status` | `stale=false`, `commit_stale=false`, `built_at_commit=fed6ea5` before docs/report changes | PASS |

### Requirements Coverage

| Requirement | Status | Evidence |
|---|---|---|
| `PHASE-25-GOVERNED-MODEL-SCOPE` | SATISFIED | Single governed public alias remains `embedding-gte-v1`; no public batch alias was introduced. |
| `PHASE-25-AUTO-WORKLOAD-INFERENCE` | SATISFIED | Threshold-2 classifier, relay metadata wiring, and authenticated array smoke prove no-header batch inference. |
| `PHASE-25-HEADER-OVERRIDE-COMPATIBILITY` | SATISFIED | Header-first classifier tests pass and docs now state optional/override-only semantics. |
| `PHASE-25-TEI-BATCH-SAFETY` | SATISFIED | Safe concurrency envelope preserved; governed arrays above 4 fail closed and docs describe the cap. |
| `PHASE-25-CLIENT-SMOKE-VALIDATION` | SATISFIED | Smoke defaults, no-token safety, authenticated single/array smokes, docs gates, and token hygiene are all verified. |

### Human Verification Completed

1. **Authenticated single-input smoke**
   - Command shape: secure env export via `/home/ubuntu/.local/bin/atius-vault-env router-ai-atius`, then `python3 scripts/smoke-embeddings.py`.
   - Observed: `embeddings ok: model=embedding-gte-v1 type=openai dimension=768 rows=1 mode=single`.

2. **Authenticated two-item array smoke without workload header**
   - Command shape: same secure env export with `ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE=array`.
   - Observed: `embeddings ok: model=embedding-gte-v1 type=openai dimension=768 rows=2 mode=array`.

No token values, bearer values, API keys, or reconstructable secret values were written into this report.

### Warnings

- The first verifier pass found that `2f135dd1` had regressed the manual after `dd7071f7`. The manual block was restored and the exact `25-03-PLAN.md` doc checks now pass.
- A smoke bug was found during UAT closeout: `embedding-gte-v1` default dimension still fell through to 1536. Commit `fed6ea5d` fixed the default resolver and added regression coverage.
- `go test ./relay` hangs in this environment; focused relay validation is performed by compiling `./relay` and running the test binary.

### Gaps Summary

No open gaps remain for Phase 25.

---

_Verified: 2026-07-05T14:04:46Z_
_Verifier: gsd-verifier plus orchestrator re-check after closing the docs gap_
