# Manual operacional - router-ai-atius

## Caminhos importantes

| Item | Caminho |
|---|---|
| Deploy ativo | `/home/ubuntu/GitHub/containers/router-ai-atius` |
| Runtime data/logs montados no pod | `/home/ubuntu/GitHub/containers/router-ai-atius/data`, `/home/ubuntu/GitHub/containers/router-ai-atius/logs` |
| Compose source of truth | `/home/ubuntu/GitHub/containers/router-ai-atius/podman-compose.yml` |
| Admin / fork-sync | `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router` |
| Recovery script | `/home/ubuntu/GitHub/containers/router-ai-atius/scripts/recreate-pod.sh` |
| CLI operacional | `/home/ubuntu/GitHub/containers/router-ai-atius/bin/clianything` |
| Docs locais | `/home/ubuntu/GitHub/containers/router-ai-atius/docs` |

## Status atual validado

- Podman pod: `atius-ai-router`
- Containers em runtime atual: `router-ai-atius`, `model-detailed-hotfix`, `postgres`, `redis`, infra pause
- Backend local: `http://127.0.0.1:3000`
- Model proxy local: `http://127.0.0.1:3001`
- Alias Apache para `/v1/`: `http://127.0.0.1:3300`
- Publico: `https://router.atius.com.br`

Validacao rapida:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
bin/clianything status
bin/clianything coverage --strict
bin/clianything providers --all
```

`bin/clianything status --strict` e mais rigoroso: ele falha quando algum check fica `fail` ou `degraded`. Em 2026-06-15, apos o ajuste do middleware, `model-detailed` responde `healthy` porque o healthcheck nao depende mais de `/v1/models` autenticado.

## Rotas Apache relevantes

Estado observado em 2026-06-15:

- `https://router.atius.com.br/api/` e `/` apontam para `127.0.0.1:3000`.
- `https://router.atius.com.br/v1/` aponta para `127.0.0.1:3300/v1/`, que cai no container `model-detailed-hotfix` e repassa ao backend.
- Docs/i18n usam porta `3003` quando o serviço de docs esta ativo.

Validacao:

```bash
apache2ctl configtest
curl -sI https://router.atius.com.br/api/status | head -1
curl -sI http://127.0.0.1:3000/api/status | head -1
curl -sI http://127.0.0.1:3001/health | head -1
```

## Providers e roteamento

Estado validado pelo banco:

| Channel | Tipo | Status | Base URL | Modelos |
|---|---:|---:|---|---|
| `MiniMax - OpenAI-Compatible` | `1` | enabled | `https://api.minimax.io` | `MiniMax-M3` |
| `DeepSeek - OpenAI-Compatible` | `43` | enabled | `https://api.deepseek.com` | `deepseek-v4-flash`, `deepseek-v4-pro` |
| `MiniMax - Anthropic-Compatible` | `14` | enabled | `https://api.minimax.io/anthropic` | `MiniMax-M2.7`, `MiniMax-M2.5`, `MiniMax-M2.5-highspeed`, `MiniMax-M3` |
| `DeepSeek - Anthropic-Compatible` | `14` | enabled | `https://api.deepseek.com/anthropic` | `deepseek-v4-flash`, `deepseek-v4-pro` |
| `OpenAI Codex OAuth` | `57` | enabled | OAuth local / sem `base_url` | `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark` |
| `MiniMax - Embeddings` | `1` | enabled | `https://api.minimax.io` | `embo-01` |
| `OpenAI - Embeddings` | `1` | disabled | `https://api.openai.com/v1` | `text-embedding-3-small`, `text-embedding-3-large` |

Comando:

```bash
bin/clianything providers --all
```

Endpoints ativos:

- OpenAI-Compatible: `/v1/models`, `/v1/chat/completions`, `/v1/responses`
- Anthropic-Compatible: `/v1/messages`
- Embeddings: `/v1/embeddings`

## Catalogo `/v1/models` Go-owned e precificacao

Estado validado em 2026-06-15:

