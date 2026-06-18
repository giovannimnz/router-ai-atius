---
phase: phase-20-go-native-model-router
plan: "02"
subsystem: api
tags: [go, modelcatalog, openai-compatible, anthropic, clianything, graphify]

requires:
  - phase: phase-20-go-native-model-router
    provides: Go catalog/pricing foundation from 20-01
provides:
  - Go-owned enriched /v1/models catalog response
  - Root data-only OpenAI-compatible and Anthropic-selected model lists
  - Deterministic model ordering tests and OpenAPI/docs contract
affects: [model-routing, sdk-compatibility, clianything, runtime-docs]

tech-stack:
  added: []
  patterns:
    - Catalog DTO fields keep internal pricing provenance behind json:"-"
    - Controller builds /v1/models from service/modelcatalog after user/group/token filtering
    - Public model-list root payload contains only data

key-files:
  created:
    - service/modelcatalog/catalog.go
    - service/modelcatalog/catalog_test.go
    - bin/clianything
    - docs/CLIANYTHING.md
    - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md
    - runtime/model-detailed/model_detailed_fastapi.py
    - scripts/smoke-embeddings.py
    - tests/test_clianything.py
    - tools/clianything.py
    - tools/clianything_endpoints.json
  modified:
    - dto/pricing.go
    - controller/model.go
    - controller/model_list_test.go
    - router/relay-router.go
    - docs/openapi/relay.json

key-decisions:
  - "Use existing GET /v1/models as the only public Go catalog endpoint; no /internal/v1/models was added."
  - "Keep pricing_source and pricing_estimated internal with json:\"-\" while exposing zero public price values for missing prices."
  - "Use api_format=anthropic and Anthropic headers to select Anthropic-capable models while preserving the same root data-only payload."

patterns-established:
  - "Model list response roots use explicit gin.H{\"data\": ...} only for model-list modes."
  - "Model ordering is locked in controller tests using a representative MiniMax, DeepSeek, OpenAI/Codex and embeddings fixture."

requirements-completed:
  - PHASE-20-GRAPHIFY-GATE
  - PHASE-20-GO-ONLY-V1-MODELS
  - PHASE-20-AUTO-FORMAT-DETECTION

duration: 14 min
completed: 2026-06-18
---

# Phase 20 Plan 02: Go Native Model Router Summary

**Go-owned `/v1/models` catalog with enriched SDK-compatible items, root data-only payloads, Anthropic selection, deterministic ordering and CLI/OpenAPI contract coverage.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-06-18T05:20:33Z
- **Completed:** 2026-06-18T05:35:00Z
- **Tasks:** 6/6
- **Files modified/created:** 15

## Accomplishments

- Added `service/modelcatalog` as the Go projection layer for endpoint labels, public prices, internal pricing provenance and deterministic ordering.
- Refactored `controller.ListModels` so OpenAI-compatible and Anthropic-selected model-list responses return only top-level `data`.
- Added route detection for `/v1/models?api_format=anthropic` while preserving Anthropic header detection.
- Updated OpenAPI and operational docs to document Go-owned `/v1/models`, public price fields and ordering.
- Restored CLIAnything support files required for strict validation on this branch.

## Task Commits

1. **Task 1: Enforce Graphify as the Phase 20 planning gate** - no code diff; verified by Graphify status before and after execution.
2. **Task 2: Make Go catalog entries rich enough for `/v1/models`** - `7bf90de7` (feat)
3. **Task 3: Serve enriched OpenAI-compatible `/v1/models` from Go** - `a408c0be` (feat)
4. **Task 4: Produce Anthropic-selected model lists in Go** - `b04214ae` (feat)
5. **Task 5: Document the corrected `/v1/models` schema** - `b6f77b0f` (docs)
6. **Task 6: Lock post-execution validation for payload and order** - covered by `a408c0be` and `b6f77b0f`

## Files Created/Modified

- `dto/pricing.go` - Added enriched public model-list fields and internal-only pricing provenance fields.
- `service/modelcatalog/catalog.go` - Added catalog projection, endpoint labels/routes, public price calculation, Anthropic capability checks and deterministic ordering.
- `service/modelcatalog/catalog_test.go` - Proves pricing provenance remains internal and endpoint labels project correctly.
- `controller/model.go` - Builds `/v1/models` from Go catalog entries and returns root `data` only for model-list modes.
- `controller/model_list_test.go` - Locks payload shape, removed fields, item fields, representative ordering and Anthropic filtered behavior.
- `router/relay-router.go` - Routes `api_format=anthropic` to the Anthropic model-list builder.
- `docs/openapi/relay.json` - Documents the corrected Go-owned `/v1/models` schema.
- `docs/CLIANYTHING.md` and `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` - Document public contract and validation workflow.
- `bin/clianything`, `tools/clianything.py`, `tools/clianything_endpoints.json`, `tests/test_clianything.py`, `runtime/model-detailed/model_detailed_fastapi.py`, `scripts/smoke-embeddings.py` - Restored from `main` because this branch lacked required validation assets.

