# Manual operacional - router-ai-atius

## Caminhos importantes

| Item | Caminho |
|---|---|
| Deploy ativo | `/home/ubuntu/GitHub/containers/router-ai-atius` |
| Runtime data/logs montados no pod | `/home/ubuntu/GitHub/containers/router-ai-atius/data`, `/home/ubuntu/GitHub/containers/router-ai-atius/logs` |
| Runtime source of truth | user unit `container-router-ai-atius.service` e imagem `ghcr.io/giovannimnz/router-ai-atius:latest` |
| Admin / fork-sync | `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router` |
| CLI operacional | `/home/ubuntu/GitHub/containers/router-ai-atius/bin/clianything` |
| Docs locais | `/home/ubuntu/GitHub/containers/router-ai-atius/docs` |
| Runbook Podman | `/home/ubuntu/GitHub/containers/router-ai-atius/docs/PODMAN.md` |
| Runbook OpenAI Codex e 1M context | `/home/ubuntu/GitHub/containers/router-ai-atius/docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md` |

Registro dedicado do corte full-Go 2026-06-18: `docs/FULL-GO-V1-MODELS-CUTOVER-2026-06-18.md`.
Registro dedicado do provider `OpenAI - Codex`, aliases `-1m`, mapping upstream e custos long-context: `docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`.

## Status atual validado

- Podman pod: `atius-ai-router`
- Containers em runtime atual: `router-ai-atius`, `redis`, infra pause. O antigo `model-detailed-hotfix` foi parado em 2026-06-18 apos cutover full-Go.
- Backend local: `http://127.0.0.1:3000`
- Caminho canônico de banco do runtime live: PgBouncer `10.1.1.1:6432` -> database `DBRouterAiAtius`
- Nao ha model proxy Python ativo no caminho `/v1/`.
- Alias Apache para `/v1/`: `http://127.0.0.1:3000`
- `data/postgres_data` é cluster PostgreSQL legado/desanexado; não é a fonte de verdade do runtime live
- Publico: `https://router.atius.com.br`
- Deploy full-Go + base URL `/v1` normalization 2026-06-18: imagem `ghcr.io/giovannimnz/router-ai-atius:latest` id `e389110f98fb8e3fce80ac8cf691a04c1c74b6268d91d5fb304bb6f574344151`.
- Rollback imediato antes da normalizacao `/v1`: `ghcr.io/giovannimnz/router-ai-atius:rollback-before-baseurl-v1-normalize-20260618122124`.
- Rollback provider consolidation: `ghcr.io/giovannimnz/router-ai-atius:rollback-before-provider-consolidation-20260618080648`.

Validacao rapida:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
scripts/podman-validate.sh
bin/clianything status
bin/clianything providers --all
```

`bin/clianything coverage --strict` e gate de paridade quando a arvore `docs/atius-router-docs/content/docs/en/api/management` estiver presente. Neste checkout de runtime ela pode estar ausente; nesse caso o comando falha por artefato de docs faltante, nao por estado dos providers.

`bin/clianything status --strict` e mais rigoroso: ele falha quando algum check fica `fail` ou `degraded`. Em 2026-06-18, o check legado `model-detailed` foi removido porque o runtime canonico passou a ser full-Go.

## Rotas Apache relevantes

Estado observado em 2026-06-18:

- `https://router.atius.com.br/api/` e `/` apontam para `127.0.0.1:3000`.
- `https://router.atius.com.br/v1/` aponta direto para o backend Go em `127.0.0.1:3000`; o catalogo e o relay nao dependem de container adicional.
- `https://router.atius.com.br/health` aponta para `127.0.0.1:3000/api/status`.
- Docs/i18n usam porta `3003` quando o serviço de docs esta ativo.
- Os botões e links `Docs` do router devem apontar para rotas internas do
  mesmo host: ingles em `/en/docs` e portugues em `/pt/docs`. Nao restaurar
  redirect para `https://docs.newapi.pro`.
- Aliases legados de OpenAPI JSON (`/docs.json`, `/docs/openapi.json`,
  `/json` e `/json/`) devem ser servidos localmente pelo docs app em `3003`
  como JSON OpenAPI 3.x. Nao apontar esses aliases para o HTML da SPA Go nem
  para o sidecar aposentado `model-detailed`.

Validacao:

```bash
apache2ctl configtest
/usr/bin/curl -fsS https://router.atius.com.br/api/status >/dev/null
/usr/bin/curl -fsS http://127.0.0.1:3000/api/status >/dev/null
/usr/bin/curl -fsS https://router.atius.com.br/health >/dev/null
/usr/bin/curl -fsS https://router.atius.com.br/docs.json | python3 -c 'import json,sys; j=json.load(sys.stdin); assert j["openapi"].startswith("3.")'
/usr/bin/curl -fsS https://router.atius.com.br/docs/openapi.json | python3 -c 'import json,sys; j=json.load(sys.stdin); assert j["paths"]'
scripts/smoke-docs-links.sh
```

## Providers e roteamento

Estado validado pelo banco:

| Channel | Tipo | Status | Base URL | Modelos |
|---|---:|---:|---|---|
| `MiniMax` | `35` | disabled | `https://api.minimax.io` | restaurado como canal consolidado unico, mas mantido desabilitado no estado final |
| `DeepSeek` | `43` | enabled | `https://api.deepseek.com` | `deepseek-v4-pro`, `deepseek-v4-flash` via canal consolidado unico |
| `OpenAI - Codex` | `57` | enabled | OAuth local / sem `base_url` | `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark` |
| `MiniMax - Anthropic-Compatible`, `MiniMax - Embeddings`, `DeepSeek - Anthropic-Compatible`, `Codex - Embeddings` | legado | disabled | historico | rotas substituidas pelos canais unificados |

