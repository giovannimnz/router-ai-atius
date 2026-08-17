# k8s/router-ai-atius

Estado desta arvore:

- manifestos para shadow deployment e cutover planejado
- namespace dedicado `router-ai-atius`
- sem secrets reais em git
- sem ingress controller nesta fase

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
- Redis nesta trilha inicial e efemero para shadow; se producao exigir
  restauracao de estado, isso deve ser promovido para passo explicito no
  cutover.
- `/app/logs` usa `emptyDir` no shadow inicial.
- `/data` usa PVC proprio do router para manter a semantica do runtime atual.

## TEI

- TEI e dependencia externa desta fase.
- O contrato live atual fica fora do cluster, no `horistic-srv`:
  - embeddings: `http://10.21.1.21:3115`
  - reranker: `http://10.21.1.21:31216`
- O ConfigMap shadow guarda esses hints explicitamente em
  `TEI_BASE_URL` e `TEI_RERANKER_BASE_URL`.
- `tei-gte.ai-search.svc.cluster.local` nao e mais a fonte de verdade desta
  arvore; use-o apenas se a dependencia TEI for internalizada e revalidada no
  proprio k3s.
- Nao mudar recursos do TEI a partir desta arvore.
