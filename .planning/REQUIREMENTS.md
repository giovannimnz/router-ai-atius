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

- [ ] **SDK-04**: Channel coexistence sem breaking. Canal tipo 57 com
  flag `backend` em `relay` (padrão, HTTP `/backend-api/codex/responses`
  atual). 100% Go nativo, sem sidecar Python. Admin configura via
  formulário de canal. Canais existentes mantêm comportamento inalterado.

---

## Planned Requirements — v2.15 Docs Convergence

- [ ] **DOCS-01**: A documentação deixa de existir como repo de runtime separado
  em `/home/ubuntu/docker/Atius/atius-router-docs` e passa a viver dentro do
  repo principal `router-ai-atius` em `docs/atius-router-docs/` como submodule
  canônico. A fase deve cobrir a mecânica de checkout/update, impacto no fluxo
  de upstream sync e como o `router-ai-atius` referencia esse submodule no dia a dia.

- [ ] **DOCS-02**: O cutover preserva branding Atius, logo SVG válida, PT-BR,
  e rotas críticas (`/pt/`, `/pt/docs/`, `/pt/docs/skills/`, `/en/`). Runtime,
  Apache, build e deploy deixam de depender do repo standalone anterior.

- [ ] **DOCS-03**: `~/GitHub/omni-srv-admin` passa a ser a fonte de gestão do
  sync/rebrand/patch/deploy/rollback da documentação integrada. O destino do
  remote separado `atius-router-docs` (arquivar, remover, manter apenas como
  espelho transitório) deve ficar definido e documentado.

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
| DOCS-01 | 09 | Docs source integrado ao repo principal |
| DOCS-02 | 09 | Cutover runtime/deploy sem repo standalone |
| DOCS-03 | 09 | Gestão via omni-srv-admin + destino do remote separado |

---

## Last updated

2026-06-06 — Milestone v2.14 requirements defined (4 REQs)
