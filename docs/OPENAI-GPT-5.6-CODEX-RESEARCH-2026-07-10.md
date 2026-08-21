# OpenAI GPT-5.6 / Codex Research - 2026-07-10

## Contexto

Pesquisa exploratoria para entender a chegada do `gpt-5.6` no ecossistema Codex/OpenAI e o impacto no fork `router-ai-atius`.

Escopo:

- confirmar o que existe oficialmente no OpenAI Docs/API
- mapear o que o router ja suporta sem mudanca
- listar o que precisa mudar antes de dizermos que suportamos `gpt-5.6` corretamente

## Confirmado no OpenAI Docs

### Familia de modelos

Fontes oficiais:

- [Model guidance - Migration quickstart](https://developers.openai.com/api/docs/guides/latest-model#update-api-and-model-parameters)
- [Building agents - How to choose](https://developers.openai.com/tracks/building-agents#how-to-choose)
- [Data controls - endpoint/model support](https://developers.openai.com/api/docs/guides/your-data#api-endpoint-tool-and-model-support)

Achados:

- existem `gpt-5.6-sol`, `gpt-5.6-terra` e `gpt-5.6-luna`
- o alias `gpt-5.6` roteia para `gpt-5.6-sol`
- OpenAI recomenda:
  - `gpt-5.6-sol` para capability frontier
  - `gpt-5.6-terra` para equilibrio entre inteligencia e custo
  - `gpt-5.6-luna` para workloads de alto volume e menor latencia
- para interfaces conversacionais, a recomendacao oficial e usar `gpt-5.6-terra` para chat recorrente e delegar para `gpt-5.6` nos casos mais pesados

### Endpoints

Fonte oficial:

- [Data controls - endpoint/model support](https://developers.openai.com/api/docs/guides/your-data#api-endpoint-tool-and-model-support)

Achados:

- `gpt-5.6-sol`, `gpt-5.6-terra` e `gpt-5.6-luna` aparecem em:
  - `/v1/responses`
  - `/v1/chat/completions`
  - `/v1/batches`
- nao ha mudanca em embeddings por causa dessa familia; embeddings seguem `text-embedding-3-*`

### Novidades de API associadas ao GPT-5.6

Fontes oficiais:

- [Model guidance - Migration quickstart](https://developers.openai.com/api/docs/guides/latest-model#update-api-and-model-parameters)
- [Responses create](https://api.openai.com/v1/responses)
- [Chat Completions create](https://api.openai.com/v1/chat/completions)
- [Prompt caching - breakpoints](https://developers.openai.com/api/docs/guides/prompt-caching#prompt-cache-breakpoints)
- [Programmatic Tool Calling](https://developers.openai.com/api/docs/guides/tools-programmatic-tool-calling#configure-programmatic-tool-calling)

Achados:

1. O path principal recomendado passa a ser `Responses API`.
2. `reasoning.effort` em GPT-5.6 suporta `none`, `low`, `medium`, `high`, `xhigh` e `max`.
3. Existe `reasoning.mode = "pro"` no Responses API, sem trocar para outro slug de modelo.
4. Existe `reasoning.context` com `auto`, `all_turns` e `current_turn`.
5. Para `store: false` ou ZDR, o fluxo stateful oficial usa `include: ["reasoning.encrypted_content"]`.
6. `prompt_cache_options` e `prompt_cache_breakpoint` passam a existir para `gpt-5.6` e familias posteriores.
7. `programmatic_tool_calling` + `allowed_callers` + `output_schema` passam a ser parte do contrato oficial.
8. O changelog oficial tambem cita multi-agent orchestration em beta no Responses API.
9. A familia GPT-5.6 aceita imagens em dimensao original com `detail=original` ou `detail=auto`.

## O que o router ja suporta sem mudar

### Discovery dinamica do Codex

Arquivos:

- [service/codex_catalog.go](/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_catalog.go:320)
- [service/codex_catalog.go](/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_catalog.go:756)

Leitura:

- a discovery do channel 5 nao depende de lista estatica no request path
- se o upstream `backend-api/codex/models` entregar slugs `gpt-5.6-*`, o pipeline atual tende a descobri-los
- a promocao no catalogo continua gated por probe `Ok`

### Responses raw tools pass-through parcial

Arquivos:

- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:860)
- [relay/helper/valid_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/relay/helper/valid_request.go:115)

Leitura:

- `OpenAIResponsesRequest.Tools` e `ToolChoice` sao `json.RawMessage`
- isso significa que uma parte de novos tools/shape pode passar direto no `/v1/responses`
- nao ha validacao estrita do shape desses campos no parser atual

### Image detail

Arquivos:

- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:301)
- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:966)

Leitura:

- o parser local ja carrega `detail` como string em `image_url` e `input_image`
- `detail=original` e `detail=auto` nao parecem exigir mudanca estrutural imediata

### Effort max provavelmente passa

Arquivos:

- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:950)
- [service/relayconvert/chat_to_responses.go](/home/ubuntu/GitHub/containers/router-ai-atius/service/relayconvert/chat_to_responses.go:394)
- [relay/channel/openai/adaptor.go](/home/ubuntu/GitHub/containers/router-ai-atius/relay/channel/openai/adaptor.go:584)

Leitura:

- `Reasoning.Effort` e `ReasoningEffort` sao strings livres
- `max` nao e bloqueado por enum local
- entao `reasoning.effort=max` tende a passar no path Responses, desde que o upstream aceite

## O que precisa mudar antes de declararmos suporte correto

### 1. Catalogo/local policy de Codex ainda esta preso em 5.5/5.4

Arquivos:

- [service/codex_catalog.go](/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_catalog.go:97)
- [relay/channel/codex/constants.go](/home/ubuntu/GitHub/containers/router-ai-atius/relay/channel/codex/constants.go:8)
- [docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md:85)

Problema:

- fallback e overrides locais ainda conhecem apenas `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark`
- `gpt-5.4` continua como default local
- a lista estatica do adaptor Codex nao cita nenhum `gpt-5.6-*`
- docs operacionais ainda listam so a familia antiga

Impacto:

- mesmo com discovery dinamica, metadata, ordering, default, docs e fallback continuam desatualizados

### 2. O schema local nao modela novidades-chave do GPT-5.6

Arquivos:

- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:829)
- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:950)
- [dto/openai_response.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_response.go:270)

Problema:

- falta `prompt_cache_options`
- falta `prompt_cache_breakpoint`
- `Reasoning` so tem `effort` e `summary`
- faltam `reasoning.mode` e `reasoning.context`
- faltam estruturas explicitas para `allowed_callers` e `output_schema`
- faltam tipos explicitamente modelados para `program`, `program_output` e afins

Impacto:

- requests com esses campos podem ser descartados localmente
- responses stateful/tool-rich podem perder metadata nas camadas que fazem parse/transformacao

### 3. Conversoes `responses <-> chat` perdem stateful features novas

Arquivos:

- [service/relayconvert/responses_request_to_chat.go](/home/ubuntu/GitHub/containers/router-ai-atius/service/relayconvert/responses_request_to_chat.go:92)
- [service/relayconvert/responses_request_to_chat.go](/home/ubuntu/GitHub/containers/router-ai-atius/service/relayconvert/responses_request_to_chat.go:97)

Problema:

- a conversao de Responses para Chat rejeita explicitamente:
  - `previous_response_id`
  - `context_management`
- esses dois campos agora sao parte importante do contrato stateful recomendado para GPT-5.6

Impacto:

- o path `/v1/chat/completions` compatibilizado via Responses nao acompanha os recursos novos de persisted reasoning

### 4. Programmatic Tool Calling esta incompleto no nosso shape local

Arquivos:

- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:229)
- [dto/openai_response.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_response.go:340)

Problema:

- `ToolCallRequest` atual nao inclui `allowed_callers`
- nao inclui `output_schema`
- `ResponsesOutput` atual entende `message`, `function_call` e poucos tipos; nao modela `program`/`program_output`

Impacto:

- o `/v1/responses` direto pode deixar algumas coisas passarem, mas nao temos parity segura para parse, telemetry, transformacao e future tooling

### 5. Prompt caching oficial novo nao existe no request local

Arquivos:

- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:76)
- [dto/openai_request.go](/home/ubuntu/GitHub/containers/router-ai-atius/dto/openai_request.go:851)

Problema:

- ainda carregamos `prompt_cache_retention`
- nao existe `prompt_cache_options`
- nao existe suporte a `prompt_cache_breakpoint` nos blocos

Impacto:

- nao conseguimos expor nem validar o contrato novo de cache do GPT-5.6

## Recomendacao objetiva para o router

### Fase 1 - baixo risco

1. Atualizar discovery/catalogo Codex para reconhecer `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`.
2. Decidir politica local para o alias `gpt-5.6`:
   - opcao conservadora: esconder alias e publicar apenas os 3 slugs explicitos
   - opcao compativel com docs: publicar `gpt-5.6` como alias de `sol`
3. Manter o default local atual ate prova live.
4. Atualizar docs e pricing placeholders.

### Fase 2 - parity de request/response

1. Adicionar em DTOs:
   - `prompt_cache_options`
   - `prompt_cache_breakpoint`
   - `reasoning.mode`
   - `reasoning.context`
   - `allowed_callers`
   - `output_schema`
2. Atualizar tipos de output Responses para `program`/`program_output`.
3. Revisar `responses_request_to_chat` para nao descartar stateful fields sem escolha explicita.

### Fase 3 - validacao live

1. Rodar discovery real no channel 5.
2. Validar `Ok` probe para `sol`, `terra`, `luna`.
3. Medir:
   - latencia
   - custo
   - quality delta
   - comportamento com `reasoning.effort=none/low/medium/high/xhigh/max`
4. Testar explicitamente:
   - `reasoning.mode=pro`
   - `reasoning.context=all_turns/current_turn`
   - `prompt_cache_options`
   - `store=false` + `include=["reasoning.encrypted_content"]`

## Recomendacao de produto para o fork

- se for publicar rapido: `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna` como novos modelos Codex curados e sem trocar o default
- se for buscar melhor UX default depois da validacao:
  - `terra` como conversational default
  - `sol` como frontier/esforco alto
  - `luna` como fast/cheap lane

## Leitura final

Hoje o repo esta bem posicionado para descobrir e promover os novos slugs `gpt-5.6-*`, mas ainda nao esta pronto para declarar suporte completo aos recursos novos do GPT-5.6. O maior delta nao e o nome do modelo: e a camada de stateful reasoning, prompt caching moderno e Programmatic Tool Calling.
