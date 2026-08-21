---
phase: 33
slug: reranker-reliability-observability-and-readiness
status: "Planning — final verification pending"
nyquist_compliant: true
wave_0_complete: false
created: 2026-08-14
---

# Phase 33 — Validation Strategy

Per-phase validation contract derived from the approved context, UI contract, and the Validation Architecture in 33-RESEARCH.md. Task and wave IDs are finalized by the planner.

## Test Infrastructure

| Property | Value |
|---|---|
| Backend framework | Go testing + testify |
| Frontend framework | Bun / node:test pure-unit pattern |
| Config files | go.mod; web/default/package.json |
| Quick backend | ./scripts/podman-admin.sh profile-run -- /usr/local/go/bin/go test ./pkg/perf_metrics ./middleware ./relay ./relay/channel ./relay/channel/advancedcustom -run Test.*(Outcome\|Routing\|Rerank\|Context\|Deadline\|Retry\|TEI) -count=1 |
| Quick frontend | ./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun test src/features/performance-metrics/lib/outcome-summary.test.ts' |
| Full backend | ./scripts/podman-admin.sh profile-run -- /usr/local/go/bin/go test ./pkg/perf_metrics ./model ./middleware ./controller ./relay ./relay/channel ./relay/channel/advancedcustom ./service/embeddinggovernor -count=1 |
| Full frontend | ./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun run typecheck && bun test src/features/performance-metrics' |
| CPU ceiling | Every heavy command uses scripts/podman-admin.sh; maximum total host CPU is 20% / 0.8 CPU |

## Sampling Rate

- After every task commit: run the quick command for the touched tier.
- After every plan wave: run the full focused backend/frontend suites relevant to that wave.
- Before verification: all focused suites, candidate readiness, synthetic rerank, rollback readback, local path, and authenticated public path must be green.
- Max feedback latency: 120 seconds for a quick focused suite; split a package set when necessary.

## Requirement Verification Map

| Requirement | Planned task | Test type | Automated command / evidence | File exists | Status |
|---|---|---|---|---|---|
| PHASE-33-METRICS-OUTCOME-TAXONOMY | 33-01 T1 (Wave 1); 33-05 T1-T3 (Wave 2) | unit + DB integration | Per-bucket v2 marker plus sum invariant; legacy-all-success/mixed windows; SQLite and disposable MySQL/PostgreSQL when available, otherwise explicit DryRun residual risk | ❌ W0 | ⬜ pending |
| PHASE-33-ROUTING-AVAILABILITY | 33-01 T2 (Wave 1); 33-06 T1-T2 (Wave 3) | middleware/controller integration | Governed Go tests for no-channel, exactly-once terminal sample and retry non-duplication | ❌ W0 | ⬜ pending |
| PHASE-33-WEIGHTED-DASHBOARD | 33-03 T1 (Wave 1); 33-05 T3, 33-10 T1-T2 (Wave 2); 33-11 T1-T3 (Wave 3); 33-14 T3 (Wave 7); 33-15 T2 (Wave 8) | pure unit + candidate/live integration + direct component render | Weighted helper and real panels; candidate-only all-v2 formulas/zero denominators/8-of-9; state-aware live partial-or-complete gate | ❌ W0 | ⬜ pending |
| PHASE-33-CHANNEL-READINESS | 33-02 T2 (Wave 1); 33-09 T1-T3 (Wave 2); 33-12 T1 (Wave 5); 33-14 T2 (Wave 7); 33-15 T1-T2 (Wave 8); 33-16 T1 (Wave 9) | service/router integration | Governed Go tests plus candidate, promotion, and final readback for channel/alias/ability/cache/route/authz | ❌ W0 | ⬜ pending |
| PHASE-33-TEI-TIMEOUT-RETRY | 33-02 T1 (Wave 1); 33-07 T1-T2 (Wave 2); 33-08 T1-T2 (Wave 4) | httptest.Server integration | Governed Go tests for parent cancellation, bounded attempt, fresh retry context, max one retry and governor lease accounting | 🟡 partial | ⬜ pending |
| PHASE-33-RERANK-BATCH-CONTRACT | 33-02 T1 (Wave 1); 33-08 T2 (Wave 4); 33-12 T1-T2 (Wave 5); 33-15 T2 (Wave 8); 33-16 T1 (Wave 9) | relay/adaptor unit + live readback | Governed Go tests and candidate/live/final smokes prove 20 accepted, 21 deterministic 4xx, and stable TEI indices | 🟡 partial | ⬜ pending |
| PHASE-33-FORK-SYNC-PERSISTENCE | 33-04 T1-T2 (Wave 1); 33-12 T1-T3 (Wave 5); 33-13 T1-T3 (Wave 6); 33-14 T1-T3 (Wave 7); 33-15 T1-T2 (Wave 8); 33-16 T1-T2 (Wave 9) | Python/shell preflight + canonical integration/readback | Omni tests/integration, protected markers/aliases, immutable candidate/rollback, final readback and cleanup | ❌ W0 | ⬜ pending |
| PHASE-33-TEST-DOC-LIVE-EVIDENCE | 33-01 through 33-04 (Wave 1); 33-12 T1-T3 (Wave 5); 33-14 T1-T3 (Wave 7); 33-15 T1-T2 (Wave 8); 33-16 T1-T2 (Wave 9) | sanitized smoke + API/UI + knowledge/Git lifecycle readback | Candidate/local/public probes, rollback, Router/Omni docs, GBrain, Obsidian, retained-worktree cleanup, fast post-commit PENDING invalidation, then synchronous terminal pre-push Graphify enforcement | ❌ W0 | ⬜ pending |

