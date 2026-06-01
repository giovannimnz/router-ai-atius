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
│   ├── billingexpr/    # Expression-based billing (expr-lang)
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

PostgreSQL 15 em container `db-newapi`. Todas as tabelas no schema `public`. Conexão via `SQL_DSN=postgres://admin:<password>@db-newapi:5432/newapi?sslmode=disable`.

### 4.1 Diagrama de Entidade-Relacionamento (核心 tabelas)

```
users ─────────────────────────────────────────────────────────────┐
  │                                                                 │
  │ 1:N                              1:N                           │
  ▼                                                                 │
tokens ──────────┐                                                   │
  │              │                                                   │
  │   N:1        │                                                   │
  │              │                                                   │
  └────────► quota_data                                             │
                  │                                                   │
                  │     N:1                                           │
                  └────────► usage_tracking                           │
                                                                    │
channels ◄───────────────────────────────────────────────────────────┘
  │                                                                  │
  │ 1:N                                                               │
  ▼                                                                     │
abilities ─────────────────────────────────────────────────────────────
  │
  │ N:1 (channel_id → channels.id)
  │
  ▼
channels (resolve ability → channel)
```

### 4.2 Tabelas — Especificação Completa

#### `users`

Contas de usuário para autenticação.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `id` | `bigint` | NOT NULL | `nextval('users_id_seq')` | PK |
| `username` | `text` | | | |
| `password` | `text` | NOT NULL | | bcrypt hash |
| `display_name` | `text` | | | |
| `role` | `bigint` | | `1` | 1=普通用户, 2+=admin |
| `status` | `bigint` | | `1` | |
| `email` | `text` | | | |
| `github_id` | `text` | | | OAuth binding |
| `access_token` | `text` | | | (unique index) |
| `quota` | `bigint` | | `0` | |
| `aff_code` | `varchar(32)` | | | affiliate code |
| `aff_count` | `bigint` | | `0` | |
| `aff_quota` | `bigint` | | `0` | |
| `aff_history` | `bigint` | | `0` | |
| `inviter_id` | `bigint` | | | quem indicou |
| `stripe_customer` | `varchar(64)` | | | |
| `created_at` | `bigint` | | | unix timestamp |
| `last_login_at` | `bigint` | | `0` | unix timestamp |
| `deleted_at` | `timestamptz` | | | soft delete |

**Indexes:** `users_pkey` (id), `idx_users_access_token` (access_token UNIQUE), `idx_users_email`, `idx_users_aff_code` (aff_code UNIQUE), `idx_users_deleted_at`, `idx_users_discord_id`, `idx_users_display_name`

---

#### `tokens`

Tokens de API por usuário. Cada token tem `key` (the actual API key string) e `remain_quota`.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `id` | `bigint` | NOT NULL | `nextval('tokens_id_seq')` | PK |
| `user_id` | `bigint` | | | FK → users.id |
| `key` | `varchar(128)` | | | API key string (unique index) |
| `status` | `bigint` | | `1` | 1=active |
| `name` | `text` | | | friendly name |
| `created_time` | `bigint` | | | unix timestamp |
| `accessed_time` | `bigint` | | | |
| `expired_time` | `bigint` | | `-1` | unix timestamp, -1=never |
| `remain_quota` | `bigint` | | `0` | quota remaining |
| `unlimited_quota` | `boolean` | | | bypass quota check |
| `model_limits_enabled` | `boolean` | | | per-model quota limits |
| `model_limits` | `text` | | | JSON: per-model quota |
| `allow_ips` | `text` | | `''` | IP whitelist |
| `used_quota` | `bigint` | | `0` | quota consumed |
| `group` | `text` | | `''` | token group |
| `cross_group_retry` | `boolean` | | | retry different group on fail |
| `deleted_at` | `timestamptz` | | | soft delete |

**Indexes:** `tokens_pkey` (id), `idx_tokens_key` (key UNIQUE), `idx_tokens_user_id`, `idx_tokens_deleted_at`

---

#### `channels`

