<!-- generated-by: gsd-doc-writer -->
# API

Este documento descreve os endpoints registrados pelo backend Go de `github.com/QuantumNous/new-api` neste fork, com foco no contrato publico `/v1`, autenticacao, rotas OpenAI/Codex, embeddings e rotas administrativas usadas para operar esses caminhos.

## Authentication

Os endpoints de relay usam token de API no formato OpenAI:

```bash
Authorization: Bearer <api-token>
```

`middleware.TokenAuth()` remove o prefixo `Bearer ` e tambem aceita chaves com prefixo `sk-`. Para compatibilidade:

| Caso | Como autentica | Observacao |
| --- | --- | --- |
| OpenAI-compatible `/v1/*` | `Authorization: Bearer <api-token>` | Caminho padrao de relay. |
| Anthropic-compatible `/v1/messages` e `/v1/models` | `x-api-key: <api-token>` com `anthropic-version` | O middleware copia `x-api-key` para `Authorization`. |
| Gemini `/v1beta/models`, `/v1beta/openai/models` e `/v1beta/models/*path` | `x-goog-api-key: <api-token>` ou `?key=<api-token>` | O middleware copia a chave para `Authorization`. |
| Realtime WebSocket `/v1/realtime` | `Sec-WebSocket-Protocol` contendo `openai-insecure-api-key.<api-token>` | O middleware transforma esse valor em `Authorization`. |
| Video content proxy `/v1/videos/:task_id/content` | Token de API ou sessao de dashboard | Usa `TokenOrUserAuth()`. |

Somente as rotas `/api` protegidas por `UserAuth`, `AdminAuth` ou `RootAuth` usam sessao de dashboard ou access token de usuario. Rotas publicas e callbacks/webhooks registrados sem esses middlewares ficam fora desse contrato. Quando a chamada autenticada nao vem de sessao, esses middlewares esperam:

```bash
Authorization: Bearer <access-token>
New-Api-User: <user-id>
```

`GET /api/user/token` gera esse access token para o usuario autenticado. Rotas `AdminAuth` exigem papel de admin; rotas `RootAuth` exigem root. As rotas de channel ainda aplicam permissoes granulares via `RequirePermission`, por exemplo `ChannelRead`, `ChannelOperate`, `ChannelWrite` e `ChannelSensitiveWrite`.

## Endpoints Overview

### Public `/v1` Relay

| Method | Path | Description | Auth Required |
| --- | --- | --- | --- |
| `GET` | `/v1/models` | Lista modelos disponiveis. Retorna OpenAI por default ou Anthropic quando recebe headers Anthropic. | API token |
| `GET` | `/v1/models/:model` | Retorna metadados de um modelo no formato OpenAI ou Anthropic. | API token |
| `GET` | `/v1/claude/models` | Lista modelos Anthropic-compatible em formato Claude. | API token |
| `POST` | `/v1/messages` | Relay Anthropic Messages. | API token |
| `POST` | `/v1/completions` | Relay OpenAI Completions legado. | API token |
| `POST` | `/v1/chat/completions` | Relay OpenAI Chat Completions. Para channel type Codex, o backend converte para Responses quando pass-through nao esta ativo. | API token |
| `POST` | `/v1/responses` | Relay OpenAI Responses API. Caminho nativo para Codex. | API token |
| `POST` | `/v1/responses/compact` | Relay de compaction para Responses. | API token |
| `POST` | `/v1/edits` | Relay OpenAI edits legado. | API token |
| `POST` | `/v1/images/generations` | Relay de geracao de imagens. | API token |
| `POST` | `/v1/images/edits` | Relay de edicao de imagens. | API token |
| `POST` | `/v1/embeddings` | Relay OpenAI-compatible de embeddings. `embedding-gte-v1` passa pelo governor Go-native. | API token |
| `POST` | `/v1/audio/transcriptions` | Relay de audio transcription. | API token |
| `POST` | `/v1/audio/translations` | Relay de audio translation. | API token |
| `POST` | `/v1/audio/speech` | Relay text-to-speech. | API token |
| `POST` | `/v1/rerank` | Relay rerank. | API token |
| `GET` | `/v1/realtime` | WebSocket Realtime. | API token via WebSocket protocol |
| `POST` | `/v1/engines/:model/embeddings` | Gemini-compatible embedding path registrado no grupo `/v1`. | API token |
| `POST` | `/v1/models/*path` | Gemini-compatible relay path registrado no grupo `/v1`. | API token |