## Wave 0 Requirements

- [ ] 33-01 T1: expand/create pkg/perf_metrics/metrics_test.go and model/perf_metric_test.go for taxonomy, invariants, aggregates and additive persistence.
- [ ] 33-01 T2: add the nearest middleware/controller fixtures for no-channel and exactly-once terminal outcomes.
- [ ] 33-02 T2: add controller/channel_readiness_test.go plus route/authz assertions.
- [ ] 33-02 T1: expand relay/channel/api_request_test.go and relay/embedding_handler_test.go for cancellation, timeout, retry, governor and the document cap.
- [ ] 33-03 T1: add web/default/src/features/performance-metrics/lib/outcome-summary.test.ts.
- [ ] 33-11 T2-T3: add direct server-render fixtures importing both real dashboard views for every approved state and responsive/accessibility contract.
- [ ] 33-04 T1-T2: add omni-srv-admin fork-sync/release preflight tests for protected files, exact aliases, candidate labels/digest and fail-closed promotion.
- [ ] 33-03 T2 and 33-04 T2: add sanitized candidate/live smoke assertions without token, prompt, document or provider-body capture.

## Deterministic Cases

- Every origin/error mapping: local 400, governor reject, no-channel 503, transport, deadline, upstream 429/5xx, malformed response and success.
- Calling terminal finish twice produces one outcome; fail-first/succeed-second produces one terminal request outcome while attempt/governor accounting remains correct.
- Model volumes differ enough to prove weighted aggregate differs from simple average.
- Every contributing bucket must carry writer/schema v2 and satisfy its own outcome-sum invariant; every complete model row exposes the v2 marker and valid sum, while every incomplete row exposes valid partial reasons with null additive KPIs. Legacy-all-success, mixed legacy/v2 and mismatched-v2 windows stay partial.
- Zero denominator renders —; a complete zero class renders 0 / 0,00%; missing additive fields render Dados parciais.
- Both real panels are rendered under loading, partial, zero, empty, error and refetch fixtures with PT-BR, fixed outcome order, accessible names, mobile/dark classes and the 8/9 incident.
- Parent cancellation reaches httptest.Server; only an allowed retry receives a fresh, bounded context.
- Plan 33-07 recomputes `ceil_to_5(max(15, 2*p99+5, observed_max+5, slo+5))` from sanitized inputs; both computed/default values must equal it and remain at most `outer_deadline-5`.
- Readiness fixtures cover channel/alias/ability/route/cache/auth/timeout failures and successful sync.
- Fork-sync fixture requires all protected reranker paths, exact two governor aliases and canonical unbounded static ceilings.
- Plans 33-14/33-15/33-16 reject any Omni script path outside the retained clean worktree and require HEAD equal to the recorded origin/main integration commit.
- One post-promotion ERR/INT/TERM trap covers digest, runtime env, readiness, rerank 20/21, weighted API, two independent headless UI routes, quota and evidence; verified rollback still exits nonzero.
- Plan 33-14 proves all seven counters, per-row sums, model-to-overall totals, exact formulas, both null zero denominators, and 8/9 on disposable candidate-only SQLite with fail-closed cleanup/absence proofs; Plan 33-15 validates every child contract before accepting only a consistent partial or complete live 24-hour state.
- Overview and models routes each retain a distinct sanitized DOM, screenshot and trace and must prove mutually exclusive page headings plus their own semantic state.
- Rollback runs restore/digest/verify-live/rerank under controlled `set +e`; any failed rollback step records `rollback_status=failed` plus deterministic `failed_step` and retains the clean Omni worktree.
- Plans 33-14/15/16 bind `planning_identity` to the exact deterministic sorted immutable-plan manifest and bind `execution_identity` to the candidate/running OCI source-revision label. Plan 33-16 requires those exact JSON-authoritative values in Router docs, GBrain page/timeline and Obsidian; the retained Omni worktree must exist/registered/clean before removal. It disables `graphify.auto_update`, installs versioned hooks, makes post-commit only invalidate PASS to atomic mode-0600 HEAD-only PENDING/NO-GO in less than 10 seconds using bounded awaited foreground helpers, and makes pre-push exclusively derive the full fingerprint and index the relevant pushed HEAD from a detached clean temporary worktree before network mutation. Only ignored graph/status outputs are published and the dirty shared checkout is preserved.
- Plan 33-16's disposable lifecycle fixture must execute—not grep—the incomplete rc 3, wrong HEAD, commit-tree drift, wrapper failure, stale `built_at_commit`, empty query, cleanup failure, existing-hooksPath conflict, dirty-origin PASS, post-commit latency below 2 seconds, exact allowlist/bounded helper counts, HEAD-only PENDING mode/atomicity, zero Graphify/network/container/build calls, no surviving background/orphan helper, and pre-push missing/stale/current, foreground-wait, exact invocation-count, retry and failure cases.

