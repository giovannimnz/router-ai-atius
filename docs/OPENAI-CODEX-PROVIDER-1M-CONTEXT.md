# OpenAI - Codex provider e modelos 1M context

Data de referencia: 2026-07-01.

Este documento descreve a implementacao do provider `OpenAI - Codex` no fork
`router-ai-atius`, incluindo catalogo publico, roteamento, custos, billing
interno, guardrails de contexto e comandos de validacao. As evidencias de
aliases `-1m` de 2026-07-01 ficam preservadas aqui apenas como historico
tecnico; no contrato final restaurado da Phase 24, esses aliases nao devem
aparecer no runtime final.

O objetivo e deixar claro como o cliente chama o router, como o router traduz o
modelo para o upstream real, como o custo e calculado e quais limites foram
observados nos testes de contexto progressivo.

## Resumo executivo

- O provider ativo e `OpenAI - Codex`.
- O channel runtime e o channel `5`, tipo `57`.
- A implementacao e Go-native; nao depende de sidecar Python/model-detailed no
  caminho canonico de `/v1/`.
- O endpoint publico principal para clientes OpenAI-compatible e:
  `/v1/chat/completions`.
- Internamente, chamadas Codex de chat sao convertidas para Responses API.
- Existem dois modos de upstream:
  - OAuth/ChatGPT Codex: envia para `https://chatgpt.com/backend-api/codex/responses`.
  - OpenAI public API: envia para `https://api.openai.com/v1/responses`.
- O modo OAuth/ChatGPT e o modo ativo no channel runtime `5`.
- O modo OpenAI public API e o caminho necessario para validar a janela publica
  de 1,05M conforme a documentacao OpenAI de GPT-5.4/GPT-5.5.
- No contrato final restaurado da Phase 24, `gpt-5.4` e o modelo Codex default
  para long-context; ele fica publicado como modelo base, sem alias `-1m`.
- `gpt-5.5` permanece ativo como modelo Codex standard, tambem sem alias `-1m`
  no runtime final.
- As referencias a `gpt-5.4-1m` e `gpt-5.5-1m` neste documento servem apenas
  como historico de validacao e rollback do experimento 1M de 2026-07-01.
- Billing, quota e logs internos preservam o modelo solicitado pelo cliente
  atraves de `OriginModelName`.
- O modelo enviado ao upstream fica rastreavel em `UpstreamModelName`.
- O caminho governado de embeddings permanece inalterado: `embedding-gte-v1` e
  o unico alias publico governado e `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1`
  continua sendo o contrato esperado.

## Fonte externa e escopo da documentacao anexada

O PDF anexado compara janelas de contexto GPT/Codex e pricing long-context. A
leitura operacional para este fork e:

- A janela de contexto e total: input + output precisam caber no limite do
  modelo.
- `gpt-5.5` e `gpt-5.4` aparecem na documentacao publica da OpenAI com janela
  proxima de `1.050.000` tokens e max output de `128.000`.
- A tabela publica de pricing separa short context e long context:
  - `gpt-5.5`: short `5/30`, long `10/45`.
  - `gpt-5.4`: short `2.5/15`, long `5/22.5`.
- Para `gpt-5.4`, prompts acima de `272000` input tokens usam a tarifa
  long-context para a sessao inteira.
- O PDF nao identifica um header ou body field especial para ativar 1M no
  backend OAuth/ChatGPT Codex.

Conclusao: o PDF valida a janela publica e o pricing long-context para os
modelos base, mas a janela 1M publica pertence ao caminho da API publica
OpenAI. O backend
OAuth/ChatGPT Codex tem limites proprios, observados por teste e tambem
refletidos no catalogo do cliente oficial Codex.

Estado operacional atual:

- `gpt-5.4` permanece habilitado e e o modelo Codex default para long-context
  no contrato final restaurado.
- `gpt-5.5` permanece habilitado como modelo Codex standard.
- `gpt-5.4-1m` e `gpt-5.5-1m` nao devem existir no catalogo/runtime final da
  Phase 24.
- `embedding-gte-v1` continua sendo o unico alias publico governado de
  embeddings, com `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1`.

## Arquivos principais

