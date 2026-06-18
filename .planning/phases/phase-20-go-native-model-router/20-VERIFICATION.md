---
phase: phase-20-go-native-model-router
verified: 2026-06-18T05:50:04Z
status: passed
score: 15/15 must-haves verified
overrides_applied: 0
---

# Phase 20: Go Native Model Router Verification Report

**Phase Goal:** Go-only model catalog and middleware removal path, with corrected Phase 20.2 target: Go-owned enriched `/v1/models`, SDK-compatible root data-only payload, Anthropic-selected Go model list when `api_format=anthropic` or Anthropic headers are present, no `/internal/v1/models` canonical route, deterministic ordering, docs/schema/tests, Graphify fresh.
**Verified:** 2026-06-18T05:50:04Z
**Status:** passed
**Re-verification:** No - initial verification of the corrected Phase 20.1/20.2 contract. An older Wave 0 `VERIFICATION.md` existed, but it had no structured `gaps:` frontmatter and predated the Go-native implementation.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A canonical Go catalog builder exists and derives endpoint labels plus pricing provenance from current pricing data. | VERIFIED | `service/modelcatalog/catalog.go` defines endpoint labels, provenance and `BuildCatalogEntry`; `controller/model.go` calls `model.GetPricing()` and `modelcatalog.BuildCatalogEntry`. |
| 2 | Missing pricing is represented explicitly as fallback. | VERIFIED | `BuildCatalogEntryForModel` sets `PricingSource: "missing"`, `PricingEstimated: true`, zero `InputPrice`/`OutputPrice`, and a zero public `pricing` object. |
| 3 | Tiered billing metadata survives catalog projection. | VERIFIED | `BuildCatalogEntry` copies `BillingMode`, `BillingExpr`, and `PricingVersion`; controller tests cover tiered billing visibility. |
| 4 | `GET /v1/models` is built from the Go catalog path and no Python enrichment is required for model-list metadata. | VERIFIED | `ListModels` builds `catalogEntriesForModels`, then emits OpenAI/Anthropic payloads from those entries. No Python call exists in `controller/model.go` or `router/relay-router.go`. |
| 5 | No `/internal/v1/models` route is added as the canonical catalog source. | VERIFIED | Grep of `router/relay-router.go` and `controller/model.go` found no `/internal/v1/models`; route registration is only `/v1/models`. Legacy references remain in middleware/backups, not as Go canonical routes. |
| 6 | OpenAI-compatible model list remains SDK-compatible by default. | VERIFIED | Default branch returns `{"data": []dto.OpenAIModels}`; model items keep `id`, `object: "model"`, `created`, and `owned_by`. |
| 7 | Public `/v1/models` root payload is `{"data":[...]}` only for model-list modes. | VERIFIED | OpenAI and Anthropic branches return only `gin.H{"data": ...}`; tests assert top-level keys are exactly `data`. |
| 8 | Public `/v1/models` does not expose `pricing_source`, `pricing_estimated`, top-level `object`, or top-level `success`. | VERIFIED | DTO fields use `json:"-"`; controller tests unmarshal raw maps and assert absence of these fields. |
| 9 | Public `/v1/models` does not expose top-level Anthropic pagination keys. | VERIFIED | Anthropic tests assert absence of `first_id`, `last_id`, and `has_more`, including the empty-list case. |
| 10 | Public `/v1/models` keeps model-level fields not explicitly removed. | VERIFIED | `buildOpenAIModelFromCatalog` copies stable enriched fields from catalog entries while preserving base OpenAI model fields. |
| 11 | Public `/v1/models` order is text models first, embeddings after, with fixed provider grouping. | VERIFIED | `SortEntries` uses category/provider ranking; `TestListModelsRepresentativeOrder` locks the exact representative order. |
| 12 | Public `/v1/models` provider groups are sorted most advanced/recent/capable first. | VERIFIED | `compareModels` ranks by known order, category, provider, version token and capacity; representative order test covers the target fixture. |
| 13 | Anthropic model list is produced by Go when `api_format=anthropic` or Anthropic headers are present. | VERIFIED | `router/relay-router.go` routes query/header intent to `controller.ListModels(...ChannelTypeAnthropic...)`; controller filters by Go catalog endpoint metadata. |
| 14 | Missing prices remain visible through zero pricing values without exposing public provenance or estimated flags. | VERIFIED | Zero-price fallback is public; internal provenance fields are hidden with `json:"-"` and tests assert they do not serialize. |
| 15 | Graphify status is fresh before and after the plan changes. | VERIFIED | `graphify status` reports `stale=false`, `commit_stale=false`, `built_at_commit=c285f97`, `current_commit=c285f97`. |

