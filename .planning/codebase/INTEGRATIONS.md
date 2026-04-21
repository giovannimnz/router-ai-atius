# INTEGRATIONS.md - External Services & Integrations

## Visão Geral

O NewAPI atua como um **gateway centralizado** para múltiplos provedores de LLM, integrando-se com serviços externos e fornecendo uma API compatível OpenAI para consumidores internos.

## Integrações com Providers de LLM

### DeepSeek

| Item | Detalhe |
|---|---|
| **URL Base** | `https://api.deepseek.com` |
| **API Keys** | 3 chaves rotativas (`DEEPSEAK_API_KEY_1/2/3`) |
| **Modelos** | DeepSeek-R1, DeepSeek-V3.2-Exp |
| **Sync Script** | `integration/scripts/sync_deepseak_channels.py` |

### Qwen (Alibaba Cloud)

| Item | Detalhe |
|---|---|
| **Modelos** | Qwen3-Max, Qwen3-VL-Plus, Qwen3-Coder-Plus |
| **Config** | Gerenciada via UI admin do NewAPI |
| **Use Cases** | Geral/Reasoning, Visão/Multimodal, Coding Longo |

### Kimi (Moonshot AI)

| Item | Detalhe |
|---|---|
| **Modelo** | Kimi-K2-Instruct-0905 |
| **Use Case** | Coding Diário |
| **Config** | Gerenciada via UI admin do NewAPI |

### MiniMax

| Item | Detalhe |
|---|---|
| **URL Base** | `https://api.minimax.io` (região global, sem `/v1` — o new-api adiciona) |
| **API Key** | `MINIMAX_API_KEY` no `~/.zshrc` (formato `sk-cp-*`, Token Plan) |
| **Channel ID** | 42 |
| **Channel Type** | 1 (OpenAI Compatible) |
| **Modelos** | MiniMax-M2.7, MiniMax-M2.5 |
| **Vendor ID** | 4 (MiniMax) |
| **Token Plan** | Weekly quota: 4.500 tokens para modelos M*, 350 images, 700 music, etc. |
| **Bruno Tests** | `integration/bruno-tests/minimax/` |

#### Pricing Pay-as-you-go (referência)

| Modelo | Input (/1M) | Output (/1M) | Cache Read (/1M) | Cache Write (/1M) |
|---|---|---|---|---|
| MiniMax-M2.7 | $0.30 | $1.20 | $0.06 | $0.375 |
| MiniMax-M2.5 | $0.30 | $1.20 | $0.03 | $0.375 |

#### Pricing no NewAPI (ratios)

| Modelo | ModelRatio | CompletionRatio |
|---|---|---|
| MiniMax-M2.7 | 0.15 | 4 |
| MiniMax-M2.5 | 0.15 | 4 |

- **ModelRatio** = `input_price / 2` (mesma fórmula do DeepSeek: $0.28 → 0.14)
- **CompletionRatio** = `output_price / input_price` (1.2 / 0.3 = 4.0)

#### Endpoint /v1/models enriquecido (middleware)

O middleware `model_detailed.py` adiciona metadata aos modelos MiniMax no `GET /v1/models`:

```json
{
  "id": "MiniMax-M2.7",
  "name": "MiniMax M2.7",
  "context_length": 245760,
  "top_provider": { "max_completion_tokens": 50000 },
  "pricing": {
    "prompt": "0.0000003",
    "completion": "0.0000012",
    "prompt_cache_hit": "0.00000006",
    "prompt_cache_miss": "0.000000375"
  }
}
```

#### Notas importantes

- O Token Plan usa a região **global** (`api.minimax.io`), não a China (`api.minimaxi.com`)
- O base_url no channel **não deve** ter `/v1` (o new-api adiciona automaticamente)
- O modelo MiniMax-M1 **não** está incluído no Token Plan atual
- O tipo de canal recomendado é **OpenAI Compatible (type 1)**, não Anthropic, pois retorna conteúdo de forma mais consistente (thinking tokens inclusos)

## Integrações Internas (Arquitetura)

### PostgreSQL

| Item | Detalhe |
|---|---|
| **Imagem** | `postgres:15-alpine` |
| **Host:Port** | `localhost:8746` (local), `atius-srv-1:8746` (via VPN) |
| **Database** | `newapi` |
| **Usuário** | `admin` |
| **Persistência** | `./data/postgres_data/` |
| **Healthcheck** | `pg_isready` a cada 10s |

### Open-WebUI

| Item | Detalhe |
|---|---|
| **Tipo** | Consumidor da API NewAPI |
| **Auth** | Via `OPENWEBUI_LITELLM_KEY` (deve alinhar com `NEWAPI_ADMIN_TOKEN`) |
| **Endpoint** | `http://localhost:3300/v1/models` |
| **Protocolo** | OpenAI-compatible REST API |

