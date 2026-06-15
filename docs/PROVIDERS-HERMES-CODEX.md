# Providers, Hermes, Codex e SDKs - router-ai-atius

Este documento registra o estado tecnico validado em 2026-06-15 e como alternar entre Atius Router, Anthropic-Compatible e OpenAI-Compatible sem confundir provider ativo com credenciais existentes.

## Fonte de verdade

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
bin/clianything providers --all
bin/clianything status
podman ps --filter pod=atius-ai-router --format '{{.Names}}\t{{.Image}}\t{{.Status}}'
```

O banco `DBRouterAiAtius` e o frontend usam a tabela `channels` para providers e a tabela `abilities` para roteamento por modelo/grupo/prioridade.

## Estado validado

- OpenAI-Compatible:
  - channel `1`, `MiniMax - OpenAI-Compatible`, type `1`, base `https://api.minimax.io`, modelo `MiniMax-M3`.
  - channel `2`, `DeepSeek - OpenAI-Compatible`, type `43`, base `https://api.deepseek.com`, modelos `deepseek-v4-flash`, `deepseek-v4-pro`.
  - channel `5`, `OpenAI Codex OAuth`, type `57`, OAuth local, modelos `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark`.
- Anthropic-Compatible:
  - channel `3`, `MiniMax - Anthropic-Compatible`, type `14`, base `https://api.minimax.io/anthropic`, modelos `MiniMax-M2.7`, `MiniMax-M2.5`, `MiniMax-M2.5-highspeed`, `MiniMax-M3`.
  - channel `7`, `DeepSeek - Anthropic-Compatible`, type `14`, base `https://api.deepseek.com/anthropic`, modelos `deepseek-v4-flash`, `deepseek-v4-pro`.
- Embeddings:
  - channel `6`, `MiniMax - Embeddings`, type `1`, base `https://api.minimax.io`, modelo `embo-01`.
  - channel `8`, `OpenAI - Embeddings`, type `1`, base `https://api.openai.com/v1`, modelos `text-embedding-3-small`, `text-embedding-3-large`, status disabled ate haver API key/quota OpenAI valida.

Grafia correta: `Compatible` com hífen no nome do channel e `Embeddings` para a frente de embeddings.

## Endpoints ativos hoje

- OpenAI-Compatible: `GET /v1/models`, `POST /v1/chat/completions`, `POST /v1/responses`
- Anthropic-Compatible: `POST /v1/messages`
- Embeddings: `POST /v1/embeddings`
- Management/CLI: `bin/clianything providers --all`, `bin/clianything embeddings`, `bin/clianything models --from-channels`, `bin/clianything channel phase19-apply --execute`

O middleware `model-detailed-hotfix` usa fila anti-rate-limit por provider/model-family nos endpoints `POST /v1/chat/completions`, `POST /v1/responses`, `POST /v1/messages` e `POST /v1/embeddings`. Os headers `X-Atius-Rate-Queue`, `X-Atius-Rate-Queue-Wait-Ms` e `X-Atius-Rate-Retry-Count` indicam o bucket aplicado, espera acumulada e retries. A fila reduz burst local, mas nao bypassa quota ou saturacao persistente do upstream.

## Modelos ativos hoje

- `MiniMax-M3`
- `deepseek-v4-flash`
- `deepseek-v4-pro`
- `MiniMax-M2.7`
- `MiniMax-M2.5`
- `MiniMax-M2.5-highspeed`
- `gpt-5.5`
- `gpt-5.4`
- `gpt-5.4-mini`
- `gpt-5.3-codex-spark`
- `embo-01`
- `text-embedding-3-small` e `text-embedding-3-large` permanecem catalogados, mas o channel OpenAI esta disabled em 2026-06-15 por credencial/quota invalida.

## Cliente OpenAI SDK

Use base URL com `/v1`:

```python
import os

from openai import OpenAI

client = OpenAI(
    base_url="https://router.atius.com.br/v1",
    api_key=os.environ["ATIUS_ROUTER_TOKEN"],
)

resp = client.chat.completions.create(
    model="MiniMax-M3",
    messages=[{"role": "user", "content": "Responda OK"}],
)
print(resp.choices[0].message.content)
```

Localmente no servidor:

```bash
export ATIUS_ROUTER_OPENAI_BASE_URL=http://127.0.0.1:3000/v1
export ATIUS_ROUTER_MODEL=MiniMax-M3
export ATIUS_ROUTER_TOKEN='<token operacional>'
python3 scripts/smoke-openai-sdk.py
```

Para o channel `OpenAI Codex OAuth` via OpenAI SDK, use streaming:

