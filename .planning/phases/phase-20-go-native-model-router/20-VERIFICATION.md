---
phase: phase-20-go-native-model-router
verified: 2026-06-26T12:40:00Z
status: human_needed
score: 9/9 must-haves verified
overrides_applied: 0
---

# Phase 20: Embedding Governor Follow-up Verification Report

**Phase Goal:** Evolve the active Go-native embedding governor and relay path so local TEI embeddings stay fully Go-owned, use metadata-only workload classification, preserve safe adaptive limits, add a disabled-by-default read-only TEI health guardrail, and keep the operational manual aligned with the production baseline.
**Verified:** 2026-06-26T12:40:00Z
**Status:** human_needed
**Re-verification:** Yes - this verifies the follow-up execution delivered by plans `20-03`, `20-04`, and `20-05` after the earlier Go-native `/v1/models` cutover.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | The Go embedding governor classifies large unlabeled workloads by metadata only, without carrying raw embedding text. | VERIFIED | `service/embeddinggovernor.Request` now carries `InputCount` and `InputChars`; tests `TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch` and `TestWorkloadHeaderOverridesMetadataClassification` pass. |
| 2 | Explicit workload headers still override derived metadata classification. | VERIFIED | `isBatch` in `service/embeddinggovernor/governor.go` prioritizes `batch/bulk` and `interactive/realtime` before thresholds; focused tests pass. |
| 3 | Adaptive defaults remain in the protected envelope `min=1`, `initial=2`, `max=3`, with batch concurrency `1`. | VERIFIED | Defaults are encoded in `service/embeddinggovernor/governor.go`; `TestLoadConfigUsesDailySafeDefaults` passes. |
| 4 | Batch and interactive adaptive feedback are tracked separately so batch latency cannot poison interactive reopening by itself. | VERIFIED | Separate EWMA/counters exist in `Snapshot`; tests `TestSplitLatencyMetricsTrackInteractiveAndBatchSeparately` and `TestBatchLatencyDoesNotBlockInteractiveScaleUp` pass. |
| 5 | Pressure failures reduce concurrency, while ordinary client 4xx errors do not close the adaptive circuit. | VERIFIED | `finishOutcomeClientError` vs `finishOutcomePressure` is implemented and exercised by `TestStatusClassificationIgnoresClientErrors`, `TestStatusClassificationReducesOnPressureFailures`, and `TestStatusClassificationKeepsSlowRequestsAsPressure`. |
| 6 | The relay passes only metadata-derived embedding input stats into the governor and preserves the public model name as the governor scope key. | VERIFIED | `dto.EmbeddingRequest.GetInputStats()` returns numeric-only stats; `relay/embedding_handler.go` passes `InputCount`, `InputChars`, `Workload`, `ChannelID`, `ChannelName`, and `publicModelName`; `TestEmbeddingHelperPassesGovernorRequestMetadata` passes. |
| 7 | The optional TEI health guardrail is read-only, disabled by default, and cannot downscale after a single bad sample. | VERIFIED | Health probe config/state exists in `service/embeddinggovernor`; tests `TestHealthProbeDisabledByDefault`, `TestHealthHysteresisIgnoresSingleBadSample`, `TestHealthHysteresisReducesAfterConsecutiveBadWindows`, and `TestHealthHysteresisHealthySampleResetsBadWindows` pass. |
| 8 | The operational manual documents the final governor behavior, env vars, Graphify gate, and controlled production monitor flow. | VERIFIED | `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` documents metadata thresholds, split metrics, TEI health env vars/defaults, smoke commands, Graphify freshness, and monitor gates. |
| 9 | Graphify is fresh against the final code/docs state after the phase follow-up changes. | VERIFIED | `graphify status` reports `stale=false`, `commit_stale=false`, `built_at_commit=495f127`, `current_commit=495f127`. |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `service/embeddinggovernor/governor.go` | Metadata thresholds, split feedback, pressure classification, health hysteresis | VERIFIED | All four concerns are implemented in the governor core with aggregate-only snapshot fields. |
| `service/embeddinggovernor/governor_test.go` | Focused regression coverage for all new governor behavior | VERIFIED | Focused tests cover metadata classification, split metrics, status classification, and health hysteresis. |
| `dto/embedding.go` | Numeric-only embedding input stats helper | VERIFIED | `GetInputStats()` returns `InputCount` and `InputChars` only. |
| `dto/embedding_test.go` | Deterministic DTO stats tests | VERIFIED | Covers nil, string, `[]string`, and mixed `[]any` inputs. |
| `relay/embedding_handler.go` | Governor request wiring from embedding DTO metadata | VERIFIED | Acquires the governor with workload and numeric metadata before upstream dispatch. |
| `relay/embedding_handler_test.go` | Proof that relay passes governor request metadata | VERIFIED | `TestEmbeddingHelperPassesGovernorRequestMetadata` captures and asserts the governor request. |
| `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | Runbook aligned with final governor behavior | VERIFIED | Manual describes limits, envs, smokes, Graphify freshness, and controlled restart path. |
| `20-03/20-04/20-05-SUMMARY.md` | Plan close-outs with commits and verification trace | VERIFIED | All three summaries exist and match the shipped code paths. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Governor focused tests | `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch\|TestWorkloadHeaderOverridesMetadataClassification\|TestLoadConfigNormalizesWorkloadMetadataThresholds\|TestLoadConfigUsesDailySafeDefaults\|TestSplitLatencyMetricsTrackInteractiveAndBatchSeparately\|TestBatchLatencyDoesNotBlockInteractiveScaleUp\|TestSnapshotContainsOnlyAggregateEmbeddingGovernorMetadata\|TestStatusClassificationIgnoresClientErrors\|TestStatusClassificationReducesOnPressureFailures\|TestStatusClassificationKeepsSlowRequestsAsPressure\|TestHealthProbeDisabledByDefault\|TestHealthHysteresisIgnoresSingleBadSample\|TestHealthHysteresisReducesAfterConsecutiveBadWindows\|TestHealthHysteresisHealthySampleResetsBadWindows)$' -count=1` | Passou | PASS |
| DTO and relay follow-up | `/usr/local/go/bin/go test ./dto ./relay ./service/embeddinggovernor -count=1` | Passou | PASS |
| Broader Go gate | `/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1` | Passou | PASS |
| Runtime health gate | `bin/clianything status --strict` | Passou no runtime atual | PASS |
| Graphify freshness | `node /home/ubuntu/.codex/gsd-core/bin/gsd-tools.cjs graphify status` | `stale=false`, `commit_stale=false` | PASS |

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|---|---|---|---|
| `PHASE-20-PYTHON-MIDDLEWARE-REMOVAL` | Middleware must not own this embeddings/governor path | SATISFIED_FOR_SCOPE | All follow-up changes stay in Go `service/embeddinggovernor`, `dto`, `relay`, and docs. |
| `PHASE-20-UPSTREAM-SYNC-GUARD` | Preserve Go-native fork-owned paths and contracts | SATISFIED | Changes stay in protected paths and maintain `max=3`, metadata-only state, and Go-only ownership. |
| `PHASE-20-SDK-SMOKES` | Runtime validation must include embeddings smoke path | HUMAN_NEEDED | The code path is verified, but the authenticated local embeddings smoke still requires a human-run token-backed check. |
| `PHASE-20-GRAPHIFY-GATE` | Graphify must be fresh when enabled | SATISFIED_WITH_NOTE | Graphify is fresh now; however, `.planning/config.json` does not yet contain all requirement-listed boolean toggles. |
| `PHASE-20-CLI-DOCS-RUNTIME-PARITY` | Docs and operational runbook stay aligned with runtime | SATISFIED_WITH_NOTE | Manual and runtime gates are aligned; `bin/clianything coverage --strict` still fails in this checkout because the management docs tree is absent. |

### Human Verification Required

1. **Authenticated local embeddings smoke**
   - Command:
     `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-pt-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py`
   - Required env:
     `ATIUS_ROUTER_TOKEN`
   - Expected:
     Exit `0` and embedding dimension `768` through the Go path `router -> governor -> TEI`.

### Warnings

- `.planning/config.json` currently lacks `graphify.require_with_gsd`, `graphify.query_before_gsd`, and `graphify.rebuild_after_changes`, even though `REQUIREMENTS.md` still lists them as desired Graphify policy keys.
- `bin/clianything coverage --strict` is not green in this checkout because `docs/atius-router-docs/content/docs/en/api/management` is absent. This is a docs-artifact gap, not a governor/relay runtime regression.

### Gaps Summary

No code or wiring gaps were found in the phase follow-up implementation. The only remaining completion blocker is the human-run authenticated embeddings smoke.

---

_Verified: 2026-06-26T12:40:00Z_  
_Verifier: the agent (gsd-verifier), reconciled by execute-phase orchestrator_