Configurações de provedores upstream. Cada channel representa uma conta/credencial de provedor.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `id` | `bigint` | NOT NULL | `nextval('channels_id_seq')` | PK |
| `type` | `bigint` | | `0` | channel type |
| `key` | `text` | NOT NULL | | API key (encrypted) |
| `open_ai_organization` | `text` | | | OpenAI org ID |
| `test_model` | `text` | | | test model name |
| `status` | `bigint` | | `1` | 1=active |
| `name` | `text` | | | display name |
| `base_url` | `text` | | | upstream base URL |
| `group` | `varchar(64)` | | `'default'` | channel group |
| `used_quota` | `bigint` | | `0` | total quota consumed |
| `model_mapping` | `text` | | | JSON: model alias → canonical |
| `status_code_mapping` | `varchar(1024)` | | `''` | |
| `priority` | `bigint` | | `0` | channel priority |
| `auto_ban` | `bigint` | | `1` | auto-ban on error |
| `other_info` | `text` | | | |
| `tag` | `text` | | | |
| `setting` | `text` | | | |
| `param_override` | `text` | | | override request params |
| `header_override` | `text` | | | override request headers |
| `remark` | `varchar(255)` | | | |
| `channel_info` | `json` | | | extra channel data |
| `settings` | `text` | | | |

**Channels ativos no DB:**

| ID | Nome | base_url |
|----|------|----------|
| 1 | MiniMax - Token Plan | `https://api.minimax.io` |
| 2 | DeepSeek API | `https://api.deepseek.com` |
| 3 | MiniMax - Anthropic Compatible | `https://api.minimax.io` |
| 4 | MiniMax-Highspeed - Anthropic Compatible | `https://api.minimax.io` |

**Indexes:** `channels_pkey` (id), `idx_channels_name`, `idx_channels_tag`

---

#### `abilities`

Mapeamento many-to-many entre channels e modelos. Define quais modelos cada channel pode atender.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `group` | `varchar(64)` | NOT NULL | | channel group (PK composite) |
| `model` | `varchar(255)` | NOT NULL | | model name (PK composite) |
| `channel_id` | `bigint` | NOT NULL | | FK → channels.id (PK composite) |
| `enabled` | `boolean` | | | is ability active |
| `priority` | `bigint` | | `0` | higher = preferred |
| `weight` | `bigint` | | `0` | load balancing weight |
| `tag` | `text` | | | |

**PK:** `(group, model, channel_id)` — unique constraint

**Abilities ativas no DB:**

| channel_id | model | group | enabled | priority |
|------------|-------|-------|---------|---------|
| 1 | MiniMax-M2.7 | default | true | 0 |
| 1 | MiniMax-M2.5-highspeed | default | true | 0 |
| 1 | MiniMax-M2.5 | default | true | 0 |
| 1 | MiniMax-M2.7-highspeed | default | true | 0 |
| 2 | deepseek-v4-pro | default | true | 0 |
| 2 | deepseek-v4-flash | default | true | 0 |
| 3 | MiniMax-M2.5 | default | true | 0 |
| 3 | MiniMax-M2.7 | default | true | 0 |
| 4 | MiniMax-M2.7-highspeed | default | true | 0 |
| 4 | MiniMax-M2.5-highspeed | default | true | 0 |

**Indexes:** `abilities_pkey` ((group, model, channel_id)), `idx_abilities_channel_id`, `idx_abilities_priority`, `idx_abilities_tag`, `idx_abilities_weight`

---

#### `models`

Catálogo de modelos conhecidos. Sincronizado com provedores oficiais.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `id` | `bigint` | NOT NULL | `nextval('models_id_seq')` | PK |
| `model_name` | `varchar(128)` | NOT NULL | | (unique with deleted_at) |
| `description` | `text` | | | |
| `icon` | `varchar(128)` | | | |
| `tags` | `varchar(255)` | | | |
| `vendor_id` | `bigint` | | | |
| `endpoints` | `text` | | | |
| `status` | `bigint` | | `1` | |
| `sync_official` | `bigint` | | `1` | sync from provider |
| `created_time` | `bigint` | | | |
| `updated_time` | `bigint` | | | |
| `deleted_at` | `timestamptz` | | | soft delete |
| `name_rule` | `bigint` | | `0` | |

**Indexes:** `models_pkey` (id), `uk_model_name_delete_at` (model_name, deleted_at UNIQUE), `idx_models_deleted_at`, `idx_models_vendor_id`

---

#### `quota_data`

