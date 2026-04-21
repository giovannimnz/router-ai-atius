# PLAN.md — Phase 1: Investigar mecanismos de customização do NewAPI

## Investigação Concluída

### Endpoint /v1/models atual
Retorna dados **hard-coded** pelo NewAPI:
```json
{"id": "deepseek-chat", "object": "model", "created": 1626777600, "owned_by": "deepseek", "supported_endpoint_types": ["openai"]}
```
- Sem `context_length`, `pricing`, `name`, `top_provider`
- `created` sempre 1626777600 (hard-coded no código fonte)

### Banco de Dados NewAPI (PostgreSQL)

**Tabela `channels`** (routing):
- `models`: lista de modelos (ex: "deepseek-chat,deepseek-reasoner")
- `base_url`: endpoint do provider
- `key`: API key
- `setting`: JSON com config (force_format, thinking_to_content, etc)
- `channel_info`: JSON com multi-key status

**Tabela `models`** (catálogo interno):
- `model_name`, `description` (texto livre com specs), `tags`, `endpoints` (JSON), `vendor_id`
- Exemplo: `"Contexto: 128K | Max Output: 32K | Input: $0.55/1M | Output: $2.19/1M"`

**Tabela `options`** (config global):
- `ModelRatio`: `{"deepseek-chat": 1, "deepseek-reasoner": 1}`
- `CompletionRatio`: `{"deepseek-chat": 1.5, "deepseek-reasoner": 1.5}`
- `ModelPrice`: `{"deepseek-chat": 0.28, "deepseek-reasoner": 0.28}`
- Estes são para **cobrança interna** do NewAPI, não para resposta API

### Conclusão
O NewAPI **não suporta** metadata enriquecida no `/v1/models` nativamente (v0.12.7). O código fonte Go hard-coded a resposta sem campos como context_length, pricing, top_provider.

### Abordagem Definida
**Middleware Proxy Python** — intercepta GET `/v1/models`, injeta metadados enriquecidos no formato OpenAI estendido, e roteia tudo transparentemente para o NewAPI backend.

- Simples, deploy rápido
- Sem modificação do NewAPI
- Mantém compatibilidade total

## Verificação
- [x] Endpoint /v1/models analisado
- [x] DB tables channels, models, options inspecionadas
- [x] Admin API testada
- [x] Abordagem definida: middleware proxy

## Próximo Passo
→ Phase 2: Configurar metadados DeepSeek + implementar middleware
