# Phase 29: k3s shadow restore and go/no-go - Research

**Researched:** 2026-07-12
**Domain:** Migração stateful Podman → k3s, restore rehearsal, shadow traffic e gates operacionais
**Confidence:** HIGH

## Checker Resolution Addendum — 2026-07-13

As decisoes abaixo substituem as recomendacoes anteriores conflitantes deste documento:

- origem live para backup: PostgreSQL 17 nativo do host em `127.0.0.1:8745`, cluster `/var/lib/postgresql/17/main`, administrado pela unit `postgresql@17-main`; o PgBouncer em `10.11.1.11:6432` e usado para cruzar identidade/invariantes, nao como origem do dump;
- runtime legado: o PostgreSQL Podman contem `DBRouterAiAtius` vazio e nunca e fonte de backup ou restore;
- backup e restore: `pg_dump` 17 direto do host, target PostgreSQL 17 k3s integralmente limpo, restore em transacao unica e retry apenas apos `no-go` explicito arquivado;
- evidencia: `cleanup.json` e historico e cluster-bound, enquanto o preflight sempre revalida cinco minutos do estado atual; `bootstrap.json` deve estar fresco, cluster-bound e vinculado ao hash atual dos manifests;
- transporte shadow: `ClusterIP`, alcancavel diretamente pelo Apache/host conforme auditoria live; nenhuma reserva de NodePort ou regra de firewall adicional faz parte da Phase 29;
- estabilidade de DiskPressure: `DiskPressure=False` e taint ausente por cinco minutos continuos, alem de pelo menos 20 GiB recuperados e alvo de 25% livre.

Qualquer referência a NodePort neste documento descreve apenas uma alternativa rejeitada; o contrato executável da Phase 29 usa exclusivamente ClusterIP.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- migrate production from Podman to k3s on the same machine and require all router stack pods to stay on atius-srv-1;
- execute Phase 29 then Phase 30 end-to-end.
- create/apply real Kubernetes Secrets outside git
- run restore rehearsal against the k3s target
- apply the `router-ai-atius` shadow stack
- run shadow smoke
- record go/no-go
- Podman remains the production source of truth.
- Public edge still points to Podman-backed Apache.
- shadow deployment is up and passes smoke with evidence, producing a real go/no-go for public cutover; or the run records an explicit no-go with concrete blockers and rollback state, without touching public traffic.

### the agent's Discretion

Não há seção explícita de discretion no `29-CONTEXT.md`.

### Deferred Ideas (OUT OF SCOPE)

Não há seção explícita de deferred ideas no `29-CONTEXT.md`.
</user_constraints>

## Summary

O estado live é **NO-GO para aplicar o shadow agora**: `atius-srv-1` está `Ready`, porém `DiskPressure=True`, com taint `node.kubernetes.io/disk-pressure:NoSchedule`; há 27–28 GB livres, uso de filesystem em 87%, image GC acima do threshold de 85%, evictions recentes e o eviction manager continua tentando recuperar `ephemeral-storage`. As métricas agora funcionam, mas mostram 47% de CPU e 70% de memória no nó. [VERIFIED: `sudo k3s kubectl get/describe/top`, kubelet stats e journal read-only em 2026-07-12]

Os manifests e scripts da Phase 22 não estão implementation-ready para a decisão locked: nenhum dos três workloads tem `nodeSelector`/required affinity para `atius-srv-1`; não há toleration, mas **não se deve adicionar toleration de DiskPressure como “correção”**; o apply sobe router junto com um Postgres vazio antes do restore; o smoke aceita sucesso parcial sem token; o Service é `ClusterIP`, sem endpoint host estável para Apache; e os PVCs herdam `local-path` com reclaim `Delete`. [VERIFIED: `k8s/router-ai-atius/*` e `scripts/k3s-router-*`] [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/] [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/]

O backup citado está fora desta worktree, no checkout principal, e o SQL tem apenas 643 bytes. Isso exige substituição por backup novo criado por `pg_dump` 17 diretamente do PostgreSQL 17 nativo do host. O PostgreSQL Podman contém `DBRouterAiAtius` vazio e não pode participar como fonte; o PgBouncer `10.11.1.11:6432` é somente cross-check. Tamanho isolado não prova corrupção, mas o artefato antigo não sustenta um gate de produção. [VERIFIED: verdade live corrigida + filesystem read-only]

