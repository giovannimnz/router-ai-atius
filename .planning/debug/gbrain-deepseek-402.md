# Debug: GBrain usando DeepSeek sem saldo

status: resolved
opened: 2026-08-17
surface: dashboard/overview

## Sintoma

O painel de producao exibia `deepseek-v4-flash` com taxa de sucesso de 0%.

## Evidencias

- Os 13 registros das ultimas 48 horas para `deepseek-v4-flash` eram erros HTTP 402.
- A resposta upstream era `Insufficient Balance`.
- Todos usavam o token `GBrain Graphify`, o canal `DeepSeek` e a rota `/v1/messages`.
- O log do GBrain associa os mesmos request IDs a falhas de expansao de consulta.
- A configuracao efetiva mantinha `chat_model` e `expansion_model` em
  `anthropic:deepseek-v4-flash`.
- O Graphify CLI ja estava configurado com backend OpenAI-compatible e
  `gpt-5.4-mini`; ele nao foi o emissor direto desses erros.

## Hipotese confirmada

O GBrain herdou uma configuracao antiga de expansao/chat via Anthropic e enviava
consultas ao modelo DeepSeek sem saldo. A correcao deve mudar ambos os modelos
para `openai:gpt-5.4-mini`, reiniciar o servico residente e validar uma consulta
real com expansao.

## Validacao previa

Uma chamada direta em producao a `/v1/chat/completions` com `gpt-5.4-mini`
retornou com sucesso `ATIUS_GPT54_MINI_OK`.

## Correcao

- GBrain `chat_model` e `models.chat`: `litellm:gpt-5.4-mini`, usando o
  transporte OpenAI-compatible em `/v1/chat/completions`.
- GBrain `expansion_model` e `models.expansion`:
  `anthropic:gpt-5.4-mini`, usando `/v1/messages`.
- O wrapper do GBrain nao possui mais fallback para DeepSeek.
- O wrapper do Graphify fixa `gpt-5.4-mini` nos backends OpenAI, Claude e
  DeepSeek redirecionado ao Router.
- Os fallbacks ativos do Hermes foram migrados para `gpt-5.4-mini`.
- O registro de precos do GBrain recebeu os aliases do Router para que o
  controle de orcamento nao bloqueie chat, expansion ou embeddings.

## Validacao final

- `gbrain models doctor --skip=zeroentropyai`: 5/5 modelos alcancaveis.
- Consulta GBrain com expansion Anthropic: sucesso.
- Brainstorm GBrain com chat OpenAI-compatible: sucesso, 12 ideias.
- Graphify OpenAI: 7 nos, 9 arestas.
- Graphify Anthropic: 7 nos, 9 arestas.
- Graphify invocado como DeepSeek, mas redirecionado: 7 nos, 8 arestas.
- Logs do Router confirmaram `gpt-5.4-mini` em `/v1/chat/completions`,
  `/v1/messages` e `/v1/responses`.
- O ultimo request DeepSeek permaneceu o erro antigo `107847`, em
  2026-08-17 21:21:55 BRT; nenhum novo request foi gerado depois da correcao.