Comando:

```bash
bin/clianything providers --all
```

Endpoints ativos:

- OpenAI-Compatible: `/v1/models`, `/v1/chat/completions`, `/v1/responses`
- Anthropic-Compatible: `/v1/messages`
- Embeddings: `/v1/embeddings`

## Catalogo `/v1/models` Go-owned e precificacao

Estado validado em 2026-06-18:

- O backend Go e o dono do catalogo enriquecido em `/v1/models`; o middleware Python nao e fonte do contrato de model-list.
- O root publico de `/v1/models` em modos model-list contem somente `data`: nao expor top-level `object`, `success`, `first_id`, `last_id` ou `has_more`.
- Os campos enriquecidos publicos esperados por item sao `pricing`, `supported_endpoint_types`, `endpoint_routes` e `billing_mode` quando disponivel.
- O objeto publico `pricing` expoe somente `input` e `output`; nao expoe `unit`.
- O payload publico nao expoe os aliases redundantes `input_price`/`output_price`, `quota_type`, `enable_groups` ou `supported_endpoint_type_labels`.
- `pricing_version` e campo interno e nao deve aparecer no payload publico de `/v1/models`.
- `pricing_source` e `pricing_estimated` ficam internos e nao devem aparecer no payload publico de `/v1/models`.
- A ordenacao publica e deterministica: texto antes de embeddings; providers MiniMax, DeepSeek e OpenAI/OpenAI Codex; dentro do provider, modelos mais recentes/capazes primeiro.
- Regras de variante: versao numerica maior sobe; `-highspeed` fica acima da variante sem `-highspeed`; `pro` fica acima de `flash`, que fica como variante secundaria.
- Contrato final da Phase 24 para Codex: `gpt-5.4` e o modelo default de long-context; nao republicar `gpt-5.4-1m` nem `gpt-5.5-1m`.
- `api_format=anthropic` e headers Anthropic selecionam modelos Anthropic-capable no catalogo Go, mantendo o mesmo root `{"data":[...]}`.
- Graphify fica obrigatorio no fluxo GSD desta area quando habilitado no checkout. Em 2026-06-18, este checkout retornou `graphify status: disabled` e nao tinha `.planning/config.json`; nesse caso registrar Graphify como indisponivel e usar testes/CLI/smoke como evidencias.
- Para modelos ainda sem preco cadastrado, o comportamento esperado do catalogo enriquecido e retornar `0.00` ate o cadastro real ou estimado ser feito.
- O cadastro de modelos token-priced deve usar `ModelRatio` e `CompletionRatio`; nao usar `ModelPrice` para esses modelos, porque `ModelPrice` muda a semantica para cobranca fixa/request.
- Backup antes do cadastro de precos em 2026-06-15: `/home/ubuntu/GitHub/containers/router-ai-atius/backups/clianything/20260615_063018_options.sql`.

Precos cadastrados/validados no backend:

| Modelo | Input $/M | Output $/M | Fonte |
|---|---:|---:|---|
| `MiniMax-M3` | 0.30 | 1.20 | backend |
| `MiniMax-M2.7` | 0.30 | 1.20 | backend |
| `MiniMax-M2.5` | 0.30 | 1.20 | backend |
| `MiniMax-M2.1` | 0.30 | 1.20 | backend |
| `MiniMax-M2.1-highspeed` / `-hs` | 0.60 | 2.40 | estimado |
| `MiniMax-M2.5-highspeed` / `-hs` | 0.60 | 2.40 | estimado |
| `MiniMax-M2.7-highspeed` / `-hs` | 0.60 | 2.40 | estimado |
| `deepseek-v4-flash` | 0.14 | 0.28 | backend |
| `deepseek-v4-pro` | 0.435 | 0.87 | backend |
| `gpt-5.5` | 5.00 | 30.00 | estimado/standard |
| `gpt-5.4` | 5.00 | 22.50 | OpenAI standard long-context |
| `gpt-5.4-mini` | 0.75 | 4.50 | estimado/standard |
| `gpt-5.3-codex-spark` | 1.75 | 14.00 | estimado |
| `embo-01` | 0.069 | 0.069 | estimado |
| `text-embedding-3-small` | 0.02 | 0.02 | OpenAI |
| `text-embedding-3-large` | 0.13 | 0.13 | OpenAI |

Comandos de verificacao:

```bash
source <(/home/ubuntu/.local/bin/atius-vault-env router-ai-atius)

bin/clianything api GET /api/pricing --bearer "$ATIUS_ROUTER_ADMIN_TOKEN"
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" 'http://127.0.0.1:3000/v1/models?api_format=anthropic'
```

### Codex OAuth embeddings Go-native

Estado validado em 2026-06-18:

- `text-embedding-3-small` e `text-embedding-3-large` pertencem ao channel 5 `OpenAI - Codex`, tipo `57`, mas estao desativados no runtime atual porque o upstream retornou `429 insufficient_quota`.
- O channel 5 compartilha a mesma credencial OAuth do Codex para chat/responses e embeddings. Nao copiar chave para outro canal.
- `Codex - Embeddings` pode existir como canal desabilitado para historico/fallback manual, mas nao deve virar rota ativa.
- `/v1/embeddings` no backend Go roteia para o adaptador Codex OAuth e usa `https://api.openai.com/v1/embeddings` no upstream.
- O resultado publico validado em 2026-06-18 foi roteamento local correto com retorno upstream `429 insufficient_quota`; isso classifica a falha como quota/licenca upstream, nao erro local de selecao de canal.
- Implementacao principal: `relay/channel/codex/adaptor.go`, `service/codex_credential_refresh_task.go`, `controller/model.go` e `service/modelcatalog/catalog.go`.
- Testes de protecao: `relay/channel/codex/adaptor_test.go`, `service/codex_credential_refresh_task_test.go` e `controller/model_list_test.go`.

