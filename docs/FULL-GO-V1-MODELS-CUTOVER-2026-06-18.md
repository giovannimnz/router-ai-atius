---
title: Full-Go `/v1` and `/v1/models` Cutover
date: 2026-06-18
status: validated
scope: production
---

# Full-Go `/v1` and `/v1/models` Cutover - 2026-06-18

## Outcome

Production `https://router.atius.com.br/v1/` is now served directly by the Go router. The Python/FastAPI `model-detailed` middleware is retired from the canonical runtime path.

Do not reintroduce Python, `model-detailed`, `127.0.0.1:3300`, `127.0.0.1:3399`, or pod port `3001` as owner of `/v1/`, `/v1/models`, detailed model metadata, Codex embeddings, or provider routing.

## Runtime State

| Area | Final state |
|---|---|
| Public `/v1/` | Apache -> `http://127.0.0.1:3000/v1/` |
| Public `/v1/models` | Go controller/modelcatalog |
| Public `/health` | Apache -> `http://127.0.0.1:3000/api/status` |
| Pod | `atius-ai-router` |
| Running containers | infra, `postgres`, `redis`, `router-ai-atius` |
| Exposed host ports | only `3000` |
| Retired service | `container-model-detailed.service` inactive |
| Production image | `ghcr.io/giovannimnz/router-ai-atius:latest` |
| Validated image id | `e389110f98fb8e3fce80ac8cf691a04c1c74b6268d91d5fb304bb6f574344151` |

## Active Public Models

`/v1/models` with a valid token returns exactly the active working set below:

1. `MiniMax-M3`
2. `MiniMax-M2.7-highspeed`
3. `MiniMax-M2.7`
4. `gpt-5.5`
5. `gpt-5.4`
6. `gpt-5.4-mini`
7. `gpt-5.3-codex-spark`

Public catalog invariants:

- Root object is `{"data":[...]}` only.
- No top-level `object`, `success`, `first_id`, `last_id`, or `has_more`.
- No item-level `pricing_version`, `pricing_source`, or `pricing_estimated`.
- `MiniMax-M3` stays above older MiniMax models.
- `-highspeed` stays above the same family/version without `-highspeed`.
- Higher numeric versions sort first.
- `pro` sorts above `flash` when both are enabled in the same family.

## Provider State

| Provider/channel | Type | Runtime state | Reason |
|---|---:|---|---|
| `MiniMax` | `35` | active | OpenAI-compatible and Anthropic-compatible routes passed strict smoke |
| `OpenAI - Codex` | `57` | active | GPT chat routes passed strict smoke through channel 5 |
| `DeepSeek` | `43` | disabled | upstream returned `401 invalid api key` |
| `MiniMax` embeddings / `embo-01` | `35` | disabled | upstream returned `429 rate limit exceeded(RPM)` |
| Codex embeddings / `text-embedding-3-*` | `57` | disabled | upstream returned `429 insufficient_quota` |
| split legacy channels | mixed | disabled | replaced by consolidated provider channels |

Disabled routes must fail closed and must not select old split channels.

## Base URL Normalization

Go now tolerates provider base URLs with or without trailing `/v1`.

Implemented in:

- `relay/common/relay_utils.go`
- `relay/common/relay_utils_test.go`
- `relay/channel/minimax/relay-minimax.go`
- `relay/channel/deepseek/adaptor.go`

Rules:

- Trim whitespace and trailing slash.
- Avoid duplicated paths such as `/v1/v1/chat/completions`.
- For provider-root builders like MiniMax and DeepSeek, strip trailing `/v1` before appending Anthropic/native provider paths.
- Keep this behavior in Go, not in middleware or Apache rewrites.

Production proof: channel 1 `MiniMax` was temporarily set to `https://api.minimax.io/v1`, Go was restarted to refresh cache, the full active public smoke passed for OpenAI-compatible and Anthropic-compatible MiniMax routes, then the channel was reverted to `https://api.minimax.io`.

## Validation Commands

Do not paste real tokens into docs. Use environment variables only.

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius

PATH=/usr/local/go/bin:$PATH go test ./common ./controller ./service/modelcatalog ./relay/common ./relay/channel/minimax ./relay/channel/deepseek ./relay/channel/codex ./service/openaicompat -count=1
python3 -m py_compile tools/clianything.py scripts/smoke-provider-consolidation.py scripts/smoke-embeddings.py
python3 -m unittest discover -s tests -p 'test_clianything*.py'
bin/clianything status --strict
bin/clianything providers --all
sudo apache2ctl configtest
ATIUS_ROUTER_ACTIVE_ONLY=1 python3 scripts/smoke-provider-consolidation.py
```

Final observed catalog check:

```text
status=200
count=7
first=[MiniMax-M3, MiniMax-M2.7-highspeed, MiniMax-M2.7, gpt-5.5]
leaked_internal=false
keys=[data]
```

## Rollback Points

- Image rollback before `/v1` base URL normalization: `ghcr.io/giovannimnz/router-ai-atius:rollback-before-baseurl-v1-normalize-20260618122124`
- Image rollback before provider consolidation: `ghcr.io/giovannimnz/router-ai-atius:rollback-before-provider-consolidation-20260618080648`
- Apache backups:
  - `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf.bak-remove-model-detailed-20260618121341`
  - `/etc/apache2/sites-available/router.atius.com.br-le-ssl.conf.bak-full-go-source-*`
- Channel DB backups from the temporary `/v1` base URL probe:
  - `backups/clianything/20260618_124100_channels.sql`
  - `backups/clianything/20260618_124250_channels.sql`

## Sync Guards

Protect these files from upstream sync regressions:

- `controller/model.go`
- `controller/model_list_test.go`
- `service/modelcatalog/`
- `relay/common/relay_utils.go`
- `relay/common/relay_utils_test.go`
- `relay/channel/minimax/`
- `relay/channel/deepseek/`
- `relay/channel/codex/`
- `service/openaicompat/policy.go`
- `tools/clianything.py`
- `scripts/smoke-provider-consolidation.py`
- `docs/`
- `.planning/`

Abort a sync/deploy if any of these happen:

- `/v1/` points to Python/model-detailed again.
- `/v1/models` exposes `pricing_version`, `pricing_source`, or `pricing_estimated`.
- MiniMax/DeepSeek require active split channels again.
- `ChatGPT Subscription (Codex)` returns as the channel label for type `57`.
- Provider base URL ending `/v1` produces `/v1/v1/...`.

## Related Records

- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`
- `.planning/phases/phase-20-go-native-model-router/20-UAT.md`
- `.planning/phases/phase-20-go-native-model-router/20-VALIDATION.md`
- `.planning/phases/phase-20-go-native-model-router/20-VERIFICATION.md`
- `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/UPSTREAM-SYNC-GUARDS.md`