## Runtime-Gated Automated Verifications

| Behavior | Requirement | Why runtime-gated | Automated instructions |
|---|---|---|---|
| Candidate image capability | PHASE-33-FORK-SYNC-PERSISTENCE | Requires built image and immutable digest | 33-14 validates retained clean Omni HEAD against `33-OMNI-INTEGRATION.json`, then directly inspects labels/source carriers/runtime env and invokes its deploy `--verify-candidate`; evidence-only grep cannot pass. |
| Functional candidate and production smoke | PHASE-33-TEST-DOC-LIVE-EVIDENCE | Requires authenticated runtime/TEI | 33-14/15 invoke the sanitized smoke against candidate/local/public paths; 33-16 repeats final live readback before synchronization. |
| Rollback | PHASE-33-TEST-DOC-LIVE-EVIDENCE | Runtime state transition | 33-15 wraps every post-promotion gate in one rollback trap, restores/verifies old digest plus functional rerank on any nonzero, records evidence, then exits nonzero. |
| Historical metric completeness and formula proof | PHASE-33-WEIGHTED-DASHBOARD | Pre-migration rows lack writer markers | 33-01/05 test bucket semantics; 33-14 proves formulas/zero denominators on isolated all-v2 candidate storage; 33-15 accepts consistent partial live history with null KPIs/`Dados parciais` or verifies exact complete formulas. |
| Terminal Graphify freshness | PHASE-33-TEST-DOC-LIVE-EVIDENCE | Canonical execution writes SUMMARY/STATE/ROADMAP/REQUIREMENTS, VERIFICATION/phase.complete, todo and PROJECT commits after Plan 33-16 task verification | Plan 33-16 installs a branch-agnostic post-commit invalidator for exact `docs(33-16)`/`docs(phase-33)` subjects and a range-aware synchronous pre-push closeout. Each matching commit atomically records mode-0600 HEAD-only PENDING/NO-GO in less than 10 seconds using only bounded awaited `git rev-parse`/`git show`/`mktemp`/`chmod`/`mv`/`rm`; the fixture requires less than 2 seconds, zero Graphify/network/container/build calls, and no surviving background/orphan helper. For missing/stale exact PASS, pre-push derives the full fingerprint and indexes the relevant pushed HEAD once in foreground through the CPU wrapper, permits at least 720 seconds, validates `built_at_commit`/three nonempty direct queries, publishes ignored output+mode-0600 state only, preserves dirty origin state, and blocks failure before network mutation. |