```bash
export ATIUS_ROUTER_OPENAI_BASE_URL=http://127.0.0.1:3000/v1
export ATIUS_ROUTER_MODEL=gpt-5.5
export ATIUS_ROUTER_STREAM=1
export ATIUS_ROUTER_TOKEN='<token operacional>'
python3 scripts/smoke-openai-sdk.py
```

## Cliente Anthropic SDK

Use o endpoint Messages:

```python
import os

import anthropic

client = anthropic.Anthropic(
    base_url="https://router.atius.com.br",
    api_key=os.environ["ATIUS_ROUTER_TOKEN"],
)

resp = client.messages.create(
    model="MiniMax-M3",
    max_tokens=32,
    messages=[{"role": "user", "content": "Responda OK"}],
)
print(resp.content[0].text)
```

Localmente no servidor:

```bash
export ATIUS_ROUTER_ANTHROPIC_BASE_URL=http://127.0.0.1:3000
export ATIUS_ROUTER_MODEL=MiniMax-M3
export ATIUS_ROUTER_TOKEN='<token operacional>'
python3 scripts/smoke-anthropic-sdk.py
```

## Hermes apontando para Atius Router

Quando o objetivo for usar o router como broker:

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

`~/.hermes/.env`:

```bash
ATIUS_ROUTER_BASE_URL=https://router.atius.com.br
```

## Hermes apontando para Anthropic direto

Quando existir chave Anthropic e a intencao for bypassar o router:

```yaml
model:
  provider: anthropic
  default: claude-sonnet-4-5
fallback_providers:
  - provider: anthropic
    model: claude-sonnet-4-5
```

## Hermes apontando para OpenAI direto

```yaml
model:
  provider: openai
  default: gpt-5
fallback_providers:
  - provider: openai
    model: gpt-5
```

## Hermes custom OpenAI-compatible

Use para um endpoint que fala OpenAI-compatible:

```yaml
model:
  provider: custom
  default: gpt-5
  base_url: ${OPENAI_BASE_URL}
  api_mode: openai_chat
fallback_providers:
  - provider: custom
    model: gpt-5
```

## Observacoes de compatibilidade

- Hermes v0.16 nao considera `fallback_providers` como lista de strings. Use objetos com `provider` e `model`.
- A existencia de credenciais em `~/.hermes/auth` ou `data/codex-home/.codex/auth.json` nao significa, sozinha, que o provider esta ativo no router. Confirme sempre via `bin/clianything providers --all`.
- Em 2026-06-15, `OpenAI Codex OAuth` esta ativo no banco e mapeado para os modelos Codex listados acima.
- Smoke real validado em 2026-06-15: OpenAI SDK + `gpt-5.5` retorna OK quando `ATIUS_ROUTER_STREAM=1`; sem streaming, o upstream retorna `400 Stream must be set to true`.
- O roteamento de embeddings funcional inclui `MiniMax - Embeddings`; `OpenAI - Embeddings` esta disabled ate key/quota valida. DeepSeek embeddings continuam fora do catalogo ativo.
- Em 2026-06-13 20:05 BRT, o container real do proxy e `model-detailed-hotfix`; referencias de health ainda aparecem como `model-detailed` porque esse e o nome logico do check.
- Para trocar provider de producao, altere `channels`, `abilities` e modelos declarados, depois valide com `bin/clianything providers --all` e teste SDK.
- DeepSeek localmente roteia, mas o upstream pode retornar `402 Insufficient Balance`.
- MiniMax embeddings localmente roteia como `embo-01`, mas o upstream pode retornar `429` por rate limit.
- Em 2026-06-15, os quatro channels solicitados ficaram enabled: MiniMax OpenAI-Compatible, MiniMax Anthropic-Compatible, DeepSeek OpenAI-Compatible e DeepSeek Anthropic-Compatible. A base local correta do MiniMax OpenAI-Compatible e `https://api.minimax.io` sem `/v1`.
- GBrain em 2026-06-15 usa wrapper `/home/ubuntu/.local/bin/gbrain`, com `OPENAI_BASE_URL=http://127.0.0.1:3001/v1`, `embedding_model=openai:embo-01` no file-plane `~/.gbrain/config.json`, e chega ao router; o bloqueio observado e `rate limit exceeded(RPM)` do MiniMax.
- `bin/clianything coverage --strict` valida que os endpoints administrativos documentados continuam cobertos pelo manifesto CLI.
- `bin/clianything status` deve reportar `model-detailed=ok`; o healthcheck do middleware usa backend publico/status e trata `/v1/models` autenticado como fallback.
- `scripts/smoke-routing-matrix.py` pode ser executado com `uv run --with openai --with anthropic` para validar SDKs reais sem instalar dependencias globais.
- Nunca cole tokens reais em docs, vault, PRs ou logs.