- O backend Go e o dono do catalogo enriquecido em `/v1/models`; o middleware Python nao e fonte do contrato de model-list.
- O root publico de `/v1/models` em modos model-list contem somente `data`: nao expor top-level `object`, `success`, `first_id`, `last_id` ou `has_more`.
- Os campos enriquecidos publicos esperados por item sao `pricing`, `input_price`, `output_price`, `supported_endpoint_types`, `supported_endpoint_type_labels`, `billing_mode` e `pricing_version` quando disponiveis.
- `pricing_source` e `pricing_estimated` ficam internos e nao devem aparecer no payload publico de `/v1/models`.
- A ordenacao publica e deterministica: texto antes de embeddings; providers MiniMax, DeepSeek e OpenAI/OpenAI Codex; dentro do provider, modelos mais recentes/capazes primeiro.
- `api_format=anthropic` e headers Anthropic selecionam modelos Anthropic-capable no catalogo Go, mantendo o mesmo root `{"data":[...]}`.
- Graphify fica obrigatorio no fluxo GSD desta area: status fresco antes/depois de mudancas em codigo, docs ou `.planning/`.
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

Regras praticas:

- Para clientes OpenAI-Compatible, usar `/v1/chat/completions`, `/v1/responses`, `/v1/models` etc com `base_url=https://router.atius.com.br/v1`.
- Para clientes Anthropic-Compatible, usar `/v1/messages` com `base_url=https://router.atius.com.br` ou `https://router.atius.com.br/v1` conforme SDK/cliente.
- Para embeddings locais, usar o proxy `model-detailed-hotfix` em `http://127.0.0.1:3001/v1`; nele o adapter converte `embo-01` para resposta OpenAI-compatible.
- Provider funcional de embeddings em 2026-06-15: `MiniMax - Embeddings` com `embo-01`. `OpenAI - Embeddings` esta desabilitado ate existir API key/quota OpenAI valida.
- `model-detailed-hotfix` converte OpenAI-Compatible para Claude Messages quando o channel final e Anthropic-Compatible.
- `/v1/models` sem token deve retornar 401. Isso nao indica falha.
- DeepSeek roteia para os modelos `deepseek-v4-flash` e `deepseek-v4-pro`; se o upstream retornar `402 Insufficient Balance`, o roteamento local esta funcionando e o bloqueio e financeiro.
- MiniMax embeddings passa pela fila anti-rate-limit do middleware, mas ainda pode retornar `429` quando o upstream bloquear por quota/RPM persistente; isso nao deve ser mascarado como erro local do router.
- `MiniMax - OpenAI-Compatible` usa `base_url=https://api.minimax.io` no NewAPI local; nao usar `/v1` nesse campo porque o backend ja monta o path OpenAI-compatible.

## Fila anti-rate-limit do middleware

O `model-detailed-hotfix` tem fila local por familia de provider/modelo antes de encaminhar para o NewAPI e para os upstreams. Ela reduz rajadas concorrentes e repete apenas respostas transitorias de rate/high-load, como `429`, `529`, `502`, `503` e `504`, sem retryar erros de quota, balance, token invalido ou modelo inexistente.

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

Valido em 2026-06-15:

- OpenAI SDK local contra `http://127.0.0.1:3000/v1` com `MiniMax-M3`: OK.
- Anthropic SDK local contra `http://127.0.0.1:3000` com `MiniMax-M3`: OK.
- OpenAI SDK local contra `http://127.0.0.1:3000/v1` com `gpt-5.5` via `OpenAI Codex OAuth`: OK com `ATIUS_ROUTER_STREAM=1`.
- Router log mostrou conversao `OpenAI-Compatible` -> `Claude Messages` via channel 3 e channel 7 quando aplicavel.
- `OpenAI Codex OAuth` esta ativo no channel 5 com modelos `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini` e `gpt-5.3-codex-spark`.
- `data/codex-home/.codex/auth.json` existe e tem `auth_mode: chatgpt`; esse arquivo e credencial de runtime e nao deve ser copiado para docs/logs.
- Os smoke scripts ficam em `scripts/smoke-openai-sdk.py`, `scripts/smoke-anthropic-sdk.py`, `scripts/smoke-embeddings.py` e `scripts/smoke-routing-matrix.py`; todos exigem `ATIUS_ROUTER_TOKEN` via env.
- Para smoke de embeddings, o default local e `http://127.0.0.1:3001/v1`.

