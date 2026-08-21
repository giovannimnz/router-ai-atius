---
phase: 27
phase_name: codex-official-docs-ci-and-release-alignment
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 2, patterns: 2, surprises: 2 }
missing_artifacts: ["27-VERIFICATION.md", "27-UAT.md"]
---

# Phase 27 Learnings: Codex official docs and CI

## Decisions

### OpenAI Docs e a fonte primária para CI/Codex
O runbook local registra comandos e inputs oficiais em vez de depender de exemplos antigos do fork.

**Source:** 27-01-SUMMARY.md

### API key e o padrão de automação
Auth gerenciada por ChatGPT fica restrita a runner privado e cenário avançado; CI normal usa segredo dedicado.

**Source:** 27-01-SUMMARY.md

### GitHub Action usa inputs oficiais
`openai/codex-action@v1` passou a receber `effort` diretamente, removendo shim interno em argumentos.

**Source:** 27-01-SUMMARY.md

## Lessons

### Planejamento ausente precisa ser materializado antes da execução
O init da fase retornou zero planos até CONTEXT, RESEARCH e PLAN serem criados de forma canônica.

**Source:** 27-01-SUMMARY.md

### Docs de auth precisam separar CI de operador interativo
Misturar `auth.json`, API key e login ChatGPT sem contexto cria risco de copiar credenciais ou quebrar renovação independente.

**Source:** 27-01-SUMMARY.md

## Patterns

### Docs oficiais -> runbook PT-BR -> workflow validado
Confirmar a API oficial, traduzir a decisão operacional e então validar YAML e strings obrigatórias.

**Source:** 27-01-SUMMARY.md

### Auth independente por consumidor
CLI, Router e CI devem ter ciclos de credencial separados para evitar rotação cruzada.

**Source:** 27-01-SUMMARY.md, 32-CONTEXT.md

## Surprises

### O input correto existia, mas o workflow usava workaround
A Action já expunha `effort`; o fork ainda configurava reasoning por shim em `codex-args`.

**Source:** 27-01-SUMMARY.md

### O hotfix posterior provou o custo de credenciais acopladas
Copiar apenas access token do CLI salvou o channel 5, mas produziu uma janela temporária sem refresh próprio.

**Source:** 32-CONTEXT.md
