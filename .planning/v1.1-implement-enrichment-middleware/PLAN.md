# PLAN.md — Phase 2: Implementar middleware de enriquecimento

## Objetivo
Criar middleware Python proxy que intercepta GET `/v1/models` e retorna JSON enriquecido com metadata DeepSeek.

## Formato de Saída Esperado
```json
{
  "data": [
    {
      "id": "deepseek-chat",
      "object": "model",
      "created": 1735689600,
      "owned_by": "deepseek",
      "name": "DeepSeek V3.2",
      "context_length": 131072,
      "top_provider": {
        "max_completion_tokens": 8192
      },
      "pricing": {
        "prompt": "0.00000028",
        "completion": "0.00000042",
        "prompt_cache_hit": "0.000000028"
      },
      "supported_endpoint_types": ["openai"]
    },
    {
      "id": "deepseek-reasoner",
      "object": "model",
      "created": 1735689600,
      "owned_by": "deepseek",
      "name": "DeepSeek V3.2 Reasoner",
      "context_length": 131072,
      "top_provider": {
        "max_completion_tokens": 65536
      },
      "pricing": {
        "prompt": "0.00000028",
        "completion": "0.00000042",
        "prompt_cache_hit": "0.000000028"
      },
      "supported_endpoint_types": ["openai"]
    }
  ],
  "object": "list"
}
```

## Specs DeepSeek V3.2
| Campo | deepseek-chat | deepseek-reasoner |
|-------|---------------|-------------------|
| context_length | 131072 | 131072 |
| max_completion_tokens | 8192 | 65536 |
| pricing prompt (cache miss) | $0.28/1M | $0.28/1M |
| pricing completion | $0.42/1M | $0.42/1M |
| pricing prompt (cache hit) | $0.028/1M | $0.028/1M |

## Implementação
- Python http.server como reverse proxy
- Porta: 3001 (interna ao container), exposta na 3301
- GET /v1/models → intercepta, enriquece, retorna
- Demais endpoints → proxy transparente para NewAPI (localhost:3000)
- Sem dependências externas (stdlib only)

## Verificação
- [ ] GET /v1/models retorna formato correto
- [ ] GET /health proxy transparente funciona
- [ ] POST /v1/chat/completions proxy transparente funciona
