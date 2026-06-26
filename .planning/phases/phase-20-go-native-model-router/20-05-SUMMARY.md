---
phase: phase-20-go-native-model-router
plan: "05"
subsystem: embeddings
tags: [go, embeddings, governor, tei, docs]

requires:
  - phase: phase-20-go-native-model-router
    provides: Governor core defaults and relay metadata wiring from Plans 20-03 and 20-04
provides:
  - Disabled-by-default TEI health hysteresis inside the Go embedding governor
  - Deterministic health guardrail tests proving one bad sample never downscales by itself
  - Updated operator runbook for env defaults, smoke commands, Graphify freshness, and production monitor gates
affects: [service/embeddinggovernor, relay-embeddings, operator-docs]

tech-stack:
  added: []
  patterns:
    - Read-only TEI health probes debounce noisy samples through consecutive bad windows
    - Health-derived downscale stays separate from pressure-failure cooldown logic
    - Authenticated runtime smoke stays env-gated and never persists secrets

key-files:
  created:
    - .planning/phases/phase-20-go-native-model-router/20-05-SUMMARY.md
  modified:
    - service/embeddinggovernor/governor.go
    - service/embeddinggovernor/governor_test.go
    - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md

key-decisions:
  - "Health probing is effective only when both EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED=true and a valid EMBEDDING_GOVERNOR_HEALTH_PROBE_URL are set."
  - "A single timeout or slow health sample never reduces concurrency; the default threshold is 3 consecutive bad windows."
  - "Runtime embeddings smoke remains explicitly limited by ATIUS_ROUTER_TOKEN availability and is reported as skipped when the env var is absent."

duration: resumed close-out; task commits landed between 2026-06-26T09:14:15-03:00 and 2026-06-26T09:20:31-03:00
completed: 2026-06-26
status: complete
---

# Phase 20 Plan 05: TEI Health Guardrail Summary

**Read-only TEI health hysteresis for the Go embedding governor, with deterministic tests and an updated operational runbook.**

## Accomplishments

- Added optional TEI health probe state to `service/embeddinggovernor`, with safe normalization for timeout/interval/threshold defaults and aggregate-only snapshot fields.
- Ensured health is advisory and conservative: one bad sample cannot downscale or start cooldown; consecutive bad windows block scale-up and reduce concurrency gradually toward `min=1`.
- Updated `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` to match the final governor behavior, env vars, focused tests, embeddings smoke expectations, Graphify freshness gate, and controlled production monitor checks.

## Task Commits

1. **Task 1 RED: add failing health hysteresis tests** - `0c6cbe06`
2. **Task 1 GREEN: implement TEI health hysteresis guardrail** - `0e8cad5a`
3. **Task 2: update operator runbook and validation gates** - `5ae16f0b`

## Files Created/Modified

- `service/embeddinggovernor/governor.go` - adds health probe config/state, hysteresis handling, aggregate snapshot fields, and scale-up blocking during sustained bad windows.
- `service/embeddinggovernor/governor_test.go` - adds focused health guardrail tests plus normalization coverage.
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - documents metadata classification, split governor metrics, health env vars, Graphify freshness, smoke commands, and production monitor gates.
- `.planning/phases/phase-20-go-native-model-router/20-05-SUMMARY.md` - execution close-out for this plan.

## Verification Results

- `/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestHealthProbeDisabledByDefault|TestHealthHysteresisIgnoresSingleBadSample|TestHealthHysteresisReducesAfterConsecutiveBadWindows|TestHealthHysteresisHealthySampleResetsBadWindows)$' -count=1` - PASS
- `/usr/local/go/bin/go test ./service/embeddinggovernor -count=1` - PASS
- `/usr/local/go/bin/go test ./dto ./service/embeddinggovernor ./relay -count=1` - PASS
- `/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1` - PASS
- `node /home/ubuntu/.codex/gsd-core/bin/gsd-tools.cjs graphify status` after the docs commit - PASS (`stale=false`, `commit_stale=false`, `built_at_commit=5ae16f0`)
- `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-pt-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py` - SKIPPED (`ATIUS_ROUTER_TOKEN` ausente no ambiente)

## Decisions Made

- The TEI health signal remains disabled by default and read-only when enabled.
- A health-derived downscale never starts cooldown and never bypasses the protected envelope `min=1`, `initial=2`, `max=3`.
- Client `4xx` errors remain non-pressure outcomes; health hysteresis complements, but does not replace, the existing pressure-failure path.

## Deviations from Plan

### Environment limitations

- `ATIUS_ROUTER_TOKEN` was not present in the executor environment. The authenticated `embedding-pt-v1` dimension-768 smoke was not executed, and no token was invented, written, or persisted.

### Context differences

- `.planning/PROJECT.md` was referenced in the plan context but was absent on disk in this branch. Execution used `20-05-PLAN.md`, `20-03-SUMMARY.md`, `20-04-SUMMARY.md`, the current code, and the operational manual as the primary authorities, matching the user instruction to trust the plan and code over the historically inconsistent `STATE.md`.

## Threat/Guardrail Notes

- No Python relay, sidecar, or extra container was introduced.
- The health probe uses only Go stdlib HTTP/context primitives and exposes only aggregate state in snapshots/docs.
- Automatic concurrency remains bounded by `min=1`, `initial=2`, `max=3`; `4` stays manual/turbo only.

## Known Stubs

None.

## Threat Flags

None.

## Self-Check: PASSED

- `service/embeddinggovernor/governor.go`, `service/embeddinggovernor/governor_test.go`, `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`, and `.planning/phases/phase-20-go-native-model-router/20-05-SUMMARY.md` exist on disk.
- Commits `0c6cbe06`, `0e8cad5a`, and `5ae16f0b` exist in git history.
- The plan-scoped write set stayed within the files explicitly allowed by the user.
