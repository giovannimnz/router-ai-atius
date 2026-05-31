# Atius AI Router — Documentacao

## Visao Geral

Gateway de API unificado que agrega MiniMax, DeepSeek e 40+ provedores AI atras de uma unica API compativel com OpenAI/Anthropic.

## Pasta `minimax/`

Documentacao oficial baixada de [platform.minimax.io](https://platform.minimax.io/docs/).

### Arquivos Principais

| Arquivo | Descricao |
|---------|-----------|
| `api-overview.md` | Visao geral das capacidades da API MiniMax |
| `text-anthropic-api-compatible.md` | API compativel com Anthropic (SDK Claude) |
| `text-chat-anthropic.md` | Chat format Anthropic (role-play, multi-turn) |
| `text-openai-api-compatible.md` | API compativel com OpenAI |
| `text-chat-openai.md` | Chat format OpenAI |
| `text-generation.md` | API de geracao de texto (endpoint nativo MiniMax) |
| `prompt-caching-anthropic.md` | Explicit Prompt Caching (cache_control Anthropic) |
| `rate-limits.md` | **Limites de rate limit RPM/TPM por modelo** |
| `pricing-token-plan.md` | Precos do Token Plan (assinatura) |
| `pricing-paygo.md` | Precos Pay-as-you-Go |
| `models-intro.md` | Visao geral dos modelos MiniMax |
| `models-anthropic-list.md` | Lista de modelos compatíveis Anthropic |
| `models-openai-list.md` | Lista de modelos compatíveis OpenAI |
| `text-m2-function-call.md` | Tool use e function calling nos modelos M2 |
| `text-m2-reasoning.md` | Dados de raciocínio e benchmarks |
| `quickstart-sdk.md` | Quickstart usando Anthropic SDK |
| `quickstart-preparation.md` | Preparacao da conta e obtencao de API key |
| `error-codes.md` | Códigos de erro da API MiniMax |
| `text-chat-guide.md` | Guia de chat de texto (role-play, multi-turn) |
| `text-generation-guide.md` | Guia de geracao de texto |
| `token-plan-mini-agent.md` | Mini-Agent com Token Plan |
| `rate-limits.md` | Rate limits: **500 RPM / 20M TPM** para modelos de texto |

### Rate Limits (MiniMax)

| Modelo | RPM | TPM |
|--------|-----|-----|
| MiniMax-M2.7 / M2.7-hs / M2.7-highspeed | 500 | 20,000,000 |
| MiniMax-M2.5 / M2.5-hs / M2.5-highspeed | 500 | 20,000,000 |
| MiniMax-M2.1 / M2.1-hs / M2.1-highspeed | 500 | 20,000,000 |
| MiniMax-M2 | 500 | 20,000,000 |

> **Nota**: TPM de 20M e muito alto (equivalente a ~333K tokens/segundo). O limite pratico e o RPM de 500 req/min (~8.3 req/s).

### Precos (Input/Output por Million de Tokens)

| Modelo | Input | Output | Cache Read | Cache Write |
|--------|-------|--------|------------|-------------|
| MiniMax-M2.7 | $0.30 | $1.20 | $0.06 | $0.375 |
| MiniMax-M2.7-hs (highspeed) | $0.30 | $2.40 | $0.06 | $0.375 |
| MiniMax-M2.5 | $0.30 | $1.20 | $0.03 | $0.375 |
| MiniMax-M2.5-hs (highspeed) | $0.30 | $2.40 | $0.03 | $0.375 |
| MiniMax-M2.1 | $0.30 | $1.20 | $0.03 | $0.375 |
| MiniMax-M2.1-hs (highspeed) | $0.30 | $2.40 | $0.03 | $0.375 |

> Cache write tokens = 1.25x preco de input. Cache read tokens = 0.1x preco de input.

---

## OpenAPI Spec

Arquivo `openapi.json` — especificacao completa dos endpoints da API do router.

### Endpoints Principais

| Metodo | Path | Descricao | Provider |
|--------|------|-----------|----------|
| POST | `/v1/chat/completions` | Chat completions (OpenAI compat.) | MiniMax / DeepSeek |
| POST | `/v1/messages` | Messages (Anthropic compat.) | MiniMax (via `/anthropic/v1/messages`) |
| POST | `/v1/completions` | Completions (legacy) | MiniMax |
| POST | `/v1/embeddings` | Embeddings | MiniMax |
| POST | `/v1/audio/speech` | Text-to-Speech | MiniMax |
| POST | `/v1/audio/transcriptions` | Speech-to-Text | MiniMax |
| POST | `/v1/images/generations` | Image generation | MiniMax |
| POST | `/v1/rerank` | Rerank | MiniMax |
| GET | `/v1/models` | Lista modelos | Auto (OpenAI / Anthropic) |
| GET | `/v1/models/{model}` | Detalhes de modelo | Auto |
| GET | `/api/status` | Status do sistema + API Info | Dashboard |
| POST | `/api/user/register` | Registro | Auth |
| POST | `/api/user/login` | Login | Auth |

### Autenticacao

```bash
# Header padrao
curl -H "Authorization: Bearer <token>" https://router.atius.com.br/v1/chat/completions

# Header Anthropic (para /v1/messages)
curl -H "Authorization: Bearer <token>" \
     -H "x-api-key: <token>" \
     -H "anthropic-version: 2023-06-01" \
     https://router.atius.com.br/v1/messages
```

### Exemplo: Chat Completions

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-atius-xxxxx",
    base_url="https://router.atius.com.br/v1"
)

