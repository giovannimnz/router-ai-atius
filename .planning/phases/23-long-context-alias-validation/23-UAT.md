---
status: partial
phase: 23-long-context-alias-validation
source: [23-CONTEXT.md, 23-01-PLAN.md]
started: 2026-07-01T00:00:00Z
updated: 2026-07-01T16:25:00Z
---

## Current Test

Live validation reached the real upstream context boundary. The router-side catalog, alias mapping, pricing, response model preservation, small `/v1/chat/completions`, streaming, and base-model guardrails are passing. The remaining gap is upstream support: with the mandated mapping to `gpt-5.5` and `gpt-5.4`, the current Codex upstream rejects the 1M alias requests with `context_length_exceeded`.

## Evidence Logs

- `logs/long-context-aliases/20260701T162425Z-long-context-aliases.jsonl`: final `/v1/models` preflight after last deploy.
- `logs/long-context-aliases/20260701T154947Z-long-context-aliases.jsonl`: small non-stream chat for base and alias models.
- `logs/long-context-aliases/20260701T155001Z-long-context-aliases.jsonl`: streaming smoke for base and alias models.
- `logs/long-context-aliases/20260701T161432Z-long-context-aliases.jsonl`: base model 250k accept and 300k local reject guard.
- `logs/long-context-aliases/20260701T161454Z-long-context-aliases.jsonl`: alias progressive large-context run.
- `logs/long-context-aliases/20260701T161658Z-long-context-aliases.jsonl`: `gpt-5.4-1m` 900k nominal pass, observed prompt usage about 888k tokens.
- `logs/long-context-aliases/20260701T161724Z-long-context-aliases.jsonl`: `gpt-5.4-1m` 950k nominal upstream rejection.
- `logs/long-context-aliases/20260701T162417Z-long-context-aliases.jsonl`: final structured upstream error propagation check.

## Tests

### 1. Static Harness Safety
expected: Local syntax/static tests pass and prove the script has model allowlist, no token printing, large-step confirmation, 1M cost gate, and gitignored JSONL logs.
result: pass
observed: `bash -n scripts/test-long-context-aliases.sh` and `python3 scripts/test_long_context_aliases_static_test.py` passed. Focused Go tests for catalog, mapping, pricing, context guard, Codex chat-via-Responses, and Responses SSE aggregation also passed.

### 2. Catalog Preflight
expected: `/v1/models` exposes `gpt-5.5`, `gpt-5.5-1m`, `gpt-5.4`, and `gpt-5.4-1m`, preserving existing base models and alias pricing.
result: pass
observed: Final preflight returned HTTP 200 with all four models. Evidence: `logs/long-context-aliases/20260701T162425Z-long-context-aliases.jsonl`.

### 3. Chat Completions Base And Alias Smoke
expected: `POST /v1/chat/completions` accepts `gpt-5.5`, `gpt-5.5-1m`, `gpt-5.4`, and `gpt-5.4-1m`, returns HTTP 200, returns usage, and keeps response `model` equal to the requested model.
result: pass
observed: All four small non-stream requests passed after routing Codex chat-completions through Responses aggregation. Evidence: `logs/long-context-aliases/20260701T154947Z-long-context-aliases.jsonl`.

### 4. Streaming Smoke
expected: Streaming `/v1/chat/completions` returns HTTP 200 and streamed `data:` events for base and alias models, preserving alias visibility in streamed chunks.
result: pass
observed: All four streaming smoke checks passed. Alias stream chunks use the requested alias as public model. Evidence: `logs/long-context-aliases/20260701T155001Z-long-context-aliases.jsonl`.

### 5. Original Model Context Guard
expected: Base models continue to accept normal/large-but-valid payloads and reject oversized requests before upstream/billing when the estimated prompt exceeds the original limit.
result: pass
observed: `gpt-5.5` and `gpt-5.4` accepted 250k nominal requests and locally rejected 300k nominal requests with HTTP 400 because the conservative prompt estimate exceeded 272000 tokens. Evidence: `logs/long-context-aliases/20260701T161432Z-long-context-aliases.jsonl`.

### 6. Alias Progressive Reasoning
expected: Aliases route through `/v1/chat/completions`, use alias pricing, preserve requested model externally, and progressively accept long-context requests up to 1M.
result: partial
observed: `gpt-5.4-1m` passed 300k, 500k, 750k, and 900k nominal requests; 900k nominal produced prompt usage around 888k tokens. `gpt-5.5-1m` rejected 300k and larger requests from upstream. Evidence: `logs/long-context-aliases/20260701T161454Z-long-context-aliases.jsonl` and `logs/long-context-aliases/20260701T161658Z-long-context-aliases.jsonl`.

### 7. Alias 1M Execution
expected: `gpt-5.5-1m` and `gpt-5.4-1m` complete 1M `/v1/chat/completions` reasoning requests with HTTP 200.
result: blocked
blocked_by: upstream-context-window
reason: Both aliases are required to map to the real upstream base models. The current upstream rejects `gpt-5.5-1m` at 300k+ and `gpt-5.4-1m` at 950k/1M with `context_length_exceeded`. A direct upstream probe and beta-header attempts produced the same rejection for `gpt-5.5`.

### 8. Runtime Traceability And Billing
expected: Logs and responses preserve requested alias visibility while upstream receives `gpt-5.5`/`gpt-5.4`; billing uses alias pricing.
result: pass
observed: Channel model mapping maps aliases to base upstream models; public chat responses and stream chunks preserve the requested alias. Ratio settings and `/v1/models` expose `gpt-5.5-1m` at 10 input / 45 output and `gpt-5.4-1m` at 5 input / 22.5 output.

### 9. Upstream Error Propagation
expected: Upstream context-window failures are returned as structured OpenAI-style errors instead of generic `response.failed` messages.
result: pass
observed: Final post-deploy check for `gpt-5.5-1m` 300k returned HTTP 400 with structured upstream error code `context_length_exceeded`. Evidence: `logs/long-context-aliases/20260701T162417Z-long-context-aliases.jsonl`.

## Summary

total: 9
passed: 7
partial: 1
blocked: 1
pending: 0

## Gaps

- The router-side implementation is corrected and deployed, but the 1M acceptance criterion is blocked by the real upstream context window under the mandated base-model mapping.
- To reach true 1M for both aliases, the deployment needs an upstream model, account entitlement, or documented header/feature flag that actually enables 1M context for `gpt-5.5` and `gpt-5.4`.
