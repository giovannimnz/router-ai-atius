---
slug: reranker-errors-last-24h
status: resolved
trigger: "O dashboard de desempenho das ultimas 24 horas mostra reranker-gte-multilingual-v1 com 88,89% de sucesso, enquanto embedding-gte-v1 e gpt-5.4 mostram 100%; identificar a causa e propor/corrigir melhorias de confiabilidade, latencia e vazao."
created: 2026-08-14T13:55:00-03:00
updated: 2026-08-14T18:54:09-03:00
---

# Debug: erros do reranker nas ultimas 24 horas

## Symptoms

- expected: o alias governado `reranker-gte-multilingual-v1` deve responder de forma confiavel e previsivel para payloads validos, com erros de cliente separados de falhas do backend.
- actual: dashboard de 24 horas mostra sucesso de `88,89%` para o reranker; `embedding-gte-v1` e `gpt-5.4` mostram `100%`.
- error: o dashboard nao exibe o status/error code das chamadas que falharam.
- timeline: janela movel das ultimas 24 horas em 2026-08-14; reranker foi republicado na imagem principal no mesmo dia.
- reproduction: trafego real autenticado em `POST /v1/rerank`; o painel agrega o resultado em `https://router.atius.com.br/dashboard/overview`.

## Constraints

- Nao registrar tokens, keys, prompts/documentos sensiveis ou valores de secrets.
- Preservar todas as mudancas preexistentes da working tree compartilhada.
- Builds, compilacao, testes pesados e Graphify somente com teto de 20% da CPU total (`0.8 CPU`).
- Distinguir erro de payload/cliente, fila/governor, Router, rede e TEI antes de propor retry ou esconder falha.
- O backend TEI privado e o runtime em producao so podem sofrer mutacao depois de backup/readback e causa comprovada.

## Current Focus

- hypothesis: confirmada — um invalid_request HTTP 400 local foi contado como falha operacional; o incidente 503 de provisionamento/cache e separado.
- test: concluido por correlacao DB/log/runtime e leitura integral do handler e pipeline de perf_metrics.
- expecting: confirmado.
- next_action: sessao concluida; tratar a separacao de erros 4xx na metrica e o readiness de cache como tarefa dedicada, sem mutacao operacional nesta sessao.

reasoning_checkpoint:
  hypothesis: "RerankHelper rejeita payloads com mais de 20 documentos antes do governor/TEI, e controller/relay.go registra todo erro terminal como perf success=false; por isso um erro de cliente derruba a taxa operacional para 8/9."
  confirming_evidence:
    - "O registro individual da unica falha em perf_metrics e HTTP 400 invalid_request, 10,7ms, com mensagem de limite de 20 documentos."
    - "relay/rerank_handler.go retorna esse 400 antes de acquireRerankGovernor e antes de qualquer request upstream."
    - "controller/relay.go chama RecordRelaySample(relayInfo, false, 0) para qualquer newAPIError terminal."
    - "PerfMetric armazena apenas request_count e success_count; nao existe dimensao de status/client_error."
  falsification_test: "A hipotese seria falsa se o request tivesse chegado ao governor/TEI, se o status nao fosse invalid_request 400, ou se o pipeline excluisse 4xx client-side; nenhuma dessas condicoes ocorreu."
  fix_rationale: "Excluir somente invalid_request 400 client-side da metrica operacional preserva o error log separado e evita esconder 429/5xx/governor/upstream; um readiness gate apos cache sync evita os 503 de provisionamento."
  blind_spots: "Nenhum patch/teste foi aplicado por ordem do orquestrador; o ponto exato do workflow de provisionamento que deve esperar/forcar cache sync ainda precisa ser escolhido no slice de implementacao."

## Evidence

- timestamp: 2026-08-14T13:55:00-03:00
  source: dashboard reportado pelo usuario
  finding: taxa global 96,30%, latencia media 2,83s, vazao 2,57 t/s; reranker 88,89%, embeddings e gpt-5.4 100%.
- timestamp: 2026-08-14T13:55:00-03:00
  source: Graphify
  finding: grafo fresh no commit dc125b0 com 32618 nos e 78153 edges; query combinada do sintoma nao retornou nos, portanto a investigacao deve usar logs, DB e buscas focadas.
- timestamp: 2026-08-14T13:55:00-03:00
  source: GBrain, Obsidian e memoria Codex
  finding: caminho e Go-native/governado; alias usa canal 10 Advanced Custom, TEI privado em horistic-srv, limite de 20 documentos e backend historicamente sensivel ao tamanho do batch.
- timestamp: 2026-08-14T21:49:07.994Z
  source: git status
  finding: checkout na branch codex/ptbr-wayland-syncfix-20260814 esta ahead 4 e fortemente sujo; nenhum path conhecido de reranker/governor aparece modificado, e esta sessao de debug e untracked.
