# Model Ratios - Referência de Preços

## O que é ModelRatio?

O `ModelRatio` é o multiplicador usado para calcular o custo em quotas baseado nos tokens de input.

**Fórmula:** `custo = tokens × ModelRatio`

Onde 1 quota = $0.002 (dólares).

## O que é CompletionRatio?

O `CompletionRatio` é o multiplicador adicional para tokens de output (completion tokens).

**Fórmula completa:** `custo = (prompt_tokens × ModelRatio × CompletionRatio) + (completion_tokens × ModelRatio × CompletionRatio)`

Simplificando: `custo = (prompt_tokens + completion_tokens × CompletionRatio) × ModelRatio`

## Ratios Configurados

### DeepSeek

| Modelo | Input ($/1M) | ModelRatio | Output ($/1M) | CompletionRatio |
|--------|-------------|------------|---------------|----------------|
| deepseek-chat | $0.28 | 0.14 | $0.42 | 1.5 |
| deepseek-reasoner | $0.28 | 0.14 | $0.42 | 1.5 |
| deepseek-r1 | $0.28 | 0.14 | $0.42 | 1.5 |
| deepseek-v3.2 | $0.28 | 0.14 | $0.42 | 1.5 |

**Exemplo:** 100 prompt + 50 completion = 100×0.14 + 50×0.14×1.5 = 14 + 10.5 = 24.5 quotas

### MiniMax

| Modelo | Input ($/1M) | ModelRatio | Output ($/1M) | CompletionRatio |
|--------|-------------|------------|---------------|----------------|
| MiniMax-M2.7 | $0.30 | 0.15 | $1.20 | 4.0 |
| MiniMax-M2.5 | $0.30 | 0.15 | $1.20 | 4.0 |

**Exemplo:** 100 prompt + 50 completion = 100×0.15 + 50×0.15×4.0 = 15 + 30 = 45 quotas

### Kimi (Moonshot)

| Modelo | Input ($/1M) | ModelRatio | Output ($/1M) | CompletionRatio |
|--------|-------------|------------|---------------|----------------|
| kimi-k2 | ~$0.40 | 0.20 | ~$2.00 | 5.0 |
| kimi-k2-0905 | ~$0.40 | 0.20 | ~$2.00 | 5.0 |

**Exemplo:** 100 prompt + 50 completion = 100×0.20 + 50×0.20×5.0 = 20 + 50 = 70 quotas

### Qwen3

| Modelo | Input ($/1M) | ModelRatio | Output ($/1M) | CompletionRatio |
|--------|-------------|------------|---------------|----------------|
| qwen3-coder-plus | $0.325 | 0.1625 | $1.95 | 6.0 |
| qwen3-max | $0.325 | 0.1625 | $1.95 | 6.0 |
| qwen3-vl-plus | $0.325 | 0.1625 | $1.95 | 6.0 |

**Exemplo:** 100 prompt + 50 completion = 100×0.1625 + 50×0.1625×6.0 = 16.25 + 48.75 = 65 quotas

## Fórmula para Calcular ModelRatio

```
ModelRatio = (preço $/1K tokens) / 0.002
           = (preço $/1M tokens) / 2
```

Exemplo para $0.30/1M: `0.30 / 2 = 0.15`

## Fórmula para Calcular CompletionRatio

```
CompletionRatio = (preço output $/1M) / (preço input $/1M)
```

Exemplo para DeepSeek: `$0.42 / $0.28 = 1.5`

## SQL para Atualizar Banco de Dados

```sql
-- ModelRatios
UPDATE options SET value = '{
  "deepseek-chat": 0.14,
  "deepseek-reasoner": 0.14,
  "deepseek-r1": 0.14,
  "deepseek-v3.2": 0.14,
  "MiniMax-M2.7": 0.15,
  "MiniMax-M2.5": 0.15,
  "kimi-k2": 0.20,
  "kimi-k2-0905": 0.20,
  "qwen3-coder-plus": 0.1625,
  "qwen3-max": 0.1625,
  "qwen3-vl-plus": 0.1625
}' WHERE key = 'ModelRatio';

-- CompletionRatios
UPDATE options SET value = '{
  "deepseek-chat": 1.5,
  "deepseek-reasoner": 1.5,
  "deepseek-r1": 1.5,
  "deepseek-v3.2": 1.5,
  "MiniMax-M2.7": 4.0,
  "MiniMax-M2.5": 4.0,
  "kimi-k2": 5.0,
  "kimi-k2-0905": 5.0,
  "qwen3-coder-plus": 6.0,
  "qwen3-max": 6.0,
  "qwen3-vl-plus": 6.0
}' WHERE key = 'CompletionRatio';
```

## Histórico de Correções

- **22/04/2026**: Corrigido bug onde ratios estavam causando cobrança 17-28x maior que o correto.
  - Problema: DeepSeek estava usando 0.27/2 = 0.135 (deveria ser 0.14)
  - Problema: MiniMax, Kimi, Qwen3 não tinham ratios hardcoded
  - Resultado: 494.532.070 quotas creditadas aos usuários afetados
