# ARCHITECTURE.md - System Architecture

## Visao Geral

O sistema e um **gateway LLM centralizado** baseado em NewAPI (fork do OneAPI mantido por QuantumNous), que roteia requisicoes de multiplos consumidores para diversos provedores de modelos de linguagem. A arquitetura segue um padrao **client-proxy-provider** com persistencia em PostgreSQL, middleware de enriquecimento de metadados e integracao de busca em tempo real.

**URL publica:** https://router.atius.com.br
**Diretorio raiz:** `/home/ubuntu/docker/ai-apps/new-api/`

---

## Diagrama de Arquitetura

```
                              ECOSISTEMA ATIUS
┌────────────────────────────────────────────────────────────────────────────────┐
│                                                                                │
│  CONSUMIDORES                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐  ┌─────────────────────┐    │
│  │ Open-WebUI │  │ OpenClaw   │  │ GSD-2        │  │ Search-Engine       │    │
│  │ (Chat UI)  │  │ (AI Agent) │  │ (Planner)    │  │ (Middleware GSD)    │    │
│  │ :8080      │  │            │  │              │  │ :4000               │    │
│  └──────┬─────┘  └──────┬─────┘  └──────┬───────┘  └──────────┬──────────┘    │
│         │               │                │                     │               │
│         └───────────────┼────────────────┼─────────────────────┘               │
│                         │ Bearer Token (sk-...)                                │
│                         ▼                                                      │
│  ┌──────────────────────────────────────────────────────────────────────┐     │
│  │                    CAMADA DE PROXY / GATEWAY                          │     │
│  │                                                                      │     │
│  │  ┌────────────────────────────────────────────────────────────┐      │     │
│  │  │  MODEL-DETAILED (Middleware de Enriquecimento) :3300        │      │     │
│  │  │  - Intercepta GET /v1/models                                │      │     │
│  │  │  - Injeta metadata DeepSeek (context, pricing, features)    │      │     │
│  │  │  - Header X-Model-Metadata-Enriched: true                   │      │     │
│  │  │  - Proxy transparente para demais endpoints                 │      │     │
│  │  │  Python 3.12-slim | python3 model_detailed.py              │      │     │
│  │  └───────────────────────────┬────────────────────────────────┘      │     │
│  │                              │ proxy interno                         │     │
│  │                              ▼                                       │     │
│  │  ┌────────────────────────────────────────────────────────────┐      │     │
│  │  │  NEWAPI GATEWAY (porta interna 3000, exposta via middleware)│      │     │
│  │  │  ┌──────────────────────────────────────────────────────┐  │      │     │
│  │  │  │  API Layer (OpenAI-compatible)                        │  │      │     │
│  │  │  │  GET  /v1/models          (lista modelos)             │  │      │     │
│  │  │  │  POST /v1/chat/completions (chat)                     │  │      │     │
│  │  │  │  GET  /v1/model/info       (LiteLLM format, opcional) │  │      │     │
│  │  │  │  GET  /api/status          (status rapido)            │  │      │     │
│  │  │  │  GET  /health              (healthcheck)              │  │      │     │
│  │  │  └──────────────────────────────────────────────────────┘  │      │     │
│  │  │  ┌──────────────────────────────────────────────────────┐  │      │     │
│  │  │  │  Admin UI                                             │  │      │     │
│  │  │  │  POST /api/user/login       (autenticacao)            │  │      │     │
│  │  │  │  GET  /api/channel/         (listar channels)         │  │      │     │
│  │  │  │  PUT  /api/channel/         (atualizar channels)      │  │      │     │
│  │  │  │  GET  /api/option/          (listar opcoes)           │  │      │     │
│  │  │  │  PUT  /api/option/          (atualizar opcoes)        │  │      │     │
│  │  │  └──────────────────────────────────────────────────────┘  │      │     │
│  │  │  ┌──────────────────────────────────────────────────────┐  │      │     │
│  │  │  │  Channel Router (modelo → provider mapping via DB)    │  │      │     │
│  │  │  └──────────────────────────────────────────────────────┘  │      │     │
│  │  │  Imagem: calciumion/new-api:latest | CPU limit: 0.5        │      │     │
│  │  └──────────────────────────────────┬─────────────────────────┘      │     │
│  │                                     │                                 │     │
│  └─────────────────────────────────────┼─────────────────────────────────┘     │
│                                        │                                        │
│                    ┌───────────────────┼───────────────────┐                   │
│                    ▼                   ▼                   ▼                   │
│  PROVIDERS UPSTREAM                                                            │
│  ┌────────────────────┐ ┌──────────────────┐ ┌──────────────────────┐         │
│  │ DeepSeek           │ │ Qwen/DashScope   │ │ Moonshot/Kimi        │         │
│  │ api.deepseek.com   │ │ dashscope-intl   │ │ (via iFlow)          │         │
│  │ 3 chaves rotativas │ │ aliyuncs.com     │ │                      │         │
│  │ V3.2, R1           │ │ qwen3-* models   │ │ kimi-k2-0905         │         │
│  └────────────────────┘ └──────────────────┘ └──────────────────────┘         │
│  ┌────────────────────────────────────┐                                        │
│  │ MiniMax                            │                                        │
│  │ api.minimax.io (regiao global)     │                                        │
│  │ Token Plan (sk-cp-*)               │                                        │
│  │ MiniMax-M2.7, MiniMax-M2.5         │                                        │
│  └────────────────────────────────────┘                                        │
│                                                                                │
│  ┌──────────────────────────────────────────────────────────────────────┐     │
│  │  PERSISTENCIA                                                        │     │
│  │  PostgreSQL 15-alpine | db-newapi | Porta 8746:5432                  │     │
│  │  Tables: channels, tokens, options, users                            │     │
│  │  Volume: ./data/postgres_data:/var/lib/postgresql/data               │     │
│  │  Healthcheck: pg_isready a cada 10s, 10 retries                      │     │
│  └──────────────────────────────────────────────────────────────────────┘     │
│                                                                                │
│  ┌──────────────────────────────────────────────────────────────────────┐     │
│  │  INTEGRACOES                                                         │     │
│  │  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐      │     │
│  │  │ SearXNG       │  │ Whisper.cpp  │  │ Search-Engine        │      │     │
│  │  │ Meta-search   │  │ Speech-to-Txt│  │ FastAPI middleware   │      │     │
│  │  │ settings.yml  │  │ ggml-small   │  │ /v1/chat/completions │      │     │
│  │  └───────────────┘  └──────────────┘  │ proxy + grounding    │      │     │
│  │                                       └──────────────────────┘      │     │
│  └──────────────────────────────────────────────────────────────────────┘     │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

---

## Estrutura de Diretorios

```
/home/ubuntu/docker/ai-apps/new-api/
│
├── docker-compose.yml            # Compose principal (new-api, model-detailed, db-newapi)
├── .env                          # Variaveis de ambiente unificadas
├── start.sh                      # Inicializacao de servicos
├── management.sh                 # Menu interativo de gerenciamento (13 opcoes)
├── reload-newapi.sh              # Force-recreate new-api para aplicar mudancas de .env
├── disk-health.sh                # Monitoramento e limpeza segura de disco
├── recreate-all.sh               # Recriacao completa do stack (destrutivo)
├── backup-restore.sh             # Backup e restauracao interativa de dados
│
├── data/
│   ├── logs/                     # Logs rotativos (oneapi-YYYYMMDDHHMMSS.log)
│   ├── postgres_data/            # Volume persistente do PostgreSQL
│   └── one-api.db                # Legado SQLite (migrado para PostgreSQL)
│
├── integration/
│   ├── docker-compose.yml        # Compose alternativo (referencia/legacy)
│   ├── .env                      # Env file estendido com chaves de API e tokens
│   ├── GUIA_MUDANCA_CHAVES_API.md # Documentacao de rotacao de chaves
│   ├── Models_gsd.json           # Catalogo completo de modelos com metadados
│   │
│   ├── middleware/
│   │   └── model_detailed.py     # Middleware de enriquecimento de metadata DeepSeek
│   │
│   ├── scripts/
│   │   ├── normalize_models_real_only.py    # Normalizacao de channels (alias → real)
│   │   ├── sync_deepseak_channels.py        # Sync de 3 channels DeepSeek via API admin
│   │   ├── sync_iflow_channel_keys.py       # Sync de chaves iFlow por slot
│   │   ├── sync_openrouter_channels.py      # Sync de channels OpenRouter
│   │   ├── update_api_keys.sh               # Script interativo de atualizacao de chaves
│   │   ├── test_all_models.sh               # Teste completo de todos os modelos
│   │   ├── verify_stack.sh                  # Verificacao de stack (5 steps smoke test)
│   │   └── backup_integration_state.sh      # Backup de estado da integracao
│   │
│   ├── search-engine/
│   │   ├── Dockerfile            # Imagem customizada (Python 3.12 + libgomp1 + ffmpeg)
│   │   └── app/
│   │       ├── main.py           # FastAPI middleware: busca SearXNG + proxy + STT Whisper
│   │       └── requirements.txt  # fastapi, uvicorn, httpx, python-multipart
│   │
│   ├── searxng/
│   │   └── settings.yml          # Configuracao SearXNG (formato JSON habilitado)
│   │
│   ├── whisper.cpp/              # Checkout completo do whisper.cpp (build local)
│   │
│   └── backups/                  # Backups timestampados de estado da integracao
│       ├── 20260224-013956/
│       └── 20260224-014007/
│
└── .planning/                    # GSD planning artifacts
    ├── PROJECT.md                # Visao do projeto e requisitos ativos
    ├── ROADMAP.md                # Roadmap de fases do milestone v1.1
    ├── MILESTONES.md             # Historico de milestones
    ├── STATE.md                  # Estado atual do planning
    └── codebase/                 # Documentacao tecnica do codigo
        ├── ARCHITECTURE.md       # Este arquivo
        ├── STACK.md              # Stack tecnologico
        ├── STRUCTURE.md          # Estrutura do projeto
        ├── CONCERNS.md           # Preocupacoes tecnicas
        ├── CONVENTIONS.md        # Convencoes de codigo
        ├── INTEGRATIONS.md       # Integracoes externas
        └── TESTING.md            # Estrategia de testes