### Gemini And Alternate Public Relay

| Method | Path | Description | Auth Required |
| --- | --- | --- | --- |
| `GET` | `/v1beta/models` | Lista modelos no formato Gemini. | API token |
| `GET` | `/v1beta/openai/models` | Lista modelos OpenAI-compatible no namespace Gemini OpenAI. | API token |
| `POST` | `/v1beta/models/*path` | Relay Gemini API, incluindo caminhos como generateContent/embedContent. | API token |
| `POST` | `/v1/video/generations` | Gera video pelo relay de task. | API token |
| `GET` | `/v1/video/generations/:task_id` | Consulta task de video. | API token |
| `POST` | `/v1/videos` | Video OpenAI-compatible. | API token |
| `GET` | `/v1/videos/:task_id` | Consulta video OpenAI-compatible. | API token |
| `GET` | `/v1/videos/:task_id/content` | Proxy do conteudo do video. | API token ou sessao |
| `POST` | `/v1/videos/:video_id/remix` | Remix de video. | API token |
| `POST` | `/kling/v1/videos/text2video` | Relay Kling text-to-video. | API token |
| `POST` | `/kling/v1/videos/image2video` | Relay Kling image-to-video. | API token |
| `GET` | `/kling/v1/videos/text2video/:task_id` | Consulta task Kling text-to-video. | API token |
| `GET` | `/kling/v1/videos/image2video/:task_id` | Consulta task Kling image-to-video. | API token |
| `POST` | `/jimeng/` | Relay Jimeng official API. | API token |

### Registered But Not Implemented

Estas rotas existem no router Go e retornam `501` com `code: "api_not_implemented"`:

| Method | Path |
| --- | --- |
| `POST` | `/v1/images/variations` |
| `GET` | `/v1/files` |
| `POST` | `/v1/files` |
| `DELETE` | `/v1/files/:id` |
| `GET` | `/v1/files/:id` |
| `GET` | `/v1/files/:id/content` |
| `POST` | `/v1/fine-tunes` |
| `GET` | `/v1/fine-tunes` |
| `GET` | `/v1/fine-tunes/:id` |
| `POST` | `/v1/fine-tunes/:id/cancel` |
| `GET` | `/v1/fine-tunes/:id/events` |
| `DELETE` | `/v1/models/:model` |

### Administrative Endpoints

| Method | Path | Description | Auth Required |
| --- | --- | --- | --- |
| `GET` | `/api/status` | Status publico do sistema. | No |
| `GET` | `/api/models` | Lista modelos por tipo de canal para o dashboard. | User |
| `GET` | `/api/user/token` | Gera access token de usuario para chamadas `/api`. | User |
| `GET` | `/api/token/`, `/api/token/search`, `/api/token/:id` | Lista, busca e detalhe de tokens de API do usuario. | User |
| `POST` | `/api/token/`, `/api/token/:id/key`, `/api/token/batch`, `/api/token/batch/keys` | Cria token, revela chave ou faz operacoes em lote. | User |
| `PUT` | `/api/token/` | Atualiza token de API. | User |
| `DELETE` | `/api/token/:id` | Remove token de API. | User |
| `GET` | `/api/usage/token/` | Consulta uso por token em modo read-only. | API token read-only |
| `GET` | `/api/channel/`, `/api/channel/search`, `/api/channel/models`, `/api/channel/models_enabled`, `/api/channel/ops`, `/api/channel/:id` | Consulta canais/provedores e modelos disponiveis. | Admin + permission |
| `POST` | `/api/channel/`, `/api/channel/status/batch`, `/api/channel/:id/status`, `/api/channel/fetch_models`, `/api/channel/fix` | Cria canal, altera status, busca modelos upstream ou corrige abilities. | Admin + permission |
| `PUT` | `/api/channel/`, `/api/channel/tag` | Atualiza canal ou tags. | Admin + permission |
| `DELETE` | `/api/channel/:id`, `/api/channel/disabled` | Remove canal especifico ou canais disabled. | Admin + permission |
| `POST` | `/api/channel/:id/key` | Revela chave de canal depois de verificacao segura. | Root |
| `POST` | `/api/channel/:id/codex/refresh` | Atualiza credencial OAuth armazenada de um canal Codex. | Admin + ChannelSensitiveWrite |
| `GET` | `/api/channel/:id/codex/usage` | Consulta uso upstream Codex/WHAM para o canal. | Admin + ChannelRead |
| `GET` | `/api/channel/:id/codex/usage/reset-credits` | Consulta creditos de reset de rate limit Codex. | Admin + ChannelRead |
| `POST` | `/api/channel/:id/codex/usage/reset` | Consome reset credit para uso Codex. | Admin + ChannelOperate |
| `POST` | `/api/option/codex_catalog/sync` | Sincroniza catalogo Codex para um canal Codex. | Root |
| `GET` | `/api/models/`, `/api/models/search`, `/api/models/:id`, `/api/models/missing`, `/api/models/sync_upstream/preview` | Consulta metadados de modelos, missing models e preview de sync upstream. | Admin |
| `POST` | `/api/models/`, `/api/models/sync_upstream` | Cria metadado de modelo ou aplica sync upstream. | Admin |
| `PUT` | `/api/models/` | Atualiza metadado de modelo. | Admin |
| `DELETE` | `/api/models/:id` | Remove metadado de modelo. | Admin |
| `GET` | `/api/authz/catalog` | Catalogo de permissoes administrativas. | Admin |
| `GET` | `/api/log/`, `/api/log/stat`, `/api/log/search`, `/api/log/channel_affinity_usage_cache` | Logs globais e estatisticas administrativas. | Admin |
| `GET` | `/api/log/self`, `/api/log/self/stat`, `/api/log/self/search`, `/api/log/token` | Logs do usuario ou do token. | User/API token conforme rota |
| `GET` | `/api/data/`, `/api/data/users`, `/api/data/flow` | Dados globais de quota/flow. | Admin |
| `GET` | `/api/data/self`, `/api/data/flow/self` | Dados de quota/flow do usuario autenticado. | User |
| `GET` | `/internal/v1/models` | Lista abilities com `channel_type` para uso interno. | No |

