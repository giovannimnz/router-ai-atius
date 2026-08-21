---
phase: 31
phase_name: planning-health-normalization-and-legacy-archive
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 2, lessons: 2, patterns: 2, surprises: 1 }
missing_artifacts: ["31-PLAN.md", "31-SUMMARY.md", "31-VERIFICATION.md", "31-UAT.md"]
---

# Phase 31 Learnings: planning health normalization

## Decisions

### Arquivar e normalizar antes de deletar
O objetivo foi reduzir warnings sem perder contexto histórico de milestones e fases legacy.

**Source:** 31-CONTEXT.md

### Pendência operacional real deve virar milestone futuro
Shadow restore e cutover k3s permaneceram nas Phases 29/30 de v2.16, enquanto resíduos antigos e trabalho já resolvido foram fechados ou arquivados.

**Source:** ROADMAP.md, STATE.md

## Lessons

### Status textual e artefatos precisam concordar
Uma fase marcada `Complete` sem PLAN/SUMMARY/VERIFICATION continua deixando dívida de auditabilidade, mesmo quando a organização do roadmap melhorou.

**Source:** 31-CONTEXT.md, ROADMAP.md

### Parser de dependências não deve receber datas em texto livre
O manager interpretou `2026-07-10` como dependências `2026`, `07`, `10`; dependências precisam citar apenas phases de forma estruturada.

**Source:** init.manager output de 2026-07-10

## Patterns

### Health warnings como backlog tipado
Tratar cada código `W002`, `W005`, `W006`, `W019` e `I001` com decisão explícita de corrigir, arquivar ou justificar.

**Source:** 31-CONTEXT.md

### Milestone separa execução operacional de higiene documental
Não misturar cutover k3s manual com fechamento de branches, docs ou artefatos GSD.

**Source:** ROADMAP.md

## Surprises

### O roadmap ficou coerente antes de a evidência da própria Phase 31 ficar completa
A fase foi marcada complete com zero planos e sem summary, deixando um gap documental que este LEARNINGS registra explicitamente.

**Source:** ROADMAP.md, init.phase-op 31