| Area | Arquivo |
|---|---|
| Registro do adaptor Codex | `relay/relay_adaptor.go` |
| Implementacao do channel type 57 | `relay/channel/codex/adaptor.go` |
| OAuth/key parser do Codex | `relay/channel/codex/oauth_key.go` |
| Auto-refresh de credencial Codex | `service/codex_credential_refresh.go`, `service/codex_credential_refresh_task.go` |
| Conversao Chat Completions -> Responses | `relay/chat_completions_via_responses.go` |
| Politica que sempre usa Responses para Codex | `service/openaicompat/policy.go` |
| Agregacao Responses -> Chat Completions | `relay/channel/openai/chat_via_responses.go` |
| Model mapping/alias | `relay/helper/model_mapped.go` |
| Catalogo publico `/v1/models` | `controller/model.go`, `service/modelcatalog/catalog.go` |
| Pricing/ratios | `setting/ratio_setting/model_ratio.go` |
| Guardrail de contexto Codex | `controller/codex_context_limit.go` |
| Teste progressivo 1M | `scripts/test-long-context-aliases.sh` |

## Channel runtime

Estado validado com `bin/clianything providers --all`:

| Campo | Valor |
|---|---|
| Channel ID | `5` |
| Nome | `OpenAI - Codex` |
| Tipo | `57` |
| Grupo | `default` |
| Status | enabled |
| Modelos expostos no canal | `gpt-5.5,gpt-5.4,gpt-5.4-mini,gpt-5.3-codex-spark` |
| Model mapping | `{}` ou vazio para os modelos base restaurados |

O provider usa credencial OAuth Codex. A credencial real nunca deve ser
registrada em docs, logs manuais ou scripts. Para testes, usar sempre
placeholders como `ROUTER_TEST_KEY` ou `COLOQUE_UM_TOKEN_DE_TESTE_AQUI`.

O channel `5` atual nao e um canal API-key da OpenAI publica. Ele usa OAuth
ChatGPT/Codex, logo nao consegue provar por si so o entitlement da API publica
`/v1/responses` para 1,05M em ambos os modelos.

## Endpoints suportados

### Publico para clientes

O contrato publico OpenAI-compatible para chat e:

```http
POST /v1/chat/completions
```

O cliente pode enviar:

```json
{
  "model": "gpt-5.4",
  "messages": [
    {
      "role": "user",
      "content": "Responda apenas OK."
    }
  ],
  "max_completion_tokens": 16,
  "stream": false
}
```

Tambem sao suportados os modelos base:

- `gpt-5.5`
- `gpt-5.4`

E os demais modelos Codex publicados no catalogo:

- `gpt-5.4-mini`
- `gpt-5.3-codex-spark`

### Upstream real no modo OAuth/ChatGPT

O adaptor Codex nao envia `/v1/chat/completions` diretamente ao upstream. Para
o channel runtime atual, o router converte a chamada para Responses e usa:

```text
/backend-api/codex/responses
```

Para compaction/compact:

```text
/backend-api/codex/responses/compact
```

Para embeddings Codex/OpenAI, quando habilitados, o adaptor usa a API OpenAI
padrao:

```text
https://api.openai.com/v1/embeddings
```

No contrato final restaurado, nenhum alias `-1m` e publicado no runtime. O
modelo de chat long-context default permanece `gpt-5.4`.

### Upstream real no modo OpenAI public API

Quando um channel Codex tipo `57` for configurado explicitamente com
`base_url=https://api.openai.com` ou `base_url=https://api.openai.com/v1` e uma
API key publica da OpenAI, o adaptor usa:

```text
https://api.openai.com/v1/responses
```

Nesse modo:

- `max_output_tokens` e preservado.
- `temperature` e preservado.
- headers de ChatGPT/Codex, como `chatgpt-account-id`, `OpenAI-Beta` e
  `originator`, nao sao enviados.
- uma credencial OAuth JSON e rejeitada para `/v1/responses`; o modo publico
  exige API key propria da API OpenAI.
- `/v1/responses/compact` nao e suportado no modo publico, porque esse endpoint
  e interno do backend Codex/ChatGPT.

## Headers upstream do Codex

No modo OAuth/ChatGPT, o adaptor Codex monta headers a partir da credencial
OAuth resolvida:

