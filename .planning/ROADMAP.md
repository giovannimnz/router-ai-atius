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

## v1.5 — API Unification & Model Listing ✓

### Phase 1: Anthropic Channels Setup ✓

**slug:** `router-anthropic-channels`

- Session timeout 12h (MaxAge: 43200)
- Canais type=14 (Anthropic) criados: id=3 (MiniMax), id=4 (MiniMax-Highspeed)
- Relay /v1/messages funcionando (Claude → OpenAI conversion)
- Abilities populadas para M2.1, M2.5, M2.7 nos canais Anthropic
- Nota 2026-06-18: este desenho de canais Anthropic separados foi substituido pela consolidacao Go-native; MiniMax opera como canal unico type=35 e DeepSeek como canal unico type=43.

### Phase 2: /v1/claude/models Endpoint ✓

**slug:** `claude-models-endpoint`

- `GET /v1/claude/models` adicionado
- Fix `IsModelLimitsEnabled()` em middleware/auth.go
- Bruno test collection criada (3/3 passing)

### Phase 3: Model Unification via Middleware [SUPERSEDED]

**slug:** `model-unification`

- Proposta historica: unificar `/v1/models` com `?api_format=openai|anthropic`
- Proposta historica: FastAPI como entry point para listagem (Option A)
- Proposta historica: Go vira pure relay (downstream)
- Proposta historica: deprecar `/v1/claude/models` com headers Sunset
- Proposta historica: internal endpoint `/internal/v1/models` no Go

**Resolution:** Nao pendente. Esta direcao foi substituida pelo cutover Go-native consolidado na `phase-20-go-native-model-router`, onde o Go passou a ser o owner de `GET /v1/models` e a dependencia do middleware Python foi removida do caminho canonico.

**Current canonical state:**

- `GET /v1/models` pertence ao backend Go
- o middleware/FastAPI nao e o owner do catalogo publico
- `model-detailed` pode permanecer apenas para superfícies auxiliares legadas/docs, nao como source of truth de `/v1/models`

---

## v2.12 — pt-native upstream sync [IN PROGRESS]

**Status:** Phase 7 executed locally on `feat/pt-native`; legacy Phase 8 moved to Phase 21 for current sequencing; Phase 21 ready for planning
**Goal:** Re-submeter a tradução PT-BR para o upstream QuantumNous/new-api em um PR limpo, com escopo mínimo (idioma nativo), sem inflar com código do fork Atius.

**Background:** PR #5245 aberto pelo fork giovannimnz/router-ai-atius contra QuantumNous/new-api está poluído: 60 arquivos modificados, 49757 adições, 76 deleções, 15 commits. 95% do PR é fork Atius contaminando (PODMAN, model-detailed, .planning, docker-compose rebrand, login route, vitest, login.tsx, routeTree.gen.ts). Apenas 5 arquivos são escopo legítimo de tradução PT: i18n.go, pt.yaml, pt.json, config.ts, languages.ts.

### Phase 7: feat-pt-native-branch

**Goal:** Criar branch `feat/pt-native` no fork baseado em `upstream/main`, contendo apenas os 5 arquivos nativos de tradução PT (mesmo padrão de zh/en), sem nenhuma contaminação do fork Atius.
**Status:** Executed locally on 2026-06-17; branch `feat/pt-native` ready for commit/push handoff
**Requirements:** TBD
**Depends on:** —
**Plans:** 1 plan (8 tasks)

Plans:

- [x] 01-feat-pt-native-branch-PLAN.md (created via inline plan)

### Phase 8: feat-pt-native-pr [SUPERSEDED]

**Goal:** Superseded numbering placeholder for the clean PT-BR upstream PR handoff. The active/current phase is now Phase 21.
**Status:** Closed as moved to Phase 21 on 2026-06-26; do not plan or execute this phase number.
**Requirements:** TBD
**Depends on:** Phase 7
**Plans:** 0 plans

Plans:

- [x] Moved to Phase 21 (phase number corrected after Phase 20 completion)

### Phase 21: feat-pt-native-pr

**Goal:** Implementar primeiro neste fork o suporte PT-BR 100% nativo conforme o `upstream/main` atual de `QuantumNous/new-api`, sem camada i18n custom e sem arquivo fora do padrão; depois preparar um PR upstream limpo se o resultado local for aprovado.
**Status:** Executed locally on 2026-07-05; ready for commit/push handoff
**Milestone:** v2.12 — pt-native upstream sync
**Requirements:** PHASE-21-UPSTREAM-NATIVE-I18N, PHASE-21-REUSE-EXISTING-TRANSLATIONS, PHASE-21-PT-BR-COVERAGE, PHASE-21-LOCAL-FIRST-VALIDATION, PHASE-21-UPSTREAM-PR-HYGIENE
**Depends on:** Phase 20
**Plans:** 5 plans (4 waves)

