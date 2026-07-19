# OpenAI Codex OAuth: contexto e pricing canônico no Router

Estado validado em 2026-07-19 para o channel `OpenAI - Codex`, type `57`.

## Contrato canônico

O channel type `57` usa OAuth/ChatGPT contra `chatgpt.com/backend-api/codex`. Ele
nao e um channel OpenAI Platform autenticado por API key. Os dois contratos nao
podem compartilhar automaticamente contexto, output limit ou pricing.

| Modelo | `context_window.context_length` | `max_completion_tokens` | Referencia Standard USD/1M input/output |
|---|---:|---|---:|
| `gpt-5.6-sol` | `272000` | `128000` (fallback oficial) | `5 / 30` |
| `gpt-5.6-terra` | `272000` | `128000` (fallback oficial) | `2.5 / 15` |
| `gpt-5.6-luna` | `272000` | `128000` (fallback oficial) | `1 / 6` |
| `gpt-5.5` | `272000` | `128000` (fallback oficial) | `5 / 30` |
| `gpt-5.3-codex-spark` | `128000` | omitido | nao publicado na API |

`context_length` representa a janela total do contrato publico do Router. A
resolucao e feita campo a campo, nesta ordem obrigatoria:

1. se o discovery do channel OAuth Codex ativo publicar contexto e limite de
   saida, os dois valores do OAuth vencem;
2. se o discovery publicar somente contexto, o Router conserva esse contexto
   e completa apenas `max_completion_tokens` pela pagina oficial do modelo
   referenciada pela tabela Standard da OpenAI;
3. valores da OpenAI Platform nunca substituem um campo que o OAuth publicou.

O discovery ativo de 2026-07-19 publica `context_window=272000`, mas nao publica
`max_output_tokens` nem `max_completion_tokens`. As paginas oficiais de Sol,
Terra, Luna e 5.5 publicam `Max output: 128,000 tokens`; portanto, o contrato
resultante combina `context_length=272000` do OAuth com
`max_completion_tokens=128000` da referencia oficial.

Payload esperado para GPT-5.6 no `/v1/models`:

```json
{
  "id": "gpt-5.6-terra",
  "owned_by": "codex",
  "provider": "OpenAI Codex",
  "context_window": {
    "context_length": 272000,
    "max_completion_tokens": 128000
  },
  "billing_mode": "dollar_cost",
  "pricing": {
    "input": 2.5,
    "output": 15,
    "cached_input": 0.25,
    "cache_write": 3.125,
    "unit": "usd_per_1m_tokens",
    "prompt": 0.0000025,
    "completion": 0.000015,
    "compatibility_unit": "usd_per_token",
    "scope": "openai_api_standard_reference"
  }
}
```

O transporte upstream continua autenticado por OAuth/ChatGPT, mas o settlement
interno do Router usa `billing_mode=dollar_cost`. `InputPrice` e `OutputPrice`
sao precos absolutos em USD por 1M tokens. `CacheRatio` e `CreateCacheRatio`
guardam, respectivamente, `cached_input/input` e `cache_write/input`; os quatro
valores alimentam a cobranca, `/pricing`, `/models/metadata` e `/v1/models`.
Isso elimina a divergencia entre o valor cobrado e o valor anunciado aos
clientes. Limites de plano/credits do ChatGPT continuam
sendo uma restricao separada do upstream.

## Coleta automatica de pricing

O Router coleta a tabela diretamente do endpoint Markdown oficial
`https://developers.openai.com/api/docs/pricing.md`:

- refresh imediato no startup do master;
- refresh diario as `04:00` em `America/Sao_Paulo` (`UTC-3` no estado atual);
- `Accept: text/markdown`, limite de resposta de 1 MiB e `ETag`/`If-None-Match`;
- parser restrito ao primeiro bloco `Standard`; ele preserva as linhas validas
  para acompanhar futuros modelos descobertos pelo Codex;
- consulta complementar das paginas oficiais dos modelos presentes na tabela
  Standard para extrair `max output tokens`; falha de qualquer pagina rejeita a
  promocao parcial e conserva o ultimo snapshot valido;
- promocao atomica: Sol, Terra, Luna e 5.5 sao obrigatorios, e qualquer modelo
  ausente, duplicado ou valor invalido rejeita o payload inteiro;
- snapshot valido persistido em `CodexOpenAIReferencePricing` na tabela
  `options`;
- merge atomico dos modelos Codex em `InputPrice`, `OutputPrice`, `CacheRatio`
  e `CreateCacheRatio`, preservando todos os outros providers;
- escrita dos mapas somente quando algum valor efetivo mudou; `304 Not
  Modified` conserva os valores e tambem repara eventual drift local usando o
  ultimo snapshot valido;
- a atualizacao de input/output/cache e do snapshot usa uma unica transacao,
  row locks/CAS e um lock local, impedindo billing com metade antiga e metade
  nova;
- depois do commit, `InputPrice`, `OutputPrice`, `CacheRatio` e
  `CreateCacheRatio` tambem sao publicados no runtime sob um unico write lock;
  o settlement le os quatro campos sob um unico read lock;