- `Authorization: Bearer TOKEN_REDACTED`
- `chatgpt-account-id: <account_id>`
- `OpenAI-Beta: responses=experimental`, quando o header nao veio preenchido
- `originator: codex_cli_rs`, quando o header nao veio preenchido
- `Content-Type: application/json`
- `Accept: text/event-stream` quando a requisicao upstream e stream

Para embeddings, o adaptor nao adiciona headers especificos de ChatGPT/Codex
como `chatgpt-account-id`; ele usa o caminho OpenAI embeddings com OAuth.

No modo OpenAI public API, o adaptor monta:

- `Authorization: Bearer sk-REDACTED`
- `Content-Type: application/json`
- `Accept: application/json` ou `Accept: text/event-stream`

Esse modo nao deve ser apontado para OpenAI-compatible generico. Ele e aceito
somente quando o host da base URL e `api.openai.com`.

## Fluxo historico de request para aliases 1M (2026-07-01)

Fluxo normal para `/v1/chat/completions` com `gpt-5.4-1m`:

1. Cliente chama `POST /v1/chat/completions` com `model: "gpt-5.4-1m"`.
2. O router cria `RelayInfo` mantendo `OriginModelName = "gpt-5.4-1m"`.
3. A selecao de canal usa `abilities` do grupo/token e encontra o channel Codex
   `5`.
4. `model_mapping` do canal traduz o upstream para `gpt-5.4`.
5. `ModelMappedHelper` grava `UpstreamModelName = "gpt-5.4"` e marca
   `IsModelMapped = true`.
6. O request body enviado ao upstream recebe o modelo mapeado.
7. A politica `ShouldChatCompletionsUseResponsesPolicy` retorna `true` para
   qualquer channel Codex.
8. `chatCompletionsViaResponses` converte o payload OpenAI Chat Completions
   para OpenAI Responses.
9. Para Codex non-stream, o router forca stream upstream porque o backend
   OAuth/ChatGPT Codex real exige streaming. No modo OpenAI public API isso
   tambem e aceito, e o router agrega o SSE antes de devolver non-stream.
10. `OaiResponsesStreamToChatHandler` agrega o SSE do upstream e retorna uma
    resposta non-stream OpenAI-compatible quando o cliente pediu `stream:false`.
11. O campo publico `model` na resposta volta como `gpt-5.4-1m`, preservando o
    alias solicitado.
12. Usage, quota e billing usam `OriginModelName`, entao cobram o alias
    long-context, nao o modelo base.

Fluxo equivalente para o alias desabilitado `gpt-5.5-1m`:

```text
cliente model gpt-5.5-1m
  -> sem channel disponivel no grupo default
  -> HTTP 503 model_not_found
```

## Catalogo publico `/v1/models`

O catalogo publico e montado pelo backend Go:

- `controller/model.go`
- `service/modelcatalog/catalog.go`

A fonte do array `data` e a combinacao de:

- `abilities` habilitadas por grupo/token;
- rows em `models` para nome, descricao, endpoints e status;
- `setting/ratio_setting` para pricing;
- ownership preferido por modelo/canal.

O payload publico deve manter root simples:

```json
{
  "data": []
}
```

Nao expor no payload publico:

- `pricing_source`
- `pricing_estimated`
- `pricing_version`
- top-level `success`
- top-level `object`
- top-level pagination fields

Contrato final restaurado:

- `/v1/models` deve expor `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini` e
  `gpt-5.3-codex-spark`.
- `gpt-5.4` e o modelo Codex default para long-context.
- `gpt-5.4-1m` e `gpt-5.5-1m` nao devem aparecer no runtime final.

Exemplos do contrato publico final:

```json
{
  "id": "gpt-5.5",
  "object": "model",
  "created": 1626777600,
  "owned_by": "codex",
  "name": "OpenAI Codex GPT-5.5",
  "provider": "OpenAI Codex",
  "supported_endpoint_types": ["openai"],
  "endpoint_routes": {
    "openai": "/v1/chat/completions"
  },
  "pricing": {
    "input": 5,
    "output": 30
  }
}
```

