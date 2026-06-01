# Atius AI Router — Getting Started

## Prerequisites

| Requirement | Version | Notes |
|------------|---------|-------|
| Docker | 24+ | |
| Docker Compose | 2.20+ | |
| Git | any recent | for clone |
| curl | any | for testing |
| Python 3.10+ | optional | only if modifying middleware |
| Go 1.22+ | optional | only if modifying backend |

## 1. Clone e Setup

```bash
git clone https://github.com/giovannimnz/router-ai-atius.git
cd atius-ai-router
```

## 2. Configurar Ambiente

```bash
# Arquivo de variáveis de ambiente
cat .env

# Edite com suas credenciais
nano .env
```

Variáveis obrigatórias:

```bash
# Banco (gerado automaticamente pelo docker-compose)
POSTGRES_USER=admin
POSTGRES_PASSWORD=<sua_senha>
POSTGRES_DB=newapi

# Security
SESSION_SECRET=<random_secret_32_chars>

# Proxy (se atrás de Cloudflare/Apache)
TRUST_PROXY=true
```

## 3. Subir Containers

```bash
# Sobe todos os serviços (new-api, model-detailed, db-newapi)
docker compose up -d

# Verificar status
docker compose ps

# Saída esperada:
# NAME             IMAGE                            STATUS
# model-detailed   router-ai-atius-model-detailed   Up 7 hours
# new-api          ghcr.io/giovannimnz/...          Up 7 hours
# db-newapi        postgres:15-alpine                Up 7 hours
```

## 4. Verificar Health

```bash
# Middleware (enriquecido) — porta 3300
curl -s http://localhost:3300/api/status | python3 -m json.tool | head -20

# New-API direto — porta 3301
curl -s http://localhost:3301/api/status | python3 -m json.tool | head -20
```

Resposta esperada (200 OK):
```json
{
    "data": {
        "api_info": [
            {
                "color": "blue",
                "description": "Chat Completions API...",
                "route": "OpenAI Compatible",
                "url": "https://router.atius.com.br/v1/chat/completions"
            }
        ]
    }
}
```

## 5. Obter Token de Acesso

Tokens são criados via interface admin do NewAPI:

```bash
# Via curl (registrar + login)
curl -X POST http://localhost:3300/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"..."}'

curl -X POST http://localhost:3300/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"..."}'
```

Ou via interface web em `https://router.atius.com.br`.

## 6. Testar Endpoints

```bash
TOKEN="<SEU_TOKEN>"

# Listar modelos (enriquecidos)
curl http://localhost:3300/v1/models \
  -H "Authorization: Bearer $TOKEN"

# Chat completion
curl -X POST http://localhost:3300/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "messages": [{"role": "user", "content": "Hi"}],
    "max_tokens": 50
  }'

# Mensagem Anthropic
curl -X POST http://localhost:3300/v1/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "x-api-key: $TOKEN" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.7",
    "max_tokens": 50,
    "messages": [{"role": "user", "content": "Hi"}]
  }'
```

## 7. Usar com SDKs

### Python (OpenAI)

```bash
pip install openai
```

```python
from openai import OpenAI

client = OpenAI(
    api_key="<TOKEN>",
    base_url="https://router.atius.com.br/v1"
)

response = client.chat.completions.create(
    model="MiniMax-M2.7",
    messages=[{"role": "user", "content": "Hi"}],
    max_tokens=50
)
print(response.choices[0].message.content)
```

### Python (Anthropic)

```bash
pip install anthropic
```

```python
import anthropic

client = anthropic.Anthropic(
    api_key="<TOKEN>",
    base_url="https://router.atius.com.br/v1"
)

message = client.messages.create(
    model="MiniMax-M2.7",
    max_tokens=50,
    messages=[{"role": "user", "content": "Hi"}]
)
print(message.content[0].text)
```

## 8. Rodar Bruno Tests

```bash
# Todos os testes
./scripts/run-bruno-tests.sh

# Com saída verbose
./scripts/run-bruno-tests.sh --verbose

# Teste específico
bruno tests/atius-router-tests/list-models.bru --env .env
```

Suite localizada em `integration/bruno-tests/atius-router-tests/`.

## 9. Acesso ao Dashboard Admin

```bash
# URL
https://router.atius.com.br

# Funcionalidades:
# - Gerenciar channels
# - Verusage de tokens
# - Configurar rates e quotas
# - Dashboard de billing
```

## 10. shutdown

```bash
# Parar todos os containers
docker compose down

# Parar e remover volumes (PERDE DADOS)
docker compose down -v

# restart
docker compose restart
docker compose restart new-api
docker compose restart model-detailed
```

## Troubleshooting

| Problema | Solução |
|----------|---------|
| `Connection refused` em :3300 | Verificar `docker compose ps`, `docker compose logs model-detailed` |
| `Invalid token` | Token inválido ou expirado. Obter novo via `/api/user/login` |
| 521 (Cloudflare) | Apache backend offline. Verificar `systemctl status apache2` |
| Bruno tests falham | Verificar se containers estão `Up` e se new-api responde em :3301 |
| Middleware não enriches models | Ver logs: `docker compose logs model-detailed`, checar `:3001` alcançável |

## Próximos Passos

- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — entender como funciona por dentro
- [docs/CONFIGURATION.md](CONFIGURATION.md) — configurar rates, modelos, channels
- [docs/DEVELOPMENT.md](DEVELOPMENT.md) — modificar código
- [docs/TESTING.md](TESTING.md) — adicionar testes

---

_Last updated: 2026-05-31_
