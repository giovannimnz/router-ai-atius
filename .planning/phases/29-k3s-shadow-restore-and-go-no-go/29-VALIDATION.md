---
phase: 29
phase_slug: k3s-shadow-restore-and-go-no-go
created: 2026-07-13
status: planned
nyquist: complete
---

# Phase 29 — Estrategia de Validacao Nyquist

## Objetivo

Cada tarefa que altera comportamento operacional possui um gate automatizado que observa o resultado live correspondente. Self-tests e validacao estatica podem antecipar falhas, mas nao substituem evidencia do host/cluster. Toda operacao pesada executa por `./scripts/podman-admin.sh profile-run`; a evidencia deve conter `cpu.max` equivalente a no maximo `80000 100000` e cada container k3s deve declarar limit de CPU de no maximo `800m` (normalmente `500m`).

## Wave 0 — Instrumentacao obrigatoria

Wave 0 acontece dentro das primeiras tarefas de cada plano antes da mutacao live. Nenhum gate posterior pode ser considerado satisfeito enquanto estes comandos/modos nao existirem e seus casos negativos nao falharem.

| ID | Artefato | Instrumentacao exigida | Prova minima |
|---|---|---|---|
| W0-01 | `scripts/k3s-router-cleanup.sh` | allowlist literal, `--live`, bytes before/after, JSON sanitizado | rejeita glob/path fora da lista/prune/volumes/backups e falha abaixo dos gates de disco |
| W0-02 | `scripts/k3s-router-preflight.sh` | `--live`, estabilidade configuravel, consumo de evidencias | amostra cinco minutos e falha com DiskPressure/taint/label/digest/Secret/quota divergentes |
| W0-03 | `scripts/k3s-router-bootstrap.sh` | label pós-estabilidade, Vault→Secret por tmpfs, imagem Podman→containerd | key names exatas, temporario removido, label exclusivo e digests concordantes |
| W0-04 | `scripts/k3s-router-restore-rehearsal.sh` | restore fail-closed e enumeracao PVC UID→PV | `ON_ERROR_STOP`, target limpo e readback Retain antes do import |
| W0-05 | `bin/clianything` | backend k3s explicito | status/query read-only no pod unico Ready, sem `podman exec` |
| W0-06 | `scripts/k3s-router-smoke.sh` | strict mode sem skips | token ausente e cada contrato divergente falham |
| W0-07 | `scripts/k3s-router-go-no-go.sh` | agregacao com checksums/frescor/default no-go | somente matriz integral verde produz go |

## Requirement → Gate Map

| Requirement | Comportamento observavel | Plano/tarefa | Comando live | Evidencia |
|---|---|---|---|---|
| PHASE-22-K3S-PREFLIGHT | >=20 GiB recuperados, >=25% livre, cinco minutos sem DiskPressure/taint | 29-01 T2 | `PHASE29_EXECUTE=1 ./scripts/podman-admin.sh profile-run -- scripts/k3s-router-cleanup.sh --live ...` seguido de `k3s-router-preflight.sh --live` | `cleanup.json`, `preflight.json`, `cpu.max` |
| PHASE-22-K3S-PREFLIGHT | label dedicado aplicado depois da estabilidade e exclusivo no srv1 | 29-01 T3 | `k3s-router-bootstrap.sh --live` + `kubectl get nodes -l atius.com.br/router-ai-atius-node=true` | node/name/label sem dados sensiveis |
| PHASE-22-K3S-PREFLIGHT | imagem live arm64 imutavel importada sob quota | 29-01 T3 | `k3s-router-bootstrap.sh --live` | source image ID, archive SHA-256, manifest/containerd/runtime digest e cpu.max |
| PHASE-22-K3S-PREFLIGHT | Secret real via Vault sem vazamento | 29-01 T3 | `k3s-router-bootstrap.sh --live` | somente keys `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `SESSION_SECRET`; tmpfs removido |
| PHASE-22-STATEFUL-DATA | dump fresco da origem live correta | 29-02 T1 | `k3s-router-backup.sh --live --source-port 8745 --pgbouncer-port 6432 ...` | origem/identidade, tamanho, SHA-256, invariantes e cpu.max |
| PHASE-22-STATEFUL-DATA | restore em target limpo e todos os PVs Retain | 29-02 T2, 29-03 T2 | restore/apply live + `kubectl get pv -o json` filtrado por namespace/claim UID | restore result e readback Retain de PostgreSQL e router |
| PHASE-22-RUNTIME-PARITY | CLIAnything opera PostgreSQL k3s | 29-03 T1 | `bin/clianything status/query --backend k3s` sob profile-run | status e identidade nao sensivel do DB |
| PHASE-22-RUNTIME-PARITY | Redis/router Ready no srv1 pelo ClusterIP | 29-03 T2 | `k3s-router-apply-shadow.sh --live` | pod nodeName/readiness, Service ClusterIP, EndpointSlice, digests e quota |
| PHASE-22-RUNTIME-PARITY | health/auth/model shape/embedding 768 | 29-03 T3 | `k3s-router-smoke.sh --strict` contra `spec.clusterIP:3000` | status/assertions sem token/payload |
| Todos | decisao formal e rollback intacto | 29-04 T1-T3 | `k3s-router-go-no-go.sh --live`, rollback check e smoke Podman local | `decision.json`, rollback evidence, Apache read-only |

## Task → Verify Map

| Plano | Tarefa | Verify obrigatorio | Falha Nyquist se |
|---|---|---|---|
| 29-01 | T1 manifests | server-side dry-run sob profile-run + leitura live do Service | comprova apenas texto local ou nao prova ClusterIP |
| 29-01 | T2 cleanup | cleanup live + preflight live de 300s | nao mede bytes before/after ou nao observa condition/taint |
| 29-01 | T3 bootstrap | label/Secret/imagem live + preflight consumindo evidencia | key values aparecem, temporario resta, label nao e exclusivo ou digest/quota diverge |
| 29-02 | T1 backup | dump live 8745 e cross-check 6432 | usa backup antigo/PgBouncer como dump source ou omite checksum/quota |
| 29-02 | T2 restore | restore live + query JSON de todos os PVs | qualquer PV da namespace nao estiver Retain |
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

`go` requer todos os registros acima atuais e com checksums consistentes. Qualquer evidencia ausente, stale ou vermelha produz `no-go` com `failed_gates`; no-go completo e resultado valido da fase, mas nao autoriza a Phase 30.
