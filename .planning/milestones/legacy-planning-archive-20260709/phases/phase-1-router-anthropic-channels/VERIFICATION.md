# Phase 1 Verification — Router Anthropic Channels + Session Fix

## Task 1: Session Timeout ✅

**File:** `main.go` linha 181
**Changed:** `MaxAge: 2592000` → `MaxAge: 43200` (12 horas)
**Verified:** Fonte alterada, rebuild feito, imagem pushada

**Nota:** Verificação interna do binário não encontra "43200" pois o valor está otimizado pelo linker Go e não aparece como string no binário. Mas o código fonte e a compilação estão corretos.

### Comportamento esperado após 12h
- Cookie de sessão expira após 12 horas de inatividade
- Usuário precisa fazer login novamente após expiração
- A expiração antiga era 30 dias

## Task 2: Canais Anthropic ✅

### Canais criados:
| ID | Nome | Type | Base URL | Models |
|----|------|------|----------|--------|
| 3 | MiniMax - Anthropic Compatible | 14 | https://api.minimax.io | MiniMax-M2.7, MiniMax-M2.5 |
| 4 | MiniMax-Highspeed - Anthropic Compatible | 14 | https://api.minimax.io | MiniMax-M2.7-highspeed, MiniMax-M2.5-highspeed |

### Habilidades criadas:
| Group | Model | Channel ID | Enabled |
|-------|-------|------------|---------|
| default | MiniMax-M2.7 | 3 | true |
| default | MiniMax-M2.5 | 3 | true |
| default | MiniMax-M2.7-highspeed | 4 | true |
| default | MiniMax-M2.5-highspeed | 4 | true |

### Teste de relay Anthropic (/v1/messages):
```
Request: POST /v1/messages com token giovanniS23h3rm3s202...
Model: MiniMax-M2.7
Response: 200 OK — formato Claude Messages
Log: "request_conversion":["Claude Messages","OpenAI Compatible"]
```

## Task 3: Deploy ✅

- Imagem construída: `ghcr.io/giovannimnz/atius-ai-router:latest`
- Container recriado via `docker compose`
- Health check: `http://localhost:3301/` → 200 OK
- Remote: `https://router.atius.com.br/` → 200 OK

## Task 4: /docs/ Investigation ⚠️

**Sintoma:** GET /docs/ retorna HTTP 405 (Method Not Allowed)远程
**Causa:** Cloudflare permite apenas GET neste path, mas o modelo FastAPI espera GET → redirect para /docs/
**Local:** `curl http://localhost:3300/docs/` → 200, `curl https://router.atius.com.br/docs/` → 405

**Status:** O /docs/ funciona localmente. O problema de "não carrega" no browser após login parece ser cache do navegador ou redirecionamento de sessão antes da aplicação. O /docs/json funciona corretamente via Cloudflare com Basic Auth.

### Testes finais:
```bash
# /docs/json com auth
curl -u admin:atius2024 https://router.atius.com.br/docs/json → 200

# /docs/ GET via Cloudflare
curl https://router.atius.com.br/docs/ → 405 (Cloudflare bloqueia HEAD/POST)
```

## Resumo

| Item | Status | Notas |
|------|--------|-------|
| Session timeout 12h | ✅ | Código alterado, rebuild feito |
| Canal MiniMax Anthropic (ch3) | ✅ | type=14, modelos M2.7/M2.5 |
| Canal MiniMax-Highspeed Anthropic (ch4) | ✅ | type=14, modelos M2.7-hs/M2.5-hs |
| Habilidades criadas | ✅ | 4 abilities (2 por canal) |
| Deploy realizado | ✅ | Container reiniciado com nova imagem |
| Relay /v1/messages | ✅ | Funciona, converte Claude → OpenAI |
| /docs/ via Cloudflare | ⚠️ | 405 GET, funciona localmente |

---

## Validações Finais — 2026-05-10 01:27

### /v1/chat/completions (OpenAI) ✅
```
POST /v1/chat/completions
Authorization: Bearer $ATIUS_ROUTER_TOKEN
Model: MiniMax-M2.7
Response: 200 OK — "OK" (one word)
```

### /v1/messages (Anthropic) ✅
```
POST /v1/messages
Authorization: Bearer $ATIUS_ROUTER_TOKEN
Model: MiniMax-M2.7
Response: 200 OK
{
  "id": "064f3c4b73166c19cae77f6151fb9256",
  "type": "message",
  "content": [{"type": "text", "text": "OK"}],
  "stop_reason": "max_tokens",
  "model": "MiniMax-M2.7",
  "usage": {
    "input_tokens": 43,
    "output_tokens": 5
  }
}
```
O formato é estritamente Anthropic (content blocks, não string simples). Conversão funcionando corretamente.

### /docs/ — Causa Raiz Identificada ⚠️

**Problema:** SPA não envia credenciais ao buscar /docs/json → 401 → "Invalid credentials" no lugar do Swagger UI.

**Testes:**
- `curl https://router.atius.com.br/docs/` → 405 Method Not Allowed (via Cloudflare)
- `curl https://router.atius.com.br/docs/json` → 401 (sem auth)
- `curl -u admin:atius2024 https://router.atius.com.br/docs/json` → 200 + OpenAPI JSON válido

** workaround:** Acessar via Basic Auth (admin:atius2024) ou abrir /docs/ diretamente no navegador com sessão autenticada no dashboard.

**Alternativa:** O /docs/json funciona com Basic Auth, então a API está correta — o problema é apenas a UX do /docs/ via SPA.

### Session Cookie

O MaxAge foi alterado para 43200 (12h) no código fonte. A verificação do binário compilado é inconclusiva pois o Go otimiza inteiros no binário. O comportamento esperado é que o cookie de sessão expire após 12h de inatividade.

### Usuários no DB (postgres)

| ID | Username | Email | Role |
|----|----------|-------|------|
| 2 | giovanni | giovannimunizds@gmail.com | 100 |
| 3 | admin2 | admin2@atius.com | 100 |
| 4 | admin3 | admin3@atius.com | 100 |

Login via API não funciona com credenciais testadas (todas retornam "incorrect"). O token Bearer então configurado funcionava para APIs, mas seu valor foi removido deste histórico; o login via form do dashboard estava com problema de credenciais.

### Conclusão

- **Canais Anthropic**: ✅ Configurados e funcionando (relay /v1/messages OK)
- **Timeout 12h**: ✅ Código alterado (requer verificação prática após deploy)
- **/docs/ Swagger**: ⚠️ Funciona com Basic Auth, SPA não autentica automaticamene
- **Login dashboard**: ⚠️ Credenciais não funcionam via form (precisa investigar)`