## Request/Response Formats

### Model List

`GET /v1/models` retorna OpenAI-compatible por default:

```json
{
  "data": [
    {
      "id": "model-name",
      "object": "model",
      "created": 1626777600,
      "owned_by": "provider",
      "name": "Display name",
      "provider": "OpenAI Codex",
      "supported_endpoint_types": ["openai-response", "openai"],
      "endpoint_routes": {
        "openai-response": "/v1/responses",
        "openai": "/v1/chat/completions"
      },
      "pricing": {
        "input": 0,
        "output": 0
      },
      "billing_mode": "tiered_expr"
    }
  ]
}
```

O root publico confirmado pelo controller e testes e `{"data":[...]}`. Use `supported_endpoint_types` e `endpoint_routes` para decidir se um modelo deve ir para `/v1/chat/completions`, `/v1/responses`, `/v1/responses/compact`, `/v1/messages`, `/v1/embeddings`, `/v1/rerank` ou `/v1/images/generations`.

Com headers Anthropic (`x-api-key` + `anthropic-version`), o mesmo `GET /v1/models` retorna `data` com itens Anthropic. Para listagem Gemini, use `GET /v1beta/models`, que retorna `models` e `nextPageToken`.

### Chat Completions

`POST /v1/chat/completions` aceita o DTO OpenAI-compatible `GeneralOpenAIRequest`. Para chat, `messages` e obrigatorio, exceto requests FIM com `prefix`/`suffix`.

```json
{
  "model": "model-name",
  "messages": [
    {"role": "user", "content": "Responda OK"}
  ],
  "stream": true,
  "temperature": 0.2
}
```

Para channel type Codex, a politica `ShouldChatCompletionsUseResponsesPolicy` sempre habilita conversao Chat Completions -> Responses quando pass-through global/canal nao esta ativo. O adaptor Codex envia o request convertido ao upstream Codex Responses.

### Responses

`POST /v1/responses` exige `model` e `input`.

```json
{
  "model": "model-name",
  "input": "Responda OK",
  "instructions": "Use respostas curtas",
  "stream": true
}
```

O DTO de Responses aceita campos como `include`, `conversation`, `context_management`, `previous_response_id`, `reasoning`, `tools`, `tool_choice`, `text`, `stream_options`, `prompt_cache_key`, `prompt_cache_retention`, `metadata`, `truncation` e `max_tool_calls`.

`POST /v1/responses/compact` usa um DTO menor:

```json
{
  "model": "model-name",
  "input": "...conteudo longo...",
  "instructions": "Compacte mantendo fatos importantes",
  "previous_response_id": "resp_..."
}
```

### Embeddings

`POST /v1/embeddings` aceita:

