# Phase 26 Research - codex-dynamic-discovery-and-curated-catalog

**Date:** 2026-07-07
**Status:** Ready for planning

## Current state

What already exists in `router-ai-atius`:

- Codex OAuth refresh and JWT-derived `account_id` handling.
- Codex routing to `chatgpt.com/backend-api/codex/responses`.
- OpenAI public `/v1/responses` compatibility when configured explicitly.
- Chat -> Responses conversion.
- Shared chat/responses/embeddings credential model for channel `5`.
- Go-owned curated public `/v1/models`.

What is still static:

- Codex discovery for admin fetch-models / sync paths is hardcoded in `controller/codex_fetch_models.go`.

## External repo findings

The reviewed `openai-oauth` project demonstrates that:

- account-aware Codex discovery can be dynamic;
- the same Codex/ChatGPT OAuth token cache can drive `/v1/responses`, `/v1/chat/completions`, and `/v1/models`;
- the project is intentionally local/personal and does not claim to be appropriate as a hosted shared router.

## Architectural interpretation

`openai-oauth` is useful as a transport/auth reference, not as a hosted service architecture to adopt directly.

The right move for `router-ai-atius` is:

1. preserve our Go-owned curated catalog;
2. add dynamic discovery as an upstream input;
3. enrich/merge asynchronously;
4. publish only validated local results.

## Recommended source design

### Source A — dynamic Codex availability

Purpose: determine which Codex models the active account can use today.

Output should be raw candidate model ids plus discovery metadata.

Recommendation:

- Source A should be the primary availability truth.
- The local cache should be the fallback/runtime shield.
- Two discovery paths on the same machine with the same user add little resilience and more churn unless they are truly independent credential contexts.

### Source B — metadata enrichment

Best fit already present in this repo:

- existing upstream metadata sync flow (`controller/model_sync.go`) for models/vendors

This can provide a large part of the generic metadata surface, but not all Codex-specific values.

### Local override layer

Required for:

- a parent `context_window` field/group
- `max_tokens` as the model token ceiling
- `max_completion_tokens` as the output-token ceiling
- preferred endpoint order
- default model policy
- exclusions / denylists / curated labels

## Join strategy

Recommended precedence:

1. Source A determines entitlement/availability.
2. Source B contributes metadata candidates.
3. Local override layer wins for fork-owned rules and missing Codex-specific fields.

## Promotion strategy

Recommended candidate flow:

1. discover candidate
2. enrich candidate
3. diff against current local promoted state
4. if changed, probe model with prompt requesting only `Ok`
5. promote only if probe succeeds and metadata is complete enough
6. refresh local runtime caches

## Storage options

### JSON in options

Pros:
- quick
- low migration cost

Cons:
- poor auditability
- awkward diffing/history
- weak admin observability

### Dedicated DB tables

Pros:
- explicit state machine
- auditable
- queryable
- better future admin UX

Cons:
- higher implementation cost

**Recommendation:** dedicated DB tables or at least one structured table for discovery state, not pure opaque option blobs.

## Scheduler options

### Reuse current upstream-model-update scheduler

Pros:
- already exists
- already refreshes cache and persists update state
- already has notification flow

Cons:
- currently model-id oriented, not rich candidate-state oriented

### Separate Codex-only scheduler

Pros:
- cleaner if the state machine becomes substantially richer

Cons:
- duplicates scheduler concepts already present

**Recommendation:** attempt reuse first; split only if reuse makes the code brittle.

## Gain assessment

### High gain

- lower operational drift between Codex entitlement and curated catalog
- less manual intervention when Codex models change
- stronger basis for automatic but safe catalog updates

### Medium gain

- better model metadata quality in downstream UI/API consumers
- clearer operator state around discovered vs promoted models

### Low gain

- little immediate end-user impact outside catalog freshness and correctness

## Recommendation

Proceed.

This phase is worth it because it strengthens the operational correctness of the Codex provider without giving up the safety of a local curated catalog.
