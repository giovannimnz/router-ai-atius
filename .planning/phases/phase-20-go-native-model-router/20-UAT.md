---
status: complete
phase: 20-go-native-model-router
source: [20-VERIFICATION.md]
started: 2026-06-26T12:40:30Z
updated: 2026-06-26T13:01:51Z
---

## Current Test

[testing complete]

## Tests

### 1. Authenticated local embeddings smoke
expected: `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-pt-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py` deve sair com `0` usando `ATIUS_ROUTER_TOKEN`
result: pass
observed: `embeddings ok: model=embedding-pt-v1 type=openai dimension=768`

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

None yet.
