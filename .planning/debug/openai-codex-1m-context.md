---
status: partially_resolved
trigger: OpenAI Codex 1M context aliases fail with context_length_exceeded despite attached documentation
created: 2026-07-01T00:00:00Z
updated: 2026-07-01T19:14:31-03:00
---

# Debug Session: openai-codex-1m-context

## Symptoms

- Expected: gpt-5.5-1m and gpt-5.4-1m support 1M context through /v1/chat/completions when configured per attached PDF.
- Actual: previous progressive tests returned upstream context_length_exceeded for gpt-5.5-1m at 300k+ and gpt-5.4-1m at 950k/1M.
- Reproduction: scripts/test-long-context-aliases.sh with MODEL=aliases and large SIZES.

## Current Focus

- hypothesis tested: router is not sending a required Codex 1M opt-in signal/header/body field from the attached documentation.
- result: no such opt-in field was identified in the attached PDF or in the official Codex client request shape.
- next_action: validate runtime image after adding explicit public OpenAI API mode support to the Codex adaptor.

## Evidence

- Attached PDF says context window is total input + output and cites public OpenAI API docs/pricing for GPT-5.4/GPT-5.5 1.05M context.
- Attached PDF does not identify a ChatGPT/Codex OAuth header or body field that activates 1M.
- Runtime channel 5 is `OpenAI - Codex` type 57 with OAuth JSON credential and default `chatgpt.com` base, not an OpenAI public API-key channel.
- Current OAuth credential cannot call public `https://api.openai.com/v1/responses`; direct probe returned missing `api.responses.write` scope.
- Direct ChatGPT/Codex backend probes with Codex metadata/beta headers still returned `context_length_exceeded` for large `gpt-5.5` and `gpt-5.4` payloads.
- Official OpenAI Codex client catalog observed in `openai/codex` source lists `gpt-5.5` with 272000 max context in ChatGPT-account Codex mode and `gpt-5.4` with 1000000 max context but 272000 default context.
- Local container image before this fix matched the latest local image, so prior validation was not using a stale build.
- Code correction added explicit Codex public OpenAI API mode for `base_url=https://api.openai.com` or `/v1`: `/v1/responses`, API-key auth, no ChatGPT headers, and `max_output_tokens` preserved.
- Post-build runtime loaded image `079481f584d19335c9cb5fc7071ba14bbcce541a2424d39ecfac26c8283eae57`.
- Post-build `/v1/models`, small non-stream chats for all four GPT/Codex models, and streaming smoke for `gpt-5.5-1m` passed.
- Post-build base-model 300k guard passed: `gpt-5.5` and `gpt-5.4` returned local HTTP 400 context-limit errors before upstream.
- Post-build `gpt-5.5-1m` 300k still returned upstream HTTP 400 `context_length_exceeded` through the OAuth/ChatGPT channel.
- Direct public OpenAI API smoke with the environment `OPENAI_API_KEY` returned sanitized `401 invalid_api_key`, so no valid public API credential is currently available for 1M UAT.
- Follow-up decision: `gpt-5.4-1m` is confirmed useful on the Codex endpoint; `gpt-5.5-1m` is not. Runtime disabled `gpt-5.5-1m` and kept `gpt-5.4-1m` enabled.
- Post-disable validation: `GET /v1/models` no longer lists `gpt-5.5-1m`; `gpt-5.4-1m` remains listed; `POST /v1/chat/completions` returns HTTP 200 for `gpt-5.4-1m` and HTTP 503 `model_not_found` for `gpt-5.5-1m`.
- Hermes follow-up: `~/.hermes/config.yaml` was routing GPT/Codex models through `api_mode: anthropic_messages` and `base_url` with `/v1`, which could send requests as Anthropic-compatible or produce `/v1/v1` paths.
- Hermes fix: root `model` and named `Atius-Router` provider now use `api_mode: chat_completions`, base URL `https://router.atius.com.br`, and direct `model.aliases` for `gpt-5.4-1m`, `gpt-5.4`, and `gpt-5.5`.
- Hermes validation: `hermes --ignore-rules -m gpt-5.4-1m -z`, `-m gpt-5.4 -z`, and `-m gpt-5.5 -z` all returned `OK` after adding aliases.
- Router validation from Hermes/API follow-up: authenticated `/v1/models` lists `gpt-5.4-1m` and omits `gpt-5.5-1m`; `/v1/chat/completions` small calls returned HTTP 200 for `gpt-5.4-1m`, `gpt-5.4`, and `gpt-5.5`; usage logs show `channel_id=5`.

## Eliminated

- Stale container image as the cause of the previous long-context failures.
- Missing alias mapping as the original cause; after disabling `gpt-5.5-1m`,
  runtime DB currently keeps only `gpt-5.4-1m -> gpt-5.4`.
- Missing pricing/catalog entries as the cause; aliases are visible and billed by `OriginModelName`.
- A documented ChatGPT/Codex OAuth opt-in header as the cause; none was found in the provided PDF or official request shape.

## Resolution

- Keep current OAuth/ChatGPT Codex behavior unchanged for channel 5.
- Add support for a separate public OpenAI API mode inside the Codex adaptor, gated by `api.openai.com` base URL.
- Validate with focused tests:
  - `go test ./relay/channel/codex -run 'TestCodex' -count=1`
  - `go test ./relay/channel/codex ./relay/channel/openai ./relay/helper ./controller ./setting/ratio_setting ./service/openaicompat -run 'TestCodex|TestOaiResponsesStreamToChatHandler|TestModelMappedHelperCodexLongContextAliases|TestValidateCodexContextWindow|TestCodexLongContextAliasPricingRatios|TestShouldChatCompletionsUseResponsesPolicyAlwaysEnablesCodex' -count=1`
  - `python3 -m unittest scripts/test_long_context_aliases_static_test.py`
- Runtime resolution: keep `gpt-5.4-1m` enabled and disable `gpt-5.5-1m`.
- Remaining optional path: if future OpenAI public API credentials become available, a separate public-API Codex channel can be used to retest `gpt-5.5-1m`; do not re-enable it on the current OAuth/ChatGPT Codex channel without passing progressive UAT.
