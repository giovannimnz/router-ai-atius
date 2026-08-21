---
slug: embedding-gte-success-rate
status: fixed
trigger: "Identificar e corrigir por completo a queda da taxa de sucesso do embedding-gte-v1 de 100% para 53,33% em producao."
created: 2026-08-21T05:00:00-03:00
updated: 2026-08-21T06:25:00-03:00
---

# Debug: queda da taxa de sucesso do embedding-gte-v1

## Symptoms

- expected: embedding-gte-v1 deve atender payloads validos com 100% de sucesso operacional, como ocorria anteriormente.
- actual: dashboard de producao mostra taxa de sucesso de 53,33% para embedding-gte-v1.
- error: os codigos e classes das falhas ainda precisam ser correlacionados em perf_metrics, logs do Router, governor e TEI.
- timeline: queda significativa observada em 2026-08-21; o modelo ja apresentou 100% anteriormente.
- reproduction: consultar a janela movel de 24 horas em https://router.atius.com.br/dashboard/overview e correlacionar requisicoes de embedding-gte-v1 com logs e runtime local TEI no horistic-srv.

## Constraints

- Nao mascarar erros de cliente como sucessos nem perseguir 100% apenas alterando a formula da metrica.
- Distinguir payload invalido, fila/governor, Router, rede, capacidade e TEI antes de aplicar retry.
- Preservar o limite publico e o sub-batching governado existente.
- Nao registrar tokens, API keys ou textos de embedding sensiveis.
- Preservar todas as alteracoes preexistentes da working tree compartilhada.
- Builds e testes pesados devem usar o wrapper com limite de 20% da CPU total.

## Current Focus

- resolved: true
- conclusion: "o incidente combinou sete falhas reais durante o reboot/aquecimento da unica replica TEI com perda de nove amostras de sucesso do bucket corrente durante restart do Router. O governor estava sem probes em producao e continuou admitindo fila para um upstream indisponivel; perf_metrics mantinha o bucket atual apenas em memoria."
- production_state: "probes de health/capacity ativos no endpoint HA 31115, admissao fecha rapidamente com 503 coerente quando o Service fica unhealthy, flush do bucket atual executado antes e depois do drain HTTP, e dois pods TEI fixados por imagem/revisao no horistic-srv."

## Evidence

- timestamp: 2026-08-21T06:24:00-03:00
  checked: imagem final, runtime Podman, banco, K3s e trafego publico depois do restart de producao
  found: imagem `localhost/router-ai-atius:prod-20260821-ui-embedding-ha` (`5f8a7b71461e`) esta running; o binario contem o endpoint `31115`; canal 11 preserva embeddings em `31115` e reranker em `31216`; os dois pods estao Ready, sem restart, e o EndpointSlice tem dois backends. O smoke final passou em 10/10 chamadas sequenciais, 4/4 concorrentes, lote de 5 recomposto em 5 vetores de 768 dimensoes e reranker com 2 resultados. O bucket atual registra 273/273 sucessos; a janela movel de 24h registra 382/389 (98,20%) porque preserva as sete falhas reais antigas.
  implication: o caminho atual esta em 100% desde a remediacao; o percentual historico nao deve ser adulterado e chegara a 100% quando o bucket antigo sair da janela de 24h.

- timestamp: 2026-08-21T05:26:50-03:00
  checked: carga publica continua durante rollout completo do deployment `tei-gte`
  found: 300 de 300 chamadas autenticadas a `POST /v1/embeddings` retornaram payload OpenAI `object=list` com vetor de 768 dimensoes; o teste atravessou a substituicao das duas replicas antigas, mantendo apenas um endpoint Ready durante cada warm-up de aproximadamente dez minutos.
  implication: perda e recriacao controlada de um pod nao interrompem mais o modelo publico nem derrubam sua taxa de sucesso recente.

- timestamp: 2026-08-21T05:16:00-03:00
  checked: deployment, pods, Endpoints, PDB, ResourceQuota e logs TEI em `ebeddings-local`
  found: deployment `tei-gte` concluiu `2/2`, ambos os pods sem restart, Service privado `10.21.1.21:31115` com dois endpoints, PDB `minAvailable=1`, imagem fixa `sha256:16c0a827...`, modelo fixo em `9bbca17d...`; warm-up observado abaixo do limit de 8Gi por pod.
  implication: o risco de processo, pod e rollout foi mitigado sem alterar a exigencia de hospedagem no `horistic-srv`; o risco de perda fisica do host continua explicitamente fora desta HA de pods.

