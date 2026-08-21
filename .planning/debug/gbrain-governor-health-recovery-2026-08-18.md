# Debug: GBrain health 100% apos recuperacao do governor/runtime

status: resolved
opened: 2026-08-18
surface: gbrain/models-doctor + router/runtime

## Sintoma

O GBrain deixou de completar consultas com embeddings locais. Os sinais visiveis
foram:

- `gbrain models doctor` oscilando nos probes de embeddings e reranker.
- `gbrain query` falhando com
  `query embed deadline 20000ms exceeded`.
- Router retornando `429 embedding governor queue timeout before dispatch`.
- Em algumas tentativas, o request ficou preso por minutos e terminou em
  `408 embedding governor request was canceled before dispatch`.

## O que nao estava quebrado

- O endpoint TEI local de embeddings em `http://10.21.1.21:3115/embed`
  respondeu 200 de forma direta.
- O endpoint TEI local de rerank em `http://10.21.1.21:31216/rerank`
  respondeu 200 de forma direta.

Conclusao: o problema nao era o `horistic-srv` nem os modelos GTE em si.

## Causa raiz confirmada

O runtime do Router/rootless Podman entrou em estado ruim apos carga pesada de
rerank e fila do governor. Havia evidencia de:

- fila interna sem despacho efetivo,
- conexao persistente presa do container do Router para o endpoint local de
  embeddings,
- container de storage rootless travado impedindo restart limpo do servico.

Com isso, o GBrain estava configurado corretamente, mas dependia de um caminho
de runtime degradado entre Router e TEI local.

## Correcao executada

- Confirmado que o canal produtivo canonico e `Atius Local Embeddings`.
- Consolidado o nome do reranker para `reranker-gte-v1` no fork e nos testes.
- Ajustado o GBrain para:
  - `search.reranker.top_n_in = 5`
  - `search.reranker.timeout_ms = 45000`
- Recuperado o runtime rootless do Podman:
  - remocao de artefato `Storage` travado,
  - limpeza direcionada do mount preso,
  - `systemctl --user reset-failed container-router-ai-atius.service`,
  - `systemctl --user start container-router-ai-atius.service`.

## Validacao final

- `gbrain models doctor --json`:
  - `total = 6`
  - `ok = 6`
  - `failed = 0`
- Router:
  - `/v1/embeddings` voltou a responder 200 em ~0,60s
  - `/v1/rerank` voltou a responder 200 em ~0,84s
- `gbrain query` voltou a completar sem o erro de timeout de embeddings.
- `./scripts/podman-admin.sh status` mostrou backend HTTP 200 e canal
  `Atius Local Embeddings` ativo com:
  - `embedding-gte-v1`
  - `reranker-gte-v1`

## Formato e tamanho das chamadas

- GBrain chat:
  - modelo logico: `litellm:gpt-5.4-mini`
  - endpoint do Router: `POST /v1/chat/completions`
  - formato: OpenAI-compatible chat completions
- GBrain expansion:
  - modelo logico: `anthropic:gpt-5.4-mini`
  - endpoint do Router: `POST /v1/messages`
  - formato: Anthropic Messages
- GBrain embeddings:
  - modelo logico: `openai:embedding-gte-v1`
  - endpoint do Router: `POST /v1/embeddings`
  - formato: OpenAI-compatible embeddings
- GBrain reranker:
  - modelo logico: `llama-server-reranker:reranker-gte-v1`
  - endpoint do Router: `POST /v1/rerank`
  - formato: contrato Router rerank, convertido internamente para TEI nativo
- Graphify:
  - backend `openai` usa OpenAI-compatible chat completions
  - backend `claude` usa Anthropic Messages
  - backend logico `deepseek` agora tambem usa `gpt-5.4-mini` no Router via
    OpenAI-compatible chat completions

## Tuning final aplicado

- Wrapper do GBrain agora limpa variaveis antigas de DeepSeek para impedir
  drift por ambiente herdado.
- Wrappers do Graphify passaram a fixar explicitamente:
  - `DEEPSEEK_MODEL=gpt-5.4-mini`
  - `GRAPHIFY_DEEPSEEK_MODEL=gpt-5.4-mini`
- O worker de jobs do GBrain deixou de rodar solto em shell antiga e passou a
  ser mantido por `systemd --user` como `gbrain-jobs-worker.service`.
- O endpoint HTTP MCP do GBrain foi reiniciado pelo mesmo caminho gerenciado e
  herdando o wrapper canonico.
- Rerank pesado do GBrain foi reduzido no cliente com:
  - `search.reranker.top_n_in = 3`

## Impacto medido

- Antes do tuning final do reranker:
  - `4302` prompt tokens
  - `104s` em `/v1/rerank`
- Depois do tuning final:
  - `2376` prompt tokens
  - `62s` em `/v1/rerank`

Conclusao: o governor atual ficou saudavel; o ajuste necessario nesta etapa foi
reduzir o payload encaminhado ao reranker, nao mudar a logica central do
governor.

## Risco residual

Durante a recuperacao houve limpeza de artefatos globais de container no host.
Isso pede verificacao separada das stacks rootless nao relacionadas ao Router
para garantir que continuem exatamente no estado esperado.

## Complemento - taxa de sucesso em 18/08/2026

- O painel de 24h ainda refletia falhas antigas, mas os sinais ao vivo do TEI
  estavam saudaveis:
  - `te_request_success == te_request_count`
  - `te_queue_size = 0`
- Causa raiz confirmada para `embedding-gte-v1` no historico:
  - `400 embedding-gte-v1 supports at most 4 input items per request`
  - ocorrencias em `17/08/2026 16:25` e `17/08/2026 16:26`
- Causa raiz confirmada para `reranker-gte-v1` no historico:
  - `503 Nenhum canal disponível para modelo reranker-gte-multilingual-v1`
  - ocorrencias em `17/08/2026 17:26`, `18:04` e `20:07`
- Mitigacoes aplicadas no codigo do Router:
  - alias legado no distribuidor:
    `reranker-gte-multilingual-v1 -> reranker-gte-v1`
  - sub-batching transparente no handler de embeddings governados acima de 4
    inputs, com recomposicao da resposta OpenAI-compatible
- Mitigacoes operacionais ja presentes no cliente:
  - `~/.graphify/embeddings.json` usa `max_batch_size = 4`
  - `~/.local/bin/gbrain` exporta `GBRAIN_EMBED_PROVIDER_BATCH_SIZE = 4`
- Implicacao pratica:
  - a taxa do painel tende a subir sozinha conforme a janela movel de 24h
    expulsa esses erros antigos
  - as novas mudancas evitam repetir exatamente esses dois modos de falha
