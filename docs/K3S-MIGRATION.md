# Migração k3s do router-ai-atius — Phase 29

## Limite desta fase

A Phase 29 prepara e valida o runtime shadow em `atius-srv-1`. Ela não muda o
tráfego público, não recarrega Apache, não para Podman e não autoriza a
aposentadoria do runtime atual.

Enquanto a decisão final não for `go`:

- produção continua no pod Podman rootless `atius-ai-router`; uma user unit é
  validada somente quando existir;
- Apache continua enviando `/`, `/api/`, `/v1/` e `/health` ao backend Go em
  `127.0.0.1:3000`;
- a Phase 30 permanece bloqueada.

Estado live atual em `2026-07-13`:

- `shadow-apply.json` e `smoke.json` foram publicados com `PASS`;
- a decisão formal mais recente ficou `no-go`;
- o único gate vermelho é `live-stability`, porque o host está com
  `37966614528` bytes livres, porém só `18%` de espaço livre total;
- nenhum tráfego público foi movido, e Podman continua como produção e
  rollback.

Estado live adicional da Phase 30 em `2026-07-13`:

- `manifest.json` da Phase 30 foi gerado em
  `~/.local/state/router-ai-atius/phase30/run-20260713T160941Z` com status
  `READY_WITH_PHASE29_OVERRIDE`;
- a topologia validada para o cutover é:
  - host PG17 `DBRouterAiAtius`: `34` tabelas;
  - k3s PG17 `DBRouterAiAtius`: `35` tabelas;
  - Podman `postgres` `DBRouterAiAtius`: `0` tabelas;
- o primeiro teste live de cutover de PgBouncer foi revertido:
  o arquivo foi repointado para o ClusterIP k3s, mas o backend novo falhou em
  `password authentication failed for user "admin"`, então o mapping voltou
  para `127.0.0.1:8745`.
- a causa raiz foi fechada no mesmo dia: o role `admin` do k3s estava com
  segredo SCRAM diferente do host/PgBouncer. Os scripts agora sincronizam o
  SCRAM exato do host antes do repoint e a restauração da Phase 29 reaplica esse
  mesmo segredo, em vez de gerar um novo a partir da senha em texto.

O shadow usa exclusivamente Service `ClusterIP`. A auditoria do host provou
acesso à rede de Services, portanto NodePort, Ingress e `hostPort` não fazem
parte deste contrato. Qualquer um deles resulta em `no-go`.

## Segurança e contenção

- Não gravar Secret YAML, token, DSN, senha ou payload de API em evidência,
  log, Markdown ou Git.
- A fonte dos valores é o HashiCorp Vault, profile `router-ai-atius`. O fluxo
  usa somente os nomes `POSTGRES_PASSWORD`, `REDIS_PASSWORD` e
  `SESSION_SECRET`; os valores ficam em tmpfs durante o bootstrap.
- A origem canônica do dump é PostgreSQL 17 no host,
  `127.0.0.1:8745`, unit `postgresql@17-main`. PgBouncer
  `10.11.1.11:6432` é somente cross-check; PostgreSQL Podman nunca é fonte.
- Comandos pesados ou live são executados por
  `scripts/podman-admin.sh profile-run`, limitado a no máximo 800m (20% do
  host). Cada container k3s declara request e limit de CPU exatamente `500m`.
- Evidência operacional fica fora do Git, sob
  `~/.local/state/router-ai-atius/phase29/run-<UTC>`, owner atual e mode `0700`;
  cada JSON fica `0600`.

Exemplo de diretório para uma execução autorizada:

```bash
export PHASE29_EVIDENCE_ROOT="$HOME/.local/state/router-ai-atius/phase29"
export PHASE29_EVIDENCE_DIR="$PHASE29_EVIDENCE_ROOT/run-$(date -u +%Y%m%dT%H%M%SZ)"
install -d -m 0700 "$PHASE29_EVIDENCE_DIR"
```

## Ordem obrigatória 29-01 a 29-04

### 29-01 — cleanup, preflight e bootstrap

1. Executar cleanup somente com a allowlist implementada.
2. Exigir pelo menos 20 GiB recuperados e 25% livres.
3. Revalidar o estado atual por cinco minutos contínuos, com
   `DiskPressure=False`, sem taint e com pelo menos 32 GiB livres por amostra.
4. Só então aplicar o label exclusivo, criar o Secret a partir do Vault e
   importar a imagem imutável no containerd.

Artefatos consumidos pelo gate final:

- `cleanup.json`: cluster UID, bytes recuperados, percentual livre,
  `stable_seconds>=300` e `cpu.max<=800m`;
- `bootstrap.json`: cluster UID, freshness, hash dos manifests, label exclusivo,
  nomes exatos das Secret keys e digest imutável concordante.

### 29-02 — backup e restore PostgreSQL 17

O backup deve ser novo, checksummed e produzido diretamente do PostgreSQL 17
do host. O restore sobe primeiro somente PostgreSQL no k3s, prova que o target
está integralmente limpo, usa `ON_ERROR_STOP` e transação única, e confirma
`Retain` pelo UID do claim antes da importação.

Antes de aplicar `systemctl set-property --runtime`, o backup captura
`FragmentPath`, `CPUQuotaPerSecUSec` e `DropInPaths` e rejeita qualquer
`/run/systemd/.../50-CPUQuota.conf` preexistente. Assim ele nunca sobrescreve
quota runtime de outro operador. O reset usa somente
`systemctl set-property --runtime ... CPUQuota=` e exige restauração exata;
não usa `rm` em `/run/systemd` nem `systemctl revert`.

Artefatos consumidos:

