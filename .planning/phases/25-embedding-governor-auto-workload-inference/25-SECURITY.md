---
phase: 25
slug: embedding-governor-auto-workload-inference
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-05
---

# Phase 25 — Security

Per-phase security contract: threat register, accepted risks, and audit trail for `embedding-gte-v1` governor auto-workload inference.

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Client -> Go router `/v1/embeddings` | Public embedding requests enter the relay and are parsed into DTO metadata. | Bearer-authenticated request body; raw embedding input remains outside governor state. |
| Relay -> embedding governor | Relay asks the governor for a lease before dispatching governed local embeddings. | Public model alias, optional workload header, input count, input chars, channel metadata. |
| Go router -> local TEI upstream | Governed requests are forwarded to the local TEI service after relay safety checks. | Embedding request payload; governed arrays above 4 fail closed before upstream dispatch. |
| Operator shell -> smoke script | Operators run authenticated smokes with credentials from HashiCorp Vault. | `ATIUS_ROUTER_TOKEN` is process-local and must not be printed, logged, committed, or documented as a literal. |
| Repo docs -> operators | Manual communicates safe defaults and override semantics. | Operational instructions; no secret values. |

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-25-01 | Tampering | `(*Governor).ClassifyWorkload` | medium | mitigate | Header mapping is deterministic and covered by `TestWorkloadHeaderOverridesMetadataClassification`; invalid headers fall through to metadata inference. | closed |
| T-25-02 | Denial of Service | `Config.BatchInputCountThreshold` | medium | mitigate | Invalid/missing threshold normalizes to `2`; daily safe defaults preserve `MaxConcurrency=3` and `BatchConcurrency=1`. | closed |
| T-25-03 | Information Disclosure | `Request`, `Snapshot`, service tests | high | mitigate | Governor request/snapshot data remains aggregate metadata-only; snapshot privacy test rejects raw input, auth, token, and secret strings. | closed |
| T-25-04 | Denial of Service | `EmbeddingHelper` governed TEI cap | high | mitigate | Governed `embedding-gte-v1` requests with `InputCount > 4` are rejected before governor acquisition and upstream dispatch. | closed |
| T-25-05 | Tampering | `X-Embedding-Workload` override | medium | mitigate | Header remains classifier metadata only; TEI cap is enforced independently so `interactive` cannot bypass input-count safety. | closed |
| T-25-06 | Information Disclosure | relay errors and tests | high | mitigate | Relay cap/governor reject paths avoid raw input, bearer tokens, and serialized request bodies; focused relay tests cover this path. | closed |
| T-25-07 | Information Disclosure | `scripts/smoke-embeddings.py` | high | mitigate | `_scrub` is preserved, missing-token path exits before network, and authenticated smokes were run without printing token values. | closed |
| T-25-08 | Tampering | operator docs | medium | mitigate | Manual now documents optional override semantics, `InputCount >= 2`, no public batch alias, and arrays above 4 fail-closed. | closed |
| T-25-09 | Repudiation | smoke output | low | accept | Smoke output intentionally reports only model, type, dimension, row count, and mode; request identity remains in existing auth/logging outside this phase. | closed |
| T-25-SC | Tampering | package installs | low | accept | No npm, pip, or cargo install is part of this phase; Python smoke uses stdlib only. | closed |

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-25-01 | T-25-09 | Smoke output is intentionally minimal and operationally sufficient; request identity is handled by existing router auth logs. | gsd-secure-phase | 2026-07-05 |
| AR-25-02 | T-25-SC | No dependency installation is introduced by Phase 25. | gsd-secure-phase | 2026-07-05 |

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-05 | 10 | 10 | 0 | Codex / gsd-secure-phase |

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-05
