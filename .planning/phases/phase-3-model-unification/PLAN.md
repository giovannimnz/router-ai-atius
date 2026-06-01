# Phase 3 Plan — Model Unification via FastAPI Middleware

## Objective

Unificar listagem de modelos OpenAI e Anthropic em `GET /v1/models` via FastAPI middleware, com `api_format`区分 formato. Deprecar `GET /v1/claude/models`.

## Tasks

### Task 1: Adicionar endpoint interno no Go para listar modelos
**File:** `router/relay-router.go`
**New route:** `GET /internal/v1/models` (sem auth, retorna abilities com channel_type)

```go
internalRouter := router.Group("/internal/v1")
internalRouter.Use(middleware.RouteTag("internal"))
{
    internalRouter.GET("/models", controller.ListAllModelsWithChannel)
}
```

**Controller:** `controller/model.go` — `ListAllModelsWithChannel()`
- Query abilities com channel_type (JOIN channels)
- Retorna JSON com `[{model, channel_type, enabled, priority, ...}]`
- Sem autenticação

**Verification:**
```bash
curl -s http://localhost:3001/internal/v1/models | python3 -m json.tool | head -30
```

### Task 2: Modificar FastAPI para detectar api_format
**File:** `integration/middleware/model_detailed_fastapi.py`
**Detection logic:**
```python
# Na rota GET /v1/models
api_format = request.query_params.get("api_format", "openai")
# Se não especificado, default OpenAI
# Alternativa: detectar via Accept header ou anthropic-version
```

Adicionar query param `?api_format=anthropic|openai` (default: `openai`).

### Task 3: FastAPI chama Go internamente e transforma resposta
**File:** `integration/middleware/model_detailed_fastapi.py`

```python
# GET /v1/models
# 1. Chamar http://new-api:3000/internal/v1/models
# 2. Receber abilities com channel_type
# 3. Se api_format=openai: filtrar channel_type=0, formatar OpenAI
# 4. Se api_format=anthropic: filtrar channel_type=14, formatar Anthropic
# 5. Enriquecer com pricing/context_length (lógica existente)
```

**Estrutura de resposta OpenAI:**
```json
{
  "data": [
    {
      "id": "MiniMax-M2.7",
      "object": "model",
      "created": 1626777600,
      "owned_by": "atius",
      "api_format": "openai"
    }
  ],
  "object": "list"
}
```

**Estrutura de resposta Anthropic:**
```json
{
  "data": [
    {
      "id": "MiniMax-M2.7",
      "created_at": "2021-07-20T10:40:00Z",
      "display_name": "MiniMax-M2.7",
      "type": "model",
      "api_format": "anthropic"
    }
  ],
  "has_more": false
}
```

### Task 4: Adicionar deprecação em /v1/claude/models
**File:** `router/relay-router.go` OU `controller/model.go`

Na rota `GET /v1/claude/models`:
```go
c.Header("Deprecation", "true")
c.Header("Sunset", "Sat, 01 Jan 2027 00:00:00 GMT")
c.Header("Link", "</v1/models?api_format=anthropic>; rel=\"successor-version\"")
```

Retornar redirect 301 para `/v1/models?api_format=anthropic` OU manter 200 com body原有的 e headers de deprecação.

### Task 5: Rebuild e deploy
**Build Go:**
```bash
docker run --rm \
  -v /home/ubuntu/docker/Atius/router-ai-atius:/app \
  -v /home/ubuntu:/hosthome \
  -w /app \
  -e GO111MODULE=on \
  -e CGO_ENABLED=0 \
  golang:1.26.1-alpine \
  sh -c "go build -ldflags '-s -w -X github.com/QuantumNous/new-api/common.Version=atius-dev-v3' -o /hosthome/new-api-build/new-api ."
```

**Deploy:**
```bash
docker stop new-api && docker cp /home/ubuntu/new-api-build/new-api new-api:/new-api && docker start new-api
```

**Rebuild FastAPI:**
```bash
cd /home/ubuntu/docker/Atius/router-ai-atius/integration/middleware && \
docker build -f Dockerfile.fastapi -t ghcr.io/giovannimnz/router-ai-atius/model-detailed:latest .
```

**Redeploy FastAPI:**
```bash
docker stop model-detailed && docker run -d --name model-detailed \
  -p 3300:3300 \
  -e DOCS_USERNAME=admin -e DOCS_PASSWORD=atius2024 \
  --restart unless-stopped \
  ghcr.io/giovannimnz/router-ai-atius/model-detailed:latest
```

### Task 6: Atualizar Bruno tests
**File:** `integration/bruno-tests/atius-claude-models/`

Criar/atualizar:
- `unified-models-openai.bru` — GET /v1/models?api_format=openai
- `unified-models-anthropic.bru` — GET /v1/models?api_format=anthropic
- `deprecated-claude-models.bru` — GET /v1/claude/models (verifica header Deprecation)

**Verification:**
```bash
cd integration/bruno-tests/atius-claude-models && \
/home/ubuntu/.nvm/versions/node/v24.13.1/bin/bru run . --env-var "baseUrl=https://router.atius.com.br" --env-var "apiToken=9cfec16339f2306085cc45124b1b62e691f621fe82bbdc92"
```

### Task 7: Teste E2E
```bash
# OpenAI format
curl -s "https://router.atius.com.br/v1/models?api_format=openai" \
  -H "Authorization: Bearer 9cfec16339f2306085cc45124b1b62e691f621fe82bbdc92" \
  | python3 -m json.tool | head -30

# Anthropic format
curl -s "https://router.atius.com.br/v1/models?api_format=anthropic" \
  -H "Authorization: Bearer 9cfec16339f2306085cc45124b1b62e691f621fe82bbdc92" \
  | python3 -m json.tool | head -30

# Deprecated endpoint
curl -sv "https://router.atius.com.br/v1/claude/models" \
  -H "Authorization: Bearer 9cfec16339f2306085cc45124b1b62e691f621fe82bbdc92" 2>&1 | grep -i "deprecation\|sunset\|link\|HTTP/"
```

## Files Modified

| File | Change |
|------|--------|
| `router/relay-router.go` | Adicionar `/internal/v1/models` route |
| `controller/model.go` | Adicionar `ListAllModelsWithChannel()` |
| `integration/middleware/model_detailed_fastapi.py` | Detectar api_format, chamar Go, transformar resposta |
| `integration/middleware/Dockerfile.fastapi` | Rebuild |

## Verification

1. `GET /v1/models?api_format=openai` retorna array com `api_format: "openai"`
2. `GET /v1/models?api_format=anthropic` retorna array com `api_format: "anthropic"`
3. `GET /v1/claude/models` retorna header `Deprecation: true`
4. Bruno tests 5/5 pass
5. Sem breaking changes no relay (POST /v1/chat/completions e /v1/messages funcionam)
