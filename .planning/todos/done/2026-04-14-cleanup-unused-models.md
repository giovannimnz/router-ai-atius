---
created: 2026-04-14T04:46
title: Remove unused models and clean up catalog
area: billing
files:
  - integration/middleware/model_detailed.py
---

## Problem

Catálogo tinha 7 modelos não usados (Qwen, Kimi, etc.) sem canais configurados.

## Solution

- Removidos: deepseek-r1, deepseek-v3.2, qwen/qwen3.6-plus:free, qwen3-coder-plus, qwen3-max, qwen3-vl-plus, kimi-k2-0905
- Recriados: deepseek-chat e deepseek-reasoner no catálogo (não existiam antes!)
- Billing ratios limpos: só deepseek-chat e deepseek-reasoner
- Vendors Alibaba e Moonshot mantidos (podem ser usados no futuro)

## State Final

2 modelos: deepseek-chat, deepseek-reasoner
6 canais: Key1-Chat, Key1-Reasoner, Key2-Chat, Key2-Reasoner, Key3-Chat, Key3-Reasoner
3 canais antigos desabilitados (id 28, 29, 30)
