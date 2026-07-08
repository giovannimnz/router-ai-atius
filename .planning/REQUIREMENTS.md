# REQUIREMENTS - Router AI Atius Go-only model routing

Status: active
Created: 2026-06-17T15:45:00-03:00
Updated: 2026-06-18T12:35:00-03:00

## Validated Requirements

### PHASE-20-GRAPHIFY-GATE

Graphify must be treated as part of the GSD loop when this checkout has Graphify configuration enabled:

- `.planning/config.json` contains `graphify.enabled=true`.
- `.planning/config.json` contains `graphify.auto_update=true`.
- `.planning/config.json` contains `graphify.require_with_gsd=true`.
- `.planning/config.json` contains `graphify.query_before_gsd=true`.
- `.planning/config.json` contains `graphify.rebuild_after_changes=true`.
- A fresh Graphify build exists before planning/execution decisions when Graphify is enabled.
- Current runtime checkout note, 2026-06-18: `graphify status` returned `disabled` and `.planning/config.json` is absent, so validation must record Graphify as unavailable rather than fresh.

### PHASE-20-GO-ONLY-V1-MODELS

Go must own `GET /v1/models` as the single enriched model catalog endpoint:

- Do not create `/internal/v1/models` as a canonical catalog endpoint.
- Preserve OpenAI-compatible model-list shape by default.
- Add stable enriched fields from Go, including endpoint labels and public pricing values.
- Public `/v1/models` root payload must be `{"data":[...]}` only for all model-list modes.
- Public `/v1/models` must not expose top-level `object` or `success`.
- Public `/v1/models` must not expose top-level Anthropic pagination keys such as `first_id`, `last_id`, or `has_more`.
- Public `/v1/models` must not expose `pricing_source` or `pricing_estimated`.
- Public `/v1/models` must not expose `pricing_version`.
- Public `/v1/models` must keep all model-level fields not explicitly removed.
- Public `/v1/models` must return text models first and embeddings models after.
- Within text and embeddings categories, public `/v1/models` must group providers in this fixed order: MiniMax, DeepSeek, OpenAI/OpenAI Codex.
- Within each provider group, public `/v1/models` must sort from most advanced/recent/capable to least by descending `created`, version token and variant capacity.
- Variant capacity ordering must prefer `large > small`, `highspeed > standard`, and higher capability tiers such as `pro > flash` when models share a base family/version.
- Missing pricing must be explicit as `0.00` or equivalent zero price in public `/v1/models`; provenance or estimated-state tracking may remain internal and must not leak into the public model-list payload.

### PHASE-20-AUTO-FORMAT-DETECTION

Go must detect client/API intent:

- Anthropic model-list clients are detected through Anthropic headers and `api_format=anthropic`.
- OpenAI-compatible model-list clients remain default.
- `/v1/messages` is Anthropic-compatible client intent.
- `/v1/chat/completions` and `/v1/responses` are OpenAI-compatible client intent.
- Selected upstream channel family may differ from client intent when conversion support exists.

### PHASE-20-PYTHON-MIDDLEWARE-REMOVAL

Python middleware must be removed from the core API path:

- Middleware must not be required for `/v1/models` enrichment.
- Middleware must not be required for OpenAI/Anthropic route selection.
- Middleware must not be required for provider queue/retry behavior.
- Middleware must not be required for MiniMax embeddings conversion.
- Any retained middleware route must be docs/static-only and explicitly documented.

### PHASE-20-CODEX-EMBEDDINGS-SHARED-OAUTH

Codex embeddings must be implemented inside the existing Go/Codex channel path:

- `text-embedding-3-small` and `text-embedding-3-large` must be available through the `OpenAI - Codex` channel when upstream quota/licensing permits, not through a copied OpenAI key.
- The active Codex embeddings route must share the same OAuth credential as Codex chat/responses.
- Do not introduce an extra container, Python service, or sidecar as the canonical owner for Codex embeddings.
- A separate `Codex - Embeddings` channel may exist only as disabled historical/manual fallback surface; do not promote it as the default active route.
- Upstream `429 insufficient_quota` for Codex embeddings is a quota/licensing result when the selected channel is channel 5; it must not be misclassified as local routing failure.
- If Codex embeddings return upstream `429 insufficient_quota`, remove/disable those models from active routing and `/v1/models` until quota/licensing is fixed.

### PHASE-20-PROVIDER-CHANNEL-CONSOLIDATION

MiniMax and DeepSeek must be operated as one active channel per provider:

