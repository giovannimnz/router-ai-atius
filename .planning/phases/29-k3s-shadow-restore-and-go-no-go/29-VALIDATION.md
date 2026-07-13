---
phase: 29
phase_slug: k3s-shadow-restore-and-go-no-go
created: 2026-07-13
status: planned
nyquist: complete
---

# Phase 29 — Estrategia de Validacao Nyquist

## Objetivo

Cada tarefa que altera comportamento operacional possui um gate automatizado que observa o resultado live correspondente. Self-tests e validacao estatica podem antecipar falhas, mas nao substituem evidencia do host/cluster. Toda operacao pesada executa por `./scripts/podman-admin.sh profile-run`; a evidencia deve conter `cpu.max` equivalente a no maximo `80000 100000` (20% deste host) e cada container k3s deve declarar request e limit de CPU exatamente `500m`.

## Wave 0 — Instrumentacao obrigatoria

Wave 0 acontece dentro das primeiras tarefas de cada plano antes da mutacao live. Nenhum gate posterior pode ser considerado satisfeito enquanto estes comandos/modos nao existirem e seus casos negativos nao falharem.

| ID | Artefato | Instrumentacao exigida | Prova minima |
|---|---|---|---|
| W0-01 | `scripts/k3s-router-cleanup.sh` | allowlist literal, `--live`, bytes before/after, JSON sanitizado | rejeita glob/path fora da lista/prune/volumes/backups e falha abaixo dos gates de disco |
| W0-02 | `scripts/k3s-router-preflight.sh` | `--live`, estabilidade atual configuravel e consumo de evidencias | nao cria artefato proprio; revalida por cinco minutos com pelo menos 32 GiB livres e promove `cleanup.json` cluster-bound a `go`, falhando com DiskPressure/taint/espaco/quota divergentes |
| W0-03 | `scripts/k3s-router-bootstrap.sh` | label pós-estabilidade, Vault→Secret por tmpfs, imagem Podman→containerd | `bootstrap.json` fresco, cluster-bound e manifest-bound; key names exatas, temporario removido, label exclusivo e digests concordantes |
| W0-04 | `scripts/k3s-router-restore-rehearsal.sh` | restore PostgreSQL 17 atomico, fail-closed e enumeracao PVC UID→PV | target integralmente limpo, `ON_ERROR_STOP` + transacao unica, readback Retain antes do import e estado canonico global permitindo retry somente de `restore.json` no-go arquivado |
| W0-05 | `bin/clianything` | backend k3s explicito | status/query read-only no pod unico Ready, sem `podman exec` |
| W0-06 | `scripts/k3s-router-smoke.sh` | strict mode sem skips | token ausente e cada contrato divergente falham |
| W0-07 | `scripts/k3s-router-go-no-go.sh` | agregacao com checksums/frescor/default no-go | somente matriz integral verde produz go |

## Requirement → Gate Map

