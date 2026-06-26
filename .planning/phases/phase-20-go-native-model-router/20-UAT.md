---
status: testing
phase: 20-go-native-model-router
source: [20-VERIFICATION.md]
started: 2026-06-26T12:40:30Z
updated: 2026-06-26T12:40:30Z
---

## Current Test

number: 1
name: Authenticated local embeddings smoke
expected: |
  Rodar o smoke autenticado contra `http://127.0.0.1:3000/v1` com `ATIUS_ROUTER_TOKEN`.
  O comando deve sair com exit `0` e retornar embedding de dimensão `768`,
  validando o caminho Go `router -> governor -> TEI`.
awaiting: user response

## Tests

### 1. Authenticated local embeddings smoke
expected: `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-pt-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py` deve sair com `0` usando `ATIUS_ROUTER_TOKEN`
result: pending

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps

None yet.