- `backup.json`: origem host PostgreSQL 17, unit, cross-check PgBouncer,
  `pg_dump` normalizado `17.x`, SHA-256, quotas, inventário v2 e DDL completo
  com owners, ACLs, comments, security labels, estado `pg_database`,
  `pg_db_role_setting` e large objects;
- `restore.json`: backup SHA-256, cluster UID, target PostgreSQL 17 limpo,
  placement `atius-srv-1`, invariantes, igualdade source/backup/target do
  inventário v2 e todos os PVs em `Retain`.

Retry só é permitido depois de `no-go` explícito e arquivado pela cadeia de
restore. Não reutilizar um target parcialmente restaurado.

### 29-03 — apply shadow, CLIAnything e smoke estrito

Redis e router só podem iniciar depois do restore verde. O apply comprova
placement, CPU, imagens imutáveis, PVs `Retain`, EndpointSlices Ready e Services
`ClusterIP`. O CLIAnything deve operar pelo backend k3s sem depender de
`podman exec`.

O smoke estrito exige token apenas em memória e valida:

- health HTTP 200;
- `/v1/models` sem autenticação HTTP 401;
- `/v1/models` autenticado HTTP 200 e root apenas `data`;
- ausência de `pricing_source`, `pricing_estimated` e `pricing_version`;
- catálogo esperado;
- `embedding-gte-v1` com dimensão 768.

Artefatos consumidos:

- `shadow-apply.json`;
- `smoke.json`.

### 29-04 — rollback read-only e decisão

O agregador sempre cria uma prova rollback nova no mesmo processo, com
`run_id` e nonce próprios; evidência anterior nunca é reutilizada. O check lê,
sem mutar:

- pod e containers Podman; a user unit é opcional e, se existir, deve estar ativa;
- limites de CPU/memória;
- `http://127.0.0.1:3000/api/status`;
- `bin/clianything status --backend podman`;
- `apache2ctl configtest`, `apache2ctl -S` e SHA-256 do vhost fixo;
- seleção exata de `router.atius.com.br:443` para
  `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf`;
- bloco `<VirtualHost *:443>` efetivo desse arquivo com exatamente um
  `ProxyPass` para cada path `/`, `/api/`, `/v1/` e `/health`, todos no target
  Podman esperado, sem duplicata, target concorrente ou target k3s.

Em seguida execute o agregador:

```bash
PHASE29_EXECUTE=1 ./scripts/podman-admin.sh profile-run -- \
  scripts/k3s-router-go-no-go.sh --live \
  --evidence-dir "$PHASE29_EVIDENCE_DIR" \
  --output "$PHASE29_EVIDENCE_DIR/decision.json"
```

O agregador começa em `no-go`, calcula SHA-256 de toda a cadeia, valida
freshness, cluster UID e identidades cruzadas e repete readbacks live de:

- estabilidade do node por no mínimo cinco minutos;
- label e nomes das Secret keys;
- placement, readiness, CPU e digests dos três Pods;
- conjunto exato de controllers: StatefulSet PostgreSQL, Deployments Redis e
  router, e os dois ReplicaSets ativos, sem controller extra;
- cadeia exata Pod -> ReplicaSet -> Deployment por owner UID para Redis/router
  e Pod -> StatefulSet para PostgreSQL;
- igualdade semântica de workloads, images e storage entre `smoke.json`,
  `shadow-apply.json` e o snapshot live atual;
- conjunto exato de PVC/PV por claim UID com `Retain`;
- conjunto exato de Services/EndpointSlices `ClusterIP`, cada slice ligado ao
  mesmo pod UID/IP aprovado, sem NodePort, Ingress ou `hostPort`;
- CLIAnything k3s;
- Podman e Apache intactos.

`--verify-existing` é rejeitado. Revalidar exige novo `--live`, que recompõe a
cadeia, relê identidades live e gera outro rollback com `run_id`. JSON manual
ou copiado nunca autoriza a Phase 30.

## Interpretação da decisão

`decision.json` contém somente status sanitizado: `run_id`, timestamp UTC, commit,
manifest/image digest, cluster UID, checksums, gates, blockers e estado
read-only independente de Podman/Apache, mais o mapa exato de identidades live.

- `decision: "go"`: todos os gates passaram no mesmo run,
  `failed_gates` está vazio e `phase30_authorized` é `true`. Isso autoriza apenas
  iniciar o workflow separado da Phase 30.
- `decision: "no-go"`: `failed_gates` lista causas objetivas e
  `phase30_authorized` é `false`. Este é um resultado válido da Phase 29; não é
  justificativa para relaxar gates nem alterar o edge.

Nunca promover um JSON manualmente, copiar evidência entre clusters ou editar
checksums. Corrigir o blocker na fase proprietária, produzir uma nova cadeia e
reexecutar o gate.

## Verificação local sem live

Estes checks não acessam cluster, Podman ou Apache e não fazem deploy/build:

```bash
bash tests/phase29-k3s-router-go-no-go-selftest.sh
bash tests/phase29-k3s-router-restore-selftest.sh
bash -n scripts/k3s-router-{backup,go-no-go,rollback-check}.sh \
  tests/phase29-k3s-router-{restore,go-no-go}-selftest.sh
shellcheck -x scripts/k3s-router-{backup,go-no-go,rollback-check}.sh \
  tests/phase29-k3s-router-{restore,go-no-go}-selftest.sh
git diff --check -- scripts/k3s-router-go-no-go.sh \
  scripts/k3s-router-rollback-check.sh \
  tests/phase29-k3s-router-go-no-go-selftest.sh \
  docs/K3S-MIGRATION.md
```

Procedimentos de cutover, reload do Apache e aposentadoria do Podman pertencem
exclusivamente à Phase 30 e não são documentados nem executados aqui.
