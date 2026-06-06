# REQUIREMENTS.md — Atius AI Router

**Milestone:** v2.14 — Codex SDK Transformer
**Last updated:** 2026-06-06

---

## Active Requirements — v2.14

### Transformer Module

- [x] **SDK-01**: Sidecar Python com HTTP bridge para Codex SDK. Expõe endpoints
  `/v1/codex/run` (one-shot) e `/v1/codex/thread` (stateful com thread_id).
  Traduz requests do router Go → chamadas `thread.run()` no `openai-codex` SDK.
  Suporta modelos: `gpt-5.4`, `gpt-5-codex`, `gpt-5.1-codex`, `gpt-5.2-codex`,
  `gpt-5.3-codex`, `gpt-5.3-codex-spark`. Sidecar roda como microserviço
  gerenciado por PM2 ou systemd, independente do ciclo de vida do container Go.

- [ ] **SDK-02**: Login explícito + armazenamento próprio. Admin cola authorization
  code do OAuth flow no campo do dashboard OU importa JSON manual com
  `{access_token, refresh_token, account_id}`. Resultado armazenado em
  `data/codex/license.json`. Token refresh automático via goroutine
  `codex_credential_refresh_task.go` depois de autenticado. Nunca reutiliza
  credenciais do host (`~/.codex/auth.json`) silenciosamente. Comportamento
  equivalente a `hermes login` → pede código → armazena → usa.

### Usage & Visibility

- [ ] **SDK-03**: Dashboard de usage/saldo no admin frontend. Gráfico de consumo
  do mês, dados parseados do endpoint `/backend-api/wham/usage`. Mostra: total
  de requests, tokens consumidos, custo estimado em USD, dias restantes no
  ciclo de billing. Endpoint REST: `GET /api/channel/:id/codex/usage` (já
  existe) com resposta parseada em JSON limpo.

### Safety

- [ ] **SDK-04**: Channel coexistence sem breaking. Campo `backend` no canal
  tipo 57 com valores `relay` (padrão, HTTP `/backend-api/codex/responses`
  atual) ou `sdk` (roteia para o sidecar Python). Ambos coexistem no mesmo
  tipo de canal. Admin escolhe backend via dropdown no formulário de canal.
  Canais existentes mantêm comportamento `relay` inalterado.

---

## Future Requirements (deferred)

- TypeScript SDK runtime (v2.14 é Python-first)
- Multi-key / multi-account Codex
- Faturamento próprio do router via Codex
- Outros provedores SDK (Claude, Gemini, etc.)

---

## Out of Scope

- Outros provedores SDK — Codex only
- TypeScript SDK runtime — Python-first
- Multi-key / multi-account Codex
- Faturamento próprio do router via Codex — é só relay/transformer

---

## Traceability

| REQ-ID | Phase | ROADMAP |
|--------|-------|---------|
| SDK-01 | 01 | Sidecar Python + HTTP bridge |
| SDK-02 | 02 | Login explícito + storage licença |
| SDK-03 | 03 | Dashboard usage/saldo |
| SDK-04 | 04 | Channel coexistence |

---

## Last updated

2026-06-06 — Milestone v2.14 requirements defined (4 REQs)