- remover um preco direto apaga somente `InputPrice`/`OutputPrice`; ratios de
  cache so mudam quando enviados explicitamente, preservando a troca segura
  entre `dollar_cost` e o modo de ratio;
- fallback embarcado igual a tabela oficial de 18/07/2026 quando ainda nao
  existe snapshot persistido.

`input`, `output`, `cached_input` e `cache_write` usam
`unit=usd_per_1m_tokens`. `prompt` e `completion` sao aliases de compatibilidade
em `usd_per_token`; eles evitam que clientes Hermes antigos multipliquem um
valor ja expresso por milhao e produzam avisos de custo 1.000.000 vezes maiores.
Spark permanece sem `pricing`, pois nao existe preco oficial de API publicado.
Como os valores ficam no PostgreSQL externo, nenhuma alteracao futura de preco
exige rebuild da image ou novo arquivo dentro do container.

## Persistencia de limites

Os limites OAuth efetivos ficam em `codex_catalog_candidates`, nas colunas
`context_window_tokens`, `max_tokens` e `max_completion_tokens`. Overrides
operacionais ficam em `options.CodexCatalogMetadataOverrides`; portanto, uma
correcao futura de limite tambem nao exige rebuild. O snapshot
`options.CodexOpenAIReferencePricing` guarda ainda o limite oficial de saida.
O payload final sempre resolve os campos separadamente: o valor OAuth vence;
somente a lacuna recebe o valor oficial persistido.

## Contrato API-key separado

As paginas da OpenAI Platform informam `1.050.000` tokens de contexto, `922.000`
de input e `128.000` de output para GPT-5.6 Sol, Terra e Luna. O contexto
Platform nao substitui o discovery OAuth de `272000`; apenas o campo de saida
ausente e completado com `128000`. Os precos Standard da faixa ate `272K` sao
usados como tabela canonica em USD do Router, com
`scope=openai_api_standard_reference`.

Fontes oficiais:

- [Codex CLI 0.144.6: contexto corrigido para 272.000](https://learn.chatgpt.com/docs/changelog#month-2026-07)
- [Codex pricing: plano/credits versus API key](https://learn.chatgpt.com/docs/pricing)
- [Codex auth: API key usa pricing padrao da Platform](https://learn.chatgpt.com/docs/auth#sign-in-with-an-api-key)
- [OpenAI API pricing: tabela Standard por 1M tokens](https://developers.openai.com/api/docs/pricing)
- [OpenAI API pricing em Markdown, fonte do collector](https://developers.openai.com/api/docs/pricing.md)
- [GPT-5.6 Sol na API](https://developers.openai.com/api/docs/models/gpt-5.6-sol)
- [GPT-5.6 Terra na API](https://developers.openai.com/api/docs/models/gpt-5.6-terra)
- [GPT-5.6 Luna na API](https://developers.openai.com/api/docs/models/gpt-5.6-luna)

## Hermes Agent

O Router publica `prompt` e `completion` em USD/token para Hermes antigos e
mantem `input` e `output` em USD/1M para o contrato humano do catalogo. Com os
valores oficiais Standard, nenhum modelo Codex atual ultrapassa os thresholds
de `20 USD/M` de input ou `100 USD/M` de output do warning do Hermes.

Para normalizar os contextos de uma configuracao Hermes sem reformatar o YAML:

```bash
python3 scripts/normalize-hermes-codex-metadata.py \
  --config "$HOME/.hermes/config.yaml" \
  --default gpt-5.6-sol

python3 scripts/normalize-hermes-codex-metadata.py \
  --config "$HOME/.hermes/config.yaml" \
  --default gpt-5.6-sol \
  --write
```

O primeiro comando e check-only e retorna `1` quando existe drift. `--write`
cria backup em `~/.hermes/backups/codex-model-metadata/` antes da escrita.

## Gates de validacao

1. `gpt-5.4` e `gpt-5.4-mini` nao aparecem no `/v1/models` quando discovery os marca `visibility: hide`.
2. Sol, Terra, Luna e 5.5 retornam `context_window.context_length=272000`.
3. Spark retorna `context_window.context_length=128000`.
4. Sol, Terra, Luna e 5.5 publicam `max_completion_tokens=128000` enquanto o OAuth nao informar outro limite; se informar, o valor OAuth vence.
5. Sol, Terra, Luna e 5.5 publicam o pricing Standard oficial, unidades e `scope`; Spark fica sem preco inventado.
6. Modelos com `InputPrice` e `OutputPrice` publicam `billing_mode=dollar_cost`, e o settlement usa os mesmos valores de input/output/cache do catalogo.
7. A selecao no Hermes nao exibe `Expensive Model Warning` para esses modelos.
8. Uma segunda coleta sem mudanca nao regrava nenhum mapa de preco nem o snapshot.
9. `GET /v1/models/{id}` aplica os mesmos filtros de grupo, token model-limit e billing de `GET /v1/models`; um modelo fora do catalogo visivel retorna `model_not_found`.