**Score:** 15/15 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `controller/model.go` | Go-owned `/v1/models` response shapes and catalog wiring | VERIFIED | `ListModels` filters visible models, builds catalog entries, and emits root `data` only for OpenAI/Anthropic model-list modes. |
| `controller/model_list_test.go` | Contract tests for payload, order, Anthropic filtering and empty list | VERIFIED | Tests assert exact root keys, absent fields, representative order, Anthropic order and empty Anthropic payload. |
| `service/modelcatalog/catalog.go` | Canonical model metadata projection, pricing provenance and deterministic sorting | VERIFIED | Contains endpoint labels, owner resolution, pricing provenance, public price projection, Anthropic capability and sort logic. |
| `service/modelcatalog/catalog_test.go` | Catalog serialization/provenance test | VERIFIED | Covers missing pricing provenance as internal-only and endpoint label serialization. Note: narrower than the original 20-01 task text, but controller tests cover the public contract. |
| `dto/pricing.go` | Public model-list DTOs and internal-only provenance fields | VERIFIED | `OpenAIModels`, `AnthropicModel`, `ModelCatalogEntry` and `ModelCatalogPricing` exist; provenance fields use `json:"-"`. |
| `router/relay-router.go` | Client model-list intent detection | VERIFIED | `api_format=anthropic` or Anthropic headers route to Anthropic list builder; default routes to OpenAI list builder. |
| `docs/openapi/relay.json` | Corrected public schema | VERIFIED | Valid JSON; `ModelsResponse` requires only `data` and has `additionalProperties: false`; description documents Go-owned behavior and ordering. |
| `docs/CLIANYTHING.md` | Operator-facing contract docs | VERIFIED | Documents Go-owned `/v1/models`, root `data` only, hidden provenance and `api_format=anthropic`. |
| `tools/clianything.py` | Runtime status coverage for `/v1/models` | VERIFIED | Status includes the `v1-models` check and treats unauthenticated 401 as expected. |
| `tests/test_clianything.py` | CLIAnything coverage and model-list public field checks | VERIFIED | Unit tests passed; tests assert public model enrichment does not expose internal provenance fields. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `router/relay-router.go` | `controller.ListModels(...ChannelTypeAnthropic...)` | Query/header detection | WIRED | `api_format=anthropic` and Anthropic headers call the Anthropic model-list path. |
| `router/relay-router.go` | `controller.ListModels(...ChannelTypeOpenAI...)` | Default `/v1/models` branch | WIRED | Default branch remains OpenAI-compatible. |
| `controller/model.go` | `service/modelcatalog/catalog.go` | `catalogEntriesForModels`, `BuildCatalogEntry`, `SortEntries`, `IsAnthropicCapable` | WIRED | Controller uses catalog service for projection, ordering and Anthropic filtering. |
| `service/modelcatalog/catalog.go` | `model.GetPricing` / `ratio_setting` | Pricing projection and provenance | WIRED | Catalog path reads pricing rows and ratio/price helpers to classify source and public prices. |
| `controller/model.go` | Public response DTOs | `buildOpenAIModelFromCatalog`, `buildAnthropicModelFromCatalog` | WIRED | DTOs serialize public fields and hide internal provenance. |
| `.planning/config.json` | Graphify gate | `graphify.*` settings | WIRED | Required Graphify settings are true and `build_timeout` is `600`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `controller/model.go` | `userModelNames` | `model.GetGroupEnabledModels`, token model limits, user/token groups | Yes | FLOWING |
| `controller/model.go` | `catalogEntries` | `catalogEntriesForModels` -> `model.GetPricing()` -> `modelcatalog.BuildCatalogEntry` | Yes | FLOWING |
| `controller/model.go` | `userOpenAiModels` | Catalog entries converted by `buildOpenAIModelFromCatalog` | Yes | FLOWING |
| `controller/model.go` | `useranthropicModels` | Catalog entries filtered by `modelcatalog.IsAnthropicCapable` | Yes | FLOWING |
| `docs/openapi/relay.json` | `/v1/models` schema | `ModelsResponse` component | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Go catalog/controller focused tests | `PATH=/usr/local/go/bin:$PATH go test ./service/modelcatalog ./controller -run 'TestModelCatalog|TestListModels|TestBuildOpenAIModel|TestChannelOwnerName' -count=1` | `ok` for both packages | PASS |
| Model-list payload/order/Anthropic tests | `PATH=/usr/local/go/bin:$PATH go test ./controller -run 'TestListModels.*Order|TestListModels.*Payload|TestListModels.*Anthropic' -count=1` | `ok github.com/QuantumNous/new-api/controller` | PASS |
| CLIAnything tests and strict coverage | `python3 -m unittest tests.test_clianything -v && python3 -m json.tool docs/openapi/relay.json >/dev/null && bin/clianything coverage --strict` | 33 tests OK; relay JSON valid; coverage 100.0%, docs 158, manifest 158, missing 0, extra 0, problems 0 | PASS |
| Runtime status smoke | `bin/clianything status` | Exit 0; pod/backend/db/v1-models OK; `model-detailed` health row fails with connection reset | PASS_WITH_NOTE |
| Graphify freshness | `node "$HOME/.Codex/get-shit-done/bin/gsd-tools.cjs" graphify status` | `stale=false`, `commit_stale=false`, commit `c285f97` | PASS |