- timestamp: 2026-08-21T04:43:42-03:00
  checked: canal 11 no Postgres, unit Podman e ConfigMap shadow K3s
  found: canal `Atius Local` migrou de `3115` para `31115` em `base_url` e somente na rota advanced de embeddings, preservando o reranker em `31216`; probes live e shadow usam `/health` e `/metrics` no NodePort HA. Uma migration idempotente protege restores com o endpoint legado.
  implication: banco, defaults, recovery SQL, frontend de canais e os dois caminhos de runtime nao podem divergir silenciosamente de volta ao listener retirado.

- timestamp: 2026-08-21T06:21:00-03:00
  checked: host `horistic-srv`, `last -x`, `k3s crictl inspect`, `k3s crictl logs`
  found: o host reiniciou as `2026-08-20 23:57-03`; o container antigo de embeddings terminou as `23:57:54`; o novo container `tei-gte-69f755946f-lw79q` iniciou as `23:58:01`, mas so registrou `Starting HTTP server: 0.0.0.0:3115` e `Ready` as `2026-08-21 00:07:34-03`.
  implication: o `connect refused` das `00:00:11` aconteceu enquanto o novo TEI ainda nao aceitava conexoes; a indisponibilidade real do upstream durou tempo suficiente para explicar toda a rajada inicial de falhas do router.

- timestamp: 2026-08-21T06:24:00-03:00
  checked: unit live `container-router-ai-atius.service`, `podman inspect router-ai-atius`, env do processo `/new-api`
  found: a unit live passa apenas `EMBEDDING_GOVERNOR_ENABLED=true`, `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1,reranker-gte-v1` e `EMBEDDING_GOVERNOR_BATCH_MODELS=`; os envs `EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED/URL` e `EMBEDDING_GOVERNOR_CAPACITY_PROBE_ENABLED/URL` nao sao enviados ao container, embora `k8s/router-ai-atius/configmap.yaml` ja os defina.
  implication: o governor tinha guardrails implementados em codigo, mas estava cego em producao Podman por deriva de configuracao entre os caminhos k3s e rollback/live.

- timestamp: 2026-08-21T06:12:00-03:00
  checked: canais 9/10/11 no Postgres e logs da janela 2026-08-21 00:00-02:00
  found: apenas o canal 11 existe e esta enabled como `Atius Local Embeddings` em `http://10.21.1.21:3115`; todos os 7 erros e os sucessos subsequentes de `embedding-gte-v1` na janela do incidente foram gravados com `channel_id=11`.
  implication: a hipoteses de roteamento por canal legado 9/10 ou divergencia de cache para outro upstream perde forca; o incidente aconteceu no caminho canonico do canal 11.

- timestamp: 2026-08-21T06:14:00-03:00
  checked: reachability atual e host remoto `horistic-srv`
  found: `10.21.1.21:3115` responde `HTTP/1.1 200 OK` em `/health` e aceita TCP atualmente, mas o proprio host `horistic-srv` reporta `up 2 hours, 43 minutes` as `2026-08-21T02:40:14-03:00`, indicando reboot por volta de `2026-08-20 23:57-03`, minutos antes da sequencia de falhas das `00:00`.
  implication: existe uma causa operacional plausivel e temporalmente alinhada no host do TEI; agora e preciso confirmar como o reboot afetou a disponibilidade/arranque do servico de embeddings em 3115 e por que o governor nao fechou admissao logo apos o primeiro connect refused.

- timestamp: 2026-08-21T06:07:00-03:00
  checked: git diff -- main.go pkg/perf_metrics/flush.go pkg/perf_metrics/flush_test.go service/embeddinggovernor/governor.go service/embeddinggovernor/governor_test.go relay/embedding_handler.go relay/embedding_handler_test.go
  found: ha edits locais interrompidos em dois grupos distintos: (1) flush explicito de perf_metrics no shutdown para persistir o bucket corrente; (2) uma mudanca separada em relay/embedding_handler para sub-batching transparente acima de 4 inputs e alinhamento do nome governado do reranker.
  implication: o primeiro grupo trata apenas a durabilidade da metrica; o segundo e mais amplo que o incidente e nao pode ser assumido como remediacao validada das 7 falhas reais sem prova causal adicional.

