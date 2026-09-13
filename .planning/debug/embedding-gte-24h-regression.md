---
slug: embedding-gte-24h-regression
status: fixed
trigger: "Investigar e corrigir preventivamente a degradacao atual de embedding-gte-v1: 90,77% de sucesso, 33,22s de latencia media e 68,09% do trafego nas ultimas 24h, preservando reranker local e governor Go-native."
created: 2026-08-28T03:20:00-03:00
updated: 2026-08-28T04:34:00-03:00
---

# Debug: regressao de 24h do embedding-gte-v1

## Symptoms

- expected: `embedding-gte-v1` deve permanecer operacionalmente estavel sob a carga local do GBrain/Graphify, sem cascatas de fila, falsos health guardrails ou perda de amostras.
- actual: janela de 24h reportada pelo dashboard mostra 90,77% de sucesso, 33,22s de latencia media e 68,09% do trafego; `reranker-gte-v1` aparece em 100%.
- error: classes e horarios exatos ainda precisam ser correlacionados entre `perf_metrics`, logs do Router, snapshot do governor, unit Podman e TEI/K3s no `horistic-srv`.
- timeline: regressao observada em 2026-08-28 sobre as ultimas 24 horas; incidentes semelhantes em 21/08 e 23/08 foram anteriormente corrigidos.
- reproduction: consultar a janela movel de 24h e correlacionar requests de `embedding-gte-v1` com runtime do Router e TEI; executar apenas smokes sinteticos sanitizados e limitados.

## Constraints

- Preservar dirty work preexistente e nao criar commit.
- Nao mascarar falhas reais nem adulterar a formula da metrica.
- Nao expor segredos, inputs de embedding ou payloads de usuario.
- Manter governor Go-native, channel 11 e TEI/reranker no `horistic-srv`.
- Toda tarefa CPU-heavy deve usar `scripts/podman-admin.sh` com limite de 20%.
- Qualquer mudanca live deve ter backup, rollback e validacao end-to-end.

## Current Focus

- hypothesis: um reset transitorio Router->TEI foi amplificado pelo governor porque todo 5xx entra em pressure, reduz a concorrencia a 1 e aplica cooldown global de 10 min; com demanda concorrente, os pedidos aguardam e expiram no timeout interativo de 30 s.
- test: comparar buckets de `perf_metrics` com journal do Router, configuracao efetiva do governor, codigo de classificacao de outcome e estado/live metrics do TEI/K3s.
- expecting: uma falha upstream inicial seguida de `embedding_governor_queue_timeout`, sem crash-loop atual de TEI; os KPIs 90,77%/33,22 s devem reconciliar como media global do painel, nao como medida por-modelo.
- next_action: monitor retained 24-hour history while validating new traffic and consumer-specific health
- reasoning_checkpoint:
    hypothesis: "O reset transitorio do upstream e a politica global de pressure/cooldown, sob demanda concorrente, causaram as 14 falhas secundarias de fila."
    confirming_evidence:
      - "Journal: reset por peer as 21:16:47 BRT, seguido por 14 rejeicoes 429 `embedding_governor_queue_timeout` entre 21:17:19 e 21:25:45 BRT."
      - "Codigo: status >=500 entra em `finishOutcomePressure`; `finish` configura concorrencia=min=1 e cooldown=10 min, enquanto acquire vence no timeout interativo de 30 s."
      - "Banco: bucket 21:00 tem 35 requests/20 sucessos; o total de 24 h de embeddings e 47/32 (68,09%)."
    falsification_test: "Uma execucao futura que mostre o reset sem cooldown/queue timeout, ou queue timeouts sem uma falha pressure/health/capacity anterior, invalida este encadeamento."
    fix_rationale: "Separar falha de transporte transitoria de pressure sustentada, ou exigir janela de falhas antes do cooldown global, impede que um unico reset descarte pedidos saudaveis sem remover os guardrails de health/capacidade."
    blind_spots: "Nao ha snapshot historico do governor nem serie historica do `te_queue_size`; a causa de baixo nivel do reset (rede versus processo upstream) nao e atribuivel com a evidencia retida."
    candidate_causes:
      - "environment: reset de conexao por peer no salto Router->TEI"
      - "code/config: todo 5xx e tratado como pressure global com cooldown de 10 min"
    and_gate: "yes — o reset inicial por si so produziu um 500; a regressao de 15 falhas exigiu tambem demanda concorrente durante o cooldown/timeout da fila."

## Evidence

