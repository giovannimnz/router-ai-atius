---
phase: 22
phase_name: k3s-migration-preflight-and-cutover-plan-for-router-ai-atius
project: router-ai-atius
generated: "2026-07-10T11:25:00-03:00"
counts: { decisions: 3, lessons: 3, patterns: 2, surprises: 2 }
missing_artifacts: ["22-VERIFICATION.md", "22-UAT.md"]
---

# Phase 22 Learnings: k3s migration preflight and cutover

## Decisions

### Podman permanece rollback ate existir shadow evidence
O runtime publico nao foi movido. Apache continua apontando para o backend Podman enquanto backup, restore, shadow smoke e go/no-go nao estiverem comprovados.

**Source:** 22-03-SUMMARY.md, 22-04-SUMMARY.md

### O contrato k3s preserva o backend Go-only
Os manifests e runbooks mantêm `/v1/` no Go, sem `model-detailed`, namespace dedicado e secrets fora do git.

**Source:** 22-01-SUMMARY.md, 22-02-SUMMARY.md

### Cutover e uma acao manual separada
A fase produziu preflight, manifests, backup, shadow e rollback, mas deixou o apply/cutover atrás de opt-in e moveu a execução real para as Phases 29/30.

**Source:** 22-03-SUMMARY.md, 22-04-SUMMARY.md, ROADMAP.md

## Lessons

### Cluster Ready nao significa pronto para stateful cutover
Ausencia de Metrics API, storage `local-path` RWO/Delete sem expansion e ausencia de IngressClass sao blockers reais mesmo com nodes Ready.

**Source:** 22-01-SUMMARY.md, 22-04-SUMMARY.md

### Validar manifests exige sintaxe e schema server-side
Parse YAML isolado nao basta; o validator combina parse local e dry-run contra a API quando o recurso existe.

**Source:** 22-02-SUMMARY.md

### Backup sem restore rehearsal nao fecha o gate
O backup foi executado, mas o shadow apply permaneceu adiado porque faltavam secrets provisionados e evidência operacional do cluster.

**Source:** 22-03-SUMMARY.md

## Patterns

### Opt-in destrutivo explicito
Comandos mutáveis usam variavel de confirmacao, preflight read-only e checklist antes de tocar runtime publico.

**Source:** 22-03-SUMMARY.md, 22-04-SUMMARY.md

### Rollback-first migration
Manter o runtime anterior ativo e documentar o retorno antes de alterar edge ou dados stateful.

**Source:** 22-04-SUMMARY.md

## Surprises

### O principal gap era infraestrutura do cluster, nao manifest
Os manifests ficaram validos, mas storage, métricas e ingress impediram avanço seguro.

**Source:** 22-01-SUMMARY.md, 22-04-SUMMARY.md

### A fase de cutover terminou corretamente sem cutover
O resultado correto foi `deferred`, porque cumprir o gate significava recusar uma promoção sem evidência.

**Source:** 22-04-SUMMARY.md
