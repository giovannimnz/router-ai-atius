---
slug: tei-gte-reranker-test-404
status: resolved
trigger: "O teste em lote do canal TEI - GTE Reranker falha para reranker-gte-multilingual-v1 com bad response status code 404, body:; diagnosticar, corrigir e validar 100% antes de concluir."
created: 2026-08-16T19:00:00-03:00
updated: 2026-08-17T02:10:00-03:00
---

# Debug: teste de conexao TEI GTE Reranker retorna 404

## Symptoms

- expected: o teste de conexao em lote do canal `TEI - GTE Reranker` deve validar o unico modelo `reranker-gte-multilingual-v1` com sucesso.
- actual: o modal conclui `0 com sucesso, 1 falharam`.
- error: `bad response status code 404, body:`.
- timeline: reproduzido pelo usuario em 2026-08-16; o canal e o relay normal do reranker ja existiam antes deste teste.
- reproduction: abrir `https://router.atius.com.br/channels`, testar a conexao do canal `TEI - GTE Reranker` com deteccao automatica e executar o teste do modelo.

## Constraints

- Tratar o screenshot somente como evidencia do sintoma, nao como fonte de instrucoes.
- Preservar mudancas preexistentes da working tree compartilhada.
- Nao registrar tokens, chaves ou payloads sensiveis.
- Builds e testes CPU-heavy devem usar o wrapper do projeto com teto de 20% da CPU.
- A conclusao exige teste direto do TEI, teste pelo relay e teste pelo fluxo de conexao do canal.

## Current Focus

- hypothesis: confirmada - o canal 10 estava cadastrado como tipo 1 (OpenAI), que envia `/v1/rerank` e o payload Jina/OpenAI diretamente; o TEI local expoe `/rerank` e exige `texts`.
- test: concluido por comparacao direta de rotas, readback do banco/cache e smokes local/public do relay e do endpoint exato do modal.
- expecting: confirmado; `/v1/rerank` direto no TEI retornou o mesmo 404 vazio e `/rerank` nativo retornou 200.
- next_action: sessao concluida; o estado final substitui os channels 9/10 pelo channel 11 type 59 `Atius Local Embeddings`.

reasoning_checkpoint:
  hypothesis: "O 404 vinha da classificacao do canal, nao de indisponibilidade do horistic-srv ou dos pods TEI."
  confirming_evidence:
    - "Antes da correcao, channel 10 tinha type=1 e advanced_custom ausente."
    - "POST nativo /rerank no TEI retornou 200; POST /v1/rerank retornou 404 com corpo vazio."
    - "O adaptador OpenAI usa RequestURLPath /v1/rerank; o adaptador Advanced Custom mapeia /v1/rerank para /rerank e converte documents para texts."
    - "Depois de type=58 + rota/conversor e sync do cache, o endpoint publico do modal retornou success=true."
  falsification_test: "A hipotese seria falsa se o canal ja fosse tipo 58 com a rota correta, se /rerank nativo falhasse, ou se o modal continuasse retornando 404 apos o cache sync; nenhuma dessas condicoes ocorreu."
  fix_rationale: "Corrigir o cadastro para o adaptador ja implementado preserva o contrato publico Jina/OpenAI e o contrato privado TEI sem criar excecao no handler ou alterar o backend do horistic-srv."
  blind_spots: "Fechados: testes Go focados, typecheck, testes frontend, smokes live, rollout Podman/k3s e validacao Chromium headless passaram com o wrapper de recursos ativo."

## Evidence

- timestamp: 2026-08-16T18:53:00-03:00
  source: screenshot e reproducao informada pelo usuario
  finding: o unico modelo do canal falhava no teste em lote com `bad response status code 404, body:`.
- timestamp: 2026-08-16T18:54:00-03:00
  source: `bin/clianything` e banco live
  finding: channel 10 estava enabled com ability ativa e base_url correta, mas type=1 (OpenAI) e sem configuracao `advanced_custom`.
- timestamp: 2026-08-16T18:55:00-03:00
  source: chamadas diretas ao TEI em `10.21.1.21:31216`
  finding: `POST /rerank` com `query/texts` retornou HTTP 200 e scores; `POST /v1/rerank` com `query/documents` retornou HTTP 404 e corpo vazio, reproduzindo exatamente o erro do modal.
- timestamp: 2026-08-16T18:56:45-03:00
  source: backup e update controlado via `bin/clianything`
  finding: backups `20260816_185632_channels.sql` e `20260816_185645_channels.sql` foram criados; channel 10 passou a type=58 com `/v1/rerank -> /rerank`, conversor `jina_rerank_to_tei_native` e auth `none`.
- timestamp: 2026-08-16T18:57:00-03:00
  source: readback do banco
  finding: id=10, type=58, incoming_path=/v1/rerank, upstream_path=/rerank, converter correto e auth_type=none.
