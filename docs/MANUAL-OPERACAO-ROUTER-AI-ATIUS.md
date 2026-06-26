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

Registro dedicado do corte full-Go 2026-06-18: `docs/FULL-GO-V1-MODELS-CUTOVER-2026-06-18.md`.

## Status atual validado

- Podman pod: `atius-ai-router`
- Containers em runtime atual: `router-ai-atius`, `postgres`, `redis`, infra pause. O antigo `model-detailed-hotfix` foi parado em 2026-06-18 apos cutover full-Go.
- Backend local: `http://127.0.0.1:3000`
- Nao ha model proxy Python ativo no caminho `/v1/`.
- Alias Apache para `/v1/`: `http://127.0.0.1:3000`
- Publico: `https://router.atius.com.br`
- Deploy full-Go + base URL `/v1` normalization 2026-06-18: imagem `ghcr.io/giovannimnz/router-ai-atius:latest` id `e389110f98fb8e3fce80ac8cf691a04c1c74b6268d91d5fb304bb6f574344151`.
- Rollback imediato antes da normalizacao `/v1`: `ghcr.io/giovannimnz/router-ai-atius:rollback-before-baseurl-v1-normalize-20260618122124`.
- Rollback provider consolidation: `ghcr.io/giovannimnz/router-ai-atius:rollback-before-provider-consolidation-20260618080648`.

Validacao rapida:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
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

Validacao:

```bash
apache2ctl configtest
curl -sI https://router.atius.com.br/api/status | head -1
curl -sI http://127.0.0.1:3000/api/status | head -1
curl -sI https://router.atius.com.br/health | head -1
```

## Providers e roteamento

Estado validado pelo banco:

| Channel | Tipo | Status | Base URL | Modelos |
|---|---:|---:|---|---|
| `MiniMax` | `35` | enabled | `https://api.minimax.io` | `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.7` |
| `DeepSeek` | `43` | disabled | `https://api.deepseek.com` | desativado temporariamente: upstream retornou `401 invalid api key` |
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
- Os campos enriquecidos publicos esperados por item sao `pricing`, `input_price`, `output_price`, `supported_endpoint_types`, `supported_endpoint_type_labels` e `billing_mode` quando disponivel.
- `pricing_version` e campo interno e nao deve aparecer no payload publico de `/v1/models`.
- `pricing_source` e `pricing_estimated` ficam internos e nao devem aparecer no payload publico de `/v1/models`.
- A ordenacao publica e deterministica: texto antes de embeddings; providers MiniMax, DeepSeek e OpenAI/OpenAI Codex; dentro do provider, modelos mais recentes/capazes primeiro.
- Regras de variante: versao numerica maior sobe; `-highspeed` fica acima da variante sem `-highspeed`; `pro` fica acima de `flash`, que fica como variante secundaria.
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
| `gpt-5.4` | 2.50 | 15.00 | estimado/standard |
| `gpt-5.4-mini` | 0.75 | 4.50 | estimado/standard |
| `gpt-5.3-codex-spark` | 1.75 | 14.00 | estimado |
| `embo-01` | 0.069 | 0.069 | estimado |
| `text-embedding-3-small` | 0.02 | 0.02 | OpenAI |
| `text-embedding-3-large` | 0.13 | 0.13 | OpenAI |

Comandos de verificacao:

