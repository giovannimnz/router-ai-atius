---
phase: 25-embedding-governor-auto-workload-inference
plan: "02"
subsystem: api
tags: [go, relay, embeddings, governor, tei]
requires:
  - phase: 25-embedding-governor-auto-workload-inference
    provides: Explicit governor scope and classifier helpers from plan 25-01
provides:
  - Relay metadata capture for header and no-header embedding requests
  - Fail-closed max input cap of 4 for governed local TEI requests
  - Tests proving header override cannot bypass the governed TEI cap
affects: [smoke-validation, operator-docs, relay-embeddings]
tech-stack:
  added: []
  patterns:
    - Enforce governed TEI safety before request conversion and upstream dispatch
    - Use synthetic governor rejects in relay tests to avoid network dependency
key-files:
  created: []
  modified:
    - relay/embedding_handler.go
    - relay/embedding_handler_test.go
key-decisions:
  - "Governed local TEI requests with more than 4 input items now fail closed at the relay boundary."
  - "Unknown models are not subject to the TEI cap and continue through the existing no-op governor path."
patterns-established:
  - "Relay tests for governor metadata should stop at the governor hook instead of relying on upstream transport."
  - "The public model alias is the scope key for TEI cap enforcement, before any upstream mapping."
requirements-completed:
  - PHASE-25-GOVERNED-MODEL-SCOPE
  - PHASE-25-AUTO-WORKLOAD-INFERENCE
  - PHASE-25-HEADER-OVERRIDE-COMPATIBILITY
  - PHASE-25-TEI-BATCH-SAFETY
  - PHASE-25-CLIENT-SMOKE-VALIDATION
coverage:
  - id: D1
    description: Relay forwards public model, header metadata, input count, and character count to the governor for header and no-header requests
    requirement: PHASE-25-AUTO-WORKLOAD-INFERENCE
    verification:
      - kind: unit
        ref: "relay/embedding_handler_test.go#TestEmbeddingHelperPassesGovernorRequestMetadata"
        status: pass
      - kind: other
        ref: "timeout 20s /tmp/relay.test.bin -test.run '^TestEmbeddingHelperPassesGovernorRequestMetadata$' -test.v"
        status: pass
    human_judgment: false
  - id: D2
    description: Governed embedding-gte-v1 arrays above 4 fail closed before governor acquisition or upstream dispatch
    requirement: PHASE-25-TEI-BATCH-SAFETY
    verification:
      - kind: unit
        ref: "relay/embedding_handler_test.go#TestEmbeddingHelperRejectsGovernedInputAboveTEICap"
        status: pass
      - kind: other
        ref: "timeout 20s /tmp/relay.test.bin -test.run '^TestEmbeddingHelperRejectsGovernedInputAboveTEICap$' -test.v"
        status: pass
    human_judgment: false
  - id: D3
    description: DTO parsing and governor package behavior remain green alongside the relay cap change
    requirement: PHASE-25-CLIENT-SMOKE-VALIDATION
    verification:
      - kind: unit
        ref: "/usr/local/go/bin/go test ./dto ./service/embeddinggovernor -count=1 -timeout 60s"
        status: pass
    human_judgment: false
duration: 10 min
completed: 2026-07-05
status: complete
---

# Phase 25 Plan 02: Relay cap and metadata Summary

**The relay now sends no-header embedding metadata to the governor and rejects governed `embedding-gte-v1` arrays above 4 items before TEI dispatch.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-07-05T10:26:30Z
- **Completed:** 2026-07-05T10:36:37Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added a fail-closed `maxGovernedTEIInputCount = 4` check in `EmbeddingHelper` using the public model alias before upstream mapping.
- Expanded relay metadata coverage to prove header batch, no-header single string, and no-header array requests carry only workload/count/chars metadata to the governor.
- Added relay tests proving `interactive` cannot bypass the TEI cap and unknown models are not capped by the local-governed rule.

## Task Commits

Production work was committed in one atomic code commit for this plan:

1. **Plan implementation:** `7c0058d4` (`feat(25-02): enforce governed tei batch cap`)

## Files Created/Modified

- `relay/embedding_handler.go` - enforces the governed TEI cap before conversion/upstream dispatch.
- `relay/embedding_handler_test.go` - covers header/no-header metadata capture and fail-closed cap behavior.

## Decisions Made

- The TEI client batch-size invariant is enforced fail-closed in the router instead of inventing transparent sub-batching and response recomposition.
- Unknown models keep the existing no-op governor path so the TEI-specific cap stays narrowly scoped to governed local embeddings.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- In this environment, the `go test ./relay ...` wrapper hangs even for `-run '^$'`, but `go test -c ./relay` followed by executing `/tmp/relay.test.bin` runs the targeted tests normally. Validation used the compiled relay test binary plus package tests for `dto` and `service/embeddinggovernor`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Ready for `25-03`. Docs and smoke tooling can now describe the threshold-2 classifier and the fail-closed cap of 4 as the real runtime contract.

## Verification Results

- `timeout 20s /tmp/relay.test.bin -test.run '^TestEmbeddingHelperPassesGovernorRequestMetadata$' -test.v` - PASS
- `timeout 20s /tmp/relay.test.bin -test.run '^TestEmbeddingHelperRejectsGovernedInputAboveTEICap$' -test.v` - PASS
- `timeout 60s /tmp/relay.test.bin -test.run '^(TestEmbeddingHelperPassesGovernorRequestMetadata|TestEmbeddingHelperRejectsGovernedInputAboveTEICap)$' -test.v` - PASS
- `/usr/local/go/bin/go test ./dto ./service/embeddinggovernor -count=1 -timeout 60s` - PASS
- `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status` - AUTO-UPDATE RUNNING for HEAD `7c0058d4` at summary authoring time

## Self-Check: PASSED

- Governed arrays above 4 are rejected before the governor/upstream path.
- No raw embedding input text was added to governor metadata or relay errors.
- Header override remains metadata-only and cannot bypass TEI input-count safety.

---
*Phase: 25-embedding-governor-auto-workload-inference*
*Completed: 2026-07-05*