```json
{
  "id": "gpt-5.4",
  "object": "model",
  "created": 1626777600,
  "owned_by": "codex",
  "name": "OpenAI Codex GPT-5.4",
  "provider": "OpenAI Codex",
  "supported_endpoint_types": ["openai"],
  "endpoint_routes": {
    "openai": "/v1/chat/completions"
  },
  "pricing": {
    "input": 5,
    "output": 22.5
  }
}
```

## Modelos e precos

O fork calcula preco publico token-priced a partir de:

```text
input_price = ModelRatio * 2
output_price = input_price * CompletionRatio
```

Para estes modelos, `quota_type = 0`. Nao usar `ModelPrice`, porque
`ModelPrice` representa outra semantica de cobranca.

| Modelo publico | Modelo upstream | ModelRatio | CompletionRatio | Input USD/1M | Output USD/1M | Observacao |
|---|---|---:|---:|---:|---:|---|
| `gpt-5.5` | `gpt-5.5` | 2.5 | 6 | 5 | 30 | preco standard |
| `gpt-5.4` | `gpt-5.4` | 2.5 | 4.5 | 5 | 22.5 | modelo Codex default para long-context no contrato final |
| `gpt-5.4-mini` | `gpt-5.4-mini` | 0.375 | 6 | 0.75 | 4.5 | preco standard |
| `gpt-5.3-codex-spark` | `gpt-5.3-codex-spark` | 0.875 | 8 | 1.75 | 14 | preco standard/estimado |

Historico 2026-07-01: os aliases `-1m` cobravam preco long-context desde o
primeiro token, mas essa variacao nao faz mais parte do runtime final da Phase
24.

## Billing, quota, usage e cache

O billing usa `OriginModelName`:

- `relay/helper/price.go` consulta `ratio_setting.GetModelRatio`,
  `GetCompletionRatio`, `GetCacheRatio` e familias relacionadas usando
  `info.OriginModelName`.
- `service/quota.go` registra debito e pre-consumo por
  `relayInfo.OriginModelName`.
- `service/text_quota.go` e `service/task_billing.go` tambem usam o modelo de
  origem para manter rastreabilidade.

Consequencia no contrato final:

- Cliente pede `gpt-5.4`.
- Upstream recebe `gpt-5.4`.
- Billing cobra o mesmo modelo solicitado.
- Logs internos continuam distinguindo modelo solicitado e upstream real quando
  houver mapping.

Uso retornado:

- Quando o upstream retorna `usage`, o router propaga `prompt_tokens`,
  `completion_tokens`, `total_tokens` e detalhes disponiveis.
- Quando usage nao vem completo, o router estima usage a partir do texto de
  resposta e tokens de prompt estimados.
- Tokens cached/reasoning sao preservados quando aparecem no payload upstream.

Cache billing:

- Nao ha campo publico novo criado para os aliases.
- Se cache ratio estiver configurado para algum modelo, a leitura continua pelo
  mesmo caminho de `OriginModelName`.
- Nao foi criado pricing especial de cached input para `gpt-5.5-1m` ou
  `gpt-5.4-1m`.

## Guardrails de contexto

Arquivo: `controller/codex_context_limit.go`.

Limites internos por modelo no contrato final:

| Modelo | Limite de input | Max output |
|---|---:|---:|
| `gpt-5.5` | 272000 tokens | nao definido no guard |
| `gpt-5.4` | 272000 tokens | nao definido no guard |

O guard roda antes de pre-consume/billing e antes de chamar upstream. Para
modelos base, isso preserva a limitacao original e evita enviar payload
claramente maior que o contrato standard.

Historico 2026-07-01: os aliases `gpt-5.5-1m` e `gpt-5.4-1m` chegaram a usar
guardrails de `1050000` input tokens e `128000` max output, mas isso nao faz
parte do runtime final restaurado pela Phase 24.

O estimador de contexto usa o maior valor entre:

- contagem do tokenizer local;
- estimativa conservadora baseada em palavras;
- tokens ja calculados pelo pipeline;
- overhead de mensagens/tools quando o formato e OpenAI.

Esse desenho foi adotado porque o tokenizer local pode subcontar payloads muito
grandes em relacao ao upstream real.

## Resultado dos testes 1M em 2026-07-01

O router-side foi validado como correto para:

- catalogo;
- grupos/permissoes;
- mapping;
- pricing;
- chat small non-stream;
- streaming;
- billing por alias;
- resposta publica preservando alias;
- erro upstream estruturado.

Resultado observado no upstream real:

| Modelo | Resultado |
|---|---|
| `gpt-5.5` | 250k aceito; 300k rejeitado localmente pelo guard standard |
| `gpt-5.4` | 250k aceito; 300k rejeitado localmente pelo guard standard |
| `gpt-5.5-1m` | upstream rejeitou 300k+ com `context_length_exceeded` |
| `gpt-5.4-1m` | passou 300k, 500k, 750k e 900k nominal; rejeitou 950k/1M com `context_length_exceeded` |

Validacao pos-build em 2026-07-01:

- Imagem local buildada e carregada pelo container:
  `079481f584d19335c9cb5fc7071ba14bbcce541a2424d39ecfac26c8283eae57`.
- `bin/clianything status --strict`: OK.
- `/v1/models` autenticado: OK para `gpt-5.5`, `gpt-5.5-1m`, `gpt-5.4`,
  `gpt-5.4-1m` e `gpt-5.4-mini`.
- `/v1/chat/completions` small non-stream: OK para `gpt-5.5`, `gpt-5.5-1m`,
  `gpt-5.4` e `gpt-5.4-1m`.
- `/v1/chat/completions` streaming pequeno: OK para `gpt-5.5-1m`.
- Guard pos-build dos modelos base em `300000`: `gpt-5.5` e `gpt-5.4`
  rejeitaram localmente com HTTP 400 antes do upstream.
- Alias pos-build `gpt-5.5-1m` em `300000`: HTTP 400 do upstream OAuth com
  `context_length_exceeded`.
- Checagem direta da API publica OpenAI com a `OPENAI_API_KEY` disponivel no
  ambiente retornou `401 invalid_api_key`; portanto nao ha credencial publica
  valida neste ambiente para fechar UAT 1M via `https://api.openai.com/v1/responses`.

Validacao apos desabilitar `gpt-5.5-1m` em 2026-07-01:

- Channel `5` `OpenAI - Codex` passou a expor:
  `gpt-5.5,gpt-5.4,gpt-5.4-1m,gpt-5.4-mini,gpt-5.3-codex-spark`.
- `channels.model_mapping` passou a manter apenas:
  `{"gpt-5.4-1m":"gpt-5.4"}`.
- `abilities.enabled=false` para `gpt-5.5-1m`.
- `models.status=0` para `gpt-5.5-1m`.
- `/v1/models` autenticado: `gpt-5.5-1m` ausente; `gpt-5.4-1m` presente.
- `/v1/chat/completions` com `gpt-5.4-1m`: HTTP 200.
- `/v1/chat/completions` com `gpt-5.5`: HTTP 200.
- `/v1/chat/completions` com `gpt-5.5-1m`: HTTP 503 `model_not_found`.

Conclusao operacional:

- O router esta preparado para os aliases `-1m` no catalogo, mapping, pricing,
  usage e billing.
- O channel OAuth/ChatGPT atual nao entrega 1M completo para ambos aliases.
- `gpt-5.5-1m` nao pode ser considerado validado para 1M enquanto apontar para
  o upstream OAuth/ChatGPT atual.
- `gpt-5.4-1m` demonstrou contexto muito maior que o padrao, mas a chamada de
  1M input nominal ainda falhou porque a janela total inclui output reservado e
  overhead do backend.
- A validacao 100% do requisito "ambos aliases aceitam 1M" exige um channel
  Codex/OpenAI publico com API key e permissao `responses.write`, usando
  `https://api.openai.com/v1/responses`.

## Comandos de validacao

Nunca inserir token real em arquivos. Usar variaveis de ambiente.

```bash
export ROUTER_BASE_URL="https://router.atius.com.br"
export ROUTER_TEST_KEY="COLOQUE_UM_TOKEN_DE_TESTE_AQUI"
```

Catalogo:

```bash
curl -s "$ROUTER_BASE_URL/v1/models" \
  -H "Authorization: Bearer $ROUTER_TEST_KEY" \
  | jq '.data[] | select(.id | test("^gpt-5\\.(5|4)(-mini)?$|^gpt-5\\.3-codex-spark$"))'
```