- timestamp: 2026-08-28T03:21:30-03:00
  checked: Router Podman/systemd state and effective non-secret governor variables
  found: "router-ai-atius active since 2026-08-23 05:56:55 BRT, restart_count=0; health/capacity probes enabled for TEI; governed models are embedding-gte-v1 and reranker-gte-v1."
  implication: "The incident was not caused by a Router restart or disabled governor; health/capacity feedback was active."

- timestamp: 2026-08-28T03:25:00-03:00
  checked: live production `perf_metrics` through the Router SQL DSN (aggregate only)
  found: "24 h/default: embedding-gte-v1=47 requests, 32 successes, 68.09%, 11.33 s average; reranker-gte-v1=14/14, 100.00%, 61.01 s. The 21:00 BRT embedding bucket is 35 requests, 20 successes, 12.37 s."
  implication: "Exactly 15 embedding failures are concentrated in one hour; reranker did not share the failure mode."

- timestamp: 2026-08-28T03:25:00-03:00
  checked: Router journal, 2026-08-27 21:15-21:27 BRT, sanitised to model/channel/error class
  found: "At 21:16:47 BRT channel 11 received `read: connection reset by peer` from the TEI embeddings upstream and emitted one 500. It was followed by 14 `embedding_governor_queue_timeout before dispatch` 429s from 21:17:19 through 21:25:45 BRT."
  implication: "The queue timeouts are secondary governor rejections, not independent client validation failures."

- timestamp: 2026-08-28T03:26:00-03:00
  checked: governor outcome and acquire code in `service/embeddinggovernor/governor.go`
  found: "A >=500 outcome is `finishOutcomePressure`; pressure sets currentConcurrency=min (1) and cooldown=10 min. Acquire waits through cooldown and its interactive context expires after 30 s as `embedding_governor_queue_timeout`."
  implication: "A single upstream 500 can deterministically cascade into 429s when requests arrive during the cooldown."

- timestamp: 2026-08-28T03:27:00-03:00
  checked: live TEI service and K3s-agent/container runtime on horistic-srv via private SSH
  found: "Three independent /health probes to the embeddings service returned HTTP 200 in 0.9-2.2 ms; `te_queue_size` is 0; two embedding TEI pods/containers have remained SANDBOX_READY since before the incident; no matching embedding error/restart log was retained for the incident window."
  implication: "TEI is currently healthy and no evidence supports a sustained embedding pod outage; the low-level source of the earlier reset remains unproven."

- timestamp: 2026-08-28T03:28:00-03:00
  checked: performance panel implementation and per-model database values
  found: "The panel uses an unweighted simple average across models. Current per-model values 68.09/11.33, 95.00/55.20, 100.00/61.01, and 100.00/5.34 produce exactly 90.77% and 33.22 s."
  implication: "90.77% and 33.22 s are global panel KPIs, not embedding-gte-v1 KPIs; the model-specific degradation is 68.09% success."

- timestamp: 2026-08-28T03:28:00-03:00
  checked: TEI capacity-probe parser and current TEI Prometheus payload
  found: "The configured `/metrics` payload exposes `te_queue_size`; the parser treats any positive queue as 100% used and immediately blocks scale-up. Current queue is zero; no historic capacity snapshot is retained."
  implication: "Capacity feedback is a plausible additional amplifier under load, but was not proven active during this incident."

- timestamp: 2026-08-28T03:25:41-03:00
  checked: live concurrent reranker and embedding traffic during the investigation
  found: "A reranker request ended as `do_request_failed` after 126 s while its HPA scaled; 49 s later two otherwise healthy embedding requests expired as `embedding_governor_queue_timeout`. Embedding TEI pods stayed Ready and later served sub-second traffic."
  implication: "The single global cooldown coupled reranker pressure into embedding availability; the failure cascade was reproduced independently of the earlier 21:16 embedding reset."

- timestamp: 2026-08-28T04:30:00-03:00
  checked: candidate image build and controlled Podman cutover
  found: "Image `localhost/router-ai-atius:fix-embedding-halfopen-20260828` exists as ID `763238cfe99d...`, is the image running in `router-ai-atius`, and the backed-up previous image `1166573ec85d...` remains available. Unit env keeps both probes enabled and threshold 3."
  implication: "The source correction is active in production with an explicit rollback path."

- timestamp: 2026-08-28T04:32:00-03:00
  checked: direct and consumer smoke after cutover
  found: "Local/public status returned 200; `graphify-embed` returned 5/5 ordered vectors of 768 dimensions through the public Router; final readback found 22 post-cutover embedding consumes on channel 11, zero journal errors and 2.1 s average; Hermes connected to `gbrain_http`."
  implication: "Router, TEI, transparent 4+1 sub-batching and direct Graphify consumer path are healthy on the deployed image."

