# CLIAnything - gestao 100% por CLI do router-ai-atius

Arquivo principal:

```bash
/home/ubuntu/GitHub/containers/router-ai-atius/bin/clianything
```

O CLI foi criado para operar o deployment vivo do `router-ai-atius` sem depender da UI. Ele usa `podman exec postgres psql` para os recursos persistidos no Postgres, comandos tipados para operacoes de dominio, mapeamento dos endpoints administrativos documentados e wrappers seguros para chamadas HTTP locais da API.

## Modelo de seguranca

- Saida normal redige campos sensiveis: keys, tokens, passwords, secrets, cookies, override headers, payloads privados e configuracoes que podem carregar credenciais.
- Escritas sao dry-run por padrao. `create`, `update` e `delete` so alteram o banco com `--execute`.
- Antes de qualquer escrita com `--execute`, o CLI cria backup `pg_dump --data-only --column-inserts` em:

```bash
/home/ubuntu/GitHub/containers/router-ai-atius/backups/clianything/
```

- O comando `query` aceita somente SQL read-only (`select`, `with`, `show`, `explain`) e bloqueia verbos de escrita.
- Para endpoints de API que exigem autenticacao, recupere tokens do HashiCorp Vault machine/automation e passe via `--bearer`, `ATIUS_ROUTER_ADMIN_TOKEN` ou `ATIUS_ROUTER_TOKEN` somente no processo atual; nao salve tokens em docs, shell history compartilhado, logs ou issues.
- `clianything api`, `clianything endpoint invoke` e `clianything call` tambem sao dry-run para `POST`, `PUT`, `PATCH`, `DELETE` e endpoints classificados como `api-action`, mesmo quando o metodo HTTP e `GET`. No caso de `api`, essa classificacao vale quando o path bate no manifesto; path fora do manifesto segue protegido por metodo mutante.
- Respostas JSON e corpos HTTP nao-JSON passam por redaction antes de imprimir.
- `clianything coverage --strict` falha se a documentacao MDX e o manifesto de paridade divergirem, se qualquer endpoint depender de `cli_command` iniciado por `clianything api`, ou se a arvore de docs management nao estiver presente no checkout.

## Gates obrigatorios

Use estes comandos antes de declarar paridade do CLI:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
python3 -m py_compile tools/clianything.py scripts/smoke-provider-consolidation.py scripts/smoke-embeddings.py
python3 -m json.tool tools/clianything_endpoints.json >/dev/null
python3 -m unittest discover -s tests -p 'test_clianything*.py'
# Quando docs/atius-router-docs/content/docs/en/api/management existir:
bin/clianything coverage --strict
ATIUS_ROUTER_ACTIVE_ONLY=1 python3 scripts/smoke-provider-consolidation.py
```

`bin/clianything status --strict` tambem deve ser rodado. O contrato atual de `/v1/models` e Go-owned: o backend Go monta o catalogo enriquecido e retorna root `{"data":[...]}`. `/v1/models` sem token continua retornando HTTP 401 esperado. O relay Go normaliza `base_url` com ou sem `/v1`, evitando paths duplicados como `/v1/v1/...`.

## Comandos principais

```bash
# Saude do pod, HTTP e DB
clianything status
clianything status --strict

# Cobertura CLI x API administrativa documentada
clianything coverage --strict
clianything coverage --details --format json

# Recursos cobertos
clianything resources

# Schema de qualquer recurso
clianything schema channels

# Listar registros
clianything list channels --limit 10
clianything list users --filter role=100 --limit 20

# Buscar por id ou filtro
clianything get channels --id 3 --format json
clianything get users --filter username=admin

# Exportar
clianything export channels --format json > channels.redacted.json
clianything export logs --limit 500 --format csv > logs.csv

# SQL read-only
clianything query 'select count(*) as total from channels' --format json

# Dry-run de update
clianything update channels --id 3 --set status=1

# Escrita real com backup automatico
clianything update channels --id 3 --set status=1 --execute

# Consolidacao Go-native de providers; compat legado do nome Phase 19
clianything channel phase19-apply --execute