Tracking de quota por (usuário, modelo). Atualizado em tempo real a cada requisição.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `id` | `bigint` | NOT NULL | `nextval('quota_data_id_seq')` | PK |
| `user_id` | `bigint` | | | FK → users.id |
| `username` | `varchar(64)` | | `''` | denormalizado |
| `model_name` | `varchar(64)` | | `''` | |
| `created_at` | `bigint` | | | unix timestamp |
| `token_used` | `bigint` | | `0` | tokens consumidos |
| `count` | `bigint` | | `0` | número de requisições |
| `quota` | `bigint` | | `0` | quota alocada |

**Indexes:** `quota_data_pkey` (id), `idx_qdt_created_at`, `idx_qdt_model_user_name` (model_name, username), `idx_quota_data_user_id`

---

#### `usage_tracking`

Log histórico de consumo por período. Usado para relatórios e análise de custos.

| Coluna | Tipo | Nullable | Default | Descrição |
|--------|------|----------|---------|-----------|
| `id` | `bigint` | NOT NULL | `nextval('usage_tracking_id_seq')` | PK |
| `collected_at` | `bigint` | NOT NULL | | timestamp coleta |
| `period_start` | `bigint` | NOT NULL | | início do período |
| `period_end` | `bigint` | NOT NULL | | fim do período |
| `model_name` | `varchar(64)` | NOT NULL | | |
| `channel_id` | `bigint` | | | |
| `channel_name` | `varchar(128)` | | | |
| `token_name` | `varchar(128)` | | | token key usado |
| `requests` | `integer` | | `0` | |
| `prompt_tokens` | `bigint` | | `0` | |
| `completion_tokens` | `bigint` | | `0` | |
| `total_tokens` | `bigint` | | `0` | |
| `estimated_costusd` | `numeric(12,6)` | | `0` | **custo em USD** |
| `avg_tokens_per_request` | `numeric(12,2)` | | `0` | |
| `created_at` | `timestamptz` | | `now()` | |

**Indexes:** `usage_tracking_pkey` (id), `idx_tracking_collected_at`, `idx_tracking_model`, `idx_tracking_period`

**Dados atuais no DB:**

| model_name | channel_id | prompt_tokens | completion_tokens | estimated_costusd |
|------------|------------|--------------|-------------------|-------------------|
| MiniMax-M2.7-highspeed | 1 | 1,912,603 | 35,877 | $0.616833 |

---

#### `options`

Chave-valor genérico para configuração. Armazenado como `text`.

| Coluna | Tipo | Nullable | Default |
|--------|------|----------|---------|
| `key` | `text` | NOT NULL | |
| `value` | `text` | | |

**PK:** `(key)`

**Keys relevantes:**

| key | Conteúdo | Descrição |
|-----|----------|-----------|
| `InputPrice` | JSON com preços input por modelo | Preços MiniMax: M2.7 $0.30, M2.5 $0.30, deepseek-v4-flash $0.14, deepseek-v4-pro $0.435 |
| `OutputPrice` | JSON com preços output por modelo | M2.7 $1.20, M2.5 $1.20, deepseek-v4-flash $0.28, deepseek-v4-pro $0.87 |
| `ModelRatio` | JSON com ratios de conversão | MiniMax-M2.x = 0.15, deepseek-v4-flash = 0.07, deepseek-v4-pro = 0.2175 |
| `console_setting.api_info` | JSON array com info da API | URLs dos endpoints expostas no `/api/status` |
| `operation_settings` | JSON | `{"self_use": true, "price_configured": true}` |
| `performance_setting.monitor_enabled` | boolean | |
| `group.default.model.pricing.MiniMax-M2.1*` | JSON | `{"prompt_price": 0.0000003, "completion_price": 0.0000012}` |

---

#### Demais tabelas

| Tabela | Função |
|--------|--------|
| `checkins` | Daily check-in rewards |
| `custom_oauth_providers` | OAuth custom providers config |
| `logs` | Operation logs |
| `midjourneys` | Midjourney tasks |
| `passkey_credentials` | WebAuthn passkeys |
| `prefill_groups` | Prefill group configs |
| `redemptions` | Coupon/code redemptions |
| `setups` | Initial setup state |
| `subscription_orders` | Subscription orders |
| `subscription_plans` | Plan definitions |
| `subscription_pre_consume_records` | Pre-consumption for subscriptions |
| `tasks` | Async task tracking |
| `top_ups` | Top-up transactions |
| `two_fa_backup_codes` | 2FA backup codes |
| `two_fas` | 2FA settings |
| `user_oauth_bindings` | OAuth user bindings |
| `user_subscriptions` | Active subscriptions |
| `vendors` | Vendor/provider definitions |

