# Atius AI Router

<!-- Badges -->
[![License](https://img.shields.io/github/license/giovannimnz/router-ai-atius)](https://github.com/giovannimnz/router-ai-atius)
[![Version](https://img.shields.io/github/v/tag/giovannimnz/router-ai-atius?filter=v*)](https://github.com/giovannimnz/router-ai-atius/releases)
[![New-API](https://img.shields.io/badge/New--API-0.12.14-blue)](https://github.com/QuantumNous/new-api)
[![i18n](https://img.shields.io/badge/i18n-7%20locales-green)](#internationalization)
[![Runtime](https://img.shields.io/badge/Podman-compatible-purple)](#container-runtime)

> **Unified LLM gateway** that aggregates **MiniMax, DeepSeek** and **40+ AI providers** behind a single **OpenAI / Anthropic-compatible API**. Fork of [QuantumNous/new-api](https://github.com/QuantumNous/new-api) hardened for the **Atius Capital** production stack.

---

## At a Glance

| Item | Value |
|------|-------|
| Fork URL | <https://github.com/giovannimnz/router-ai-atius> |
| Upstream URL | <https://github.com/QuantumNous/new-api> |
| Fork version | `0.12.14.2` (base `0.12.14` + suffix `.2`) |
| Latest models | `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.7-hs`, `DeepSeek-V3.2-Exp` |
| Stack | NewAPI (Go 1.22+) · FastAPI middleware · PostgreSQL 15 · Podman / Docker |
| Default port | `3301` (NewAPI), `3300` (middleware) |
| Public URL | `https://router.atius.com.br` (Cloudflare → Apache → :3300/:3301) |

---

## Why this fork exists

`QuantumNous/new-api` is a great open-source gateway, but the Atius deployment needs:

1. **A Python middleware** that enriches `/v1/models` with per-model metadata (pricing tiers, context windows, capability flags) — data that the upstream Go binary does not generate.
2. **A CJK→latin strip filter** that prevents MiniMax responses in Chinese / Japanese / Korean from leaking into Portuguese / English client output (`v1.7`, planned).
3. **Atius branding** (logo, footer, "Atius Router" titles, "Atius Capital" attribution) consistent across the SPA, embed assets, and API responses.
4. **First-class Podman support** — the entire stack runs on Podman quadlets in the Atius mesh.
5. **Bilingual (en + pt-BR) i18n** with deep validation and 0/0/0 sync guarantees per locale.

Everything else is identical to upstream — including pricing, billing, channels, model parsing, OAuth, WebAuthn.

---

## Technical Stack

| Layer | Technology | Why |
|-------|------------|-----|
| **Gateway** | Go 1.22+ · Gin · GORM v2 | High-throughput HTTP, low memory, easy to deploy as a single static binary |
| **Frontend** | React 19 · TypeScript · Rsbuild · Base UI · Tailwind CSS | Server-side rendering disabled (CSR-only SPA), embedded in Go binary via `embed` |
| **Middleware** | Python 3.11+ · FastAPI · Pydantic v2 | High-level model enrichment + easy integration with data science libs |
| **Database** | PostgreSQL 15 (container `db-newapi`) | Mature, ACID, JSON columns for flexible billing rules |
| **Cache** | Redis (go-redis) + in-memory LRU | Token bucket rate limiting + response cache |
| **Auth** | JWT · WebAuthn / Passkeys · OAuth (GitHub, Discord, OIDC) | All upstream-supported |
| **Orchestration** | Podman quadlets · Docker Compose (legacy) | Podman is the production runtime |
| **i18n** | i18next v26 + react-i18next + 7 locales | en, zh, fr, ja, pt-BR, ru, vi — synced to 0/0/0 |

---

## Available Models

The router currently exposes **6 MiniMax models** plus **2 DeepSeek models** in the default configuration. All MiniMax models are routed through the **Atius-MiniMax channel** in PostgreSQL.

| Model | Provider | Context | Max Output | Streaming | Tools | Notes |
|-------|----------|---------|------------|-----------|-------|-------|
| `MiniMax-M3` | MiniMax | 1,048,576 | 64,000 | ✅ | ✅ | Flagship — 1M context, deep reasoning |
| `MiniMax-M2.7` | MiniMax | 245,760 | 50,000 | ✅ | ✅ | Production standard |
| `MiniMax-M2.7-highspeed` | MiniMax | 245,760 | 50,000 | ✅ | ✅ | High-throughput variant |
| `MiniMax-M2.7-hs` | MiniMax | 245,760 | 50,000 | ✅ | ✅ | Short alias for `M2.7-highspeed` |
| `MiniMax-M2.5` | MiniMax | 245,760 | 50,000 | ✅ | ✅ | Legacy production |
| `MiniMax-M2.5-highspeed` | MiniMax | 245,760 | 50,000 | ✅ | ✅ | Legacy high-throughput |
| `deepseek-chat` | DeepSeek | 131,072 | 8,192 | ✅ | ❌ | Reasoning-class chat |
| `deepseek-reasoner` | DeepSeek | 131,072 | 65,536 | ✅ | ❌ | Long-form reasoning |

### MiniMax-M3 — 1M context, the new flagship

`MiniMax-M3` is the latest MiniMax API. The fork ships first-class metadata for it in the middleware (`model_detailed.py`) and in the database (channel abilities). 1,048,576 token context enables long-document analysis, multi-turn agentic workflows, and code-base-wide reasoning.

### `M2.7-hs` / `M2.5-hs` aliases

`M2.7-hs` and `M2.5-hs` are **vendor-friendly short aliases** mapped to their `-highspeed` siblings. The mapping is set in the **NewAPI channel config** (`model_mapping` column, JSONB) and the **Atius-Router middleware** (`KNOWN_MODELS` table) so both layers resolve them consistently.

---

## Container Runtime — Podman-first

The Atius infrastructure runs on **Podman** (`podman 4.x+`) with **systemd-managed quadlets**. Docker Compose is supported for development.

| Component | Container | Image | Port (host) | Network |
|-----------|-----------|-------|-------------|---------|
| Gateway | `new-api` | `ghcr.io/giovannimnz/router-ai-atius:local` | `3301` | `newapi-internal`, `atius-shared` |
| Middleware | `model-detailed` | `router-ai-atius-model-detailed:latest` | `3300` | `newapi-internal`, `atius-shared` |
| Database | `db-newapi` | `postgres:15-alpine` | (internal only) | `newapi-internal` |

### Podman quadlet example (`.container`)

```ini
[Unit]
Description=Atius Router (new-api)
After=network-online.target

[Container]
Image=ghcr.io/giovannimnz/router-ai-atius:local
PublishPort=3301:3000
Network=newapi-internal.network
Network=atius-shared.network
Volume=/srv/Atius/router/data:/data:Z
EnvironmentFile=/srv/Atius/router/.env
Environment=TZ=America/Sao_Paulo
Environment=LANG=pt_BR.UTF-8
AutoUpdate=registry
HealthCmd=curl -fsS http://localhost:3000/api/status
HealthInterval=30s

[Service]
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker Compose (development)

```bash
docker compose up -d
docker compose ps
docker compose logs -f new-api
```

---

## Routing Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                  Apache 2.4 (router.atius.com.br:443)              │
│  Cloudflare proxy → Let's Encrypt SSL → vhosts/site-routing       │
└────────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
   /v1/*               /docs  /openapi.json      /api/*  /  (SPA)
        │                       │                       │
        ▼                       ▼                       ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  FastAPI middle  │  │  FastAPI middle  │  │    NewAPI (Go)   │
│   (port 3300)     │  │   (port 3300)     │  │   (port 3301)    │
│                  │  │                  │  │                  │
│ Model enrichment │  │ OpenAPI schema   │  │  Relay / billing │
│ CJK strip (v1.7) │  │ API reference    │  │  Channels / auth │
│ Healthchecks     │  │                  │  │  Admin dashboard │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                                │                       │
                                └───────────┬───────────┘
                                            ▼
                              ┌──────────────────┐
                              │   PostgreSQL 15   │
                              │   (db-newapi)     │
                              │                  │
                              │ users · tokens   │
                              │ channels · logs  │
                              │ abilities · etc  │
                              └──────────────────┘
```

### Client request flow (OpenAI chat completions)

```
Client (OpenAI / Anthropic SDK)
   │
   ├─► POST /v1/chat/completions ──► Apache vhost
   │                                            │
   │                                            ▼
   │                                  NewAPI (Go) :3301
   │                                            │
   │                                            ├─► JWT verify (token → user)
   │                                            ├─► Distributor middleware
   │                                            │     Token → Abilities → Channel
   │                                            ├─► RelayFormat (OpenAI/Claude/Gemini)
   │                                            └─► Upstream adaptor (minimax)
   │                                                         │
   │                                                         ▼
   │                                                api.minimax.io/anthropic/v1
   │                                                         │
   │                                            ◄── SSE streaming ◄──┤
   │
   └─► POST /v1/messages ──► same path with RelayFormatClaude
```

### /v1/models enrichment flow

```
Client GET /v1/models
   │
   ▼
Apache → middleware :3300
   │
   ├─► Query NewAPI /api/models (Go side, channel + abilities)
   ├─► Merge with KNOWN_MODELS metadata (model_detailed.py)
   │     - context_length, max_tokens
   │     - capability flags (tools, vision, audio)
   │     - pricing tier (input/output/cache)
   │     - vendor alias map (M2.7-hs → M2.7-highspeed)
   ├─► Strip CJK if v1.7 enabled (planned)
   └─► OpenAI-compatible JSON response with enriched metadata
```

---

## Python Middleware — `model-detailed`

`model-detailed` is a **FastAPI** service that:

1. **Enriches `/v1/models`** with per-model metadata (capability flags, pricing, context windows).
2. **Resolves aliases** like `M2.7-hs` → `M2.7-highspeed` so the `/v1/models` listing is consistent.
3. **Exposes `/docs`** and `/openapi.json` for OpenAPI 3.1 schema introspection.
4. **Provides healthchecks** via `/healthz` and `/readyz` (used by the `docker-compose.yml` `healthcheck` section).

### Key file: `model_detailed.py`

```python
KNOWN_MODELS = {
    "MiniMax-M3": {
        "context_length": 1_048_576,
        "max_tokens": 64_000,
        "supports_tools": True,
        "supports_vision": False,
        "tier": "flagship",
    },
    "MiniMax-M2.7-highspeed": {
        "context_length": 245_760,
        "max_tokens": 50_000,
        "supports_tools": True,
        "tier": "standard-highspeed",
    },
    # ... M2.7, M2.5, M2.5-highspeed, M2.7-hs, M2.5-hs
}
```

The middleware rewrites the NewAPI `/v1/models` payload in-place — **zero changes needed** in the Go binary.

---

## CJK Strip Filter — v1.7 (planned)

MiniMax upstream responses sometimes contain **CJK characters** (Chinese / Japanese / Korean) leaking from model output, even when the user prompt is in Portuguese or English. Examples observed:

- `重新生成` (Chinese for "regenerate")
- `もう一度` (Japanese for "once more")

**v1.7 plan:** Add a regex post-filter in the NewAPI relay that strips CJK chars from response text before returning to the client. Implementation in `.planning/phases/v1.7-cjk-strip-filter/PLAN.md`.

```go
var cjkRegex = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]`)

func StripCJK(s string) string {
    return cjkRegex.ReplaceAllString(s, "")
}
```

Toggleable per-channel via `ChannelSettings.StripCJK bool`.

---

## Internationalization (i18n)

The frontend ships **7 locales**, all **synced 0/0/0** (missing/extras/untranslated):

| Locale | Code | Coverage | Source strings |
|--------|------|----------|----------------|
| English (base) | `en` | 100% | 4,525 keys |
| Chinese | `zh` | 100% | 4,525 keys |
| French | `fr` | 100% | 4,525 keys |
| Japanese | `ja` | 100% | 4,525 keys |
| **Brazilian Portuguese** | `pt-BR` | **94% translated**, 6% brand/tech names kept in EN | 4,525 keys |
| Russian | `ru` | 100% | 4,525 keys |
| Vietnamese | `vi` | 100% | 4,525 keys |

The `pt-BR` translation is the community contribution we plan to send upstream as "thanks" to QuantumNous for the open-source base.

### Sync report (enforced by `bun run i18n:sync`)

```json
{
  "base": "pt-BR.json",
  "locales": {
    "pt-BR": { "missingCount": 0, "extrasCount": 0, "untranslatedCount": 0 },
    "en":    { "missingCount": 0, "extrasCount": 0, "untranslatedCount": 0 },
    "zh":    { "missingCount": 0, "extrasCount": 0, "untranslatedCount": 0 },
    ...
  }
}
```

### Tests (vitest, 17 passing)

```
✓ src/i18n/__tests__/locales-integrity.test.ts (4 tests)
✓ src/i18n/__tests__/languages-config.test.ts (3 tests)
✓ src/i18n/__tests__/i18n-runtime.test.ts (7 tests)
✓ src/components/__tests__/language-switcher.test.tsx (2 tests)
✓ src/components/ui/dropdown-menu.test.tsx (2 tests)
```

Run with: `bun run test`

---

## Main Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/chat/completions` | Chat completions (OpenAI compat.) |
| POST | `/v1/messages` | Messages (Anthropic compat.) |
| POST | `/v1/completions` | Completions (legacy) |
| POST | `/v1/embeddings` | Embeddings |
| POST | `/v1/audio/speech` | Text-to-Speech |
| POST | `/v1/audio/transcriptions` | Speech-to-Text |
| POST | `/v1/images/generations` | Image generation |
| POST | `/v1/rerank` | Rerank |
| GET  | `/v1/models` | List models (**enriched** by middleware) |
| GET  | `/healthz` | Middleware liveness |
| GET  | `/readyz` | Middleware readiness |
| GET  | `/docs` | OpenAPI 3.1 interactive docs (Swagger UI) |
| GET  | `/openapi.json` | OpenAPI 3.1 schema (machine-readable) |
| GET  | `/api/status` | System status + API info |
| POST | `/api/user/register` | User registration |
| POST | `/api/user/login` | User login |

### Public vs internal URLs

| Service | Public (Cloudflare → Apache) | Internal (container network) |
|---------|------------------------------|-------------------------------|
| Middleware | `https://router.atius.com.br/v1/*` (Apache → :3300) | `http://model-detailed:3001` |
| NewAPI | `https://router.atius.com.br/api/*` (Apache → :3301) | `http://new-api:3000` |
| NewAPI SPA | `https://router.atius.com.br/` (Apache → :3301) | n/a |
| PostgreSQL | n/a (not exposed) | `postgres://db-newapi:5432` |

---

## Quick Start

### 1. Clone

```bash
git clone https://github.com/giovannimnz/router-ai-atius.git
cd router-ai-atius
```

### 2. Configure

```bash
cp .env.example .env
# Edit .env with your API keys
```

Required env vars:

| Var | Source | Notes |
|-----|--------|-------|
| `MINIMAX_API_KEY` | MiniMax dashboard | Token Plan key |
| `DEEPSEEK_API_KEY_*` | DeepSeek dashboard | 1 key per DeepSeek channel |
| `POSTGRES_PASSWORD` | Self-generated | Used by NewAPI to connect to `db-newapi` |
| `TELEGRAM_BOT_TOKEN` | BotFather | For Telegram notification integration |

### 3. Boot

```bash
# Podman (production)
podman play kube deployment.yaml

# Docker (development)
docker compose up -d
```

### 4. Verify

```bash
docker compose ps
curl http://localhost:3301/api/status
curl http://localhost:3300/healthz
```

### 5. Smoke-test the models

```bash
TOKEN=$(docker exec db-newapi psql newapi admin -t -c "SELECT key FROM tokens WHERE name = 'GiovanniMuniz';")
curl -X POST http://localhost:3301/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"MiniMax-M3","messages":[{"role":"user","content":"Hello"}],"max_tokens":50}'
```

---

## Usage Examples

### OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_TOKEN",
    base_url="https://router.atius.com.br/v1",
)

resp = client.chat.completions.create(
    model="MiniMax-M3",
    messages=[{"role": "user", "content": "2+2=?"}],
    max_tokens=10,
)
print(resp.choices[0].message.content)
```

### Anthropic SDK (via /v1/messages)

```python
import anthropic

client = anthropic.Anthropic(
    api_key="YOUR_TOKEN",
    base_url="https://router.atius.com.br",
)

msg = client.messages.create(
    model="MiniMax-M3",
    max_tokens=10,
    messages=[{"role": "user", "content": "2+2=?"}],
)
print(msg.content[0].text)
```

### Cherry Studio / CC Switch

The router advertises itself as **OpenAI / Anthropic compatible**. To connect Cherry Studio or CC Switch:

1. Set base URL to `https://router.atius.com.br/v1`
2. Set API key to your token
3. Pick any model from `/v1/models` (enriched listing)

---

## Atius Branding

All branding assets are bundled with the build:

- **`/logo.png`** + **`/logo.svg`** — Atius Router logo (replaces upstream new-api logo in nav and admin)
- **`/favicon.ico`** — Atius favicon
- **`<title>Atius Router</title>`** — page title
- **`<meta name="description" content="Unified AI API gateway and admin dashboard.">`**
- **Footer** — "© 2026 Atius Capital. Todos os direitos reservados."

Branding is applied at **build time** via the `Dockerfile`:

```dockerfile
# ATIUS BRANDING: replace ALL embedded assets in dist AFTER builder copy
RUN find ./web/default/dist -type f \( -name "*.png" -o -name "*.ico" -o -name "*.svg" \) -delete 2>/dev/null || true
COPY web/default/public/logo.png ./web/default/dist/logo.png
COPY web/default/public/logo.svg ./web/default/dist/logo.svg
COPY web/default/public/favicon.ico ./web/default/dist/favicon.ico
```

So the `new-api` Go binary at runtime serves the **Atius assets** with zero configuration changes.

---

## Channel Configuration

The MiniMax channel is configured in the database (`channels` table) with:

```sql
UPDATE channels SET
  test_model = 'MiniMax-M2.7',
  models = '["MiniMax-M3","MiniMax-M2.7","MiniMax-M2.7-highspeed","MiniMax-M2.7-hs","MiniMax-M2.5","MiniMax-M2.5-highspeed"]'::jsonb,
  model_mapping = '{}'::jsonb
WHERE id = 1;
```

| Field | Value | Why |
|-------|-------|-----|
| `type` | `35` (custom) | MiniMax relay type |
| `base_url` | `https://api.minimax.io` | MiniMax endpoint |
| `key` | `sk-cp-...` (Bearer) | Encrypted at rest with AES-256-GCM |
| `test_model` | `MiniMax-M2.7` | Used by the channel healthcheck |
| `models` | All 6 MiniMax models | Distributor matches these against token abilities |
| `model_mapping` | `{}` | Empty (no alias translation needed) |

---

## Deployment Topology

```
                    ┌──────────────┐
                    │  Cloudflare  │
                    │   (proxy)    │
                    └──────┬───────┘
                           │ HTTPS
                           ▼
                    ┌──────────────┐
                    │    Apache    │
                    │  (router.    │
                    │   atius)     │
                    └──────┬───────┘
                           │ vhost routing
        ┌──────────────────┼──────────────────┐
        │ /v1/* /docs      │ /api/*           │ /
        ▼                  ▼                  ▼
  ┌──────────┐       ┌──────────┐       ┌──────────┐
  │  model-  │       │  new-api │       │  new-api │
  │ detailed │       │  :3301   │       │  :3301   │
  │  :3300   │       │   (Go)   │       │   (SPA)  │
  └─────┬────┘       └─────┬────┘       └─────┬────┘
        │                  │                  │
        └────────────┬─────┴──────────────────┘
                     ▼
              ┌──────────────┐
              │  PostgreSQL  │
              │  (db-newapi) │
              └──────────────┘
```

3 containers, 1 database, 1 reverse proxy, 1 CDN.

---

## Maintenance Operations

### Sync fork with upstream

```bash
./fork-sync/bin/sync.sh
```

Pulls latest `QuantumNous/new-api:main`, merges, runs `i18n:sync`, runs `bun run test`, builds, restarts.

### Run the test suite

```bash
cd web/default
bun run test        # vitest, 17 tests
bun run typecheck   # tsc -b
bun run lint        # eslint
bun run i18n:sync   # ensure 0/0/0
```

### Rebuild and redeploy

```bash
docker build -t ghcr.io/giovannimnz/router-ai-atius:local .
docker stop new-api
docker rm new-api
docker run -d --name new-api --network router-ai-atius_newapi-internal --network atius-shared -p 3301:3000 \
  -v $(pwd)/data:/data --env-file .env ghcr.io/giovannimnz/router-ai-atius:local
```

### Purge Cloudflare cache (after deployment)

```bash
ZONE=5b998a5d911f5a4102b6179df7f4518d
curl -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE/purge_cache" \
  -H "X-Auth-Email: $CF_AUTH_EMAIL" -H "X-Auth-Key: $CF_GLOBAL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"prefixes":["router.atius.com.br/static/"]}'
```

---

## License

GNU Affero General Public License v3.0 — see [LICENSE](LICENSE).

Inherited from [QuantumNous/new-api](https://github.com/QuantumNous/new-api) (AGPL-3.0).

Atius Router is a community fork; **upstream branding is preserved** in source comments and the `FORK.md` document.

---

## References

- [QuantumNous/new-api](https://github.com/QuantumNous/new-api) — upstream
- [MiniMax API Reference](https://platform.minimax.io/docs/api-reference)
- [OpenAI API](https://platform.openai.com/docs/api-reference)
- [Anthropic API](https://docs.anthropic.com/en/api)
- [Podman Quadlets](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- [i18next v26 docs](https://www.i18next.com/)