## Multi-Source Coverage Audit

The context source contains locked decisions as an ordered list without `D-XX` identifiers; `LOCK-01` through `LOCK-08` below are audit-local ordinal labels and do not invent or alter source decisions.

| Source | ID | Feature / requirement | Plan coverage | Status | Notes |
|---|---|---|---|---|---|
| GOAL | — | Operationally correct and actionable reranker health across outcome taxonomy, routing availability, weighted UI, readiness, bounded TEI behavior, batch contract, fork persistence and live evidence | 33-01 through 33-16 | COVERED | The 16-plan/9-wave DAG ends in candidate proof, transactional promotion, exact knowledge readback, fast Git invalidation, and a consumed terminal pre-push Graphify gate. |
| REQ | PHASE-33-METRICS-OUTCOME-TAXONOMY | Preserve request success while classifying and persisting non-sensitive terminal outcomes cross-DB | 33-01, 33-05, 33-06 | COVERED | Exact-once recorder, additive schema, aggregation and routing finalization. |
| REQ | PHASE-33-ROUTING-AVAILABILITY | Record no-channel before relay, once per terminal request, distinct from TEI/provider failure | 33-01, 33-06 | COVERED | Middleware and controller integration share one terminal recorder. |
| REQ | PHASE-33-WEIGHTED-DASHBOARD | Weighted request success/service availability with additive compatibility and explicit outcome classes | 33-03, 33-05, 33-10, 33-11, 33-14, 33-15 | COVERED | Pure/component tests plus isolated all-v2 candidate formulas and state-aware live partial-or-complete proof. |
| REQ | PHASE-33-CHANNEL-READINESS | Authenticated bounded readiness proving channel 10, alias, ability, route and cache consistency | 33-02, 33-09, 33-12, 33-14, 33-15, 33-16 | COVERED | Unit/integration contract is exercised against candidate, promoted runtime, and final readback. |
| REQ | PHASE-33-TEI-TIMEOUT-RETRY | Parent cancellation, bounded inference and at most one safe retry without accounting duplication | 33-02, 33-07, 33-08 | COVERED | Timeout is `ceil_to_5(max(15, 2*p99+5, observed_max+5, slo+5))` and must be at most `outer_deadline-5`. |
| REQ | PHASE-33-RERANK-BATCH-CONTRACT | Keep 20-document cap, deterministic 4xx for 21 and client sub-batching with stable index recomposition | 33-02, 33-08, 33-12, 33-15, 33-16 | COVERED | Router-side transparent sub-batching remains excluded per source. |
| REQ | PHASE-33-FORK-SYNC-PERSISTENCE | Preserve governed reranker, baseline, protected files, exact aliases and pre-cutover candidate gates | 33-04, 33-12, 33-13, 33-14, 33-15, 33-16 | COVERED | Clean pinned Omni worktree is retained through candidate/live verification and removed only after final readback. |
| REQ | PHASE-33-TEST-DOC-LIVE-EVIDENCE | Deterministic backend/frontend tests, CPU-governed checks, local/public proof, rollback and synchronized records | 33-01 through 33-04, 33-12, 33-14, 33-15, 33-16 | COVERED | Runtime evidence is sanitized; GBrain/Obsidian require exact readback; canonical commits invalidate stale proof quickly and pre-push synchronously establishes/enforces exact current Graphify state. |
| RESEARCH | R-01 | Closed terminal outcome with exact-once finalize and origin-aware classification | 33-01, 33-05, 33-06 | COVERED | Covers retry double-counting and invisible no-channel pitfalls. |
| RESEARCH | R-02 | Additive schema/API with per-bucket completeness, cross-DB aggregation, candidate formula proof and state-aware live validation | 33-01, 33-05, 33-10, 33-11, 33-14, 33-15 | COVERED | Legacy/mixed windows remain partial; formulas are proven separately on disposable all-v2 candidate storage. |
| RESEARCH | R-03 | Fresh bounded context per attempt and rerank-specific safe retry budget | 33-02, 33-07, 33-08 | COVERED | Governor lease, settlement and terminal metrics remain coherent. |
| RESEARCH | R-04 | Selection-based readiness after atomic cache synchronization | 33-02, 33-09, 33-12, 33-14, 33-15, 33-16 | COVERED | DB-only or sleep-based readiness cannot pass. |
| RESEARCH | R-05 | Candidate-before-promotion source/image/config gates with immutable rollback digest | 33-04, 33-12, 33-13, 33-14, 33-15, 33-16 | COVERED | The retained clean Omni commit is the only script authority and cleanup follows final readback. |
| RESEARCH | R-06 | Existing Go/Gin/GORM/React stack only; no external package installation | 33-01 through 33-16 | COVERED | Package legitimacy audit is not applicable and no install task exists. |
| RESEARCH | R-07 | Deterministic Nyquist cases and all heavy checks inside the 20% CPU wrapper | 33-01 through 33-16 | COVERED | Every task has an automated gate; heavy backend/frontend/image work uses project containment. |
| RESEARCH | R-08 | Authz, bounded polling, cache/race safety, sanitized evidence and governed operational sinks | 33-09, 33-12, 33-14, 33-15, 33-16 | COVERED | Threat models and final fail-closed readbacks cover the research security domain. |
| RESEARCH | R-09 | Disable asynchronous Graphify and consume canonical final commits through versioned hooks without disturbing a dirty shared checkout | 33-16 | COVERED | Config is in scope; post-commit only records fast PENDING, pre-push owns detached-worktree indexing and PASS, and ignored status/output, hooksPath transaction, lifecycle fixtures, 720-second caller allowance and push blocking are explicit. |
| CONTEXT | LOCK-01 | Preserve `request_success_rate`, never alone as TEI health | 33-01, 33-05, 33-10, 33-11 | COVERED | Request success and service availability are separate weighted signals. |
| CONTEXT | LOCK-02 | Explicit client, governor, routing, transport/timeout, upstream and success taxonomy | 33-01, 33-05, 33-06, 33-10, 33-11 | COVERED | Fixed terminal taxonomy propagates storage to UI. |
| CONTEXT | LOCK-03 | Measure no-channel before relay selection with one terminal sample | 33-01, 33-06 | COVERED | Exact-once request recorder spans middleware and relay. |
| CONTEXT | LOCK-04 | Compute overall indicators from weighted underlying counts | 33-03, 33-05, 33-10, 33-11, 33-14, 33-15 | COVERED | Simple mean is rejected by unit/render, isolated candidate, and state-aware live gates. |
| CONTEXT | LOCK-05 | Require cache/ability sync readiness before smoke | 33-02, 33-09, 33-12, 33-14, 33-15, 33-16 | COVERED | Authenticated readiness is bounded and fail-closed. |
| CONTEXT | LOCK-06 | Keep 20-document limit and document client sub-batching with stable indices | 33-02, 33-08, 33-12, 33-15, 33-16 | COVERED | Both 20-accepted and 21-rejected paths are live gates. |
| CONTEXT | LOCK-07 | Propagate context/deadline and allow one retry only for safe transport/5xx | 33-02, 33-07, 33-08 | COVERED | Computed/default values are independently recomputed from sanitized p99, observed maximum, SLO and outer deadline. |
| CONTEXT | LOCK-08 | Never promote without complete governed reranker, channel 10 and exact two aliases | 33-04, 33-12, 33-13, 33-14, 33-15, 33-16 | COVERED | Candidate source, label, env, readiness, transactional live checks and final readback are mandatory. |

Explicit exclusions are not gaps: removing/weakening the governor, transparent Router sub-batching without approved recomposition/billing design, Phases 29/30 public k3s cutover and protected upstream identity changes remain out of scope exactly as recorded in CONTEXT/RESEARCH.

## Validation Sign-Off

- [x] Planner replaced every assignment placeholder with concrete plan/task/wave IDs.
- [x] Every implementation task has an automated verification or an explicit Wave 0 dependency.
- [x] No three consecutive implementation tasks lack automated feedback.
- [x] All Wave 0 gaps are assigned before dependent implementation.
- [x] No watch-mode flags are used.
- [x] Heavy commands remain inside the 20% CPU wrapper.
- [x] Candidate promotion is fail-closed and rollback digest is retained.
- [x] nyquist_compliant: true reflects complete planned automated coverage; the plan checker still validates plan correctness.

**Approval:** pending plan checker