```bash
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
- Para embeddings OpenAI-compatible, usar `/v1/embeddings` somente quando o modelo estiver anunciado em `/v1/models`. No runtime atual, o provider ativo e `Local TEI - GTE Embeddings` com o alias publico `embedding-pt-v1`, dimensao `768`, apontando para `http://10.1.1.4:3000`.
- Provider de embeddings MiniMax: canal unico `MiniMax` tipo `35` com `embo-01`; a conversao OpenAI -> MiniMax native acontece no adaptador Go, mas o modelo fica fora de `channels.models` ate o RPM/quota upstream estabilizar.
- Provider de embeddings Codex: `OpenAI - Codex` channel 5 com `text-embedding-3-small` e `text-embedding-3-large`; fica desativado ate a quota/licenca upstream aceitar chamadas reais.
- Nao reativar `OpenAI - Embeddings` separado nem depender de chave OpenAI duplicada para estes modelos; a regra do fork e compartilhar a credencial Codex OAuth.
- MiniMax e DeepSeek nao precisam de canais duplicados por protocolo: o tipo do canal identifica o provider e o relay Go escolhe URL/formato conforme o endpoint recebido (`/v1/chat/completions`, `/v1/messages`, `/v1/embeddings`).
- `/v1/models` sem token deve retornar 401. Isso nao indica falha.
- DeepSeek esta desativado no runtime atual porque o upstream retornou `401 invalid api key`. Reativar somente apos trocar a chave e passar a matriz UAT.
- MiniMax embeddings pode retornar `429` quando o upstream bloquear por quota/RPM persistente; nesse caso manter `embo-01` fora do catalogo ativo ate o smoke rigido passar.
- `MiniMax` e `DeepSeek` devem preferir `base_url` no provider root (`https://api.minimax.io`, `https://api.deepseek.com`), mas o relay Go tambem aceita base URL com `/v1` e normaliza automaticamente. Nao usar sufixos especificos como `/anthropic` no canal consolidado.

## Governor de embeddings Go-native

Estado atualizado em 2026-06-26:

- O governor de embeddings roda dentro do proprio processo Go do router; nao ha sidecar, middleware Python, container adicional ou rota `model-detailed` no caminho canonico.
- Implementacao principal: `service/embeddinggovernor/` e integracao em `relay/embedding_handler.go`.
- Escopo default: `embedding-pt-v1` e `embedding-pt-v1-batch`. Outros modelos passam pelo relay normal sem fila do governor.
- Default operacional: concorrencia inicial `1`, minima `1`, maxima `4`, batch limitado a `1`, fila interativa `128`, fila batch `512`, timeout interativo `30s`, timeout batch `5m`, cooldown `10m`.
- O header opcional `X-Embedding-Workload: batch` classifica a chamada como batch; `interactive`/`realtime` forcam tratamento interativo. Sem header, `embedding-pt-v1-batch` e batch e `embedding-pt-v1` e interativo.
- Em erro upstream `5xx`, erro de request, ou chamada acima de `EMBEDDING_GOVERNOR_SLOW_REQUEST_DURATION`, o governor reduz a concorrencia para o minimo e entra em cooldown. A reabertura e gradual por janela de sucesso e por demanda interativa saudavel.
- O governor mantem EWMA de latencia, contadores de falha/lentidao, picos de fila/execucao e timestamps de escala. Ele pode subir antes de uma janela fixa quando ha fila interativa, nao ha cooldown/falha recente e a latencia media esta abaixo do alvo.
- Quando a fila esta cheia ou expira antes de despachar para o upstream, o router retorna `429` com codigo `embedding_governor_queue_full`, `embedding_governor_batch_queue_full` ou `embedding_governor_queue_timeout`.

Variaveis de ambiente suportadas:

```bash
EMBEDDING_GOVERNOR_ENABLED=true
EMBEDDING_GOVERNOR_MODELS=embedding-pt-v1,embedding-pt-v1-batch
EMBEDDING_GOVERNOR_BATCH_MODELS=embedding-pt-v1-batch
EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY=1
EMBEDDING_GOVERNOR_MIN_CONCURRENCY=1
EMBEDDING_GOVERNOR_MAX_CONCURRENCY=4
EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=1
EMBEDDING_GOVERNOR_QUEUE_LIMIT=128
EMBEDDING_GOVERNOR_BATCH_QUEUE_LIMIT=512
EMBEDDING_GOVERNOR_INTERACTIVE_TIMEOUT=30s
EMBEDDING_GOVERNOR_BATCH_TIMEOUT=5m
EMBEDDING_GOVERNOR_COOLDOWN=10m
EMBEDDING_GOVERNOR_SLOW_REQUEST_DURATION=2m
EMBEDDING_GOVERNOR_LATENCY_TARGET=90s
EMBEDDING_GOVERNOR_SCALE_UP_MIN_INTERVAL=30s
EMBEDDING_GOVERNOR_SCALE_DOWN_IDLE=10m
EMBEDDING_GOVERNOR_SUCCESS_WINDOW=8
```

