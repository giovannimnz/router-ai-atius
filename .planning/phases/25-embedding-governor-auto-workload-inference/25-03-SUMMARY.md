---
phase: 25-embedding-governor-auto-workload-inference
plan: "03"
subsystem: testing
tags: [python, docs, smoke, embeddings, operations]
requires:
  - phase: 25-embedding-governor-auto-workload-inference
    provides: Relay metadata capture and fail-closed TEI cap from plan 25-02
provides:
  - Smoke defaults for embedding-gte-v1 at 768 dimensions on the local Go router
  - Array-mode no-header smoke support for automatic workload inference
  - Operator docs for optional header override and fail-closed TEI cap
affects: [operator-docs, graphify, gbrain, smoke-validation]
tech-stack:
  added: []
  patterns:
    - Smoke tooling keeps token redaction and exits 2 before network when the token is absent
    - Manual docs describe override headers and automatic defaults separately
key-files:
  created: []
  modified:
    - scripts/smoke-embeddings.py
    - tests/test_clianything.py
    - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md
key-decisions:
  - "The smoke default request omits X-Embedding-Workload and uses array mode only through an explicit env toggle."
  - "Live authenticated smoke remains conditional on ATIUS_ROUTER_TOKEN and is blocked when the token is absent."
patterns-established:
  - "Routing smoke helpers should validate row count plus vector dimension when testing embedding arrays."
  - "Manual secure export examples should avoid literal ATIUS_ROUTER_TOKEN= snippets in docs."
requirements-completed:
  - PHASE-25-GOVERNED-MODEL-SCOPE
  - PHASE-25-AUTO-WORKLOAD-INFERENCE
  - PHASE-25-HEADER-OVERRIDE-COMPATIBILITY
  - PHASE-25-TEI-BATCH-SAFETY
  - PHASE-25-CLIENT-SMOKE-VALIDATION
coverage:
  - id: D1
    description: Smoke defaults target embedding-gte-v1 on the local Go router and support explicit array mode without adding the workload header
    requirement: PHASE-25-AUTO-WORKLOAD-INFERENCE
    verification:
      - kind: other
        ref: "python3 -m py_compile scripts/smoke-embeddings.py"
        status: pass
      - kind: unit
        ref: "tests/test_clianything.py#Phase19ProviderRoutingTests.test_smoke_embeddings_helpers_cover_payload_shape_and_redaction"
        status: pass
    human_judgment: false
  - id: D2
    description: Docs explain optional workload header semantics, threshold 2, and fail-closed arrays above 4 without leaking token literals
    requirement: PHASE-25-HEADER-OVERRIDE-COMPATIBILITY
    verification:
      - kind: other
        ref: "rg checks from 25-03-PLAN.md against docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md"
        status: pass
    human_judgment: false
  - id: D3
    description: Missing-token smoke exits before network and leaves authenticated live validation as an explicit manual gate
    requirement: PHASE-25-CLIENT-SMOKE-VALIDATION
    verification:
      - kind: other
        ref: "env -u ATIUS_ROUTER_TOKEN python3 scripts/smoke-embeddings.py; test \"$?\" -eq 2"
        status: pass
    human_judgment: true
    rationale: "Authenticated /v1/embeddings smoke requires ATIUS_ROUTER_TOKEN from a secure runtime source, and the token was absent in this shell."
duration: 12 min
completed: 2026-07-05
status: complete
---

# Phase 25 Plan 03: Smoke and docs Summary

**The embeddings smoke and operator manual now default to `embedding-gte-v1` at 768 dimensions, with no-header array validation and explicit fail-closed TEI cap guidance.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-05T10:29:00Z
- **Completed:** 2026-07-05T10:41:09Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Updated `scripts/smoke-embeddings.py` to default to `http://127.0.0.1:3000/v1`, `embedding-gte-v1`, expected dimension `768`, and optional array-mode requests without adding `X-Embedding-Workload`.
- Extended `tests/test_clianything.py` to cover the new default payload shape, array payload shape, and existing redaction behavior.
- Updated the governor section of the operator manual with threshold `2`, `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true`, explicit optional-header semantics, and fail-closed arrays above 4.

## Task Commits

Production work was committed in one atomic code commit for this plan:

1. **Plan implementation:** `dd7071f7` (`feat(25-03): update embedding smoke contract`)

## Files Created/Modified

- `scripts/smoke-embeddings.py` - switches defaults to the governed local embedding and adds array-mode validation.
- `tests/test_clianything.py` - covers default payload, array payload, dimensions, and scrubbed output.
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - documents the Phase 25 runtime contract and token-safe validation commands.

## Decisions Made

- The no-header path is the default client contract; operators keep `X-Embedding-Workload` only as a steering override.
- Authenticated live smoke remains a manual gate rather than faking success when the token is absent.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `ATIUS_ROUTER_TOKEN` was not exported in this shell, so the authenticated `/v1/embeddings` smoke stayed blocked by missing setup. The automated no-token exit-2 gate passed, which is the expected fallback for this environment.
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` already had an unrelated unstaged diff in the top router-docs section. Only the Phase 25 governor/doc hunk was committed for this plan.

## User Setup Required

External runtime validation still needs a secure token export for the live smoke:

- Export `ATIUS_ROUTER_TOKEN` from the secure runtime source into the current shell only.
- Run the single-item and array-mode smoke commands documented in `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`.

## Next Phase Readiness

Phase 25 execution is ready for final phase-level closeout after Graphify finishes refreshing against the latest HEAD and the manual authenticated smoke is either completed or explicitly accepted as blocked by missing token.

## Verification Results

- `python3 -m py_compile scripts/smoke-embeddings.py` - PASS
- `python3 -m unittest tests.test_clianything.Phase19ProviderRoutingTests.test_smoke_embeddings_helpers_cover_payload_shape_and_redaction -v` - PASS
- `env -u ATIUS_ROUTER_TOKEN python3 scripts/smoke-embeddings.py; test "$?" -eq 2` - PASS
- `rg` checks from `25-03-PLAN.md` against `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - PASS
- `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status` - AUTO-UPDATE RUNNING for HEAD `dd7071f7` at summary authoring time

## Self-Check: PASSED

- Smoke defaults and docs now target `embedding-gte-v1` and `768`.
- No public `*-batch` alias or token literal was introduced in the new Phase 25 docs/smoke contract.
- The no-token failure mode remains explicit and safe.

---
*Phase: 25-embedding-governor-auto-workload-inference*
*Completed: 2026-07-05*