**Primary recommendation:** Planejar Phase 29 em gates fail-closed: aliviar DiskPressure de forma sustentável → revalidar cinco minutos do estado atual → produzir com `pg_dump` 17 e verificar backup do PostgreSQL 17 host → gerar bootstrap fresco vinculado aos manifests → pinning rígido dos pods em `atius-srv-1` → subir somente PostgreSQL 17 → provar target integralmente limpo → restaurar em transação única e validar → subir Redis/router shadow → smoke autenticado completo via `ClusterIP` diretamente alcançável pelo host → decisão formal; retry do restore só após no-go explícito e Phase 30 só inicia com todos os gates verdes. [VERIFIED: síntese de repo + auditoria live] [CITED: https://www.postgresql.org/docs/current/backup-dump.html]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|---|---|---|---|
| Pinning de todos os pods | Orquestração k3s | Node `atius-srv-1` | Scheduler deve impor hostname; preferência não basta. [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/] |
| Restore de produção | Database / Storage | Orquestração k3s | Postgres deve estar isolado do app até o restore e a validação terminarem. [CITED: https://www.postgresql.org/docs/current/backup-dump.html] |
| Shadow endpoint | API / Backend | Host networking | Service ClusterIP expõe o router ao host sem alterar Apache público. [CITED: https://kubernetes.io/docs/concepts/services-networking/service/] |
| Cutover Apache | Host edge | Service k3s | Apache preserva TLS/paths e troca somente os upstreams Go (`/v1`, `/api`, `/health`, catch-all). [VERIFIED: `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf`] |
| Rollback | Host edge | Podman | Primeiro restaura Apache; Podman permanece ativo e validável durante soak. [VERIFIED: `docs/K3S-MIGRATION.md` e scripts de rollback] |
| Persistência | local-path no srv1 | Backup externo | PVC local não é HA e acompanha o nó; backup é a proteção real. [CITED: https://kubernetes.io/docs/concepts/storage/volumes/#local] |

<phase_requirements>
## Phase Requirements

`.planning/REQUIREMENTS.md` não existe nesta worktree. Os IDs abaixo são os associados à Phase 29 no ROADMAP. [VERIFIED: repo]

| ID | Description | Research Support |
|---|---|---|
| PHASE-22-K3S-PREFLIGHT | Preflight k3s | Gates de DiskPressure, métricas, pinning, capacidade e endpoint. [VERIFIED: ROADMAP + live cluster] |
| PHASE-22-STATEFUL-DATA | Dados stateful | Restore staged, validação do dump e política Retain. [VERIFIED: ROADMAP + manifests] |
| PHASE-22-RUNTIME-PARITY | Paridade do runtime | Smoke fail-closed de health, auth, shape e embeddings. [VERIFIED: ROADMAP + smoke scripts] |
</phase_requirements>

## Project Constraints (from AGENTS.md)

- Toda tarefa pesada deve usar no máximo 20% da CPU total; no host de 4 vCPU, máximo 0.8 CPU. Builds/testes pesados usam `scripts/podman-admin.sh`. [VERIFIED: `AGENTS.md`]
- Em k3s, cada container dos Pods gerenciados deve pedir e limitar exatamente `500m`; a validação de manifests deve falhar para qualquer valor diferente. [VERIFIED: `AGENTS.md` + gate atual]
- Produção deve permanecer Go-only, sem `model-detailed`; `/v1/models` mantém root `{"data":[...]}` sem campos internos. [VERIFIED: `AGENTS.md`]
- `embedding-gte-v1` permanece governado pelo router Go e deve retornar dimensão 768. [VERIFIED: `AGENTS.md`]
- Nenhum segredo pode entrar em repo, docs, logs ou chat; HashiCorp Vault é a fonte autoritativa. [VERIFIED: `AGENTS.md`]
- Não alterar/remover branding/attribution protegidos de new-api/QuantumNous. [VERIFIED: `AGENTS.md`]
- Graphify é obrigatório, mas não substitui leitura/testes; o graph estava fresco e não retornou nós para as consultas desta fase. [VERIFIED: Graphify read-only]
- O vault Obsidian foi consultado, mas não recebeu nota porque o usuário proibiu qualquer escrita fora deste arquivo. [VERIFIED: restrição do usuário]

## Standard Stack

### Core

| Componente | Versão/estado | Purpose | Why Standard |
|---|---|---|---|
| k3s | v1.35.5+k3s1 | Scheduler, Service, StatefulSet, PVC | Runtime live do cluster. [VERIFIED: `k3s --version`] |
| PostgreSQL target | imagem por digest, major 17 obrigatória | DB shadow/target | O script e a evidência devem rejeitar target que não reporte `server_version_num` 17; restore pelo `psql` do próprio Pod PG17. [VERIFIED: manifest + contrato live] |
| Redis image | `redis:7-alpine` | Cache shadow | Já fixado, efêmero no shadow. [VERIFIED: `redis.yaml`] |
| Apache | 2.4.58 | TLS e edge público | Edge live atual; não introduzir Ingress. [VERIFIED: `apache2ctl -v` + vhost live] |
| Podman | 4.9.3 | Produção/rollback durante soak | Runtime atual preservado até decisão final. [VERIFIED: `podman --version` + CONTEXT] |

### Supporting

| Ferramenta | Versão | Purpose | When to Use |
|---|---|---|---|
| `pg_dump` | 17.x | Criar backup lógico | Executar diretamente no host contra o PostgreSQL 17 canônico em `127.0.0.1:8745`; rejeitar outra major. [VERIFIED: verdade live] |
| `psql` | 17.x no Pod target | Restore plain SQL | Usar `-X --set ON_ERROR_STOP=on --single-transaction` no PostgreSQL 17 k3s. [VERIFIED: contrato do restore] [CITED: https://www.postgresql.org/docs/current/backup-dump.html] |
| curl | 8.5.0 | Smokes HTTP | Shadow e público. [VERIFIED: host CLI] |
| Python | 3.12.3 | Assertions JSON/embeddings | Script de smoke existente. [VERIFIED: host CLI + scripts] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|---|---|---|
| ClusterIP | Ingress | Fora do escopo e cluster não tem IngressClass; a auditoria live já provou host → Service. [VERIFIED: locked decision + live cluster] |
| ClusterIP | NodePort/hostPort/port-forward | Exposição adicional ou processo efêmero sem necessidade; o endpoint shadow permanece declarativo e interno. [VERIFIED: locked decision] |
| local-path | Storage distribuído | Melhor HA, mas expande escopo; decisão atual exige mesmo host e rollback por backup. [ASSUMED] |

**Installation:** nenhuma dependência externa deve ser instalada na Phase 29; usar componentes já disponíveis. [VERIFIED: environment audit]

## Package Legitimacy Audit

Não aplicável: a fase não instala packages npm/PyPI/crates. Imagens existentes devem ser pinadas por digest após validar arquitetura `arm64`, mas isso é supply-chain de container, não package install. [VERIFIED: manifests + node architecture]

## Architecture Patterns

### System Architecture Diagram

```text
backup novo + checksum
        |
        v
[Gate A: srv1 sem DiskPressure estável]
        | falha -> NO-GO, Podman/Apache intactos
        v
namespace + config + Secret + Postgres/PVC (pinados srv1)
        |
        v
prova target PG17 integralmente limpo
        |
        v
restore psql -- ON_ERROR_STOP --single-transaction --> validação estrutural/contagens
        | falha -> apagar/recriar target shadow somente; NO-GO
        v
Redis + Router/PVC (pinados srv1)
        |
        v
Service ClusterIP --> smoke direto do host no spec.clusterIP:3000
        | falha -> NO-GO, Apache continua 127.0.0.1:3000
        v
GO Phase 29 --> Phase 30: backup vhost -> retarget Apache -> public smoke
        | falha durante soak
        v
restore vhost -> reload Apache -> smoke Podman
```

### Recommended Project Structure

```text
k8s/router-ai-atius/
├── namespace.yaml
├── configmap.yaml
├── postgres.yaml       # PostgreSQL 17 pinned srv1, PVC protected
├── redis.yaml          # pinned srv1
└── router.yaml         # pinned srv1, Service ClusterIP
scripts/
├── k3s-router-preflight.sh
├── k3s-router-backup.sh
├── k3s-router-restore-rehearsal.sh   # Wave 0 gap
├── k3s-router-apply-shadow.sh        # staged, never monolithic
├── k3s-router-smoke.sh               # token mandatory
└── k3s-router-rollback-check.sh
```

### Pattern 1: Hard node pinning

**What:** adicionar a cada PodSpec de Postgres, Redis e router `nodeSelector: {kubernetes.io/hostname: atius-srv-1}` (ou required node affinity por `metadata.name`). [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/]

**When to use:** sempre nesta fase; é requisito locked, não otimização. [VERIFIED: user constraint]

```yaml
# Source: Kubernetes official docs
spec:
  nodeSelector:
    kubernetes.io/hostname: atius-srv-1
```

Não adicionar toleration para `node.kubernetes.io/disk-pressure`; isso permitiria scheduling em condição que já está causando eviction. O gate exige `DiskPressure=False`, taint ausente e estabilidade observada antes do apply. [VERIFIED: live cluster] [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/]

### Pattern 2: Restore before application start

**What:** aplicar namespace/config/secret/PostgreSQL 17 primeiro; aguardar readiness; provar que o schema target está integralmente limpo; restaurar atomicamente com `psql -X --set ON_ERROR_STOP=on --single-transaction`; validar; somente depois aplicar Redis/router. Uma nova tentativa só pode ocorrer com opt-in explícito sobre evidência anterior `no-go`, arquivada antes do retry. [CITED: https://www.postgresql.org/docs/current/backup-dump.html]

```bash
# Source: PostgreSQL official docs; valores vêm do Secret/Vault em runtime
psql -X --set ON_ERROR_STOP=on --single-transaction -d DBRouterAiAtius < DBRouterAiAtius.sql
```

O restore deve registrar checksum do dump, server/client versions, exit code, schema/tables esperados e contagens/invariantes não sensíveis. [ASSUMED]

### Pattern 3: Stable shadow endpoint

**What:** Service `ClusterIP`, porta `3000`, resolvido do objeto live e acessado diretamente pelo host para o smoke shadow. A auditoria live já provou host → rede de Services; não criar Ingress, NodePort, hostPort, port-forward persistente nem regra de firewall nesta fase. [CITED: https://kubernetes.io/docs/concepts/services-networking/service/]

```yaml
# Source: Kubernetes official docs
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 3000
      targetPort: http
```

O gate deve ler `spec.clusterIP`, exigir EndpointSlice local/Ready e executar o smoke contra esse endereço sem alterar o Apache público. [VERIFIED: auditoria live + decisão vinculante]

### Pattern 4: PVC protection

**What:** `storageClassName: local-path`, `WaitForFirstConsumer` e pinning no mesmo PodSpec; após binding e antes de restore, mudar o PV produzido para `Retain`, ou impedir qualquer delete de PVC no rollback normal. [CITED: https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode] [CITED: https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/]

### Anti-Patterns to Avoid

- **Tolerar DiskPressure:** contorna o scheduler, mas não elimina eviction/image GC. [VERIFIED: live journal] [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/]
- **Apply monolítico:** router pode inicializar/migrar DB vazio antes do restore. [VERIFIED: current apply script/manifests]
- **`latest` + `IfNotPresent`:** resultado depende do cache local; pin por digest aprovado. [VERIFIED: current manifest] [ASSUMED]
- **Smoke opcional:** token ausente hoje retorna exit 0; gate deve falhar. [VERIFIED: `k3s-router-smoke.sh`]
- **Delete namespace/PVC como rollback:** com reclaim `Delete`, pode destruir o restore ensaiado. [VERIFIED: live StorageClass] [CITED: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Scheduling | script que move pods após start | nodeSelector/required affinity | Scheduler aplica invariant antes de bind. [CITED: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/] |
| Restore error detection | grep de logs | `psql -X --set ON_ERROR_STOP=on` | Falha com exit não-zero em erro SQL. [CITED: https://www.postgresql.org/docs/current/backup-dump.html] |
| Edge shadow | túnel shell persistente ou exposição adicional | Service ClusterIP | Endpoint declarativo, interno e já alcançável pelo host. [CITED: https://kubernetes.io/docs/concepts/services-networking/service/] |
| Secret storage | `.env` no repo | Vault → temporary env/stdin → Kubernetes Secret | Política do projeto proíbe persistência de segredo. [VERIFIED: `AGENTS.md`] |

**Key insight:** o cutover não é um único apply; é uma cadeia de provas reversíveis, e cada gate deve impedir automaticamente o próximo estágio. [ASSUMED]

## Runtime State Inventory

| Category | Items Found | Action Required |
|---|---|---|
| Stored data canônica | PostgreSQL 17 host em `127.0.0.1:8745`, cluster `/var/lib/postgresql/17/main`, unit `postgresql@17-main`; PgBouncer `10.11.1.11:6432` para cross-check; backup antigo de 643 bytes fora da worktree; PVC target ainda inexistente. [VERIFIED: verdade live + repo principal + cluster] | `pg_dump` 17 direto do host, checksum, restore atômico em target PG17 integralmente limpo e validação de invariantes. |
| Stored data legada | PostgreSQL Podman com `DBRouterAiAtius` vazio. [VERIFIED: verdade live] | Excluir de toda seleção de origem; nunca consultar para backup ou restore. |
| Live service config | Apache live aponta Go paths e catch-all para `127.0.0.1:3000`; docs continuam em `127.0.0.1:3003`. [VERIFIED: live vhost] | Phase 30 troca apenas regras do router; não tocar docs/assets. |
| OS-registered state | `container-router-ai-atius.service` é rollback; k3s é system service. [VERIFIED: repo docs/live inspection] | Manter Podman ativo durante soak; não restartar no shadow. |
| Secrets/env vars | Secret namespace não existe; valores reais devem vir do Vault. [VERIFIED: live cluster + AGENTS] | Criar fora do git, sem imprimir valores; remover arquivo temporário com segurança. |
| Build artifacts | Imagem Podman local é arm64; manifest usa tag `latest`. [VERIFIED: podman inspect + manifest] | Resolver/pinar digest arm64 antes do apply. |

## Common Pitfalls

### Pitfall 1: “28 GB livres” ser tratado como saudável

**What goes wrong:** scheduler bloqueia novos pods e kubelet evicta workloads. **Why:** thresholds usam percentual/ephemeral-storage e image GC, não apenas GB absolutos. **Avoid:** exigir `DiskPressure=False`, taint ausente, journal sem loop de reclaim e margem sustentada. **Warning:** evictions, image GC >85%, taint NoSchedule. [VERIFIED: live kubelet config, node condition e journal]

### Pitfall 2: Restore parcial parecer sucesso

**What goes wrong:** `psql` continua após erro por padrão ou deixa estado parcial. **Avoid:** provar target integralmente limpo, usar `-X --set ON_ERROR_STOP=on --single-transaction`, validar invariantes pós-restore e bloquear retry que não parta de no-go explícito arquivado. [CITED: https://www.postgresql.org/docs/current/backup-dump.html]

### Pitfall 3: PVC deletado junto com namespace

**What goes wrong:** `local-path` live usa reclaim `Delete`. **Avoid:** PV `Retain`, backup externo validado e rollback sem delete. [VERIFIED: live StorageClass] [CITED: https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/]

### Pitfall 4: Apache cutover incompleto

**What goes wrong:** mudar só `/v1/` deixa `/api`, `/health`, login ou SPA no Podman, criando runtime misto. **Avoid:** inventário exato das regras Go e smoke por path; preservar docs `:3003` e aliases estáticos. [VERIFIED: live vhost]

### Pitfall 5: Router em outro node

**What goes wrong:** PVC local e app divergem ou workload viola decisão locked. **Avoid:** pinning em todos os três PodSpecs e assertion pós-rollout de `.spec.nodeName == atius-srv-1`. [CITED: https://kubernetes.io/docs/concepts/storage/volumes/#local]

## Code Examples

### Gate read-only de scheduling

```bash
sudo -n k3s kubectl get node atius-srv-1 \
  -o jsonpath='{.status.conditions[?(@.type=="DiskPressure")].status}{"\n"}'
sudo -n k3s kubectl get pods -n router-ai-atius \
  -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName,READY:.status.containerStatuses[*].ready
```

[VERIFIED: Kubernetes CLI live]

### Gate de Service/endpoints

```bash
sudo -n k3s kubectl -n router-ai-atius get svc router-ai-atius -o wide
sudo -n k3s kubectl -n router-ai-atius get endpointslice \
  -l kubernetes.io/service-name=router-ai-atius -o wide
```

[CITED: https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/]

### Smoke fail-closed

```bash
: "${K3S_ROUTER_BASE_URL:?required}"
: "${ATIUS_ROUTER_TOKEN:?required}"
scripts/k3s-router-smoke.sh
```

[VERIFIED: correction pattern for current script]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| Metrics unavailable | Metrics live (`kubectl top`) | confirmado 2026-07-12 | Remove blocker de observabilidade, não DiskPressure. [VERIFIED: live cluster] |
| Suposição de que ClusterIP não era acessível pelo host | ClusterIP resolvido live e acessado diretamente | auditoria Phase 29 | Evita exposição adicional e mantém Apache intacto. [VERIFIED: auditoria live] |
| Apply all-at-once | Restore staged e atômico em target integralmente limpo | recomendação Phase 29 | Impede app contra DB vazio e evita estado parcial. [VERIFIED: contrato corrigido] |

**Deprecated/outdated:** “Metrics API not available” no CONTEXT está desatualizado; métricas estavam disponíveis nesta pesquisa. [VERIFIED: `kubectl top nodes`]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | `kubectl port-forward` não deve ser endpoint durável. | Alternatives | Baixo; ClusterIP já é declarativo e alcançável pelo host. |
| A2 | Storage distribuído é fora do escopo da Phase 29. | Alternatives | Médio; usuário pode optar por expandir escopo. |
| A3 | Invariantes de contagem/objetos devem ser definidos para validar restore. | Pattern 2 | Alto; sem critérios, restore pode passar incompleto. |

## Open Questions — RESOLVED

1. **RESOLVED — origem exata dos dados de producao:** o backup fresco sai com `pg_dump` 17 diretamente do PostgreSQL 17 nativo do host em `127.0.0.1:8745`, database `DBRouterAiAtius`, cluster `/var/lib/postgresql/17/main`, unit `postgresql@17-main`. O PgBouncer em `10.11.1.11:6432` serve para cruzar identidade, user/version e invariantes nao sensiveis; nao e a origem do dump. O backup antigo permanece invalido como gate.

   O PostgreSQL Podman possui `DBRouterAiAtius` vazio e nunca e fonte.

2. **RESOLVED — transporte shadow:** usar somente o `ClusterIP` do Service `router-ai-atius`. A auditoria live provou conectividade host → rede de Services; nao selecionar NodePort, nao abrir firewall, nao usar Ingress/hostPort e nao alterar Apache na Phase 29.

3. **RESOLVED — janela de estabilidade:** exigir `DiskPressure=False` e ausencia de `node.kubernetes.io/disk-pressure` por cinco minutos continuos, amostrados no maximo a cada 30 segundos, depois de recuperar pelo menos 20 GiB e atingir alvo de pelo menos 25% livre.

4. **RESOLVED — validade das evidencias:** `cleanup.json` registra reclaim historico e deve ser do cluster atual; o preflight nao cria JSON proprio e sempre revalida cinco minutos do estado atual. `bootstrap.json` deve ser do cluster atual, ter no maximo uma hora e corresponder ao hash corrente dos manifests. `backup.json` e `restore.json` registram backup/restore, com restore atomico, target limpo e retry somente de no-go explicito arquivado.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| k3s | cluster apply/read | ✓ | 1.35.5+k3s1 | — |
| Metrics API | capacity gate | ✓ | live | — |
| local-path | PVC | ✓ | default, WaitForFirstConsumer/Delete | backup + Retain |
| IngressClass | ingress | ✗ | — | nao necessario; host alcanca ClusterIP diretamente |
| Apache | Phase 30 edge | ✓ | 2.4.58 | rollback vhost |
| Podman | rollback | ✓ | 4.9.3 | — |
| pg_dump/psql | backup/restore | ✓ | pg_dump 17 host / psql 17 target | rejeitar major divergente |
| curl/Python/jq | smokes | ✓ | 8.5.0 / 3.12.3 / 1.7 | — |

[VERIFIED: live environment audit]

**Missing dependencies with no fallback:** capacidade saudável do `atius-srv-1`; é blocker operacional, não binário ausente. [VERIFIED: live cluster]

**Missing dependencies with fallback:** nenhuma para exposicao shadow; IngressClass nao e necessario porque o host alcanca ClusterIP diretamente. [VERIFIED: auditoria live + decisao locked]

## Validation Architecture

### Test Framework

| Property | Value |
|---|---|
| Framework | Bash gates + kubectl + HTTP/Python smoke [VERIFIED: repo] |
| Config file | nenhum; scripts em `scripts/k3s-router-*` [VERIFIED: repo] |
| Quick run command | `bash -n scripts/k3s-router-*.sh` [VERIFIED: executado] |
| Full suite command | preflight read-only → server dry-run → staged restore → smoke autenticado [ASSUMED: plano proposto] |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| PHASE-22-K3S-PREFLIGHT | srv1 saudável e exclusivo | integration/read-only | `scripts/k3s-router-preflight.sh` + assertions | ⚠️ precisa endurecer |
| PHASE-22-STATEFUL-DATA | dump restaura sem erro e está completo | integration | `scripts/k3s-router-restore-rehearsal.sh` | ❌ Wave 0 |
| PHASE-22-RUNTIME-PARITY | health/auth/models/embedding | integration | `scripts/k3s-router-smoke.sh` | ⚠️ token deve ser obrigatório |

### Sampling Rate

- **Per task commit:** `bash -n scripts/k3s-router-*.sh` e render/dry-run dos manifests. [ASSUMED]
- **Per wave merge:** preflight e assertions read-only. [ASSUMED]
- **Phase gate:** restore + shadow smoke completos, evidência arquivada, sem tráfego público. [VERIFIED: CONTEXT]

### Wave 0 Gaps

- [ ] `scripts/k3s-router-restore-rehearsal.sh` — restore staged e fail-closed.
- [ ] assertions de node pinning para os três workloads.
- [ ] smoke exigir token e embeddings; “skipped” nunca é GO.
- [ ] backup verifier (checksum, SQL parse/restore e invariantes).
- [ ] ClusterIP resolvido e alcancavel diretamente do host, com EndpointSlice local/Ready e sem exposicao adicional.
- [ ] run evidence/go-no-go artifact definido pelo planner.

[VERIFIED: gap analysis of current repo]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | yes | smoke com token real carregado do Vault, sem logging. [VERIFIED: AGENTS + smoke] |
| V3 Session Management | yes | preservar `SESSION_SECRET` entre runtimes para evitar invalidar sessões no cutover. [ASSUMED] |
| V4 Access Control | yes | `/v1/models` sem token deve continuar 401. [VERIFIED: CONTEXT/smoke] |
| V5 Input Validation | yes | assertions exatas de JSON e dimensão 768. [VERIFIED: smoke] |
| V6 Cryptography | yes | TLS termina no Apache; secrets-encryption está habilitado no k3s. [VERIFIED: live configs] |

### Known Threat Patterns for k3s/Apache/Postgres

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| Secret vazado em env/temp/log | Information Disclosure | Vault, arquivo mode 0600 quando inevitável, cleanup e nunca `set -x`. [VERIFIED: AGENTS] |
| Endpoint shadow exposto indevidamente | Information Disclosure / DoS | manter Service ClusterIP e provar EndpointSlice local/Ready; nenhuma exposição adicional. [VERIFIED: decisão vinculante] |
| Restore sobre DB incorreto ou parcial | Tampering | origem host PG17, target PG17 integralmente limpo, transação única, ON_ERROR_STOP, retry somente de no-go arquivado e invariantes. [CITED: https://www.postgresql.org/docs/current/backup-dump.html] |
| PVC apagado | Tampering / DoS | reclaim Retain e backup externo verificado. [CITED: https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/] |
| Supply-chain por tag `latest` | Tampering | pin de digest arm64 aprovado. [ASSUMED] |

## Sources

### Primary (HIGH confidence)

- Repo: `29-CONTEXT.md`, ROADMAP, `docs/K3S-MIGRATION.md`, `k8s/router-ai-atius/*`, `scripts/k3s-router-*` — escopo e gaps. [VERIFIED: codebase grep/read]
- Cluster/host live read-only em 2026-07-12 — node conditions, taints, metrics, storage, namespace, Apache, filesystem e journal. [VERIFIED: live commands]
- https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ — node selection.
- https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ — taints/tolerations.
- https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode — delayed binding.
- https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming — reclaim behavior.
- https://kubernetes.io/docs/concepts/services-networking/service/ — ClusterIP e Services.
- https://www.postgresql.org/docs/current/backup-dump.html — dump/restore fail-closed.

### Secondary (MEDIUM confidence)

- Context7 `/websites/kubernetes_io` e `/websites/postgresql_current`; o seam classificou provider como MEDIUM, embora os links retornados sejam docs oficiais. [VERIFIED: research seam]

### Tertiary (LOW confidence)

- Assumptions A1–A3 e controle de supply-chain que ainda exige validação live.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versões e estado live verificados.
- Architecture: HIGH — gaps confirmados no repo e padrões ancorados em docs oficiais.
- Pitfalls: HIGH — DiskPressure/evictions e riscos dos scripts foram observados diretamente.

**Research date:** 2026-07-12
**Valid until:** 2026-07-19 para estado live; padrões estáveis até 2026-08-11.