Regras praticas:

- Para clientes OpenAI-Compatible, usar `/v1/chat/completions`, `/v1/responses`, `/v1/models` etc com `base_url=https://router.atius.com.br/v1`.
- Para clientes Anthropic-Compatible, usar `/v1/messages` com `base_url=https://router.atius.com.br` ou `https://router.atius.com.br/v1` conforme SDK/cliente.
- Para embeddings OpenAI-compatible, usar `/v1/embeddings` somente quando o modelo estiver anunciado em `/v1/models`. No runtime atual, o provider ativo e `Local TEI - GTE Embeddings` com o alias publico `embedding-gte-v1`, dimensao `768`, apontando para `http://10.1.1.4:3000`.
- Provider de embeddings MiniMax: canal unico `MiniMax` tipo `35` com `embo-01`; a conversao OpenAI -> MiniMax native acontece no adaptador Go, mas o provider fica restaurado e desabilitado no estado final da Phase 24.
- Provider de embeddings Codex: `OpenAI - Codex` channel 5 com `text-embedding-3-small` e `text-embedding-3-large`; fica desativado ate a quota/licenca upstream aceitar chamadas reais.
- Nao reativar `OpenAI - Embeddings` separado nem depender de chave OpenAI duplicada para estes modelos; a regra do fork e compartilhar a credencial Codex OAuth.
- MiniMax e DeepSeek nao precisam de canais duplicados por protocolo: o tipo do canal identifica o provider e o relay Go escolhe URL/formato conforme o endpoint recebido (`/v1/chat/completions`, `/v1/messages`, `/v1/embeddings`).
- `/v1/models` sem token deve retornar 401. Isso nao indica falha.
- DeepSeek deve permanecer ativo como canal consolidado unico no estado final restaurado.
- MiniMax embeddings pode retornar `429` quando o upstream bloquear por quota/RPM persistente; nesse caso manter `embo-01` fora do catalogo ativo e manter o provider MiniMax desabilitado no estado final.
- `MiniMax` e `DeepSeek` devem preferir `base_url` no provider root (`https://api.minimax.io`, `https://api.deepseek.com`), mas o relay Go tambem aceita base URL com `/v1` e normaliza automaticamente. Nao usar sufixos especificos como `/anthropic` no canal consolidado.

## Governor de embeddings Go-native

Estado atualizado em 2026-07-05:

- O governor de embeddings roda dentro do proprio processo Go do router; nao ha sidecar, middleware Python, container adicional ou rota `model-detailed` no caminho canonico.
- Implementacao principal: `service/embeddinggovernor/` e integracao em `relay/embedding_handler.go`.
- Escopo default: somente `embedding-gte-v1`. Outros modelos passam pelo relay normal sem fila do governor.
- `embedding-gte-v1` e o unico alias publico governado; `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` nao muda durante a recuperacao/catalog restore.
- Envelope automatico protegido: `min=1`, `initial=2`, `max=0` e `batch_concurrency=0`, fila interativa `128`, fila batch `512`, timeout interativo `30s`, timeout batch `10m`, cooldown `10m`. Valor `0` em `EMBEDDING_GOVERNOR_MAX_CONCURRENCY` ou `EMBEDDING_GOVERNOR_BATCH_CONCURRENCY` significa sem teto estatico no router; a capacidade passa a crescer pelo feedback do governor, pelos sinais de health/capacidade/latencia/cooldown e pela capacidade real dos pods TEI disponiveis.
- Classificacao de workload e metadata-only. `X-Embedding-Workload` e opcional para clientes normais e fica como override operacional para operadores. Ordem de precedencia:
  1. `X-Embedding-Workload: batch|bulk|interactive|realtime`;
  2. thresholds locais derivados do request (`InputCount >= 2` ou `InputChars >= 12000`).
