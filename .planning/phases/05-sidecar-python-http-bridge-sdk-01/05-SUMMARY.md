---
phase: 05
plan: 05-PLAN
status: completed
date: 2026-06-06
commits:
  - feat(05-01): add codex sidecar Python app (FastAPI + openai-codex SDK bridge)
  - feat(05-02): add Dockerfile for codex sidecar (python:3.12-slim + openai-codex)
  - feat(05-03): add codex-sidecar service to docker-compose.yml
  - feat(05-04): add Go handler for Codex sidecar proxy (service/codex_sdk.go)
  - feat(05-05): wire Codex adaptor for SDK sidecar backend (GetRequestURL, DoRequest, DoResponse)
key-files:
  created:
    - integration/codex-sidecar/main.py
    - integration/codex-sidecar/requirements.txt
    - integration/codex-sidecar/Dockerfile
    - service/codex_sdk.go
  modified:
    - docker-compose.yml
    - relay/channel/codex/adaptor.go
    - dto/channel_settings.go
---

# Phase 05: Sidecar Python + HTTP Bridge (SDK-01) — Summary

**Status:** Completed (awaiting containerized build on deployment host)

## What was built

Microserviço Python (FastAPI) que encapsula o `openai-codex` SDK v0.1.0b3 e
expõe endpoints HTTP na porta 1456. O router Go foi estendido com handler
e adaptor wiring para proxyar requests `backend=sdk` para o sidecar.

## Architecture

```
[Client] → [Router Go] → isSDKBackend? → [codex-sidecar:1456] → [openai-codex SDK]
                         → else         → [chatgpt.com/backend-api/codex/responses]
```

## Changes by file

| File | Change |
|---|---|
| `integration/codex-sidecar/main.py` | FastAPI app: `/health`, `/v1/codex/run` (one-shot), `/v1/codex/thread` (stateful), SSE streaming |
| `integration/codex-sidecar/requirements.txt` | `openai-codex>=0.1.0b3`, `fastapi`, `uvicorn` |
| `integration/codex-sidecar/Dockerfile` | `python:3.12-slim` + pip install + uvicorn na 1456 |
| `docker-compose.yml` | Serviço `codex-sidecar` (build, network, healthcheck, volume `./data/codex`) |
| `service/codex_sdk.go` | `ProxyCodexSDKRequest()`, `ProxyCodexSDKThread()`, `ProxyCodexSDKStream()` |
| `relay/channel/codex/adaptor.go` | `isSDKBackend()`, `GetRequestURL` → sidecar URL, `DoRequest` → service proxy, `DoResponse` → forward body |
| `dto/channel_settings.go` | Campo `CodexBackend string` ("relay" default, "sdk" para sidecar) |

## Decisions implemented

| D-NN | Decision | Implementation |
|------|----------|---------------|
| D-01 | Podman container | Serviço no docker-compose.yml, rede `new-api-network` |
| D-02 | FastAPI + uvicorn | `main.py` com lifespan, async endpoints, pydantic models |
| D-03 | Sidecar gerencia SDK | `Codex().__enter__()` no startup, `__exit__` no shutdown |
| D-04 | Thread state in-memory | `_threads: dict[str, CodexThread]` no escopo do módulo |
| D-05 | Streaming SSE | `ProxyCodexSDKStream()` + resposta `text/event-stream` |

## Verification

| Check | Result |
|---|---|
| Python syntax (ast.parse) | ✅ OK |
| Compose YAML (yaml.safe_load) | ✅ OK |
| Go build (gofmt/go vet) | ⚠️ N/A — Go não instalado neste host. Build via container |
| podman compose build | ⚠️ Pending — requer deployment host (SRV-1) |
| curl health check | ⚠️ Pending — sidecar precisa subir primeiro |

## Pending (post-deployment)

- `podman compose build codex-sidecar` no SRV-1
- `podman compose up -d codex-sidecar`
- `curl http://localhost:1456/health`
- `go build ./...` dentro do container `new-api`
- Criar canal Codex tipo 57 com `codex_backend: "sdk"` no admin e testar relay
