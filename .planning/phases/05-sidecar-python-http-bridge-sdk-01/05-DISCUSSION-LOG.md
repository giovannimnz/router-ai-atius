# Phase 05: Sidecar Python + HTTP Bridge (SDK-01) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-06
**Phase:** 05-Sidecar Python + HTTP Bridge (SDK-01)
**Areas discussed:** Runtime & packaging, HTTP framework, SDK lifecycle, Thread state

---

## Runtime & Packaging

| Option | Description | Selected |
|--------|-------------|----------|
| Docker container no compose | Serviço `codex-sidecar` no docker-compose.yml, rede `newapi-internal` | ✓ |
| PM2/systemd no host | Processo Python gerenciado pelo PM2 no host (fora do container) | |

**User's choice:** Timeout — best-judgment default Docker (consistente com new-api, model-detailed, db-newapi)
**Notes:** Container padrão garante isolamento. Não depende do host ter Python/Codex SDK instalado.

---

## HTTP Framework

| Option | Description | Selected |
|--------|-------------|----------|
| FastAPI + uvicorn | Async nativo, já usado no model-detailed middleware | ✓ |
| Flask | Simples, mas sync-only | |
| aiohttp | Async mas sem OpenAPI/docs automáticos | |

**User's choice:** Timeout — best-judgment default FastAPI (padrão existente no projeto)
**Notes:** Modelo `integration/middleware/Dockerfile.fastapi` reutilizável.

---

## SDK Lifecycle

| Option | Description | Selected |
|--------|-------------|----------|
| Sidecar gerencia app-server | `Codex()` context manager inicia/shutdown codex subprocess | ✓ |
| Assume app-server externo | Sidecar só conecta num codex já rodando em outra porta | |

**User's choice:** Timeout — best-judgment default managed (SDK faz spawn automático)
**Notes:** O `openai-codex` SDK já gerencia o subprocesso nativamente.

---

## Thread State

| Option | Description | Selected |
|--------|-------------|----------|
| In-memory dict | `thread_id → Codex thread handle` em dict Python | ✓ |
| SQLite | Persistência local — threads sobrevivem a restart | |
| Redis | Persistência distribuída — suporte multi-instância | |

**User's choice:** Timeout — best-judgment default in-memory (minimal pra v2.14)
**Notes:** Persistência postergada pra Phase 08 ou milestone futuro se necessário.

---

## Claude's Discretion

Todas as 4 áreas foram defaulted por timeout. O agente tem flexibilidade para
ajustar detalhes de implementação que não contradigam D-01..D-04.

## Deferred Ideas

- Thread persistence (SQLite/Redis) — Phase 08+
- Streaming SSE do sidecar — avaliar durante implementação