- timestamp: 2026-08-21T05:05:00-03:00
  checked: .planning/debug/knowledge-base.md
  found: arquivo de knowledge base nao existe; nao ha padrao resolvido reaproveitavel para este sintoma.
  implication: a investigacao precisa partir do codigo e das metricas atuais, sem atalho por incidente conhecido.

- timestamp: 2026-08-21T05:06:00-03:00
  checked: busca textual por embedding-gte-v1, perf_metrics, embeddinggovernor e success_rate
  found: o caminho governado de embeddings passa por relay/embedding_handler.go e service/embeddinggovernor/governor.go; a telemetria de taxa de sucesso referencia perf_metrics em controller/model/perf_metrics e no frontend do dashboard.
  implication: a queda de 53,33% pode ser explicada por eventos registrados no handler/governor e agregados em perf_metrics, nao apenas por indisponibilidade do TEI.

- timestamp: 2026-08-21T05:09:00-03:00
  checked: relay/embedding_handler.go, controller/relay.go, service/quota.go, service/text_quota.go e pkg/perf_metrics/metrics.go
  found: perf_metrics incrementa sucesso apenas apos PostConsumeQuota/PostTextConsumeQuota em respostas completas; qualquer newAPIError final no relay grava amostra de falha via controller/relay.go.
  implication: o dashboard mede taxa de sucesso final do request para embedding-gte-v1 e necessariamente cai com rejeicoes do governor, erros upstream/TEI e outros erros finais do relay.

- timestamp: 2026-08-21T05:12:00-03:00
  checked: artefatos locais em logs/, runtime/ e data/
  found: existem logs recentes do router, inclusive oneapi-20260816153359.log com eventos do canal 9 "TEI - GTE Embeddings"; ainda nao ha classificacao da janela exata de 2026-08-21.
  implication: ha evidencias operacionais locais suficientes para testar a hipotese por classe de erro sem depender apenas da leitura do codigo.

- timestamp: 2026-08-21T05:15:00-03:00
  checked: logs recentes do router para embedding-gte-v1
  found: em 2026-08-16/17 ha falhas reais do caminho governado com status 429 "embedding governor queue timeout before dispatch", 408 "embedding governor request was canceled before dispatch", 500 por EOF no POST /v1/embeddings e sinais de indisponibilidade do backend TEI via connection refused; tambem ha 400 por input array > 4 itens em 2026-08-17.
  implication: ja existe evidencia direta de que a taxa de sucesso do modelo pode cair por saturacao/indisponibilidade operacional do caminho governado, embora ainda falte confirmar a distribuicao na janela de 24h exibida em 2026-08-21.

- timestamp: 2026-08-21T05:18:00-03:00
  checked: logs/oneapi-20260820085112.log, logs/oneapi-20260820085339.log, logs/oneapi-20260820235631.log e logs/oneapi-20260821014837.log
  found: a atividade de embedding-gte-v1 na janela mais proxima do incidente esta concentrada em logs/oneapi-20260820235631.log e continua em logs/oneapi-20260821014837.log; os dois arquivos anteriores de 2026-08-20 nao trazem entradas do modelo.
  implication: a reconstrução do incidente deve partir do burst entre 2026-08-21 00:00 e 01:53, sem dispersar para arquivos irrelevantes.

- timestamp: 2026-08-21T05:21:00-03:00
  checked: contagem de record consume log e record error log para embedding-gte-v1 em logs/oneapi-20260820235631.log e logs/oneapi-20260821014837.log
  found: na amostra reconstruida ha 12 sucessos e 7 falhas; 6 falhas sao 429 "embedding governor queue timeout before dispatch" entre 00:00:44 e 00:05:29 de 2026-08-21, precedidas por 1 falha 500 as 00:00:11 por connect refused no POST para http://10.21.1.21:3115/v1/embeddings.
  implication: a classe de erro dominante no incidente local nao e payload invalido; e uma cascata operacional iniciada por indisponibilidade do TEI/upstream e amplificada por timeout de fila no governor.