---

## 5. Sistema de Billing

### 5.1 Arquitetura Geral

O billing é baseado em **expressões** (`pkg/billingexpr/`) avaliadas com `expr-lang`. Cada modelo tem uma expressão que define seu custo real em USD por requisição.

```
Requisição 完成
    │
    ▼
service/billing_session.go
    │
    ├─► Extrai prompt_tokens, completion_tokens da resposta upstream
    ├─► Carrega expressão do modelo (de options / model config)
    ├─► Eval(expr, { p: prompt_tokens, c: completion_tokens })
    ├─► Calcula estimated_costUSD
    │
    ▼
usage_tracking (insere registro)
    │
    ▼
tokens.used_quota (incrementa)
    │
    ▼
quota_data (incrementa token_used)
```

### 5.2 Expression Language (billingexpr)

**File:** `pkg/billingexpr/expr.md` — especificação completa

**Biblioteca:** `expr-lang/expr` (Go)

**Variáveis disponíveis:**

| Variável | Tipo | Descrição |
|----------|------|-----------|
| `p` | int64 | Prompt tokens (input). **Auto-exclude** cache/image/audio se incluídos via `cr`/`cc`/`img` |
| `c` | int64 | Completion tokens (output). **Auto-exclude** `img_o`/`ao` se incluídos |
| `cr` | int64 | Cache read (prompt cache hit) tokens |
| `cc` | int64 | Cache create tokens (5min TTL / standard) |
| `cc1h` | int64 | Cache create tokens (1h TTL, Claude-only) |
| `img` | int64 | Image input tokens |
| `img_o` | int64 | Image output tokens |
| `ai` | int64 | Audio input tokens |
| `ao` | int64 | Audio output tokens |
| `len` | int64 | Total input context length (para condições) |

**Exemplo de expressão:**

```
v1: p * 0.30 / 1_000_000 + c * 1.20 / 1_000_000
```

**Exemplo com cache (MiniMax prompt caching):**

```
v1: (p - cr - cc) * 0.30 / 1_000_000 + cr * 0.06 / 1_000_000 + cc * 0.375 / 1_000_000 + c * 1.20 / 1_000_000
```

Onde:
- `0.30` = $0.30/M input (preço base)
- `0.06` = $0.06/M cache read (0.2× input)
- `0.375` = $0.375/M cache write (1.25× input)
- `1.20` = $1.20/M output

### 5.3 Model Ratio vs Expression-based Billing

O DB guarda dois sistemas de pricing side-by-side:

**1. ModelRatio (options key)**
- Ratio relativo ao custo base (o1 = 1.0)
- Usado pelo sistema de ratio clássico
- Exemplo: `MiniMax-M2.7` → 0.15 (15% do custo do o1)

**2. InputPrice/OutputPrice (options keys)**
- Preços absolutos em USD por 1M tokens
- Usado como input para expressões de billing

**Fluxo de cálculo de custo:**

```
InputPrice[$/1M] + OutputPrice[$/1M]
    │
    ▼
billing_expr.eval(prompt_tokens, completion_tokens)
    │
    ▼
estimated_costUSD = (prompt_tokens/1M * input_price) + (completion_tokens/1M * output_price)
```

### 5.4 Preços MiniMax (do DB)

**InputPrice (options key):**

| Modelo | Preço $/1M |
|--------|-----------|
| MiniMax-M2.7 | 0.30 |
| MiniMax-M2.5 | 0.30 |
| MiniMax-M2.7-hs | 0.30 |
| MiniMax-M2.5-hs | 0.30 |
| deepseek-v4-flash | 0.14 |
| deepseek-v4-pro | 0.435 |

**OutputPrice (options key):**

| Modelo | Preço $/1M |
|--------|-----------|
| MiniMax-M2.7 | 1.20 |
| MiniMax-M2.5 | 1.20 |
| deepseek-v4-flash | 0.28 |
| deepseek-v4-pro | 0.87 |

**Cache Prices (MiniMax prompt caching):**

