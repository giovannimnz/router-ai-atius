# Atius AI Router — Testing

## 1. Test Suite Overview

| Suite | Location | Tool | Purpose |
|-------|----------|------|---------|
| API Tests | `integration/bruno-tests/` | Bruno CLI | End-to-end API validation |
| Go Unit Tests | `**/*_test.go` | `go test` | Backend logic |
| Go Integration | `**/*_internal_test.go` | `go test -tags=integration` | DB-level tests |
| Frontend | `web/default/` | Bun + Vitest | React component tests |

## 2. Bruno API Tests

### 2.1 Setup

```bash
# Instalar Bruno CLI
# https://www.usebruno.com/downloads

# Bruno está em: integration/bruno-tests/atius-router-tests/
```

### 2.2 Run All Tests

```bash
./scripts/run-bruno-tests.sh
```

### 2.3 Run Specific Test

```bash
# List models
bruno tests/atius-router-tests/list-models.bru --env .env

# MiniMax M2.7
bruno tests/atius-router-tests/minimax-m27.bru --env .env

# DeepSeek
bruno tests/atius-router-tests/deepseek-chat.bru --env .env
```

### 2.4 Available Tests

| File | Endpoint | Model | Assertions |
|------|----------|-------|-----------|
| `list-models.bru` | GET /v1/models | — | 200, `object: "list"`, models array |
| `minimax-m27.bru` | POST /v1/chat/completions | MiniMax-M2.7 | 200, content not empty |
| `minimax-m25.bru` | POST /v1/chat/completions | MiniMax-M2.5 | 200, content not empty |
| `deepseek-chat.bru` | POST /v1/chat/completions | deepseek-chat | 200, content not empty |
| `deepseek-reasoner.bru` | POST /v1/chat/completions | deepseek-reasoner | 200, reasoning content |

### 2.5 Environment File

Bruno tests expect `.env` in project root:

```bash
# .env
TOKEN=your_api_token_here
BASE_URL=http://localhost:3300
```

## 3. Go Unit Tests

### 3.1 Run All Tests

```bash
# Inside container
docker exec new-api go test ./...

# Outside (requires Go toolchain)
cd /home/ubuntu/docker/Atius/router-ai-atius
go test ./...
```

### 3.2 Run Specific Package

```bash
docker exec new-api go test ./service/...
docker exec new-api go test ./relay/...
docker exec new-api go test ./middleware/...
```

### 3.3 Run with Coverage

```bash
docker exec new-api go test -cover ./...
docker exec new-api go test -coverprofile=coverage.out ./...
docker exec new-api go tool cover -html=coverage.out
```

### 3.4 Key Test Files

| File | What it tests |
|------|--------------|
| `service/error_test.go` | Error formatting/handling |
| `service/task_billing_test.go` | Task billing calculations |
| `service/text_quota_test.go` | Text quota logic |
| `controller/model_list_test.go` | Model list endpoint |
| `controller/channel_test.go` | Channel operations |
| `controller/channel_test_internal_test.go` | Internal channel tests |
| `relay/api_request_test.go` | HTTP request building |
| `payment_webhook_availability_test.go` | Payment webhook logic |

### 3.5 Integration Tests

```bash
# Run with integration tag (requires DB)
docker exec new-api go test -tags=integration ./...
```

## 4. Frontend Tests

```bash
cd web/default

# Run all tests
bun run test

# Run with coverage
bun run test --coverage

# Watch mode
bun run test --watch

# Typecheck
bun run typecheck

# Lint
bun run lint
```

## 5. Manual API Testing

### 5.1 Health Check

```bash
# Middleware (enriched)
curl http://localhost:3300/api/status

# Direct new-api
curl http://localhost:3301/api/status
```

### 5.2 Model List

```bash
curl -s http://localhost:3300/v1/models \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

### 5.3 Chat Completion

```bash
curl -X POST http://localhost:3300/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "messages": [{"role": "user", "content": "Say hello in 3 words"}],
    "max_tokens": 20
  }' | python3 -m json.tool
```

### 5.4 Streaming Chat Completion

```bash
curl -X POST http://localhost:3300/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "messages": [{"role": "user", "content": "Count to 5"}],
    "max_tokens": 20,
    "stream": true
  }'
```

### 5.5 Anthropic Messages

```bash
curl -X POST http://localhost:3300/v1/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "x-api-key: $TOKEN" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "max_tokens": 50,
    "messages": [{"role": "user", "content": "Hi"}]
  }'
```

## 6. Database Testing

### 6.1 Direct Query

```bash
docker exec -it db-newapi psql -U admin -d newapi

# List channels
SELECT id, name, base_url FROM channels;

# List models
SELECT id, name, context_length FROM models;

# Check tokens
SELECT id, user_id, key FROM tokens LIMIT 5;
```

### 6.2 Test User Registration

```bash
curl -X POST http://localhost:3300/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'
```

## 7. Adding a New Test

### Bruno Test

```bash
# Create .bru file
touch integration/bruno-tests/atius-router-tests/my-new-test.bru
```

```bru
meta {
  name: My New Test
  type: http
  method: post
  path: /v1/chat/completions
}

headers {
  Content-Type: application/json
  Authorization: Bearer {{TOKEN}}
}

body {
  json: {
    model: MiniMax-M2.7
    messages: [
      {
        role: user
        content: Hi
      }
    ]
    max_tokens: 20
  }
}

assert {
  status: 200
  res.status: 200
  res.body.choices[0].message.content: (len) > 0
}
```

### Go Test

```go
// service/myfeature_test.go
package service

import (
    "testing"
)

func TestMyFeature(t *testing.T) {
    result := MyFunction("input")
    if result != "expected" {
        t.Errorf("expected 'expected', got '%s'", result)
    }
}
```

## 8. CI/CD

GitHub Actions workflows:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `sync.yml` | Daily 03:00 UTC | Sync with upstream |
| `release.yml` | On tag `v*` | Create GitHub release |

No formal CI pipeline for tests yet. Manual execution via `./scripts/run-bruno-tests.sh`.

---

_Last updated: 2026-05-31_
