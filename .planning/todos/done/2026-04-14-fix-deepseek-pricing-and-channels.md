---
created: 2026-04-14T04:27
title: Fix DeepSeek pricing and split 3 channels into 6
area: billing
files:
  - integration/middleware/model_detailed.py
  - .planning/codebase/ARCHITECTURE.md
---

## Problem

O NewAPI está contabilizando valores errados para os modelos DeepSeek porque o `ModelPrice` na tabela `options` não corresponde ao pricing real por token.

O endpoint `/v1/models` retorna pricing dividido por 1M tokens:
- prompt: "0.00000028" (cache miss: $0.28/1M)
- completion: "0.00000042" ($0.42/1M)  
- prompt_cache_hit: "0.000000028" (cache hit: $0.028/1M)

Mas o `ModelPrice` atual está como `0.28` (sem dividir por 1M), fazendo o billing ficar ~1 milhão de vezes maior que o correto.

## Solution

1. Corrigir `ModelPrice` para ratio 0.14 (não 0.28, que dobrava o billing)
   - O NewAPI usa ModelPrice como ratio direto na fórmula de quota
   - Fórmula: quota = tokens * ModelPrice * CompletionRatio
   - Display: dollars = quota / 500,000
   - Com ModelPrice=0.14: 500K*0.14 + 500K*0.14*1.5 = 175K quota = $0.35 ✓
   - Com ModelPrice=0.28 (antigo): dava $0.70 para o mesmo consumo (2x errado)

2. Recalcular `used_quota` dos canais: multiplicar por 0.5 (0.14/0.28)
   - Key 1: 51,521,463 → 25,760,732 ($51.52 display)
   - Key 2: 50,260,750 → 25,130,375 ($50.26 display)
   - Key 3: 42,840,375 → 21,420,188 ($42.84 display)

3. Recalcular `used_quota` do usuário: soma dos canais = 72,311,295 ($144.62)

4. Canais separados por modelo: ainda pendente (requer 6 canais: 3 keys × 2 modelos)

## Notes

- 3 canais atuais: Key1, Key2, Key3 (cada com ambos modelos)
- Proposal: 6 canais — Key1-chat, Key1-reasoner, Key2-chat, Key2-reasoner, Key3-chat, Key3-reasoner
- `ModelRatio` e `CompletionRatio` parecem corretos (1.5x para output)