```

---

## Composicao Docker - Servicos

### 1. new-api (Gateway Principal)

| Atributo | Valor |
|---|---|
| **Imagem** | `calciumion/new-api:latest` |
| **Container** | `new-api` |
| **Porta interna** | 3000 (exposta, nao mapeada diretamente no compose principal) |
| **CPU Limit** | 0.5 cores |
| **Restart** | `always` |
| **Dependencia** | `db-newapi` (via `depends_on`) |
| **Redes** | `newapi-internal`, `atius-shared` |
| **Volume** | `./data:/data` (logs + dados) |

**Variaveis de ambiente:**
- `SQL_DSN=postgres://admin:password123@db-newapi:5432/newapi?sslmode=disable`
- `TZ=America/Recife`, `LANG=en_US.UTF-8`

**Responsabilidades:**
- Gateway LLM com API compativel OpenAI (`/v1/chat/completions`, `/v1/models`)
- Roteamento dinamico via channels (modelo → provider)
- Balanceamento de carga entre multiplas API keys do mesmo provider
- UI administrativa para gestao de channels, tokens e opcoes globais
- Rate limiting interno e gerenciamento de quotas

### 2. model-detailed (Middleware de Enriquecimento de Metadata)

| Atributo | Valor |
|---|---|
| **Imagem** | `python:3.12-slim` |
| **Container** | `model-detailed` |
| **Porta publica** | 3300 (host) → 3001 (container) |
| **Backend** | `http://new-api:3000` (via rede `newapi-internal`) |
| **CPU Limit** | 0.1 cores |
| **Restart** | `always` |
| **Dependencia** | `new-api` (via `depends_on`) |
| **Redes** | `newapi-internal`, `atius-shared` |
| **Volume** | `./integration/middleware:/app:ro` (montagem read-only) |
| **Comando** | `python3 model_detailed.py` |

