---
phase: phase-20-go-native-model-router
plan: "04"
subsystem: api
tags: [go, embeddings, relay, governor, dto]

requires:
  - phase: phase-20-go-native-model-router
    provides: Governor core metadata thresholds and pressure classification from Plan 20-03
provides:
  - Metadata-only embedding input stats helper for relay-side workload signaling
  - Relay wiring that forwards workload, input count, and coarse input size to the Go governor
  - Deterministic relay coverage for governor request metadata and queue reject headers
affects: [relay-embeddings, embedding-governor, dto]

tech-stack:
  added: []
  patterns:
    - Metadata for embedding workload stays numeric-only and is derived from ParseInput before upstream conversion
    - Relay governor scope keeps the public model alias before upstream model mapping
    - Relay debug logging records converted request size instead of the raw JSON request body

key-files:
  created:
    - dto/embedding_test.go
    - relay/embedding_handler_test.go
  modified:
    - dto/embedding.go
    - relay/embedding_handler.go

key-decisions:
  - "EmbeddingHelper now derives governor stats from the original dto.EmbeddingRequest before upstream conversion."
  - "Only InputCount and InputChars cross the relay->governor boundary; raw embedding text and serialized request bodies do not."
  - "Queue rejects continue to surface the governor error code and Retry-After header through the existing relay error path."

patterns-established:
  - "Embedding input stats are computed once in dto/embedding.go and reused by the relay without adding public request fields."
  - "Relay governor acquire uses a package-local hook so tests can capture metadata without dispatching upstream traffic."

requirements-completed:
  - PHASE-20-PYTHON-MIDDLEWARE-REMOVAL
  - PHASE-20-UPSTREAM-SYNC-GUARD
  - PHASE-20-SDK-SMOKES

coverage:
  - id: D1
    description: Metadata-only embedding input stats helper returns input count and total character length for nil, string, []string, and []any inputs
    requirement: PHASE-20-PYTHON-MIDDLEWARE-REMOVAL
    verification:
      - kind: unit
        ref: "dto/embedding_test.go#TestEmbeddingInputStatsNilInput"
        status: pass
      - kind: unit
        ref: "dto/embedding_test.go#TestEmbeddingInputStatsCountsStringAndSlices"
        status: pass
      - kind: unit
        ref: "/usr/local/go/bin/go test ./dto -run 'Test.*Embedding.*Stats|Test.*Embedding.*Input' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: Embedding relay forwards public model, channel metadata, workload, input count, and input chars to the Go governor before upstream dispatch
    requirement: PHASE-20-UPSTREAM-SYNC-GUARD
    verification:
      - kind: unit
        ref: "relay/embedding_handler_test.go#TestEmbeddingHelperPassesGovernorRequestMetadata"
        status: pass
      - kind: unit
        ref: "/usr/local/go/bin/go test ./relay -run '^TestEmbeddingHelperPassesGovernorRequestMetadata$' -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: Relay and governor packages compile and pass package-level verification with the metadata contract wired through
    requirement: PHASE-20-SDK-SMOKES
    verification:
      - kind: integration
        ref: "/usr/local/go/bin/go test ./dto ./relay ./service/embeddinggovernor -count=1"
        status: pass
      - kind: integration
        ref: "/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1"
        status: pass
    human_judgment: false

duration: 6 min
completed: 2026-06-26
status: complete
---

# Phase 20 Plan 04: Embedding Relay Governor Wiring Summary

**O relay Go de `/v1/embeddings` agora envia apenas metadata numérica de carga ao governor e preserva o contrato público de erro/resposta existente.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-06-26T08:59:37-03:00
- **Completed:** 2026-06-26T09:05:11-03:00
- **Tasks:** 2/2
- **Files modified:** 4

## Accomplishments

- Adicionou um helper em `dto/embedding.go` que calcula `InputCount` e `InputChars` a partir de `ParseInput()` sem expor texto bruto.
- Ligou `relay/embedding_handler.go` ao governor com `Workload`, `InputCount` e `InputChars`, preservando `publicModelName` como chave de escopo antes do model mapping.
- Cobriu o caminho do relay com teste determinístico que captura o `embeddinggovernor.Request` e valida `Retry-After` + error code no reject path.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add embedding input stats helper** - `bc810fc4` (test), `1264f834` (feat)
2. **Task 2: Pass metadata and status through the embedding relay** - `fdedbbf4` (test), `708a0363` (feat)

## Files Created/Modified

- `dto/embedding.go` - adiciona `EmbeddingInputStats` e o helper `GetInputStats()` com contagem metadata-only.
- `dto/embedding_test.go` - cobre nil input, string, `[]string` e `[]any` sem usar texto real de usuário.
- `relay/embedding_handler.go` - passa `InputCount`/`InputChars` ao governor, usa hook local para testes e deixa de logar o JSON bruto.
- `relay/embedding_handler_test.go` - prova o request enviado ao governor e o contrato de `Retry-After` no reject path.

## Decisions Made

- O governor continua recebendo o alias público (`publicModelName`) antes de qualquer upstream model mapping.
- O boundary relay->governor carrega apenas `Workload`, `InputCount` e `InputChars`; nenhum texto de embedding ou body serializado cruza esse boundary.
- O log de debug do relay passa a registrar apenas o tamanho do request convertido, reduzindo risco de vazamento de conteúdo sensível.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- A retomada começou em um estado parcial ilegal: os commits do Task 1 já existiam (`bc810fc4`, `1264f834`), mas o close-out documental ainda não tinha sido criado. A execução foi retomada a partir do Task 2 sem reexecutar nem reverter o Task 1.

## Verification Results

- `/usr/local/go/bin/go test ./dto -run 'Test.*Embedding.*Stats|Test.*Embedding.*Input' -count=1` - PASS
- `/usr/local/go/bin/go test ./relay -run '^TestEmbeddingHelperPassesGovernorRequestMetadata$' -count=1` - PASS
- `/usr/local/go/bin/go test ./dto ./relay ./service/embeddinggovernor -count=1` - PASS
- `/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1` - PASS

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Ready for the next relay/governor follow-up. The Go-native embeddings path now emits workload metadata to the governor without reintroducing Python ownership, sidecars, or raw request telemetry.

## Self-Check: PASSED

- `bc810fc4`, `1264f834`, `fdedbbf4`, and `708a0363` exist in git history.
- `dto/embedding_test.go` and `relay/embedding_handler_test.go` exist on disk.
- All focused and plan-level verification commands listed above passed after Task 2 completion.

---
*Phase: phase-20-go-native-model-router*
*Completed: 2026-06-26*