Plans:

**Wave 1**

- [x] 21-01-PLAN.md - clean upstream lane and translation inventory

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 21-02-PLAN.md - backend native PT-BR locale and go-i18n validation
- [x] 21-03-PLAN.md - default frontend native PT-BR locale, wiring, and sync validation

**Wave 3** *(blocked on Wave 2 default frontend completion)*

- [x] 21-04-PLAN.md - classic frontend native PT-BR locale, wiring, and build validation

**Wave 4** *(blocked on all implementation validation)*

- [x] 21-05-PLAN.md - upstream-ready single commit, leak checks, duplicate search, and PR handoff

### Phase 22: K3s migration preflight and cutover plan for router-ai-atius

**Goal:** Preparar e validar a migração do runtime `router-ai-atius` de Podman rootless para k3s sem perder o contrato full-Go, sem reintroduzir `model-detailed`, com backup/rollback testado e cutover bloqueado por aprovação manual.
**Status:** Complete (2026-07-09; artifacts ready, public cutover deferred)
**Milestone:** v2.15 — k3s transition and deferred runtime validation
**Requirements:** PHASE-22-K3S-PREFLIGHT, PHASE-22-RUNTIME-PARITY, PHASE-22-STATEFUL-DATA, PHASE-22-CUTOVER-ROLLBACK
**Depends on:** Phase 20 runtime full-Go. Independe da Phase 21, mas deve evitar contaminar o worktree/branch `feat/pt-native`.
**Plans:** 4 plans (4 waves)

Plans:
**Wave 1**

- [x] 22-01-PLAN.md - k3s cluster/runtime preflight and migration contract

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 22-02-PLAN.md - Kubernetes manifests, secret templates, and dry-run validation

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 22-03-PLAN.md - backup, restore rehearsal, and shadow deployment validation

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 22-04-PLAN.md - production cutover, rollback, and operator handoff

### Phase 23: long-context-alias-validation

**Goal:** Validar os aliases experimentais `gpt-5.5-1m` e `gpt-5.4-1m` com testes progressivos de raciocinio/contexto ate aproximadamente 1M tokens, preservando seguranca operacional, custos explicitos, rastreabilidade de alias/upstream e pricing long-context.
**Status:** Complete (2026-07-09; local/static harness validated, live expensive runs deferred by design)
**Milestone:** v2.15 — k3s transition and deferred runtime validation
**Requirements:** PHASE-23-LONG-CONTEXT-CATALOG, PHASE-23-LONG-CONTEXT-STREAMING, PHASE-23-LONG-CONTEXT-REASONING, PHASE-23-LONG-CONTEXT-BILLING-TRACE
**Depends on:** Phase 20 runtime full-Go and deployed alias pricing fix. Independe da Phase 21 e nao altera o plano de migracao da Phase 22.
**Plans:** 1 plan

Plans:

- [x] 23-01-PLAN.md - progressive 1M long-context alias validation harness and UAT

### Phase 24: router-db-catalog-recovery-and-canonical-host-db

**Goal:** Recuperar totalmente o runtime/catalogo do `router-ai-atius` no banco canonico do host via PgBouncer, restaurando `OpenAI - Codex`, DeepSeek e o caminho Go-native de embeddings, eliminando o drift entre `newapi` e o nome correto do banco, e reaplicando a consolidacao de canais/provedores sem reintroduzir aliases `-1m` nem embeddings Codex desabilitados.
**Status:** Complete (2026-07-04)
**Requirements:** PHASE-24-CANONICAL-HOST-DB, PHASE-24-CATALOG-RESTORE, PHASE-24-PROVIDER-CONSOLIDATION, PHASE-24-EMBEDDING-GOVERNOR-PRESERVE, PHASE-24-CUTOVER-ROLLBACK
**Depends on:** Phase 20 runtime full-Go; uses the local evidence and backups collected on 2026-07-03/04. Supersedes the partial embedding-only repair as the next recovery step. Independent of Phases 21, 22, and 23.
**Plans:** 4 plans

Plans:

- [x] 24-01-PLAN.md - canonical host DB inventory, freeze, backup, and restore source selection
- [x] 24-02-PLAN.md - canonical DB rename/migration and full router catalog restore
- [x] 24-03-PLAN.md - provider/channel consolidation, OpenAI Codex/GPT recovery, and embedding governor preservation
- [x] 24-04-PLAN.md - runtime cutover, docs reconciliation, and end-to-end validation

### Phase 25: embedding-governor-auto-workload-inference