- `MiniMax` must use channel type `35` and base URL `https://api.minimax.io`; it owns OpenAI-compatible chat, Anthropic-compatible messages, and `embo-01` embeddings when the upstream route is healthy.
- `DeepSeek` must use channel type `43` and base URL `https://api.deepseek.com`; it owns OpenAI-compatible chat and Anthropic-compatible messages when the upstream key is valid.
- The Go relay must infer upstream URL and request/response conversion from endpoint format and channel type; operators must not need separate `*-OpenAI-Compatible`, `*-Anthropic-Compatible`, or `*-Embeddings` active channels.
- `OpenAI - Codex` is the canonical channel name for type `57`; legacy label `ChatGPT Subscription (Codex)` must not reappear in backend or frontend labels.
- `MiniMax - Anthropic-Compatible`, `MiniMax - Embeddings`, `DeepSeek - Anthropic-Compatible`, `OpenAI - Embeddings`, and `Codex - Embeddings` must stay disabled unless an operator performs an explicit manual break-glass action.
- Any model/provider that fails strict production UAT due upstream invalid key, quota, or persistent RPM limit must be disabled/removed from active `channels.models` until corrected, so `/v1/models` only advertises working models.
- `bin/clianything channel phase19-apply` is legacy-named compatibility and must apply the consolidated state, not recreate split channels.

### PHASE-20-BASEURL-V1-NORMALIZATION

Provider URL construction must be tolerant of common operator base URL variants:

- Generic OpenAI-compatible relay URLs must accept provider base URLs with or without trailing slash.
- Generic OpenAI-compatible relay URLs must accept provider base URLs ending in `/v1` without producing duplicated `/v1/v1/...` request paths.
- MiniMax and DeepSeek provider-root URL builders must normalize a trailing `/v1` before appending Anthropic/native provider paths.
- This normalization belongs in Go (`relay/common` and provider adaptors), not in a Python middleware or sidecar.
- Regression tests must cover root base URL, trailing slash, and trailing `/v1` cases.

### PHASE-20-CLI-DOCS-RUNTIME-PARITY

Operators need CLI and docs parity:

- `bin/clianything coverage --strict` remains 100%.
- `bin/clianything models` or an equivalent typed command audits Go catalog metadata and pricing provenance.
- Docs state the Go-owned `/v1/models` contract and no longer describe Python as the catalog owner after cutover.
- Runtime restart docs continue to use user systemd services, not direct `podman restart router-ai-atius`.

### PHASE-20-UPSTREAM-SYNC-GUARD

Fork sync/merge must preserve the Atius-specific Go-native routing contract:

- `controller/model.go`, `controller/model_list_test.go`, `service/modelcatalog/`, `relay/common/relay_utils.go`, `relay/common/relay_utils_test.go`, `relay/channel/codex/`, `service/codex_*.go`, `.dockerignore`, `docs/` and `.planning/` must be treated as fork-owned/protected paths unless a human intentionally ports the upstream equivalent.
- Upstream sync must not reintroduce `pricing_version` into public `/v1/models`.
- Upstream sync must not move `/v1/models` ownership back to Python/model-detailed.
- Upstream sync must not remove the Go base URL normalization that accepts provider base URLs ending in `/v1`.
- Upstream sync must not split Codex embeddings into an active independent OpenAI key/channel by default.
- After sync, the first public model should remain the most recent/capable MiniMax text model, including `MiniMax-M3` when enabled and visible.

### PHASE-20-SDK-SMOKES

Cutover cannot be claimed without SDK/runtime validation:

- OpenAI SDK model-list smoke works against `/v1/models`.
- OpenAI-compatible model-list tests assert the exact grouped order for a representative fixture containing MiniMax, DeepSeek, OpenAI/Codex and embeddings models.
- Anthropic SDK model-list or equivalent request smoke works against Go-owned behavior.
- MiniMax, DeepSeek, Codex OAuth streaming and embeddings routes are represented in the routing matrix when enabled; disabled routes must be covered by negative tests proving they do not route.
- Known upstream quota/rate-limit failures are classified as upstream, not local router failures.

## Phase 21 Requirements

