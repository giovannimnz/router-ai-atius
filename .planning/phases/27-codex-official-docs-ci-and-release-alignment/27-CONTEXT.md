# Phase 27: codex-official-docs-ci-and-release-alignment - Context

**Gathered:** 2026-07-08  
**Status:** Ready for execution  
**Source:** repo state after Phase 26, official OpenAI/Codex docs, and local operational notes

<domain>
## Phase Boundary

This phase does not change the Go routing contract or Codex catalog semantics from Phase 26.

It closes the docs/CI/auth/release gap around Codex by aligning this fork with the official OpenAI/Codex guidance for:

- `codex exec` in non-interactive automation
- `openai/codex-action` in GitHub Actions
- API-key automation versus ChatGPT-managed auth in CI/CD
- Docs MCP / official docs as the primary source of truth
- PT-BR-first operator output for this fork
</domain>

<decisions>
## Implementation Decisions

- **D-01 — Official docs first:** OpenAI/Codex official docs override community repos and local folklore for CI/auth guidance.
- **D-02 — GitHub Actions prefers `openai/codex-action`:** Repo workflows that already use Codex in GitHub Actions should align with the official action contract instead of custom CLI/auth glue.
- **D-03 — API key is the default automation path:** `CODEX_API_KEY` / `OPENAI_API_KEY` guidance stays the default for automation. ChatGPT-managed auth is advanced/private-only.
- **D-04 — ChatGPT-managed auth is private-runner only:** `auth.json` persistence guidance must stay restricted to trusted self-hosted/private runners and never be recommended for public/open-source workflows.
- **D-05 — PT-BR output remains local policy:** The fork keeps PT-BR release/operator documentation even when the authoritative source is English OpenAI docs.
- **D-06 — Phase 27 is mostly docs + workflow alignment:** The likely code touch is CI workflow input alignment for `openai/codex-action`; the rest is documentation and validation.
</decisions>

<canonical_refs>
## Canonical References

**Repo files**
- `.github/workflows/sync.yml`
- `.github/workflows/release.yml`
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`
- `docs/CI-RELEASE-WATCHDOG.md`
- `docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`

**Official OpenAI/Codex docs**
- `https://developers.openai.com/codex/noninteractive`
- `https://developers.openai.com/codex/github-action`
- `https://developers.openai.com/codex/auth/ci-cd-auth`
- `https://developers.openai.com/api/docs/guides/tools-connectors-mcp`
- `https://developers.openai.com/codex/sdk`
</canonical_refs>

<specifics>
## Specific Ideas

- Replace unofficial `codex-args` reasoning tuning in `sync.yml` with the official `effort` input when using `openai/codex-action`.
- Add a dedicated PT-BR runbook that captures:
  - when to use `codex exec`
  - when to use `openai/codex-action`
  - when API key auth is the default
  - when ChatGPT-managed auth is allowed and when it is forbidden
  - how Docs MCP fits the operator workflow
- Keep the release workflow PT-BR note generation unchanged if it is already aligned with the fork policy.
</specifics>

<deferred>
## Deferred Ideas

- Re-architecting release or sync jobs into multi-job Codex pipelines is out of scope if the current automation is already functionally aligned.
- Rewriting non-Codex GitHub Actions workflows is out of scope.
- Introducing new secrets, auth flows, or runner topology is out of scope.
</deferred>

---

*Phase: 27-codex-official-docs-ci-and-release-alignment*  
*Context gathered: 2026-07-08*