- Nao exponha um alias publico `*-batch`: sem alias publico batch; batch e uma classe operacional interna do mesmo modelo `embedding-gte-v1`.
- arrays governados de `embedding-gte-v1` acima de `4` itens fazem fail-closed no relay antes do acquire do governor ou dispatch upstream. O header `interactive` nao bypassa esse cap, porque o TEI local nao tem caminho seguro de recomposicao transparente de resposta para batches maiores.
- Feedback adaptativo agora fica separado entre interativo e batch. O governor mantem EWMA/counters distintos para cada classe, para que catch-up lento nao envenene a reabertura interativa.
- Classificacao de falha tambem ficou mais estrita. So pressao real reduz concorrencia: `429`, `5xx`, falha de transporte/timeout ou request acima do slow threshold da propria classe. Erros comuns de cliente `4xx` nao reduzem concorrencia por si sós.
- Em pressao real, o governor reduz para `min=1` e entra em cooldown. Durante o cooldown, novos despachos governados ficam segurados na fila ate expirar ou ate o timeout do request; a reabertura e gradual por janela de sucesso e por demanda interativa saudavel.
- Guardrail de health do TEI agora existe, mas e advisory, read-only e disabled-by-default. Ele so liga quando `EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED=true` e `EMBEDDING_GOVERNOR_HEALTH_PROBE_URL` apontam para um endpoint HTTP/HTTPS sem auth extra. O probe faz apenas `GET`, nao envia token/header secreto e nao controla deploy/restart.
- Uma unica amostra ruim de `/health` nao reduz concorrencia e nao arma cooldown. O governor apenas incrementa `health_bad_windows`.
- Apos `3` janelas ruins consecutivas (default minimo), o governor bloqueia scale-up. Cada janela ruim adicional pode reduzir a concorrencia em `1` ate `min=1`. Uma amostra saudavel reseta o contador de janelas ruins.
- Isso existe porque `/health` curto pode atrasar durante inferencia CPU-bound sem erro real de embeddings. O sinal de health e deliberadamente mais conservador que o sinal do trafego real.
- Guardrail de capacidade dos pods TEI tambem existe, read-only e disabled-by-default. Ele so liga quando `EMBEDDING_GOVERNOR_CAPACITY_PROBE_ENABLED=true` e `EMBEDDING_GOVERNOR_CAPACITY_PROBE_URL` apontam para um endpoint interno sem auth extra que exponha uso e pods. Se `capacity_used_percent`, `cpu_usage_percent` ou `memory_usage_percent` estiverem acima de `EMBEDDING_GOVERNOR_CAPACITY_MAX_USED_PERCENT=80`, o governor bloqueia scale-up imediatamente; depois de `EMBEDDING_GOVERNOR_CAPACITY_BAD_WINDOW_THRESHOLD=3` janelas ruins, reduz concorrencia gradualmente em `1` ate `min=1`. Pods degradados (`pods_ready < pods_total`) tambem bloqueiam scale-up.
- O capacity probe aceita JSON ou texto estilo Prometheus. Campos reconhecidos incluem `capacity_used_percent`, `capacity_free_percent`, `cpu_usage_percent`, `memory_usage_percent`, `pods_ready` e `pods_total`.
- Quando a fila esta cheia ou expira antes de despachar para o upstream, o router retorna `429` com codigo `embedding_governor_queue_full`, `embedding_governor_batch_queue_full` ou `embedding_governor_queue_timeout`.
- Snapshot/telemetria agregada relevante do governor: `interactive_average_latency_ms`, `batch_average_latency_ms`, `interactive_completed`, `batch_completed`, `interactive_slow`, `batch_slow`, `health_probe_enabled`, `health_bad_windows`, `last_health_status`, `last_health_latency_ms`, `last_health_at`, `capacity_probe_enabled`, `capacity_bad_windows`, `last_capacity_status`, `last_capacity_used_percent`, `last_capacity_free_percent`, `last_capacity_ready_pods`, `last_capacity_total_pods`, `last_capacity_latency_ms`, `last_capacity_at`. Nenhum desses campos deve carregar input bruto, token ou URL secreta.

Variaveis de ambiente suportadas:

```bash
EMBEDDING_GOVERNOR_ENABLED=true
EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1
EMBEDDING_GOVERNOR_BATCH_MODELS=
EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true
EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD=2
EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY=2
EMBEDDING_GOVERNOR_MIN_CONCURRENCY=1
EMBEDDING_GOVERNOR_MAX_CONCURRENCY=0
EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=0
EMBEDDING_GOVERNOR_QUEUE_LIMIT=128
EMBEDDING_GOVERNOR_BATCH_QUEUE_LIMIT=512
EMBEDDING_GOVERNOR_INTERACTIVE_TIMEOUT=30s
EMBEDDING_GOVERNOR_BATCH_TIMEOUT=10m
EMBEDDING_GOVERNOR_COOLDOWN=10m
EMBEDDING_GOVERNOR_SLOW_REQUEST_DURATION=2m
EMBEDDING_GOVERNOR_BATCH_SLOW_REQUEST_DURATION=10m
EMBEDDING_GOVERNOR_LATENCY_TARGET=90s
EMBEDDING_GOVERNOR_SCALE_UP_MIN_INTERVAL=30s
EMBEDDING_GOVERNOR_SCALE_DOWN_IDLE=10m
EMBEDDING_GOVERNOR_SUCCESS_WINDOW=8
EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED=false
EMBEDDING_GOVERNOR_HEALTH_PROBE_URL=
EMBEDDING_GOVERNOR_HEALTH_PROBE_TIMEOUT=30s
EMBEDDING_GOVERNOR_HEALTH_PROBE_INTERVAL=30s
EMBEDDING_GOVERNOR_HEALTH_BAD_WINDOW_THRESHOLD=3
EMBEDDING_GOVERNOR_HEALTH_SLOW_DURATION=10s
EMBEDDING_GOVERNOR_CAPACITY_PROBE_ENABLED=false
EMBEDDING_GOVERNOR_CAPACITY_PROBE_URL=
EMBEDDING_GOVERNOR_CAPACITY_PROBE_TIMEOUT=30s
EMBEDDING_GOVERNOR_CAPACITY_PROBE_INTERVAL=30s
EMBEDDING_GOVERNOR_CAPACITY_MAX_USED_PERCENT=80
EMBEDDING_GOVERNOR_CAPACITY_BAD_WINDOW_THRESHOLD=3
```

Significado operacional dos envs novos:

