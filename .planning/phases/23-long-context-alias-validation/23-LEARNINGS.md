---
phase: 23
phase_name: long-context-alias-validation
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 3, patterns: 2, surprises: 2 }
missing_artifacts: ["23-VERIFICATION.md"]
---

# Phase 23 Learnings: long-context alias validation

## Decisions

### Testes caros de 1M permanecem opt-in
O harness exige `ENABLE_1M=YES_I_ACCEPT_COSTS` e mantém evidência JSONL para impedir gasto acidental e permitir auditoria.

**Source:** 23-01-SUMMARY.md

### Aliases 1M nao pertencem ao catalogo final
O trabalho provou o limite upstream, e a Phase 24 removeu os aliases públicos `-1m`, preservando os logs como evidência histórica.

**Source:** 23-UAT.md, 24-03-SUMMARY.md

### Chat Completions e o surface primario do harness
O teste mantém o contrato do cliente real e verifica streaming, billing, traceability e propagação estruturada de erro.

**Source:** 23-01-SUMMARY.md, 23-UAT.md

## Lessons

### Capacidade anunciada em docs publicas nao garante entitlement OAuth Codex
O router aceitou o caminho local, mas o upstream do channel 5 rejeitou contextos antes de 1M.

**Source:** 23-UAT.md

### Base model e alias precisam de guards diferentes
Os modelos base mantêm limite local conservador; aliases experimentais podem avançar progressivamente e registrar onde o upstream rejeita.

**Source:** 23-UAT.md

### Resultado parcial pode ser tecnicamente completo
Harness, segurança, billing e erros passaram; a aceitação 1M ficou bloqueada por upstream, não por ausência de implementação local.

**Source:** 23-01-SUMMARY.md, 23-UAT.md

## Patterns

### Progressive context ladder
Executar small smoke, streaming, degraus crescentes e parar no primeiro reject estruturado, sempre gravando usage e origem do erro.

**Source:** 23-UAT.md

### Historical evidence sem public exposure
Preservar logs e docs forenses, mas remover aliases não suportados de código, pricing e `/v1/models`.

**Source:** 24-03-SUMMARY.md

## Surprises

### GPT-5.4 chegou perto, mas nao atingiu 1M
O alias passou perto de 900k nominal e foi rejeitado em 950k/1M pelo upstream real.

**Source:** 23-UAT.md

### O blocker era o canal OAuth, nao a API pública descrita
Documentação de janela pública não forneceu flag/header equivalente para o backend ChatGPT/Codex usado pelo channel 5.

**Source:** 23-UAT.md