# Clone manual fica disabled por default e bloqueia splits legados sem override explicito
clianything channel clone-keyed --source-id 2 --name 'DeepSeek Teste' --type 43 --base-url https://api.deepseek.com --models deepseek-v4-pro,deepseek-v4-flash

# Chamada HTTP local
clianything api GET /api/status
clianything api GET /api/channel/ --bearer "$ATIUS_ROUTER_ADMIN_TOKEN"

# Endpoint documentado pelo manifesto, com guards de dry-run/execute
clianything endpoint list --classification read-only
clianything endpoint show GET /api/channel/models
clianything endpoint invoke GET /api/channel/models --bearer "$ATIUS_ROUTER_ADMIN_TOKEN"
clianything call system.status-get --base-url http://127.0.0.1:3000
```

## Cobertura de endpoints administrativos

O manifesto de paridade fica em:

```bash
/home/ubuntu/GitHub/containers/router-ai-atius/tools/clianything_endpoints.json
```

Ele e gerado a partir dos MDX de management:

```bash
python3 -m json.tool tools/clianything_endpoints.json >/dev/null
bin/clianything coverage --strict
```

Estado validado em 2026-06-15:

- `159` arquivos `.mdx` de management.
- `158` endpoints de management documentados.
- `158` endpoints no manifesto.
- `100%` de cobertura, zero missing, zero extra, zero problema.
- `auth.mdx` e tratado como documento de referencia sem operation.
- Classificacoes atuais: `38 api-action`, `38 db-crud`, `43 read-only`, `36 auth-flow`, `3 external-webhook`, `0 unsupported-safe`.
- Paridade de `api-action`: `10` endpoints usam subcomandos de dominio e `28` usam `clianything endpoint invoke`; nenhum depende de `clianything api`.
- `NewAPI.apifox.json` nao e a fonte de verdade para management; ele cobre outra superficie (`/v1`, `/v1beta`, audio/video/image) e subconta os endpoints administrativos.

Classificacoes usadas:

| Classificacao | Significado | Padrao seguro |
|---|---|---|
| `db-crud` | Operacao CRUD que pode ser feita direto no Postgres com backup/dry-run. | `list/get` read-only; escrita exige `--execute`. |
| `api-action` | Executa logica de backend alem de CRUD simples. | Dry-run ate passar `--execute`. |
| `read-only` | Consulta sem mutacao esperada. | Pode executar direto. |
| `auth-flow` | Login, OAuth, 2FA, passkey ou verification flow. | Nao usar em automacao generica sem contexto. |
| `external-webhook` | Callback externo de pagamento/webhook. | Nao simular em rotina comum. |
| `unsupported-safe` | Endpoint conhecido, mas bloqueado pelo CLI por seguranca. | Fica bloqueado. |

Comandos uteis:

```bash
# Resumo por grupo/classificacao
clianything coverage

# Ver endpoints mapeados
clianything endpoint list --group channel-management
clianything endpoint show GET /api/channel/test/{id}

# Invocar endpoint documentado com parametros de path/query
clianything endpoint invoke GET /api/channel/test/{id} --param id=3 --execute
clianything endpoint invoke POST /api/models/sync_upstream
clianything call system.status-get --base-url http://127.0.0.1:3000
```

## Recursos cobertos

| Frente do frontend | Recurso CLI | Tabela |
|---|---|---|
| Channel Management / Providers | `channels`, `abilities`, `providers`, `embeddings` | `channels`, `abilities` |
| Model Management | `models`, `vendors` | `models`, `vendors` |
| System Settings | `options`, `setups` | `options`, `setups` |
| User Management | `users` | `users` |
| API Token | `tokens` | `tokens` |
| Usage Log / Analytics | `logs`, `usage_tracking`, `quota_data`, `perf_metrics` | respectivas tabelas |
| Task Log / Drawing Log | `tasks`, `midjourneys` | respectivas tabelas |
| Redemption | `redemptions` | `redemptions` |
| Wallet / Top-up / Subscription | `top_ups`, `subscription_plans`, `subscription_orders`, `user_subscriptions`, `subscription_pre_consume_records` | respectivas tabelas |
| Groups | `prefill_groups` ou alias `groups` | `prefill_groups` |
| OAuth | `custom_oauth_providers`, `user_oauth_bindings` | respectivas tabelas |
| 2FA / Passkeys | `two_fas`, `two_fa_backup_codes`, `passkey_credentials` | respectivas tabelas |
| Check-in | `checkins` | `checkins` |

## Atalhos de dominio

```bash
# Providers ativos e seus modelos/abilities
clianything providers

