---
phase: 29-k3s-shadow-restore-and-go-no-go
plan: 04
subsystem: infra
tags: [k3s, go-no-go, rollback, sha256, clusterip, podman, apache]
requires:
  - phase: 29-01
    provides: cleanup and bootstrap evidence contracts
  - phase: 29-02
    provides: PostgreSQL 17 backup and restore evidence contracts
  - phase: 29-03
    provides: shadow apply, CLIAnything k3s and strict smoke evidence contracts
provides:
  - fail-closed Phase 29 evidence aggregator and sanitized decision schema
  - read-only Podman and Apache rollback proof
  - executable Phase 29 runbook without cutover instructions
affects: [phase-30, k3s-cutover, podman-rollback]
tech-stack:
  added: []
  patterns: [atomic sanitized JSON evidence, checksum-bound evidence chain, run-bound fresh rollback, exact live identity map, default no-go]
key-files:
  created:
    - scripts/k3s-router-go-no-go.sh
    - tests/phase29-k3s-router-go-no-go-selftest.sh
  modified:
    - scripts/k3s-router-rollback-check.sh
    - docs/K3S-MIGRATION.md
key-decisions:
  - "NO-GO is a valid formal result and returns success after a valid decision artifact is written."
  - "GO requires one fresh cluster-bound checksum chain plus current read-only cluster, Podman and Apache readbacks."
  - "Every decision generates a unique rollback artifact in the same run; external decision verification is rejected."
  - "Mutable live image references, including any unresolved Wave 3 tag, block GO instead of being repaired by Plan 29-04."
patterns-established:
  - "Evidence chain: restore binds backup, apply binds restore/bootstrap, smoke binds apply/restore/image, decision checksums every artifact."
  - "Phase 29 final gates never contain Apache, Podman, manifest or k3s mutation commands."
requirements-completed: []
coverage:
  - id: D1
    description: Fail-closed GO/NO-GO aggregator with freshness, cluster UID and checksum-chain validation
    requirement: PHASE-22-K3S-PREFLIGHT
    verification:
      - kind: unit
        ref: tests/phase29-k3s-router-go-no-go-selftest.sh
        status: pass
    human_judgment: false
  - id: D2
    description: Read-only Podman and Apache rollback proof consumable by the decision gate
    requirement: PHASE-22-RUNTIME-PARITY
    verification:
      - kind: unit
        ref: scripts/k3s-router-rollback-check.sh --self-test
        status: pass
      - kind: integration
        ref: scripts/k3s-router-rollback-check.sh --live
        status: unknown
    human_judgment: true
    rationale: Live execution was explicitly prohibited for this implementation pass.
  - id: D3
    description: Phase 29 runbook documents evidence order, Vault names, CPU caps and Phase 30 authorization boundary
    verification:
      - kind: other
        ref: shellcheck, bash -n and git diff --check
        status: pass
    human_judgment: false
duration: 25min
completed: 2026-07-13
status: implementation_complete_pending_live
---

# Phase 29 Plan 04: GO/NO-GO e rollback Summary

**Gate fail-closed com cadeia SHA-256, freshness e cluster UID, acompanhado de prova read-only de Podman/Apache e contrato ClusterIP para a Phase 30**

## Resultado

A implementação do Plan 29-04 está concluída e validada somente com fixtures
leves. Nenhum comando live, deploy, build ou commit foi executado. Portanto não
existe `decision.json` operacional desta execução, os requirements não foram
marcados como completos e a Phase 30 continua bloqueada.

## Implementado

- `scripts/k3s-router-go-no-go.sh` começa em `no-go`, gera rollback fresh com
  `run_id`, valida
  `cleanup/bootstrap/backup/restore/shadow-apply/smoke/rollback`, aplica janelas
  de freshness, cluster UID e SHA-256 cruzados e só autoriza a Phase 30 com
  todos os gates verdes no mesmo run. `--verify-existing` falha fechado.
- O readback live implementado verifica estabilidade de cinco minutos,
  >=32 GiB e >=25% livres, label/Secret keys, Pods `500m` em `atius-srv-1`,
  imagens imutáveis, conjunto exato de controllers/ReplicaSets, cadeia de owner
  UIDs, igualdade semântica de workloads/images/storage com o snapshot smoke,
  EndpointSlices por pod UID/IP e PVC/PV exatos ligados a apply/smoke/manifests.
- `scripts/k3s-router-rollback-check.sh` trata a user unit como opcional, exige
  pod/containers/endpoint, limites, `status --backend podman` e vhost Apache
  fixo com `configtest` e `apache2ctl -S`; a seleção `router.atius.com.br:443`,
  o bloco efetivo e as rotas únicas são registrados independentemente.
- Wave 2 usa `pg_dump_version` normalizada `17.x`, reset de quota por
  `systemctl set-property --runtime ... CPUQuota=` sem apagar drop-ins e
  inventário PostgreSQL v2 com igualdade source/backup/target. Um
  `50-CPUQuota.conf` runtime preexistente é rejeitado antes de qualquer write.
- `docs/K3S-MIGRATION.md` agora cobre apenas a ordem 29-01 a 29-04, nomes/path
  de Vault sem valores, CPU <=20%, evidências e interpretação de GO/NO-GO.
  Procedimentos de cutover e aposentadoria Podman foram removidos para a Phase
  30.

## Verificação executada

- `bash tests/phase29-k3s-router-restore-selftest.sh` — PASS.
- `bash tests/phase29-k3s-router-go-no-go-selftest.sh` — PASS.
- `bash -n` nos três scripts e dois selftests do escopo — PASS.
- `shellcheck -x` nos três scripts e dois selftests do escopo — PASS.
- `git diff --check` nos sete arquivos autorizados — PASS; os três arquivos
  untracked também não produziram diagnóstico de whitespace no no-index check.
- Fixtures cobrem conjunto integral verde, gate vermelho, checksum adulterado,
  evidência stale, artefato ausente, quota runtime preexistente, Apache
  retargetado, vhost falso, route duplicado/ausente, controller/PVC/EndpointSlice
  extra, owner UID quebrado, divergência smoke workload/image/storage e
  substituição de pod no mapa live.

## Commits

Nenhum. O usuário proibiu commits nesta execução.

## Desvios do plano

### Restrição explícita do usuário

Os verifies live do PLAN não foram executados. Não houve acesso live ao cluster,
Podman ou Apache, nem atualização de STATE/ROADMAP/requirements. A implementação
foi validada exclusivamente por selftests leves e análise estática.

### Dependência Wave 3 preservada

O plano não alterou Wave 3, OAuth ou manifests. O gate 29-04 apenas consome os
contratos de `shadow-apply.json` e `smoke.json` e bloqueia GO diante de imagem,
identidade ou cadeia divergente.

## Decisão operacional

**Não emitida.** Ausência de execução live significa `phase30_authorized=false`
por contrato. O próximo passo autorizado é corrigir/verificar a Wave 3 em seu
escopo proprietário e, somente com autorização live separada, executar os
comando 29-04 documentado para produzir `rollback-<run_id>.json`,
`live-identity-<run_id>.json` e `decision.json`.

## Known Stubs

Nenhum. Valores vazios nos scripts são defaults internos de argumentos CLI e
não fluem para UI nem simulam evidência operacional.

## Self-Check: PASSED

- Todos os sete arquivos do escopo existem.
- Selftests e verificações estáticas passaram.
- Nenhuma claim de decisão live ou commit foi feita.

---
*Phase: 29-k3s-shadow-restore-and-go-no-go*
*Implementation completed: 2026-07-13; live decision pending explicit authorization*