Base empirica dos threads `019f010f-421b-7243-ac95-46c2b287e868` e `019f017f-51fb-7ea1-972a-d4b91434be7d`:

- Inicio do GBrain: `3667 stale chunks`, `Embedded: 0`, schema migrado de `1536` para `768`.
- Tentativas com paginas grandes deram timeout sem gravar vetores; por isso o governor nao deve iniciar acima de `1`.
- `GBRAIN_EMBED_PROVIDER_BATCH_SIZE=4` foi o sub-batch que passou em slug que falhava; esse e o limite operacional seguro para catch-up.
- Com carga viva, TEI ficou tipicamente em `~115%` a `148%` CPU e `~1.4GiB` RSS; em tentativa mais pesada chegou a `~7.8GiB` RSS, ainda sem OOM, mas com erro upstream/rate-limit e reinicio/unready.
- Em host de 4 cores, load observado `~5.3` a `>6` durante concurrency `2`, ou seja, pressao CPU ja acima de `1.3x` a `1.5x` a capacidade nominal. Memoria sobrou; gargalo principal e CPU/readiness do TEI.
- Resultado operacional: batch/catch-up fica conservador em `1`; o teto `4` e apenas ceiling adaptativo para fila interativa saudavel, nao baseline.

Validacao minima apos mudanca no governor:

```bash
go test ./service/embeddinggovernor ./relay -count=1
ATIUS_ROUTER_TOKEN=... python3 scripts/smoke-embeddings.py --base-url http://127.0.0.1:3000/v1
```

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

Exemplo conceitual, sem secrets:

```yaml
model:
  provider: custom
  default: MiniMax-M3
  base_url: ${ATIUS_ROUTER_BASE_URL}
  api_mode: anthropic_messages
fallback_providers:
  - provider: custom
    model: MiniMax-M3
```

No `.env`, `ATIUS_ROUTER_BASE_URL` deve apontar para `https://router.atius.com.br`.

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
- Router Go deve rotear Anthropic/OpenAI automaticamente via canal unico do provider quando o provider estiver ativo. Runtime atual: `MiniMax` channel 1 ativo; `DeepSeek` channel 2 desativado por `401 invalid api key`.
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
- `~/.gbrain/config.json` deve usar `embedding_model: openai:embedding-pt-v1` e `embedding_dimensions: 768`.
- O GBrain deve chegar direto ao Go router; o endpoint ativo de embeddings e o canal `Local TEI - GTE Embeddings`, alias `embedding-pt-v1`, upstream `http://10.1.1.4:3000`.
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
export ATIUS_ROUTER_TOKEN='<token operacional>'
ATIUS_ROUTER_ACTIVE_ONLY=1 python3 scripts/smoke-provider-consolidation.py
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" 'http://127.0.0.1:3000/v1/models?api_format=anthropic'
```

Sem `ATIUS_ROUTER_TOKEN`, os scripts de smoke retornam `exit 2` antes de chamar rede.

Ultima bateria operacional estrita em 2026-06-18:

- MiniMax ativo: `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.7` passaram via OpenAI-compatible e Anthropic-compatible.
- OpenAI - Codex ativo: `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark` passaram via OpenAI-compatible.
- DeepSeek fica desativado porque a chave upstream cadastrada retornou `401 invalid api key`.
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
