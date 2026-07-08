---
phase: 25
phase_slug: embedding-governor-auto-workload-inference
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-05
---

# Phase 25 Validation Strategy

## Validation Architecture

Phase 25 must prove five surfaces:

1. governed model scope stays limited to `embedding-gte-v1`;
2. workload classification is metadata-only and defaults to automatic inference;
3. explicit `X-Embedding-Workload` remains an override, not a requirement;
4. governed TEI requests cannot exceed the max client batch size of `4`;
5. docs and smoke tooling validate `embedding-gte-v1` at `768` dimensions without leaking tokens.

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + Testify; Python stdlib `unittest`/`py_compile` |
| **Config file** | `go.mod`; Python uses script-local stdlib only |
| **Quick run command** | `/usr/local/go/bin/go test ./service/embeddinggovernor ./dto ./relay -run '^(TestAcquireNoopsForNonGovernedModel|TestLoadConfigUsesDailySafeDefaults|TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestEmbeddingHelperPassesGovernorRequestMetadata|TestEmbeddingHelperRejectsGovernedInputAboveTEICap)$' -count=1` |
| **Full suite command** | `/usr/local/go/bin/go test ./service/embeddinggovernor ./dto ./relay -count=1 && python3 -m py_compile scripts/smoke-embeddings.py && python3 -m unittest tests.test_clianything.Phase19ProviderRoutingTests.test_smoke_embeddings_helpers_cover_payload_shape_and_redaction -v` |
| **Estimated runtime** | ~60 seconds |

## Sampling Rate

- **After every task commit:** run the task-specific `<automated>` command from the active plan.
- **After every plan wave:** run the full suite command above.
- **Before `$gsd-verify-work`:** full suite green plus conditional authenticated smoke when `ATIUS_ROUTER_TOKEN` is present.
- **Max feedback latency:** 90 seconds for automated checks.

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 25-01-01 | 01 | 1 | PHASE-25-GOVERNED-MODEL-SCOPE, PHASE-25-AUTO-WORKLOAD-INFERENCE, PHASE-25-HEADER-OVERRIDE-COMPATIBILITY | T-25-01 / T-25-02 / T-25-03 | Header-first, metadata-only classifier with default threshold `2` and no raw input in governor state | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestAcquireNoopsForNonGovernedModel|TestLoadConfigUsesDailySafeDefaults|TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestLoadConfigNormalizesWorkloadMetadataThresholds)$' -count=1` | yes | pending |
| 25-01-02 | 01 | 1 | PHASE-25-TEI-BATCH-SAFETY, PHASE-25-CLIENT-SMOKE-VALIDATION | T-25-02 / T-25-03 | Safe concurrency defaults and aggregate-only snapshots remain intact | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestLoadConfigUsesDailySafeDefaults|TestWorkloadHeaderOverridesMetadataClassification|TestSplitLatencyMetricsTrackInteractiveAndBatchSeparately|TestBatchLatencyDoesNotBlockInteractiveScaleUp|TestSnapshotContainsOnlyAggregateEmbeddingGovernorMetadata)$' -count=1` | yes | pending |
| 25-02-01 | 02 | 2 | PHASE-25-AUTO-WORKLOAD-INFERENCE, PHASE-25-HEADER-OVERRIDE-COMPATIBILITY, PHASE-25-CLIENT-SMOKE-VALIDATION | T-25-05 / T-25-06 | Relay passes only public model plus workload/count/chars metadata to governor | unit | `/usr/local/go/bin/go test ./relay -run '^TestEmbeddingHelperPassesGovernorRequestMetadata$' -count=1` | yes | pending |
| 25-02-02 | 02 | 2 | PHASE-25-GOVERNED-MODEL-SCOPE, PHASE-25-TEI-BATCH-SAFETY | T-25-04 / T-25-05 / T-25-06 | Governed `embedding-gte-v1` arrays over `4` fail closed before governor acquire/upstream dispatch | unit | `/usr/local/go/bin/go test ./relay -run '^(TestEmbeddingHelperPassesGovernorRequestMetadata|TestEmbeddingHelperRejectsGovernedInputAboveTEICap)$' -count=1 && /usr/local/go/bin/go test ./dto ./relay ./service/embeddinggovernor -count=1` | no for new cap test | pending |
| 25-03-01 | 03 | 3 | PHASE-25-CLIENT-SMOKE-VALIDATION, PHASE-25-AUTO-WORKLOAD-INFERENCE | T-25-07 | Smoke defaults use `embedding-gte-v1`, dimension `768`, no default workload header, and preserve token redaction | unit/script | `python3 -m py_compile scripts/smoke-embeddings.py && python3 -m unittest tests.test_clianything.Phase19ProviderRoutingTests.test_smoke_embeddings_helpers_cover_payload_shape_and_redaction -v && env -u ATIUS_ROUTER_TOKEN python3 scripts/smoke-embeddings.py; test "$?" -eq 2` | yes | pending |
| 25-03-02 | 03 | 3 | PHASE-25-GOVERNED-MODEL-SCOPE, PHASE-25-HEADER-OVERRIDE-COMPATIBILITY, PHASE-25-TEI-BATCH-SAFETY | T-25-08 | Docs forbid sidecar/model-detailed/public batch aliases, state optional header semantics, and avoid token literals | static docs | Individual positive `rg` checks from `25-03-PLAN.md` for each required doc fact, plus token-pattern negative grep from `25-03-PLAN.md` | yes | pending |

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.
- New tests are added in the same files as the implementation tasks:
  - `service/embeddinggovernor/governor_test.go`
  - `relay/embedding_handler_test.go`
  - `tests/test_clianything.py`

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Authenticated single-input smoke returns dimension `768` | PHASE-25-CLIENT-SMOKE-VALIDATION | Requires `ATIUS_ROUTER_TOKEN` from a secure runtime source | `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py` |
| Authenticated two-item array smoke returns two `768`-dim vectors without header | PHASE-25-AUTO-WORKLOAD-INFERENCE, PHASE-25-CLIENT-SMOKE-VALIDATION | Requires `ATIUS_ROUTER_TOKEN`; proves client-facing no-header behavior | `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE=array python3 scripts/smoke-embeddings.py` |

Do not print, echo, commit, or paste token values. If `ATIUS_ROUTER_TOKEN` is absent, record authenticated smoke as blocked and keep the no-token exit-2 check as the automated safety gate.

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or explicit manual setup.
- [x] Sampling continuity: no 3 consecutive tasks without automated verify.
- [x] Wave 0 covers all currently missing references.
- [x] No watch-mode flags.
- [x] Feedback latency target is below 90 seconds for automated checks.
- [x] `nyquist_compliant: true` set in frontmatter.

**Approval:** pending execution
