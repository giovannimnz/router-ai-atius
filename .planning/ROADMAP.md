# ROADMAP.md - atius-ai-router

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

- Configurar `origin` → `https://github.com/giovannimnz/atius-ai-router.git`
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
- Protection: `if: github.repository == 'giovannimnz/atius-ai-router'`

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

## v1.4 — Planning

Next milestone TBD. Possible directions:
- v1.4: Monitoring & Health Checks (logs centralizados, métricas, alerting)
- v1.4: Additional Providers (Gemini, Claude via Anthropic API)
- v1.4: Rate Limiting & Quota Management (monitoramento de uso por token)
- v1.4: Failover & HA (múltiplas instâncias, load balancing)
