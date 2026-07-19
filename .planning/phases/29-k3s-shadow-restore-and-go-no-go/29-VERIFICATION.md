---
phase: 29-k3s-shadow-restore-and-go-no-go
verified: 2026-07-19T20:21:06Z
status: passed
score: 7/7 must-haves verified
decision: go
---

# Phase 29 Verification Report

**Goal:** restaurar e validar o stack k3s sem alterar o edge antes de uma decisao formal.
**Status:** passed

| # | Criterio | Resultado | Evidencia |
|---|---|---|---|
| 1 | Secrets fora do git | PASS | Recursos aplicados ao namespace; nenhum valor aparece no diff ou nos relatórios. |
| 2 | Restore real | PASS | Dump final restaurado com `ON_ERROR_STOP`; contagens criticas origem/destino coincidiram. |
| 3 | Stateful data | PASS | PostgreSQL 17 e PVC `20Gi` Bound; app PVC `10Gi` Bound; PVs com `Retain`. |
| 4 | Shadow Ready | PASS | Router, PostgreSQL e Redis ficaram `Ready` em `atius-srv-1`. |
| 5 | CPU governada | PASS | Requests e limits de app, DB e Redis totalizam `500m` por pod. |
| 6 | Smoke funcional | PASS | Health, auth negativa/positiva, catálogo, embedding 768 e Codex passaram. |
| 7 | Cluster apto | PASS | `DiskPressure=False`, pods sem eviction e app sem restart no gate final. |

## Decisao

**GO.** Todos os bloqueios conhecidos foram reavaliados. A ausencia de HA e de IngressClass nao bloqueia este alvo, pois o edge continua no Apache local e o workload foi explicitamente fixado no mesmo host. Podman permaneceu intacto para rollback.

## Segurança

Nenhum token, senha, DSN completo ou Secret foi registrado neste relatório.
