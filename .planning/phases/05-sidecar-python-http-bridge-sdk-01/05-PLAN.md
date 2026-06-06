---
phase: 05
phase_name: sidecar-python-http-bridge-sdk-01
wave: 1
depends_on: []
files_modified:
  - integration/codex-sidecar/main.py
  - integration/codex-sidecar/requirements.txt
  - integration/codex-sidecar/Dockerfile
  - podman-compose.yml
  - service/codex_sdk.go
  - relay/channel/codex/adaptor.go
autonomous: false
requirements_addressed: [SDK-01]
must_haves:
  - sidecar responde a POST /v1/codex/run com resposta do Codex SDK
  - sidecar responde a POST /v1/codex/thread com thread_id stateful
  - GET /health retorna 200
  - podman compose up codex-sidecar sobe sem erro
  - Canal Codex tipo 57 com backend=sdk faz relay via sidecar
  - Endpoint /v1/codex/run aceita stream: true e retorna text/event-stream
---

# Phase 05: Sidecar Python + HTTP Bridge (SDK-01) — Plan

**Goal:** Criar microserviço Python que encapsula o `openai-codex` SDK e expõe
endpoints HTTP. O router Go proxyia requests `backend=sdk` para o sidecar.

**Decisions (from CONTEXT.md):**
- D-01: Podman container no podman-compose.yml, rede newapi-internal
- D-02: FastAPI + uvicorn
- D-03: Sidecar gerencia app-server, token auto-refresh
- D-04: Thread state in-memory (dict Python)
- D-05: Streaming SSE suportado

## Tasks

### Task 01: Criar estrutura do sidecar Python

<read_first>
- integration/middleware/main.py (modelo FastAPI existente)
- integration/middleware/Dockerfile.fastapi (modelo Dockerfile)
- integration/middleware/requirements.txt (dependências de referência)
- relay/channel/codex/constants.go (ModelList, ChannelName)
</read_first>

<acceptance_criteria>
- integration/codex-sidecar/main.py existe e tem app FastAPI com routes
- integration/codex-sidecar/requirements.txt existe e contém `openai-codex`, `fastapi`, `uvicorn`
- python3 integration/codex-sidecar/main.py --check-syntax retorna 0 (sem erro de import)
- main.py contém endpoints: POST /v1/codex/run, POST /v1/codex/thread, GET /health
</acceptance_criteria>

<action>
Criar diretório `integration/codex-sidecar/` com 3 arquivos:

1. `main.py` — FastAPI app com:
   - `GET /health` → retorna `{"status": "ok", "models": [...]}`
   - `POST /v1/codex/run` — aceita `{model, prompt, stream?}` → chama `thread.run(prompt)` no SDK → retorna `{final_response, usage, thread_id}`
   - `POST /v1/codex/thread` — aceita `{model, prompt, thread_id?, stream?}` → stateful thread (in-memory dict `thread_id → handle`)
   - Suporte SSE: se `stream: true`, retorna `text/event-stream` com eventos `data: {...}`
   - Startup: `Codex()` context manager com lifespan FastAPI
   - Auth: lê `data/codex/license.json` (SDK-02) ou fallback `~/.codex/auth.json`

2. `requirements.txt`:
   ```
   openai-codex>=0.1.0b3
   fastapi>=0.100.0
   uvicorn>=0.22.0
   ```

3. Modelos suportados: `gpt-5.4`, `gpt-5-codex`, `gpt-5.1-codex`, `gpt-5.2-codex`, `gpt-5.3-codex`, `gpt-5.3-codex-spark` (de `relay/channel/codex/constants.go`)
</action>

### Task 02: Criar Dockerfile do sidecar

<read_first>
- integration/middleware/Dockerfile.fastapi
- integration/codex-sidecar/requirements.txt
</read_first>

<acceptance_criteria>
- integration/codex-sidecar/Dockerfile existe
- docker build -t localhost/codex-sidecar:local -f integration/codex-sidecar/Dockerfile . (sem erro)
- Imagem contém openai-codex instalado (pip freeze | grep openai-codex)
</acceptance_criteria>

<action>
Criar `integration/codex-sidecar/Dockerfile`:
- Base: `python:3.12-slim`
- WORKDIR /app
- COPY requirements.txt .
- RUN pip install --no-cache-dir -r requirements.txt
- COPY main.py .
- EXPOSE 1456
- CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "1456"]
- HEALTHCHECK: curl http://localhost:1456/health
</action>

### Task 03: Adicionar serviço codex-sidecar ao podman-compose.yml

<read_first>
- podman-compose.yml (serviços existentes: new-api, model-detailed, db-newapi)
</read_first>

<acceptance_criteria>
- podman-compose.yml contém serviço `codex-sidecar`
- Serviço está na rede `newapi-internal`
- Porta 1456 exposta internamente (sem bind no host)
- Healthcheck configurado
- volume para `./data/codex:/app/data` (para license.json)
</acceptance_criteria>

<action>
Adicionar bloco de serviço em `podman-compose.yml`:

