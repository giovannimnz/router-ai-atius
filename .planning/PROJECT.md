# PROJECT.md - atius-ai-router

## Project Overview

Gateway LLM centralizado (NewAPI) fork mantido no GitHub como `giovannimnz/atius-ai-router`, rodando em Docker como roteador de modelos AI para o ecossistema Atius. Baseado em [QuantumNous/new-api](https://github.com/QuantumNous/new-api), com PostgreSQL para persistência de channels, tokens e configurações.

**URL pública:** https://router.atius.com.br
**Git remote:** `https://github.com/giovannimnz/atius-ai-router.git`
**Parent upstream:** `https://github.com/QuantumNous/new-api.git`
**Stack:** Go (NewAPI), Docker Compose, PostgreSQL 15

## Architecture Summary

- **NewAPI Gateway** → IP 192.168.0.2:3000 (rede atius-shared), porta 3300 host (via middleware)
- **PostgreSQL** → Porta 8746 (host), IP 192.168.0.x:5432
- **Middleware Python** → Porta 3300 (host), proxy enriquecendo `/v1/models`
- **Consumidores:** Open-WebUI, OpenClaw, GSD-2, Search-Engine
- **Providers:** DeepSeek (3 chaves rotativas), Qwen, Kimi/Moonshot, MiniMax

## Container Network

```
Rede: atius-shared (192.168.0.0/20)
├── new-api:       192.168.0.2:3000
├── model-detailed: 192.168.0.x:3001 (via middleware 3300:host)
└── db-newapi:     192.168.0.x:5432
```

## Fork Workflow

Este projeto segue o workflow de fork documentado em `FORK_MIGRATION.md`:

- **Versionamento:** `X.Y.Z.N` (upstream base + suffix incremental)
- **Sync:** `./scripts/sync-fork.sh` — fetch upstream + merge + restore overrides + bump + push
- **Override protection:** `model_detailed.py`, `.planning/`, `docker-compose.yml` são protegidos
- **Release:** Tags git `vX.Y.Z.N` via `./scripts/version-bump.sh`

### Git Remotes

```
origin  → https://github.com/giovannimnz/atius-ai-router.git (fetch/push)
upstream → https://github.com/QuantumNous/new-api.git (fetch only)
```

## Active Milestones

### v1.3 — Testing Infrastructure & CLI (current)
Estabelecer infraestrutura de testes com Bruno CLI e documentação.

**Goal:** Suite de testes de API automatizados e skill para agentes.

**Target deliverables:**
1. Bruno CLI instalado e configurado
2. Collection `atius-router-tests/` com 5 requests
3. Skill `bruno-cli` criada em `~/.agents/skills/`
4. Script `run-bruno-tests.sh` funcional

## Technical Decisions

| Decisão | Racional | Data |
|---------|----------|------|
| Middleware Python para enrichment | NewAPI é closed-source (imagem Docker pré-built) | 2026-04-14 |
| Fork suffix versioning `X.Y.Z.N` | Mantém rastreabilidade com upstream | 2026-04-21 |
| Bruno CLI para testing | Formato texto legível, versionável com Git | 2026-04-21 |
| Delay 500ms entre requests | NewAPI tem rate limiting | 2026-04-21 |
| IP 192.168.0.2 para new-api | IP na rede atius-shared (não usar 172.20.0.x) | 2026-04-21 |

## API Endpoints

| Endpoint | Método | Descrição | Status |
|----------|--------|-----------|--------|
| `/v1/models` | GET | Lista modelos (enriquecidos) | ✅ OK |
| `/v1/chat/completions` | POST | Chat completion | ✅ OK |

### Modelos Disponíveis

| Modelo | Provider | Context | Max Output | Status |
|--------|----------|---------|------------|--------|
| deepseek-chat | DeepSeek | 131072 | 8192 | ✅ OK |
| deepseek-reasoner | DeepSeek | 131072 | 65536 | ✅ OK |
| MiniMax-M2.7 | MiniMax | 245760 | 50000 | ✅ OK |
| MiniMax-M2.5 | MiniMax | 245760 | 50000 | ✅ OK |

## Constraints

- NewAPI é closed-source (imagem Docker pré-construída) — customização via API admin e DB
- Customizações locais não existem no upstream — nunca serão sobrescritas exceto `docker-compose.yml`
- GitHub MCP sem credentials — `gh` CLI não autenticado

## Last Updated
2026-04-21