Small chat non-stream:

```bash
curl -s "$ROUTER_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $ROUTER_TEST_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "messages": [
      {
        "role": "user",
        "content": "Responda apenas OK."
      }
    ],
    "max_completion_tokens": 16,
    "stream": false
  }' | jq .
```

Streaming:

```bash
curl -N "$ROUTER_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $ROUTER_TEST_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.5",
    "messages": [
      {
        "role": "user",
        "content": "Conte de 1 a 5."
      }
    ],
    "max_completion_tokens": 64,
    "stream": true
  }'
```

Harness progressivo local:

```bash
ROUTER_BASE_URL="https://router.atius.com.br" \
ROUTER_TEST_KEY="$ROUTER_TEST_KEY" \
MODEL=all \
SIZES="small 10000 50000 100000" \
bash scripts/test-long-context-aliases.sh
```

Guard dos modelos base:

```bash
ROUTER_BASE_URL="https://router.atius.com.br" \
ROUTER_TEST_KEY="$ROUTER_TEST_KEY" \
MODEL=base \
SIZES="250000 300000" \
BASE_EXPECT_REJECT_FROM=300000 \
AUTO_CONFIRM_LARGE_STEPS=YES_I_ACCEPT_COSTS \
bash scripts/test-long-context-aliases.sh
```

Alias long-context, com custo real:

```bash
ROUTER_BASE_URL="https://router.atius.com.br" \
ROUTER_TEST_KEY="$ROUTER_TEST_KEY" \
MODEL=base \
SIZES="250000 300000" \
BASE_EXPECT_REJECT_FROM=300000 \
AUTO_CONFIRM_LARGE_STEPS=YES_I_ACCEPT_COSTS \
bash scripts/test-long-context-aliases.sh
```

## Variaveis relevantes para payload grande

Defaults atuais no codigo:

| Variavel | Default | Funcao |
|---|---:|---|
| `MAX_REQUEST_BODY_MB` | 128 | limite do body descomprimido |
| `STREAMING_TIMEOUT` | 300 | timeout de streaming em segundos |
| `STREAM_SCANNER_MAX_BUFFER_MB` | 128 | buffer maximo por linha SSE |
| `RELAY_IDLE_CONN_TIMEOUT` | 90 | idle keep-alive do relay HTTP |
| `RELAY_TIMEOUT` | 0 | timeout geral do relay; `0` significa sem override |

Para payloads reais perto de 1M, avaliar antes do teste:

- custo do request;
- tempo maximo esperado;
- tamanho do body;
- limites de proxy/load balancer;
- memoria do container;
- timeout do cliente.

Nao alterar essas variaveis direto em producao sem registrar impacto e rollback.

## Configuracao DB esperada

Consulta segura de resumo, sem secrets:

```bash
bin/clianything providers --all
```

Validacao SQL sem expor chaves:

```bash
podman exec postgres psql -U admin -d DBRouterAiAtius -Atc \
  "select id, name, type, status, models, model_mapping from channels where id=5;"

podman exec postgres psql -U admin -d DBRouterAiAtius -Atc \
  "select model, channel_id, \"group\", enabled from abilities where model in ('gpt-5.5','gpt-5.4','gpt-5.4-mini','gpt-5.3-codex-spark') order by model;"

podman exec postgres psql -U admin -d DBRouterAiAtius -Atc \
  "select model_name, description, endpoints, status from models where model_name in ('gpt-5.5','gpt-5.4','gpt-5.4-mini','gpt-5.3-codex-spark') order by model_name;"
```

Nunca consultar ou imprimir `channels.key` em logs compartilhados.

## Portainer, Podman e path do repo

Path atual do checkout operacional:

```text
/home/ubuntu/GitHub/containers/router-ai-atius/
```

Neste host, a validacao local foi feita pelo servico user systemd e Podman:

```bash
systemctl --user is-active container-router-ai-atius.service
bin/clianything status --strict
```

Se o container for operado via Portainer, manter o mesmo contrato:

- imagem do fork, nao `calciumion/new-api:latest`;
- volumes `data/` e `logs/` preservados;
- envs sensiveis fora de docs;
- DB sem migracao destrutiva;
- `/v1/` apontando para o backend Go do router.

