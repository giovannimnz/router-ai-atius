---
phase: 30-k3s-public-cutover-and-rollback-soak
verified: 2026-07-19T20:21:06Z
status: passed
score: 8/8 must-haves verified
decision: stay-on-k3s
---

# Phase 30 Verification Report

**Goal:** mover o edge publico para k3s somente apos GO e manter rollback verificavel.
**Status:** passed

| # | Criterio | Resultado | Evidencia |
|---|---|---|---|
| 1 | Gate da Phase 29 | PASS | `29-VERIFICATION.md` registra 7/7 e decisao GO. |
| 2 | Apache valido | PASS | `apachectl configtest` retornou `Syntax OK`; app/API apontam para `10.43.102.221:3000`. |
| 3 | Health publico | PASS | Health publico retornou HTTP 200. |
| 4 | Catalogo publico | PASS | Root somente `data`, ordenacao/filtros preservados, sem metadata interna. |
| 5 | Metadata Codex | PASS | Sol/Terra/Luna expõem `context_length=272000`, `max_completion_tokens=128000` e pricing USD/1M correto. |
| 6 | Relay real | PASS | Embedding 768 e `/v1/responses` non-stream HTTP 200/`completed`. |
| 7 | Soak inicial | PASS | Mais de 15 minutos Ready antes do ultimo rollout; apos config final, novo pod Ready com zero restart. |
| 8 | Rollback preservado | PASS | Unit Podman `inactive`, nao removida; backups de DB e Apache preservados. |

## Decisao Final

**STAY ON K3S.** Nao houve criterio de rollback no soak inicial. A observacao estendida continua operacional, sem manter a fase artificialmente aberta.

## Risco Residual Aceito

O deployment e single-node e usa `local-path`; portanto nao oferece HA. Esse risco foi aceito explicitamente para manter o Router fixo em `atius-srv-1`, com backup/restore e rollback Podman como controles compensatorios.
