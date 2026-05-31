# Atius AI Router — Architecture

## 1. Overview

Layered Go application that acts as an API gateway aggregating 40+ AI providers. Requests flow through relay adapters specific to each provider.

```
                    ┌─────────────────────────────────────┐
                    │           Client Layer              │
                    │   (OpenAI SDK, Anthropic SDK, curl)  │
                    └──────────────┬──────────────────────┘
                                   │ HTTPS :3300 (middleware)
                                   ▼
                    ┌─────────────────────────────────────┐
                    │      model-detailed (Python FastAPI) │
                    │  Port 3001 → 3300 (host)             │
                    │  • Enriches GET /v1/models           │
                    │  • Forwards other requests to new-api │
                    └──────────────┬──────────────────────┘
                                   │ Internal network
                                   ▼
                    ┌─────────────────────────────────────┐
                    │          new-api (Go + Gin)          │
                    │  Port 3000 → 3301 (host)             │
                    │  • Relay handlers (openai/claude/etc) │
                    │  • Auth, rate-limiting, distribution  │
                    └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────┼──────────────────────┐
                    ▼              ▼                      ▼
            ┌───────────┐  ┌─────────────┐        ┌────────────┐
            │ PostgreSQL │  │  Upstream   │        │   Redis    │
            │ db-newapi │  │  Providers  │        │  (cache)   │
            │ :5432     │  │ (api.minimax│        │            │
            └───────────┘  │ .io, etc)   │        └────────────┘
                           └─────────────┘
```

## 2. Directory Structure

```
router-ai-atius/
├── controller/          # HTTP request handlers (route -> controller)
│   ├── relay.go        # Main relay handler dispatcher
│   ├── channel.go     # Channel CRUD
│   ├── model.go       # Model management
│   ├── token.go       # Token management
│   ├── billing*.go    # Billing controllers
│   └── oauth*.go      # OAuth handlers
├── service/           # Business logic layer
│   ├── channel.go     # Channel selection/routing
│   ├── channel_select.go
│   ├── quota.go       # Quota management
│   ├── task.go        # Task operations
│   ├── billing_session.go
│   └── error.go
├── model/            # GORM data models
│   ├── main.go       # DB connection, migrations, common cols
│   ├── channel.go    # Channel model
│   ├── token.go      # Token model
│   └── ...
├── relay/           # Upstream AI provider adapters
│   ├── channel/      # Provider-specific adapters
│   │   ├── minimax/    # MiniMax adapter
│   │   ├── deepseek/   # DeepSeek adapter
│   │   ├── openai/     # OpenAI adapter
│   │   ├── claude/     # Claude adapter
│   │   ├── aws/        # AWS Bedrock
│   │   ├── gemini/     # Google Gemini
│   │   └── ... (40+ more)
│   ├── relay_adaptor.go   # Main relay struct
│   ├── api_request.go    # HTTP to upstream
│   └── common_handler/   # Shared relay logic
├── middleware/       # Gin middleware chain
│   ├── auth.go          # JWT/auth validation
│   ├── rate-limit.go    # Token-bucket rate limiting
│   ├── distributor.go   # Channel distribution/selection
│   ├── cache.go         # Response caching
│   └── cors.go
├── router/         # Gin route registration
│   ├── api.go           # /api/* routes
│   ├── relay.go         # /v1/* relay routes
│   └── dashboard.go     # Admin UI routes
├── setting/         # Configuration management
│   ├── ratio_setting/  # Model pricing ratios
│   ├── rate_limit.go   # Rate limit config
│   └── system_setting/
├── pkg/
│   ├── billingexpr/    # Expression-based billing
│   ├── cachex/         # Extended caching utilities
│   └── ionet/          # Network utilities
├── web/             # Frontend React apps
│   ├── default/        # React 19 + Rsbuild + Radix UI + Tailwind
│   └── classic/       # React 18 + Vite + Semi Design
├── integration/
│   ├── middleware/    # Python middleware (model enrichment)
│   │   └── model_detailed_fastapi.py   # FastAPI proxy
│   └── bruno-tests/  # API test suite
├── agent-harness/   # CLI tool (Click) for agent interaction
└── scripts/         # Operational scripts
```

## 3. Request Flow

### 3.1 Chat Completions (OpenAI format)

```
POST /v1/chat/completions
    │
    ▼
[Middleware Chain]
    ├─► auth.go          — validate JWT, extract user
    ├─► rate-limit.go    — check RPM/TPM quota
    ├─► distributor.go   — select channel by model ability
    │
    ▼
[relay.go controller]
    ├─► Parse RelayChatCompletionRequest
    ├─► Route to provider-specific adapter (minimax/deepseek/etc)
    │
    ▼
[relay/channel/minimax/]
    ├─► Convert OpenAI format → MiniMax native format
    ├─► api_request.go — HTTP POST to upstream
    │
    ▼
[Upstream: api.minimax.io /v1/text/chatcompletion_v2]
    │
    ▼
[Response]
    ├─► Convert MiniMax response → OpenAI format
    ├─► Inject metadata
    │
    ▼
client
```

### 3.2 Messages (Anthropic format)

