# Atius AI Router

<!-- Badges -->
[![License](https://img.shields.io/github/license/giovannimnz/router-ai-atius)](https://github.com/giovannimnz/router-ai-atius)
[![Version](https://img.shields.io/github/v/tag/giovannimnz/router-ai-atius?filter=v*)](https://github.com/giovannimnz/router-ai-atius/releases)
[![New-API](https://img.shields.io/badge/New--API-0.12.14-blue)](https://github.com/QuantumNous/new-api)

## What is it

Unified LLM gateway that aggregates MiniMax, DeepSeek and 40+ AI providers behind a single OpenAI/Anthropic-compatible API. Fork of [QuantumNous/new-api](https://github.com/QuantumNous/new-api) adapted for the Atius ecosystem.

| Item | Value |
|------|-------|
| Fork URL | https://github.com/giovannimnz/router-ai-atius |
| Parent URL | https://github.com/QuantumNous/new-api |
| Fork version | `0.12.14.2` (base: 0.12.14 + suffix `.2`) |
| Stack | NewAPI (Go 1.22+) · PostgreSQL 15 · Python middleware |

## Technical Stack

| Layer | Technology |
|-------|------------|
| Gateway | Go + Gin + GORM v2 |
| Frontend | React 19 + TypeScript + Rsbuild + Radix UI + Tailwind CSS |
| Middleware | Python FastAPI (model enrichment) |
| Database | PostgreSQL 15 (container `db-newapi`) |
| Cache | Redis (go-redis) + in-memory |
| Auth | JWT, WebAuthn/Passkeys, OAuth (GitHub, Discord, OIDC) |
| Orchestration | Docker Compose |

## Available Models

| Model | Provider | Context | Max Output |
|-------|----------|---------|------------|
| `MiniMax-M2.7` | MiniMax | 245760 | 50000 |
| `MiniMax-M2.7-highspeed` | MiniMax | 245760 | 50000 |
| `MiniMax-M2.5` | MiniMax | 245760 | 50000 |
| `MiniMax-M2.5-highspeed` | MiniMax | 245760 | 50000 |
| `deepseek-chat` | DeepSeek | 131072 | 8192 |
| `deepseek-reasoner` | DeepSeek | 131072 | 65536 |

### Highspeed Aliases

| Alias | Maps to |
|-------|---------|
| `MiniMax-M2.7-hs` | `MiniMax-M2.7-highspeed` |
| `MiniMax-M2.5-hs` | `MiniMax-M2.5-highspeed` |

Mapping done automatically via `model_mapping` in the MiniMax channel in the database.

## Routing Architecture

```
Client (OpenAI/Anthropic SDK)
    │
    ├─► POST /v1/chat/completions ──► RelayFormatOpenAI ──► minimax adaptor ──► /v1/text/chatcompletion_v2
    │
    ├─► POST /v1/messages ──────────► RelayFormatClaude ──► minimax adaptor ──► /anthropic/v1/messages
    │
    ├─► GET  /v1/models ─────────────► Python middleware ──► enrichment ──────► enriched response
    │
    ├─► POST /v1/embeddings ─────────────────────────────────────────► minimax
    ├─► POST /v1/audio/speech ─────────────────────────────────────────► minimax TTS
    ├─► POST /v1/audio/transcriptions ─────────────────────────────────► minimax STT
    ├─► POST /v1/images/generations ───────────────────────────────────► minimax image
    └─► POST /v1/rerank ───────────────────────────────────────────────► minimax rerank

Channel Selection (distribute middleware):
    Token ─► Abilities table ─► matching (group, model) ─► channel_id
    ─► Channels table ─► base_url, model_mapping
```

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
| GET | `/v1/models` | List models (enriched) |
| GET | `/api/status` | System status + API Info |
| POST | `/api/user/register` | Registration |
| POST | `/api/user/login` | Login |

### Access URLs

| Service | URL | Notes |
|---------|-----|-------|
| Middleware (host) | `http://localhost:3300` | Via Python middleware |
| Middleware (Docker) | `http://model-detailed:3001` | Via `newapi-internal` network |
| NewAPI (host) | `http://localhost:3301` | Direct (no enrichment) |
| NewAPI (Docker) | `http://new-api:3000` | Via `atius-shared` network |
| PostgreSQL (host) | `localhost:8746` | psycopg2 |

## Quick Start

```bash
# 1. Clone and enter directory
git clone https://github.com/giovannimnz/router-ai-atius.git
cd atius-ai-router

# 2. Configure environment variables
cp .env.example .env
# Edit .env with your API keys

# 3. Start containers
docker compose up -d

# 4. Check status
docker compose ps

# 5. Test models endpoint
curl http://localhost:3300/v1/models \
  -H "Authorization: Bearer <YOUR_TOKEN>"
```

## Usage Examples

### OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="<YOUR_TOKEN>",
    base_url="https://router.atius.com.br/v1"
)

response = client.chat.completions.create(
    model="MiniMax-M2.7",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Explain quantum computing simply."}
    ],
    max_tokens=1024,
    temperature=0.7
)
print(response.choices[0].message.content)
```

### Anthropic SDK

```python
import anthropic

client = anthropic.Anthropic(
    api_key="<YOUR_TOKEN>",
    base_url="https://router.atius.com.br/v1"
)

message = client.messages.create(
    model="MiniMax-M2.7",
    max_tokens=1024,
    system="You are a helpful assistant.",
    messages=[
        {"role": "user", "content": "Explain quantum computing simply."}
    ]
)
print(message.content[0].text)
```

### curl

```bash
# Chat completions
curl -X POST https://router.atius.com.br/v1/chat/completions \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 50
  }'

# Anthropic messages
curl -X POST https://router.atius.com.br/v1/messages \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -H "x-api-key: <YOUR_TOKEN>" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## MiniMax Pricing (per Million tokens)

| Model | Input | Output | Cache Read | Cache Write |
|-------|-------|--------|------------|-------------|
| M2.7 | $0.30 | $1.20 | $0.06 | $0.375 |
| M2.7-hs | $0.30 | $2.40 | $0.06 | $0.375 |
| M2.5 | $0.30 | $1.20 | $0.03 | $0.375 |
| M2.5-hs | $0.30 | $2.40 | $0.03 | $0.375 |

> Cache write tokens = 1.25× input price. Cache read tokens = 0.1× input price.

## Rate Limits

| Model | RPM | TPM |
|-------|-----|-----|
| All M2.x | 500 | 20,000,000 |

> TPM of 20M ≈ ~333K tokens/second. Practical limit = RPM 500 (~8.3 req/s).

## Available Scripts

| Script | Function |
|--------|----------|
| `scripts/sync-fork.sh` | Sync with upstream + version bump |
| `scripts/version-bump.sh` | Semantic versioning X.Y.Z.N |
| `scripts/run-bruno-tests.sh` | Run Bruno CLI test suite |
| `scripts/deploy-ghcr.sh` | Build + push to GHCR |
| `scripts/auto-sync-deploy.sh` | Auto sync + deploy |

## Git Workflow

```bash
# Remotes
origin   → https://github.com/giovannimnz/router-ai-atius.git
upstream → https://github.com/QuantumNous/new-api.git

# Weekly sync (auto via GitHub Actions)
./scripts/sync-fork.sh --dry-run

# Version bump check
./scripts/version-bump.sh --check

# Restore protected files after sync
git checkout HEAD -- integration/middleware/model_detailed.py
git checkout HEAD -- docker-compose.yml
```

## Versioning

Fork uses `X.Y.Z.N`:
- `X.Y.Z` = upstream NewAPI base version
- `N` = fork suffix (incremented on each sync)

```bash
cat VERSION  # 0.12.14.2
git tag -l "v0.12.*"
```

## Troubleshooting

```bash
# Containers won't start
docker compose down && docker compose up -d
docker compose ps

# Test directly on new-api (no middleware)
docker exec new-api curl localhost:3000/v1/models

# Bruno tests fail
./scripts/run-bruno-tests.sh

# View logs
docker compose logs -f new-api
docker compose logs -f model-detailed
```

## Links

| Resource | URL |
|---------|-----|
| Fork | https://github.com/giovannimnz/router-ai-atius |
| Parent | https://github.com/QuantumNous/new-api |
| Router UI | https://router.atius.com.br |
| Swagger Docs | internal on Docker network |

---

_Last updated: 2026-05-31_
