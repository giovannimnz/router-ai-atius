# REQUIREMENTS - Router AI Atius Go-only model routing

Status: active
Created: 2026-06-17T15:45:00-03:00
Updated: 2026-06-17T21:14:33-03:00

## Validated Requirements

### PHASE-20-GRAPHIFY-GATE

Graphify must be enabled for this workstream and treated as part of the GSD loop:

- `.planning/config.json` contains `graphify.enabled=true`.
- `.planning/config.json` contains `graphify.auto_update=true`.
- `.planning/config.json` contains `graphify.require_with_gsd=true`.
- `.planning/config.json` contains `graphify.query_before_gsd=true`.
- `.planning/config.json` contains `graphify.rebuild_after_changes=true`.
- A fresh Graphify build exists before planning/execution decisions.

### PHASE-20-GO-ONLY-V1-MODELS

Go must own `GET /v1/models` as the single enriched model catalog endpoint:

- Do not create `/internal/v1/models` as a canonical catalog endpoint.
- Preserve OpenAI-compatible model-list shape by default.
- Add stable enriched fields from Go, including endpoint labels and public pricing values.
- Public `/v1/models` root payload must be `{"data":[...]}` only for all model-list modes.
- Public `/v1/models` must not expose top-level `object` or `success`.
- Public `/v1/models` must not expose top-level Anthropic pagination keys such as `first_id`, `last_id`, or `has_more`.
- Public `/v1/models` must not expose `pricing_source` or `pricing_estimated`.
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

### PHASE-20-CLI-DOCS-RUNTIME-PARITY

Operators need CLI and docs parity:

- `bin/clianything coverage --strict` remains 100%.
- `bin/clianything models` or an equivalent typed command audits Go catalog metadata and pricing provenance.
- Docs state the Go-owned `/v1/models` contract and no longer describe Python as the catalog owner after cutover.
- Runtime restart docs continue to use user systemd services, not direct `podman restart router-ai-atius`.

### PHASE-20-SDK-SMOKES

Cutover cannot be claimed without SDK/runtime validation:

- OpenAI SDK model-list smoke works against `/v1/models`.
- OpenAI-compatible model-list tests assert the exact grouped order for a representative fixture containing MiniMax, DeepSeek, OpenAI/Codex and embeddings models.
- Anthropic SDK model-list or equivalent request smoke works against Go-owned behavior.
- MiniMax, DeepSeek, Codex OAuth streaming and MiniMax embeddings routes are represented in the routing matrix.
- Known upstream quota/rate-limit failures are classified as upstream, not local router failures.

## Out Of Scope

- Removing protected upstream project identity/branding.
- Printing or committing provider secrets, tokens, OAuth files or channel keys.
- Pruning containers or resetting runtime DB state.