**Variaveis de ambiente:**
- `MIDDLEWARE_PORT=3001`
- `NEWAPI_BACKEND_URL=http://new-api:3000`

**Responsabilidades:**
- **Intercepta** requisicoes `GET /v1/models` destinadas ao NewAPI
- **Enriquece** a resposta com metadata detalhada para modelos:
  - `deepseek-chat`: nome, context_length=131072, max_completion_tokens=8192, pricing (prompt=$0.28/1M, completion=$0.42/1M, cache_hit=$0.028/1M)
  - `deepseek-reasoner`: nome, context_length=131072, max_completion_tokens=65536, mesmo pricing
  - `MiniMax-M2.7`: nome, context_length=245760, max_completion_tokens=50000, pricing (prompt=$0.30/1M, completion=$1.20/1M, cache_hit=$0.06/1M, cache_miss=$0.375/1M)
  - `MiniMax-M2.5`: nome, context_length=245760, max_completion_tokens=50000, pricing (prompt=$0.30/1M, completion=$1.20/1M, cache_hit=$0.03/1M, cache_miss=$0.375/1M)
- **Proxy transparente** para todos os demais endpoints (POST /v1/chat/completions, PUT, DELETE, etc.)
- Adiciona header `X-Model-Metadata-Enriched: true` nas respostas enriquecidas
- Fallback para resposta original do NewAPI em caso de falha de parse JSON