- timestamp: 2026-08-21T05:24:00-03:00
  checked: service/embeddinggovernor/governor.go (Acquire, finish, classifyFinishOutcome, cooldown/reject logic)
  found: falhas 500/429 entram como finishOutcomePressure, forcam currentConcurrency para MinConcurrency e ativam cooldown; porem requests ja enfileirados continuam aguardando ate expirar em 30s e viram 429 de queue timeout. Erros 4xx de cliente nao contam como pressure.
  implication: o padrao observado em log e coerente com o codigo; ainda falta saber se probes de health/capacity deveriam ter impedido a admissao enquanto o TEI estava indisponivel.

- timestamp: 2026-08-21T05:26:00-03:00
  checked: LoadConfigFromEnv e documentacao operacional do governor
  found: probes de health e capacity sao disabled-by-default no codigo; a documentacao tambem os descreve como opt-in e mostra exemplos habilitados apenas quando configurados explicitamente.
  implication: se o processo em producao nao recebeu envs explicitos de probe, o governor nao tinha mecanismo preventivo de health/capacity para fechar admissao antes da fila expirar.

- timestamp: 2026-08-21T05:29:00-03:00
  checked: /proc/468420/environ do processo /new-api em execucao no container router-ai-atius
  found: o processo ativo expoe apenas EMBEDDING_GOVERNOR_ENABLED=true e EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1,reranker-gte-v1; nao ha envs EMBEDDING_GOVERNOR_HEALTH_PROBE_* nem EMBEDDING_GOVERNOR_CAPACITY_PROBE_*.
  implication: na instancia em execucao em 2026-08-21 o governor operava sem guardrails preventivos de health/capacity, o que permite a cascata observada de fila expirada apos indisponibilidade do TEI.

- timestamp: 2026-08-21T05:32:00-03:00
  checked: tabela perf_metrics no Postgres da instancia ativa
  found: a consulta agregada de 24h retorna exatamente request_count=15, success_count=8 e success_rate=53.33 para embedding-gte-v1; por bucket local, 2026-08-21 00:00:00 tem 9 requests e 2 sucessos (22,22%), 2026-08-21 01:00:00 tem 1/1, 2026-08-20 05:00:00 tem 3/3 e 2026-08-20 03:00:00 tem 2/2.
  implication: o valor do dashboard esta reproduzido e a degradacao inteira se concentra no bucket de meia-noite de 2026-08-21.

- timestamp: 2026-08-21T05:34:00-03:00
  checked: schema da tabela logs no Postgres
  found: a tabela usa a coluna created_at, nao create_time.
  implication: a correlacao final precisa consultar created_at para extrair a amostra autoritativa de erros/sucessos do banco.

- timestamp: 2026-08-21T05:36:00-03:00
  checked: tabela logs no Postgres para model_name=embedding-gte-v1 nas ultimas 24h
  found: os 7 erros do banco coincidem exatamente com a janela de 2026-08-21 00:00:11 ate 00:05:29 e com as mesmas mensagens vistas em arquivo (1x status_code=500 upstream error: do request failed, 6x status_code=429 embedding governor queue timeout before dispatch). Contudo, a tabela logs tem 19 registros type=2 de consumo no mesmo periodo, acima dos 8 sucessos de perf_metrics.
  implication: os erros operacionais estao confirmados; agora e preciso separar registros de consumo que nao entram em perf_metrics, provavelmente testes/admin, para nao misturar fontes de verdade.

- timestamp: 2026-08-21T05:39:00-03:00
  checked: metadados dos consume logs type=2 e comparacao com perf_metrics por bucket
  found: apenas 2 registros de consumo das ultimas 24h sao testes explicitos de canal (token_id=0, token_name/content "Teste de modelo do canal" em 2026-08-20 05:01). Ainda assim, restam 17 consumos reais do token GBrain Graphify, enquanto perf_metrics soma so 8 sucessos. O bucket 2026-08-21 00:00:00 bate exatamente (9 requests, 2 sucessos), mas o bucket 2026-08-21 01:00:00 tem 10 consumos reais em log e so 1 sucesso persistido em perf_metrics.
  implication: a divergencia principal nao e trafego de teste; ha forte indicio de perda de amostras de perf_metrics do bucket 01:00, possivelmente por reset do processo/hotBuckets antes do flush.

- timestamp: 2026-08-21T05:42:00-03:00
  checked: inicio de logs/oneapi-20260821014837.log e estado do container router-ai-atius
  found: o arquivo de log atual comeca com bootstrap completo as 01:48:37 de sexta-feira, 21 de agosto de 2026, provando restart do processo antes do flush natural das 02:00; o sucesso de embedding as 01:53 aparece no processo novo, enquanto os sucessos de 01:15-01:23 estavam no processo anterior.
  implication: a perda dos 9 sucessos do bucket 01:00 e explicada por restart entre requests e flush, o que confirma uma fragilidade do perf_metrics baseado em hotBuckets somente em memoria.

