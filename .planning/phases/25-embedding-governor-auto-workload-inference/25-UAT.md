---
status: complete
phase: 25-embedding-governor-auto-workload-inference
source:
  - 25-01-SUMMARY.md
  - 25-02-SUMMARY.md
  - 25-03-SUMMARY.md
started: 2026-07-05T12:36:01Z
updated: 2026-07-05T13:44:56Z
---

## Current Test

[testing complete]

## Tests

### 1. Explicit governed-model helpers and default scope for embedding-gte-v1
expected: Explicit governed-model helpers and default scope for embedding-gte-v1
result: pass
source: automated
coverage_id: D1

### 2. Header-first, metadata-only workload classification with threshold 2 and auto-workload toggle
expected: Header-first, metadata-only workload classification with threshold 2 and auto-workload toggle
result: pass
source: automated
coverage_id: D2

### 3. Governor defaults preserve the Phase 20 safety envelope and aggregate-only snapshots
expected: Governor defaults preserve the Phase 20 safety envelope and aggregate-only snapshots
result: pass
source: automated
coverage_id: D3

### 4. Relay forwards public model, header metadata, input count, and character count to the governor for header and no-header requests
expected: Relay forwards public model, header metadata, input count, and character count to the governor for header and no-header requests
result: pass
source: automated
coverage_id: D1

### 5. Governed embedding-gte-v1 arrays above 4 fail closed before governor acquisition or upstream dispatch
expected: Governed embedding-gte-v1 arrays above 4 fail closed before governor acquisition or upstream dispatch
result: pass
source: automated
coverage_id: D2

### 6. DTO parsing and governor package behavior remain green alongside the relay cap change
expected: DTO parsing and governor package behavior remain green alongside the relay cap change
result: pass
source: automated
coverage_id: D3

### 7. Smoke defaults target embedding-gte-v1 on the local Go router and support explicit array mode without adding the workload header
expected: Smoke defaults target embedding-gte-v1 on the local Go router and support explicit array mode without adding the workload header
result: pass
source: automated
coverage_id: D1

### 8. Docs explain optional workload header semantics, threshold 2, and fail-closed arrays above 4 without leaking token literals
expected: Docs explain optional workload header semantics, threshold 2, and fail-closed arrays above 4 without leaking token literals
result: pass
source: automated
coverage_id: D2

### 9. Missing-token smoke exits before network and leaves authenticated live validation as an explicit manual gate
expected: The safe no-token smoke exits before network, and authenticated /v1/embeddings validation remains an explicit manual gate until ATIUS_ROUTER_TOKEN is exported from a secure runtime source. With that token available, both the default single-input smoke and ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE=array smoke should return embeddings for embedding-gte-v1 with 768-dimensional vectors and no required X-Embedding-Workload header.
result: pass

## Summary

total: 9
passed: 9
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