### Probe Execution

No phase-declared `probe-*.sh` files and no conventional `scripts/*/tests/probe-*.sh` files were found.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| `PHASE-20.1-CATALOG` | `20-01-PLAN.md` | Canonical Go catalog builder and pricing provenance foundation | SATISFIED | Catalog builder, provenance, endpoint labels and owner helper reuse are implemented and tested. |
| `PHASE-20-GRAPHIFY-GATE` | `20-02-PLAN.md` | Graphify enabled and fresh for GSD loop | SATISFIED | `.planning/config.json` has required settings; status fresh at `c285f97`. |
| `PHASE-20-GO-ONLY-V1-MODELS` | `20-02-PLAN.md` | Go owns enriched `/v1/models` contract | SATISFIED | Controller builds OpenAI/Anthropic lists from catalog; schema/docs/tests lock root `data`, removed fields and ordering. |
| `PHASE-20-AUTO-FORMAT-DETECTION` | `20-02-PLAN.md` | Go detects Anthropic query/header model-list intent | SATISFIED | Router selects Anthropic list on `api_format=anthropic` or Anthropic headers; default remains OpenAI. |
| `PHASE-20-PYTHON-MIDDLEWARE-REMOVAL` | Not claimed by 20-01/20-02 frontmatter | Middleware must not be required for `/v1/models` enrichment or route selection | SATISFIED_FOR_CORRECTED_20_2 | Go now owns `/v1/models`; retained middleware/backups still contain legacy references but are not the canonical route. Queue/retry and embeddings conversion removal are outside the corrected 20.2 target verified here. |
| `PHASE-20-CLI-DOCS-RUNTIME-PARITY` | Not claimed by 20-01/20-02 frontmatter | CLI/docs parity for operators | SATISFIED_FOR_SCOPE | `bin/clianything coverage --strict` passes; docs and OpenAPI describe Go-owned `/v1/models`. |
| `PHASE-20-SDK-SMOKES` | Not claimed by 20-01/20-02 frontmatter | SDK/runtime validation | PARTIAL_SCOPE_NOTE | Controller tests prove SDK-compatible root/data and item fields. Live SDK smoke was not part of the supplied verification commands. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---:|---|---|---|
| `runtime/model-detailed/model_detailed_fastapi.py` | 907, 1145 | Legacy `/internal/v1/models` references | INFO | Non-canonical middleware code retained outside the Go `/v1/models` route; not blocking corrected 20.2 target. |
| `runtime/model-detailed/model_detailed_fastapi.py` | 784, 785, 957, 996, 1048, 1236 | Legacy public/enrichment fields | INFO | Middleware still has old shapes, but Go public model-list route is corrected and does not depend on it. |
| `tools/clianything.py` | 186, 200 | `return []` | INFO | Empty defaults in CLI parsing/helpers, not user-visible stubs and not part of `/v1/models` route behavior. |

### Human Verification Required

None. The checked deliverable is API/schema/test behavior and was verified with code tracing plus focused automated tests. Real SDK client smoke could be useful later, but it was not a must-have for the corrected Phase 20.2 target supplied for this verification.

### Gaps Summary

No blocking gaps found. The corrected Phase 20.2 target is achieved in the Go codepath: `/v1/models` is Go-owned, root payload is data-only for OpenAI/Anthropic model-list modes, Anthropic selection is handled in Go, no Go canonical `/internal/v1/models` route was added, deterministic ordering is implemented and tested, docs/schema match the public contract, and Graphify is fresh.

Non-blocking notes:

- `bin/clianything status` exits 0 but still reports `model-detailed` health as `fail` with connection reset. Backend, DB and `v1-models` rows are OK; this does not block the Go-owned `/v1/models` contract.
- Legacy Python middleware/backups still mention `/internal/v1/models` and old enrichment payload fields. They are not wired into the verified Go canonical route and should be addressed only when a later phase removes or retires middleware paths entirely.

---

_Verified: 2026-06-18T05:50:04Z_
_Verifier: the agent (gsd-verifier)_
