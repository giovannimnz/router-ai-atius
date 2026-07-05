---
phase: 25-embedding-governor-auto-workload-inference
plan: "01"
subsystem: infra
tags: [go, embeddings, governor, tei]
requires:
  - phase: phase-20-go-native-model-router
    provides: Metadata-only workload classification and split governor feedback
provides:
  - Explicit governed-model helpers for the local embedding governor
  - Auto-workload config and threshold-2 classifier contract
  - Service tests for header priority, metadata inference, and privacy-safe fallback
affects: [relay-embeddings, operator-docs, smoke-validation]
tech-stack:
  added: []
  patterns:
    - Export governor scope checks from the same config used by Acquire
    - Keep workload classification metadata-only and header-first
key-files:
  created: []
  modified:
    - service/embeddinggovernor/governor.go
    - service/embeddinggovernor/governor_test.go
key-decisions:
  - "EMBEDDING_GOVERNOR_AUTO_WORKLOAD defaults to true so unlabeled governed requests classify inside the router."
  - "The batch count threshold is now 2, while valid explicit headers still override metadata inference."
patterns-established:
  - "Relay-facing scope checks must use embeddinggovernor.IsGovernedModel instead of duplicating model lists."
  - "Disabling auto-workload skips metadata thresholds but preserves explicit workload headers and batch-model fallback."
requirements-completed:
  - PHASE-25-GOVERNED-MODEL-SCOPE
  - PHASE-25-AUTO-WORKLOAD-INFERENCE
  - PHASE-25-HEADER-OVERRIDE-COMPATIBILITY
  - PHASE-25-TEI-BATCH-SAFETY
  - PHASE-25-CLIENT-SMOKE-VALIDATION
coverage:
  - id: D1
    description: Explicit governed-model helpers and default scope for embedding-gte-v1
    requirement: PHASE-25-GOVERNED-MODEL-SCOPE
    verification:
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestAcquireNoopsForNonGovernedModel"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestIsGovernedModelMatchesDefaultScope"
        status: pass
    human_judgment: false
  - id: D2
    description: Header-first, metadata-only workload classification with threshold 2 and auto-workload toggle
    requirement: PHASE-25-AUTO-WORKLOAD-INFERENCE
    verification:
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestWorkloadHeaderOverridesMetadataClassification"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestAutoWorkloadDisabledFallsBackToHeadersAndBatchModels"
        status: pass
    human_judgment: false
  - id: D3
    description: Governor defaults preserve the Phase 20 safety envelope and aggregate-only snapshots
    requirement: PHASE-25-TEI-BATCH-SAFETY
    verification:
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestLoadConfigUsesDailySafeDefaults"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestSplitLatencyMetricsTrackInteractiveAndBatchSeparately"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestBatchLatencyDoesNotBlockInteractiveScaleUp"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestSnapshotContainsOnlyAggregateEmbeddingGovernorMetadata"
        status: pass
    human_judgment: false
duration: 14 min
completed: 2026-07-05
status: complete
---

# Phase 25 Plan 01: Governor service contract Summary

**The embedding governor now exposes explicit governed-model checks and auto-workload classification for `embedding-gte-v1` with a default batch threshold of 2.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-07-05T10:12:00Z
- **Completed:** 2026-07-05T10:26:12Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `AutoWorkload`, exported `IsGovernedModel`, and explicit `ClassifyWorkload` behavior to the governor contract.
- Reduced the no-header batch count threshold from 4 to 2 while preserving valid header override priority.
- Expanded governor tests to cover default scope, disabled auto-workload fallback, and privacy-safe aggregate behavior.

## Task Commits

Production work was committed in one atomic code commit for this plan:

1. **Plan implementation:** `feea79f0` (`feat(25-01): add governor auto workload contract`)

## Files Created/Modified

- `service/embeddinggovernor/governor.go` - adds `AutoWorkload`, exported scope/classifier helpers, and threshold-2 classifier behavior.
- `service/embeddinggovernor/governor_test.go` - covers default scope, header priority, disabled auto-workload fallback, and unchanged safety/privacy expectations.

## Decisions Made

- Default router behavior is authoritative for unlabeled governed requests via `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true`.
- Metadata thresholds remain local to the governor and never require raw embedding input text.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first pass of the new fallback test assumed metadata inference should stay interactive even when `BatchModels` was configured. The contract is the opposite: once auto-workload is disabled, `BatchModels` remains the final fallback. The test was corrected and the focused/package test suites passed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Ready for `25-02`. Relay code can now depend on `embeddinggovernor.IsGovernedModel(...)` and the threshold-2 classifier contract without duplicating governor scope logic.

## Verification Results

- `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestAcquireNoopsForNonGovernedModel|TestLoadConfigUsesDailySafeDefaults|TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestLoadConfigNormalizesWorkloadMetadataThresholds|TestAutoWorkloadDisabledFallsBackToHeadersAndBatchModels|TestIsGovernedModelMatchesDefaultScope)$' -count=1 -timeout 20s` - PASS
- `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestLoadConfigUsesDailySafeDefaults|TestWorkloadHeaderOverridesMetadataClassification|TestSplitLatencyMetricsTrackInteractiveAndBatchSeparately|TestBatchLatencyDoesNotBlockInteractiveScaleUp|TestSnapshotContainsOnlyAggregateEmbeddingGovernorMetadata)$' -count=1 -timeout 20s` - PASS
- `/usr/local/go/bin/go test ./service/embeddinggovernor -count=1 -timeout 30s` - PASS
- `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status` - PASS (`commit_stale=false` at current HEAD before this plan commit)

## Self-Check: PASSED

- Focused and package-level governor tests passed.
- No raw request text, token, or channel-secret metadata was introduced into governor state or snapshots.
- The automatic governor safety envelope remains bounded at `min=1`, `initial=2`, `max=3`, `batch_concurrency=1`.

---
*Phase: 25-embedding-governor-auto-workload-inference*
*Completed: 2026-07-05*
