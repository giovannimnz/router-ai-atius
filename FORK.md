# FORK.md — atius-ai-router Technical Documentation

Fork de [QuantumNous/new-api](https://github.com/QuantumNous/new-api) adaptado para o ecossistema Atius.

## Overview

| Item | Valor |
|------|-------|
| Fork URL | https://github.com/giovannimnz/atius-ai-router |
| Parent URL | https://github.com/QuantumNous/new-api |
| Fork Version | `0.12.14.2` (base: 0.12.14 + suffix) |
| Git Branch | `main` |
| Stack | NewAPI (Go) + PostgreSQL 15 + Python middleware |

## Table of Contents

1. [Fork Purpose](#1-fork-purpose)
2. [Local Modifications](#2-local-modifications)
3. [Sync Workflow](#3-sync-workflow)
4. [Versioning](#4-versioning)
5. [Protected Files](#5-protected-files)
6. [Container Architecture](#6-container-architecture)
7. [API Endpoints](#7-api-endpoints)

---

## 1. Fork Purpose

Este fork adapta o NewAPI para o ecossistema Atius:

- **Gateway LLM centralizado** — roteamento de modelos AI para todo o domínio
- **Middleware de enriquecimento** — metadados completos para DeepSeek e MiniMax
- **PostgreSQL** — persistência de channels, tokens e configurações
- **Bruno CLI tests** — suite de testes de API versionada

## 2. Local Modifications

### 2.1 Middleware de Enriquecimento

**Arquivo:** `integration/middleware/model_detailed.py`

Proxy Python que intercepta `GET /v1/models` e adiciona metadados:

```python
# Modelos enriquecidos
- deepseek-chat:    131072 context, 8192 max output
- deepseek-reasoner: 131072 context, 65536 max output
- MiniMax-M2.7:     245760 context, 50000 max output
- MiniMax-M2.5:     245760 context, 50000 max output
```

### 2.2 Docker Compose Customizado

**Arquivo:** `docker-compose.yml`

Mudanças vs upstream:
- Container `new-api` com limits de CPU customizados
- Container `model-detailed` (middleware Python) exposto na porta 3300
- Redes `atius-shared` e `newapi-internal`
- Volumes para `data/` e `data/postgres_data/`

### 2.3 Bruno Tests Suite

**Diretório:** `integration/bruno-tests/atius-router-tests/`

Suite de testes de API:
- `list-models.bru` — GET /v1/models
- `deepseek-chat.bru` — POST /v1/chat/completions
- `deepseek-reasoner.bru` — POST /v1/chat/completions
- `minimax-m27.bru` — POST /v1/chat/completions
- `minimax-m25.bru` — POST /v1/chat/completions

Executar: `./scripts/run-bruno-tests.sh`

### 2.4 Agent Harness

**Diretório:** `agent-harness/`

CLI Click para gerenciar NewAPI via agentes:
- `container` — status/start/stop/restart
- `channel` — list/create/delete
- `model` — list/enabled/disabled
- `api` — status/health

### 2.5 Scripts Customizados

| Script | Função |
|--------|--------|
| `scripts/sync-fork.sh` | Merge upstream + proteção + version bump |
| `scripts/version-bump.sh` | Versionamento X.Y.Z.N baseado em upstream |
| `scripts/run-bruno-tests.sh` | Executar suite de testes Bruno |

### 2.6 GitHub Actions Workflows

| Workflow | Gatilho | Função |
|----------|---------|--------|
| `sync.yml` | Diário 03:00 UTC + manual | Sync automático com upstream |
| `release.yml` | Tags `v*` | GitHub Release |

## 3. Sync Workflow

### Fluxo Automático

```
GitHub Actions (daily 03:00 UTC)
    ↓
sync.yml: sync-check job
    ↓ (has_changes == true)
sync.yml: sync job
    ↓
sync-fork.sh
    ├─ git fetch upstream
    ├─ git merge upstream/main -X theirs
    ├─ Restore protected files
    ├─ Restore fork-only files
    ├─ git commit (if changes)
    ├─ version-bump.sh
    ├─ git push origin main
    └─ git push origin vX.Y.Z.N
    ↓
release.yml: (detecta tag)
    └─ GitHub Release criado
```

### Fluxo Manual

```bash
# Sync com upstream (dry-run)
./scripts/sync-fork.sh --dry-run

# Sync completo
./scripts/sync-fork.sh

# Com estratégia "ours" (preferir fork em conflitos)
./scripts/sync-fork.sh --strategy ours
```

### Estratégias de Merge

| Estratégia | Comportamento |
|------------|---------------|
| `theirs` (default) | Prefere mudanças do upstream em conflitos |
| `ours` | Prefere mudanças do fork em conflitos |

## 4. Versioning

Fork usa `X.Y.Z.N` onde:
- `X.Y.Z` = versão base do upstream NewAPI (de git tags)
- `N` = suffix do fork (incrementado em cada sync)

**Exemplo:** `0.12.14.2` → base `0.12.14`, suffix `.2`

### Version Bump Logic

```
Se upstream base mudou → suffix = 1
Se upstream base igual → suffix++
```

### Version File

```bash
cat VERSION  # 0.12.14.2
```

### Tags

Tags usam formato `vX.Y.Z.N`:
```bash
git tag -l "v0.12.*"
```

## 5. Protected Files

Arquivos que NUNCA são sobrescritos pelo merge do upstream:

| Arquivo | Proteção | Razão |
|---------|----------|-------|
| `docker-compose.yml` | git checkout HEAD | Configuração Atius (redes, portas) |
| `integration/middleware/model_detailed.py` | git checkout HEAD | Lógica custom de enrichment |
| `integration/middleware/model_details.py` | git checkout HEAD | Versão anterior (backup) |
| `integration/middleware/model_enrichment.py` | git checkout HEAD | Versão anterior (backup) |

Arquivos que existem SÓ no fork (verificados pós-merge):

| Arquivo/Dir | Razão |
|-------------|-------|
| `.planning/` | Documentação Atius (roadmap, milestones) |
| `agent-harness/` | CLI custom para agentes |
| `integration/bruno-tests/` | Suite de testes de API |
| `scripts/run-bruno-tests.sh` | Wrapper de testes |
| `.github/workflows/sync.yml` | Workflow de sync diário |
| `.github/workflows/release.yml` | Workflow de release |

### Restore Commands

Se um protected file for sobrescrito:

```bash
git checkout HEAD -- integration/middleware/model_detailed.py
git checkout HEAD -- docker-compose.yml
git commit -m "chore: restore fork overrides"
git push origin main
```

## 6. Container Architecture

```
Rede: atius-shared (192.168.0.0/20)
Rede: newapi-internal (172.20.0.0/16)

┌─────────────────────────────────────────────────────┐
│  new-api (calciumion/new-api)                      │
│  IP: 192.168.0.2:3000                             │
│  Exposes: 3000 (interno)                          │
│  Limits: 0.5 CPU                                  │
└─────────────────────────────────────────────────────┘
                        ↑
┌─────────────────────────────────────────────────────┐
│  model-detailed (middleware Python)                 │
│  IP: 192.168.0.x:3001                             │
│  Host: 0.0.0.0:3300 → 3001                       │
│  Limits: 0.1 CPU                                  │
│  Enriches: GET /v1/models                          │
└─────────────────────────────────────────────────────┘
                        ↑
┌─────────────────────────────────────────────────────┐
│  db-newapi (postgres:15-alpine)                   │
│  IP: 192.168.0.x:5432                             │
│  Host: 0.0.0.0:8746 → 5432                       │
│  Limits: 0.5 CPU                                  │
│  Database: newapi                                  │
└─────────────────────────────────────────────────────┘
```

### URLs

| Serviço | URL | Notas |
|---------|-----|-------|
| Middleware (host) | `http://localhost:3300` | Via Python middleware |
| Middleware (Docker) | `http://model-detailed:3001` | Via newapi-internal |
| NewAPI (Docker) | `http://new-api:3000` | Via atius-shared |
| PostgreSQL (host) | `localhost:8746` | psycopg2 connection |

## 7. API Endpoints

### 7.1 /v1/models

**Método:** GET

Retorna lista de modelos disponíveis com metadados enriquecidos.

```bash
curl http://localhost:3300/v1/models \
  -H "Authorization: Bearer $TOKEN"
```

**Resposta (exemplo):**
```json
{
  "data": [
    {
      "id": "deepseek-chat",
      "object": "model",
      "created": 1735689600,
      "owned_by": "deepseek",
      "name": "DeepSeek V3.2",
      "context_length": 131072,
      "top_provider": {
        "max_completion_tokens": 8192
      },
      "pricing": {
        "prompt": "0.00000028",
        "completion": "0.00000042",
        "prompt_cache_hit": "0.000000028"
      }
    }
  ],
  "object": "list",
  "success": true
}
```

**Header:** `X-Model-Metadata-Enriched: true` (presente quando enrichment ativo)

### 7.2 /v1/chat/completions

**Método:** POST

Chat completion standard OpenAI-compatible.

```bash
curl -X POST http://localhost:3300/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 50
  }'
```

## 8. Channels & Tokens

Channels e tokens são gerenciados via API admin do NewAPI (interno) ou via CLI (`agent-harness/`).

Tokens de API disponíveis:
- DeepSeek: múltiplos channels com key rotation
- MiniMax: configurado
- Kimi/Moonshot: disponível no upstream

## 9. Git Remotes

```bash
origin   → https://github.com/giovannimnz/atius-ai-router.git (fetch/push)
upstream → https://github.com/QuantumNous/new-api.git (fetch only)
```

## 10. Troubleshooting

### Sync falhou com conflitos

```bash
git merge --abort
./scripts/sync-fork.sh --strategy ours
```

### Protected file sobrescrito

```bash
git checkout HEAD -- integration/middleware/model_detailed.py
git checkout HEAD -- docker-compose.yml
git commit -m "chore: restore overrides" && git push
```

### Containers não sobem

```bash
cd ~/docker/ai-apps/atius-ai-router
docker compose down && docker compose up -d
docker compose ps
```

### Bruno tests falham

```bash
./scripts/run-bruno-tests.sh .  # Ver saída
docker exec new-api curl localhost:3000/v1/models  # Teste direto
```

## 11. Links

- **Fork:** https://github.com/giovannimnz/atius-ai-router
- **Parent:** https://github.com/QuantumNous/new-api
- **Router UI:** https://router.atius.com.br (via Apache proxy)
- **Swagger Docs:** interno na rede Docker

---

_Last updated: 2026-04-21_