| Requirement | Comportamento observavel | Plano/tarefa | Comando live | Evidencia |
|---|---|---|---|---|
| PHASE-22-K3S-PREFLIGHT | >=20 GiB recuperados, >=25% livre e estado atual revalidado por cinco minutos sem DiskPressure/taint | 29-01 T2 | cleanup live seguido de `PHASE29_LIVE=1 PHASE29_REQUIRE_STABLE_SECONDS=300 k3s-router-preflight.sh --live --require-cleanup-evidence ...` | `cleanup.json` cluster-bound com `status=go`, `stable_seconds>=300`, bytes, espaco e `cpu_max`; o preflight nao produz JSON separado |
| PHASE-22-K3S-PREFLIGHT | label dedicado aplicado depois da estabilidade e exclusivo no srv1 | 29-01 T3 | `k3s-router-bootstrap.sh --live` + `kubectl get nodes -l atius.com.br/router-ai-atius-node=true` | `bootstrap.json` com cluster UID atual, `generated_at_epoch` dentro de uma hora e `manifest_sha256` igual aos manifests correntes |
| PHASE-22-K3S-PREFLIGHT | imagem live arm64 imutavel importada sob quota | 29-01 T3 | `k3s-router-bootstrap.sh --live` | campos reais de `bootstrap.json`: source image ID, archive SHA-256, manifest/containerd digest, `cpu_max` e `manifest_sha256` |
| PHASE-22-K3S-PREFLIGHT | Secret real via Vault sem vazamento | 29-01 T3 | `k3s-router-bootstrap.sh --live` | `bootstrap.json` registra somente nomes `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `SESSION_SECRET`; tmpfs removido |
| PHASE-22-STATEFUL-DATA | dump fresco da origem PostgreSQL 17 host | 29-02 T1 | `k3s-router-backup.sh --live --source-host 127.0.0.1 --source-port 8745 --pgbouncer-host 10.11.1.11 --pgbouncer-port 6432 ...` | `backup.json`: source `host-postgresql`, major 17, pg_dump 17, tamanho, SHA-256, invariantes, cliente <=400m, backend <=400m e agregado <=800m |
| PHASE-22-STATEFUL-DATA | PostgreSQL Podman vazio excluido como origem | 29-02 T1 | self-test e contrato de selecao da origem | nenhuma consulta/dump usa o container legado; desvio falha antes do backup |
| PHASE-22-STATEFUL-DATA | restore atomico em PostgreSQL 17 target integralmente limpo e todos os PVs Retain | 29-02 T2, 29-03 T2 | restore live com `cleanup.json` e `bootstrap.json` + query dos PVs por namespace/claim UID | `restore.json`: `clean_before_restore=true`, target major 17, backup source host-postgresql-17, restore_passed, retry_of e readback Retain |
| PHASE-22-RUNTIME-PARITY | CLIAnything opera PostgreSQL k3s | 29-03 T1 | `bin/clianything status/query --backend k3s` sob profile-run | status e identidade nao sensivel do DB |
| PHASE-22-RUNTIME-PARITY | Redis/router Ready no srv1 pelo ClusterIP | 29-03 T2 | `k3s-router-apply-shadow.sh --live` | pod nodeName/readiness, Service ClusterIP, EndpointSlice, digests e quota |
| PHASE-22-RUNTIME-PARITY | health/auth/model shape/embedding 768 | 29-03 T3 | `k3s-router-smoke.sh --strict` contra `spec.clusterIP:3000` | status/assertions sem token/payload |
| Todos | decisao formal e rollback intacto | 29-04 T1-T3 | `k3s-router-go-no-go.sh --live`, rollback check e smoke Podman local | `decision.json`, rollback evidence, Apache read-only |

## Task → Verify Map

| Plano | Tarefa | Verify obrigatorio | Falha Nyquist se |
|---|---|---|---|
| 29-01 | T1 manifests | server-side dry-run sob profile-run + leitura live do Service | comprova apenas texto local ou nao prova ClusterIP |
| 29-01 | T2 cleanup | cleanup live + preflight atual live de 300s | `cleanup.json` nao pertence ao cluster atual, nao mede bytes before/after ou a janela atual nao observa espaco/condition/taint |
| 29-01 | T3 bootstrap | label/Secret/imagem live + preflight consumindo evidencia | `bootstrap.json` esta stale, pertence a outro cluster, nao corresponde ao hash atual dos manifests, expoe values, deixa temporario, label nao e exclusivo ou digest/quota diverge |
| 29-02 | T1 backup canônico | pg_dump 17 direto do PostgreSQL 17 host 8745 e cross-check PgBouncer 10.11.1.11:6432 | usa backup antigo/PgBouncer como dump source ou unit/data directory/major 17, checksum ou quota nao sao provados em `backup.json` |
| 29-02 | T1 exclusao legada | self-test da selecao de origem | qualquer consulta ou dump usa o PostgreSQL Podman vazio |
| 29-02 | T2 restore | restore live atomico + `restore.json` + query JSON de todos os PVs | target nao e integralmente limpo/PG17, import nao e single-transaction, retry aceita algo alem de no-go explicito arquivado ou qualquer PV nao esta Retain |
| 29-03 | T1 CLIAnything | status e query read-only k3s | ainda depende de Podman ou selecao ambigua passa |
| 29-03 | T2 shadow | apply live + readback Retain de >=2 PVs | router sobe antes do restore, digest/node/ClusterIP/quota diverge |
| 29-03 | T3 smoke | strict smoke live contra ClusterIP | token/gate ausente e tratado como skip/sucesso |
| 29-04 | T1 decisao | agregador live + schema da decisao | gate ausente/stale permite go |
| 29-04 | T2 rollback | unit/endpoint/Apache live read-only | Podman ou edge nao esta testavel/intacto |
| 29-04 | T3 fechamento | revalidacao do decision + snapshot cluster | verifica apenas documentacao |

## Gates de higiene e seguranca

- Evidencias ficam fora do Git e nunca contem Secret YAML, valores, DSN, token ou response payload sensivel.
- Scripts que carregam Vault nao usam tracing; valores nao entram em argv. O env-file temporario vive em tmpfs, mode `0600`, e e removido por `trap` em sucesso, erro e sinal.
- Cleanup usa paths exatos e recusa prune, glob, volumes, PVC/PV/local-path, databases, backups e logs ativos.
- O label so e aplicado depois da evidencia de estabilidade; nenhuma toleration, remocao manual de taint ou mudanca de eviction threshold e permitida.
- `ClusterIP` e o unico transporte shadow. Apache e Podman permanecem sem mutacao na Phase 29.

## Gate final

`go` requer os artefatos reais `cleanup.json`, `bootstrap.json`, `backup.json`, `restore.json` e `decision.json`, com cluster UID, frescor e checksums aplicaveis consistentes. `cleanup.json` pode ser historico, desde que seja do cluster atual e o preflight reexecute a janela atual de cinco minutos; `bootstrap.json` deve estar fresco e corresponder aos manifests atuais. Qualquer evidencia ausente, stale quando aplicavel ou vermelha produz `no-go` com `failed_gates`; no-go completo e resultado valido da fase, mas nao autoriza a Phase 30.
