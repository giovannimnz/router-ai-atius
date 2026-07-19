# K3S runtime - router-ai-atius

## Estado canonico

Desde 2026-07-19, o runtime publico do Router esta no k3s do
`atius-srv-1`:

- namespace: `router-ai-atius`;
- edge: Apache/Cloudflare;
- backend Apache do app/API: `http://10.43.102.221:3000`;
- docs continuam fora do k3s Router em `http://127.0.0.1:3003`;
- app: `Deployment/router-ai-atius`, uma replica, `500m` request/limit;
- banco: `StatefulSet/router-ai-atius-postgres`, PostgreSQL 17, `500m`;
- Redis: `Deployment/router-ai-atius-redis`, efemero, `500m`;
- storage: `local-path`, RWO, single-node, PVs protegidos com `Retain`;
- node selector obrigatorio: `kubernetes.io/hostname=atius-srv-1`;
- release: `ghcr.io/giovannimnz/router-ai-atius:v2.17.3`;
- image ID: `7b3f62f2694046caacb99ed28a29bad3eb3b1d3a3978e5bc2055e582d0d06f29`;
- manifest OCI importado no CRI: `sha256:4d41867fa1332220d6a04ce87bf3eb893f2129f5f4bb52898fd62ec21e12b064`.

O Router Podman fica parado, mas preservado como rollback pela user unit
`container-router-ai-atius.service`. Nao remover a imagem, o unit ou o banco
fonte anterior durante o soak operacional.

## Decisoes de arquitetura

- O Router permanece Go-only; `model-detailed` nao participa de `/v1/`.
- Nao existe Ingress adicional. Apache continua sendo o edge e aponta para o
  `ClusterIP` do Service.
- A replica unica usa `maxSurge: 0` e `maxUnavailable: 1`. O host nao reserva
  CPU para dois pods Router de `500m` durante rollout.
- Nao tolerar `node.kubernetes.io/disk-pressure`. Se `DiskPressure=True`, o
  rollout e `no-go`; adicionar toleration provoca eviction loop e nao resolve
  capacidade.
- A imagem e importada no namespace containerd `k8s.io` e usada com tag de
  release + `imagePullPolicy: IfNotPresent`. Como o GHCR e privado para pulls
  anonimos, a preimportacao autenticada e obrigatoria enquanto nao houver um
  `imagePullSecret` provisionado no namespace.
- Secrets reais permanecem fora do git no Secret
  `router-ai-atius-secrets`.
- O TEI e dependencia externa pela faixa OCI DRG em
  `http://10.21.1.21:3115`; esta arvore nao gerencia seus recursos.

## CPU e storage

Cada pod normal desta stack segue a unidade operacional do host:

```text
1 pod = requests.cpu 500m = limits.cpu 500m
```

Antes de apply ou rollout:

```bash
sudo k3s kubectl get node atius-srv-1 \
  -o jsonpath='{.status.conditions[?(@.type=="DiskPressure")].status}{"\n"}'
df -h /
scripts/k3s-router-validate-manifests.sh
```

O resultado deve ser `False`. `local-path` nao e HA; por isso backup validado e
reclaim `Retain` sao obrigatorios.

## Backup canonico

```bash
./scripts/podman-admin.sh profile-run -- \
  /usr/bin/bash scripts/k3s-router-backup.sh
```

`k3s-router-backup.sh` usa `umask 077` e aceita:

- `ROUTER_BACKUP_SOURCE=auto`: detecta o backend ativo no vhost Apache;
- `ROUTER_BACKUP_SOURCE=k3s`: dump direto do StatefulSet PostgreSQL;
- `ROUTER_BACKUP_SOURCE=podman`: dump do `SQL_DSN` runtime do rollback.

No modo k3s, o script salva recursos sem Kubernetes Secrets e gera dump com
permissao `600`. O dump so e aceito quando contem tabelas e o marcador final do
PostgreSQL.

Backup k3s validado depois do cutover:

```text
backups/k3s-router-20260719T201321Z
source_mode=k3s
dump_bytes=53737224
```

## Apply e rollout

```bash
scripts/k3s-router-validate-manifests.sh
sudo k3s kubectl apply -f k8s/router-ai-atius/
sudo k3s kubectl -n router-ai-atius rollout status \
  deployment/router-ai-atius --timeout=240s
```

O manifest usa `imagePullPolicy: IfNotPresent`: recriacao depois de garbage
collection tenta recuperar a release da GHCR, enquanto o cache CRI local evita
pull desnecessario. Para preaquecer ou operar sem acesso temporario a GHCR:

```bash
podman save --format oci-archive \
  ghcr.io/giovannimnz/router-ai-atius:v2.17.3 | \
  sudo k3s ctr -n k8s.io images import --digests -
```

Depois da importacao, o nome do tag no manifest deve existir em
`sudo k3s ctr -n k8s.io images list`; um alias criado fora do namespace
`k8s.io` nao e suficiente para o kubelet.

## Smoke shadow ou publico

```bash
K3S_ROUTER_BASE_URL=http://10.43.102.221:3000 \
ATIUS_ROUTER_TOKEN="$ATIUS_ROUTER_TOKEN" \
scripts/k3s-router-smoke.sh

K3S_ROUTER_BASE_URL=https://router.atius.com.br \
ATIUS_ROUTER_TOKEN="$ATIUS_ROUTER_TOKEN" \
scripts/k3s-router-smoke.sh
```

O script valida:

- health `200`;
- `/v1/models` sem token `401`;
- `/v1/models` autenticado com root `data` only;
- ausencia de `pricing_source`, `pricing_estimated` e `pricing_version`;
- `embedding-gte-v1` com dimensao `768` no mesmo base URL informado.

O ultimo item depende de `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=${base_url}/v1`.
Nao voltar a usar a variavel generica que fazia o smoke cair no default
`127.0.0.1:3000` e testar acidentalmente o Podman.

## Evidencia do cutover

O handoff final usou um freeze curto do Router Podman, seguido de dump e restore
com `ON_ERROR_STOP`. As contagens fonte e destino convergiram:

```text
channels=4
models=11
abilities=9
tokens=13
users=11
logs=88677
```

Artefatos locais protegidos:

- pre-cutover: `backups/k3s-router-20260719T195842Z`;
- handoff final: `backups/k3s-router-cutover-20260719T200250Z`;
- backup pos-cutover: `backups/k3s-router-20260719T201321Z`;
- vhosts: `/var/backups/router-ai-atius/`.

O catalogo publico validado publica Sol/Terra/Luna com contexto OAuth `272000`,
output oficial fallback `128000` e precos Standard `5/30`, `2.5/15`, `1/6`.
`/v1/responses` nao streaming e embeddings tambem passaram no endpoint publico.
O profile k3s consulta `10.21.1.21:3115/health` e `/metrics`; `te_queue_size>0`
e tratado como saturacao de capacidade e bloqueia scale-up do governor.

## Rollback

1. Restaurar os dois vhosts a partir de `/var/backups/router-ai-atius/`.
2. Rodar `sudo apache2ctl configtest`.
3. Recarregar Apache.
4. Iniciar `systemctl --user start container-router-ai-atius.service`.
5. Esperar `http://127.0.0.1:3000/api/status` retornar `200`.
6. Rodar o smoke publico.

Nao apontar Apache ao Podman enquanto o unit estiver inativo. Nao apagar o
StatefulSet/PVC durante rollback; primeiro estabilizar o edge, depois decidir a
fonte de dados e reconciliar escritas.

## Registro detalhado

O incidente, restore, release e provas HTTP estao em
`docs/K3S-CUTOVER-2026-07-19.md`.