```
POST /v1/messages
    │
    ▼
[relay.go] → claude_handler.go
    │
    ▼
[relay/channel/minimax/]
    ├─► Convert Claude format → MiniMax /anthropic/v1/messages
    ├─► api_request.go — HTTP POST
    │
    ▼
[Upstream: api.minimax.io /anthropic/v1/messages]
```

### 3.3 GET /v1/models (Enriched)

```
GET /v1/models
    │
    ▼
[model-detailed Python middleware :3300]
    ├─► Forward to new-api :3001
    ├─► Intercept response
    ├─► Enrich each model with:
    │     • context_length
    │     • max_output_tokens
    │     • pricing (prompt/completion/cache)
    │
    ▼
client (enriched response)
```

## 4. Database Schema (PostgreSQL)

### Key Tables

| Tabela | Função |
|--------|--------|
| `channels` | Provider configurations (base_url, key, abilities) |
| `abilities` | Channel ↔ Model mapping (which channel handles which model) |
| `tokens` | User API tokens |
| `users` | User accounts |
| `model_options` | Model-specific settings |
| `options` | General settings (api_info stored here as JSON) |
| `channel_group` | Channel grouping |
| `channel_affinity` | Affinity rules for channel selection |

### Channels Table

```
id | name                  | group | base_url                | key (encrypted)
---+-----------------------+-------+-------------------------+----------------
3  | MiniMax - Anthropic   |       | https://api.minimax.io  | ***
4  | MiniMax-Highspeed      |       | https://api.minimax.io  | ***
1  | MiniMax - Token Plan  |       | https://api.minimax.io  | ***
2  | DeepSeek API          |       | https://api.deepseek.com| ***
```

### Distribution Logic (distributor.go)

```
request.model
    │
    ▼
lookup abilities table WHERE model = request.model
    │
    ▼
channels matching (group, model) → channel_id
    │
    ▼
channels table WHERE id = channel_id → base_url + model_mapping
```

## 5. Middleware Python (model-detailed)

**File:** `integration/middleware/model_detailed_fastapi.py`

Enriches `/v1/models` responses with:

```python
Model metadata fields added:
- context_length       # max context window
- max_output_tokens   # from top_provider.max_completion_tokens
- pricing             # prompt/completion prices from DB or defaults
```

Request flow:
```
Client → :3300 (middleware) → :3001 (new-api) → response → middleware enriches → Client
```

## 6. Docker Compose Architecture

```yaml
Services:
  new-api:
    image: ghcr.io/giovannimnz/router-ai-atius:local
    ports: ["3301:3000"]
    environment:
      SQL_DSN: postgres://admin:***@db-newapi:5432/newapi?sslmode=disable
    networks: [newapi-internal, atius-shared]
    cpu: 0.5

  model-detailed:
    build: ./integration/middleware (Dockerfile.fastapi)
    ports: ["3300:3001"]
    environment:
      NEWAPI_BACKEND_URL: http://new-api:3000
    networks: [newapi-internal, atius-shared]
    cpu: 0.1

  db-newapi:
    image: postgres:15-alpine
    ports: ["8746:5432"]
    networks: [newapi-internal]
    cpu: 0.5

Networks:
  atius-shared:    192.168.0.0/20
  newapi-internal: 172.20.0.0/16
```

## 7. Relay Adapters (relay/channel/)

Each adapter implements the provider's specific request/response format:

| Adapter | Upstream Endpoint | Format |
|---------|------------------|--------|
| `minimax/` | `/v1/text/chatcompletion_v2`, `/anthropic/v1/messages` | MiniMax native |
| `deepseek/` | `/chat/completions` | OpenAI compatible |
| `openai/` | `/chat/completions` | OpenAI |
| `claude/` | `/v1/messages` | Anthropic |
| `gemini/` | `/v1beta/models/...:generateContent` | Google AI |
| `aws/` | Bedrock endpoints | AWS sigv4 |
| `ollama/` | `/api/chat` | Ollama native |

## 8. Rate Limiting

Token-bucket algorithm per user/token:

- RPM (requests per minute)
- TPM (tokens per minute)

Stored in Redis for distributed rate limiting across instances.

## 9. Auth Flow

```
Request → auth.go middleware
    ├─► Extract Bearer token from Authorization header
    ├─► Validate JWT signature
    ├─► Check token exists in DB
    ├─► Extract user_id, quota info
    │
    ▼
Set context: user_id, token_id, quota
    │
    ▼
Next middleware / handler
```

## 10. Key Files Reference

| File | Purpose |
|------|---------|
| `relay/relay_adaptor.go` | Core relay struct, holds HTTP client + upstream config |
| `relay/api_request.go` | Generic HTTP request builder for upstream calls |
| `relay/channel/minimax/adaptor.go` | MiniMax-specific request/response conversion |
| `controller/relay.go` | Main dispatcher, routes to appropriate relay handler |
| `middleware/distributor.go` | Channel selection based on model abilities |
| `service/channel_select.go` | Channel selection logic |
| `model/main.go` | GORM connection setup, shared column helpers |

---

_Last updated: 2026-05-31_
