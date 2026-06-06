# Phase 05: Sidecar Python + HTTP Bridge (SDK-01) - Context

**Gathered:** 2026-06-06
**Status:** Ready for planning (refined by user)

## Phase Boundary

Criar microserviço Python que encapsula o `openai-codex` SDK (v0.1.0b3)
e expõe endpoints HTTP para o router Go proxyar. O sidecar traduz
requests do formato do router → chamadas `thread.run()` no SDK Codex.
Suporta modelos `gpt-5.4`, `gpt-5-codex`, `gpt-5.1-codex`, `gpt-5.2-codex`,
`gpt-5.3-codex`, `gpt-5.3-codex-spark`.

## Implementation Decisions

### Runtime & Packaging
- **D-01:** Container Podman no mesmo `podman-compose.yml`. Serviço `codex-sidecar`
  na rede `newapi-internal`. Imagem base: `python:3.12-slim` com `pip install
  openai-codex`. Stack é sempre Podman — `podman compose`, não Docker.

### HTTP Framework
- **D-02:** FastAPI com uvicorn. Já usado no `model-detailed` middleware
  (`integration/middleware/Dockerfile.fastapi`). Async nativo — importante
  porque chamadas SDK podem levar minutos. Integrado no sistema como
  serviço Podman, mesmo padrão do `model-detailed`.

### SDK Lifecycle
- **D-03:** Sidecar gerencia o ciclo de vida do `codex` app-server. O SDK
  Python spawna automaticamente o binário `codex` como subprocesso. O sidecar
  inicia no startup (`Codex()` context manager), health check, graceful
  shutdown no SIGTERM. Token refresh automático — depois de autenticado
  (SDK-02), a goroutine `codex_credential_refresh_task.go` renova
  automaticamente sem pedir código. Nunca interrompe o usuário.

### Thread State
- **D-04:** Estado dos threads em memória (dict Python: `thread_id → Codex
  thread handle`). v2.14 é minimal — persistência (SQLite/Redis) seria
  overengineering. Se precisar de threads persistentes entre restarts,
  refatora na Phase 08.

### Streaming
- **D-05:** Sidecar suporta streaming SSE. Endpoint `/v1/codex/run` aceita
  `stream: true` no request body e retorna `text/event-stream`. O router Go
  faz proxy transparente do SSE do sidecar → cliente final. Usa o mesmo
  padrão de streaming do relay HTTP existente (`oaiResponsesStreamHandler`).

### Claude's Discretion
4 áreas foram defaulted inicialmente por timeout. Refinadas pelo usuário:
D-01 (Podman), D-02 (FastAPI integrado), D-03 (auto-refresh token),
D-04 (in-memory), D-05 (streaming SSE adicionado). Agente tem flexibilidade
para detalhes de implementação que não contradigam D-01..D-05.

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Requirements
- `.planning/REQUIREMENTS.md` — SDK-01 scope, acceptance criteria
- `.planning/ROADMAP.md` § Phase 05 — Goal, verification steps

### Project Context
- `.planning/PROJECT.md` — Project conventions, AGENTS.md rules, v2.14 premissas
- `.planning/STATE.md` — Current milestone status, execution order

### Existing Codex Integration (study before implementing)
- `relay/channel/codex/adaptor.go` — Adaptor atual (HTTP relay). Padrão a seguir para o novo handler que proxyia pro sidecar
- `relay/channel/adapter.go` — Adaptor interface (`GetRequestURL`, `SetupRequestHeader`, `DoRequest`, etc.)
- `relay/relay_adaptor.go` — Factory: `case constant.APITypeCodex: return &codex.Adaptor{}`
- `constant/channel.go` — `ChannelTypeCodex = 57`
- `constant/api_type.go` — `APITypeCodex`
- `relay/common/relay_info.go` — `RelayInfo` struct (ChannelBaseUrl, ApiKey, IsStream, etc.)

### Middleware Pattern (modelo FastAPI)
- `integration/middleware/main.py` — FastAPI app com health check, proxy, auth
- `integration/middleware/Dockerfile.fastapi` — Dockerfile FastAPI com uvicorn
- `docker-compose.yml` — Serviço `model-detailed` como template pro `codex-sidecar`

### Codex SDK
- `pip install openai-codex` v0.1.0b3
- GitHub: `openai/codex/sdk/python/README.md`
- Auth: reusa `~/.codex/auth.json` automaticamente OU aceita credentials via env/header (SDK-02 na próxima fase)

## Existing Code Insights

### Reusable Assets
- **Dockerfile.fastapi** (`integration/middleware/Dockerfile.fastapi`): Template direto — copiar, ajustar entrypoint, trocar `pip install` dependencies
- **docker-compose.yml service block** (`model-detailed`): Template para o serviço `codex-sidecar` — network, restart, healthcheck, depends_on
- **Adaptor interface** (`relay/channel/adapter.go`): O novo handler Go que proxyia pro sidecar implementa a mesma interface. `GetRequestURL()` aponta pra `http://codex-sidecar:1456` em vez de `chatgpt.com`

### Established Patterns
- **Services na rede `newapi-internal`**: DNS interno do Docker resolve `codex-sidecar` → container IP. Mesmo padrão de `db-newapi` e `model-detailed`
- **Healthcheck via wget**: `wget -O- -q http://localhost:1456/health` (padrão dos serviços existentes)
- **Go proxy pattern**: `service/` faz `http.NewRequestWithContext` → `client.Do()` → resposta parseada. Mesmo padrão de `service/codex_wham_usage.go`

### Integration Points
- **Router** (`router/api-router.go` § channel routes): Rotas Codex já registradas. Nova rota opcional: `/api/channel/:id/codex/sdk/status` pra verificar saúde do sidecar
- **docker-compose.yml**: Adicionar serviço `codex-sidecar` (porta 1456 interna, sem bind no host)
- **Go Adaptor**: Estender `relay/channel/codex/adaptor.go` com branch `if backend == "sdk"` → proxyia pra sidecar em vez de chatgpt.com. Ou criar novo handler em `service/codex_sdk.go`

## Specific Ideas

Nenhuma — discussão foi defaulted por timeout. Sidecar segue padrão FastAPI
existente, Docker compose, sem surpresas.

## Deferred Ideas

- **Thread persistence (SQLite/Redis)**: Postergado pra Phase 08 ou milestone futuro.

---

*Phase: 05-Sidecar Python + HTTP Bridge (SDK-01)*
*Context gathered: 2026-06-06*
