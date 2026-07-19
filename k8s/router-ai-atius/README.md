# k8s/router-ai-atius

Estado desta arvore desde 2026-07-19:

- runtime publico implantado no k3s fixo do `atius-srv-1`
- namespace dedicado `router-ai-atius`
- sem secrets reais em git
- Apache como edge, sem ingress controller adicional
- release `ghcr.io/giovannimnz/router-ai-atius:v2.17.3`
- app, PostgreSQL e Redis limitados a `500m` por pod

## Secrets

Nunca commitar um env preenchido.

Criar secret a partir do template:

```bash
cp k8s/router-ai-atius/secret.example.env /tmp/router-ai-atius.secret.env
$EDITOR /tmp/router-ai-atius.secret.env
kubectl -n router-ai-atius create secret generic router-ai-atius-secrets \
  --from-env-file=/tmp/router-ai-atius.secret.env \
  --dry-run=client -o yaml | sudo -n k3s kubectl apply -f -
```

## Dry-run

```bash
scripts/k3s-router-validate-manifests.sh
```

## Shadow apply

```bash
RUN_K3S_ROUTER_APPLY=1 scripts/k3s-router-apply-shadow.sh
```

O script e util para recriacao/rehearsal. Em producao, valide primeiro
`DiskPressure=False`; nao adicione toleration de disk pressure.

## Restore rehearsal

Antes de qualquer cutover:

- registrar `backup directory`
- registrar `target namespace`
- registrar `restore command`
- registrar `kubectl get pods -n router-ai-atius`
- registrar verificacao de DB via `bin/clianything` ou equivalente
- registrar `shadow smoke`
- registrar `go/no-go`

## Stateful warnings

- Postgres usa `local-path` e portanto nao e HA.
- PVC do router tambem usa `local-path`.
- Os PVs ativos usam reclaim `Retain` para impedir remocao junto com o PVC.
- Redis nesta trilha inicial e efemero para shadow; se producao exigir
  restauracao de estado, isso deve ser promovido para passo explicito no
  cutover.
- `/app/logs` usa `emptyDir` no shadow inicial.
- `/data` usa PVC proprio do router para manter a semantica do runtime atual.

## TEI

- TEI e dependencia externa desta fase.
- O endpoint efetivo fica no ConfigMap como `http://10.21.1.21:3115`, pela
  faixa OCI DRG.
- Health e capacidade sao lidos sem auth em `/health` e `/metrics`.
  `te_queue_size=0` indica fila livre; qualquer valor positivo bloqueia
  scale-up como saturacao observavel, sem inferir CPU/memoria inexistentes.
- Nao mudar recursos do TEI a partir desta arvore.

## Imagem

O Deployment usa a tag de release e `imagePullPolicy: IfNotPresent`. A release
deve existir na GHCR e pode ser preimportada no namespace CRI `k8s.io` para
rollout offline; nao use `Never`, pois isso impede recuperacao apos GC local.

## Backup

`scripts/k3s-router-backup.sh` detecta o runtime pelo backend Apache. Use
`ROUTER_BACKUP_SOURCE=k3s` ou `podman` para forcar uma fonte durante rehearsal
ou rollback. Os dumps usam `umask 077` e nunca exportam Kubernetes Secrets.

## Rollback

O Router Podman permanece preservado e parado. Restaurar os vhosts salvos em
`/var/backups/router-ai-atius/`, validar Apache e iniciar
`container-router-ai-atius.service`; consultar `docs/K3S-MIGRATION.md`.