- timestamp: 2026-08-14T21:49:07.994Z
  source: Graphify status
  finding: grafo existe e esta fresh no commit dc125b0, sem commit_stale; consultas relacionais podem ser usadas sem rebuild CPU-heavy.
- timestamp: 2026-08-14T21:49:07.994Z
  source: GBrain CLI
  finding: consulta historica nao executou porque o cliente recusou o startup parameter statement_timeout; GBrain fica indisponivel nesta etapa e nao sera tratado como evidencia da causa.
- timestamp: 2026-08-14T21:50:07.088Z
  source: Graphify query
  finding: query por reranker/rerank/governor/TEI/dashboard/logs retornou zero nos e zero edges; o grafo nao roteia este sintoma e buscas focadas sao necessarias.
- timestamp: 2026-08-14T21:50:40.198Z
  source: bin/clianything status e providers --all
  finding: Router, DB e Podman estao saudaveis; canal 10 TEI - GTE Multilingual Reranker esta enabled com uma ability ativa e base privada http://10.21.1.21:31216. Isso elimina indisponibilidade permanente atual.
- timestamp: 2026-08-14T21:50:40.198Z
  source: GBrain MCP
  finding: busca pelo alias encontrou nota do incidente de 2026-08-14 confirmando que o reranker governado faz parte do contrato da imagem live; nao trouxe ainda o erro individual da janela.
- timestamp: 2026-08-14T21:51:16.965Z
  source: evidencia read-only fornecida pelo orquestrador
  finding: perf_metrics tem 9 requests do reranker nas ultimas 24h, 8 success e 1 failure; a falha foi HTTP 400 as 11:36:12 BRT por limite de 20 documentos, em 10,7ms, antes de governor/TEI. Cinco 503 No available channel ocorreram antes/ao redor da criacao e sync do canal 10 e nao foram contabilizados. Depois houve oito HTTP 200; TEI permaneceu Ready e sem restart/error/OOM.
- timestamp: 2026-08-14T21:52:57.297Z
  source: leitura de relay/rerank_handler.go
  finding: o limite maxGovernedTEIRerankDocuments=20 retorna ErrorCodeInvalidRequest HTTP 400 com skip-retry antes de calcular/acquirir governor ou executar o adaptor upstream.
- timestamp: 2026-08-14T21:52:57.297Z
  source: leitura de controller/relay.go e pkg/perf_metrics
  finding: todo newAPIError terminal apos relayInfo chama RecordRelaySample com success=false; Sample/PerfMetric preservam somente request_count e success_count, e QuerySummaryAll calcula success_count/request_count sem status ou classe de erro.
- timestamp: 2026-08-14T18:54:09-03:00
  source: documentacao operacional
  finding: diagnostico e proposta registrados em 20-PROJETOS/21-PROJETOS-ATIVOS/omni-srv-admin/2026-08-14-router-reranker-24h-88-89-debug.md no vault AiSecondBrain.

## Eliminated

- hypothesis: falha ou saturacao do TEI causou a taxa 88,89%.
  evidence: a unica falha contabilizada foi rejeitada localmente em 10,7ms antes do governor/upstream; dois pods TEI ficaram Ready, sem restarts/errors/OOM, e oito requests posteriores retornaram 200.
  timestamp: 2026-08-14T21:52:57.297Z
- hypothesis: ausencia persistente do canal 10 causou a taxa 88,89%.
  evidence: os cinco 503 ocorreram numa janela separada de provisionamento antes/ao redor do cache sync e nao entraram em perf_metrics; apos o sync o canal ficou enabled e houve oito sucessos.
  timestamp: 2026-08-14T21:52:57.297Z

## Resolution

- root_cause: "A metrica de Performance mistura erro de cliente com indisponibilidade: o fail-closed de >20 documentos retorna invalid_request HTTP 400 antes do governor/TEI, mas controller/relay.go registra qualquer erro terminal como success=false; sem classe/status no schema, o dashboard mostra 8/9 = 88,89%. Separadamente, cinco 503 vieram da janela entre provisionamento do canal 10 e sincronizacao do cache."
- fix: "Proposto, nao aplicado: adicionar classificacao focada no call site de perf_metrics para nao contar ErrorCodeInvalidRequest+HTTP 400 como tentativa operacional, mantendo RecordErrorLog como visibilidade separada; preservar 429/5xx/upstream/governor como falhas. Adicionar teste de regressao dessa matriz e gate/smoke de readiness apos cache sync no provisionamento."
- verification:
  - "DB/log correlacionados com o bucket de 11:00 BRT e o request HTTP 400 de 11:36:12."
  - "Fluxo de codigo confirma rejeicao antes do governor/TEI e gravacao incondicional de failure."
  - "Runtime atual saudavel; oito respostas 200 apos cache sync; TEI sem restarts/errors/OOM."
- files_changed: []