- timestamp: 2026-08-28T04:33:00-03:00
  checked: GBrain query consumer after Router smoke
  found: "`gbrain call query` exceeded 45 s with empty stdout, but the Router database recorded its embedding requests as successful in 0-1 s. GBrain status snapshot and queue were responsive."
  implication: "The remaining GBrain command timeout is downstream of embedding (query/database/rerank/client path), not a failure of `embedding-gte-v1`; track it separately without rolling back the Router fix."

- timestamp: 2026-08-28T04:36:00-03:00
  checked: GBrain full doctor, effective search-mode overrides and post-adjustment query
  found: "GBrain embedding provider passed in 524 ms with 768 dimensions and DB alignment. The query used `search.reranker.timeout_ms=45000`, while live local reranks took 57-126 s; an outer 45 s timeout killed the CLI before fail-open. After the reversible override `45000 -> 10000`, the same query returned rc=0 in about 14 s."
  implication: "GBrain, Obsidian content retrieval, Hermes and Codex calls through GBrain remain available even when optional local reranking is saturated; embedding required no consumer-side change."

## Eliminated

- hypothesis: "A sustained current TEI outage or Router crash-loop caused the full 24 h regression."
  evidence: "Router restart_count=0; TEI health is 200 on three probes, current queue=0, and both embedding pods are ready."

- hypothesis: "The reranker or a shared, continuous local-TEI outage explains the embedding errors."
  evidence: "Reranker has 14/14 successes in the same 24 h window; no matching embedding pod restart/error was retained at the incident time."

## Resolution

- root_cause: "AND-gate: a transient Router->TEI connection reset at 2026-08-27 21:16:47 BRT produced one upstream 500; the governor classifies every >=500 as global pressure, resets concurrency to 1 and applies a 10-minute cooldown. Concurrent embedding work then waited until the 30-second interactive timeout and generated 14 secondary 429 queue-timeout failures. The originating low-level cause of the reset is not determined from retained TEI/K3s evidence."
- fix: "Applied and deployed: pressure still reduces concurrency to min=1 and cooldown still blocks scale-up, but Acquire no longer blocks the minimum half-open slot during cooldown. Health guardrail continues to fail-fast sustained unhealthy upstreams. Runtime normalization now enforces health_bad_window_threshold >=3. The dashboard health KPIs now use request-count weighting. Active API/manual docs describe half-open cooldown and transparent upstream chunks of at most four. A stale Podman validator was aligned with the actual service name and Podman-first renderer."
- verification:
  - "Production DB aggregate reconciled model-specific 47/32=68.09% and the 21:00 bucket 35/20."
  - "Router journal established causal order: reset/500 then 14 queue-timeout/429 failures."
  - "Code path established the exact 500 -> pressure/min=1/cooldown=10m -> 30s queue-timeout mechanism."
  - "Live TEI/K3s checks show recovery: health 200 x3, queue 0, two embedding pods ready."
  - "UI calculation reproduces the reported global 90.77% and 33.22s from the four current model summaries."
  - "Focused tests passed for embeddinggovernor, relay and perf_metrics; KPI helper passed 2/2 and frontend typecheck/build passed."
  - "The deployed container image ID matches the inspected candidate; Podman config, runtime limits and container cgroups pass the canonical verifier."
- residual_risk: "The governor still shares one adaptive concurrency pool between embedding and reranker, although cooldown can no longer create a total cross-model blackout. Historical governor snapshots and TEI queue series are not persisted. GBrain 0.46.28 exports three embedding env controls that it does not consume, relies on Router sub-batching for arrays above four, and currently fails open from optional reranking after 10 s because reranker p95 is far above that budget."
- files_changed:
  - "service/embeddinggovernor/governor.go"
  - "service/embeddinggovernor/governor_test.go"
  - "relay/embedding_handler_test.go"
  - "web/default/src/features/performance-metrics/lib/summary.ts"
  - "web/default/src/features/performance-metrics/lib/summary.test.ts"
  - "web/default/src/features/dashboard/components/models/performance-overview.tsx"
  - "web/default/src/features/dashboard/components/overview/performance-health-panel.tsx"
  - "scripts/podman-validate.sh"
  - "docs/API.md"
  - "docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md"
  - ".planning/debug/embedding-gte-24h-regression.md"
  - "/home/ubuntu/.config/systemd/user/container-router-ai-atius.service (runtime image reference; backup retained)"
  - "GBrain DB config `search.reranker.timeout_ms` (runtime override 45000 -> 10000; rollback value recorded)"
