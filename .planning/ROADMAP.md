# ROADMAP.md - atius-ai-router

## v1.4 — Model Aliases & Token Management ✓
**Status:** COMPLETE

### Phase 1: Model Aliases Setup ✓
**slug:** `model-aliases-hs`

- Adicionados `MiniMax-M2.7-hs` e `MiniMax-M2.5-hs` na tabela `models`
- Configurado `model_mapping` no canal 1: `MiniMax-M2.7-hs` → `MiniMax-M2.7-highspeed` (e M2.5)
- Adicionados ao `ModelRatio` (0.15), `CompletionRatio` (4.0), `InputPrice` (0.3), `OutputPrice` (1.2)
- Abilities criadas para ambos os aliases no canal MiniMax
- Router reiniciado e validado: GET /v1/models lista os aliases, POST funciona

### Phase 2: API Key Giovanni-Acc ✓
**slug:** `api-key-giovanni-acc`

- Token criado: `9cfec16339f2306085cc45124b1b62e691f621fe82bbdc92`
- Nome: Giovanni-Acc, user_id: giovanni (id=2)
- Quota ilimitada, sem expiracao
- Testado com GET /v1/models e POST MiniMax-M2.7-hs — OK

---

## v1.3 — Testing Infrastructure & CLI ✓
**Status:** COMPLETE

### Phase 1: Bruno CLI Setup ✓
**slug:** `bruno-cli-install`

- Bruno CLI instalado em `/home/ubuntu/.nvm/versions/node/v24.13.1/bin/bru`
- Versão 3.2.2
- Verificado funcional

### Phase 2: Collection Creation ✓
**slug:** `bruno-collection`

- Criado `integration/bruno-tests/atius-router-tests/`
- 5 requests cobrindo todos os modelos
- Environment `.env.local` com variáveis em camelCase

### Phase 3: Test Suite ✓
**slug:** `bruno-tests`

- `list-models.bru` — GET /v1/models
- `deepseek-chat.bru` — POST /v1/chat/completions
- `deepseek-reasoner.bru` — POST /v1/chat/completions
- `minimax-m27.bru` — POST /v1/chat/completions
- `minimax-m25.bru` — POST /v1/chat/completions

**Tests Passing:** 5/5 ✅

### Phase 4: Wrapper Script ✓
**slug:** `bruno-runner-script`

- `scripts/run-bruno-tests.sh` criado
- Lê variáveis de `.env.local`
- Usa delay 500ms para evitar rate limiting

### Phase 5: Skill Creation ✓
**slug:** `bruno-skill`

- Skill em `~/.agents/skills/bruno-cli/SKILL.md`
- Documentação completa de uso
- Troubleshooting section

---

## v1.2 — Fork Migration & Sync Workflow ✓
**Completed:** 2026-04-21

### Phase 1: Git Setup & Remotes ✓
**slug:** `fork-git-setup`

- Configurar `origin` → `https://github.com/giovannimnz/router-ai-atius.git`
- Configurar `upstream` → `https://github.com/QuantumNous/new-api.git`
- Testar `git fetch upstream` e `git fetch origin`
- Verificar compatibilidade de história git
- Primeiro push para origin (force-push)

### Phase 2: Fork Sync Script (`sync-fork.sh`) ✓
**slug:** `fork-sync-script`

- Criar `scripts/sync-fork.sh` em bash puro
- Implementar fetch upstream + merge com `--strategy`
- Implementar proteção de arquivos locais
- Testar com dry-run

### Phase 3: Version Bump Script (`version-bump.sh`) ✓
**slug:** `fork-version-bump`

- Criar `scripts/version-bump.sh`
- Parsear versão atual de `VERSION` file
- Comparar com upstream latest tag
- Aplicar lógica: base changed → suffix=1, same → suffix++
- Criar git tag `vX.Y.Z.N`

### Phase 4: FORK.md Documentation ✓
**slug:** `fork-fork-md`

- Documentar parent repo e fork purpose
- Listar todas modificações locais com rationale
- Documentar sync workflow e comandos
- Criar troubleshooting section

### Phase 5: GitHub Actions CI/CD ✓
**slug:** `fork-github-actions`

- Criar `.github/workflows/sync.yml` (scheduled sync)
- Criar `.github/workflows/release.yml` (tag-based releases)
- Protection: `if: github.repository == 'giovannimnz/router-ai-atius'`

### Phase 6: CLI-Anything: NewAPI Management ✓
**slug:** `cli-anything-newapi`

- Criar `agent-harness/` com CLI Click para NewAPI
- Comandos: container, channel, model, api
- `--json` output para consumo por agentes
- SKILL.md para auto-descoberta por agentes

---

## v1.1 — DeepSeek Model Metadata Enrichment ✓
**Completed:** 2026-04-14

### Phase 1: Investigate NewAPI Customization ✓
**slug:** `investigate-newapi-customization`

### Phase 2: Configure DeepSeek Metadata DB ✓
**slug:** `configure-deepseek-metadata-db`

### Phase 3: Implement Enrichment Middleware ✓
**slug:** `implement-enrichment-middleware`

### Phase 4: Validate Enriched Endpoint ✓
**slug:** `validate-enriched-endpoint-consumers`

---

## v1.0 — Initial Setup ✓
**Completed:** 2026-04-12

- Docker Compose com NewAPI + PostgreSQL
- Middleware proxy Python inicial
- Integração com ecossistema Atius (redes, volumes)

---

## v1.5 — API Unification & Model Listing ✓

### Phase 1: Anthropic Channels Setup ✓
**slug:** `router-anthropic-channels`

- Session timeout 12h (MaxAge: 43200)
- Canais type=14 (Anthropic) criados: id=3 (MiniMax), id=4 (MiniMax-Highspeed)
- Relay /v1/messages funcionando (Claude → OpenAI conversion)
- Abilities populadas para M2.1, M2.5, M2.7 nos canais Anthropic

### Phase 2: /v1/claude/models Endpoint ✓
**slug:** `claude-models-endpoint`

- `GET /v1/claude/models` adicionado
- Fix `IsModelLimitsEnabled()` em middleware/auth.go
- Bruno test collection criada (3/3 passing)

### Phase 3: Model Unification via Middleware [PLANNING]
**slug:** `model-unification`

- Unificar `/v1/models` com `?api_format=openai|anthropic`
- FastAPI como entry point para listagem (Option A)
- Go vira pure relay (downstream)
- Deprecar `/v1/claude/models` com headers Sunset
- Internal endpoint `/internal/v1/models` no Go

**Blocked by:** Phase 2 completion

---

## v1.6 — Future

- Monitoring & Health Checks (logs centralizados, métricas, alerting)
- Additional Providers (Gemini, Claude via Anthropic API)
- Rate Limiting & Quota Management
- Failover & HA (múltiplas instâncias, load balancing)
