# K3S migration - router-ai-atius

## Estado atual

- Runtime de producao atual: Podman rootless via `container-router-ai-atius.service`.
- Edge publico atual: Apache/Cloudflare apontando para `127.0.0.1:3000`.
- Runtime canonico atual: backend Go-only, sem `model-detailed` no caminho `/v1/`.
- Banco ativo observado no runtime: `postgresql://admin:${POSTGRES_PASSWORD}@10.1.1.1:6432/DBRouterAiAtius`.
- Redis ativo observado no runtime: `redis://:${REDIS_PASSWORD}@localhost:6379`.
- Providers ativos observados:
  - `DeepSeek`
  - `OpenAI - Codex`
  - `TEI - GTE Embeddings`
- Provider desabilitado mas preservado no catalogo:
  - `MiniMax`

Podman continua sendo o rollback ate o cutover k3s passar nos smokes.

## Decisoes bloqueantes

- Namespace dedicado: `router-ai-atius`.
- Apache/Cloudflare continuam como edge inicial; esta fase nao introduz
  ingress controller.
- `model-detailed` must not return to the `/v1/` path.
- Shadow deployment primeiro; trafego publico so muda depois.
- Banco e estado continuam com rollback explicito ate restore rehearsal e
  shadow smoke passarem.

Bloqueadores atuais observados no cluster:

- `DiskPressure` em `atius-srv-1`.
- `image filesystem` pressure e ausencia de `Metrics API` para leitura de uso
  em `atius-srv-2`/cluster.
- storage `local-path` como unica classe observada.
- ausencia de `IngressClass`.

## Preflight

Rodar antes de qualquer `kubectl apply`:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
scripts/k3s-router-preflight.sh
```

O preflight coleta:

- `graphify status`
- baseline Podman e `bin/clianything`
- providers ativos
- nodes, storage, ingress e eventos do k3s
- evidencia suficiente para `go/no-go`

Interpretacao minima:

- `DiskPressure` ou `image filesystem` pressure exigem mitigacao ou
  node-selection explicito antes de cutover.
- `Metrics API not available` nao bloqueia escrever manifests, mas bloqueia
  qualquer narrativa de capacidade madura.
- `local-path` implica persistencia single-node, `RWO`, reclaim `Delete`,
  sem volume expansion.

## Arquitetura alvo inicial

- Namespace: `router-ai-atius`
- Sem ingress controller nesta fase.
- Apache continua como edge, retargeteando manualmente para um Service k3s
  somente apos shadow validado.
- Stack inicial em k3s:
  - `Deployment/router-ai-atius` com `replicas: 1`
  - `StatefulSet/postgres`
  - `Deployment/redis`
  - `Service/router-ai-atius`
  - PVC de `/data` do router
  - PVC do Postgres
  - `emptyDir` para logs e Redis no shadow inicial

Dependencia externa/adjacente:

- TEI continua fora desta fase e so e lido como dependencia.
- Endpoint preferido para dependencias internas nesta trilha:
  `tei-gte.ai-search.svc.cluster.local`.

## Dados e secrets

- Nenhum secret real vai para git.
- Secrets sao criados em apply-time via:
  `kubectl -n router-ai-atius create secret generic router-ai-atius-secrets`.
- Campos minimos esperados:
  - `POSTGRES_PASSWORD`
  - `REDIS_PASSWORD`
  - `SESSION_SECRET`
  - `ROUTER_ADMIN_TOKEN`
  - `ATIUS_ROUTER_TOKEN`

### Backup

Antes de qualquer restore rehearsal ou shadow deploy:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
scripts/k3s-router-backup.sh
```

Backup required evidence:

- backup path
- `clianything` metadata snapshot
- dump Postgres criado com `pg_dump`
- backup de channels/tokens quando suportado pelo CLI

O backup path must be recorded before shadow deploy.

## Shadow deployment

Shadow-only, sem tocar Apache publico:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
scripts/k3s-router-validate-manifests.sh
RUN_K3S_ROUTER_APPLY=1 scripts/k3s-router-apply-shadow.sh
```

Depois do rollout:

```bash
K3S_ROUTER_BASE_URL=http://<node-or-forwarded-endpoint> \
ATIUS_ROUTER_TOKEN=<token-de-teste> \
scripts/k3s-router-smoke.sh
```

Shadow smoke minimo:

- `/api/status` ou `/health` HTTP 200
- `/v1/models` sem token HTTP 401
- `/v1/models` com token HTTP 200 com root `data` only
- campos proibidos ausentes:
  - `pricing_source`
  - `pricing_estimated`
  - `pricing_version`
- `embedding-gte-v1` retorna dimensao `768`

## Restore rehearsal

Checklist obrigatorio antes de qualquer cutover:

- record `backup directory`
- record `target namespace`
- record `restore command`
- record `kubectl get pods -n router-ai-atius`
- record `bin/clianything` ou verificacao equivalente do DB restaurado
- record `shadow smoke`
- record explicit `go/no-go`

failed restore or failed smoke means no production cutover

## Cutover manual

Sequencia exata:

1. Validar shadow com `scripts/k3s-router-smoke.sh`.
2. Fazer backup do vhost Apache atual.
3. Editar manualmente o backend target do Apache para o endpoint k3s validado.
4. Rodar `apache2ctl configtest`.
5. Recarregar Apache.
6. Rodar smoke publico.
7. Monitorar logs e manter Podman ativo durante soak.

Smokes publicos minimos:

- `https://router.atius.com.br/health` -> 200
- `https://router.atius.com.br/v1/models` sem token -> 401
- `https://router.atius.com.br/v1/models` com token -> 200 com root `data`
  apenas
- forbidden fields ausentes
- `embedding-gte-v1` -> `768`

Podman remains active during soak.

## Rollback

Sequencia de rollback:

1. restore Apache vhost backup
2. `apache2ctl configtest`
3. recarregar Apache
4. validar Podman
5. rodar smoke publico

Comandos essenciais:

```bash
systemctl --user restart container-router-ai-atius.service
bin/clianything status
podman ps --filter pod=atius-ai-router
```

## Go/no-go

Go:

- manifests passam em `--dry-run=server`
- restore rehearsal registrada
- shadow rollout Ready
- shadow smoke passa
- blockers de node/storage documentados com mitigacao real
- Apache backup pronto
- rollback validado

No-go:

- `DiskPressure` sem mitigacao
- `image filesystem` pressure sem mitigacao
- restore rehearsal falha
- shadow smoke falha
- `/v1/models` muda shape ou vaza campos internos
- `embedding-gte-v1` nao retorna `768`
- rollback nao esta documentado e testavel
