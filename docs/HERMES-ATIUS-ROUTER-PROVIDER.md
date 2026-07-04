# Hermes Agent via Atius Router

Este runbook descreve a configuracao validada do Hermes Agent para usar o
Atius Router como provider OpenAI-compatible para os modelos GPT/Codex.

Ultima validacao: 2026-07-01 19:14 America/Sao_Paulo.

## Objetivo

- Fazer o Hermes Agent chamar o Atius Router em `/v1/chat/completions`.
- Evitar que modelos GPT/Codex sejam enviados como Anthropic-compatible.
- Preservar os modelos publicados pelo router:
  - `gpt-5.5`
  - `gpt-5.4`
  - `gpt-5.4-1m`
  - `gpt-5.4-mini`
  - `gpt-5.3-codex-spark`
- Manter `gpt-5.5-1m` desabilitado enquanto nao passa na validacao 1M no
  endpoint Codex/OAuth atual.

## Arquivos

- Config principal do Hermes: `/home/ubuntu/.hermes/config.yaml`
- Cache de janelas de contexto: `/home/ubuntu/.hermes/context_length_cache.yaml`
- Router repo: `/home/ubuntu/GitHub/containers/router-ai-atius`
- Provider Codex do router: channel `5`, nome `OpenAI - Codex`, tipo `57`

## Configuracao correta

O bloco principal do Hermes deve usar `provider: custom`, URL raiz do router e
`api_mode: chat_completions`.

```yaml
model:
  provider: custom
  default: gpt-5.4-1m
  max_tokens: 64000
  context_length: 1048576
  base_url: https://router.atius.com.br
  api_key: ${ATIUS_ROUTER_API_KEY}
  api_mode: chat_completions
  aliases:
    gpt-5.4-1m: custom:atius-router/gpt-5.4-1m
    gpt-5.4: custom:atius-router/gpt-5.4
    gpt-5.5: custom:atius-router/gpt-5.5
```

O custom provider nomeado deve apontar para o mesmo endpoint:

```yaml
custom_providers:
  - name: Atius-Router
    api_mode: chat_completions
    base_url: https://router.atius.com.br
    api_key: ${ATIUS_ROUTER_API_KEY}
    models:
      gpt-5.4-1m:
        context_length: 1048576
        max_tokens: 128000
      gpt-5.4:
        context_length: 1048576
        max_tokens: 128000
      gpt-5.5:
        context_length: 272000
        max_tokens: 128000
      gpt-5.4-mini:
        context_length: 272000
        max_tokens: 128000
      gpt-5.3-codex-spark:
        context_length: 128000
        max_tokens: 32000
```

Nao usar `base_url: https://router.atius.com.br/v1` com
`api_mode: chat_completions`. O cliente OpenAI-compatible monta
`/v1/chat/completions`; se a base ja termina em `/v1`, a chamada pode virar
`/v1/v1/chat/completions`.

Nao usar `api_mode: anthropic_messages` para os modelos GPT/Codex do Atius
Router. Esse modo usa `/v1/messages` e representa Anthropic-compatible.

## Modelo e alias

No Hermes, `model.aliases` evita que `hermes -m gpt-5.5` seja autodetectado por
catalogos externos ou por outro provider. Os aliases diretos fazem o override
resolver para `custom:atius-router`.

Comandos equivalentes:

```bash
hermes --ignore-rules -m gpt-5.4-1m -z 'Responda apenas OK.'
hermes --ignore-rules -m gpt-5.4 -z 'Responda apenas OK.'
hermes --ignore-rules -m gpt-5.5 -z 'Responda apenas OK.'
```

Tambem funciona com provider explicito:

```bash
hermes --ignore-rules --provider custom -m gpt-5.5 -z 'Responda apenas OK.'
hermes --ignore-rules --provider custom:atius-router -m gpt-5.5 -z 'Responda apenas OK.'
hermes --ignore-rules --provider Atius-Router -m gpt-5.5 -z 'Responda apenas OK.'
```

## Router esperado

`GET /v1/models` autenticado deve mostrar:

- `gpt-5.5`: presente
- `gpt-5.5-1m`: ausente
- `gpt-5.4`: presente
- `gpt-5.4-1m`: presente
- `gpt-5.4-mini`: presente

O canal `OpenAI - Codex` deve expor:

```text
gpt-5.5,gpt-5.4,gpt-5.4-1m,gpt-5.4-mini,gpt-5.3-codex-spark
```

E a tabela `channels` deve manter:

```json
{"gpt-5.4-1m":"gpt-5.4"}
```

Esse mapping garante que o cliente pode pedir `gpt-5.4-1m`, enquanto o upstream
real recebe `gpt-5.4`. Logs, usage, quota e billing continuam registrando o
modelo solicitado pelo cliente.

## Validacao

Resolver do Hermes:

```bash
/home/ubuntu/.hermes/hermes-agent/venv/bin/python - <<'PY'
import sys
sys.path.insert(0, '/home/ubuntu/.hermes/hermes-agent')
from hermes_cli.runtime_provider import resolve_runtime_provider
for model in ['gpt-5.4-1m', 'gpt-5.4', 'gpt-5.5']:
    rt = resolve_runtime_provider(target_model=model)
    print(model, rt.get('provider'), rt.get('api_mode'), rt.get('base_url'))
PY
```

Resultado esperado:

```text
custom chat_completions https://router.atius.com.br
```

Router HTTP:

```bash
export ROUTER_BASE_URL='https://router.atius.com.br'
export ROUTER_TEST_KEY='COLOQUE_UM_TOKEN_DE_TESTE_AQUI'

curl -s "$ROUTER_BASE_URL/v1/models" \
  -H "Authorization: Bearer $ROUTER_TEST_KEY" \
  | jq '.data[] | select(.id | test("gpt-5\\.[45](-1m)?"))'
```

Chat pequeno:

```bash
curl -s "$ROUTER_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $ROUTER_TEST_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-1m",
    "messages": [{"role": "user", "content": "Responda apenas OK."}],
    "max_completion_tokens": 16,
    "stream": false
  }' | jq .
```

Uso interno:

```bash
bin/clianything logs --limit 20 --format json \
  | jq '.[] | select(.model_name | test("gpt-5\\.[45](-1m)?"))'
```

Esperado: `channel_id=5` para os modelos GPT/Codex do router.

## Warnings conhecidos

- `hermes config check` ainda informa `Config version: 28 -> 32`. Isso nao
  bloqueia o provider.
- O gateway pode emitir warning de `service_tier 'auto'`; isso e configuracao
  do Hermes, nao do router.
- `obsidian-rest` pode falhar no startup do gateway; isso nao altera a rota de
  inferencia do Atius Router.
- O unit `hermes-telegram.service` pode estar defasado em `TimeoutStopSec`. Nao
  foi regenerado neste ajuste para evitar alterar servico fora do escopo.
