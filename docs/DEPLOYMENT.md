<!-- generated-by: gsd-doc-writer -->
# Deployment

Este documento descreve o deployment atual do `router-ai-atius`, fork de `github.com/QuantumNous/new-api`, com foco na operacao real em Podman rootless, no pipeline de imagem GHCR e na trilha k3s que ainda nao foi promovida para producao.

## Estado Atual

- Runtime canonico de producao: Podman rootless, gerenciado por `systemctl --user`.
- Unit principal: `container-router-ai-atius.service`.
- Pod atual: `atius-ai-router`, com infra container, `router-ai-atius`, `redis` e `postgres`.
- Imagem do router: `ghcr.io/giovannimnz/router-ai-atius:latest`.
- Edge publico: Apache/Cloudflare para `https://router.atius.com.br`, com backend local em `127.0.0.1:3000`. <!-- VERIFY: Edge publico e alvo Apache/Cloudflare sao estado de infraestrutura fora do repositorio; revalidar antes de cutover. -->
- k3s: manifests e scripts existem para shadow deployment, mas o namespace `router-ai-atius` nao esta aplicado no cluster local observado.

Validacoes sem segredo feitas durante a escrita deste documento:

```bash
curl -fsS -o /dev/null -w 'local_api_status=%{http_code}\n' http://127.0.0.1:3000/api/status
curl -fsS -o /dev/null -w 'public_health=%{http_code}\n' https://router.atius.com.br/health
curl -sS -o /dev/null -w 'public_models_unauth=%{http_code}\n' https://router.atius.com.br/v1/models
```

Resultados observados: local `200`, publico `/health` `200`, publico `/v1/models` sem token `401`.

## Deployment Targets

| Target | Status | Configuracao | Uso |
| --- | --- | --- | --- |
| Podman rootless + systemd user | Canonico atual | `scripts/pull-and-restart.sh`, `scripts/podman-admin.sh`, units `container-*`/`pod-*` no user systemd | Producao atual |
| GHCR | Canonico para artefato | `.github/workflows/docker-build.yml`, `.github/workflows/docker-publish.yml`, `Dockerfile` | Build e publicacao da imagem |
| Podman compose/recovery | Trilha existente, nao canonica atual | `podman-compose.yml`, `scripts/recreate-pod.sh`, `scripts/podman-up.sh`, `scripts/podman-down.sh` | Recovery/manual, historico de migracao Docker -> Podman |
| k3s shadow | Planejado, nao promovido | `k8s/router-ai-atius/*.yaml`, `scripts/k3s-router-*.sh`, `docs/K3S-MIGRATION.md` | Shadow deployment e cutover futuro |
| Docker compose legado | Referencia historica | `docker-compose.yml` | Nao e o runtime canonico atual |
| systemd rootful generico | Exemplo legado | `new-api.service` | Nao usar como fonte de producao atual |

### Podman Rootless Canonico

O caminho operacional atual e GHCR -> Podman user unit:

```bash
scripts/pull-and-restart.sh latest
```

Para um tag versionado:

```bash
scripts/pull-and-restart.sh vX.Y.Z
```

O script puxa `ghcr.io/giovannimnz/router-ai-atius:<tag>`, retaga tags versionadas como `:latest` para a unit atual, reinicia `container-router-ai-atius.service` com `systemctl --user` e valida health local. Ele tambem possui recuperacoes pontuais para storage rootless stale e erro PostgreSQL de cached plan via restart unico do PgBouncer.

Comandos de estado:

```bash
systemctl --user status container-router-ai-atius.service --no-pager
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
scripts/podman-admin.sh limits
scripts/podman-admin.sh inspect-limits
```

O wrapper `scripts/podman-admin.sh` e obrigatorio para builds e tarefas pesadas locais. No host atual de 4 vCPU, ele calcula `cpu_limit=0.800`, `cpu_quota=80000`, `build_jobs=1` e aplica profile systemd user com `CPUQuota=80%`, que representa 20% da CPU total do host.

## Build Pipeline

O `Dockerfile` constroi:

1. frontend `web/default` com Bun;
2. frontend `web/classic` com Bun;
3. binario Go `new-api` a partir do modulo `github.com/QuantumNous/new-api`;
4. imagem final Debian slim expondo porta `3000`.

O `.dockerignore` remove do contexto de build os diretorios de runtime `data/`, `db-data/` e `backups/`. A regra operacional do fork tambem exige manter `runtime/`, `logs/` e backups fora de imagens quando eles existirem como estado local.

