---
phase: 28
phase_name: branch-hygiene-and-mainline-reconciliation
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 3, patterns: 3, surprises: 2 }
missing_artifacts: ["28-VERIFICATION.md", "28-UAT.md"]
---

# Phase 28 Learnings: branch hygiene and reconciliation

## Decisions

### Backup completo precede qualquer limpeza
Cada worktree teve status, patches, untracked list/tar e HEAD preservados, além de tags de segurança.

**Source:** 28-01-SUMMARY.md, 28-04-SUMMARY.md

### Main recebe cherry-pick seletivo, nao merge bruto de branch poluida
A reconciliação incluiu 24-28 e docs do handoff 21, excluindo PT implementation e deletions herdadas.

**Source:** 28-03-SUMMARY.md

### Política final: main local unica e uma branch PT remota especial
Foram removidos worktrees e branches locais stale; no remoto permaneceram `main` e `feat/phase21-pt-native-upstream`.

**Source:** 28-04-SUMMARY.md

## Lessons

### Branch com histórico útil ainda pode ser fonte insegura de merge
O valor estava nos commits selecionados, não na topologia completa de `feat/pt-native`.

**Source:** 28-03-SUMMARY.md

### Higiene deve validar filesystem, refs locais e refs remotas
`git worktree list`, `branch -vv`, `branch -r` e `ls-remote` são gates distintos.

**Source:** 28-04-SUMMARY.md

### Branch remota canônica precisa nascer de lane limpa
O handoff PT foi reconstruído sobre upstream atual e só então pushado para a branch preservada.

**Source:** 28-02-SUMMARY.md

## Patterns

### Snapshot, promote, reconcile, validate, clean
Esta ordem evita destruição prematura e transforma cleanup em etapa reversível.

**Source:** 28-01-SUMMARY.md through 28-04-SUMMARY.md

### Stage por allowlist
Em worktree sujo, commitar apenas paths/hunks da wave evita capturar trabalho paralelo do usuário.

**Source:** 28-03-SUMMARY.md

### Worktree principal deve ser recriado da origem após reconciliação
Depois do push, o checkout local foi reduzido a um único `main` rastreando `origin/main`.

**Source:** 28-04-SUMMARY.md

## Surprises

### Havia mais worktrees e branches concorrentes que o fluxo precisava
A higiene removeu lanes de main, reconcile, sync-fix e PT que já tinham cumprido seu papel.

**Source:** 28-04-SUMMARY.md

### A branch PT preservada nao precisa existir localmente
Manter apenas a ref remota satisfaz o handoff upstream sem convidar novos trabalhos na branch aposentada.

**Source:** 28-04-SUMMARY.md