```json
{
  "model": "embedding-gte-v1",
  "input": ["texto 1", "texto 2"],
  "encoding_format": "float",
  "dimensions": 768
}
```

Resposta OpenAI-compatible:

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [0.01, -0.02]
    }
  ],
  "model": "embedding-gte-v1",
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 0,
    "total_tokens": 10
  }
}
```

`embedding-gte-v1` e o modelo default governado em `service/embeddinggovernor`. O relay aplica fail-closed antes do dispatch upstream quando esse modelo recebe mais de 4 itens em `input`. O header opcional `X-Embedding-Workload` pode classificar a chamada como `batch`, `bulk`, `interactive` ou `realtime`; isso altera a fila operacional, nao o nome publico do modelo.

### Admin Envelope

Muitas rotas `/api/*` retornam o envelope comum:

```json
{
  "success": true,
  "message": "",
  "data": {}
}
```

Outras rotas administrativas retornam envelopes especificos do handler, mas o padrao acima e o helper `common.ApiSuccess`. Erros administrativos usam `success: false` e `message`.

## Error Codes

Relay OpenAI-compatible retorna:

```json
{
  "error": {
    "message": "mensagem com request id quando disponivel",
    "type": "new_api_error",
    "code": "invalid_request"
  }
}
```

Erros Anthropic-compatible em `/v1/messages` usam:

```json
{
  "type": "error",
  "error": {
    "type": "new_api_error",
    "message": "mensagem"
  }
}
```

Codigos observaveis no codigo incluem:

| Code | Typical HTTP Status | Meaning |
| --- | --- | --- |
| `invalid_request` | `400` ou `500` | Request body invalido ou campos obrigatorios ausentes; handlers especificos usam `400`, mas o relay pode herdar o default `500` quando `GetAndValidateRequest` falha sem status explicito. |
| `read_request_body_failed` | `400`, `413` ou `500` | Falha ao ler body; `413` quando excede limite de tamanho, `400` em caminhos com status explicito e `500` no pass-through quando herda o default. |
| `model_price_error` | `400` | Modelo sem configuracao de preco/quota valida para billing. |
| `model_not_found` | `200` ou `503` | `RetrieveModel` retorna HTTP `200` com `error.code`; o distributor retorna `503` quando nao ha canal/modelo disponivel. |
| `insufficient_user_quota` | Proibido neste fork | Tokens pessoais validos nao sao bloqueados por saldo de user, wallet, account ou subscription; consulte [Atius user quota invariant](./ATIUS-USER-QUOTA-INVARIANT.md). |
| `channel:no_available_key` | Varia por handler | Canal sem chave disponivel. |
| `channel:model_mapped_error` | Varia por handler | Falha no mapeamento de modelo para upstream. |
| `convert_request_failed` | Varia por handler | Falha ao converter request para o provider. |
| `do_request_failed` | `500` | Falha local ao executar request upstream. |
| `api_not_implemented` | `501` | Rota registrada mas nao implementada. |

`TokenAuth` retorna `401` para token ausente/invalido e `403` para usuario desabilitado ou restricao de IP/grupo. Quando o governor de embeddings rejeita por pressao, ele pode retornar `Retry-After`.

## Rate Limits

Os limites sao aplicados por middleware:

| Area | Middleware | Default no codigo |
| --- | --- | --- |
| `/api/*` | `GlobalAPIRateLimit()` | Habilitado, 180 requests por 180 segundos por IP. |
| Web/static | `GlobalWebRateLimit()` | Habilitado, 60 requests por 180 segundos por IP. |
| Rotas sensiveis `/api/*` | `CriticalRateLimit()` | Habilitado, 20 requests por 20 minutos por IP. |
| Busca de token/log | `SearchRateLimit()` | Habilitado, 10 requests por 60 segundos por usuario. |
| Relay `/v1/*` | `ModelRequestRateLimit()` | Desabilitado por default; quando habilitado, usa janela de 1 minuto, limite total default `0` e limite de sucesso default `1000`. |
| Embeddings governados | `embeddinggovernor.Acquire()` | `embedding-gte-v1` default, concorrencia inicial `2`, minima `1`, filas default `128` interativa e `512` batch. |

Quando Redis esta ativo, os limiters usam Redis; caso contrario usam limiter em memoria. Configuracoes de rate limit podem variar por ambiente via variaveis/configuracao do runtime, entao trate os defaults acima como defaults do codigo, nao como garantia operacional de uma instalacao especifica.
