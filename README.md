# Atius AI Router

<!-- Badges -->
![License](https://img.shields.io/github/license/giovannimnz/router-ai-atius)
![Version](https://img.shields.io/github/v/tag/giovannimnz/router-ai-atius?filter=v*)
![New-API](https://img.shields.io/badge/New--API-0.12.14-blue)

## O que é

Gateway LLM centralizado que agrega MiniMax, DeepSeek e 40+ provedores AI atrás de uma API única compatível com OpenAI/Anthropic. Fork de [QuantumNous/new-api](https://github.com/QuantumNous/new-api) adaptado para o ecossistema Atius.

| Item | Valor |
|------|-------|
| Fork URL | https://github.com/giovannimnz/router-ai-atius |
| Parent URL | https://github.com/QuantumNous/new-api |
| Versão fork | `0.12.14.2` (base: 0.12.14 + suffix `.2`) |
| Stack | NewAPI (Go 1.22+) · PostgreSQL 15 · Python middleware |

## Stack Técnica

| Camada | Tecnologia |
|--------|------------|
| Gateway | Go + Gin + GORM v2 |
| Frontend | React 19 + TypeScript + Rsbuild + Radix UI + Tailwind CSS |
| Middleware | Python FastAPI (enriquecimento de modelos) |
| Banco | PostgreSQL 15 (container `db-newapi`) |
| Cache | Redis (go-redis) + in-memory |
| Auth | JWT, WebAuthn/Passkeys, OAuth (GitHub, Discord, OIDC) |
| Orquestração | Docker Compose |

## Modelos Disponíveis

| Modelo | Provider | Context | Max Output |
|--------|----------|---------|------------|
| `MiniMax-M2.7` | MiniMax | 245760 | 50000 |
| `MiniMax-M2.7-highspeed` | MiniMax | 245760 | 50000 |
| `MiniMax-M2.5` | MiniMax | 245760 | 50000 |
| `MiniMax-M2.5-highspeed` | MiniMax | 245760 | 50000 |
| `deepseek-chat` | DeepSeek | 131072 | 8192 |
| `deepseek-reasoner` | DeepSeek | 131072 | 65536 |

### Alias Highspeed

| Alias | Mapa para |
|-------|-----------|
| `MiniMax-M2.7-hs` | `MiniMax-M2.7-highspeed` |
| `MiniMax-M2.5-hs` | `MiniMax-M2.5-highspeed` |

Mapping feito automaticamente pelo `model_mapping` no canal MiniMax no DB.

## Arquitetura de Routing

```
Cliente (SDK OpenAI/Anthropic)
    │
    ├─► POST /v1/chat/completions ──► RelayFormatOpenAI ──► minimax adaptor ──► /v1/text/chatcompletion_v2
    │
    ├─► POST /v1/messages ──────────► RelayFormatClaude ──► minimax adaptor ──► /anthropic/v1/messages
    │
    ├─► GET  /v1/models ─────────────► middleware Python ──► enrichment ──────► resposta enriquecida
    │
    ├─► POST /v1/embeddings ─────────────────────────────────────────► minimax
    ├─► POST /v1/audio/speech ─────────────────────────────────────────► minimax TTS
    ├─► POST /v1/audio/transcriptions ─────────────────────────────────► minimax STT
    ├─► POST /v1/images/generations ───────────────────────────────────► minimax image
    └─► POST /v1/rerank ────────────────────────────────────────────────► minimax rerank

Channel Selection (distribute middleware):
    Token ─► Abilities table ─► matching (group, model) ─► channel_id
    ─► Channels table ─► base_url, model_mapping
```

## Endpoints Principais

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/v1/chat/completions` | Chat completions (OpenAI compat.) |
| POST | `/v1/messages` | Messages (Anthropic compat.) |
| POST | `/v1/completions` | Completions (legacy) |
| POST | `/v1/embeddings` | Embeddings |
| POST | `/v1/audio/speech` | Text-to-Speech |
| POST | `/v1/audio/transcriptions` | Speech-to-Text |
| POST | `/v1/images/generations` | Image generation |
| POST | `/v1/rerank` | Rerank |
| GET | `/v1/models` | Lista modelos (enriquecidos) |
| GET | `/api/status` | Status do sistema + API Info |
| POST | `/api/user/register` | Registro |
| POST | `/api/user/login` | Login |

### URLs de Acesso

| Serviço | URL | Notas |
|---------|-----|-------|
| Middleware (host) | `http://localhost:3300` | Via Python middleware |
| Middleware (Docker) | `http://model-detailed:3001` | Via rede `newapi-internal` |
| NewAPI (host) | `http://localhost:3301` | Direto (sem enrichment) |
| NewAPI (Docker) | `http://new-api:3000` | Via rede `atius-shared` |
| PostgreSQL (host) | `localhost:8746` | psycopg2 |

## Quick Start

```bash
# 1. Clone e entre no diretório
git clone https://github.com/giovannimnz/router-ai-atius.git
cd router-ai-atius

# 2. Configure variáveis de ambiente
cp .env.example .env
# Edite .env com suas chaves de API

# 3. Suba os containers
docker compose up -d

# 4. Verifique o status
docker compose ps

# 5. Teste o endpoint de models
curl http://localhost:3300/v1/models \
  -H "Authorization: Bearer <SEU_TOKEN>"
```

## Exemplos de Uso

### OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="<SEU_TOKEN>",
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
    api_key="<SEU_TOKEN>",
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
  -H "Authorization: Bearer <SEU_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 50
  }'

# Anthropic messages
curl -X POST https://router.atius.com.br/v1/messages \
  -H "Authorization: Bearer <SEU_TOKEN>" \
  -H "x-api-key: <SEU_TOKEN>" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## Preços MiniMax (por Million tokens)

| Modelo | Input | Output | Cache Read | Cache Write |
|--------|-------|--------|------------|-------------|
| M2.7 | $0.30 | $1.20 | $0.06 | $0.375 |
| M2.7-hs | $0.30 | $2.40 | $0.06 | $0.375 |
| M2.5 | $0.30 | $1.20 | $0.03 | $0.375 |
| M2.5-hs | $0.30 | $2.40 | $0.03 | $0.375 |

> Cache write tokens = 1.25× preço de input. Cache read tokens = 0.1× preço de input.

## Rate Limits

| Modelo | RPM | TPM |
|--------|-----|-----|
| Todos M2.x | 500 | 20,000,000 |

> TPM de 20M ≈ ~333K tokens/segundo. Limite prático = RPM 500 (~8.3 req/s).

## Scripts Disponíveis

| Script | Função |
|--------|--------|
| `scripts/sync-fork.sh` | Sync com upstream + version bump |
| `scripts/version-bump.sh` | Versionamento semântico X.Y.Z.N |
| `scripts/run-bruno-tests.sh` | Executar suite de testes Bruno CLI |
| `scripts/deploy-ghcr.sh` | Build + push para GHCR |
| `scripts/auto-sync-deploy.sh` | Sync + deploy automático |

## Git Workflow

```bash
# Remotes
origin   → https://github.com/giovannimnz/router-ai-atius.git
upstream → https://github.com/QuantumNous/new-api.git

# Sync semanal (automático via GitHub Actions)
./scripts/sync-fork.sh --dry-run

# Version bump check
./scripts/version-bump.sh --check

# Restaurar protected files após sync
git checkout HEAD -- integration/middleware/model_detailed.py
git checkout HEAD -- docker-compose.yml
```

## Versionamento

Fork usa `X.Y.Z.N`:
- `X.Y.Z` = versão base do upstream NewAPI
- `N` = suffix do fork (incrementado a cada sync)

```bash
cat VERSION  # 0.12.14.2
git tag -l "v0.12.*"
```

## Troubleshooting

```bash
# Containers não sobem
docker compose down && docker compose up -d
docker compose ps

# Teste direto no new-api (sem middleware)
docker exec new-api curl localhost:3000/v1/models

# Bruno tests falham
./scripts/run-bruno-tests.sh

# Ver logs
docker compose logs -f new-api
docker compose logs -f model-detailed
```

## Links

| Recurso | URL |
|---------|-----|
| Fork | https://github.com/giovannimnz/router-ai-atius |
| Parent | https://github.com/QuantumNous/new-api |
| Router UI | https://router.atius.com.br |
| Swagger Docs | interno na rede Docker |

---

_Last updated: 2026-05-31_