Workflows relevantes:

| Workflow | Trigger | Resultado |
| --- | --- | --- |
| `.github/workflows/docker-build.yml` | push de tag ou `workflow_dispatch` com `tag` | build multi-arch `amd64`/`arm64`, push GHCR, manifest `:<tag>` e `:latest`, assinatura cosign |
| `.github/workflows/docker-publish.yml` | sucesso do `Sync Upstream + Release` ou `workflow_dispatch` | build/push GHCR e manifest multi-arch |
| `.github/workflows/docker-image-nightly.yml` | branch `nightly` ou manual | imagem `:nightly` e tag nightly versionada |
| `.github/workflows/release.yml` | push de tag nao-alpha ou manual | binarios Linux/macOS/Windows e GitHub Release |
| `.github/workflows/sync.yml` | cron diario ou manual | sync do fork, build frontend/backend e preparacao de release |

Deploy local a partir da imagem publicada:

```bash
scripts/deploy-ghcr.sh latest
```

`scripts/deploy-ghcr.sh` e apenas um wrapper de compatibilidade para `scripts/pull-and-restart.sh`.

Para build local pesado, usar sempre o wrapper de CPU:

```bash
./scripts/podman-admin.sh build -f Dockerfile -t ghcr.io/giovannimnz/router-ai-atius:<tag> .
```

Para tarefas pesadas fora de Podman:

```bash
./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun run typecheck && bun run build'
```

## Environment Setup

Nao commitar `.env`, tokens, senhas, DSNs preenchidos ou dumps com credenciais. Os arquivos versionados devem conter somente nomes de variaveis, placeholders ou templates.

Variaveis principais do runtime Podman:

| Variavel | Obrigatoria | Fonte/uso |
| --- | --- | --- |
| `POSTGRES_PASSWORD` | Sim | Senha do PostgreSQL, injetada pelo environment do runtime |
| `REDIS_PASSWORD` | Sim | Senha do Redis, injetada pelo environment do runtime |
| `SESSION_SECRET` | Sim | Secret de sessao da aplicacao |
| `SQL_DSN` | Sim quando nao derivada | DSN PostgreSQL do backend Go; nunca documentar com senha real |
| `REDIS_CONN_STRING` | Sim quando nao derivada | Conn string Redis; nunca documentar com senha real |
| `TRUST_PROXY` | Recomendado em producao | Habilita operacao atras do reverse proxy |
| `ERROR_LOG_ENABLED` | Opcional | Habilita log de erro da aplicacao |
| `BATCH_UPDATE_ENABLED` | Opcional | Habilita batch updater |
| `NODE_NAME` | Opcional | Identifica o node no runtime |
| `TZ` | Opcional | Timezone do container |

`scripts/pull-and-restart.sh` tambem pode ler `GHCR_USER` e `GHCR_TOKEN` de `/home/ubuntu/.config/router-ai-atius/.env` para login no GHCR. Nao registrar o valor do token em logs, docs ou commits.

Para k3s, os secrets esperados pelo template `k8s/router-ai-atius/secret.example.env` sao:

| Secret key | Uso |
| --- | --- |
| `POSTGRES_PASSWORD` | PostgreSQL do namespace `router-ai-atius` |
| `REDIS_PASSWORD` | Redis do namespace `router-ai-atius` |
| `SESSION_SECRET` | Sessao do router |
| `ROUTER_ADMIN_TOKEN` | Operacao/admin do router |
| `ATIUS_ROUTER_TOKEN` | Smoke autenticado do router |

Criacao do secret k3s, sem commitar o arquivo preenchido:

```bash
cp k8s/router-ai-atius/secret.example.env /tmp/router-ai-atius.secret.env
$EDITOR /tmp/router-ai-atius.secret.env
kubectl -n router-ai-atius create secret generic router-ai-atius-secrets \
  --from-env-file=/tmp/router-ai-atius.secret.env \
  --dry-run=client -o yaml | sudo -n k3s kubectl apply -f -
```

O ConfigMap k3s (`k8s/router-ai-atius/configmap.yaml`) define valores nao-secretos como `POSTGRES_HOST`, `REDIS_HOST`, `EMBEDDING_GOVERNOR_MODELS`, `TEI_BASE_URL` e `TEI_RERANKER_BASE_URL`. Como k3s ainda e shadow, reconciliar esse ConfigMap com o contrato operacional vigente antes de qualquer promocao; hoje o contrato live aponta ambos os upstreams TEI para o `horistic-srv`.