| Tipo | Preço relativo |
|------|---------------|
| Cache read | 0.2× input price |
| Cache write | 1.25× input price |

### 5.5 Exemplo de Cálculo

Requisição com 1M prompt tokens + 100K completion tokens em `MiniMax-M2.7`:

```
Custo input     = 1,000,000 × $0.30 / 1,000,000 = $0.30
Custo output    = 100,000 × $1.20 / 1,000,000  = $0.12
Custo total     = $0.42
```

Requisição com 500K prompt tokens + 50K cache hit + 50K new + 20K output (com cache):

```
Custo prompt novo = 450,000 × $0.30 / 1M = $0.135
Custo cache read  = 50,000 × $0.06 / 1M  = $0.003
Custo cache write = 50,000 × $0.375 / 1M = $0.01875
Custo output      = 20,000 × $1.20 / 1M  = $0.024
Custo total       = $0.18075
```

---

## 6. Middleware Python (model-detailed)

**File:** `integration/middleware/model_detailed_fastapi.py`

Enriches `/v1/models` responses com:

```python
# Campos adicionados por modelo:
- context_length       # max context window (e.g. 245760)
- max_output_tokens   # from top_provider.max_completion_tokens (e.g. 50000)
- pricing             # { prompt: "0.30", completion: "1.20", prompt_cache_hit: "0.06" }
```

Request flow:

```
Client → :3300 (middleware) → :3001 (new-api) → response → middleware enriches → Client
```

O middleware faz proxy puro de todas as outras requisições — só `/v1/models` é interceptado e enriquecido.

---

## 7. Docker Compose Architecture

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

---

## 8. Relay Adapters (relay/channel/)

Cada adapter implementa o formato específico de request/response do provedor:

| Adapter | Upstream Endpoint | Format |
|---------|------------------|--------|
| `minimax/` | `/v1/text/chatcompletion_v2`, `/anthropic/v1/messages` | MiniMax native |
| `deepseek/` | `/chat/completions` | OpenAI compatible |
| `openai/` | `/chat/completions` | OpenAI |
| `claude/` | `/v1/messages` | Anthropic |
| `gemini/` | `/v1beta/models/...:generateContent` | Google AI |
| `aws/` | Bedrock endpoints | AWS sigv4 |
| `ollama/` | `/api/chat` | Ollama native |

---

## 9. Rate Limiting

Token-bucket por usuário/token:
- **RPM** (requests per minute)
- **TPM** (tokens per minute)

Armazenado em Redis para rate limiting distribuído entre instâncias.

---

## 10. Auth Flow

```
Request → auth.go middleware
    ├─► Extract Bearer token from Authorization header
    ├─► Validate JWT signature
    ├─► Check token exists in DB (tokens.key)
    ├─► Extract user_id, quota info
    │
    ▼
Set context: user_id, token_id, quota
    │
    ▼
Next middleware / handler
```

---

## 11. Bug Fix: CJK Character Pollution (MiniMax)

### 11.1 Problema

MiniMax-M2.7-hs ocasionalmente emitia caracteres CJK (chineses) em contexto não-CJK (português/inglês) devido a quirks do tokenizador BBPE com temperature sampling.

**Commit:** `b5b1ac594 feat(minimax): add CJK strip filter — defense-in-depth`

### 11.2 Arquitetura de Correção (2 camadas defense-in-depth)

```
Upstream MiniMax response
    │
    ├─ Layer 1: Go Router (StripCJK no relay — ativa via DB)
    │       └─► common.StripCJK() → client
    │
    └─ Layer 2: Python Middleware (sempre ativa)
            └─► strip_cjk_from_text() → client
```

### 11.3 Layer 1 — Go Router

**Arquivos:**

| Arquivo | Mudança |
|---------|---------|
| `common/str.go` | `cjkRegex` (Unicode ranges) + `StripCJK(s string) string` |
| `dto/channel_settings.go` | `StripCJK bool` em `ChannelSettings` |
| `relay/channel/openai/relay-openai.go` | `StripCJK` em streaming e non-streaming |
| `data/migrations/migration_v0.4-strip-cjk-minimax.sql` | SQL migration para ativar `strip_cjk` |

**Unicode ranges (`common/str.go`):**