### OpenClaw

| Item | Detalhe |
|---|---|
| **Tipo** | Consumidor da API NewAPI |
| **Auth** | Bearer token compartilhado |

### SearXNG

| Item | Detalhe |
|---|---|
| **Tipo** | Motor de busca self-hosted |
| **Config** | `integration/searxng/settings.yml` |
| **Uso** | Search-engine middleware para grounding de LLMs |

### Whisper.cpp

| Item | Detalhe |
|---|---|
| **Tipo** | Speech-to-Text local |
| **Local** | `integration/whisper.cpp/` |
| **Uso** | Transcrição de áudio para input de LLM |

## Scripts de Integração

| Script | Função |
|---|---|
| `sync_deepseak_channels.py` | Sincroniza canais/channels do DeepSeek no NewAPI |
| `sync_openrouter_channels.py` | Sincroniza canais do OpenRouter |
| `sync_iflow_channel_keys.py` | Sincroniza chaves de canais iFlow |
| `normalize_models_real_only.py` | Normaliza catálogo de modelos (formato real only) |
| `test_all_models.sh` | Testa todos os modelos configurados |
| `bruno-tests/minimax/` | Testes Bruno para MiniMax-M2.7 e M2.5 |
| `update_api_keys.sh` | Atualiza API keys |
| `update_newapi_safe.sh` | Atualização segura do NewAPI |
| `verify_stack.sh` | Verifica saúde do stack completo |
| `backup_integration_state.sh` | Backup do estado de integração |

## Middleware Model Enrichment

O middleware `model-detailed` (porta 3300) intercepta `GET /v1/models` e enriquece os modelos com metadata (nome, contexto, pricing).

| Config | Valor |
|---|---|
| **Arquivo** | `integration/middleware/model_detailed.py` |
| **Porta** | 3001 (interna) → 3300 (exposta) |
| **Backend** | `http://new-api:3000` |
| **Modelos enriquecidos** | deepseek-chat, deepseek-reasoner, MiniMax-M2.7, MiniMax-M2.5 |

### Lógica de Pricing

Os preços são por **token individual** (não por milhão):

| Provedor | Modelo | Prompt | Completion | Cache Hit |
|---|---|---|---|---|
| DeepSeek | deepseek-chat | $0.00000028 | $0.00000042 | $0.000000028 |
| DeepSeek | deepseek-reasoner | $0.00000028 | $0.00000042 | $0.000000028 |
| MiniMax | MiniMax-M2.7 | $0.0000003 | $0.0000012 | $0.00000006 |
| MiniMax | MiniMax-M2.5 | $0.0000003 | $0.0000012 | $0.00000003 |

MiniMax adiciona também `prompt_cache_miss` ($0.000000375) pois possui caching write separado.

## Middleware Search-Engine

| Config | Valor |
|---|---|
| `MODEL_QUEUE_LANES` | `auto` |
| `RATE_LIMIT_RETRIES` | `2` |
| `RATE_LIMIT_BACKOFF_S` | `1.0` |
| `ENABLE_REASONING_STREAM_COMPAT` | `true` |
| `REASONING_STREAM_COMPAT_MODELS` | `deepseek-r1` |

## Redes Docker

| Rede | Tipo | Serviços |
|---|---|---|
| `newapi-internal` | Internal | `new-api`, `db-newapi` |
| `atius-shared` | External | `new-api` (alias: `litellm`) — compartilhada com outros serviços do ecossistema Atius |

## Tokens & Autenticação

| Token | Uso |
|---|---|
| `NEWAPI_ADMIN_TOKEN` | Token admin principal (sk-vXqh...) |
| `LITELLM_MASTER_KEY` | Alias para `NEWAPI_ADMIN_TOKEN` (compatibilidade) |
| `DEEPSEAK_API_KEY_1/2/3` | Chaves da API DeepSeek |
| `MINIMAX_API_KEY` | Token Plan MiniMax (formato `sk-cp-*`, no `~/.zshrc`) |

## Arquivos de Configuração de Integração

| Arquivo | Descrição |
|---|---|
| `integration/.env` | Variáveis unificadas (NewAPI + Middleware) |
| `integration/docker-compose.yml` | Compose com redes internas + externas |
| `integration/Models_gsd.json` | Catálogo de modelos |
| `integration/GUIA_MUDANCA_CHAVES_API.md` | Guia de mudança de chaves de API |
| `integration/searxng/settings.yml` | Configuração do SearXNG |

## Consumidores Conhecidos

1. **Open-WebUI** — Interface de chat para LLMs
2. **OpenClaw** — Agente de IA
3. **Search-Engine middleware** — Proxy de busca com rate limiting