## Checklist de manutencao

Antes de mudar modelos Codex:

1. Confirmar `bin/clianything providers --all`.
2. Confirmar `channels.models`, `channels.model_mapping`, `abilities` e
   `models`.
3. Confirmar que os modelos publicos aparecem em `/v1/models`.
4. Confirmar que o provider e `OpenAI Codex`, owner `codex`, route
   `/v1/chat/completions`.
5. Confirmar pricing via `/v1/models`.
6. Rodar small chat non-stream.
7. Rodar streaming smoke.
8. Confirmar logs com alias solicitado e upstream real.
9. Rodar testes Go focados quando houver mudanca de codigo.
10. Se for testar contexto grande, usar o harness com gates de custo.

## Testes automatizados relacionados

| Contrato | Teste |
|---|---|
| Contrato final Codex sem aliases `-1m` | `controller/model_list_test.go` |
| Guardrails de contexto | `controller/codex_context_limit_test.go` |
| Mapping alias -> upstream | `relay/helper/model_mapped_test.go` |
| Agregacao Responses SSE -> Chat non-stream | `relay/channel/openai/chat_via_responses_test.go` |
| Pricing ratios long-context | `setting/ratio_setting/model_ratio_test.go` |
| Politica Codex sempre via Responses | `service/openaicompat/policy_test.go` |
| Modo OAuth vs OpenAI public API do adaptor Codex | `relay/channel/codex/adaptor_test.go` |
| Harness shell seguro | `scripts/test_long_context_aliases_static_test.py` |

Comando focado:

```bash
PATH=/usr/local/go/bin:$PATH go test \
  ./controller \
  ./relay/channel/openai \
  ./relay/helper \
  ./setting/ratio_setting \
  ./service/openaicompat \
  ./relay/channel/codex \
  -run 'TestValidateCodexContextWindow|TestRequestModelNameFallsBackToOpenAIRequestModel|TestListModelsCodexContractAfterPhase24Restore|TestOaiResponsesStreamToChatHandlerAggregatesSSE|TestOaiResponsesStreamToChatHandlerPropagatesErrorEvent|TestModelMappedHelperCodexLongContextAliases|TestCodexLongContextAliasPricingRatios|TestShouldChatCompletionsUseResponsesPolicyAlwaysEnablesCodex|TestCodex' \
  -count=1
```

## Riscos e decisoes abertas

- O contrato final da Phase 24 nao publica `gpt-5.5-1m` nem `gpt-5.4-1m`.
- O historico de 2026-07-01 mostrou que o upstream OAuth/ChatGPT nao sustentou
  o experimento 1M de forma confiavel para ambos aliases.
- A API publica OpenAI e o caminho documentado para 1,05M, mas o channel runtime
  atual usa OAuth ChatGPT/Codex. Sem uma API key publica com permissao para
  `responses.write`, nao ha como fechar UAT 1M end-to-end fora do modelo base
  documentado.
- Qualquer mudanca para usar outro upstream, outro header beta ou outro modelo
  real precisa preservar:
  - visibilidade do alias solicitado;
  - billing pelo alias;
  - nao fallback para MiniMax, DeepSeek, Gemini, Claude ou OpenAI-compatible
    generico;
  - grupo `default`;
  - catalogo Go-owned;
  - secrets fora de docs/logs.

## Rollback conceitual

O rollback relevante para a Phase 24 e manter os modelos base e nao reintroduzir
os aliases `-1m` no runtime final. Se algum sync/revert voltar a expor esses
aliases, a correcao e:

1. Remover `gpt-5.5-1m` e `gpt-5.4-1m` de `channels.models` do channel `5`.
2. Limpar entradas equivalentes de `channels.model_mapping`.
3. Desabilitar/remover `abilities` dos aliases.
4. Desabilitar/remover rows dos aliases em `models`.
5. Remover ratios hardcoded dos aliases em `setting/ratio_setting/model_ratio.go`.
6. Revalidar `/v1/models`, small chat, billing e o contrato de embeddings governados.

Esse rollback nao deve remover nem renomear:

- `gpt-5.5`
- `gpt-5.4`
- `gpt-5.4-mini`
- `gpt-5.3-codex-spark`