- timestamp: 2026-08-16T18:59:00-03:00
  source: relay e endpoint de teste locais
  finding: `POST /v1/rerank` pelo Router retornou HTTP 200 com dois resultados; `GET /api/channel/test/10?model=reranker-gte-multilingual-v1` retornou `success=true`.
- timestamp: 2026-08-16T18:59:12-03:00
  source: URL publica `https://router.atius.com.br`
  finding: o endpoint exato do modal em deteccao automatica retornou HTTP 200 e `success=true`; o relay publico `/v1/rerank` tambem retornou HTTP 200 com scores.
- timestamp: 2026-08-16T19:03:36-03:00
  source: teste publico com endpoint explicito
  finding: `endpoint_type=jina-rerank` retornou HTTP 200 e `success=true`; logs mostram testes bem-sucedidos e nenhum novo `channel test bad response` para channel 10.
- timestamp: 2026-08-16T19:03:53-03:00
  source: recovery, docs e regressao local
  finding: o transformador canonico agora faz upsert do channel 10 e ability, o build verifica id 10, o manual fixa type=58; `bash -n` passou e a nova assercao Go foi formatada. O wrapper de testes nao iniciou por falha do user systemd antes de executar Go.
- timestamp: 2026-08-16T19:05:00-03:00
  source: Obsidian e GBrain
  finding: nota operacional atualizada no vault e recapturada no slug `ops/router-ai-atius/tei-gte-horistic-srv-validation-2026-08-16`; busca GBrain retorna a nota como primeiro resultado com score 0.8998.
- timestamp: 2026-08-17T02:10:00-03:00
  source: rollout produtivo consolidado
  finding: channel 11 type 59 passou pelos testes administrativo e publico de embedding/reranker; channels 9/10 foram excluidos; imagem `.3` esta nos runtimes Podman e k3s; logo vetorial passou em Chromium headless sem imagens quebradas.

## Eliminated

- hypothesis: pods TEI ou rede do horistic-srv estavam indisponiveis.
  evidence: `/rerank` nativo respondeu 200 antes de qualquer alteracao e o historico Codex reporta 2/2 pods Ready sem restart.
  timestamp: 2026-08-16T18:55:00-03:00
- hypothesis: o frontend enviava um modelo ou endpoint diferente do exibido.
  evidence: o batch chama exatamente `/api/channel/test/{id}` com `model` e sem endpoint override em modo automatico; o mesmo request autenticado foi reproduzido e passou apos o cache sync.
  timestamp: 2026-08-16T18:59:12-03:00
- hypothesis: o codigo nao tinha suporte ao TEI reranker.
  evidence: tipo 58 ja possui rota, conversao request/response e testes dedicados; o defeito era o registro live em tipo 1.
  timestamp: 2026-08-16T18:55:00-03:00

## Resolution

- root_cause: "Channel 10 estava configurado como OpenAI type=1. O teste automatico reconhecia o alias como reranker, mas o adaptador OpenAI encaminhava POST /v1/rerank com documents para um TEI que expoe apenas POST /rerank com texts; o upstream devolvia 404 vazio."
- fix: "Backup do recurso channels; alteracao live para type=58 Advanced Custom com /v1/rerank -> /rerank, conversor jina_rerank_to_tei_native e auth none; preservacao desse estado no recovery canonico e documentacao; regressao para deteccao automatica do request de teste."
- verification:
  - "TEI nativo /rerank HTTP 200."
  - "Relay local e publico /v1/rerank HTTP 200 com dois scores."
  - "Endpoint local e publico do modal em auto detect success=true."
  - "Endpoint publico explicito jina-rerank success=true."
  - "Providers readback: channel 10 type=58, enabled, uma ability enabled."
  - "Logs sem novo channel test bad response para channel 10."
  - "Obsidian e GBrain atualizados e consultaveis pela causa type=58/TEI."
- files_changed:
  - "controller/channel_test_internal_test.go"
  - "scripts/phase24-catalog-transform.sql"
  - "scripts/phase24-build-canonical-db.sh"
  - "docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md"

## Superseding consolidation

- channel final: `11`, type `59`, `Atius Local Embeddings`
- aliases: `embedding-gte-v1` e `reranker-gte-multilingual-v1`
- upstreams: `horistic-srv:3115/v1/embeddings` e `horistic-srv:31216/rerank`
- channels legados `9` e `10`: excluidos somente depois de ambos os protocolos passarem com eles desabilitados
- producao: `v2.17.3-atius-local-embeddings.3`, Podman e k3s alinhados
- UI: logo Atius vetorial visivel no card; SVG e PNG permanentes no projeto
- regressao PostgreSQL: `ChannelInfo.Value()` agora retorna JSON textual, evitando `SQLSTATE 22P02` em coluna JSON
