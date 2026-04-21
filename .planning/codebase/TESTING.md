# TESTING.md - Test Infrastructure & Strategy

## Visão Geral

O projeto possui uma **estratégia de teste operacional** focada em verificação de endpoints e saúde do stack, não em testes unitários de código. Isso se deve ao fato de que a aplicação NewAPI é uma imagem Docker pré-construída — não há código-fonte local para testes unitários.

## Tipos de Teste

### 1. Testes de Modelos (Integration Testing)

| Script | `integration/scripts/test_all_models.sh` |
|---|---|
| **Tipo** | Teste de integração com providers |
| **O que testa** | Cada modelo configurado responde corretamente |
| **Método** | Requisições HTTP para `/v1/chat/completions` |
| **Auth** | Usa `NEWAPI_ADMIN_TOKEN` |

**Padrão esperado:**
```bash
# Para cada modelo no catálogo:
curl -X POST https://router.atius.com.br/v1/chat/completions \
  -H "Authorization: Bearer $NEWAPI_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model": "<modelo>", "messages": [{"role": "user", "content": "ok"}], "max_tokens": 8}'

# Esperado: HTTP 200 com resposta válida
```

### 2. Verificação do Stack

| Script | `integration/scripts/verify_stack.sh` |
|---|---|
| **Tipo** | Health check do stack completo |
| **O que testa** | Containers rodando, DB acessível, API respondendo |
| **Método** | Docker ps + curl para healthcheck |

### 3. Testes Manuais via cURL (documentados no README)

#### Status da Aplicação

```bash
curl -sS https://router.atius.com.br/api/status
# Esperado: 200 com status OK
```

#### Health Check

```bash
curl -sS https://router.atius.com.br/health
# Esperado: 200 com health OK
```

#### Listagem de Modelos

```bash
curl -sS -H "Authorization: Bearer $NEWAPI_ADMIN_TOKEN" \
  https://router.atius.com.br/v1/models
# Esperado: 200 com lista de modelos
```

#### Chat Completion

```bash
curl -sS \
  -H "Authorization: Bearer $NEWAPI_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  https://router.atius.com.br/v1/chat/completions \
  -d '{"model": "qwen3-max", "messages": [{"role": "user", "content": "Responda apenas: ok"}], "max_tokens": 8}'
# Esperado: 200 com resposta do modelo
```

#### Validação de Token Open-WebUI

```bash
curl -sS -o /tmp/models.json -w "%{http_code}\n" \
  -H "Authorization: Bearer $OPENWEBUI_LITELLM_KEY" \
  http://localhost:3300/v1/models
# Esperado: 200 com lista não-vazia
```

## Códigos de Resposta Esperados

| Código | Significado |
|---|---|
| `200` | Sucesso |
| `401` | Token ausente/inválido/sem permissão |
| `429` | Limite de taxa atingido no provider upstream |
| `404` | Endpoint não disponível nesta versão |
| `5xx` | Erro interno ou indisponibilidade de dependência |

## Monitoramento de Saúde

### Disk Health Check

| Script | `disk-health.sh` |
|---|---|
| **Tipo** | Monitoramento de infraestrutura |
| **Threshold** | 95% de uso de disco |
| **Ação** | Alerta crítico + sugestão de limpeza |
| **Limpeza** | `--cleanup-safe` remove cache e logs antigos |

**Verifica:**
- Uso de disco do host (`df -P /`)
- Cache npm (`~/.npm/_cacache/`)
- Cache Playwright (`~/.cache/ms-playwright/`)
- Cache pip (`~/.cache/pip/`)
- Logs antigos do NewAPI (mantém 5 mais recentes)

### PostgreSQL Healthcheck

| Config | Valor |
|---|---|
| **Test** | `pg_isready -U admin -d newapi` |
| **Interval** | 10s |
| **Timeout** | 5s |
| **Retries** | 10 |

## Testes de Sync de Channels

| Script | Função |
|---|---|
| `sync_deepseak_channels.py` | Verifica se channels DeepSeek estão sincronizados com `.env` |
| `sync_openrouter_channels.py` | Verifica channels OpenRouter |
| `sync_iflow_channel_keys.py` | Verifica chaves de canais iFlow |

**Estes scripts funcionam como testes de consistência** — garantem que a configuração local (`.env`) esteja refletida no banco de dados do NewAPI.

## Troubleshooting como Teste

O README documenta fluxos de troubleshooting que funcionam como diagnósticos:

### Problema: Open-WebUI sem modelos

1. Validar token contra endpoint `/v1/models`
2. Se `401` → Alinhar tokens
3. Se `200` e lista vazia → Normalizar catálogo
4. Se `5xx` → Verificar logs e DB

### Problema: `system_disk_overloaded`

1. `./disk-health.sh` para verificar uso
2. `./disk-health.sh --cleanup-safe` para limpar
3. `./reload-newapi.sh` para recriar container

## O que NÃO Existe

| Tipo de Teste | Status |
|---|---|
| **Testes unitários** | ❌ Não aplicável (app é imagem Docker) |
| **Testes E2E automatizados** | ❌ Não implementados |
| **CI/CD pipeline** | ❌ Não configurado |
| **Testes de regressão** | ❌ Não automatizados |
| **Testes de performance** | ❌ Não implementados |
| **Testes de segurança** | ❌ Não automatizados |

## Recomendações Futuras

| Melhoria | Prioridade |
|---|---|
| Automatizar `test_all_models.sh` como cron job | Alta |
| Adicionar healthcheck para NewAPI (não só PostgreSQL) | Alta |
| Criar script de validação pós-deploy | Média |
| Adicionar testes de carga para rate limiting | Baixa |
| Implementar monitoramento com alertas | Baixa |