```yaml
  codex-sidecar:
    build:
      context: ./integration/codex-sidecar
      dockerfile: Dockerfile
    container_name: codex-sidecar
    restart: always
    networks:
      - newapi-internal
    volumes:
      - ./data/codex:/app/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:1456/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
```
</action>

### Task 04: Criar handler Go para proxy do sidecar

<read_first>
- service/codex_wham_usage.go (padrão HTTP client + proxy)
- relay/channel/codex/adaptor.go (DoRequest, GetRequestURL — padrão a seguir)
- relay/common/relay_info.go (RelayInfo struct)
</read_first>

<acceptance_criteria>
- service/codex_sdk.go existe
- Função `ProxyCodexSDKRequest(ctx, client, sidecarURL, requestBody)` existe
- Função retorna `(statusCode int, body []byte, err error)`
- Usa mesmo padrão HTTP client de service/codex_wham_usage.go
</acceptance_criteria>

<action>
Criar `service/codex_sdk.go`:

- Package `service`
- Import: `context`, `fmt`, `io`, `net/http`, `strings`, `bytes`
- Função `ProxyCodexSDKRequest(ctx context.Context, client *http.Client, sidecarBaseURL string, requestBody []byte) (statusCode int, body []byte, err error)`:
  - POST para `{sidecarBaseURL}/v1/codex/run` (one-shot) ou `/v1/codex/thread` (stateful, se requestBody contém `thread_id`)
  - Content-Type: application/json
  - Timeout: 300s (chamadas SDK podem levar minutos)
  - Retorna statusCode, body, err
- Função `ProxyCodexSDKStream(ctx context.Context, client *http.Client, sidecarBaseURL string, requestBody []byte) (*http.Response, error)`:
  - POST com `stream: true`
  - Retorna response para SSE proxy transparente
</action>

### Task 05: Integrar handler no adaptor Codex

<read_first>
- relay/channel/codex/adaptor.go (GetRequestURL, DoRequest, DoResponse)
- relay/relay_adaptor.go (factory: case APITypeCodex)
</read_first>

<acceptance_criteria>
- GetRequestURL() retorna `http://codex-sidecar:1456` quando backend=sdk
- DoRequest() proxyia para sidecar quando backend=sdk
- Canal Codex tipo 57 com backend=relay continua funcionando (sem breaking)
</acceptance_criteria>

<action>
Modificar `relay/channel/codex/adaptor.go`:

1. Em `GetRequestURL()`:
   - Verificar `info.ChannelOtherSettings.CodexBackend` (ou campo equivalente)
   - Se `"sdk"` → retornar `"http://codex-sidecar:1456"` (DNS do podman compose)
   - Se `"relay"` ou vazio → comportamento atual (`chatgpt.com/backend-api/codex/responses`)

2. Em `DoRequest()`:
   - Se backend=sdk → chamar `service.ProxyCodexSDKRequest()` em vez de `channel.DoApiRequest()`
   - Se stream → chamar `service.ProxyCodexSDKStream()`

3. Definir `CodexBackend` field no channel settings (reuso de `ChannelOtherSettings` ou `Setting` JSON):
   - Campo: `codex_backend` com valores `"relay"` (default) ou `"sdk"`
   - Parse no `SetupRequestHeader()` ou `Init()`
</action>

### Task 06: Build + validate end-to-end

<read_first>
- podman-compose.yml (serviço codex-sidecar)
- integration/codex-sidecar/Dockerfile
</read_first>

<acceptance_criteria>
- podman compose build codex-sidecar (sem erro)
- podman compose up -d codex-sidecar (container sobe)
- curl -s http://localhost:1456/health retorna 200 + JSON
- curl -s -X POST http://localhost:1456/v1/codex/run -H 'Content-Type: application/json' -d '{"model":"gpt-5.4","prompt":"say hello"}' retorna JSON com final_response
- go build ./... (sem erro no handler Go)
</acceptance_criteria>

<action>
Sequência de validação:
1. `podman compose build codex-sidecar`
2. `podman compose up -d codex-sidecar`
3. `sleep 10 && curl -s http://localhost:1456/health`
4. `curl -s -X POST http://localhost:1456/v1/codex/run -H 'Content-Type: application/json' -d '{"model":"gpt-5.4","prompt":"say hello in one word"}'`
5. `go build ./...`
6. `go vet ./service/ ./relay/channel/codex/`
</action>

---

## Verification

- [ ] sidecar responde POST /v1/codex/run → JSON com final_response
- [ ] sidecar responde POST /v1/codex/thread → stateful (mesmo thread_id retorna contexto)
- [ ] GET /health → 200 + `{"status": "ok"}`
- [ ] podman compose ps mostra codex-sidecar healthy
- [ ] go build ./... sem erro
- [ ] Canal Codex tipo 57 com backend=relay inalterado (teste de regressão: criar canal relay e testar)

## Notes

- SDK-02 (login explícito) é Phase 06 — sidecar lê `data/codex/license.json` mas o fluxo de auth é fase separada
- SDK-03 (dashboard usage) é Phase 07
- SDK-04 (channel coexistence final) é Phase 08
- Sidecar usa fallback `~/.codex/auth.json` se `data/codex/license.json` não existir (para desenvolvimento local antes do SDK-02)
