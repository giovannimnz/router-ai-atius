# Requirements — v2.18 Reranker reliability, observability, and readiness

**Status:** Active
**Created:** 2026-08-14
**Phase:** 33

## Requirement Status

- [ ] **PHASE-33-METRICS-OUTCOME-TAXONOMY** — Terminal outcome taxonomy, aggregate-only persistence, and cross-database compatibility.
- [ ] **PHASE-33-ROUTING-AVAILABILITY** — Pre-relay routing availability and exactly-once terminal accounting.
- [ ] **PHASE-33-WEIGHTED-DASHBOARD** — Weighted request-success/service-availability API and UI semantics.
- [ ] **PHASE-33-CHANNEL-READINESS** — Authenticated cache/ability/route readiness before smoke traffic.
- [ ] **PHASE-33-TEI-TIMEOUT-RETRY** — Bounded cancellation-aware TEI attempts and one safe retry.
- [ ] **PHASE-33-RERANK-BATCH-CONTRACT** — Twenty-document cap and deterministic client sub-batching contract.
- [ ] **PHASE-33-FORK-SYNC-PERSISTENCE** — Governed reranker preservation through source, image, and deploy gates.
- [ ] **PHASE-33-TEST-DOC-LIVE-EVIDENCE** — Deterministic tests, governed live evidence, synchronized records, and Git-enforced Graphify closeout.

## PHASE-33-METRICS-OUTCOME-TAXONOMY

- Preserve the existing request success rate as a request-outcome metric, not as the sole backend-health signal.
- Classify at least client/local validation, governor rejection, routing availability, transport/timeout, upstream HTTP failure, invalid upstream response, and success.
- Persist enough aggregate dimensions to compute those rates without storing prompts, documents, tokens, credentials, or full error bodies.
- Keep SQLite, MySQL, and PostgreSQL compatibility for schema/query changes.

## PHASE-33-ROUTING-AVAILABILITY

- Requests that fail because no eligible channel exists must contribute to routing availability even when failure occurs before the relay loop.
- One terminal request produces exactly one terminal outcome sample; retries must not double-count the user request.
- Metrics must distinguish routing failure from TEI/provider failure.

## PHASE-33-WEIGHTED-DASHBOARD

- The overview must not calculate overall health as a simple average of per-model percentages.
- Overall request success and service availability must be weighted from underlying counts.
- The UI must label request success separately from service availability and expose client/governor/routing/upstream classes without implying that a valid 4xx means TEI outage.
- Existing API consumers remain compatible or receive an explicit versioned/additive contract.

## PHASE-33-CHANNEL-READINESS

- Provisioning or re-enabling the reranker channel must force or await channel-cache synchronization and poll an authenticated readiness/ability condition before smoke traffic.
- Readiness must prove channel 10, `reranker-gte-multilingual-v1`, and its model ability are mutually consistent.
- Failure must be fail-closed, bounded, observable, and preserve rollback state.

## PHASE-33-TEI-TIMEOUT-RETRY

- Reranker outbound requests must inherit cancellation/deadline and have a bounded per-channel or per-request inference timeout even when global `RELAY_TIMEOUT=0`.
- Retry at most once only for explicitly retryable transport/5xx failures and never for client validation, governor rejection, or semantic 4xx.
- Timeout/retry behavior must not bypass governor accounting or duplicate settlement/metrics.

## PHASE-33-RERANK-BATCH-CONTRACT

- Preserve the TEI cap of 20 documents per rerank request.
- Return a deterministic client-facing 4xx for payloads above the cap and document client-side sub-batching of at most 20 items with stable index recomposition.
- Transparent router sub-batching is out of scope unless a separate design proves score/index semantics and billing/accounting correctness.

## PHASE-33-FORK-SYNC-PERSISTENCE

- Never promote an image that lacks `relay/channel/advancedcustom/tei_rerank.go` or regresses channel 10 / `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1,reranker-gte-multilingual-v1`.
- Preserve the live governed reranker implementation when incorporating upstream or mainline changes, including the production baseline introduced by `deb39c92d`.
- Extend `omni-srv-admin` protection to `dto/channel_settings_tei_rerank_test.go`, `k8s/router-ai-atius/configmap.yaml`, `relay/embedding_handler_test.go`, and `relay/rerank_handler.go`, in addition to already protected reranker/governor paths.
- Preflight must fail before cutover when protected markers, env aliases, focused tests, or candidate-image files are missing.

## PHASE-33-TEST-DOC-LIVE-EVIDENCE

- Add deterministic Go tests for outcome classification, single-count semantics, no-channel accounting, deadline propagation, retry policy, and the 20-document contract.
- Add frontend tests for weighted aggregation, labels, zero-data, and partial-series behavior.
- Run all heavy checks through `scripts/podman-admin.sh` within the 20% host CPU cap.
- Validate local and authenticated public paths after deployment while retaining rollback and without logging secrets or rerank documents.
- Update Router docs, `omni-srv-admin` runbooks/guards, GBrain, Obsidian, and Graphify readback.

## Out of Scope

- Removing or weakening the governor.
- Increasing/removing the TEI 20-document cap without a separately approved recomposition design.
- Treating client validation failure as provider downtime.
- Removing protected `new-api` or `QuantumNous` identity.
- Promoting k3s public cutover from Phases 29/30.

## Traceability

| Requirement | Phase | Status |
|---|---|---|
| PHASE-33-METRICS-OUTCOME-TAXONOMY | Phase 33 | Pending |
| PHASE-33-ROUTING-AVAILABILITY | Phase 33 | Pending |
| PHASE-33-WEIGHTED-DASHBOARD | Phase 33 | Pending |
| PHASE-33-CHANNEL-READINESS | Phase 33 | Pending |
| PHASE-33-TEI-TIMEOUT-RETRY | Phase 33 | Pending |
| PHASE-33-RERANK-BATCH-CONTRACT | Phase 33 | Pending |
| PHASE-33-FORK-SYNC-PERSISTENCE | Phase 33 | Pending |
| PHASE-33-TEST-DOC-LIVE-EVIDENCE | Phase 33 | Pending |