## Decisions Made

- No `/internal/v1/models` endpoint was added; `/v1/models` remains the public Go-owned catalog.
- `pricing_source` and `pricing_estimated` are retained only as internal Go fields with `json:"-"`.
- Anthropic model lists are filtered by Go endpoint metadata rather than a Python static table.
- Representative ordering is locked with an explicit fixture to prevent DB/map iteration from changing the public contract.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Restored missing validation assets from `main`**
- **Found during:** Tasks 5 and 6
- **Issue:** Current branch lacked `bin/clianything`, CLI docs, Python unit tests, runtime helper and CLI manifest files referenced by the plan and required verification commands.
- **Fix:** Restored the exact required files from `main` and updated only the `/v1/models` contract docs/schema needed by this plan.
- **Files modified:** `bin/clianything`, `docs/CLIANYTHING.md`, `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`, `runtime/model-detailed/model_detailed_fastapi.py`, `scripts/smoke-embeddings.py`, `tests/test_clianything.py`, `tools/clianything.py`, `tools/clianything_endpoints.json`
- **Verification:** `python3 -m unittest tests.test_clianything -v`; `bin/clianything coverage --strict`
- **Committed in:** `b6f77b0f`

**2. [Rule 3 - Blocking] Adapted restored catalog code to branch-local ratio API**
- **Found during:** Task 2
- **Issue:** The restored catalog helper referenced `ratio_setting.GetModelRatioInfo`, which does not exist on this branch.
- **Fix:** Used the existing `ratio_setting.GetModelRatio` return values for provenance classification.
- **Files modified:** `service/modelcatalog/catalog.go`
- **Verification:** `go test ./service/modelcatalog ./controller -run 'TestModelCatalog|TestListModels|TestBuildOpenAIModel|TestChannelOwnerName' -count=1`
- **Committed in:** `7bf90de7`

---

**Total deviations:** 2 auto-fixed (2 blocking).
**Impact on plan:** Both fixes were required to execute the stated plan on the current branch. No protected identifiers were modified or removed.

## Issues Encountered

- `go` was not on the shell PATH; verification was run with `PATH=/usr/local/go/bin:$PATH`.
- `bin/clianything status` returned exit code 0, but the `model-detailed` health row reported `fail` with `Connection reset by peer` twice. Backend, DB and `/v1/models` status rows were OK/expected. This is a runtime health issue outside the Go-owned `/v1/models` code path changed here.

## Verification Results

- `node "$HOME/.Codex/get-shit-done/bin/gsd-tools.cjs" graphify status` - PASS; final `stale=false`, `commit_stale=false`, built at `b6f77b0`.
- `go test ./service/modelcatalog ./controller -run 'TestModelCatalog|TestListModels|TestBuildOpenAIModel|TestChannelOwnerName' -count=1` - PASS with `PATH=/usr/local/go/bin:$PATH`.
- `go test ./controller -run 'TestListModels.*Order|TestListModels.*Payload|TestListModels.*Anthropic' -count=1` - PASS with `PATH=/usr/local/go/bin:$PATH`.
- `python3 -m unittest tests.test_clianything -v` - PASS; 33 tests.
- `python3 -m json.tool docs/openapi/relay.json >/dev/null` - PASS.
- `bin/clianything coverage --strict` - PASS; coverage 100.0%, docs 158, manifest 158, missing 0, extra 0, problems 0.
- `bin/clianything status` - COMMAND EXIT 0; backend/DB/v1-models OK, model-detailed row failed with connection reset.

## Known Stubs

None. Stub scan found only argparse/test-helper empty defaults in restored CLI/Python tests.

## Threat Flags

None. No new network endpoint, auth path, file access boundary or schema migration was introduced. `/v1/models` route behavior changed within the existing route.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Ready for Plan 20-03. Remaining runtime note: `model-detailed` health should be checked separately if Phase 20 later removes or retargets middleware traffic.

## Self-Check: PASSED

- All key created/modified files exist on disk.
- Commits `7bf90de7`, `a408c0be`, `b04214ae`, and `b6f77b0f` exist in git history.
- Graphify freshness was rebuilt after source/docs changes and reports `commit_stale=false`.

---
*Phase: phase-20-go-native-model-router*
*Completed: 2026-06-18*
