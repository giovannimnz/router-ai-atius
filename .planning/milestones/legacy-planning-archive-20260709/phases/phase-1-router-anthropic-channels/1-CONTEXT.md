# Phase 1 Context — Router Anthropic Channels + Session Fix

##Objetivo
1. Adicionar 2 canais Anthropic-compatible (type=14) para MiniMax e MiniMax-highspeed
2. Aumentar session timeout de 30 dias (2592000s) para 12 horas (43200s)
3. Investigar e corrigir problema do /docs/ na router dashboard

## Estado Atual

### Canais existentes
| ID | Nome | Type | Base URL | Models |
|----|------|------|----------|--------|
| 1 | MiniMax - Token Plan | 0 (OpenAI) | https://api.minimax.io | MiniMax-M2.7, M2.7-hs, M2.5, M2.5-hs |
| 2 | DeepSeek API | 0 (OpenAI) | https://api.deepseek.com | deepseek-v4-flash, deepseek-v4-pro |

### Session timeout atual
- Valor: `MaxAge: 2592000` (30 dias)
- Local: `main.go` linha ~181
- Cookie name: `session`
- Provider: `gin-contrib/sessions` com `cookie.NewStore`

### Tipo de canal Anthropic
- `ChannelTypeAnthropic = 14` (definido em `constant/channel.go`)
- Base URL padrão: `https://api.anthropic.com` (não serve para MiniMax)
- MiniMax usa gateway próprio em `https://api.minimax.io`

### Timeout de sessão
- 12 horas = 43200 segundos
- Mudar `MaxAge: 2592000` → `MaxAge: 43200`

### Problema /docs/
- HTTP 200 localmente (`curl http://127.0.0.1:3300/docs/` → 200)
- HTTP 200 via Cloudflare (`curl https://router.atius.com.br/docs/` → 200)
- Dashboard redireciona para `/sign-in` antes de carregar `/docs/`
- Possível causa: cookie session expira muito rápido ou não persiste

## Decisões

### Decisão 1: Anthropic channels para MiniMax
- Usar type=14 (Anthropic) com base_url=https://api.minimax.io
- Necessário custom model_mapping para direcionar requests Anthropic para o endpoint MiniMax correto
- MiniMax suporta formato Claude (messages API) via endpoint próprio?

### Decisão 2: Session timeout 12h
- Alterar `main.go` `MaxAge` de `2592000` para `43200`
- Rebuild da imagem Docker necessária

### Decisão 3: /docs/ não carrega
- InvestigarApache proxy para /docs vs Cloudflare caching
- Testar com browser automation para identificar onde falla

## Arquivos a modificar

1. `main.go` — session MaxAge (linha 181)
2. `docker-compose.yml` — rebuild com mudanças
3. 数据库 — inserir canais Anthropic (type=14) via SQL

## Modelo de dados

### channels table
```sql
id | name | type | base_url | key | models | status | model_mapping | ...
```

### abilities table
```sql
"group" | model | channel_id | enabled | priority | weight
```

## Comandos de verificação

```bash
# Verificar canais
docker exec db-newapi psql -U admin -d newapi -c "SELECT id, name, type, base_url FROM channels;"

# Verificar session config
grep -n 'MaxAge' /home/ubuntu/docker/Atius/router-ai-atius/main.go

# Testar /docs/
curl -s -o /dev/null -w '%{http_code}' https://router.atius.com.br/docs/
```