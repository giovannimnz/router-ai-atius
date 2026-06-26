---
phase: phase-20-go-native-model-router
plan: "03"
subsystem: infra
tags: [go, embeddings, governor, tei, backpressure]

requires:
  - phase: phase-20-go-native-model-router
    provides: Phase 20 research and pattern map for Go-native governor evolution
provides:
  - Metadata-only workload classification thresholds in the Go embedding governor
  - Split interactive and batch latency feedback with aggregate snapshot metrics
  - Pressure-failure classification that protects TEI without punishing client errors
affects: [relay-embeddings, operator-docs, runtime-validation]

tech-stack:
  added: []
  patterns:
    - Governor request metadata carries only coarse numeric workload signals
    - Adaptive feedback keeps batch and interactive latency accounting separate
    - Pressure-only failure classification is applied before reducing concurrency

key-files:
  created: []
  modified:
    - service/embeddinggovernor/governor.go
    - service/embeddinggovernor/governor_test.go

key-decisions:
  - "Automatic governor defaults now stay at min=1, initial=2, max=3, with 4 kept out of automatic scaling."
  - "Batch classification uses workload header priority first, then metadata thresholds, then the configured batch-model fallback."
  - "Only pressure failures such as 429, 5xx, transport-equivalent failures and slow-request thresholds reduce concurrency; ordinary client 4xx errors do not."

patterns-established:
  - "Snapshot JSON exports only aggregate counters, durations and timestamps; no request text or secret-bearing fields appear."
  - "Interactive scale-up reads interactive latency, so catch-up/batch latency cannot poison normal interactive reopening by itself."

requirements-completed:
  - PHASE-20-PYTHON-MIDDLEWARE-REMOVAL
  - PHASE-20-UPSTREAM-SYNC-GUARD
  - PHASE-20-SDK-SMOKES

coverage:
  - id: D1
    description: Metadata-only workload classification for unlabeled embedding requests
    requirement: PHASE-20-PYTHON-MIDDLEWARE-REMOVAL
    verification:
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestWorkloadHeaderOverridesMetadataClassification"
        status: pass
      - kind: unit
        ref: "/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestLoadConfigNormalizesWorkloadMetadataThresholds|TestLoadConfigUsesDailySafeDefaults)$' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: Split interactive and batch adaptive feedback with aggregate-only snapshot metrics
    requirement: PHASE-20-SDK-SMOKES
    verification:
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
  - id: D3
    description: Pressure-only failure classification for adaptive concurrency
    requirement: PHASE-20-UPSTREAM-SYNC-GUARD
    verification:
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestStatusClassificationIgnoresClientErrors"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestStatusClassificationReducesOnPressureFailures"
        status: pass
      - kind: unit
        ref: "service/embeddinggovernor/governor_test.go#TestStatusClassificationKeepsSlowRequestsAsPressure"
        status: pass
      - kind: integration
        ref: "/usr/local/go/bin/go test ./service/embeddinggovernor ./relay -count=1"
        status: pass
    human_judgment: false

duration: 7 min
completed: 2026-06-26
status: complete
---

# Phase 20 Plan 03: Go Native Governor Core Summary

**O governor Go-native de embeddings agora classifica carga por metadata numérica, separa feedback interativo e batch e reage só a falhas reais de pressão sobre o TEI.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-06-26T08:38:28-03:00
- **Completed:** 2026-06-26T08:45:09-03:00
- **Tasks:** 3/3
- **Files modified:** 2

## Accomplishments

- Adicionou `InputCount` e `InputChars` ao `Request` do governor, com thresholds seguros para classificar carga batch sem carregar texto de embedding.
- Separou EWMA/counters de latência entre tráfego interativo e batch, mantendo o snapshot só com métricas agregadas.
- Introduziu classificação local de status para reduzir concorrência apenas em pressão real (`429`, `5xx`, falha de transporte e lentidão por workload), preservando tráfego saudável contra erros 4xx de cliente.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add metadata-only workload classification** - `40b0732e` (test), `fcfc7894` (feat)
2. **Task 2: Split interactive and batch feedback** - `669cf15c` (test), `31352b77` (feat)
3. **Task 3: Classify pressure failures without punishing client errors** - `5eacfd54` (test), `07a3fe35` (feat), `39472e3d` (refactor)

## Files Created/Modified

- `service/embeddinggovernor/governor.go` - adiciona metadata de workload, thresholds normalizados, métricas separadas e classificação de falhas de pressão.
- `service/embeddinggovernor/governor_test.go` - cobre classificação por metadata/header, métricas split, snapshot agregado e matriz de status/falhas.

## Decisions Made

- Defaults automáticos consolidados em `min=1`, `initial=2`, `max=3`, `batch_concurrency=1`, `batch_timeout=10m` e `batch_slow_request_duration=10m`.
- `batch`/`bulk` e `interactive`/`realtime` continuam com precedência explícita sobre metadata derivada.
- Erros 4xx de cliente não fecham o circuito adaptativo; só pressão real reduz para `min` e arma cooldown.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- O executor criou todos os commits de produção, mas não concluiu o fechamento com `20-03-SUMMARY.md`. O close-out documental foi reconciliado manualmente pelo orquestrador com base nos commits existentes e na revalidação dos testes do plano.

## Verification Results

- `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch|TestWorkloadHeaderOverridesMetadataClassification|TestLoadConfigNormalizesWorkloadMetadataThresholds|TestLoadConfigUsesDailySafeDefaults|TestSplitFeedbackKeepsBatchLatencyOutOfInteractiveScale|TestPressureFailuresReduceConcurrencyButClientErrorsDoNot)$' -count=1` - PASS
- `/usr/local/go/bin/go test ./service/embeddinggovernor -count=1` - PASS
- `/usr/local/go/bin/go test ./service/embeddinggovernor ./relay -count=1` - PASS

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Ready for Plan 20-04. The relay can now be wired to pass only metadata-derived embedding request stats into the governor without changing the adaptive semantics implemented here.

## Self-Check: PASSED

- All plan-scoped production commits exist in git history.
- All focused and package-level governor tests passed after reconciliation.
- No new runtime owner, sidecar or middleware dependency was introduced.
- Automatic adaptive cap remains bounded at `max=3`.

---
*Phase: phase-20-go-native-model-router*
*Completed: 2026-06-26*
