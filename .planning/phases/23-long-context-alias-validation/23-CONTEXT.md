---
status: planned
phase: 23-long-context-alias-validation
created: 2026-07-01
updated: 2026-07-01
---

# Phase 23 Context — Long-Context Alias Validation

## Goal

Validate the original Codex models and the experimental 1M-context aliases end-to-end without changing the existing public models:

- The primary live validation is `POST /v1/chat/completions` with the client-selected alias in `model`.
- `gpt-5.5` and `gpt-5.4` continue to work for normal requests and continue to reject the 1M validation payload.
- `gpt-5.5-1m` is requested by the client and routed upstream as `gpt-5.5`.
- `gpt-5.4-1m` is requested by the client and routed upstream as `gpt-5.4`.
- `/v1/models` exposes the aliases with long-context pricing from the first token.
- Internal logs/billing/quota keep the requested alias visible while preserving the upstream model mapping.

## Scope

- Create a safe progressive validation script for the two base models and both aliases.
- Include catalog, streaming, small chat, and long-context reasoning checks.
- Support approximate context steps through 1M tokens.
- Require manual confirmation for expensive steps and a separate cost acknowledgement for 1M.
- Store local validation evidence under `logs/long-context-aliases/`.

## Out of Scope

- No automatic 1M execution.
- No production restart, rebuild, image update, or Portainer redeploy.
- No secret/token persistence in scripts, logs, docs, or GSD artifacts.
- No fallback to MiniMax, DeepSeek, Gemini, Claude, or generic OpenAI-compatible channels.

## Preconditions

- The code change that fixes alias pricing must be deployed before accepting production UAT results.
- `ROUTER_BASE_URL` must point at the router under test.
- `ROUTER_TEST_KEY` must be a disposable test token with access to group `default`.
- Large payload environment limits must be high enough for the selected step:
  - `MAX_REQUEST_BODY_MB`
  - `STREAMING_TIMEOUT`
  - `STREAM_SCANNER_MAX_BUFFER_MB`
  - `RELAY_IDLE_CONN_TIMEOUT`
  - `RELAY_TIMEOUT`, if configured

## Evidence Targets

Each run must capture JSONL entries for:

- `preflight_models`: optional alias presence, owner/provider, route, group, and pricing.
- `stream_smoke`: streaming path accepts the alias.
- `chat_reasoning`: progressive context request succeeded, usage was returned, and the response contained all expected sentinel anchors plus the reasoning verification code.
- `base_limit_guard`: original model 1M guard request was rejected, proving the base model path did not silently become the long-context SKU.

## Safety Rules

- Use only placeholders in docs and command examples:
  - `ROUTER_TEST_KEY`
  - `COLOQUE_UM_TOKEN_DE_TESTE_AQUI`
  - `sk-REDACTED`
- Do not print `ROUTER_TEST_KEY`.
- Do not run 1M unless `ENABLE_1M=YES_I_ACCEPT_COSTS` is set and the operator types the exact confirmation prompt.
- Do not treat a skipped 1M step as pass.
- Do not treat a base-model 1M acceptance as pass; it means the original context limitation is no longer being enforced.