- `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true`: default seguro. Sem header explicito, o router infere `interactive` ou `batch` para modelos governados usando apenas metadata agregada.
- `EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD=2`: threshold default para classificar arrays de input sem header como batch quando `InputCount >= 2`.
- `EMBEDDING_GOVERNOR_MAX_CONCURRENCY=0`: remove o teto estatico de concorrencia total no router. Um valor positivo reintroduz um teto operacional explicito.
- `EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=0`: remove o teto estatico separado de batch. Batch continua cedendo prioridade quando ha interativos esperando e continua limitado pela concorrencia total corrente.
- `EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED=false`: default seguro. Sem isso, o governor ignora completamente o sinal de `/health`.
- `EMBEDDING_GOVERNOR_HEALTH_PROBE_URL=`: endpoint read-only do TEI. Se vazio, invalido ou exigir auth por URL, o probe fica desabilitado.
- `EMBEDDING_GOVERNOR_HEALTH_PROBE_TIMEOUT=30s`: timeout minimo seguro. Timeouts menores que isso ficam normalizados para `30s`.
- `EMBEDDING_GOVERNOR_HEALTH_PROBE_INTERVAL=30s`: cadencia default entre probes. Tambem e normalizada para pelo menos `30s`.
- `EMBEDDING_GOVERNOR_HEALTH_BAD_WINDOW_THRESHOLD=3`: minima quantidade de janelas ruins consecutivas antes de bloquear scale-up e comecar a descer gradualmente.
- `EMBEDDING_GOVERNOR_HEALTH_SLOW_DURATION=10s`: resposta `200` acima disso conta como janela ruim, mas ainda sem cooldown imediato.
- `EMBEDDING_GOVERNOR_CAPACITY_PROBE_ENABLED=false`: default seguro. Sem isso, o governor nao consulta endpoint de capacidade.
- `EMBEDDING_GOVERNOR_CAPACITY_PROBE_URL=`: endpoint read-only interno de capacidade dos pods TEI. Se vazio, invalido ou exigir auth por URL, o probe fica desabilitado.
- `EMBEDDING_GOVERNOR_CAPACITY_PROBE_TIMEOUT=30s`: timeout minimo seguro para consulta de capacidade.
- `EMBEDDING_GOVERNOR_CAPACITY_PROBE_INTERVAL=30s`: cadencia default entre consultas de capacidade.
- `EMBEDDING_GOVERNOR_CAPACITY_MAX_USED_PERCENT=80`: acima ou igual a esse uso agregado, o governor nao aumenta concorrencia.
- `EMBEDDING_GOVERNOR_CAPACITY_BAD_WINDOW_THRESHOLD=3`: quantidade de janelas ruins consecutivas antes de reduzir concorrencia; o bloqueio de scale-up ocorre ja na primeira janela ruim.

Base empirica dos threads `019f010f-421b-7243-ac95-46c2b287e868` e `019f017f-51fb-7ea1-972a-d4b91434be7d`, que justificam o guardrail conservador:

- Inicio do GBrain: `3667 stale chunks`, `Embedded: 0`, schema migrado de `1536` para `768`.
- Tentativas com paginas grandes deram timeout sem gravar vetores no inicio; por isso o fallback tecnico deve continuar em `min=1`, mesmo com operacao diaria iniciando em `initial=2`.
- `GBRAIN_EMBED_PROVIDER_BATCH_SIZE=4` foi o sub-batch que passou em slug que falhava; o TEI tambem registrou que o backend nao suporta batch client acima de `4`.
- Antes do ajuste de Kubernetes, a conclusao sobre concorrencia estava contaminada: ate `concurrency=1` podia sofrer restart porque o kubelet matava o container por `livenessProbe` em `/health` durante inferencia CPU longa. O exit `137` veio do restart forcado por probe, nao de OOM.
- O ajuste que mudou o envelope operacional foi: `max_client_batch_size=4`, probes mais tolerantes, health monitorado com timeout de `30s`, pod novo `1/1`, `restarts=0`, limite `3 CPU / 12Gi`.
- Depois do ajuste de probes, `concurrency=1`, `2`, `3` e `4` rodaram com `errors=0` e `restarts=0`; `concurrency=4` progrediu por centenas de paginas, com checkpoints de `Embedded` subindo de `598` ate pelo menos `1590`.
- No trecho estavel pos-probes, TEI ficou tipicamente em `~1.3` a `1.7` CPU e `~1.4GiB` a `2.3GiB` RSS, com warmup perto de `7.9GiB`, abaixo do limite de `12Gi`. Picos de load/memoria do host vieram em parte de processos externos, nao do TEI.
- Decisao posterior de operacao diaria: com o governor em producao, o teto estatico `max=3` e a janela manual `4` foram removidos. Para permitir aumento horizontal de pods TEI, o default do router passa a ser `min=1`, `initial=2`, `max=0`, `batch_concurrency=0`; a autorregulacao fica em cooldown, slow-request, EWMA de latencia, health guardrail, capacity guardrail e limites de fila. Um timeout isolado de `/health` nao deve sozinho rebaixar o sistema.

Validacao local e gate operacional apos mudanca no governor:

```bash
# Testes focados dos guardrails de health/capacidade/hysteresis
/usr/local/go/bin/go test ./service/embeddinggovernor -run '^(TestHealthProbeDisabledByDefault|TestHealthHysteresisIgnoresSingleBadSample|TestHealthHysteresisReducesAfterConsecutiveBadWindows|TestHealthHysteresisHealthySampleResetsBadWindows|TestCapacityProbeBlocksScaleUpImmediatelyAndReducesAfterThreshold|TestCapacityProbeHealthySampleResetsGuardrail|TestCapacityProbeBlocksWhenPodsAreDegraded|TestDefaultCapacityProbeParsesJSONUsageAndPods|TestCapacityProbeParsesPrometheusTextUsageAndPods)$' -count=1

# Regressao do pacote do governor
/usr/local/go/bin/go test ./service/embeddinggovernor -count=1

# Gate Go mais amplo antes de deploy controlado
/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1

# Smoke autenticado de embeddings locais (dimension 768)
test -n "$ATIUS_ROUTER_TOKEN" && \
  ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 \
  ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 \
  python3 scripts/smoke-embeddings.py

# Smoke autenticado no-header com array pequeno; valida inferencia batch automatica
test -n "$ATIUS_ROUTER_TOKEN" && \
  ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 \
  ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 \
  ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE=array \
  python3 scripts/smoke-embeddings.py

# Gate Graphify do checkout
node /home/ubuntu/.codex/gsd-core/bin/gsd-tools.cjs graphify status
```