## K3s Shadow Path

A arvore `k8s/router-ai-atius/` contem:

- `namespace.yaml`: namespace dedicado `router-ai-atius`;
- `router.yaml`: `Deployment/router-ai-atius`, `Service/router-ai-atius` e PVC `/data`;
- `postgres.yaml`: `StatefulSet/router-ai-atius-postgres`, service e PVC;
- `redis.yaml`: `Deployment/router-ai-atius-redis` e service;
- `configmap.yaml`: configuracao nao secreta do router.

Estado e restricoes da trilha:

- sem ingress controller nesta fase;
- Apache/Cloudflare continuam como edge ate cutover manual;
- `local-path` e single-node, `ReadWriteOnce`, sem HA;
- Redis usa `emptyDir` no shadow inicial;
- logs do router usam `emptyDir`;
- Podman permanece como rollback ate shadow smoke, restore rehearsal e rollback estarem validados.

Preflight:

```bash
scripts/k3s-router-preflight.sh
```

Validacao server-side sem criar recursos:

```bash
scripts/k3s-router-validate-manifests.sh
```

Apply shadow explicito:

```bash
RUN_K3S_ROUTER_APPLY=1 scripts/k3s-router-apply-shadow.sh
```

Smoke shadow:

```bash
K3S_ROUTER_BASE_URL=http://<endpoint-shadow> \
ATIUS_ROUTER_TOKEN=<token-de-teste> \
scripts/k3s-router-smoke.sh
```

O smoke exige `/api/status` ou `/health` HTTP `200`, `/v1/models` sem token HTTP `401`, `/v1/models` autenticado com top-level somente `data`, ausencia dos campos internos `pricing_source`, `pricing_estimated` e `pricing_version`, e embedding `embedding-gte-v1` com dimensao `768`.

## Rollback Procedure

### Rollback Podman/GHCR

1. Escolher o tag anterior conhecido como bom.
2. Puxar e retagar para `:latest` via script:

```bash
scripts/pull-and-restart.sh vX.Y.Z
```

3. Validar unit e pod:

```bash
systemctl --user status container-router-ai-atius.service --no-pager
podman ps --filter pod=atius-ai-router --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
bin/clianything status
curl -fsS http://127.0.0.1:3000/api/status
```

Se a imagem ja estiver localmente retagada, o restart direto tambem e aceito:

```bash
systemctl --user restart container-router-ai-atius.service
```

### Rollback de Cutover k3s

O cutover k3s ainda nao e producao. Se Apache tiver sido apontado para k3s durante um teste controlado:

1. restaurar o backup do vhost Apache;
2. rodar `apache2ctl configtest`;
3. recarregar Apache;
4. validar Podman;
5. rodar smoke publico.

Comandos de apoio:

```bash
scripts/k3s-router-rollback-check.sh
systemctl --user restart container-router-ai-atius.service
bin/clianything status
podman ps --filter pod=atius-ai-router
```

Antes de qualquer cutover k3s, `scripts/k3s-router-cutover-checklist.sh` exige `CURRENT_PUBLIC_URL`, `K3S_ROUTER_BASE_URL`, `K3S_BACKUP_DIR` e `APACHE_VHOST_BACKUP_PATH`.

## Monitoring

Checks operacionais basicos:

```bash
systemctl --user status container-router-ai-atius.service --no-pager
journalctl --user -u container-router-ai-atius.service -n 160 --no-pager
podman logs router-ai-atius --tail 160
bin/clianything status
bin/clianything providers --all
curl -fsS http://127.0.0.1:3000/api/status
curl -fsS https://router.atius.com.br/health
```

O codigo suporta:

- pprof quando `ENABLE_PPROF=true`, escutando em `0.0.0.0:8005`;
- Pyroscope quando `PYROSCOPE_URL` esta configurada, com app default `new-api`;
- endpoint `/api/uptime/status` usado pelo dashboard/monitoramento interno.

Nao ha dashboard externo de monitoramento, ServiceMonitor Prometheus ou alerting k3s versionado neste repo. Se Pyroscope, Uptime Kuma, Cloudflare ou outro painel externo for usado em producao, documentar apenas nomes de variaveis e paths operacionais, nunca tokens, senhas ou URLs de webhook. <!-- VERIFY: Painel externo de monitoramento, se existir, fica fora deste repositorio. -->
