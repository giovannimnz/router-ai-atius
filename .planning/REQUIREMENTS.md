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

## Out Of Scope

- Removing protected upstream project identity/branding.
- Printing or committing provider secrets, tokens, OAuth files or channel keys.
- Pruning containers or resetting runtime DB state.