**Fluxo de interceptacao:**
```
Cliente GET /v1/models:3300
  → model-detailed recebe na porta 3001
  → Proxy GET para http://new-api:3000/v1/models
  → Parse JSON da resposta upstream
  → Para cada modelo enriquecido (deepseek-chat, deepseek-reasoner, MiniMax-M2.7, MiniMax-M2.5):
      Substitui resposta padrao por versao enriquecida com:
      - name, context_length, top_provider.max_completion_tokens
      - pricing (prompt, completion, prompt_cache_hit, [prompt_cache_miss para MiniMax])
  → Para modelos nao-enriquecidos: passa inalterados
  → Retorna JSON enriquecido com header X-Model-Metadata-Enriched
```

**Implementacao tecnica:**
- Servidor HTTP puro Python (`http.server.HTTPServer` + `BaseHTTPRequestHandler`)
- Sem dependencias externas (stdlib apenas)
- `urllib.request` para proxy HTTP com timeout de 30s
- Log em stdout para visibilidade no Docker
- Handlers completos: `do_GET`, `do_POST`, `do_PUT`, `do_DELETE`

### 3. db-newapi (PostgreSQL)

| Atributo | Valor |
|---|---|
| **Imagem** | `postgres:15-alpine` |
| **Container** | `db-newapi` |
| **Porta** | 8746 (host) → 5432 (container) |
| **CPU Limit** | 0.5 cores |
| **Restart** | `always` |
| **Redes** | `newapi-internal`, `atius-shared` |
| **Volume** | `./data/postgres_data:/var/lib/postgresql/data` |

**Credenciais:**
- Usuario: `admin`
- Senha: `password123`
- Database: `newapi`

**Healthcheck:**
- Comando: `pg_isready -U admin -d newapi`
- Intervalo: 10s, Timeout: 5s, Retries: 10

**Responsabilidades:**
- Persistencia de channels (mapeamento modelo → provider URL + API key)
- Armazenamento de tokens de API e chaves de autenticacao
- Configuracoes globais (options como precificacao e ratios)
- Gerenciamento de usuarios da UI administrativa

---

## Redes Docker

| Rede | Tipo | Funcao |
|---|---|---|
| **newapi-internal** | Bridge (external) | Isolamento interno entre new-api, db-newapi e model-detailed |
| **atius-shared** | Bridge (external) | Compartilhamento com outros servicos do ecossistema Atius (Open-WebUI, OpenClaw, etc.) |

Os servicos NewAPI estao conectados a ambas as redes, permitindo:
- Comunicacao interna isolada entre os 3 servicos do stack
- Acesso externo de consumidores via `atius-shared`

---

## Integracao Search-Engine (Middleware FastAPI)

