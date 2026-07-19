---
phase: 30-k3s-public-cutover-and-rollback-soak
completed: 2026-07-19T20:21:06Z
status: complete
decision: stay-on-k3s
---

# Phase 30 Summary

O Apache publico foi retargetado para o Service k3s `10.43.102.221:3000`. As rotas de documentacao continuam no servico local `127.0.0.1:3003`. O Podman foi parado depois do freeze final e permanece instalado e preservado como rollback.

## Resultado

- `https://router.atius.com.br/health`: HTTP 200.
- `/v1/models`: contrato root `data` preservado, sem campos internos; Sol/Terra/Luna em `272000/128000`.
- Pricing publico USD/1M: Sol `5/30`, Terra `2.5/15`, Luna `1/6`.
- `embedding-gte-v1`: 768 dimensoes via endpoint OCI DRG `10.21.1.21:3115`.
- `/v1/responses` non-stream: HTTP 200, `status=completed`.
- App k3s: `1/1 Ready`, zero restart, imagem `v2.17.3`/ID `7b3f62f...`.
- Node: `DiskPressure=False`; unit Podman: `inactive`.
- Decisao apos soak inicial: **permanecer no k3s**.

## Rollback

O rollback continua documentado em `docs/K3S-MIGRATION.md`: restaurar o Apache, iniciar `container-router-ai-atius.service` e repetir os smokes. Os backups do Apache e do banco foram preservados.
