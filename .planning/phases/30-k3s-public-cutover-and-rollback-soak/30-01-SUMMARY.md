---
phase: 30-k3s-public-cutover-and-rollback-soak
plan: 01
subsystem: infra
tags: [k3s, cutover, pgbouncer, apache, override, phase29]
status: live_ready_with_override
completed: 2026-07-13
---

# Phase 30 Plan 01 Summary

## Resultado

O preflight da Phase 30 foi implementado e executado live com sucesso sob
override explícito da Phase 29.

Artefato live gerado:

- `/home/ubuntu/.local/state/router-ai-atius/phase30/run-20260713T160941Z/manifest.json`

Status do envelope:

- `READY_WITH_PHASE29_OVERRIDE`

## Provas live

- Phase 29 consumida do run canônico com `shadow-apply.json` e `smoke.json`
  verdes.
- Override aceito somente para `failed_gates == ["live-stability"]`.
- Fonte host confirmada:
  - PostgreSQL 17 em `127.0.0.1:8745`
  - unit `postgresql@17-main`
  - data dir `/var/lib/postgresql/17/main`
  - `DBRouterAiAtius` com `34` tabelas
- Target k3s confirmado:
  - Service `router-ai-atius-postgres` em `10.43.179.157:5432`
  - `DBRouterAiAtius` com `35` tabelas
- PostgreSQL Podman legado confirmado como inelegível para dados:
  - container `postgres`
  - `DBRouterAiAtius` com `0` tabelas
- Backups/checksums capturados para:
  - `apache.vhost.conf`
  - `pgbouncer.ini`
  - `source-dbrouter.sql`
  - `db-topology.json`
  - `manifest.json`

## Implementado

- `scripts/k3s-router-cutover-preflight.sh`
  - self-tests de contrato e backup
  - modo live `--prepare`
  - descoberta do artefato mais recente da Phase 29 por mtime
  - gate de override explícito e limitado
  - captura sanitizada de topologia host/k3s/Podman
  - envelope com `manifest.json` + `SHA256SUMS`

## Desvios importantes descobertos live

- A topologia atual não bate mais com a hipótese antiga “host 34 / k3s 0”.
  Agora a verdade live é:
  - host `34`
  - k3s `35`
  - Podman `0`
- O contrato D-10 continua válido, mas a superfície “vazia” correta é o
  PostgreSQL Podman, não o PostgreSQL k3s já restaurado.

## Próximo blocker da Wave 2

Ao testar o cutover real de PgBouncer:

- o diff do `pgbouncer.ini` foi aplicado corretamente para
  `10.43.179.157:5432`;
- o primeiro reload falhou por permissão de leitura do arquivo;
- após corrigir owner/mode e recarregar, o backend novo passou a falhar com:
  `server login failed: password authentication failed for user "admin"`;
- o rollback de PgBouncer para `127.0.0.1:8745` foi executado e validado.

Esse blocker foi resolvido ainda em `2026-07-13`:

- docs oficiais confirmaram que PgBouncer com `auth_type = scram-sha-256`
  exige segredo SCRAM idêntico entre `auth_file` e backend;
- o role `admin` do PostgreSQL k3s estava com SCRAM diferente do host/userlist,
  porque a restauração regenerava o segredo a partir da senha em texto;
- `scripts/k3s-router-cutover.sh` passou a sincronizar o SCRAM exato do host
  antes do repoint;
- `scripts/k3s-router-restore-rehearsal.sh` passou a reaplicar o SCRAM exato do
  host no target k3s;
- após a correção, o cutover live de PgBouncer ficou verde e `127.0.0.1:6432`
  passou a responder `35` tabelas via backend k3s.

Portanto a Phase 30 segue em andamento e o próximo passo operacional é a etapa
Apache, não mais a correção de autenticação do PgBouncer.