O Search-Engine e um middleware FastAPI independente (definido em `integration/search-engine/`) que fornece:

### Funcionalidades Principais

1. **Busca em Tempo Real via SearXNG:**
   - Intercepta `POST /v1/chat/completions`
   - Analisa intencao de busca no texto do usuario (padroes como "pesquise", "busque", meses sem ano, `#search`)
   - Executa busca via SearXNG com rotacao inteligente de engines (google, bing, brave, duckduckgo)
   - Injeta contexto de busca no prompt antes de enviar ao NewAPI

2. **Engine Rotator:**
   - Rate-limit por engine (ex: google=4/min, bing=10/min, brave=8/min)
   - Cooldown automatico em caso de falha HTTP
   - Selecao da melhor engine disponivel a cada requisicao

3. **Fila Global Round-Robin:**
   - `MODEL_QUEUE_LANES=auto` (calculado pelo numero de chaves IFLOW)
   - Controle de concorrencia global compartilhado entre todos os modelos
   - Locks asyncicos por lane

4. **Compatibilidade de Reasoning Stream:**
   - Para modelos como `deepseek-r1`, converte `reasoning_content` para `content`
   - Permite que clientes OpenAI-standard consumam respostas de reasoning

5. **Transcricao de Audio (Whisper.cpp):**
   - Endpoint de STT usando whisper.cpp compilado localmente
   - Modelo `ggml-small.bin`, idioma padrao `pt`

### Configuracao

| Variavel | Valor Padrao | Descricao |
|---|---|---|
| `SEARXNG_URL` | `http://searxng:8080/search` | URL do SearXNG |
| `SEARXNG_LANGUAGE` | `pt-BR` | Idioma das buscas |
| `SEARXNG_LOCATION` | `Recife, Pernambuco, Brasil` | Localizacao para buscas |
| `SEARCH_RESULT_LIMIT` | `5` | Maximo de resultados |
| `RATE_LIMIT_RETRIES` | `2` | Retentativas de rate limit |
| `RATE_LIMIT_BACKOFF_S` | `1.0` | Backoff em segundos |
| `ENABLE_REASONING_STREAM_COMPAT` | `true` | Habilita compatibilidade reasoning |
| `LITELLM_PROXY_CHAT_URL` | `http://new-api:3000/v1/chat/completions` | Backend NewAPI |

---

## Scripts de Integracao

### Sincronizacao de Channels

| Script | Funcao |
|---|---|
| `sync_deepseak_channels.py` | Cria/atualiza 3 channels DeepSeek (Key 1, 2, 3) com API keys do `.env` |
| `sync_iflow_channel_keys.py` | Atualiza chaves iFlow em channels existentes (regex por "Key N") |
| `sync_openrouter_channels.py` | Sincroniza models do OpenRouter via `Models_gsd.json` |
| `normalize_models_real_only.py` | Remove aliasings e model_mappings, mantendo apenas modelos reais |

### MiniMax Channel (Configuracao Manual via DB)

| Atributo | Valor |
|---|---|
| **Channel ID** | 42 |
| **Type** | 1 (OpenAI Compatible) |
| **Base URL** | `https://api.minimax.io` (sem `/v1`) |
| **Key Source** | `MINIMAX_API_KEY` no `~/.zshrc` (Token Plan) |
| **Modelos** | MiniMax-M2.7, MiniMax-M2.5 |
| **Abilities** | Grupo `default` + `distributor` |
| **Vendor** | MiniMax (id: 4) |
| **Testes** | `integration/bruno-tests/minimax/` (Bruno CLI) |

#### Pricing no NewAPI

| Config | Valor |
|---|---|
| **ModelRatio** | `{"MiniMax-M2.7": 0.15, "MiniMax-M2.5": 0.15}` |
| **CompletionRatio** | `{"MiniMax-M2.7": 4, "MiniMax-M2.5": 4}` |
| **Formula ModelRatio** | `input_price / 2` (mesma do DeepSeek: $0.28 → 0.14) |
| **Formula CompletionRatio** | `output_price / input_price` (1.2 / 0.3 = 4.0) |

#### Middleware Enrichment

