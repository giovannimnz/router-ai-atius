# Phase 33 Context — Reranker reliability, observability, and readiness

## Origem

Nas últimas 24 horas de 2026-08-14, o dashboard mostrou `88,89%` para `reranker-gte-multilingual-v1`. A investigação reconciliou Graphify, banco, runtime, GBrain e Obsidian:

- o valor era `8/9`, causado por uma única rejeição local `HTTP 400 invalid_request` para mais de 20 documentos;
- a rejeição ocorreu antes do governor e do TEI; os dois pods TEI permaneceram saudáveis e o tráfego válido teve sucesso;
- cinco falhas `503 No available channel` pertenciam a uma janela distinta de provisionamento/cache e não entraram na métrica;
- o dashboard exibiu a média simples das porcentagens dos três modelos, não uma agregação ponderada por volume.

## Decisões travadas

- Preservar `request_success_rate`, mas nunca apresentá-la sozinha como saúde do TEI.
- Adicionar taxonomia explícita para client/local validation, governor, routing, transport/timeout, upstream e success.
- Medir falhas sem canal antes da seleção do relay, com uma única amostra terminal por request.
- Calcular indicadores gerais a partir dos contadores subjacentes, de forma ponderada.
- Exigir sync/readiness do cache/ability antes do smoke de um canal recém-criado ou reativado.
- Manter o limite de 20 documentos; orientar sub-batching no cliente preservando índices.
- Propagar context/deadline ao request TEI e limitar retry a uma única repetição apenas para transporte/5xx explicitamente seguros.
- Não promover candidato sem o reranker governado completo, canal 10 e os dois aliases do governor.

## Invariantes de produção

- `relay/channel/advancedcustom/tei_rerank.go` precisa existir na imagem candidata.
- O canal DB `id=10` e o alias `reranker-gte-multilingual-v1` permanecem ativos.
- `EMBEDDING_GOVERNOR_MODELS` contém `embedding-gte-v1,reranker-gte-multilingual-v1`.
- O runtime live que incorporou o baseline `deb39c92d` não pode ser substituído por imagem baseada somente em main/quota que remova essa capacidade.
- Rollback deve estar disponível e nenhuma evidência pode registrar payload, documento, token ou segredo.

## Superfícies prováveis

- Backend: `pkg/perf_metrics/`, `model/perf_metric.go`, `controller/relay.go`, `controller/perf_metrics.go`, `relay/rerank_handler.go`, `relay/channel/api_request.go` e provisioning/cache de channels.
- Frontend: `web/default/src/features/performance-metrics/` e `web/default/src/features/dashboard/components/overview/performance-health-panel.tsx`.
- Persistência cross-repo: guards, protected paths e preflight do `omni-srv-admin`.
- Docs/runbooks: Router, GBrain e Obsidian.

## Fora de escopo

- Remover ou enfraquecer o embedding/rerank governor.
- Fazer sub-batching transparente no Router sem design específico de recomposição, ranking e billing.
- Fazer cutover público k3s das Phases 29/30.
- Alterar identidade protegida do upstream.

## Evidência de contexto

- `.planning/debug/reranker-errors-last-24h.md`
- GBrain `ops/router-ai-atius/reranker-88-89-debug-2026-08-14`
- Obsidian `20-PROJETOS/21-PROJETOS-ATIVOS/omni-srv-admin/2026-08-14-router-reranker-24h-88-89-debug.md`
- Graphify rebuild de 2026-08-14: `37.577` nós, `78.439` arestas, `3.220` comunidades, fresh em `dc125b0`.

## Done condition do planejamento

- Research e pattern map completos.
- Planos executáveis com requirements, waves, testes, rollback e critérios live.
- Validation/Nyquist cobrindo os riscos críticos.
- Plan checker sem blockers.
- ROADMAP, STATE, GBrain, Obsidian e Graphify sincronizados.
