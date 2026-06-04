# PROJECT.md — atius-ai-router

## Project Overview

Gateway LLM centralizado (router-ai-atius) fork mantido no GitHub como
`giovannimnz/router-ai-atius`, rodando em **Podman Compose** (rootless) como
roteador de modelos AI para o ecossistema Atius. Baseado em
[QuantumNous/new-api](https://github.com/QuantumNous/new-api) com PostgreSQL
para persistência de channels, tokens e configurações.

> **Rebrand v2.11 (2026-06-02):** nome do projeto, containers, network,
> DB, host port e brand "Atius" próprios. Veja [[#rebrand-v211-changes]].

**URL pública:** https://router.atius.com.br
**Git remote:** `https://github.com/giovannimnz/router-ai-atius.git`
**Parent upstream:** `https://github.com/QuantumNous/new-api.git`
**Stack:** Go 1.22+ (new-api), Python 3.11 (FastAPI middleware), PostgreSQL 15,
**Podman Compose** (rootless)

## Architecture Summary

- **router-ai-atius Gateway** (Go) → container `router-ai-atius`,
  porta host **3030** → 3000 (Apache fronts 443 → 3030)
- **router-ai-atius-model-detailed** (FastAPI middleware) → porta host
  **3300** → 3001, proxy enriquecendo `/v1/models`
- **router-ai-atius-db** (PostgreSQL 15) → porta 5432 (internal only,
  network `atius-ai-router_internal`)
- **router-ai-atius-redis** (Redis 7) → porta 6379 (internal only,
  cache + rate limiting)
- **Consumidores:** Open-WebUI, OpenClaw, GSD-2, Search-Engine
- **Providers:** DeepSeek (3 chaves rotativas), Qwen, Kimi/Moonshot, MiniMax

## Container Network (rebrand v2.11)

```
Network: atius-ai-router_internal (rootless podman bridge)
├── router-ai-atius               :3000   (Go AI gateway)
├── router-ai-atius-model-detailed :3001  (FastAPI middleware)
├── router-ai-atius-db            :5432   (PostgreSQL 15, DBRouterAiAtius)
└── router-ai-atius-redis         :6379   (Redis 7 cache)

Host ports:
  3030 → router-ai-atius:3000            (Apache ProxyPass)
  3300 → router-ai-atius-model-detailed:3001
```

## Rebrand v2.11 changes

| Aspect | v1.x (new-api) | v2.11 (router-ai-atius) |
|--------|----------------|--------------------------|
| Container `app` | `new-api` | `router-ai-atius` |
| Container `db` | `db-newapi` | `router-ai-atius-db` |
| Container `cache` | `redis-newapi` | `router-ai-atius-redis` |
| Container `middleware` | `model-detailed` | `router-ai-atius-model-detailed` |
| Network | `newapi-internal` | `atius-ai-router_internal` |
| DB name | `newapi` | `DBRouterAiAtius` |
| Host port (api) | `3301:3000` | `3030:3000` |
| Host port (middleware) | `3300:3001` | `3300:3001` |
| Image tag | `:local` | `:latest` (canonical) |
| Tag source | (registry) | `scripts/podman-prepare-images.sh` |
| Runtime | docker compose | **podman compose** (rootless) |
| Brand | QuantumNous | Atius |

## Podman — operational commands

```bash
# Validate (no runtime needed)
./scripts/podman-validate.sh

# Populate :latest images
./scripts/podman-prepare-images.sh
# Or pin: ROUTER_AI_ATIUS_VERSION=v2.11.1-rebrand ./scripts/podman-prepare-images.sh

# Bring the stack up
./scripts/podman-up.sh
# Verify: curl http://localhost:3030/api/status

# Migrate from Docker (one-shot)
./scripts/podman-migrate-from-docker.sh

# Quadlets (systemd, survives reboots)
./scripts/podman-quadlets-install.sh
```

See `docs/PODMAN.md` for full architecture and operations.

## Fork Workflow

Este projeto segue o workflow de fork documentado em `FORK_MIGRATION.md`:

- **Versionamento:** `X.Y.Z.N` (upstream base + suffix incremental)
- **Sync:** `./scripts/sync-fork.sh` — fetch upstream + merge + restore overrides + bump + push
- **Override protection:** `model_detailed.py`, `.planning/`, `docker-compose.yml` são protegidos
- **Release:** Tags git `vX.Y.Z.N` via `./scripts/version-bump.sh`

### Git Remotes

```
origin   → https://github.com/giovannimnz/router-ai-atius.git (fetch/push)
upstream → https://github.com/QuantumNous/new-api.git (fetch only)
```

## Active Milestones

### v1.6 — Internacionalização PT-BR (current)
Adicionar Português do Brasil como idioma principal. Tradução completa de frontend, backend e DB.

**Goal:** Atius Router 100% em Português do Brasil.

**Target deliverables:**
1. Frontend `i18n/locales/pt.json` com todas as chaves traduzidas
2. Backend `i18n/locales/pt.yaml` com todas as traduções Go
3. DB: `Language=pt`, `Logo=/logo.png`, `SystemName=Atius Router`
4. BRANCH: `feat/portuguese-translation` para PR ao upstream
5. README.pt.md e documentação em PT-BR

**Status:** ✅ Translation merged in PR #2 (commit `728bb2e28`). Branch
`feat/portuguese-translation-clean` carries additional v2.11 work
(rebrand, SSO, /login, podman, codebase map) not yet merged.

### v1.7 — Documentação PT-BR
README principal em PT-BR, `README.en.md` como cópia, docs folder, fork-sync cleanup para `~/fork-sync/`.

### v1.8 — Podman Migration
Migrar Docker Compose → Podman Compose. Limpar Docker references.

**Status:** ✅ Mostly done as of 2026-06-04:
- `podman-compose.yml` rebrand v2.11 + `:latest` canonical
- 4 systemd quadlets (`podman/quadlets/*.container`)
- 6 helper scripts (`scripts/podman-{up,down,validate,prepare-images,migrate,quadlets-install}.sh`)
- `docs/PODMAN.md` complete
- `docker-compose.yml` aligned with rebrand (kept for legacy)

### v1.4 — Model Aliases & Token Management

**Goal:** Migracao gradual de `-highspeed` para `-hs` sem impacto nos clientes.

**Target deliverables:**
1. Modelos `MiniMax-M2.7-hs` e `MiniMax-M2.5-hs` adicionados ao catalog
2. `model_mapping` no canal MiniMax redirecionando `-hs` → `-highspeed`
3. Pricing configurado para ambos aliases
4. API key `Giovanni-Acc` criada com quota ilimitada

## Technical Decisions

| Decisão | Racional | Data |
|---------|----------|------|
| Middleware Python para enrichment | new-api é closed-source (imagem Docker pré-built) | 2026-04-14 |
| Fork suffix versioning `X.Y.Z.N` | Mantém rastreabilidade com upstream | 2026-04-21 |
| Bruno CLI para testing | Formato texto legível, versionável com Git | 2026-04-21 |
| Delay 500ms entre requests | new-api tem rate limiting | 2026-04-21 |
| Model aliases via `model_mapping` DB | Substitui necessidade de customizar relay Go code | 2026-05-07 |
| **Podman rootless em vez de Docker** | Sem daemon, sem sudo, systemd quadlets nativos | 2026-06-02 |
| **Host port 3030 (não 3301)** | Libera 3000 pro pm2web dashboard | 2026-06-02 |
| **Image tag `:latest` canônico** | Convençaõ Docker community, sem versionar compose | 2026-06-04 |
| **Network `atius-ai-router_internal`** | Identidade Atius, não new-api legacy | 2026-06-02 |

## API Endpoints

| Endpoint | Método | Descrição | Status |
|----------|--------|-----------|--------|
| `/v1/models` | GET | Lista modelos (enriquecidos) | ✅ OK |
| `/v1/chat/completions` | POST | Chat completion | ✅ OK |

## Modelos Disponíveis

### MiniMax (Canal 1 — Token Plan)
| Modelo (Router ID) | Upstream Alias | Provider | Context | Max Output | Input $/1M | Output $/1M | Status |
|--------------------|----------------|----------|---------|------------|------------|-------------|--------|
| MiniMax-M2.7 | MiniMax-M2.7 | MiniMax | 245760 | 50000 | $0.30 | $1.20 | ✅ OK |
| MiniMax-M2.7-highspeed | MiniMax-M2.7-highspeed | MiniMax | 245760 | 50000 | $0.30 | $1.20 | ✅ OK |
| MiniMax-M2.7-hs | → MiniMax-M2.7-highspeed | MiniMax | 245760 | 50000 | $0.30 | $1.20 | ✅ OK |
| MiniMax-M2.5 | MiniMax-M2.5 | MiniMax | 245760 | 50000 | $0.30 | $1.20 | ✅ OK |
| MiniMax-M2.5-highspeed | MiniMax-M2.5-highspeed | MiniMax | 245760 | 50000 | $0.30 | $1.20 | ✅ OK |
| MiniMax-M2.5-hs | → MiniMax-M2.5-highspeed | MiniMax | 245760 | 50000 | $0.30 | $1.20 | ✅ OK |

### DeepSeek (Canal 2)
| Modelo | Provider | Context | Max Output | Input $/1M | Output $/1M | Status |
|--------|----------|---------|------------|------------|-------------|--------|
| deepseek-v4-flash | DeepSeek | 131072 | 8192 | $0.14 | $0.28 | ✅ OK |
| deepseek-v4-pro | DeepSeek | 131072 | 65536 | $0.435 | $0.87 | ✅ OK |

**Nota:** Modelos `-hs` usam `model_mapping` no DB para redirecionar ao upstream `-highspeed`. O cliente envia `MiniMax-M2.7-hs`, o router converte para `MiniMax-M2.7-highspeed` antes de enviar ao provider. Logs e listagem mostram o ID do router (`-hs`).

## API Keys (Tokens)

| ID | Nome | Quota | Expiração | Modelos | Status |
|----|------|-------|-----------|---------|--------|
| 1 | Giovanni Hermes | Ilimitada | Nunca | Todos | ✅ Ativo |
| 3 | Alfred Router Key | Ilimitada | Nunca | Todos | ✅ Ativo |
| 4 | Gomes | Ilimitada | Nunca | Todos | ✅ Ativo |
| 5 | Giovanni-Acc | Ilimitada | Nunca | Todos | ✅ Ativo |

## Constraints

- new-api é closed-source (imagem Docker pré-construída) — customização via API admin e DB
- Customizações locais não existem no upstream — nunca serão sobrescritas exceto `docker-compose.yml`
- GitHub MCP sem credentials — `gh` CLI não autenticado
- Model aliases via `model_mapping` — implementado no DB, não requer código Go customizado
- Brand "QuantumNous/new-api" é **protegido** (AGENTS.md Rule 5) — nunca remover
- Podman 4.4+ requerido (quadlets in-tree desde 4.4)

## Obsidian cross-references

- `61-Incidents/2026-06-04-router-atius-503-new-api-crash` — fix do 503
  (db isolado da rede `atius-ai-router_internal`)
- `61-Incidents/2026-06-04-podman-cherry-pick-main` — cherry-pick Podman
- `61-Incidents/2026-06-04-podman-full-rebrand-v211` — rebrand v2.11 completo
- `61-Incidents/2026-06-04-podman-latest-tag-strategy` — `:latest` decision
- `61-Incidents/2026-06-04-podman-pre-rebrand-cleanup` — este commit
  (docker-compose.yml + PROJECT.md rebrand)

## Last Updated
2026-06-04