# Providers de embeddings e status
clianything embeddings

# Incluir channels desabilitados
clianything providers --all

# Model catalog
clianything models

# Modelos declarados nos channels
clianything models --from-channels

# Ultimos logs sem payload sensivel
clianything logs --limit 50

# Operacoes tipadas que espelham botoes/acoes do frontend
clianything channel test --id 3 --execute
clianything channel fetch-models --id 3 --execute
clianything channel balance --id 3 --execute
clianything channel enable --id 3
clianything channel enable --id 3 --execute
clianything model missing
clianything model sync-upstream --preview
clianything model sync-upstream --execute
clianything embeddings
clianything option get
clianything option set SystemName "Atius Router"
clianything ratio channels
clianything ratio fetch --execute
clianything token usage
clianything log stat
clianything task list
clianything vendor search minimax
```

## Workflow seguro de alteracao

1. Inspecione o estado atual:

```bash
clianything get channels --id 3 --format json
```

2. Rode dry-run:

```bash
clianything update channels --id 3 --set priority=0
```

3. Aplique com backup automatico:

```bash
clianything update channels --id 3 --set priority=0 --execute
```

4. Valide:

```bash
clianything providers --all
clianything status
```

## Backup e restore de tabela

Backups automaticos de escrita ficam em `backups/clianything/`. Tambem e possivel gerar backup manual:

```bash
clianything backup channels
```

Restore nao e automatizado pelo CLI para evitar sobrescrever producao por engano. Procedimento seguro:

```bash
# 1. Pare e confira a janela operacional.
# 2. Confirme o arquivo antes de executar.
ls -lh backups/clianything/*_channels.sql

# 3. Restaure manualmente no Postgres do pod.
podman exec -i postgres psql -U admin -d DBRouterAiAtius -v ON_ERROR_STOP=1 < backups/clianything/ARQUIVO_channels.sql

# 4. Valide pelo CLI.
clianything get channels --id ID --format json
clianything status
```

Nunca rode restore durante trafego de producao sem janela, backup atual e plano de rollback.

## Endpoints administrativos documentados

A documentacao Fumadocs gerada mapeia endpoints em:

```bash
/home/ubuntu/GitHub/containers/router-ai-atius/docs/atius-router-docs/content/docs/en/api/management/
```

Exemplos confirmados:

| Area | Endpoint |
|---|---|
| Channels | `GET /api/channel/`, `POST /api/channel/`, `PUT /api/channel/`, `DELETE /api/channel/:id`, test/fetch/batch/tag |
| Models | `GET /api/models/`, `POST /api/models/`, `PUT /api/models/`, `DELETE /api/models/:id`, sync upstream |
| Tokens | `GET /api/token/`, `POST /api/token/`, `PUT /api/token/`, `DELETE /api/token/:id`, batch |
| Options | `GET /api/option/`, `PUT /api/option/`, ratio sync/reset |
| Users | `GET /api/user/`, `POST /api/user/`, `PUT /api/user/`, delete/reset/passkey/2FA operations |
| Logs | `GET /api/log/`, search/stat/self/token, delete |

Quando a API precisa executar comportamento que nao e apenas persistencia de tabela, prefira um comando tipado ou `endpoint invoke`. Use `clianything api` apenas para diagnostico/adaptacao rapida:

```bash
clianything endpoint invoke METHOD /api/path --bearer "$ATIUS_ROUTER_ADMIN_TOKEN" --execute
clianything api METHOD /api/path --bearer "$ATIUS_ROUTER_ADMIN_TOKEN" --execute
```

Exemplo de guard em GET classificado como `api-action`:

```bash
clianything api GET /api/channel/test/1
# DRY-RUN API ... Nada foi enviado. Adicione --execute para aplicar.
```

## Smoke SDK

Os scripts abaixo nao embutem token; exigem `ATIUS_ROUTER_TOKEN` via env. Para automacao, a fonte canonica e o HashiCorp Vault path `kv/atius/srv1/shell-exports/home-ubuntu-merged`, lendo `ATIUS_ROUTER_API_KEY` e mapeando para `ATIUS_ROUTER_TOKEN` no shell efemero do teste:

```bash
VAULT_JSON="$(ssh -o BatchMode=yes atius-srv-3-vpn 'sudo /usr/local/sbin/atius-vault kv get -format=json kv/atius/srv1/shell-exports/home-ubuntu-merged')"
eval "$(printf '%s' "$VAULT_JSON" | python3 -c 'import json,shlex,sys; j=json.load(sys.stdin)["data"]["data"]["values"]; api=j["ATIUS_ROUTER_API_KEY"]; print("export ATIUS_ROUTER_TOKEN="+shlex.quote(api))')"

# Matriz ativa: modelos anunciados em /v1/models precisam responder 200.
ATIUS_ROUTER_ACTIVE_ONLY=1 scripts/smoke-provider-consolidation.py

# Embeddings legacy-only: default http://127.0.0.1:3000/v1, model embo-01.
scripts/smoke-embeddings.py

# Chat/model-list neste checkout: validar via curl ou SDK efemero
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" 'http://127.0.0.1:3000/v1/models?api_format=anthropic'
```

Overrides:

```bash
ATIUS_ROUTER_OPENAI_BASE_URL=http://127.0.0.1:3000/v1
ATIUS_ROUTER_ANTHROPIC_BASE_URL=http://127.0.0.1:3000
ATIUS_ROUTER_MODEL=MiniMax-M3
ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1
ATIUS_ROUTER_EMBEDDING_TYPE=query
```

## Validacao realizada em 2026-06-15

Comandos executados:

```bash
python3 -m py_compile tools/clianything.py scripts/smoke-provider-consolidation.py scripts/smoke-embeddings.py
python3 -m json.tool tools/clianything_endpoints.json
python3 -m unittest discover -s tests -p 'test_clianything*.py'
bin/clianything coverage --strict
bin/clianything resources
bin/clianything status
bin/clianything status --strict
bin/clianything providers
bin/clianything providers --all
bin/clianything schema channels --format json
bin/clianything list channels --limit 3
bin/clianything query 'select count(*) as channels from channels' --format json
bin/clianything update channels --id 1 --set priority=0
bin/clianything api GET /api/status
bin/clianything api GET /api/channel/test/1
bin/clianything api POST /api/setup --data '{}'
bin/clianything channel balance --id 1
ATIUS_ROUTER_ACTIVE_ONLY=1 python3 scripts/smoke-provider-consolidation.py
```

Resultado:

- Sintaxe Python ok.
- Manifesto JSON ok e cobertura strict `158/158` ok.
- DB `DBRouterAiAtius` acessivel via container `postgres`.
- Pod `atius-ai-router` running com 5 containers.
- Backend `/api/status` HTTP 200.
- Runtime full-Go validado em 2026-06-18: `model-detailed` parado; `clianything status` valida `backend`, `v1-models` e DB sem check de middleware Python.
- `/v1/models` sem token retorna HTTP 401 esperado.
- Dry-run de update e dry-run de `POST /api/setup` nao alteraram banco/API.
- Unit/integration tests: 37 OK, 1 skip intencional (`CLIANYTHING_RUN_BACKUP_TEST=1`).
- Smoke OpenAI/Anthropic SDK sem `ATIUS_ROUTER_TOKEN`: exit 2 esperado, sem importar SDK nem chamar rede.
- Smoke embeddings e routing matrix sem `ATIUS_ROUTER_TOKEN`: exit 2 esperado.
- Smoke real com token operacional via ambiente efemero `uv`: OpenAI SDK `MiniMax-M3` OK, Anthropic SDK `MiniMax-M3` OK, OpenAI SDK `gpt-5.5` OK com `ATIUS_ROUTER_STREAM=1`.
- O antigo middleware `model-detailed-hotfix` nao fica mais no caminho runtime de `/v1/`; nao use headers de fila como gate do estado full-Go.
- Go `/v1/models` agora enriquece pricing e metadados diretamente a partir do catalogo Go; o campo publico esperado para precos e `pricing` com `input` e `output`.
- O payload publico de `/v1/models` nao expoe `input_price`, `output_price`, `quota_type`, `enable_groups`, `supported_endpoint_type_labels` ou `pricing.unit`.
- Go relay normaliza `base_url` com ou sem `/v1`; providers MiniMax/DeepSeek tambem removem `/v1` antes de montar paths Anthropic/native.
- O root publico de `/v1/models` contem somente `data` em modos model-list. Nao expor top-level `object`, `success`, `first_id`, `last_id` ou `has_more`.
- `pricing_source`, `pricing_estimated` e `pricing_version` permanecem internos e nao sao campos publicos de `/v1/models`.
- A ordenacao publica e deterministica: modelos de texto antes de embeddings; providers MiniMax, DeepSeek e OpenAI/OpenAI Codex; dentro de cada provider, variantes mais recentes/capazes primeiro.
- `api_format=anthropic` ou headers Anthropic selecionam modelos Anthropic-capable pelo catalogo Go, preservando o mesmo root `{"data":[...]}`.
- O workflow GSD deste contrato usa Graphify quando habilitado; no checkout atual de runtime, `graphify status` retornou `disabled`, entao registrar indisponibilidade e validar por testes/CLI/smokes.
- Precificacao cadastrada no backend para `MiniMax-M2.1`, variantes MiniMax highspeed, Codex OAuth `gpt-5.5`/`gpt-5.4`/`gpt-5.4-mini`/`gpt-5.3-codex-spark`, `embo-01` e OpenAI embeddings `text-embedding-3-small`/`text-embedding-3-large`.
- Em 2026-06-18, `text-embedding-3-small` e `text-embedding-3-large` foram ligados conceitualmente ao `OpenAI - Codex` channel 5, compartilhando a credencial OAuth do Codex; runtime atual os deixa desativados porque o upstream retornou `429 insufficient_quota`.
- Em 2026-06-18, MiniMax e DeepSeek foram consolidados para um canal por provider. Runtime atual: `MiniMax` tipo `35` ativo para chat/messages; `DeepSeek` tipo `43` desativado por `401 invalid api key`; canais duplicados antigos seguem disabled.
- Testes unitarios cobrem conversao de `ModelRatio`/`CompletionRatio` para preco por token e enriquecimento de catalogo OpenAI/Anthropic/embeddings.
- Secret scan em `tools`, `scripts`, `tests`, docs e Phase 18 sem hits.
- Em 2026-06-15, `scripts/router-model-battery.py --token-id 8 --rate-requests 20 --rate-delay 0.2` validou MiniMax-M3 com 20/20 OK e embeddings `embo-01` roteando no caminho historico, bloqueado por upstream `429 rate limit exceeded(RPM)`.
- Em 2026-06-15, uma matriz efemera com OpenAI/Anthropic SDK validou o dominio publico `https://router.atius.com.br`: catalogos OpenAI/Anthropic OK, SDKs OK, Codex `gpt-5.5` OK, embeddings `embo-01` roteando mas bloqueados por upstream `429`.

Validacao rapida do catalogo enriquecido:

```bash
bin/clianything api GET /api/pricing --bearer "$ATIUS_ROUTER_ADMIN_TOKEN"
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models
curl -sS -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" 'http://127.0.0.1:3000/v1/models?api_format=anthropic'
ATIUS_ROUTER_ACTIVE_ONLY=1 scripts/smoke-provider-consolidation.py
```

Snapshot esperado em 2026-06-15:

- `MiniMax-M3`: input `$0.30/M`, output `$1.20/M`.
- `gpt-5.5`: input `$5.00/M`, output `$30.00/M`.
- `embo-01`: input/output `$0.069/M`.
- `text-embedding-3-small`: input/output `$0.02/M`.