response = client.chat.completions.create(
    model="MiniMax-M2.7",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Explain quantum computing simply."}
    ],
    max_tokens=1024,
    temperature=0.7
)
print(response.choices[0].message.content)
```

### Exemplo: Anthropic Messages (via SDK)

```python
import anthropic

client = anthropic.Anthropic(
    api_key="sk-atius-xxxxx",
    base_url="https://router.atius.com.br/v1"
)

message = client.messages.create(
    model="MiniMax-M2.7",
    max_tokens=1024,
    system="You are a helpful assistant.",
    messages=[
        {"role": "user", "content": "Explain quantum computing simply."}
    ]
)
print(message.content[0].text)
```

---

## Model Mapping (aliases `-hs`)

| Alias usado pelo cliente | Maps para (upstream MiniMax) |
|--------------------------|------------------------------|
| `MiniMax-M2.7-hs` | `MiniMax-M2.7-highspeed` |
| `MiniMax-M2.5-hs` | `MiniMax-M2.5-highspeed` |
| `MiniMax-M2.1-hs` | `MiniMax-M2.1-highspeed` |

O mapping e feito automaticamente pelo `model_mapping` no canal MiniMax no DB.

---

## Arquitetura de Routing

```
Cliente (SDK OpenAI/Anthropic)
    |
    v
Atius AI Router  (relay-v1Router)
    |
    +-- POST /v1/chat/completions --> RelayFormatOpenAI --> minimax adaptor
    |                                        (base_url: api.minimax.io)
    |                                        +-- /v1/text/chatcompletion_v2
    |
    +-- POST /v1/messages --> RelayFormatClaude --> minimax adaptor
                                         (base_url: api.minimax.io)
                                         +-- /anthropic/v1/messages
    |
    +-- POST /v1/embeddings --> embedding handler --> minimax
    +-- POST /v1/audio/speech --> minimax TTS
    +-- POST /v1/images/generations --> minimax image

Channel Selection (distribute middleware):
    Token --> Abilities table --> matching (group, model) --> channel_id
    --> Channels table --> base_url, model_mapping
```

---

## Configuracao do Banco

- **Database**: `newapi` (PostgreSQL, container `db-newapi`)
- **Tabelas principais**: `channels`, `models`, `abilities`, `options`
- **Config**: `console_setting.api_info` em `options` (armazenado em DB, carregado na memoria em startup)
