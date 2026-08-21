---
phase: 24
phase_name: router-db-catalog-recovery-and-canonical-host-db
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 3, patterns: 3, surprises: 2 }
missing_artifacts: ["24-VERIFICATION.md"]
---

# Phase 24 Learnings: router DB and catalog recovery

## Decisions

### Banco live preserva dados operacionais; snapshots restauram catalogo seletivo
Users, tokens e logs continuam do banco canônico, enquanto channels/models/abilities são reconciliados com transformações explícitas.

**Source:** 24-01-SUMMARY.md, 24-02-SUMMARY.md

### Mutacao de banco exige confirmacao dupla e rollback holdback
O builder e dry-run por padrão, exige source/target confirmados e mantém o banco anterior como holdback até a validação final.

**Source:** 24-02-SUMMARY.md

### Contrato final remove aliases 1M e preserva o governor Go
Codex base permanece, aliases `-1m` saem, DeepSeek fica consolidado, MiniMax restaurado/desabilitado e `embedding-gte-v1` continua governado.

**Source:** 24-03-SUMMARY.md, 24-04-SUMMARY.md

## Lessons

### Catalogo e dados de usuario têm estratégias de recuperação diferentes
Restaurar dump inteiro arriscaria apagar estado recente; recovery precisa separar domínios e transformar apenas o catalogo.

**Source:** 24-01-SUMMARY.md, 24-02-SUMMARY.md

### Segredo deve entrar apenas no momento da execução
O SQL mantém placeholder fail-closed e exige injeção por fonte segura, nunca por arquivo versionado.

**Source:** 24-02-SUMMARY.md

### Docs históricas precisam marcar claramente o contrato atual
Material `-1m` foi preservado para forensics, mas separado das regras operacionais vigentes para não reativar comportamento inválido.

**Source:** 24-03-SUMMARY.md

## Patterns

### Read-only inventory antes de qualquer recovery
Combinar Graphify, CLIAnything, contagens, unit DSN, inventário de dumps e `pg_restore -l` antes de decidir a fonte.

**Source:** 24-01-SUMMARY.md

### Candidate database antes de cutover
Construir, transformar, consultar e só depois apontar runtime/PgBouncer, mantendo rollback explícito.

**Source:** 24-02-SUMMARY.md

### Estado de providers deve ser validado em channels, abilities e models
Uma camada isolada pode parecer correta enquanto rows legadas ou abilities inconsistentes permanecem.

**Source:** 24-03-SUMMARY.md, 24-04-SUMMARY.md

## Surprises

### O problema parecia perda de dados, mas era drift de target/catalogo
As evidências não confirmaram delete; o risco principal era o runtime apontar para banco/nome incorreto e carregar catalogo divergente.

**Source:** 24-01-SUMMARY.md

### Graphify podia estar recente por mtime e atrasado por commit
O fechamento precisou usar reads e testes focados quando o rebuild não atualizou o commit de origem como esperado.

**Source:** 24-03-SUMMARY.md
