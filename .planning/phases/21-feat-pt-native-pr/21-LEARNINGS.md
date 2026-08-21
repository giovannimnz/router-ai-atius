---
phase: 21
phase_name: feat-pt-native-pr
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 3, patterns: 2, surprises: 2 }
missing_artifacts: ["21-VERIFICATION.md"]
---

# Phase 21 Learnings: feat-pt-native-pr

## Decisions

### O handoff upstream precisa de uma lane baseada em upstream/main
A implementacao PT-BR foi isolada em worktree dedicado e atualizada por fast-forward antes de reaplicar somente os arquivos de idioma nativo. Isso separa o PR upstream das customizacoes do fork Atius.

**Source:** 21-01-SUMMARY.md, 21-05-SUMMARY.md

### Reuso de traducoes deve ser deterministico
O preenchimento foi feito por ordem de fonte, reutilizando traducoes existentes quando a chave e o texto ingles coincidiam e traduzindo apenas o delta restante.

**Source:** 21-02-SUMMARY.md, 21-03-SUMMARY.md, 21-04-SUMMARY.md

### A branch PT especial deve existir apenas no remoto
A Phase 28 consolidou o handoff em `origin/feat/phase21-pt-native-upstream` e removeu worktrees e branches locais concorrentes.

**Source:** 28-02-SUMMARY.md, 28-04-SUMMARY.md

## Lessons

### Paridade de chaves e placeholders e um gate melhor que contagem bruta
Zero missing, extras e untranslated, mais validacao de placeholders e normalizacao `pt`, `pt-BR` e `pt_BR`, protege o comportamento real dos dois frontends.

**Source:** 21-03-SUMMARY.md, 21-04-SUMMARY.md

### Lint global nao deve contaminar um PR estreito
Os linters dos frontends encontraram divida preexistente fora do diff PT. O handoff manteve o escopo e validou build, typecheck, paridade e leak checks focados.

**Source:** 21-03-SUMMARY.md, 21-04-SUMMARY.md

### Summary verde nao substitui UAT formal
O UAT permaneceu em `testing` para parte dos itens. O handoff tecnico ficou pronto, mas os artefatos devem distinguir implementacao concluida de validacao humana completa.

**Source:** 21-UAT.md

## Patterns

### Clean-lane upstream handoff
Partir do HEAD atual de `upstream/main`, aplicar apenas o delta permitido, executar leak grep e conferir `git diff upstream/main...HEAD` antes do push.

**Source:** 21-01-SUMMARY.md, 21-05-SUMMARY.md

### Branch especial remota, zero branches locais antigas
Preservar a entrega upstream por uma unica branch remota e manter `main` como fonte de verdade local do fork.

**Source:** 28-04-SUMMARY.md

## Surprises

### O baseline upstream mudou durante a preparacao
O worktree precisou avançar para um novo `upstream/main`, acrescentando tres chaves backend ao delta de traducao.

**Source:** 21-01-SUMMARY.md, 21-02-SUMMARY.md

### O classic exigiu muito mais traducao nova que o default
O reaproveitamento cross-frontend ajudou, mas o classic ainda exigiu milhares de valores novos e expôs divida de lint independente do PT-BR.

**Source:** 21-04-SUMMARY.md
