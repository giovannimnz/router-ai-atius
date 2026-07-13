---
phase: 29-k3s-shadow-restore-and-go-no-go
plan: 02
status: complete
completed: 2026-07-13
commits:
  - aabff9ef2
  - bfe3046ef
---

# Phase 29 Plan 02: Backup e restore PostgreSQL 17

## Resultado

`GO`. O banco canonico do PostgreSQL 17 host foi copiado para um PostgreSQL 17
k3s pinado no `atius-srv-1`, sem iniciar Redis ou router antes da validacao.

## Provas live

- fonte direta `127.0.0.1:8745`, PgBouncer `10.11.1.11:6432` e unit
  `postgresql@17-main.service` convergiram no mesmo PostgreSQL 17.10;
- dump plain SQL com 52.998.363 bytes e SHA-256 checksummed;
- cliente e backend limitados a 400m cada, agregado 800m, com quota host
  restaurada para `infinity` sem restart;
- target `router-ai-atius-postgres-0` Ready no `atius-srv-1`;
- 34 tabelas, 4 channels, 7 users e 11 tokens depois do restore;
- PV vinculado pelo claim UID, node affinity `atius-srv-1` e reclaim `Retain`;
- zero Deployments de Redis/router durante o Plan 02;
- `restore.json` e estado canonico terminaram `go`, com o primeiro `no-go`
  arquivado e ligado por SHA-256 ao retry explicito.

## Desvios resolvidos

1. O usuario `admin` nao podia consultar `data_directory`; a identidade
   privilegiada passou a ser lida localmente como `postgres`, enquanto dados e
   invariantes continuam validados pelo usuario real da aplicacao.
2. `inet_server_addr()::text` retornava `127.0.0.1/32`; o snapshot agora usa
   `host(inet_server_addr())` sem relaxar a identidade.
3. A restauracao de `CPUQuota=infinity` nao e aceita pelo systemd deste host; o
   script exige ausencia de override previo e usa `systemctl revert` somente no
   drop-in runtime que ele proprio criou.
4. O runner em process group descartava stdin de comandos em background. O
   clean probe e o dump agora usam redirecionamento explicito dentro do runner,
   com self-test sentinela.
5. O scheduler encontrou `Insufficient cpu`: 3.450m de 3.500m ja estavam
   reservados no srv1. O Portainer, com apenas 116 KB de dados, foi migrado com
   backup/checksum/PV Retain para o srv2, preservando sua InstanceID e liberando
   uma unidade permanente de 500m sem reduzir requests nem `system-reserved`.

## Artefatos operacionais

- evidence root: `~/.local/state/router-ai-atius/phase29/run-20260713T074404Z`;
- backup root-only do Portainer:
  `/var/backups/portainer-move-srv2-20260713T082309Z`;
- nenhum segredo foi persistido em Git, evidencias Markdown ou logs.

## Proximo plano

`29-03`: aplicar Redis/router somente com `restore.json` verde, validar o
backend k3s do CLIAnything e executar smoke autenticado pelo ClusterIP.