## Eliminated

- hypothesis: "o incidente foi causado por roteamento em canal legado 9/10 ou por cache apontando `embedding-gte-v1` para outro upstream que nao o canal 11."
  evidence: "na janela 2026-08-21 00:00-02:00 todos os registros de `embedding-gte-v1`, incluindo os 7 erros, usam `channel_id=11`; a tabela `channels` nao tem 9/10 ativos ou mesmo presentes."
  timestamp: 2026-08-21T06:15:00-03:00

## Resolution

- root_cause: "Duas causas independentes se somaram: (1) o reboot de `horistic-srv` deixou a unica replica TEI sem listener/ready por cerca de dez minutos, enquanto a unit Podman do Router nao habilitava os probes do governor, produzindo 1 connect refused e uma cascata de 6 queue timeouts; (2) um restart do Router antes do flush horario perdeu 9 sucessos do bucket 01:00 mantido somente em memoria, amplificando artificialmente a queda para 53,33%."
- fix: "O governor agora rejeita admissao nova e libera espera enfileirada com `embedding_governor_upstream_unhealthy` quando o probe esta ruim; a unit e os templates ativam health/capacity probes com limiar de uma janela. `perf_metrics` ganhou flush explicito e recuperavel do bucket atual no shutdown. O TEI passou a duas replicas com PDB, rolling update de uma por vez, preStop, imagem/revisao fixas e Service privado em 31115; canal, migration, SQL e runbooks foram alinhados ao endpoint HA."
- verification:
  - "`go test ./pkg/perf_metrics ./service/embeddinggovernor` passou com testes de persistencia, retencao apos erro de upsert, fast reject e recuperacao de health."
  - "Imagem final `localhost/router-ai-atius:prod-20260821-ui-embedding-ha` (`5f8a7b71461e`) compilada integralmente pelo wrapper de 0.8 CPU e implantada em producao."
  - "Container live confirmou todos os envs de health/capacity, limiar 1 e a imagem nova; API local e dominio publico responderam 200."
  - "Amostra controlada pos-rollout: 15/15 requisicoes (1 inicial, 10 sequenciais, 4 concorrentes) retornaram HTTP 200 e vetores de 768 dimensoes. Uma chamada adicional apos restart tambem retornou 200/768."
  - "Antes do restart, o bucket das 04h ainda nao existia no banco; depois do shutdown gracioso foi persistido como 16 requests, 16 sucessos, 100%, provando a correcao de durabilidade."
  - "Buckets de 01h, 02h, 03h e 04h estao em 100%. A janela movel de 24h ficou em 86,79% porque preserva corretamente as sete falhas reais das 00h e subira quando esse bucket sair da janela."
  - "TEI respondeu 200 em `/health`, reportou `te_queue_size 0`, e nao houve novo erro de embedding/governor no journal apos o rollout."
  - "Rollout HA completo validado sob carga publica: 300/300 sucessos, zero falhas, duas replicas finais Ready e vetores de 768 dimensoes."
  - "Pos-restart final: 10/10 sequenciais, 4/4 concorrentes, lote publico 5/5 com 768 dimensoes e reranker 2/2; bucket corrente 273/273 (100%)."
  - "Testes focados finais passaram tres vezes no caminho relevante: model/migrations, service/embeddinggovernor, pkg/perf_metrics e relay Embedding/Rerank."
- residual_risk: "A HA e de pods/processos no host exigido. Nenhum software pode garantir continuidade durante perda fisica do proprio `horistic-srv`, storage compartilhado ou kube-proxy desse no; isso exigiria permitir uma replica em outro host com cache independente ou storage multi-node."
- files_changed:
  - "main.go"
  - "pkg/perf_metrics/flush.go"
  - "pkg/perf_metrics/flush_test.go"
  - "service/embeddinggovernor/governor.go"
  - "service/embeddinggovernor/governor_test.go"
  - "podman/systemd/router-ai-atius.env.example"
  - "podman/quadlets/router-ai-atius-new-api.container"
  - ".planning/debug/embedding-gte-success-rate.md"