Leitura correta desses gates:

- Para automacao e validacao, recupere `ATIUS_ROUTER_API_KEY` do HashiCorp Vault machine/automation e exporte como `ATIUS_ROUTER_TOKEN` apenas no shell efemero do teste.
- Sem `ATIUS_ROUTER_TOKEN`, o smoke de embeddings deve falhar com `exit 2` antes da rede. Isso e limitacao de ambiente, nao passe livre.
- Se `graphify status` retornar `stale=true` ou `commit_stale=true` num checkout com Graphify habilitado, rebuild e obrigatorio antes de assinar a mudanca. O rebuild faz parte do gate de validacao, nao do governor.
- Deploy/restart nao e automatico em execucao de plano. Quando for validar em runtime, usar somente a user unit existente:

```bash
systemctl --user restart container-router-ai-atius.service
```

- Apos esse restart controlado, os monitor gates minimos sao:
  - `bin/clianything status --strict`
  - TEI `ready=true`
  - `restarts=0`
  - `/health` interpretado por janelas consecutivas, nunca por timeout isolado
  - TEI CPU/memoria dentro do envelope diario (`limits.cpu=2`, `limits.memory=12Gi`)
  - progresso do GBrain/Obsidian seguindo adiante
  - zero embed errors no smoke/logs operacionais

## Fila anti-rate-limit do middleware legado

O `model-detailed-hotfix` foi removido do caminho runtime em 2026-06-18. Esta secao fica como historico: se algum operador reativar esse middleware manualmente, a fila local abaixo era o comportamento anterior. O runtime canonico atual nao depende dela.

Padrao em 2026-06-15:

- `MiniMax-M3` e demais modelos `minimax-*`: bucket `minimax-chat`, intervalo `1.5s`.
- `deepseek-*`: bucket `deepseek-chat`, intervalo `0.5s`.
- `gpt-*` via Codex/OpenAI: bucket `codex-chat`, intervalo `0.5s`.
- `embo-01`: bucket `minimax-embeddings`, intervalo `5.0s`.
- `text-embedding-*`: bucket `openai-embeddings`, intervalo `1.0s`.
- Max retry transitorio: `2`.
- Max espera de fila: `45s`.

Headers de diagnostico retornados quando a fila atua:

- `X-Atius-Rate-Queue`
- `X-Atius-Rate-Queue-Wait-Ms`
- `X-Atius-Rate-Retry-Count`

Isto nao garante que nunca ocorrera `429`/`529`: se o provider estiver sem quota, com RPM ja excedido antes da chamada, ou com cluster persistentemente indisponivel, o router preserva a falha upstream depois de espaçar e retryar.

## Hermes

Estado observado em 2026-06-15:

- Hermes instalado: `v0.16.0`.
- Config persistente em `~/.hermes/config.yaml` estava com `provider: custom`, `default: MiniMax-M3`, `base_url: ${MINIMAX_BASE_URL}` e `api_mode: anthropic_messages`.
- Isso aponta Hermes persistente para MiniMax direto, nao para o router Atius.
- `fallback_providers` como strings nao e aceito pelo Hermes v0.16; o formato esperado e lista de objetos com `provider` e `model`.

### Hermes via Atius Router

Estado validado em 2026-07-01:

- Documentacao dedicada: `docs/HERMES-ATIUS-ROUTER-PROVIDER.md`.
- O Hermes deve usar o Atius Router como OpenAI-compatible, nao como
  Anthropic-compatible, para modelos GPT/Codex.
- `api_mode` correto para `gpt-5.4` e `gpt-5.5`:
  `chat_completions`.
- `base_url` correto: `https://router.atius.com.br`, sem sufixo `/v1`.
- `model.aliases` deve apontar os modelos GPT/Codex para
  `custom:atius-router/...`, evitando autodeteccao ambigua do Hermes quando se
  usa `hermes -m <modelo>`.

Exemplo sem secrets:

```yaml
model:
  provider: custom
  default: gpt-5.4
  context_length: 1048576
  base_url: https://router.atius.com.br
  api_key: ${ATIUS_ROUTER_API_KEY}
  api_mode: chat_completions
  aliases:
    gpt-5.4: custom:atius-router/gpt-5.4
    gpt-5.5: custom:atius-router/gpt-5.5
fallback_providers:
  - provider: custom
    model: MiniMax-M3
```

Nao usar `ATIUS_ROUTER_BASE_URL=https://router.atius.com.br/v1` nesse modo,
pois o cliente OpenAI-compatible monta `/v1/chat/completions`.

### Hermes via MiniMax direto

```yaml
model:
  provider: custom
  default: MiniMax-M3
  base_url: ${MINIMAX_BASE_URL}
  api_mode: anthropic_messages
```

### Hermes via OpenAI

Use quando a intencao for passar pelo provider OpenAI/OpenAI Codex, nao pelo router:

```yaml
model:
  provider: openai
  default: gpt-5
```

Se usar provider custom OpenAI-compatible:

```yaml
model:
  provider: custom
  default: gpt-5
  base_url: ${OPENAI_BASE_URL}
  api_mode: openai_chat
```

## Codex / OpenAI SDK / Anthropic SDK

Valido em 2026-06-18:

- OpenAI SDK local contra `http://127.0.0.1:3000/v1` com `MiniMax-M3`: OK.
- Anthropic SDK local contra `http://127.0.0.1:3000` com `MiniMax-M3`: OK.
- OpenAI SDK local contra `http://127.0.0.1:3000/v1` com `gpt-5.5` via `OpenAI - Codex`: OK com `ATIUS_ROUTER_STREAM=1`.
- Router Go deve rotear Anthropic/OpenAI automaticamente via canal unico do provider quando o provider estiver ativo. Contrato final da Phase 24: `DeepSeek` channel 2 ativo; `MiniMax` channel 1 restaurado, mas desabilitado.
- `OpenAI - Codex` esta ativo no channel 5 com modelos `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark`.
- Embeddings Codex devem usar o mesmo channel 5 e a mesma credencial OAuth do Codex, sem servico/container adicional nem copia de chave; runtime atual deixa `text-embedding-3-*` desativado por `429 insufficient_quota`.
- `data/codex-home/.codex/auth.json` existe e tem `auth_mode: chatgpt`; esse arquivo e credencial de runtime e nao deve ser copiado para docs/logs.
- O smoke principal presente neste checkout e `scripts/smoke-provider-consolidation.py`; ele exige `ATIUS_ROUTER_TOKEN` via env para chamadas reais e valida a matriz ativa OpenAI/Anthropic/Codex.
- `scripts/smoke-embeddings.py` fica como check legado/focado de embeddings.
- Para smoke de embeddings Go-native, o default local e `http://127.0.0.1:3000/v1`.

Nao documentar ou copiar tokens desse arquivo em notes, docs ou logs.

## Protecao contra sync upstream

O fork-sync do `omni-srv-admin` usa merge strategy `theirs`. Por isso, toda customizacao abaixo precisa ficar em `protected_paths` e ser conferida em dry-run antes de merge upstream:

- `.dockerignore` com exclusoes de runtime: `/backups`, `/data`, `/logs`, `/runtime`.
- `controller/model.go` e `controller/model_list_test.go`: contrato publico de `/v1/models`, sem `pricing_version`, com ordenacao deterministica.
- `service/modelcatalog/`: catalogo Go-native e regras de ordenacao por provider/variante.
- `relay/common/relay_utils.go` e `relay/common/relay_utils_test.go`: normalizacao de `base_url` com `/v1`, slash final e path de request.
- `common/endpoint_type.go` e `common/endpoint_type_test.go`: MiniMax/DeepSeek multiprotocolo e `embo-01` como embeddings-only.
- `dto/embedding.go`, `relay/channel/minimax/` e `relay/channel/deepseek/`: roteamento Go-native de embeddings MiniMax e URLs OpenAI/Anthropic por provider unico.
- `relay/embedding_handler.go` e `service/embeddinggovernor/`: governor Go-native de embeddings locais, sem sidecar Python.
- `constant/channel.go`, `web/default/src/features/channels/constants.ts`, `web/classic/src/constants/channel.constants.js` e locales i18n: label `OpenAI - Codex`.
- `tools/clianything.py` e `tests/test_clianything.py`: `phase19-apply` deve consolidar canais e nunca recriar providers duplicados como rota ativa.
- `relay/channel/codex/`: adaptador Codex OAuth com suporte a embeddings.
- `service/codex_*.go`: refresh OAuth e protecao de referencias `shared:codex`.
- `docs/` e `.planning/`: requisitos e manual operacional do fork.

Depois de qualquer sync upstream, validar pelo menos:

```bash
go test ./common ./controller ./service/modelcatalog ./relay/channel/codex ./relay/channel/minimax ./relay/channel/deepseek ./service ./service/embeddinggovernor ./relay -run 'TestGetEndpointTypesByChannelType|TestListModelsRepresentativeOrder|TestListModelsAnthropicPayloadAndOrder|TestListModelsPayloadShapeAndPublicFields|TestCodex|TestGetRequestURL|TestConvertEmbeddingRequest|TestDoResponseForEmbedding|TestIsSharedCodexCredentialReference|TestParseSharedCodexChannelID|TestAcquire|TestGovernor' -count=1
python3 -m unittest tests.test_clianything.Phase19ProviderRoutingTests -v
bin/clianything status
bin/clianything providers --all
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models | jq '.data[0].id, any(.data[]; has("pricing_version"))'
```

## GBrain

Estado atualizado em 2026-06-26 apos a instalacao do TEI local:

- Wrapper ativo: `/home/ubuntu/.local/bin/gbrain`.
- O wrapper define `GBRAIN_STATEMENT_TIMEOUT=0`, `GBRAIN_IDLE_TX_TIMEOUT=0`, `OPENAI_BASE_URL=http://127.0.0.1:3000/v1` e `OPENAI_API_KEY` a partir de `~/.gbrain/config.json`.
- `~/.gbrain/config.json` deve usar `embedding_model: openai:embedding-gte-v1` e `embedding_dimensions: 768`.
- O GBrain deve chegar direto ao Go router; o endpoint ativo de embeddings e o canal `Local TEI - GTE Embeddings`, alias `embedding-gte-v1`, upstream `http://10.1.1.4:3000`.
- O governor Go-native protege esse caminho antes de despachar a chamada ao TEI.
- `embo-01` e `text-embedding-3-*` ficam como historico/desativados ate MiniMax/Codex aceitarem chamadas reais sem `429`.
- Backup antes da mudanca: `/home/ubuntu/.gbrain/config.json.bak-router-embeddings-20260615_030827`.

## Operacao diaria

```bash
# Saude geral
bin/clianything status

# Providers/modelos ativos
bin/clianything providers
bin/clianything embeddings
bin/clianything models --from-channels
# Quando docs management estiverem presentes:
bin/clianything coverage --strict

# Ultimos logs sem payload sensivel
bin/clianything logs --limit 50

# Logs de containers
podman logs router-ai-atius --tail 80

# Restart controlado do backend Go
systemctl --user restart container-router-ai-atius.service

# O antigo container model-detailed deve permanecer parado no runtime full-Go.
systemctl --user is-active container-model-detailed.service
podman ps -a --filter pod=atius-ai-router --format '{{.Names}}' | grep -E '^model-detailed' && exit 1 || true
```