Nao documentar ou copiar tokens desse arquivo em notes, docs ou logs.

## GBrain

Estado validado em 2026-06-15:

- Wrapper ativo: `/home/ubuntu/.local/bin/gbrain`.
- O wrapper define `GBRAIN_STATEMENT_TIMEOUT=0`, `GBRAIN_IDLE_TX_TIMEOUT=0`, `OPENAI_BASE_URL=http://127.0.0.1:3001/v1` e `OPENAI_API_KEY` a partir de `~/.gbrain/config.json`.
- `~/.gbrain/config.json` usa `embedding_model: openai:embo-01` e `embedding_dimensions: 1536`.
- O GBrain chega ao router corretamente; o bloqueio atual do provider e `rate limit exceeded(RPM)` do MiniMax em `embo-01`.
- Backup antes da mudanca: `/home/ubuntu/.gbrain/config.json.bak-router-embeddings-20260615_030827`.

## Operacao diaria

```bash
# Saude geral
bin/clianything status

# Providers/modelos ativos
bin/clianything providers
bin/clianything embeddings
bin/clianything models --from-channels
bin/clianything coverage --strict

# Ultimos logs sem payload sensivel
bin/clianything logs --limit 50

# Logs de containers
podman logs router-ai-atius --tail 80
podman logs model-detailed-hotfix --tail 80

# Restart controlado do backend Go
systemctl --user restart container-router-ai-atius.service

# Restart controlado do middleware /models
systemctl --user restart container-model-detailed.service
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

Smoke SDK local:

```bash
export ATIUS_ROUTER_TOKEN='<token operacional>'
python3 scripts/smoke-openai-sdk.py
python3 scripts/smoke-anthropic-sdk.py
python3 scripts/smoke-embeddings.py
ATIUS_ROUTER_MODEL=gpt-5.5 ATIUS_ROUTER_STREAM=1 python3 scripts/smoke-openai-sdk.py
```

Sem `ATIUS_ROUTER_TOKEN`, os scripts retornam `exit 2` antes de importar SDK ou chamar rede.

Ultima bateria operacional em 2026-06-15:

- DeepSeek OpenAI-compatible: `deepseek-v4-flash` e `deepseek-v4-pro` OK.
- OpenAI Codex OAuth: `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini` OK; `gpt-5.3-codex-spark` retornou upstream `429 The usage limit has been reached`.
- MiniMax/DeepSeek Anthropic-compatible: todos OK.
- MiniMax-M3 rate probe: 8/8 OK apos reabilitar `MiniMax - OpenAI-Compatible`, latencia min 732 ms, p50 1009 ms, max 1456 ms.
- Embeddings `embo-01` query/db: roteamento OK, upstream `429 rate limit exceeded(RPM)`.

Checkpoint publico via Cloudflare em 2026-06-15:

- `https://router.atius.com.br/health`: HTTP 200, `{"status":"healthy","backend":"connected"}`, com header `server: cloudflare`.
- `https://router.atius.com.br/api/status`: HTTP 200.
- `GET /v1/models` sem token: HTTP 401 esperado; com token: HTTP 200.
- `GET /v1/models?api_format=anthropic` com token: HTTP 200.
- OpenAI SDK efemero via `uv --with openai`: `MiniMax-M3`, `deepseek-v4-flash` e `gpt-5.5` OK pelo dominio publico.
- Anthropic SDK efemero via `uv --with anthropic`: `MiniMax-M3` e `deepseek-v4-flash` OK pelo dominio publico.
- `model-detailed-hotfix` remove blocos `thinking`/`<think>` das respostas Anthropic; validado com `MiniMax-M2.7` e `MiniMax-M3`.
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

Se o pod sumir:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
podman-compose up -d
```

Se `podman-compose` falhar:

```bash
bash scripts/recreate-pod.sh
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
