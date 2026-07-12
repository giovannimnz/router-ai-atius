# Phase 2 Plan — Fix /v1/claude/models + Cleanup

## Objective

Fix `GET /v1/claude/models` returning `{"data": null}` — root cause is middleware TokenAuth setting `modelLimitEnabled=true` from DB field instead of using `IsModelLimitsEnabled()` method. Then cleanup debug logs and commit.

## Tasks

### Task 1: Patch middleware/auth.go — usar IsModelLimitsEnabled()
**File:** `middleware/auth.go`
**Problem:** TokenAuth middleware sets `c.Set(constant.ContextKeyTokenModelLimitEnabled, token.ModelLimitsEnabled)` — campo DB booleano direto. Para Giovanni-Acc, DB=true mas `model_limits='{}'` (vazio). O método `IsModelLimitsEnabled()` filtra esse caso, mas middleware ignora.
**Fix:** Trocar para usar o método `token.IsModelLimitsEnabled()` no lugar do campo direto.

**Verification:**
```bash
curl -s "https://router.atius.com.br/v1/claude/models" \
  -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" \
  | python3 -m json.tool
# Esperado: data com array de modelos (não null)
```

### Task 2: Rebuild binary (atius-dev-final)
**Command:**
```bash
docker run --rm \
  -v /home/ubuntu/docker/Atius/router-ai-atius:/app \
  -v /home/ubuntu:/hosthome \
  -w /app \
  -e GO111MODULE=on \
  -e CGO_ENABLED=0 \
  golang:1.26.1-alpine \
  sh -c "go build -ldflags '-s -w -X github.com/QuantumNous/new-api/common.Version=atius-dev-final' -o /hosthome/new-api-build/new-api ."
```

### Task 3: Deploy
```bash
docker stop new-api && docker cp /home/ubuntu/new-api-build/new-api new-api:/new-api && docker start new-api && sleep 5
```

### Task 4: Test endpoint
```bash
curl -s "https://router.atius.com.br/v1/claude/models" \
  -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" \
  | python3 -m json.tool
```

### Task 5: Remover debug logs do controller/model.go
Remover todos os `common.SysLog("DEBUG ListClaudeModels: ...")` adicionados para debug.

### Task 6: Commit
```bash
git add -A && git commit -m "fix: resolve /v1/claude/models empty response — use IsModelLimitsEnabled() in middleware"
```

### Task 7: Push image to GHCR
```bash
docker tag ghcr.io/giovannimnz/atius-ai-router:arm64-test ghcr.io/giovannimnz/atius-ai-router:latest
docker push ghcr.io/giovannimnz/atius-ai-router:latest
docker push ghcr.io/giovannimnz/atius-ai-router:arm64-test
```

### Task 8: Criar Bruno test collection
Criar `integration/bruno-tests/atius-claude-models/` com:
- `GET /v1/claude/models` — list all Anthropic models
- `POST /v1/messages` com MiniMax — relay test
- `.env.local` com token

## Files Modified
- `middleware/auth.go` (Task 1)
- `controller/model.go` (Task 5 — remove debug logs)

## Verification
- `GET /v1/claude/models` retorna `{"data":[...models...],"has_more":false}`
- `data` é array com modelos MiniMax (não null)
- Commit no git
- Image no GHCR atualizada
