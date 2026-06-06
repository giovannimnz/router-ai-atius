---
phase: 04
wave: 1
depends_on: [03]
files_modified:
  - .planning/phases/04-prod-docs-bugfixes/04-CONTEXT.md
  - .planning/phases/04-prod-docs-bugfixes/04-PLAN.md
  - .planning/phases/04-prod-docs-bugfixes/04-SUMMARY.md
  - .planning/STATE.md
  - .planning/ROADMAP.md
  - integration/middleware/Dockerfile.fastapi
  - integration/middleware/model_detailed_fastapi.py
  - integration/middleware/decode-cookie.go
  - integration/middleware/docs/openapi.json
  - integration/middleware/static/index.html
  - integration/middleware/static/logo.svg
  - integration/middleware/scalar/*
  - 21.03-Decisoes-Arquitetura/2026-06-06-apache-proxy-nextjs-docs-licoes-phase-04.md
autonomous: true
must_haves:
  - phase 04 has CONTEXT + PLAN + SUMMARY in place
  - working tree is classified before advancing to phase 05
  - model-detailed remains healthy after middleware restore
  - Cloudflare path tests still return 200 for /v1/messages, /v1/chat/completions, /v1/models
  - docs routing still returns 200 for /pt/docs/ and /_next/* assets
  - a final keep/drop/doc-only decision exists for all current uncommitted changes
  - no advance to phase 05 until all verification checks are green
---

# Phase 04: Prod Docs Bugfixes — PLAN.md

**Phase:** 04
**Status:** Recovery / closure plan
**Date:** 2026-06-06
**Goal:** Normalizar a Phase 04 no substrate GSD, reconciliar o hardening operacional feito hoje com o repo, testar tudo end-to-end, e só então liberar a passagem para a Phase 05.

## Scope

Esta fase não implementa feature nova.
Ela fecha 3 lacunas:
1. substrate GSD incompleto (`04-PLAN.md` ausente)
2. working tree misturando restauração de middleware + docs + notas
3. necessidade de regressão completa antes de abrir a `05 — Cloudflare Cache Purge Automation`

## Out of scope

- iniciar a Phase 05
- novo refactor do new-api relay
- alterar política de roteamento além do estado já validado
- push/merge

---

## Plan 01: Inventário e classificação do working tree

### Objective
Separar exatamente o que é código a manter, o que é documentação, e o que pode ser descartado antes de qualquer commit.

### Files
- Read: `git status --short`
- Read: `git diff --stat`
- Classify: `integration/middleware/**`, `integration/docs/**`, `21.03-Decisoes-Arquitetura/**`, `.planning/**`

### Step 1: Snapshot do estado atual
Run:
```bash
cd /home/ubuntu/docker/Atius/router-ai-atius
git status --short
git diff --stat
```
Expected:
- lista fechada de paths alterados
- nenhuma mudança invisível fora desses conjuntos

### Step 2: Classificar cada grupo
Tabela obrigatória no relatório interno:
- `keep` — entra no commit da Phase 04
- `doc-only` — fica no repo/vault mas pode ir em commit separado
- `drop` — não entra e deve ser restaurado/removido

### Step 3: Gate
Não avançar enquanto todos os paths não estiverem classificados.

---

## Plan 02: Reconciliar `integration/middleware/` com o estado validado em produção

### Objective
Manter no repo exatamente o que foi necessário para subir o `model-detailed` saudável e servir `/scalar/`, `/v1/models`, `/v1/messages` e `/v1/chat/completions` via Cloudflare.

### Files
- Verify: `integration/middleware/Dockerfile.fastapi`
- Verify: `integration/middleware/model_detailed_fastapi.py`
- Verify: `integration/middleware/decode-cookie.go`
- Verify: `integration/middleware/scalar/*`
- Verify: `integration/middleware/static/*`
- Verify: `integration/middleware/docs/openapi.json`

### Step 1: Confirmar presença dos artefatos obrigatórios
Run:
```bash
cd /home/ubuntu/docker/Atius/router-ai-atius
find integration/middleware -maxdepth 2 -type f | sort
```
Expected:
- Dockerfile.fastapi presente
- scalar bundle presente
- openapi.json presente
- static files presentes

### Step 2: Confirmar que o container ativo usa o estado compatível
Run:
```bash
podman exec router-ai-atius-model-detailed env | grep -E 'NEW_API_INTERNAL_URL|NEWAPI_BACKEND_URL|MIDDLEWARE_PORT|DOCS_'
podman logs --tail 20 router-ai-atius-model-detailed
```
Expected:
- `NEW_API_INTERNAL_URL=http://10.88.2.35:3000`
- startup limpo
- sem erro `Directory '/app/scalar' does not exist`

### Step 3: Gate
Se middleware local divergir do container validado, corrigir antes de seguir.

---

## Plan 03: Regressão funcional do router via Cloudflare

### Objective
Provar que o hardening de hoje não quebrou API nem docs antes de avançar para a próxima fase.

### Files
- External verification only
- Reference docs: `21.03-Decisoes-Arquitetura/2026-06-06-apache-proxy-nextjs-docs-licoes-phase-04.md`
- Reference note: `~/GitHub/obsidian-vault/ideaverse/atius-router/21-ATIUS-ROUTER-FULL-CONFIG-2026-06-06.md`

### Step 1: API smoke — Anthropic
Run:
```bash
curl -s -X POST "https://router.atius.com.br/v1/messages" \
  -H "x-api-key: $ATIUS_ROUTER_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{"model":"MiniMax-M3","max_tokens":30,"messages":[{"role":"user","content":"ping"}]}'
```
Expected:
- HTTP 200
- payload `type: "message"`

### Step 2: API smoke — OpenAI compat
Run:
```bash
curl -s -X POST "https://router.atius.com.br/v1/chat/completions" \
  -H "Authorization: Bearer $ATIUS_ROUTER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"MiniMax-M3","messages":[{"role":"user","content":"ping"}],"max_tokens":30}'
```
Expected:
- HTTP 200
- payload `object: "chat.completion"`

### Step 3: Model list
Run:
```bash
curl -s "https://router.atius.com.br/v1/models" \
  -H "Authorization: Bearer $ATIUS_ROUTER_API_KEY"
```
Expected:
- HTTP 200
- `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `deepseek-v4-flash`, `deepseek-v4-pro`

### Step 4: Docs routing
Run:
```bash
curl -Iks https://router.atius.com.br/pt/docs/ | head
curl -Iks https://router.atius.com.br/_next/static/ | head
curl -Iks https://router.atius.com.br/assets/atius-logo.svg | head
```
Expected:
- `/pt/docs/` → Next.js headers, not Go SPA
- `/_next/` no longer falls to catch-all
- logo asset 200

### Step 5: Gate
Se qualquer endpoint falhar, corrigir antes de Phase 05.

---

## Plan 04: Fechamento GSD da Phase 04

### Objective
Deixar a Phase 04 consistente no substrate GSD, para o SDK parar de tratá-la como parcial.

### Files
- Modify: `.planning/phases/04-prod-docs-bugfixes/04-PLAN.md`
- Verify: `.planning/phases/04-prod-docs-bugfixes/04-CONTEXT.md`
- Verify: `.planning/phases/04-prod-docs-bugfixes/04-SUMMARY.md`
- Modify if needed: `.planning/STATE.md`
- Modify if needed: `.planning/ROADMAP.md`

### Step 1: Ensure triad exists
Checklist:
- [x] `04-CONTEXT.md`
- [x] `04-PLAN.md`
- [x] `04-SUMMARY.md`

### Step 2: Cross-check summary vs roadmap/state
Verify:
- `STATE.md` marks Phase 04 complete
- `ROADMAP.md` marks Phase 04 complete
- Summary verification items match reality

### Step 3: Gate
Não abrir Phase 05 enquanto a triad da Phase 04 não estiver íntegra.

---

## Plan 05: Commit gate before advancing

### Objective
Produzir um commit atómico da Phase 04/hardening somente depois de classificação + regressão + substrate íntegro.

### Files
- Exact paths from Plan 01 keep-list only

### Step 1: Stage only keep-list
Run pattern:
```bash
git add <exact keep paths>
git diff --cached --stat
```
Expected:
- sem ruído de paths descartados
- sem arquivos fora do escopo Phase 04/hardening

### Step 2: Commit
Commit message target:
```bash
git commit -m "fix(phase-04): reconcile prod docs hardening and middleware restore"
```

### Step 3: Post-commit verification
Run:
```bash
git status --short
```
Expected:
- working tree com apenas o que explicitamente ficou fora do commit
- Phase 05 ainda não iniciada

---

## Final verification checklist

- [x] `04-PLAN.md` presente
- [x] working tree classificado (`keep/drop/doc-only`)
- [x] middleware restaurado confere com o container validado
- [x] `/v1/messages` 200 via Cloudflare (DeepSeek Anthropic path validado)
- [x] `/v1/chat/completions` 200 via Cloudflare (DeepSeek OpenAI-compat validado)
- [x] `/v1/models` 200 via Cloudflare
- [x] `/pt/docs/` e `/_next/` saudáveis no origin e via CF
- [x] `STATE.md` e `ROADMAP.md` coerentes com a realidade
- [ ] `MiniMax-M3` ainda 200 via Cloudflare
- [x] logo via Cloudflare sem 404 cacheado
- [ ] commit atómico da Phase 04/hardening pronto
- [ ] só então liberar a abertura da Phase 05

## Findings from closure verification (2026-06-06)

### Working tree classification

**keep**
- `integration/middleware/**`
- `.planning/phases/04-prod-docs-bugfixes/04-CONTEXT.md`
- `.planning/phases/04-prod-docs-bugfixes/04-PLAN.md`
- `21.03-Decisoes-Arquitetura/2026-06-06-apache-proxy-nextjs-docs-licoes-phase-04.md`

**drop**
- `integration/docs/.source/index.ts`
- `integration/docs/.source/source.config.mjs`

**doc-only / out of scope now**
- `21.03-Decisoes-Arquitetura/2026-06-05-i18n-pt-nativo-infraestrutura-upstream.md`

### Production blockers before Phase 05

1. **MiniMax-M3 validation blocked by upstream weekly quota**
   - `/v1/chat/completions` with model `MiniMax-M3` → HTTP 429
   - `/v1/messages` with model `MiniMax-M3` → HTTP 429
   - router log: `weekly usage limit reached for Token Plan Max (300000000/300000000 used)`
   - `deepseek-v4-flash` still validates 200 in both formats, so the router path is healthy; the blocker is provider quota, not router breakage.

2. **Cloudflare stale 404 for logo asset — RESOLVED**
   - purge via Cloudflare API executed successfully against zone `atius.com.br`
   - `https://router.atius.com.br/assets/atius-logo.svg` now → HTTP 200 (`cf-cache-status: MISS` right after purge)
   - origin bypass had already been 200; CDN is now aligned with origin

3. **Local middleware source differs from active container**
   - `integration/middleware/model_detailed_fastapi.py` local contains broader CJK filter + helper comments + docs subpath redirect not currently running in container
   - `integration/middleware/static/index.html` was stale (`atius2024`) and was corrected back to `Bkfigt!546`
   - this means a future rebuild from repo would change runtime behavior; commit should happen consciously, not accidentally.

### Gate conclusion

**Do not advance to Phase 05 yet.**
The substrate is now clearer, but the final closure gate still has 1 external/runtime blocker:
- MiniMax quota exhaustion

Phase 05 can start once we decide one of two explicit policies:
- accept DeepSeek-based regression as sufficient while MiniMax waits for quota reset, or
- provision/rotate a fresh MiniMax credential/channel before opening the phase.
