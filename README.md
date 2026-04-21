# atius-ai-router

Fork de [QuantumNous/new-api](https://github.com/QuantumNous/new-api) adaptado para o ecossistema Atius.

## O que é

Gateway LLM centralizado rodando em Docker, fornecendo:
- **Roteamento de modelos AI** via NewAPI (OpenAI-compatible API)
- **Gerenciamento de channels e tokens** via PostgreSQL
- **Middleware de enriquecimento** para metadados de modelos
- **CLI para agentes** (Bruno tests, newapi-cli)

## Stack

- NewAPI (calciumion/new-api) — gateway OpenAI-compatible
- PostgreSQL 15 — persistência de channels/tokens
- Python middleware — enriquecimento de /v1/models
- Docker Compose — orquestração

## Modelos Disponíveis

| Modelo | Provider | Context | Max Output |
|--------|----------|---------|------------|
| deepseek-chat | DeepSeek | 131072 | 8192 |
| deepseek-reasoner | DeepSeek | 131072 | 65536 |
| MiniMax-M2.7 | MiniMax | 245760 | 50000 |
| MiniMax-M2.5 | MiniMax | 245760 | 50000 |

## Setup

```bash
# Subir containers
docker compose up -d

# Testar endpoints
./scripts/run-bruno-tests.sh
```

## Endpoints

- **API Gateway:** `http://localhost:3300` (via middleware)
- **Admin API:** interno na rede Docker
- **Swagger Docs:** interno na rede Docker

## Scripts

- `scripts/sync-fork.sh` — sync com upstream
- `scripts/version-bump.sh` — versionamento semântico
- `scripts/run-bruno-tests.sh` — executar testes de API

## Git Workflow

```
Fork de: QuantumNous/new-api
Origin: giovannimnz/atius-ai-router
Upstream: QuantumNous/new-api

# Sync semanal (automático via GitHub Actions)
./scripts/sync-fork.sh --dry-run

# Version bump
./scripts/version-bump.sh --check
```

## Versionamento

Fork usa `X.Y.Z.N` onde:
- `X.Y.Z` = versão base do upstream
- `N` = suffix do fork (incrementado em cada sync)
