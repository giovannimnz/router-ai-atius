---
phase: 29-k3s-shadow-restore-and-go-no-go
completed: 2026-07-19T20:21:06Z
status: complete
decision: go
---

# Phase 29 Summary

O stack sombra foi restaurado e validado no namespace `router-ai-atius`, fixado em `atius-srv-1`. PostgreSQL 17, Redis e Router ficaram `Ready`; cada pod respeita o contrato de `500m` CPU e os PVCs `local-path` usam politica de retencao `Retain`.

## Entregas

- Secrets aplicados fora do git e sem valores registrados nos artefatos.
- Restore final de `DBRouterAiAtius` a partir do DSN efetivo do runtime anterior.
- Contagens origem/destino iguais: `channels=4`, `models=11`, `abilities=9`, `tokens=13`, `users=11`, `logs=88677`.
- Backup canonico pre-cutover: `backups/k3s-router-cutover-20260719T200250Z`.
- Backup do runtime k3s validado: `backups/k3s-router-20260719T201321Z`, dump de `53737224` bytes, 35 tabelas e permissao `0600`.
- Shadow smoke autenticado passou para health, contrato `/v1/models`, embedding `embedding-gte-v1` com 768 dimensoes e Codex Responses.
- Decisao formal: **GO** para Phase 30.

## Incidentes Resolvidos

- `DiskPressure` durante o ensaio foi tratado sem manter toleration; o node voltou a `DiskPressure=False` antes do GO.
- A imagem foi importada no namespace CRI `k8s.io` e fixada na tag imutavel `v2.17.3`.
- O rollout usa `maxSurge: 0` para nao exigir um segundo pod de `500m` no host de 4 vCPU.

## Artefatos

- `docs/K3S-MIGRATION.md`
- `docs/K3S-CUTOVER-2026-07-19.md`
- `scripts/k3s-router-backup.sh`
- `scripts/k3s-router-smoke.sh`
- `k8s/router-ai-atius/`