**Goal:** Tornar `embedding-gte-v1` sempre governado no router e inferir automaticamente `batch` versus `interactive` quando o cliente nao enviar `X-Embedding-Workload`, preservando o header como override operacional, sem criar alias publico `*-batch` e mantendo o limite seguro do TEI.
**Status:** Complete (2026-07-05)
**Requirements:** PHASE-25-GOVERNED-MODEL-SCOPE, PHASE-25-AUTO-WORKLOAD-INFERENCE, PHASE-25-HEADER-OVERRIDE-COMPATIBILITY, PHASE-25-TEI-BATCH-SAFETY, PHASE-25-CLIENT-SMOKE-VALIDATION
**Depends on:** Phase 20 Go-native embedding governor and Phase 24 final runtime/catalog state. Uses Codex session `019f2dc6-858a-79e1-a78d-495ee5631235` as design input.
**Plans:** 3/3 plans complete

Plans:

**Wave 1**

- [x] 25-01-PLAN.md - governor classifier, model scope, defaults, and service tests

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 25-02-PLAN.md - relay no-header metadata capture and fail-closed TEI input cap

**Wave 3** *(blocked on Waves 1-2 completion)*

- [x] 25-03-PLAN.md - smoke defaults, operator docs, and conditional live validation

### Phase 26: codex-dynamic-discovery-and-curated-catalog

**Goal:** Implementar descoberta dinâmica e account-aware dos modelos Codex, com cache local, enriquecimento multi-fonte, validação automática por request mínima e promoção segura para o catálogo curado que alimenta `/v1/models`.
**Status:** Complete (2026-07-08)
**Requirements:** PHASE-26-LOCAL-CURATED-V1-MODELS, PHASE-26-DYNAMIC-CODEX-DISCOVERY, PHASE-26-MULTI-SOURCE-ENRICHMENT, PHASE-26-CANDIDATE-PROBE-GATE, PHASE-26-CODEX-METADATA-ENRICHMENT, PHASE-26-DAILY-SCHEDULED-SYNC, PHASE-26-DEFAULT-MODEL-GUARD
**Depends on:** Phase 24 final runtime/catalog state and current Codex OAuth/Responses path
**Plans:** 1 plan

Plans:

**Wave 1**

- [x] 26-01-PLAN.md - dynamic discovery, multi-source enrichment, validation gate, and curated promotion

### Phase 27: codex-official-docs-ci-and-release-alignment

**Goal:** Alinhar CI/auth/release docs e automações do fork com a documentação oficial OpenAI/Codex, mantendo PT-BR como idioma padrão das saídas operacionais, release notes e changelog do fork.
**Status:** Complete (2026-07-08)
**Requirements:** PHASE-27-OFFICIAL-DOCS-FIRST, PHASE-27-CODEX-CI-AUTH, PHASE-27-PTBR-RELEASE-OPS-DOCS
**Depends on:** Phase 26 curated catalog contract
**Plans:** 1 plan

Plans:

**Wave 1**

- [x] 27-01-PLAN.md - official docs baseline, workflow alignment, and PT-BR Codex CI/auth/release runbook

### Phase 28: branch-hygiene-and-mainline-reconciliation

**Goal:** Consolidar definitivamente o estado local/remoto do fork: fazer backup seguro dos worktrees, promover a lane limpa da Phase 21 como branch remota canônica, reconciliar seletivamente o que deve ir para `origin/main`, e então aposentar branches/worktrees stale sem perder a opção de handoff upstream PT-BR.
**Status:** Complete (2026-07-08)
**Milestone:** v2.14 — branch hygiene and mainline reconciliation
**Requirements:** PHASE-28-SAFETY-BACKUP, PHASE-28-PHASE21-CANONICAL-REMOTE, PHASE-28-MAINLINE-RECONCILIATION, PHASE-28-LOCAL-HYGIENE, PHASE-28-REMOTE-HYGIENE, PHASE-28-BRANCH-POLICY
**Depends on:** Phase 21 local execution artifacts, Phase 24/25/26/27 fork state, and current `origin/main` / `upstream/main` divergence audit.
**Plans:** 4 plans (4 waves)

Plans:

**Wave 1**

- [x] 28-01-PLAN.md - worktree inventory, git safety backup, and freeze policy

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 28-02-PLAN.md - Phase 21 canonical remote promotion and clean-lane validation

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 28-03-PLAN.md - selective mainline reconciliation branch and merge into `main`

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 28-04-PLAN.md - local/remote cleanup, worktree reset, and final hygiene verification

---

## v1.6 — Future

- Monitoring & Health Checks (logs centralizados, métricas, alerting)
- Additional Providers (Gemini, Claude via Anthropic API)
- Rate Limiting & Quota Management
- Failover & HA (múltiplas instâncias, load balancing)
