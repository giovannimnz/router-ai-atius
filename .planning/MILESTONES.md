# MILESTONES.md - Project Milestone History

## v1.1 — DeepSeek Model Metadata Enrichment (Completed 2026-04-14)

**Goal:** Enriquecer endpoint `/v1/models` com metadados DeepSeek completos
**Shipped:**
- Middleware Python reverse proxy (integration/middleware/model_enrichment.py)
- GET /v1/models retorna JSON com context_length, pricing, max_completion_tokens, name
- docker-compose.yml atualizado com serviço model-enrichment
- deepseek-chat: 131072 contexto, 8192 max output, pricing por token
- deepseek-reasoner: 131072 contexto, 65536 max output, pricing por token
- Proxy transparente para /v1/chat/completions e demais endpoints
- GSD-2 integration verificada

## v1.0 — Initial Setup & Integration (Completed 2026-04-12)

**Goal:** Configurar NewAPI como gateway LLM para ecossistema Atius
**Shipped:**
- NewAPI + PostgreSQL em Docker Compose
- 3 DeepSeek API keys rotativas
- Integração GSD-2 com atius-router provider
- Tokens configurados no auth.json
- deepseek-chat e deepseek-reasoner testados e funcionando

## v1.2 — Fork Migration & Sync Workflow (Completed 2026-04-21)

**Goal:** Migrar repo local para fork formal com workflow de sync upstream
**Shipped:**
- `scripts/sync-fork.sh` — 8-step merge workflow com proteção de arquivos locais
- `scripts/version-bump.sh` — versionamento X.Y.Z.N aware do upstream
- FORK_MIGRATION.md — documentação completa do fork e sync workflow
- `.github/workflows/sync.yml` — weekly upstream sync automatizado
- `.github/workflows/release.yml` — releases via tag semantic
- `agent-harness/` — newapi-cli Click-based CLI com container/channel/model commands

**Blockers Resolved:**
- GitHub CLI authenticated para PR creation em workflows
