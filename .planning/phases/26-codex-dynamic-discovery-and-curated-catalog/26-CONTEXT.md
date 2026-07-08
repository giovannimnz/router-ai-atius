# Phase 26: codex-dynamic-discovery-and-curated-catalog - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning
**Source:** User prompt, current `router-ai-atius` implementation, `openai-oauth` repo review, and official OpenAI/Codex docs

<domain>
## Phase Boundary

This phase upgrades Codex model handling from a static hardcoded discovery list to a dynamic, account-aware, locally cached, validation-gated curated catalog flow.

The public `/v1/models` contract does not become live-upstream. Instead, one daily job performs discovery and enrichment offline, and the Go-owned local catalog remains the request-time source of truth.
</domain>

<decisions>
## Implementation Decisions

- **D-01 — Two-phase approach approved:** The user explicitly chose the split:
  - Phase 26 = dynamic discovery + curated catalog
  - Phase 27 = official OpenAI/Codex docs, CI/auth, and release alignment
- **D-02 — `/v1/models` must stay local:** The route must read only from local curated state and must not call upstream on request.
- **D-03 — Dynamic discovery is mandatory:** Codex discovery should be dynamic and account-aware, not static-only.
- **D-04 — Local cache is mandatory:** There must be a local persisted cache/source of truth for discovered and merged data.
- **D-04b — Discovery/cache relationship:** Dynamic discovery is the primary availability truth. The local cache is the operational fallback and request-time shield when discovery is unavailable or delayed.
- **D-05 — Two-source merge model:** The design must combine:
  - Source A = dynamic Codex availability discovery
  - Source B = another metadata source for the remaining model data
  - Local override layer = fork-owned policy and gap filling
- **D-06 — Automatic promotion with safety gate:** Discovery is no longer preview/admin-only. A scheduled process may promote models automatically, but only when it detects real changes and the model passes a live validation probe.
- **D-07 — Validation probe contract:** The probe asks the model to answer only `Ok`. Promotion depends on success.
- **D-08 — Metadata enrichment requirements:** The merged curated model data must include:
  - a parent `context_window` field/group
  - `max_tokens` as the model token ceiling
  - `max_completion_tokens` as the output-token ceiling
- **D-09 — Schedule contract:** Only one scheduled job should exist, running daily at `04:00`.
- **D-10 — Default Codex model remains local policy:** New discovered models do not silently replace the fork’s chosen default model.
- **D-11 — Responses remains the preferred Codex endpoint:** Text Codex models should prefer `openai-response` / `/v1/responses` in the curated catalog and downstream snippets.
- **D-12 — Phase 27 is explicitly deferred:** Official OpenAI docs/CI/auth/release alignment stays out of this implementation phase and remains a separate next phase.
</decisions>

<canonical_refs>
## Canonical References

**Current implementation**
- `controller/codex_fetch_models.go`
- `controller/channel_upstream_update.go`
- `controller/model.go`
- `model/pricing.go`
- `service/modelcatalog/catalog.go`
- `service/codex_oauth.go`
- `service/codex_credential_refresh.go`
- `service/codex_credential_refresh_task.go`
- `relay/channel/codex/adaptor.go`
- `relay/chat_completions_via_responses.go`
- `common/endpoint_type.go`

**Operational docs**
- `docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`
- `AGENTS.md`

**External comparison**
- `README.md` attachment from `EvanZhouDev/openai-oauth`
- Official OpenAI/Codex docs:
  - Docs MCP
  - Codex SDK
  - Codex auth / CI-CD auth
  - Codex GitHub Action
  - Codex non-interactive `codex exec`
</canonical_refs>

<specifics>
## Specific Ideas

- Reuse the existing upstream-model-update machinery if feasible, rather than inventing an unrelated scheduler.
- Reuse the existing upstream metadata sync path as Source B where possible.
- Add an explicit state machine for Codex candidates: discovered, enriched, validated, promoted, rejected.
- Keep Codex-specific metadata and default-model rules in a local override layer rather than trusting a generic upstream metadata feed completely.
- Do not model “resilience” as two discovery paths on the same machine with the same user/credential. If redundancy is needed later, it should come from persisted cache or a genuinely independent credential context.
</specifics>

<deferred>
## Deferred Ideas

- Official OpenAI docs/CI/auth/release alignment belongs to Phase 27.
- Live request-time discovery is out of scope.
- Publishing unvalidated candidates in `/v1/models` is out of scope.
</deferred>

---

*Phase: 26-codex-dynamic-discovery-and-curated-catalog*
*Context gathered: 2026-07-07 via Codex discussion synthesis*
