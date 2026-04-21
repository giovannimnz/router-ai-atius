---
created: 2026-04-14T04:30
title: Fix all model billing ratios and split DeepSeek into 6 channels
area: billing
files:
  - integration/middleware/model_detailed.py
---

## Problem

Todos os modelos estavam com billing errado. ModelPrice=0.28 era usado como ratio, causando cobrança ~2x maior.

## Solution Applied

### ModelRatio (calculated as input_price * 0.5):
- deepseek-chat/reasoner: 0.14
- deepseek-r1: 0.275
- deepseek-v3.2: 0.255
- qwen3-max: 0.39
- qwen3-vl-plus: 0.0685
- qwen3-coder-plus: 0.325
- kimi-k2-0905: 0.30

### CompletionRatio (output_price / input_price):
- deepseek-chat/reasoner: 1.5
- deepseek-r1: 3.98
- deepseek-v3.2: 4.0
- qwen3-max: 5.0
- qwen3-vl-plus: 2.99
- qwen3-coder-plus: 5.0
- kimi-k2-0905: 4.17

### Historical Quotas Recalculated:
- Channel quotas: 7.5M→3.8M each (×0.14 ratio)
- User quota: 21M total (sum of channels)

### Channel Split:
- 3 old channels (multi-model) disabled (status=0)
- 6 new channels created: Key1-Chat, Key1-Reasoner, Key2-Chat, Key2-Reasoner, Key3-Chat, Key3-Reasoner
- Quotas distributed 50/50 per model per key
