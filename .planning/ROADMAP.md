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

## v2.14 — Codex Go Native SDK ✅ Complete

Goal: Assinatura Codex Pro (plano 100 USD) como provider OpenAI-compatible
dentro do router-ai-atius. 100% Go. Zero Python. Zero sidecar.
Usuário final vê Codex como mais um provider OpenAI — `/v1/chat/completions`
e `/v1/models` funcionam normalmente.

### Phase 05: Codex Go Native SDK (SDK-01/02/03/04)

**Goal:** Implementação 100% Go do adaptor Codex, sem sidecar Python.
Chat Completions → Responses API traduzido nativamente pelo router.

**Scope:**
- Relé HTTP direto para `chatgpt.com/backend-api/codex/responses`
- Chat Completions `/v1/chat/completions` convertido para Responses API
- Chat Completions response convertido de Responses API de volta
- Model list via `GetModelList()` → `/v1/models`
- Streaming suportado (SSE Responses → SSE Chat Completions)
- OAuth flow existente mantido (`controller/codex_oauth.go`)
- Auto-refresh de credentials mantido (`service/codex_credential_refresh_task.go`)
- Channel tipo 57 (Codex) com `Key = {access_token, refresh_token, account_id}`

**Verification:**
- `go build ./...` compila
- Canal Codex tipo 57 com modelo `gpt-5.4` → `POST /v1/chat/completions` retorna 200
- `GET /v1/models` lista modelos Codex
- Streaming funciona (`stream: true`)
- OAuth + auto-refresh continuam funcionando

---

## v2.16 — Codex Device Auth + Real Models + Branding 🔵 Active

Goal: Device Auth JSON upload como fluxo primário, PKCE callback paste como
secundário, modelos reais do upstream Codex, e renomeação para "OpenAI Codex OAuth".

### Phase 10: Device Auth + Real Models + Branding (DEV-01/02/03/04)

**Goal:** Upload de `auth.json` gerado por `codex login --device-auth`.
Modelos carregados do upstream real após autenticação. Renomeado.

**Scope:**
- Device Auth Upload endpoint (`POST /api/channel/codex/oauth/device/upload`)
- PKCE Callback Paste mantido como fallback
- Real model fetching do chatgpt.com/backend-api/codex/responses
- Renomeado "Codex OAuth" → "OpenAI Codex OAuth"
- Frontend: device auth UI com file upload + paste JSON
- Frontend: auto-load models após auth

**Verification:**
- Upload de `auth.json` popula credencial no canal
- Modelos reais do Codex aparecem na lista
- PKCE fallback continua funcional
- `go build ./...` + `bun run typecheck` limpos

---

## Summary

| # | Phase | Requirements | Status |
|---|-------|-------------|--------|
| 05 | Codex Go Native SDK | SDK-01/02/03/04 | ✅ Complete |

**1 phase | 4 requirements mapped | All covered ✓**

---

## Next

N/A — milestone completo.