| Campo | MiniMax-M2.7 | MiniMax-M2.5 |
|---|---|---|
| prompt | 0.0000003 | 0.0000003 |
| completion | 0.0000012 | 0.0000012 |
| prompt_cache_hit | 0.00000006 | 0.00000003 |
| prompt_cache_miss | 0.000000375 | 0.000000375 |

### Teste e Verificacao

| Script | Funcao |
|---|---|
| `test_all_models.sh` | Testa 6 modelos (qwen3-max, deepseek-r1, kimi-k2-0905, etc.) com reporte detalhado |
| `verify_stack.sh` | Smoke test de 5 steps: api status, health, models, model/info, chat |
| `update_api_keys.sh` | Script interativo para rotacao de chaves API com backup e validacao |

### Operacoes

| Script | Funcao |
|---|---|
| `backup_integration_state.sh` | Backup timestampado de .env, compose, search-engine, searxng, e DB dump |
| `reload-newapi.sh` | Force-recreate do new-api com verificacao de `/api/status` |
| `disk-health.sh` | Checa uso de disco (threshold 95%) e faz limpeza segura de cache/logs |

---

## Catalogo de Modelos (`Models_gsd.json`)

O arquivo `integration/Models_gsd.json` e o catalogo central de metadados de modelos, utilizado pelo middleware e scripts de sync:

### Provedores Configurados

| Provedor | Base URL | Modelos |
|---|---|---|
| **atius** | `https://router.atius.com.br/v1` | deepseek-v3.2, deepseek-r1, kimi-k2-0905, qwen3-max, qwen3-vl-plus, minimax-m2.7, minimax-m2.5 |
| **deepseek** | `https://api.deepseek.com` | deepseek-chat, deepseek-reasoner |
| **minimax** | `https://api.minimax.io` | MiniMax-M2.7, MiniMax-M2.5 |
| **qwen** | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | qwen3-235b-a22b, qwen3-235b-a22b-instruct-2507, qwen3-235b-a22b-thinking-2507 |

### Estrutura de Metadata por Modelo

```json
{
  "id": "deepseek-v3.2",
  "name": "DeepSeek V3.2",
  "reasoning": false,
  "input": ["text"],
  "contextWindow": 131072,
  "maxTokens": 65536,
  "cost": { "input": 0.51, "output": 2.04, "cacheRead": 0, "cacheWrite": 0 },
  "aliases": ["deepseek-chat"]
}
```

---

## Fluxos de Dados Principais

### 1. Chat Completion com Enriquecimento de Metadata

```
Cliente → GET /v1/models :3300
  → model-detailed intercepta na porta 3001
  → Proxy para new-api:3000/v1/models
  → Enriquece DeepSeek models com metadata
  → Retorna JSON enriquecido com header X-Model-Metadata-Enriched
```

### 2. Chat Completion com Busca em Tempo Real

```
Cliente → POST /v1/chat/completions
  → Search-Engine intercepta
  → Analisa intencao de busca no prompt
  → Se busca necessaria:
      → SearXNG com engine rotation
      → Injeta contexto nos messages
  → Proxy para new-api:3000/v1/chat/completions
  → Retorna resposta (com reasoning compat se necessario)
```

### 3. Sincronizacao de API Keys

```
Admin executa: ./scripts/update_api_keys.sh
  → Coleta novas chaves interativamente
  → Valida formato (sk-[hex]{32+})
  → Faz backup das chaves antigas
  → Atualiza integration/.env
  → Executa sync_iflow_channel_keys.py ou sync_deepseak_channels.py
  → Testa modelos com smoke test
  → Gera reporte de sucesso/falha
```

### 4. Gerenciamento de Channels via API Admin

```
Admin → POST /api/user/login (username/password)
  → Sessao autenticada com cookie
  → GET /api/channel/ (lista channels existentes)
  → PUT /api/channel/ (cria/atualiza channel com: name, base_url, key, models, group)
  → DB PostgreSQL atualizado
  → NewAPI usa novos channels para roteamento futuro
```

---

## Padroes de Design

