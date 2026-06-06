# Atius AI Router — Roadmap

## v2.12 — pt Native i18n Integration ✅ Complete

Goal: Integrate Portuguese locale into the entire stack — backend Go (new-api), frontend React/i18next, AND Fumadocs docs site. Zero custom code, only registration points.

### Phase 01: pt Locale Registration ✅ (2026-06-05)
### Phase 02: pt Fumadocs i18n ✅ (2026-06-05)
### Phase 03: PT Docs Bugfixes ✅ (2026-06-05)
### Phase 04: Prod Docs Bugfixes ✅ (2026-06-06)

---

## Architecture Note

The router-ai-atius stack has **3 i18n systems** — all now support `pt`:

| App | Framework | i18n mechanism | PT Status |
|---|---|---|---|
| Backend (new-api) | Go | `go-i18n` with YAML | ✅ 228 keys |
| Frontend (new-api SPA) | React | i18next + language detector | ✅ 4521 keys |
| Docs (Fumadocs) | Next.js | URL prefix + MDX per locale | ✅ 294 files |

All follow native pattern — only registration points, zero custom code.

---

## v2.14 — Codex SDK Transformer 🔵 Active

Goal: Usar assinatura Codex Pro (plano 100 USD) como módulo transformer
dentro do router-ai-atius, expondo o Codex SDK programaticamente com
visibilidade de saldo/usage. Login explícito estilo Hermes. Zero breaking
change no canal tipo 57 existente.

### Phase 05: Sidecar Python + HTTP Bridge (SDK-01)

**Goal:** Microserviço Python com `openai-codex` SDK exposto via HTTP local.
Router Go proxyia requests Codex `backend=sdk` para o sidecar.

**Requirements mapped:** SDK-01

**Scope:**
- Criar diretório `integration/codex-sidecar/` com app FastAPI
- `pip install openai-codex` no ambiente do sidecar
- Endpoints:
  - `POST /v1/codex/run` — one-shot prompt, retorna `TurnResult`
  - `POST /v1/codex/thread` — stateful com `thread_id`
  - `GET /health` — health check
- Dockerfile para o sidecar (imagem separada ou mesmo compose)
- docker-compose.yml: novo serviço `codex-sidecar` na rede `newapi-internal`
- Go: novo handler em `service/` que proxyia para o sidecar quando
  `backend=sdk`

**Verification:**
- `curl -X POST localhost:1456/v1/codex/run -d '{"model":"gpt-5.4","prompt":"hello"}'` retorna 200 com resposta
- `podman ps` mostra container `codex-sidecar` healthy
- Canal Codex tipo 57 com `backend=sdk` faz relay via sidecar

### Phase 06: Login Explícito + Armazenamento Licença (SDK-02)

**Goal:** Fluxo de login onde admin cola authorization code ou importa JSON.
Credenciais armazenadas em `data/codex/license.json`. Refresh automático.

**Requirements mapped:** SDK-02

**Scope:**
- Página no admin dashboard: `/admin/codex-auth`
  - Campo "Authorization Code": colar URL do OAuth redirect (já existe flow)
  - Botão "Importar JSON": upload de `{access_token, refresh_token, account_id}`
  - Status: mostra email, expired, last_refresh
- Go handler: `POST /api/codex/auth` — valida e persiste em `data/codex/license.json`
- Reuso da goroutine `codex_credential_refresh_task.go` para refresh
- Sidecar Python lê `data/codex/license.json` para autenticar chamadas SDK
- **NUNCA** faz fallback silencioso para `~/.codex/auth.json`

**Verification:**
- Colar authorization code → status mostra email
- Token expira → refresh automático
- `cat data/codex/license.json` mostra `{access_token, refresh_token, account_id, email, expired, last_refresh}`
- Remover `~/.codex/auth.json` → sidecar continua funcionando

### Phase 07: Dashboard Usage/Saldo (SDK-03)

**Goal:** Gráfico de consumo do mês no admin frontend, dados do `wham/usage`.

**Requirements mapped:** SDK-03

**Scope:**
- Frontend: componente React `CodexUsagePanel` no admin
  - Gráfico de barras: consumo diário (tokens)
  - Cards: total requests, tokens, custo estimado (USD), dias restantes
- Backend: endpoint `GET /api/channel/:id/codex/usage/parsed`
  - Parseia resposta JSON do `wham/usage` upstream
  - Retorna: `{daily_usage: [{date, tokens, requests}], total_tokens, total_requests, estimated_cost, cycle_days_left}`
- Reuso do handler existente `controller/codex_usage.go`

**Verification:**
- Admin abre `/admin/codex-usage` → gráfico renderiza
- Dados batem com `curl chatgpt.com/backend-api/wham/usage` direto
- Canal sem licença → mostra "Configure licença primeiro"

### Phase 08: Channel Coexistence + Validação (SDK-04)

**Goal:** Flag `backend=sdk|relay` no canal tipo 57. Ambos coexistem.
Validação end-to-end de todos os fluxos.

**Requirements mapped:** SDK-04

**Scope:**
- Model: campo `CodexBackend` no `Channel.Other` ou `Channel.Setting`
- Admin UI: dropdown "Backend" no formulário de canal Codex (relay/sdk)
- Router: `GetRequestURL()` no adaptor codex decide rota baseado no backend
- Teste end-to-end:
  - Canal `relay` → funciona como antes (HTTP `/backend-api/codex/responses`)
  - Canal `sdk` → proxyia para sidecar Python
  - Ambos simultâneos → sem interferência

**Verification:**
- Criar canal A (relay) + canal B (sdk) → ambos respondem
- Logs mostram rotas diferentes (A → chatgpt.com, B → localhost:1456)
- Nenhum canal existente quebrou

---

## Summary

| # | Phase | Requirements | Status |
|---|-------|-------------|--------|
| 05 | Sidecar Python + HTTP Bridge | SDK-01 | ⏳ Not Started |
| 06 | Login Explícito + Licença | SDK-02 | ⏳ Not Started |
| 07 | Dashboard Usage/Saldo | SDK-03 | ⏳ Not Started |
| 08 | Channel Coexistence + Validação | SDK-04 | ⏳ Not Started |

**4 phases | 4 requirements mapped | All covered ✓**

---

## Next

- [ ] `/gsd-discuss-phase 05` — gather context for Phase 05 (Sidecar Python)
- [ ] Run Phase 05: Sidecar Python + HTTP Bridge
- [ ] Run Phase 06: Login Explícito + Licença
- [ ] Run Phase 07: Dashboard Usage/Saldo
- [ ] Run Phase 08: Channel Coexistence + Validação
