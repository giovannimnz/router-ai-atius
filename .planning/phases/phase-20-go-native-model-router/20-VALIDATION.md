---
phase: phase-20-go-native-model-router
kind: validation
date: 2026-06-18
status: passed
---

# Phase 20 Validation - Final Full-Go State

## Validated Changes

- Retired Python/model-detailed from the active `/v1/` production path.
- Routed Apache `/v1/`, `/v1/models`, `/health`, `/api/` and `/` to Go on `127.0.0.1:3000`.
- Removed model-detailed from active pod/systemd runtime; pod exposes only host port `3000`.
- Implemented Go base URL normalization for provider URLs ending in `/v1`.
- Kept `/v1/models` public payload SDK-compatible: root `data` only, no internal pricing provenance fields.
- Kept provider consolidation active-only: MiniMax channel 1 and OpenAI - Codex channel 5.
- Disabled upstream-broken DeepSeek and embeddings routes so advertised models work without exception.

## Verification Evidence

| Area | Evidence |
|---|---|
| Go tests | `go test ./common ./controller ./service/modelcatalog ./relay/common ./relay/channel/minimax ./relay/channel/deepseek ./relay/channel/codex ./service/openaicompat -count=1` passed |
| Python/CLI tests | `py_compile` passed; `unittest discover` ran 34 tests with 1 intentional skip |
| Runtime health | `bin/clianything status --strict` passed after final restart |
| Apache | `apache2ctl configtest` returned `Syntax OK` |
| Public catalog | HTTP 200; 7 models; root `data`; no `pricing_version`, `pricing_source`, `pricing_estimated` |
| Public routing matrix | active-only smoke passed for MiniMax OpenAI/Anthropic and Codex OpenAI |
| Disabled routes | DeepSeek and embeddings returned HTTP 503 with no old-channel log |
| Base URL `/v1` probe | Temporarily set MiniMax base URL to `https://api.minimax.io/v1`; full active smoke passed; reverted to `https://api.minimax.io` |

## Rollback Points

- `ghcr.io/giovannimnz/router-ai-atius:rollback-before-baseurl-v1-normalize-20260618122124`
- `ghcr.io/giovannimnz/router-ai-atius:rollback-before-provider-consolidation-20260618080648`
- Apache backups:
  - `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf.bak-remove-model-detailed-20260618121341`
  - `/etc/apache2/sites-available/router.atius.com.br-le-ssl.conf.bak-full-go-source-*`
- Channel DB backups from the temporary base URL probe:
  - `/home/ubuntu/GitHub/containers/router-ai-atius/backups/clianything/20260618_124100_channels.sql`
  - `/home/ubuntu/GitHub/containers/router-ai-atius/backups/clianything/20260618_124250_channels.sql`

## Remaining Non-Local Blockers

- DeepSeek needs a valid upstream key before reactivation.
- Codex embeddings need upstream quota/licensing before reactivation.
- MiniMax `embo-01` needs upstream RPM/quota stability before reactivation.

These are deliberately excluded from active `/v1/models` so the public catalog advertises only working models.

## MiniMax Embeddings Revalidation - 2026-06-22

Scope: exhaustive validation of whether MiniMax `embo-01` should be active through the Go-native router.

### External Reference Findings

| Source | Finding |
|---|---|
| LangChain `MiniMaxEmbeddings` pinned source | Uses native MiniMax payload: `model`, `type`, `texts`; model default `embo-01`; endpoint default `https://api.minimax.chat/v1/embeddings`; successful response is top-level `vectors` plus `base_resp`. |
| LangChain API docs | Documents `embed_type_query=query`, `embed_type_db=db`, `endpoint_url`, `MINIMAX_API_KEY`, and legacy `MINIMAX_GROUP_ID`. |
| Current MiniMax official docs index | Current platform docs list text, speech, video, image, music, file, OpenAI-compatible models, and Anthropic-compatible models; embeddings is not listed in the current official docs index. |

### Local Code Findings

| Area | Finding |
|---|---|
| Go request conversion | `relay/channel/minimax/embedding.go` converts OpenAI `input` to MiniMax native `texts`, preserves `type=query/db`, and defaults invalid/missing type to `query`. |
| Go response conversion | MiniMax `vectors` are converted to OpenAI-compatible `data[].embedding`; `total_tokens` maps to prompt/total usage. |
| URL normalization | `relay/channel/minimax/relay-minimax.go` maps embeddings to `{provider-root}/v1/embeddings` and normalizes base URLs ending in `/v1`. |
| Runtime exposure | `/v1/models` does not expose `embo-01`; `/v1/embeddings` for `embo-01` returns `model_not_found` because all `embo-01` abilities are disabled. |

### Runtime Evidence

| Probe | Result |
|---|---|
| Direct upstream `https://api.minimax.io/v1/embeddings` with native `texts` and `type=query` | HTTP 200 wrapper, MiniMax `base_resp.status_code=1002`, `rate limit exceeded(RPM)`. |
| Direct upstream `https://api.minimax.io/v1/embeddings` with native `texts` and `type=db` | HTTP 200 wrapper, MiniMax `base_resp.status_code=1002`, `rate limit exceeded(RPM)`. |
| Direct upstream `https://api.minimax.io/v1/embeddings` with OpenAI-shaped `input` | HTTP 200 wrapper, MiniMax `base_resp.status_code=2013`, missing required `texts`. |
| Direct upstream `https://api.minimax.chat/v1/embeddings` with the current global key | HTTP 200 wrapper, MiniMax `base_resp.status_code=2049`, `invalid api key`. |
| Direct upstream `https://api.minimax.io/v1/chat/completions` with the same DB key | HTTP 200 chat response, confirming the credential works for MiniMax text. |
| Router `GET /v1/models` | HTTP 200, 7 models, `embo-01` absent. |
| Router `POST /v1/embeddings` for `embo-01` | HTTP 503 `model_not_found`, no available channel under group `default`. |

### Commands Run

```bash
node "$HOME/.Codex/get-shit-done/bin/gsd-tools.cjs" graphify status
bin/clianything embeddings --format json
bin/clianything providers --all --format json
PATH=/usr/local/go/bin:$PATH go test ./common ./relay/common ./relay/channel/minimax ./controller -run 'TestGetEndpointTypesByChannelTypeMiniMax|TestGetRequestURLForEmbeddings|TestConvertEmbeddingRequest|TestDoResponseForEmbedding|TestListModels' -count=1
PATH=/usr/local/go/bin:$PATH go test ./relay/channel/minimax -run 'Test' -count=1
PATH=/usr/local/go/bin:$PATH go test ./controller -run 'TestListModelsRepresentativeOrder|TestListModelsAnthropicPayloadAndOrder|TestListModelsPayloadShapeAndPublicFields' -count=1
```

### Conclusion

Keep `embo-01` disabled. The Go adapter shape is correct for the native MiniMax embeddings API, and `api.minimax.io` is the viable host for the current key, but upstream still blocks real embedding generation with `rate limit exceeded(RPM)`. Do not reintroduce a split active `MiniMax - Embeddings` route, do not expose `embo-01` in `/v1/models`, and do not switch the consolidated MiniMax channel back to legacy `api.minimax.chat` for embeddings.