| Requirement ID | Summary |
|---|---|
| PHASE-21-UPSTREAM-NATIVE-I18N | Implement Brazilian Portuguese through upstream-native backend, default frontend, and classic frontend i18n surfaces. |
| PHASE-21-PT-BR-COVERAGE | Prove complete PT-BR key, placeholder, untranslated, selector, and normalization coverage. |
| PHASE-21-REUSE-EXISTING-TRANSLATIONS | Reuse existing PT-BR translation work first, with reviewed literal/fallback classification. |
| PHASE-21-LOCAL-FIRST-VALIDATION | Validate locally against a clean current `upstream/main` lane before PR handoff. |
| PHASE-21-UPSTREAM-PR-HYGIENE | Keep any upstream PR clean, template-compliant, duplicate-aware, and free of fork/runtime/secrets content. |

### PHASE-21-UPSTREAM-NATIVE-I18N

Brazilian Portuguese must be implemented through the same native language surfaces that exist in `QuantumNous/new-api` upstream, not through a fork-only translation layer:

- Backend uses the existing upstream `i18n/` package and `i18n/locales/*.yaml` embed pattern.
- Default frontend uses the existing upstream `web/default/src/i18n/` i18next resources, `supportedLngs`, and `INTERFACE_LANGUAGE_OPTIONS`.
- Classic frontend uses the existing upstream `web/classic/src/i18n/` i18next resources, `supportedLanguages`, language selector, and preferences list.
- Do not create `i18n/pt.yaml`; backend Portuguese must live at `i18n/locales/pt.yaml`.
- Do not remove or rename upstream `i18n/` directories, because they are the native upstream mechanism.
- Do not introduce runtime-specific, Atius-specific, or sidecar translation behavior.

### PHASE-21-PT-BR-COVERAGE

Portuguese support must be complete enough to behave like every existing upstream language:

- Backend `pt` YAML keeps key parity with `i18n/locales/en.yaml`.
- Default frontend `pt.json` keeps key parity with `web/default/src/i18n/locales/en.json`.
- Classic frontend `pt.json` keeps key parity with `web/classic/src/i18n/locales/en.json`.
- Default frontend locale sync reports `missingCount=0`, `extrasCount=0`, and `untranslatedCount=0` for `pt`.
- Backend, default frontend, and classic frontend checks must fail on placeholder-token drift.
- Default frontend dynamic/static translation keys in `web/default/src/i18n/static-keys.ts` must remain covered wherever those keys are part of the upstream base locale.
- Portuguese labels use the native name `Português` in user-visible language pickers.
- Locale normalization accepts `pt`, `pt-BR`, and underscore variants where the upstream code normalizes other language variants.

### PHASE-21-REUSE-EXISTING-TRANSLATIONS

Existing PT-BR translation work must be reused before any new translation is created:

- Inventory existing fork/local translation sources before editing locale files.
- Record the inventory in `.planning/phases/21-feat-pt-native-pr/21-TRANSLATION-INVENTORY.md`.
- Reuse current or historical PT-BR strings for matching keys whenever placeholders and semantics still match.
- Reuse translations from the previous clean PT lane, current fork PT files, and any existing `pt`/`pt-BR` locale artifacts.
- For classic frontend, reuse default frontend PT strings for identical English keys before translating gaps manually.
- Do not cherry-pick or copy whole historical branches when only the PT translation map is usable.
- Do not replace an existing correct PT-BR translation with an English fallback just because sync tooling filled a missing key.
- New translation work is limited to true upstream-current gaps after reuse.
- Preserve placeholders, plural suffixes, markdown/code fragments, URLs, API/model names, and protected project identity text exactly.
- Classify same-as-English values as brand/code literals or unresolved gaps before claiming 100% coverage.

### PHASE-21-LOCAL-FIRST-VALIDATION

The implementation must be validated locally before any upstream PR is prepared:

- Build the implementation against a branch based on current `upstream/main`, not against dirty fork history.
- Keep the final code diff limited to native language files, native wiring, and narrowly justified validation/tests.
- Run backend i18n validation with Go.
- Run default frontend `bun run i18n:sync`, typecheck, and scoped lint/build checks required by upstream rules.
- Run classic frontend language parity and build checks when classic files are changed.
- Verify the resulting app can select/persist Portuguese in default and classic UI through the existing shared/native language controls, without adding custom UI flows.
- Use explicit machine checks for key parity, placeholder parity, default sync-report zero counts, and `pt-BR`/`pt_BR` normalization rather than relying only on build success.

### PHASE-21-UPSTREAM-PR-HYGIENE

If the local result is promoted upstream, the PR must follow `QuantumNous/new-api` contribution rules:

- Use `.github/PULL_REQUEST_TEMPLATE.md` without replacing its structure.
- Search existing upstream issues and PRs for Portuguese/PT-BR duplicates before opening.
- Treat issue #2924 as the current upstream Portuguese translation request unless it has changed or closed.
- Treat PR #5801 as related but not equivalent unless it has changed scope; it currently adds only `i18n/pt.yaml`, which is not the full native pattern.
- Treat closed PRs #5238 and #5245 as contaminated historical context, not reusable PR scope.
- Compare current git user to upstream core authors and disclose AI assistance in the PR body when required.
- Do not alter protected upstream project identity, organization identity, branding, metadata, module paths, or attribution.
- Do not include `.planning/`, Graphify, Obsidian, runtime docs, provider/router/governor changes, secrets, or fork-only infrastructure in the upstream PR.
- Leak checks must fail when forbidden fork/planning/runtime/secrets text appears in either the code diff or PR/comment drafts.

## Out Of Scope

- Removing protected upstream project identity/branding.
- Printing or committing provider secrets, tokens, OAuth files or channel keys.
- Pruning containers or resetting runtime DB state.

## Phase 24 Requirements

### PHASE-24-CANONICAL-HOST-DB

The router runtime must use a single canonical host PostgreSQL database through PgBouncer, and the database name must match the intended production identity:

- The canonical runtime DB path remains on the host via PgBouncer, not inside a Podman `postgres` container.
- The router must not depend on `container-postgres.service` to serve production traffic after recovery.
- The final DB name must be the intended canonical name for `router-ai-atius`, not an accidental fallback/legacy name left by a recovery flow.
- Recovery planning must explicitly inventory the current `newapi` contents, the target canonical DB name, PgBouncer bindings, and any required rename/copy/restore sequence before changing the runtime.
- No destructive DB rename or drop is allowed without a fresh validated backup and a rollback path.

### PHASE-24-CATALOG-RESTORE

The active DB must recover the full router catalog that was known-good on 2026-07-01, subject to the new exclusions requested by the user:

- Restore the `OpenAI - Codex` channel and its GPT/Codex catalog/routing surfaces.
- Restore `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, and `gpt-5.3-codex-spark`.
- Do not recreate `gpt-5.4-1m` or `gpt-5.5-1m`.
- Do not restore `text-embedding-3-small` or `text-embedding-3-large`.
- Restore `embedding-gte-v1` as the governed local embedding alias.
- Recovery must compare live DB state against local SQL snapshots/dumps and document exactly which rows are restored, transformed, skipped, or disabled.

### PHASE-24-PROVIDER-CONSOLIDATION

Provider/channel recovery must preserve the Go-native consolidated routing design:

- DeepSeek must end in one active channel with automatic OpenAI/Anthropic routing behavior owned by Go.
- MiniMax must end in one consolidated channel with automatic OpenAI/Anthropic routing behavior owned by Go.
- MiniMax provider/channel and its models must be restored but left disabled per the user request.
- DeepSeek V4 Flash and DeepSeek V4 Pro must be restored.
- `OpenAI - Codex` must not restore `channels.model_mapping` entries for `gpt-5.5-1m` or `gpt-5.4-1m`.
- Recovery must not reintroduce split active channels as the intended final design.

### PHASE-24-EMBEDDING-GOVERNOR-PRESERVE

The Go-native embedding path must remain intact through recovery:

- `embedding-gte-v1` remains the only governed public embedding alias.
- The Go governor path in `service/embeddinggovernor/` and `relay/embedding_handler.go` remains canonical.
- Recovery must verify that `EMBEDDING_GOVERNOR_*` settings remain aligned with the intended contract.
- Recovery must verify that the embeddings channel, model row, abilities row, and token/routing behavior still pass real `/v1/embeddings` and catalog checks after broader DB restoration.
- Recovery must not reintroduce Python/model-detailed ownership for embeddings.

### PHASE-24-CUTOVER-ROLLBACK

Recovery is not complete until runtime, docs, and verification agree:

- All DB/catalog mutations require a fresh full backup plus a catalog-only backup before execution.
- The final runtime must pass `bin/clianything status --strict`, authenticated `/v1/models`, authenticated `/v1/embeddings`, and representative GPT/DeepSeek routing checks.
- Documentation must be reconciled so the runtime DB path, provider inventory, and backup/restore story match reality after recovery.
- The rollback plan must name the exact backup artifacts and runtime re-point steps needed to undo the recovery if validation fails.

## Phase 25 Requirements

| Requirement ID | Summary |
|---|---|
| PHASE-25-GOVERNED-MODEL-SCOPE | Keep `embedding-gte-v1` as the single default public governed local embedding alias, with no public batch alias. |
| PHASE-25-AUTO-WORKLOAD-INFERENCE | Infer governed embedding workload from metadata when the client omits `X-Embedding-Workload`. |
| PHASE-25-HEADER-OVERRIDE-COMPATIBILITY | Preserve explicit `X-Embedding-Workload` values as operator overrides. |
| PHASE-25-TEI-BATCH-SAFETY | Keep conservative governor limits and enforce the TEI max client batch-size contract. |
| PHASE-25-CLIENT-SMOKE-VALIDATION | Prove the client-facing `/v1/embeddings` contract with tests, docs, and token-safe smoke validation. |

### PHASE-25-GOVERNED-MODEL-SCOPE

`embedding-gte-v1` must remain the single default public governed local embedding alias:

- `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` remains the intended default.
- `EMBEDDING_GOVERNOR_BATCH_MODELS=` remains empty; batch is not a public alias.
- `embedding-gte-v1-batch` or any similar `*-batch` model must not be exposed in `/v1/models`.
- Unknown or non-governed models must preserve current no-op governor behavior.

### PHASE-25-AUTO-WORKLOAD-INFERENCE

The router must classify unlabeled governed embedding requests before or at governor acquisition:

- Explicit `X-Embedding-Workload` stays highest priority.
- Without header, `input` arrays with at least 2 text items classify as batch.
- Without header, a single string classifies as interactive unless its configured character threshold marks it batch.
- Add a safe, testable `EMBEDDING_GOVERNOR_AUTO_WORKLOAD` control, defaulting to enabled for the governed local model path.
- Add or normalize `EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD=2`.
- The classifier must use metadata (`InputCount`, `InputChars`, model, header), not raw embedding text retained in governor state or snapshots.

### PHASE-25-HEADER-OVERRIDE-COMPATIBILITY

Existing operational overrides must continue to work:

- `X-Embedding-Workload: batch` and existing `bulk` behavior force batch.
- `X-Embedding-Workload: interactive` and `realtime` force interactive.
- Invalid header values fall back to safe automatic inference rather than disabling the governor.
- Docs must state that the header is optional for normal clients and only needed as an override.

### PHASE-25-TEI-BATCH-SAFETY

Automatic batch inference must not overload the local TEI deployment:

- Batch concurrency no longer has a separate static router ceiling by default; batch still yields to waiting interactive requests and shares the adaptive total pool.
- Interactive and batch accounting remain separate.
- Automatic governor concurrency has no static router ceiling by default. `EMBEDDING_GOVERNOR_MAX_CONCURRENCY=0` means the governor can scale with healthy demand and available TEI pod capacity; a positive value reintroduces an explicit cap.
- Read-only TEI capacity telemetry must be able to block scale-up when pod capacity is saturated. At or above `EMBEDDING_GOVERNOR_CAPACITY_MAX_USED_PERCENT=80`, or when `pods_ready < pods_total`, the governor must not increase concurrency; consecutive bad capacity windows may reduce concurrency gradually toward `min=1`.
- Requests with more than 4 input items must respect the TEI max client batch size 4 through existing behavior, new sub-batching, or a blocking validation that proves the current upstream path already enforces the cap.

### PHASE-25-CLIENT-SMOKE-VALIDATION

The phase is complete only when tests and runtime smokes prove the client-facing contract:

- Service tests cover governed model scope and classifier decisions.
- Relay tests cover the captured governor request for header and no-header cases.
- Docs and smoke scripts show that Graphify/GBrain/future clients can omit `X-Embedding-Workload` for normal operation.
- Authenticated `/v1/embeddings` smoke for `embedding-gte-v1` returns dimension `768` without printing tokens.

## Phase 26 Requirements

| Requirement ID | Summary |
|---|---|
| PHASE-26-LOCAL-CURATED-V1-MODELS | `/v1/models` stays local/curated and never depends on live upstream reads at request time. |
| PHASE-26-DYNAMIC-CODEX-DISCOVERY | Codex model discovery becomes dynamic and account-aware instead of static-only. |
| PHASE-26-MULTI-SOURCE-ENRICHMENT | Discovery must merge Source A availability with Source B metadata plus local Codex overrides. |
| PHASE-26-CANDIDATE-PROBE-GATE | Newly discovered Codex models must pass a minimal live `Ok` probe before promotion. |
| PHASE-26-CODEX-METADATA-ENRICHMENT | Curated Codex metadata must expose a `context_window` parent field/group containing `max_tokens` and `max_completion_tokens`. |
| PHASE-26-DAILY-SCHEDULED-SYNC | Exactly one daily `04:00` scheduled job runs discovery, enrichment, diffing, validation, and promotion. |
| PHASE-26-DEFAULT-MODEL-GUARD | The fork’s chosen default Codex model remains a local policy even when new upstream models appear. |

### PHASE-26-LOCAL-CURATED-V1-MODELS

The public model catalog must remain local and deterministic:

- `/v1/models` must not call ChatGPT/Codex/OpenAI docs or any other upstream live during request handling.
- The Go-owned local curated catalog remains the only public source of truth.
- Discovery and enrichment may mutate the local catalog asynchronously, but request-time reads stay local.

### PHASE-26-DYNAMIC-CODEX-DISCOVERY

Codex availability must be discovered dynamically:

- Source A must query the active Codex credential/runtime path and return the models that the current account can actually use upstream.
- Static-only discovery is insufficient as the long-term contract.
- The raw discovery output must be cached locally.
- Dynamic discovery is the primary availability source; the local cache is the fallback/runtime shield when discovery is temporarily unavailable.

### PHASE-26-MULTI-SOURCE-ENRICHMENT

Curated Codex model records must merge at least two sources:

- Source A decides candidate availability.
- Source B supplies metadata such as vendor/pricing/capability candidates.
- A local override layer fills or overrides Codex-specific fields and fork-owned rules.
- Merge logic must normalize names, deduplicate aliases, and preserve one canonical public entry per model.

### PHASE-26-CANDIDATE-PROBE-GATE

Promotion into the curated catalog is gated by validation:

- A candidate model must receive a minimal request asking it to reply only `Ok`.
- Only successful candidates may be auto-promoted into the active curated catalog.
- Failed or unavailable candidates remain cached locally but unpublished.

### PHASE-26-CODEX-METADATA-ENRICHMENT

Curated Codex metadata must explicitly include:

- a parent `context_window` field/group
- `max_tokens` as the model token ceiling
- `max_completion_tokens` as the maximum output-token ceiling

These values must come from enrichment plus local curation, not request-time guesswork.

### PHASE-26-DAILY-SCHEDULED-SYNC

There must be exactly one scheduled Codex discovery pipeline:

- It runs daily at `04:00`.
- It performs discovery, enrichment, diffing, validation, and promotion.
- It only writes local catalog state when a real upstream change is detected and the promotion gate passes.
- It must not rewrite unchanged state on every run.

### PHASE-26-DEFAULT-MODEL-GUARD

New Codex models must not silently change fork policy:

- The fork’s default Codex model remains locally controlled.
- A new validated model may be added automatically, but it must not replace the configured default unless local curation says so.

## Phase 27 Requirements

| Requirement ID | Summary |
|---|---|
| PHASE-27-OFFICIAL-DOCS-FIRST | OpenAI/Codex official docs and Docs MCP are the primary reference for CI/auth/release behavior. |
| PHASE-27-CODEX-CI-AUTH | Codex automation must align with official `codex exec`, `openai/codex-action`, and ChatGPT-managed auth guidance. |
| PHASE-27-PTBR-RELEASE-OPS-DOCS | Release notes, changelog and operator docs for this fork remain PT-BR-first. |

### PHASE-27-OFFICIAL-DOCS-FIRST

Community repos can inform implementation ideas, but official OpenAI/Codex docs remain authoritative for:

- CI/CD auth patterns
- `codex exec` usage
- `openai/codex-action`
- Codex SDK usage
- Docs MCP integration

### PHASE-27-CODEX-CI-AUTH

The fork’s Codex CI/auth guidance must align with official OpenAI behavior:

- Prefer `codex exec` for non-interactive task execution.
- Prefer `openai/codex-action` in GitHub Actions where Codex is part of the pipeline.
- ChatGPT-managed auth guidance must follow the official CI/CD auth path instead of bespoke unsupported token-refresh workflows in arbitrary jobs.

### PHASE-27-PTBR-RELEASE-OPS-DOCS

This fork’s operational output remains PT-BR-first:

- Release notes and operator summaries are generated in PT-BR by default.
- CI/release docs may consume upstream English references, but the fork’s published operator/release text remains Portuguese.