```go
cjkRegex = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]`)
// Ranges cobertos:
// \x{4e00}-\x{9fff}   CJK Unified Ideographs (23,870 chars — maioria)
// \x{3400}-\x{4dbf}   CJK Extension A
// \x{3000}-\x{303f}   CJK Symbols/Punctuation
// \x{ff00}-\x{ffef}   Halfwidth/Fullwidth Forms
```

**Cobertura Layer 1:**

| Endpoint | Tipo | Função |
|----------|------|--------|
| `POST /v1/chat/completions` | Non-stream | `OpenaiHandler` → `StripCJK` applied ao content string |
| `POST /v1/chat/completions` | Streaming | `sendStreamData` → `StripCJK` on raw string before delta parse |

**Limitações Layer 1:**
- Requer `strip_cjk: true` em `ChannelSettings` no DB (campo `settings` da tabela `channels`)
- Só implementado em `relay/channel/openai/relay-openai.go` (formato OpenAI — não cobre Anthropic `/v1/messages`)
- **Status atual:** `strip_cjk` NÃO está habilitado no canal type=35 (MiniMax) — Layer 1 não está ativa

### 11.4 Layer 2 — Python Middleware (Defense-in-Depth)

**Arquivo:** `integration/middleware/model_detailed_fastapi.py`

**CJK regex:**

```python
CJK_PATTERN = re.compile(r"[\u4e00-\u9fff\u3400-\u4dbf\u3000-\u303f\uff00-\uffef]")
```

**Funções de limpeza:**

| Função | O que faz |
|--------|-----------|
| `strip_cjk_from_text(text)` | Regex sub CJK characters |
| `strip_thinking_from_text(text)` | Remove blocos `<think>...</think>` |
| `clean_code_fences(text)` | Strip markdown code fences |
| `strip_thinking_blocks(body)` | Processa JSON body, aplica todas as limpezas |

**Três paths de processamento:**

| Path | Formato | Pipeline |
|------|---------|---------|
| Anthropic non-stream | `content[].text` | `strip_thinking_from_text` → `clean_code_fences` → `strip_cjk_from_text` |
| OpenAI non-stream | `choices[].message.content` | `strip_thinking_from_text` → `clean_code_fences` → `strip_cjk_from_text` |
| OpenAI streaming | SSE raw bytes | CJK strip na string raw ANTES de parsear delta |

**Fix crítico do commit b5b1ac594:** O path OpenAI Agorax faz strip CJK MESMO quando não há think block (antes, só fazia se encontrasse `<think>`).

### 11.5 Status Atual

| Layer | Ativo? | Detalhe |
|-------|--------|---------|
| Layer 1 (Go Router) | **NÃO** | `strip_cjk` não está em `settings` do canal type=35 |
| Layer 2 (Python) | **SIM** | Sempre ativa no middleware FastAPI |

**Para ativar Layer 1:**

```sql
-- Verificar settings atual
SELECT id, name, type, settings FROM channels WHERE type = 35;

-- Aplicar strip_cjk
UPDATE channels
SET settings = (
    CASE
        WHEN settings::jsonb ? 'strip_cjk' THEN settings
        ELSE (settings::jsonb || '{"strip_cjk": true}')::text
    END
)
WHERE type = 35;
```

---

## 12. Key Files Reference

| File | Purpose |
|------|---------|
| `relay/relay_adaptor.go` | Core relay struct, HTTP client + upstream config |
| `relay/api_request.go` | Generic HTTP request builder for upstream |
| `relay/channel/minimax/adaptor.go` | MiniMax request/response conversion |
| `relay/channel/openai/relay-openai.go` | OpenAI relay + StripCJK (Layer 1 CJK fix) |
| `controller/relay.go` | Main dispatcher, routes to relay handler |
| `middleware/distributor.go` | Channel selection by model abilities |
| `service/channel_select.go` | Channel selection logic |
| `service/billing_session.go` | Billing calculation post-request |
| `service/quota.go` | Quota management and enforcement |
| `model/main.go` | GORM connection setup, shared column helpers |
| `common/str.go` | String utils incl. `StripCJK()` |
| `dto/channel_settings.go` | `ChannelSettings` struct incl. `StripCJK` |
| `pkg/billingexpr/expr.md` | Billing expression language spec |
| `integration/middleware/model_detailed_fastapi.py` | FastAPI proxy + Layer 2 CJK strip |

---

_Last updated: 2026-05-31_
