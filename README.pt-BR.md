# Atius AI Router

<!-- Badges -->
[![Licença](https://img.shields.io/github/license/giovannimnz/router-ai-atius)](https://github.com/giovannimnz/router-ai-atius)
[![Versão](https://img.shields.io/github/v/tag/giovannimnz/router-ai-atius?filter=v*)](https://github.com/giovannimnz/router-ai-atius/releases)
[![New-API](https://img.shields.io/badge/New--API-0.12.14-blue)](https://github.com/QuantumNous/new-api)
[![i18n](https://img.shields.io/badge/i18n-7%20locales-green)](#internacionalização)
[![Runtime](https://img.shields.io/badge/Podman-compatible-purple)](#runtime-de-containers)

> **Gateway LLM unificado** que agrega **MiniMax, DeepSeek** e **40+ provedores de IA** atrás de uma única **API compatível com OpenAI / Anthropic**. Fork do [QuantumNous/new-api](https://github.com/QuantumNous/new-api) preparado para produção no ecossistema **Atius Capital**.

---

## Resumo

| Item | Valor |
|------|-------|
| URL do fork | <https://github.com/giovannimnz/router-ai-atius> |
| URL upstream | <https://github.com/QuantumNous/new-api> |
| Versão do fork | `0.12.14.2` (base `0.12.14` + sufixo `.2`) |
| Modelos mais recentes | `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.7-hs`, `DeepSeek-V3.2-Exp` |
| Stack | NewAPI (Go 1.22+) · middleware FastAPI · PostgreSQL 15 · Podman / Docker |
| Porta padrão | `3301` (NewAPI), `3300` (middleware) |
| URL pública | `https://router.atius.com.br` (Cloudflare → Apache → :3300/:3301) |

---

## Por que este fork existe

`QuantumNous/new-api` é um excelente gateway open-source, mas a implantação Atius precisa de:

1. **Um middleware Python** que enriquece `/v1/models` com metadados por modelo (faixas de preço, janelas de contexto, flags de capacidade) — dados que o binário Go upstream não gera.
2. **Um filtro strip CJK** que impede que respostas MiniMax em chinês / japonês / coreano vazem para a saída de clientes em português / inglês (`v1.7`, planejado).
3. **Branding Atius** (logo, footer, títulos "Atius Router", atribuição "Atius Capital") consistente entre a SPA, assets embedded e respostas API.
4. **Suporte first-class a Podman** — todo o stack roda em quadlets Podman na malha Atius.
5. **i18n bilíngue (en + pt-BR)** com validação profunda e garantia 0/0/0 de sync por locale.

Todo o resto é idêntico ao upstream — incluindo preços, billing, canais, parsing de modelos, OAuth, WebAuthn.

---

## Stack Técnica

| Camada | Tecnologia | Por quê |
|--------|------------|---------|
| **Gateway** | Go 1.22+ · Gin · GORM v2 | HTTP de alta vazão, pouca memória, fácil de deployar como um único binário estático |
| **Frontend** | React 19 · TypeScript · Rsbuild · Base UI · Tailwind CSS | SSR desabilitado (SPA CSR-only), embedded no binário Go via `embed` |
| **Middleware** | Python 3.11+ · FastAPI · Pydantic v2 | Enriquecimento de modelos de alto nível + integração fácil com libs de data science |
| **Banco** | PostgreSQL 15 (container `db-newapi`) | Maduro, ACID, colunas JSON para regras flexíveis de billing |
| **Cache** | Redis (go-redis) + in-memory LRU | Rate limiting token bucket + cache de respostas |
| **Auth** | JWT · WebAuthn / Passkeys · OAuth (GitHub, Discord, OIDC) | Todos suportados upstream |
| **Orquestração** | Podman quadlets · Docker Compose (legado) | Podman é o runtime de produção |
| **i18n** | i18next v26 + react-i18next + 7 locales | en, zh, fr, ja, pt-BR, ru, vi — sincronizados 0/0/0 |

---

## Modelos Disponíveis

O router expõe actualmente **6 modelos MiniMax** mais **2 modelos DeepSeek** na configuração padrão. Todos os modelos MiniMax são roteados pelo **canal Atius-MiniMax** no PostgreSQL.

| Modelo | Provider | Contexto | Max Output | Streaming | Tools | Notas |
|--------|----------|----------|------------|-----------|-------|-------|
| `MiniMax-M3` | MiniMax | 1.048.576 | 64.000 | ✅ | ✅ | Flagship — 1M de contexto, raciocínio profundo |
| `MiniMax-M2.7` | MiniMax | 245.760 | 50.000 | ✅ | ✅ | Padrão de produção |
| `MiniMax-M2.7-highspeed` | MiniMax | 245.760 | 50.000 | ✅ | ✅ | Variante de alta vazão |
| `MiniMax-M2.7-hs` | MiniMax | 245.760 | 50.000 | ✅ | ✅ | Alias curto de `M2.7-highspeed` |
| `MiniMax-M2.5` | MiniMax | 245.760 | 50.000 | ✅ | ✅ | Produção legada |
| `MiniMax-M2.5-highspeed` | MiniMax | 245.760 | 50.000 | ✅ | ✅ | Legado alta vazão |
| `deepseek-chat` | DeepSeek | 131.072 | 8.192 | ✅ | ❌ | Chat classe raciocínio |
| `deepseek-reasoner` | DeepSeek | 131.072 | 65.536 | ✅ | ❌ | Raciocínio long-form |

### MiniMax-M3 — 1M de contexto, o novo flagship

`MiniMax-M3` é a mais recente API MiniMax. O fork traz metadata de primeira classe para ele no middleware (`model_detailed.py`) e na base de dados (abilidades do canal). Contexto de 1.048.576 tokens habilita análise de documentos longos, workflows agentic multi-turn, e raciocínio sobre bases de código inteiras.

### Aliases `M2.7-hs` / `M2.5-hs`

`M2.7-hs` e `M2.5-hs` são **aliases curtos vendor-friendly** mapeados para os irmãos `-highspeed`. O mapeamento está configurado no **canal NewAPI** (coluna `model_mapping`, JSONB) e no **middleware Atius-Router** (tabela `KNOWN_MODELS`) para que ambas as camadas os resolvam consistentemente.

---

## Runtime de Containers — Podman-first

A infra-estrutura Atius roda em **Podman** (`podman 4.x+`) com **quadlets geridos por systemd**. Docker Compose é suportado para desenvolvimento.

| Componente | Container | Imagem | Porta (host) | Network |
|------------|-----------|--------|--------------|---------|
| Gateway | `new-api` | `ghcr.io/giovannimnz/router-ai-atius:local` | `3301` | `newapi-internal`, `atius-shared` |
| Middleware | `model-detailed` | `router-ai-atius-model-detailed:latest` | `3300` | `newapi-internal`, `atius-shared` |
| Banco | `db-newapi` | `postgres:15-alpine` | (interno apenas) | `newapi-internal` |

### Exemplo de quadlet Podman (`.container`)

```ini
[Unit]
Description=Atius Router (new-api)
After=network-online.target

[Container]
Image=ghcr.io/giovannimnz/router-ai-atius:local
PublishPort=3301:3000
Network=newapi-internal.network
Network=atius-shared.network
Volume=/srv/Atius/router/data:/data:Z
EnvironmentFile=/srv/Atius/router/.env
Environment=TZ=America/Sao_Paulo
Environment=LANG=pt_BR.UTF-8
AutoUpdate=registry
HealthCmd=curl -fsS http://localhost:3000/api/status
HealthInterval=30s

[Service]
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker Compose (desenvolvimento)

```bash
docker compose up -d
docker compose ps
docker compose logs -f new-api
```

---

## Arquitectura de Roteamento

```
┌────────────────────────────────────────────────────────────────────┐
│                  Apache 2.4 (router.atius.com.br:443)              │
│  Cloudflare proxy → Let's Encrypt SSL → vhosts/site-routing       │
└────────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
        ▼                       ▼                       ▼
   /v1/*               /docs  /openapi.json      /api/*  /  (SPA)
        │                       │                       │
        ▼                       ▼                       ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  FastAPI middle  │  │  FastAPI middle  │  │    NewAPI (Go)   │
│   (porta 3300)    │  │   (porta 3300)    │  │   (porta 3301)   │
│                  │  │                  │  │                  │
│ Enriquecimento   │  │ Schema OpenAPI   │  │  Relay / billing │
│ CJK strip (v1.7) │  │ API reference    │  │  Canais / auth   │
│ Healthchecks     │  │                  │  │  Admin dashboard │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                                │                       │
                                └───────────┬───────────┘
                                            ▼
                              ┌──────────────────┐
                              │   PostgreSQL 15   │
                              │   (db-newapi)     │
                              │                  │
                              │ users · tokens   │
                              │ channels · logs  │
                              │ abilities · etc  │
                              └──────────────────┘
```

### Fluxo de request do cliente (chat completions OpenAI)

```
Cliente (SDK OpenAI / Anthropic)
   │
   ├─► POST /v1/chat/completions ──► Apache vhost
   │                                            │
   │                                            ▼
   │                                  NewAPI (Go) :3301
   │                                            │
   │                                            ├─► Verificar JWT (token → user)
   │                                            ├─► Middleware Distributor
   │                                            │     Token → Abilities → Canal
   │                                            ├─► RelayFormat (OpenAI/Claude/Gemini)
   │                                            └─► Adaptador upstream (minimax)
   │                                                         │
   │                                                         ▼
   │                                                api.minimax.io/anthropic/v1
   │                                                         │
   │                                            ◄── streaming SSE ◄──┤
   │
   └─► POST /v1/messages ──► mesmo caminho com RelayFormatClaude
```

### Fluxo de enriquecimento /v1/models

```
Cliente GET /v1/models
   │
   ▼
Apache → middleware :3300
   │
   ├─► Consultar NewAPI /api/models (lado Go, canais + abilities)
   ├─► Mesclar com metadata KNOWN_MODELS (model_detailed.py)
   │     - context_length, max_tokens
   │     - flags de capacidade (tools, vision, audio)
   │     - faixa de preço (input/output/cache)
   │     - mapa de aliases de vendor (M2.7-hs → M2.7-highspeed)
   ├─► Strip CJK se v1.7 estiver habilitada (planejado)
   └─► Resposta JSON compatível OpenAI com metadata enriquecido
```

---

## Middleware Python — `model-detailed`

`model-detailed` é um serviço **FastAPI** que:

1. **Enriquece `/v1/models`** com metadata por modelo (flags de capacidade, preços, janelas de contexto).
2. **Resolve aliases** como `M2.7-hs` → `M2.7-highspeed` para que a listagem de `/v1/models` seja consistente.
3. **Expõe `/docs`** e `/openapi.json` para introspecção do schema OpenAPI 3.1.
4. **Fornece healthchecks** via `/healthz` e `/readyz` (usados pela secção `healthcheck` do `docker-compose.yml`).

### Ficheiro-chave: `model_detailed.py`

```python
KNOWN_MODELS = {
    "MiniMax-M3": {
        "context_length": 1_048_576,
        "max_tokens": 64_000,
        "supports_tools": True,
        "supports_vision": False,
        "tier": "flagship",
    },
    "MiniMax-M2.7-highspeed": {
        "context_length": 245_760,
        "max_tokens": 50_000,
        "supports_tools": True,
        "tier": "standard-highspeed",
    },
    # ... M2.7, M2.5, M2.5-highspeed, M2.7-hs, M2.5-hs
}
```

O middleware reescreve o payload de `/v1/models` do NewAPI no local — **zero mudanças necessárias** no binário Go.

---

## Filtro Strip CJK — v1.7 (planejado)

As respostas upstream do MiniMax às vezes contêm **caracteres CJK** (chinês / japonês / coreano) vazando do output do modelo, mesmo quando o prompt do usuário é em português ou inglês. Exemplos observados:

- `重新生成` (chinês para "regenerar")
- `もう一度` (japonês para "mais uma vez")

**Plano v1.7:** Adicionar um post-filter regex no relay NewAPI que remove caracteres CJK do texto de resposta antes de devolver ao cliente. Implementação em `.planning/phases/v1.7-cjk-strip-filter/PLAN.md`.

```go
var cjkRegex = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{3000}-\x{303f}\x{ff00}-\x{ffef}]`)

func StripCJK(s string) string {
    return cjkRegex.ReplaceAllString(s, "")
}
```

Toggleável por canal via `ChannelSettings.StripCJK bool`.

---

## Internacionalização (i18n)

O frontend traz **7 locales**, todos **sincronizados 0/0/0** (missing/extras/untranslated):

| Locale | Code | Cobertura | Source strings |
|--------|------|-----------|----------------|
| English (base) | `en` | 100% | 4.525 chaves |
| Chinese | `zh` | 100% | 4.525 chaves |
| French | `fr` | 100% | 4.525 chaves |
| Japanese | `ja` | 100% | 4.525 chaves |
| **Português Brasileiro** | `pt-BR` | **94% traduzido**, 6% brand/tech mantidos em EN | 4.525 chaves |
| Russian | `ru` | 100% | 4.525 chaves |
| Vietnamese | `vi` | 100% | 4.525 chaves |

A tradução `pt-BR` é a contribuição da comunidade que planejamos enviar upstream como "agradecimento" à QuantumNous pela base open-source.

### Sync report (forçado por `bun run i18n:sync`)

```json
{
  "base": "pt-BR.json",
  "locales": {
    "pt-BR": { "missingCount": 0, "extrasCount": 0, "untranslatedCount": 0 },
    "en":    { "missingCount": 0, "extrasCount": 0, "untranslatedCount": 0 },
    "zh":    { "missingCount": 0, "extrasCount": 0, "untranslatedCount": 0 },
    ...
  }
}
```

### Tests (vitest, 17 passando)

```
✓ src/i18n/__tests__/locales-integrity.test.ts (4 tests)
✓ src/i18n/__tests__/languages-config.test.ts (3 tests)
✓ src/i18n/__tests__/i18n-runtime.test.ts (7 tests)
✓ src/components/__tests__/language-switcher.test.tsx (2 tests)
✓ src/components/ui/dropdown-menu.test.tsx (2 tests)
```

Corra com: `bun run test`

---

## Endpoints Principais

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/v1/chat/completions` | Chat completions (compat. OpenAI) |
| POST | `/v1/messages` | Messages (compat. Anthropic) |
| POST | `/v1/completions` | Completions (legado) |
| POST | `/v1/embeddings` | Embeddings |
| POST | `/v1/audio/speech` | Text-to-Speech |
| POST | `/v1/audio/transcriptions` | Speech-to-Text |
| POST | `/v1/images/generations` | Geração de imagens |
| POST | `/v1/rerank` | Rerank |
| GET  | `/v1/models` | Listar modelos (**enriquecido** pelo middleware) |
| GET  | `/healthz` | Liveness do middleware |
| GET  | `/readyz` | Readiness do middleware |
| GET  | `/docs` | OpenAPI 3.1 docs interactivos (Swagger UI) |
| GET  | `/openapi.json` | Schema OpenAPI 3.1 (machine-readable) |
| GET  | `/api/status` | Estado do sistema + info da API |
| POST | `/api/user/register` | Registo de utilizador |
| POST | `/api/user/login` | Login |

### URLs Públicas vs Internas

| Serviço | Pública (Cloudflare → Apache) | Interna (network de containers) |
|---------|------------------------------|----------------------------------|
| Middleware | `https://router.atius.com.br/v1/*` (Apache → :3300) | `http://model-detailed:3001` |
| NewAPI | `https://router.atius.com.br/api/*` (Apache → :3301) | `http://new-api:3000` |
| NewAPI SPA | `https://router.atius.com.br/` (Apache → :3301) | n/a |
| PostgreSQL | n/a (não exposto) | `postgres://db-newapi:5432` |

---

## Quick Start

### 1. Clonar

```bash
git clone https://github.com/giovannimnz/router-ai-atius.git
cd router-ai-atius
```

### 2. Configurar

```bash
cp .env.example .env
# Edite .env com as suas API keys
```

Vars de ambiente requeridas:

| Var | Origem | Notas |
|-----|--------|-------|
| `MINIMAX_API_KEY` | Dashboard MiniMax | Chave do Token Plan |
| `DEEPSEEK_API_KEY_*` | Dashboard DeepSeek | 1 chave por canal DeepSeek |
| `POSTGRES_PASSWORD` | Auto-gerada | Usada pelo NewAPI para conectar ao `db-newapi` |
| `TELEGRAM_BOT_TOKEN` | BotFather | Para integração de notificações Telegram |

### 3. Subir

```bash
# Podman (produção)
podman play kube deployment.yaml

# Docker (desenvolvimento)
docker compose up -d
```

### 4. Verificar

```bash
docker compose ps
curl http://localhost:3301/api/status
curl http://localhost:3300/healthz
```

### 5. Smoke-test dos modelos

```bash
TOKEN=$(docker exec db-newapi psql newapi admin -t -c "SELECT key FROM tokens WHERE name = 'GiovanniMuniz';")
curl -X POST http://localhost:3301/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"MiniMax-M3","messages":[{"role":"user","content":"Olá"}],"max_tokens":50}'
```

---

## Exemplos de Uso

### OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    api_key="SEU_TOKEN",
    base_url="https://router.atius.com.br/v1",
)

resp = client.chat.completions.create(
    model="MiniMax-M3",
    messages=[{"role": "user", "content": "2+2=?"}],
    max_tokens=10,
)
print(resp.choices[0].message.content)
```

### Anthropic SDK (via /v1/messages)

```python
import anthropic

client = anthropic.Anthropic(
    api_key="SEU_TOKEN",
    base_url="https://router.atius.com.br",
)

msg = client.messages.create(
    model="MiniMax-M3",
    max_tokens=10,
    messages=[{"role": "user", "content": "2+2=?"}],
)
print(msg.content[0].text)
```

### Cherry Studio / CC Switch

O router anuncia-se como **compatível com OpenAI / Anthropic**. Para conectar o Cherry Studio ou CC Switch:

1. Defina base URL para `https://router.atius.com.br/v1`
2. Defina API key como o seu token
3. Escolha qualquer modelo de `/v1/models` (listagem enriquecida)

---

## Branding Atius

Todos os assets de branding são bundled com o build:

- **`/logo.png`** + **`/logo.svg`** — Logo Atius Router (substitui o logo new-api upstream no nav e admin)
- **`/favicon.ico`** — Favicon Atius
- **`<title>Atius Router</title>`** — título da página
- **`<meta name="description" content="Unified AI API gateway and admin dashboard.">`**
- **Footer** — "© 2026 Atius Capital. Todos os direitos reservados."

O branding é aplicado em **build time** via `Dockerfile`:

```dockerfile
# ATIUS BRANDING: replace ALL embedded assets in dist AFTER builder copy
RUN find ./web/default/dist -type f \( -name "*.png" -o -name "*.ico" -o -name "*.svg" \) -delete 2>/dev/null || true
COPY web/default/public/logo.png ./web/default/dist/logo.png
COPY web/default/public/logo.svg ./web/default/dist/logo.svg
COPY web/default/public/favicon.ico ./web/default/dist/favicon.ico
```

Assim, o binário Go `new-api` em runtime serve os **assets Atius** sem zero alterações de configuração.

---

## Configuração de Canal

O canal MiniMax é configurado na base de dados (tabela `channels`) com:

```sql
UPDATE channels SET
  test_model = 'MiniMax-M2.7',
  models = '["MiniMax-M3","MiniMax-M2.7","MiniMax-M2.7-highspeed","MiniMax-M2.7-hs","MiniMax-M2.5","MiniMax-M2.5-highspeed"]'::jsonb,
  model_mapping = '{}'::jsonb
WHERE id = 1;
```

| Campo | Valor | Por quê |
|-------|-------|---------|
| `type` | `35` (custom) | Tipo de relay MiniMax |
| `base_url` | `https://api.minimax.io` | Endpoint MiniMax |
| `key` | `sk-cp-...` (Bearer) | Encriptado em repouso com AES-256-GCM |
| `test_model` | `MiniMax-M2.7` | Usado pelo healthcheck do canal |
| `models` | Todos os 6 modelos MiniMax | O Distributor combina estes com as abilities do token |
| `model_mapping` | `{}` | Vazio (sem tradução de alias necessária) |

---

## Topologia de Deployment

```
                    ┌──────────────┐
                    │  Cloudflare  │
                    │   (proxy)    │
                    └──────┬───────┘
                           │ HTTPS
                           ▼
                    ┌──────────────┐
                    │    Apache    │
                    │  (router.    │
                    │   atius)     │
                    └──────┬───────┘
                           │ vhost routing
        ┌──────────────────┼──────────────────┐
        │ /v1/* /docs      │ /api/*           │ /
        ▼                  ▼                  ▼
  ┌──────────┐       ┌──────────┐       ┌──────────┐
  │  model-  │       │  new-api │       │  new-api │
  │ detailed │       │  :3301   │       │  :3301   │
  │  :3300   │       │   (Go)   │       │   (SPA)  │
  └─────┬────┘       └─────┬────┘       └─────┬────┘
        │                  │                  │
        └────────────┬─────┴──────────────────┘
                     ▼
              ┌──────────────┐
              │  PostgreSQL  │
              │  (db-newapi) │
              └──────────────┘
```

3 containers, 1 base de dados, 1 reverse proxy, 1 CDN.

---

## Operações de Manutenção

### Sincronizar fork com upstream

```bash
./fork-sync/bin/sync.sh
```

Puxa o último `QuantumNous/new-api:main`, faz merge, corre `i18n:sync`, corre `bun run test`, builda, reinicia.

### Correr a suite de testes

```bash
cd web/default
bun run test        # vitest, 17 tests
bun run typecheck   # tsc -b
bun run lint        # eslint
bun run i18n:sync   # garantir 0/0/0
```

### Rebuild e redeploy

```bash
docker build -t ghcr.io/giovannimnz/router-ai-atius:local .
docker stop new-api
docker rm new-api
docker run -d --name new-api --network router-ai-atius_newapi-internal --network atius-shared -p 3301:3000 \
  -v $(pwd)/data:/data --env-file .env ghcr.io/giovannimnz/router-ai-atius:local
```

### Purgar cache do Cloudflare (após deploy)

```bash
ZONE=5b998a5d911f5a4102b6179df7f4518d
curl -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE/purge_cache" \
  -H "X-Auth-Email: $CF_AUTH_EMAIL" -H "X-Auth-Key: $CF_GLOBAL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"prefixes":["router.atius.com.br/static/"]}'
```

---

## Licença

GNU Affero General Public License v3.0 — veja [LICENSE](LICENSE).

Herdado do [QuantumNous/new-api](https://github.com/QuantumNous/new-api) (AGPL-3.0).

O Atius Router é um fork comunitário; **o branding upstream é preservado** em comentários no source e no documento `FORK.md`.

---

## Referências

- [QuantumNous/new-api](https://github.com/QuantumNous/new-api) — upstream
- [MiniMax API Reference](https://platform.minimax.io/docs/api-reference)
- [OpenAI API](https://platform.openai.com/docs/api-reference)
- [Anthropic API](https://docs.anthropic.com/en/api)
- [Podman Quadlets](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html)
- [i18next v26 docs](https://www.i18next.com/)