| Padrao | Aplicacao |
|---|---|
| **Proxy/Gateway** | NewAPI como ponto unico de entrada para todos os LLMs |
| **Middleware Chain** | model-detailed → new-api → provider (para /v1/models) |
| **Channel Router** | Mapeamento dinamico modelo → provider via PostgreSQL |
| **API Compatibility** | Interface 100% compativel OpenAI para consumidores |
| **Health Check** | PostgreSQL com healthcheck obrigatorio antes do NewAPI |
| **Engine Rotation** | SearXNG com rotacao inteligente para evitar rate-limit |
| **Global Queue** | Round-robin com lanes auto-escalaveis por numero de API keys |
| **Read-Only Middleware** | model_detailed.py montado como volume read-only no container |
| **Graceful Degradation** | Middleware fallback para resposta original em caso de erro |

---

## Modelo de Deploy

| Aspecto | Detalhe |
|---|---|
| **Orquestracao** | Docker Compose (plugin `docker compose` preferido, fallback `docker-compose`) |
| **Restart Policy** | `always` para todos os servicos |
| **Persistencia** | Bind mounts para `./data/` (logs + postgres_data) |
| **Configuracao** | `.env` com variaveis de ambiente (unificado entre compose principal e integration) |
| **CPU Limits** | new-api: 0.5, db-newapi: 0.5, model-detailed: 0.1 |
| **Healthcheck** | PostgreSQL: pg_isready a cada 10s, 10 retries |
| **Reload de .env** | `./reload-newapi.sh` com force-recreate e polling de /api/status |

---

## Consumidores Conhecidos

| Consumidor | Uso |
|---|---|
| **Open-WebUI** | Interface de chat web, consome `/v1/models` e `/v1/chat/completions` |
| **OpenClaw** | Agente AI, consome API OpenAI-compatible |
| **GSD-2** | Planner/agente, resolve provider `atius-router` |
| **Search-Engine** | Middleware de busca, proxy + grounding via SearXNG |

---

## Consideracoes de Escala e Performance

| Aspecto | Estrategia |
|---|---|
| **Multiplas API Keys** | 3 chaves DeepSeek rotativas para distribuicao de carga |
| **Rate Limiting** | Middleware gerencia retries (2) e backoff (1s) |
| **Queue Lanes** | `MODEL_QUEUE_LANES=auto` calculado pelo numero de chaves IFLOW |
| **CPU Limits** | Isolamento por container (0.5/0.5/0.1 cores) |
| **Healthcheck** | PostgreSQL deve estar saudavel antes do NewAPI iniciar |
| **Disk Health** | Monitoramento continuo com threshold de 95% e limpeza automatica |
| **Log Rotation** | Disk-health.sh mantem apenas 5 logs mais recentes do NewAPI |

---

## Seguranca

| Aspecto | Pratica |
|---|---|
| **Tokens** | Bearer tokens (`sk-...`) para todas as chamadas API protegidas |
| **Middleware Read-Only** | `model_detailed.py` montado como `:ro` no container |
| **Env Protegido** | `integration/.env` com permissoes restritas (chmod 600 recomendado) |
| **Backups** | Timestampados com DB dump via pg_dump |
| **Network Isolation** | `newapi-internal` para comunicacao interna, `atius-shared` para consumo externo |
| **Token Rotation** | Script interativo para atualizacao periodica de chaves API |

---

## Monitoramento e Troubleshooting

| Cenario | Acao |
|---|---|
| `system_disk_overloaded` | `./disk-health.sh --cleanup-safe` + `./reload-newapi.sh` |
| Open-WebUI sem modelos | Validar token contra `/v1/models`, normalizar com `normalize_models_real_only.py` |
| Chaves expiradas | Executar `./scripts/update_api_keys.sh` |
| Stack verification | `./scripts/verify_stack.sh` (5-step smoke test) |
| Logs do NewAPI | `./data/logs/oneapi-*.log` (rotacao por timestamp) |
| Logs do container | `docker logs new-api --tail 50` |

---

*Ultima atualizacao: 2026-04-20*
*Versao: 3.0 (com MiniMax provider, middleware enrichment e Bruno tests)*
