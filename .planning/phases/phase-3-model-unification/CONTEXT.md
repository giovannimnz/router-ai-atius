# Phase 3 Context — Model Unification via Middleware

**Status:** Ready for planning
**Gathered:** 2026-05-12

## Phase Boundary

Unificar listagem de modelos OpenAI e Anthropic em `GET /v1/models` via middleware FastAPI (model_detailed), com `api_format`区分OpenAI vs Anthropic. Deprecar `GET /v1/claude/models`.

## Decisões de Implementação

### Decisão 1: Arquitetura — Opção A (Middleware como entry point único)
- FastAPI (`model_detailed_fastapi.py`) vira o ponto de entrada para listagem de modelos
- Go (`new-api`) vira pure relay downstream
- FastAPI chama Go via HTTP interno (`http://new-api:3000`) apenas para buscar abilities
- FastAPI enriquece resposta com pricing/context_length
- Middleware retorna formato correto baseado em header ou query param

### Decisão 2: Detecção de Formato
- Header `Accept: application/json` (padrão) → OpenAI format
- Header `Accept: application/x-www-form-urlencoded` OU query param `?format=anthropic` OU header `anthropic-version` → Anthropic format
- Alternativa mais simples: query param `?api_format=openai|anthropic` (explícito, fácil de testar)

### Decisão 3: Estrutura de Resposta Unificada
```json
{
  "data": [
    {
      "id": "MiniMax-M2.7",
      "object": "model",
      "api_format": "openai",
      "created": 1626777600,
      "owned_by": "atius",
      "context_length": 1000000,
      "input_price": 0.3,
      "output_price": 1.2
    },
    {
      "id": "MiniMax-M2.7",
      "object": "model",
      "api_format": "anthropic",
      "created_at": "2021-07-20T10:40:00Z",
      "display_name": "MiniMax-M2.7",
      "type": "model"
    }
  ]
}
```
**Problema:** IDs duplicados na lista. Discriminar por `api_format` é suficiente.

### Decisão 4: Deprecação de /v1/claude/models
- Manter rota no Go por enquanto (não quebra clientes existentes)
- Na resposta, adicionar header `Deprecation: true` e `Sunset: <date>`
- Documentar migração: clientes devem usar `/v1/models?api_format=anthropic`

### Decisão 5: Acesso ao DB
- FastAPI NÃO acessa DB diretamente
- FastAPI chama `GET /internal/v1/models` no Go (endpoint interno, sem auth)
- Go retorna abilities com channel_type (14=Anthropic, 0=OpenAI)
- FastAPI transforma e enrich

## Canonical References

- `/home/ubuntu/docker/Atius/router-ai-atius/integration/middleware/model_detailed_fastapi.py` — FastAPI middleware atual
- `/home/ubuntu/docker/Atius/router-ai-atius/controller/model.go` — ListModels e ListClaudeModels
- `/home/ubuntu/docker/Atius/router-ai-atius/router/relay-router.go` — rotas
- `/home/ubuntu/docker/Atius/router-ai-atius/model/ability.go` — GetGroupEnabledModels
- `/home/ubuntu/docker/Atius/router-ai-atius/.planning/phases/phase-2-claude-models-endpoint/PLAN.md` — endpoint anterior

## Blocker Conhecido
- Upstream MiniMax retorna 503 intermitente (não é do router)
- Não impacta esta fase

## Out of Scope
- Acesso DB direto pelo FastAPI (mantém Go como source of truth)
- Alteração de canais ou abilities (já configurado)
- Autenticação adicional (já existe)
