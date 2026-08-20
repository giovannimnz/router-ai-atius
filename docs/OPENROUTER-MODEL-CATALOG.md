# Contrato OpenRouter do catalogo de modelos

Estado revisado em 2026-08-17.

## Objetivo

`GET /v1/models` continua sendo implementado pelo backend Go e passa a publicar, alem dos campos historicos do Router, os metadados interoperaveis do catalogo OpenRouter que podem ser determinados localmente sem consulta live a terceiros.

Referencias primarias:

- OpenRouter Models API: <https://openrouter.ai/docs/api/api-reference/models/list-all-models-and-their-properties>
- OpenRouter Models guide: <https://openrouter.ai/docs/guides/overview/models>
- GTE embeddings: <https://huggingface.co/Alibaba-NLP/gte-multilingual-base>
- GTE reranker: <https://huggingface.co/Alibaba-NLP/gte-multilingual-reranker-base>

## Campos publicos

Cada item publica, quando conhecido:

- `id`, `canonical_slug`, `name`, `created`, `description`
- `context_length`, `architecture`, `top_provider`
- `pricing.prompt` e `pricing.completion` como strings em USD por token
- `pricing.request`, `pricing.image`, `pricing.input_cache_read` e `pricing.input_cache_write` quando aplicaveis
- `supported_parameters`, `default_parameters`, `per_request_limits`, `supported_voices`, `knowledge_cutoff` e `expiration_date`
- `links.details` somente quando existir uma rota local de endpoint-details compativel; hoje ele e omitido para nao anunciar link inexistente

Os campos Atius `object`, `owned_by`, `provider`, `context_window`, `supported_endpoint_types`, `endpoint_routes` e os campos legados de pricing sao mantidos para compatibilidade retroativa. `pricing.input` e `pricing.output` continuam em USD por 1M tokens e sao identificados por `unit=usd_per_1m_tokens`; `prompt` e `completion` usam `compatibility_unit=usd_per_token`.

Datas desconhecidas usam `created=0`. Nao usar o antigo placeholder global `1626777600` como se fosse a data real de lancamento.

## Filtros compativeis

O endpoint aceita os filtros OpenRouter que o catalogo local consegue responder deterministicamente:

- `q`
- `input_modalities`, `output_modalities`
- `supported_parameters`
- `context`
- `min_price`, `max_price`, `min_output_price`, `max_output_price`, em USD por 1M tokens
- `arch`, `model_authors`, `providers`
- `offset`, `limit`
- `sort=newest|context-high-to-low|pricing-low-to-high|pricing-high-to-low`

Sem query parameters, a resposta preserva a ordem governada do fork e inclui texto, embeddings e reranker. Ordenacoes que dependem de telemetria inexistente, como popularidade, throughput, latencia e indices externos, retornam HTTP `400` em vez de inventar valores.

## Divergencias deliberadas

O contrato local nao e um clone byte a byte do OpenRouter:

- O root permanece estritamente `{"data":[...]}`. `total_count` e `links` no root nao podem ser adicionados porque o contrato protegido do fork proibe campos top-level adicionais.
- O default nao filtra somente `output_modalities=text`; embeddings e reranker precisam permanecer descobriveis sem parametro adicional.
- `id` preserva o alias roteavel local sem prefixo de provider. `canonical_slug` fornece o identificador qualificado.
- Nao sao publicados scores, categorias, ZDR, regiao ou benchmarks que o Router nao mede. Filtros reconhecidos que dependem desses dados retornam HTTP `400` em vez de serem ignorados.
- Campos internos `pricing_source`, `pricing_estimated` e `pricing_version` nunca aparecem na API publica.

## GTE local

| Alias roteavel | Canonical slug | Contexto | Arquitetura | Rota |
|---|---|---:|---|---|
| `embedding-gte-v1` | `alibaba-nlp/gte-multilingual-base` | 8192 | `text->embeddings` | `/v1/embeddings` |
| `reranker-gte-v1` | `alibaba-nlp/gte-multilingual-reranker-base` | 8192 | `text->rerank` | `/v1/rerank` |

O alias legado `reranker-gte-multilingual-v1` e migrado de forma transacional no startup e nao e mais publicado nem aceito como habilidade ativa. Logs e agregados historicos tambem sao normalizados para `reranker-gte-v1`, evitando duas series para o mesmo modelo no dashboard.

Os dois modelos pertencem ao channel `Atius Local Embeddings`, type `59`, e executam no `horistic-srv`. O reranker deve anunciar endpoint type `jina-rerank`; anunciar `openai` ou `/v1/chat/completions` e incorreto.

No catalogo administrativo, o fornecedor canonico e `Atius Local`. Tanto a coluna `Icone` do modelo quanto o badge de `Fornecedor` usam a chave compartilhada `AtiusLocal`, resolvida para o mesmo SVG do channel `Atius Local Embeddings`; nao cadastrar um icone Lobe alternativo para estes dois campos.

## Contexto Codex pinado

`gpt-5.6-sol` publica `context_length=1000000` no channel `ChatGPT - Codex` e em `GET /v1/models`. Esse limite e uma politica deliberada do Router e vence o valor de contexto retornado pelo discovery OAuth somente para Sol. Terra e Luna continuam seguindo o contexto do discovery; `max_completion_tokens` continua sendo resolvido separadamente e preserva o valor OAuth quando presente.

## Auditoria de consumo USD do channel 11

O card de consumo usa `channels.used_quota / quota_per_unit`. Com `quota_per_unit=500000` e `used_quota=7111`, o valor nominal desde a criacao do channel consolidado e:

```text
7111 / 500000 = USD 0.014222
```

O total confere exatamente com logs normais: `883` quota de embeddings mais `6228` quota de reranker. Os `11` pontos adicionais presentes em logs de teste nao entram em `channels.used_quota`, por desenho do teste administrativo.

Esse numero e contabilidade nominal do Router, nao custo medido de energia, GPU ou operacao do `horistic-srv`. O embeddings usa ratio efetivo `0.035`; o reranker sem preco absoluto usa o fallback de self-use `37.5`, que domina o valor exibido. O catalogo torna esse preco efetivo transparente, mas esta mudanca nao redefine a politica financeira.

Os channels antigos foram encerrados com `128747880` e `11141` unidades, e o novo channel iniciou em zero. Uma soma historica meramente contabil seria `128766132 / 500000 = USD 257.532264`; ela nao deve substituir o card atual sem uma decisao explicita de migrar o contador historico.

## Validacao minima

```bash
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" \
  https://router.atius.com.br/v1/models

curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" \
  'https://router.atius.com.br/v1/models?output_modalities=embeddings'

curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" \
  'https://router.atius.com.br/v1/models?q=gte&sort=context-high-to-low'
```

Nunca registrar o bearer em documentacao, historico Git ou evidencias.