Nao usar `podman restart router-ai-atius` como rotina operacional neste runtime. Em 2026-06-15, restart direto do container Go falhou durante cleanup/unmount e precisou ser recuperado pelo user unit `container-router-ai-atius.service`.

## CLIAnything Phase 18

Estado validado em 2026-06-15:

- Manifesto `tools/clianything_endpoints.json` com 158 endpoints de management.
- `bin/clianything coverage --strict` com 100% de cobertura, zero missing/extra/problemas.
- Classificacoes atuais: `38 api-action`, `38 db-crud`, `43 read-only`, `36 auth-flow`, `3 external-webhook`, `0 unsupported-safe`.
- Paridade de `api-action`: `10` endpoints usam subcomandos de dominio e `28` usam `endpoint invoke`; nenhum depende de `clianything api`.
- Comandos tipados para operacoes de frontend: `channel`, `model`, `option`, `ratio`, `token`, `log`, `task`, `vendor`.
- `endpoint list/show/invoke` para qualquer endpoint documentado, com manifest gate.
- `api` generico mantido para diagnostico, mas nao conta sozinho como paridade; quando o path bate no manifesto, tambem respeita dry-run de endpoints `api-action`.
- Escritas DB e acoes API continuam dry-run por padrao; `--execute` e obrigatorio.
- `bin/clianything embeddings` mostra os providers de embeddings ativos e seus statuses.

Exemplos:

```bash
bin/clianything channel test --id 3 --execute
bin/clianything channel fetch-models --id 3 --execute
bin/clianything model sync-upstream --preview
bin/clianything ratio channels
bin/clianything embeddings
bin/clianything endpoint list --classification api-action
```

Smoke local:

```bash
source <(/home/ubuntu/.local/bin/atius-vault-env router-ai-atius)
ATIUS_ROUTER_ACTIVE_ONLY=1 python3 scripts/smoke-provider-consolidation.py
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" 'http://127.0.0.1:3000/v1/models?api_format=anthropic'
```

Sem `ATIUS_ROUTER_TOKEN`, os scripts de smoke retornam `exit 2` antes de chamar rede.

Ultima bateria operacional estrita em 2026-06-18 (historico pre-Phase 24):

- MiniMax ativo naquele momento: `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.7` passaram via OpenAI-compatible e Anthropic-compatible.
- OpenAI - Codex ativo: `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark` passaram via OpenAI-compatible.
- DeepSeek ficou desativado naquele momento porque a chave upstream cadastrada retornou `401 invalid api key`.
- Embeddings ficam fora do catalogo ativo: `embo-01` por `429 rate limit exceeded(RPM)` e `text-embedding-3-*` por `429 insufficient_quota`.
- Rotas desativadas foram cobertas por negativo no smoke e nao devem gerar log de selecao de canal antigo.

Checkpoint publico via Cloudflare em 2026-06-15:

- `https://router.atius.com.br/health`: HTTP 200, `{"status":"healthy","backend":"connected"}`, com header `server: cloudflare`.
- `https://router.atius.com.br/api/status`: HTTP 200.
- `GET /v1/models` sem token: HTTP 401 esperado; com token: HTTP 200.
- `GET /v1/models?api_format=anthropic` com token: HTTP 200.
- OpenAI SDK efemero via `uv --with openai`: `MiniMax-M3`, `deepseek-v4-flash` e `gpt-5.5` OK pelo dominio publico.
- Anthropic SDK efemero via `uv --with anthropic`: `MiniMax-M3` e `deepseek-v4-flash` OK pelo dominio publico.
- `model-detailed-hotfix` nao participa mais do caminho runtime; nao use comportamento antigo desse middleware como gate de validacao.
- MiniMax-M3 burst probe sem delay: 16/20 HTTP 200 e 4/20 HTTP 529 upstream `The server cluster is currently under high load`; com delay de 0.25s, 8/8 HTTP 200.

## Backup e restore de tabela

Backups automaticos de escrita ficam em:

```bash
/home/ubuntu/GitHub/containers/router-ai-atius/backups/clianything/
```

Backup manual:

```bash
bin/clianything backup channels
```

Restore e manual e deve ser feito somente em janela operacional, com backup atual e rollback definido:

```bash
ls -lh backups/clianything/*_channels.sql
podman exec -i postgres psql -U admin -d DBRouterAiAtius -v ON_ERROR_STOP=1 < backups/clianything/ARQUIVO_channels.sql
bin/clianything get channels --id ID --format json
bin/clianything status
```

## Recovery

Se o backend Go precisar reiniciar:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
systemctl --user restart container-router-ai-atius.service
bin/clianything status
```

Se o pod sumir ou a unit falhar, nao usar comandos destrutivos sem snapshot. Primeiro capturar estado:

```bash
podman ps --filter pod=atius-ai-router
podman pod ps
systemctl --user status container-router-ai-atius.service --no-pager
```

Depois:

```bash
bin/clianything status
bin/clianything providers --all
```

## Cuidados

- Nao rodar `docker system prune` ou `podman system prune` em producao sem backup verificado.
- Nao editar `data/postgres_data` diretamente.
- Nao expor `channels.key`, `tokens.key`, `users.password`, `custom_oauth_providers.client_secret`, `two_fas.secret`, `passkey_credentials.*` ou overrides de headers.
- Antes de update/delete pelo CLI, sempre guardar o caminho do backup impresso.
