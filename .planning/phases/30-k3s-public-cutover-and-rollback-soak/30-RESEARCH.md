# Phase 30: k3s public cutover and rollback soak - Research

**Pesquisado em:** 2026-07-13
**Domínio:** cutover reversível Apache/PgBouncer para Services ClusterIP k3s, smoke público, soak e aposentadoria separada de Podman router/Redis e backend PostgreSQL 17 host
**Confiança:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** O Apache deve trocar apenas os upstreams do router atualmente em
  `127.0.0.1:3000`; rotas de docs/assets em `127.0.0.1:3003` permanecem intactas.
- **D-02:** O destino sera o `ClusterIP` persistente do Service validado na Phase 29. O IP
  exato e o checksum da configuracao devem entrar na evidencia de cutover.
- **D-03:** Antes da troca, criar backups do vhost, da fonte
  `DBRouterAiAtius` no PostgreSQL 17 host, dos metadados do cluster PG17 host,
  do estado k3s, do estado Podman e das configuracoes auxiliares afetadas.
- **D-04:** Smoke publico deve cobrir health, catalogo de modelos e chamadas autenticadas
  non-stream/stream nos contratos relevantes, distinguindo falha interna de
  falha do upstream.
- **D-05:** O soak deve ter checks periodicos, criterio objetivo de rollback e registro de
  disponibilidade, Pods, restarts, eventos, armazenamento e resposta publica.
- **D-06:** Qualquer gate critico falho reverte Apache para o router Podman e
  PgBouncer para o PostgreSQL 17 host em `127.0.0.1:8745`, seguido de smoke.
- **D-07:** Retirement trata separadamente router/Redis Podman e PG17 host;
  a auditoria live provou dependencias de ATS, Horistic, GBrain e Omni Fleet,
  portanto `postgresql@17-main` permanece active/enabled nesta fase.
  Nenhum dado, dump, imagem, volume ou unit é apagado; retenção mínima de 7 dias.
- **D-08:** Nao implementar nem configurar Headroom nesta fase.
- **D-09:** CPU total <=20%, Pods normais com requests/limits 500m e segredos
  exclusivamente do Vault.
- **D-10:** A fonte/rollback DB é o PostgreSQL 17 host com 34 tabelas; a database
  homônima no container tem 0 tabelas e é inelegível.

### the agent's Discretion

Não há seção explícita `## the agent's Discretion` no `30-CONTEXT.md`.

### Deferred Ideas (OUT OF SCOPE)

Não há seção explícita `## Deferred Ideas` no `30-CONTEXT.md`.

### Restrições diretas adicionais do usuário

- Retarget somente dos upstreams `127.0.0.1:3000` do Apache para Service
  ClusterIP; preservar docs `:3003`; manter backup/checksum/configtest/reload,
  smoke público autenticado, soak objetivo, segredos Vault, CPU <=20%, Pods
  500m e Headroom fora do escopo.
- Correcao live vinculante de 2026-07-13: o PgBouncer `10.11.1.11:6432`
  aponta `DBRouterAiAtius` para o PostgreSQL 17 host em `127.0.0.1:8745`,
  cluster `/var/lib/postgresql/17/main`, unit `postgresql@17-main`, com 34
  tabelas. A database homonima no container PostgreSQL tem 0 tabelas e nao pode
  ser usada como fonte ou rollback.
</user_constraints>

## Summary

A Phase 30 deve ser planejada como duas mudanças pequenas, ordenadas e reversíveis: (1) trocar somente a entrada `DBRouterAiAtius` do PgBouncer host do PostgreSQL 17 host em `127.0.0.1:8745` para o `ClusterIP:5432` do Service PostgreSQL 17 validado na Phase 29; (2) trocar no vhost enabled somente as 16 diretivas que contêm `127.0.0.1:3000` para o `ClusterIP:3000` do Service router. O PgBouncer está ativo, escuta em `6432` e a entrada atual foi verificada sem revelar credenciais. A fonte host possui 34 tabelas; a database homônima no container PostgreSQL possui 0 e fica explicitamente excluída da cadeia de dados. O vhost live contém 38 diretivas de docs em `127.0.0.1:3003`, que devem permanecer byte-for-byte inalteradas. [VERIFIED: correção live obrigatória de 2026-07-13 + inspeção read-only de `/etc/pgbouncer/pgbouncer.ini`, systemd e `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf`]

O cutover deve falhar fechado se a Phase 29 não entregar `GO` com ClusterIPs estáveis, EndpointSlices prontos, restore validado, imagem imutável, PVs `Retain`, smoke shadow completo e backend k3s do CLIAnything. O `tools/clianything.py` desta worktree aceita apenas `CLIANYTHING_DB_BACKEND=host|podman`; portanto o planner não pode declarar operação k3s pronta sem a extensão/validação entregue pela Phase 29. [VERIFIED: `30-CONTEXT.md`, artefatos Phase 29 e `tools/clianything.py:28-34,166-176`]

O soak não pode ser “observar e ver”: deve registrar baseline imediatamente antes do cutover, durar no mínimo 30 minutos, amostrar a cada 60 segundos, executar matriz sintética completa a cada 5 minutos e reverter por qualquer falha crítica definida abaixo. O gate bloqueante exige >=30 amostras e >=6 matrizes; 24 horas permanece como monitoramento pós-operação recomendado. [RESOLVED: autorização do usuário 2026-07-13]

**Recomendação primária resolvida:** executar `preflight da fonte host PG17 → backup/checksums → PgBouncer repoint/reload/DB smoke → Apache retarget/configtest/graceful reload/public smoke → soak objetivo de 30 minutos → retirement autônomo após PASS`. Qualquer falha restaura primeiro Apache para router Podman e depois PgBouncer para o PostgreSQL 17 host, com smoke de rollback completo. Após PASS, retirar router/Redis Podman e `DBRouterAiAtius` como source antigo, mas manter `postgresql@17-main` active/enabled: a auditoria live encontrou mappings PgBouncer para cinco databases e conexoes diretas de ATS/Horistic, alem de GBrain e Omni Fleet. [VERIFIED: auditoria live 2026-07-13 + autorização do usuário + configuração live] [CITED: https://httpd.apache.org/docs/2.4/stopping.html] [CITED: https://www.pgbouncer.org/usage.html]

## Architectural Responsibility Map

| Capacidade | Tier primário | Tier secundário | Rationale |
|---|---|---|---|
| Seleção do backend público | Apache host edge | Service router k3s | Apache mantém TLS/paths e troca somente upstreams `:3000`. [VERIFIED: vhost live] |
| Seleção do PostgreSQL | PgBouncer host | Service PostgreSQL k3s | Uma única entrada lógica muda; clientes continuam em `127.0.0.1:6432`. [VERIFIED: config live] |
| Execução da aplicação | API/Backend k3s | Database/Redis k3s | Router, Postgres e Redis já devem estar aprovados pela Phase 29. [VERIFIED: `30-CONTEXT.md`] |
| Smoke público | API pública | Providers upstream | Deve separar erro local/edge/DB de erro de quota/auth/provider upstream. [VERIFIED: AGENTS.md e scripts de smoke] |
| Soak | Operação k3s | Apache/PgBouncer | Observa Pods, restarts, eventos, storage e contratos públicos. [VERIFIED: `30-CONTEXT.md`] |
| Rollback | Apache + PgBouncer host | router Podman + PostgreSQL 17 host | Restaura Apache ao router Podman e `DBRouterAiAtius` a `127.0.0.1:8745`; a database vazia do container não participa. [VERIFIED: correção live] |
| Aposentadoria Podman | user-systemd/Podman | armazenamento de rollback | Router/Redis param; imagens, volumes, dumps e unit files permanecem. [VERIFIED: constraints] |
| Preservação DB host | systemd/PgBouncer/PostgreSQL 17 | inventário de dependências | Mantém `postgresql@17-main` active/enabled porque ATS, Horistic, GBrain, Omni Fleet e outras databases compartilham o cluster; nunca apaga dados. [VERIFIED: auditoria live 2026-07-13] |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|---|---|---|
| PHASE-22-CUTOVER-ROLLBACK | Cutover público e rollback testável | Ordem PgBouncer→Apache, backups, checksums, reloads, smokes e rollback inverso. [VERIFIED: ROADMAP + pesquisa] |
| PHASE-20-GO-ONLY-V1-MODELS | `/v1/models` permanece Go-owned | Inventário das regras `/v1/models`, shape `{data}` e campos proibidos. [VERIFIED: vhost + AGENTS.md] |
| PHASE-25-CLIENT-SMOKE-VALIDATION | Validar clientes/contratos reais | Matriz non-stream, stream e Responses com classificação de falhas. [VERIFIED: ROADMAP + scripts existentes] |
</phase_requirements>

## Project Constraints (from AGENTS.md)

- Tarefas pesadas não podem exceder 20% da CPU total; usar `scripts/podman-admin.sh profile-run` quando aplicável. [VERIFIED: `AGENTS.md`]
- Pods normais k3s devem ter `requests.cpu=limits.cpu=500m`; múltiplos containers dividem o teto total de `500m`. [VERIFIED: `AGENTS.md`]
- Preservar runtime full-Go, catálogo `/v1/models` com root `{"data":[...]}`, ordenação protegida e sem campos internos de pricing. [VERIFIED: `AGENTS.md`]
- Não reintroduzir `model-detailed`, sidecar ou container extra no caminho `/v1/`. [VERIFIED: `AGENTS.md`]
- Vault é a fonte de segredos; valores não entram em docs, logs, chat, evidência ou commits. [VERIFIED: `AGENTS.md`]
- Não remover ou renomear identificadores protegidos de new-api/QuantumNous. [VERIFIED: `AGENTS.md`]
- Graphify está presente e não stale no worktree; a consulta específica da Phase 30 retornou zero nós, portanto o plano usa os artefatos de fase e a correção live como fontes primárias. [VERIFIED: `graphify status/query` em 2026-07-13]
- O Obsidian foi consultado read-only; não foi escrito porque o usuário restringiu mutações ao `30-RESEARCH.md`. [VERIFIED: restrição direta]

## Standard Stack

### Core

| Componente | Versão/estado | Finalidade | Por que usar |
|---|---|---|---|
| k3s | `v1.35.5+k3s1` | Services, Pods, EndpointSlices e storage | Runtime alvo já instalado. [VERIFIED: host CLI] |
| Apache HTTP Server | `2.4.58` | TLS e reverse proxy público | Edge atual; `configtest` e graceful reload são nativos. [VERIFIED: host CLI] [CITED: https://httpd.apache.org/docs/2.4/programs/apachectl.html] |
| PgBouncer | `1.25.2` | Endpoint DB estável em `127.0.0.1:6432` | Permite trocar só o backend de `DBRouterAiAtius`. [VERIFIED: host CLI/config] [CITED: https://www.pgbouncer.org/usage.html] |
| Podman | `4.9.3` | rollback de router/Redis durante soak | Runtime de aplicação/cache deve permanecer disponível até aprovação. [VERIFIED: host CLI + CONTEXT] |
| systemd | `255` | lifecycle Apache/PgBouncer/k3s/PG17 e user units Podman | Mecanismo instalado de operação. [VERIFIED: host CLI] |

### Supporting

| Ferramenta | Versão | Finalidade | Quando usar |
|---|---|---|---|
| curl | `8.5.0` | status/latência/stream HTTP | smokes e sampling. [VERIFIED: host CLI] |
| Python | `3.12.3` | assertions JSON/SSE | scripts de smoke existentes/novos. [VERIFIED: host CLI] |
| jq | `1.7` | leitura de estado k3s não sensível | evidence sanitizada. [VERIFIED: host CLI] |
| CLIAnything | repo local | API/DB/provider checks | somente após backend k3s da Phase 29 passar. [VERIFIED: `tools/clianything.py`] |

**Instalação:** nenhuma dependência externa deve ser instalada nesta fase. [VERIFIED: environment audit]

## Package Legitimacy Audit

Não aplicável: a fase não instala packages npm/PyPI/crates. [VERIFIED: escopo]

## Architecture Patterns

### System Architecture Diagram

```text
[GO Phase 29 + ClusterIPs + checksums]
                 |
                 v
       [backup/checksum baseline]
                 |
                 v
PgBouncer DBRouterAiAtius: host PG17 127.0.0.1:8745 -> POSTGRES_CLUSTER_IP:5432
                 | config validation + RELOAD + WAIT_CLOSE + DB smoke
                 | falha -> restore PgBouncer backup -> RELOAD -> DB smoke host PG17
                 v
Apache router rules only: 127.0.0.1:3000 -> ROUTER_CLUSTER_IP:3000
                 | configtest + graceful reload + public smoke matrix
                 | falha -> restore vhost -> configtest/reload -> public router Podman smoke
                 v
       [30m soak; sample 60s; matriz 5m] [RESOLVED]
                 | crítico -> rollback Apache para router Podman -> rollback PgBouncer para host PG17
                 v
      [PASS checksummed; retirement autônomo]
                 |
                 v
 stop/disable Podman router/Redis -> remove containers/pod allowlisted only
 inventory all host PG17 DBs + PgBouncer mappings + workloads
 preserve postgresql@17-main active/enabled (shared cluster; KEEP_SERVICE only)
 preserve data dir + databases + images + volumes + dumps + checksums + units
```

### Recommended Project Structure

```text
scripts/
├── k3s-router-cutover.sh          # novo; guarded, duas etapas, evidence
├── k3s-router-public-smoke.sh     # novo; matriz pública completa
├── k3s-router-soak.sh             # novo; sampling e critérios
├── k3s-router-rollback.sh         # novo; explícito, ordem inversa
├── k3s-router-podman-retire.sh    # novo; router/Redis + preservação
└── k3s-router-host-pg-preserve.sh # novo; inventário e prova KEEP_SERVICE PG17
docs/
├── K3S-MIGRATION.md
└── PODMAN.md
```

### Pattern 1: Mutation by exact old target

**What:** scripts devem exigir contagem exata antes de editar: 16 linhas `127.0.0.1:3000` no vhost enabled e uma entrada `DBRouterAiAtius` com host/port esperados. Qualquer divergência aborta sem alteração. As 38 linhas `127.0.0.1:3003` recebem checksum/inventário pré e pós e devem permanecer idênticas. [VERIFIED: vhost/config live]

**When to use:** em cutover e rollback; evita replacement amplo e configuração mista. [VERIFIED: constraints]

```bash
# Source: padrão prescritivo derivado do inventário live
test "$(grep -c '127\.0\.0\.1:3000' "$VHOST")" -eq 16
test "$(grep -c '127\.0\.0\.1:3003' "$VHOST")" -eq 38
```

### Pattern 2: Atomic file replacement + checksum

**What:** copiar arquivo original para diretório timestamped `0700`, gerar candidato no mesmo filesystem, preservar owner/mode, validar candidato, gravar `sha256sum`, então substituir com `install`/`mv` atômico. Não editar in-place sem candidato. [ASSUMED]

**When to use:** vhost e `pgbouncer.ini`. Para PgBouncer, comparar representação sanitizada de todas as entradas antes/depois e permitir diferença somente em host/port de `DBRouterAiAtius`. [VERIFIED: user constraint]

### Pattern 3: PgBouncer reload-aware cutover

**What:** backup/checksum → validar candidato com processo de preflight que não abra listeners conflitantes → substituir config → `RELOAD` → `WAIT_CLOSE` → `SHOW DATABASES` sanitizado → conexão/query read-only via `127.0.0.1:6432`. `RELOAD` faz novas conexões usarem parâmetros novos e fecha conexões antigas quando liberadas; `WAIT_CLOSE` confirma ativação. [CITED: https://www.pgbouncer.org/usage.html]

**When to use:** antes do Apache cutover e antes de retirar `DBRouterAiAtius` como source no PostgreSQL 17 host. [VERIFIED: correção live]

### Pattern 4: Apache configtest then graceful reload

**What:** validar o candidato e a configuração instalada, executar reload graceful, depois testar público. `apachectl configtest` verifica sintaxe; graceful restart também verifica sintaxe e aborta se houver erro. `ProxyPassReverse` precisa acompanhar cada `ProxyPass`/Rewrite proxy para redirects não escaparem do edge. [CITED: https://httpd.apache.org/docs/2.4/configuring.html] [CITED: https://httpd.apache.org/docs/2.4/stopping.html] [CITED: https://httpd.apache.org/docs/2.4/mod/mod_proxy.html]

### Pattern 5: Same smoke against shadow, public and rollback

**What:** uma matriz parametrizada por base URL deve validar health, models auth/unauth, chat non-stream, chat stream SSE e Responses; o mesmo script roda no ClusterIP antes, no público depois e no router Podman após rollback, sempre com `DBRouterAiAtius` novamente no PostgreSQL 17 host. [VERIFIED: constraints + correção live + scripts existentes]

**Classificação obrigatória:** HTTP 502/503/504, falha de conexão ao ClusterIP, DB indisponível, payload/SSE inválido ou 5xx local são críticos; 401/403 inesperado é auth local; 429/insufficient_quota/erro provider com evidência de dispatch correto é upstream e deve ser registrado separadamente, não mascarado como sucesso funcional. [VERIFIED: AGENTS.md e histórico de smokes]

### Anti-Patterns to Avoid

- **Trocar `3003`:** quebra docs/assets e viola decisão locked. [VERIFIED: CONTEXT + vhost]
- **Cutover Apache antes do PgBouncer:** router k3s pode continuar dependendo da fonte PostgreSQL 17 host e produzir uma promoção parcial. [VERIFIED: correção live]
- **Usar a database vazia do container:** cria perda lógica imediata porque ela possui 0 tabelas; preflight, dump, cutover e rollback devem provar a fonte host com 34 tabelas. [VERIFIED: correção live]
- **Replacement global `127.0.0.1`:** altera docs, aliases e outros serviços. [VERIFIED: vhost]
- **Usar NodePort/Ingress/hostPort:** Phase 29 auditou alcance host→ClusterIP e a decisão final exige ClusterIP. [VERIFIED: `phase29-diskpressure-audit.md` + `30-CONTEXT.md`]
- **Aceitar smoke sem token:** o script atual sai 0 quando token falta; isso nunca pode aprovar cutover. [VERIFIED: `scripts/k3s-router-smoke.sh`]
- **Parar/remover Podman durante soak:** elimina rollback imediato. [VERIFIED: CONTEXT]
- **`podman system prune` ou remover volumes:** há precedente documentado de perda e o usuário exige preservação. [VERIFIED: `docs/PODMAN.md`]
- **Implementar Headroom:** explicitamente fora do escopo. [VERIFIED: CONTEXT]

## Don't Hand-Roll

| Problema | Não construir | Usar | Por quê |
|---|---|---|---|
| Syntax gate Apache | parser próprio | `apachectl configtest` | Parser oficial da configuração carregada. [CITED: https://httpd.apache.org/docs/2.4/programs/apachectl.html] |
| Drain de conexões PgBouncer | sleeps cegos | `RELOAD; WAIT_CLOSE; SHOW DATABASES;` | Semântica administrativa oficial. [CITED: https://www.pgbouncer.org/usage.html] |
| Descoberta de backend Service | IP de Pod | ClusterIP + EndpointSlice | ClusterIP é estável; EndpointSlice prova endpoints. [CITED: https://kubernetes.io/docs/concepts/services-networking/service/] |
| Lifecycle de unit | apagar unit file | `systemctl --user stop/disable` | Preserva caminho de re-enable/start. [CITED: https://github.com/systemd/systemd/blob/main/docs/FAQ.md] |
| Assertions JSON/SSE | grep de body | scripts Python existentes/parametrizados | Valida contrato, não apenas HTTP 200. [VERIFIED: scripts repo] |

**Key insight:** rollback só é real se Apache voltar ao router Podman e PgBouncer voltar à fonte PostgreSQL 17 host com 34 tabelas; preservar o container de database vazio não oferece rollback de dados. [VERIFIED: síntese da arquitetura corrigida]

## Runtime State Inventory

| Categoria | Items Found | Action Required |
|---|---|---|
| Stored data | PostgreSQL 17 host em `127.0.0.1:8745`, cluster `/var/lib/postgresql/17/main`, unit `postgresql@17-main`, contém a fonte `DBRouterAiAtius` com 34 tabelas; PostgreSQL k3s deve conter restore aprovado; a database no container tem 0 tabelas. [VERIFIED: correção live + Phase 29 gate] | Dump fresco/checksummed exclusivamente da fonte host; provar contagens; preservar data dir, database, dumps e PV `Retain`; nunca usar a database vazia. |
| Live service config | Vhost enabled com 16 diretivas router `:3000` e 38 docs `:3003`; PgBouncer tem entrada `DBRouterAiAtius`; Services k3s fornecem ClusterIPs. [VERIFIED: live read-only] | Backups/checksums; mudar apenas router e uma entrada DB; registrar ClusterIPs exatos. |
| OS-registered state | Apache/PgBouncer/k3s e `postgresql@17-main` são system units; router/Redis Podman usam user units, incluindo `container-router-ai-atius.service`. [VERIFIED: repo/live + correção live] | Não desabilitar até aprovação; tratar Podman e PG17 host separadamente; preservar todos os unit files. |
| Secrets/env vars | Tokens e DB auth vêm do Vault/Secrets; não foram lidos/impressos nesta pesquisa. [VERIFIED: AGENTS.md] | Carregar em processo; nunca `set -x`; evidence só com status/metadados sanitizados. |
| Build artifacts | Imagem k3s imutável deve vir da Phase 29; imagens/containers/volumes Podman compõem rollback de router/Redis; data dir/database/dumps PG17 compõem rollback DB. [VERIFIED: CONTEXT] | Registrar digest/checksum; remover containers apenas após aprovação; não remover imagens, volumes, data dir, database ou dumps. |

## Common Pitfalls

### Pitfall 1: Vhost available e enabled divergirem

**What goes wrong:** backup/edição do arquivo errado não muda o runtime ou perde correções live. **Why:** os arquivos available/enabled já divergem em rotas docs. **Avoid:** resolver `readlink -f`/inode do enabled, guardar ambos e editar somente o arquivo efetivamente carregado. **Warning:** checksums diferentes ou `apachectl -S` aponta outro caminho. [VERIFIED: inspeção live]

### Pitfall 2: Reload PgBouncer parecer instantâneo, mas conexões antigas persistirem

**What goes wrong:** parte do tráfego continua no PostgreSQL 17 host. **Why:** conexões existentes são fechadas ao serem liberadas. **Avoid:** `WAIT_CLOSE`, `SHOW DATABASES`, query nova e verificar conexões/porta `8745` antes de retirar `DBRouterAiAtius` como source. [CITED: https://www.pgbouncer.org/usage.html]

### Pitfall 3: ClusterIP existir sem endpoints prontos

**What goes wrong:** Apache/PgBouncer recebe timeout/refused. **Avoid:** validar EndpointSlice IP/port/ready, Pods Ready e conexão host→ClusterIP imediatamente antes de cada troca. [CITED: https://kubernetes.io/docs/tasks/debug/debug-application/debug-service/]

### Pitfall 4: Stream “200” sem stream válido

**What goes wrong:** proxy bufferiza/trunca SSE ou resposta termina sem evento final. **Avoid:** exigir múltiplos eventos/deltas, terminal válido, ausência de HTML e timeout máximo. [ASSUMED]

### Pitfall 5: Soak sem baseline

**What goes wrong:** restart/evento/latência preexistente é atribuído ao cutover ou regressão real é ignorada. **Avoid:** snapshot T-0 de restartCount, pod UID, resourceVersion, PV usage, events, latência e status público; comparar deltas. [VERIFIED: Kubernetes observability primitives] [ASSUMED: thresholds]

## Code Examples

### Inventário seguro do Service e EndpointSlice

```bash
# Source: Kubernetes official docs
sudo -n k3s kubectl -n router-ai-atius get svc router-ai-atius router-ai-atius-postgres -o wide
sudo -n k3s kubectl -n router-ai-atius get endpointslice \
  -l kubernetes.io/service-name=router-ai-atius -o wide
```

[CITED: https://kubernetes.io/docs/tasks/debug/debug-application/debug-service/]

### Gate Apache

```bash
# Source: Apache HTTP Server official docs
sudo -n apachectl configtest
sudo -n systemctl reload apache2
sudo -n systemctl is-active apache2
```

[CITED: https://httpd.apache.org/docs/2.4/programs/apachectl.html] [CITED: https://httpd.apache.org/docs/2.4/stopping.html]

### Gate PgBouncer sem credenciais em evidence

```sql
-- Source: PgBouncer official docs; executar pela console administrativa segura
RELOAD;
WAIT_CLOSE;
SHOW DATABASES;
```

Persistir somente nome, host, port e status necessários, com qualquer campo sensível redigido. [CITED: https://www.pgbouncer.org/usage.html] [VERIFIED: política AGENTS.md]

### Matriz pública mínima

```bash
: "${ATIUS_ROUTER_TOKEN:?required}"
ATIUS_ROUTER_BASE_URL=https://router.atius.com.br python3 scripts/smoke-openai-sdk.py
ATIUS_ROUTER_BASE_URL=https://router.atius.com.br ATIUS_ROUTER_STREAM=1 python3 scripts/smoke-openai-sdk.py
ATIUS_ROUTER_BASE_URL=https://router.atius.com.br python3 scripts/smoke-provider-consolidation.py
```

Os scripts precisam ser parametrizados/validados para cobrir explicitamente `/v1/responses`; não assumir cobertura pelo nome. [VERIFIED: scripts existentes + requirement]

## Soak Acceptance Contract

| Sinal | Sampling | Aprovação | Rollback imediato |
|---|---|---|---|
| Público health/models | 5 min | 100% dos checks 2xx/401 esperados | 2 falhas consecutivas ou 1 falha >5 min [ASSUMED] |
| Chat non-stream/stream/Responses | 15 min | 100% contrato local válido; upstream errors classificados | qualquer 5xx local, payload/SSE inválido ou auth regression [ASSUMED] |
| Pods | 5 min | Ready estável, restart delta 0 | CrashLoopBackOff, OOMKilled, eviction ou restart delta >0 [ASSUMED] |
| Node | 5 min | Ready=True, DiskPressure=False, taint ausente | DiskPressure=True/taint/NotReady [VERIFIED: Phase 29 gate] |
| Storage | 15 min | PVC Bound, PV Retain, uso sem crescimento anômalo | PVC/PV não Bound, read-only/fs error, livre <20% [ASSUMED] |
| Eventos | 5 min | sem Warning novo relevante | FailedMount, FailedScheduling, Evicted, Unhealthy recorrente [ASSUMED] |
| PgBouncer/DB | 5 min | nova conexão/query via 6432; backend k3s | query falha, backend volta ao host `8745` sem rollback declarado, pool errors [ASSUMED] |
| Duração | 60s/300s | mínimo bloqueante de 30 minutos, >=30 amostras e >=6 matrizes; retirement autônomo após PASS | qualquer gate crítico [RESOLVED: autorização do usuário 2026-07-13] |

## State of the Art

| Abordagem antiga | Abordagem atual | Quando | Impacto |
|---|---|---|---|
| NodePort proposto em `29-RESEARCH.md` | ClusterIP host-reachable | auditoria 2026-07-12 | Não abrir porta externa; respeitar decisão Phase 30. [VERIFIED: debug audit + CONTEXT] |
| Apache como única troca | PgBouncer primeiro, Apache depois | evidência corrigida 2026-07-13 | Permite retirar a fonte host PG17 sem dependência oculta do router. [VERIFIED: correção live + live config] |
| Checklist manual informativo | scripts guarded com evidence e rollback | Phase 30 recommendation | Torna gates reproduzíveis e fail-closed. [ASSUMED] |

**Deprecated/outdated:** a recomendação NodePort da Phase 29 research/pattern map está superada pela auditoria live que provou alcance host→ClusterIP e pela decisão vinculante da Phase 30. [VERIFIED: `phase29-diskpressure-audit.md` + `30-CONTEXT.md`]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | RESOLVED — soak bloqueante de 30 minutos, amostra a cada 60s e matriz a cada 300s; 24h fica como monitoramento prolongado pós-operação. | Summary/Soak | Resolvido por autorização explícita do usuário em 2026-07-13. |
| A2 | Atomic replacement via candidato no mesmo filesystem é o padrão de implementação. | Pattern 2 | Médio; permissões/tooling host podem exigir variante. |
| A3 | Thresholds de restart/latência/erros propostos são aceitáveis. | Soak | Alto; baseline real pode exigir ajuste. |
| A4 | Validação offline/candidato do PgBouncer pode ser feita sem listener conflitante. | Pattern 3 | Médio; comando exato deve ser provado no plano/execução. |
| A5 | SSE deve exigir múltiplos eventos e terminal válido. | Pitfall 4 | Baixo; contrato exato depende do modelo/provider escolhido. |

## Open Questions — RESOLVED

1. **RESOLVED — janela de soak e retenção dos artefatos.**
   - Decisão: soak bloqueante de 30 minutos, com pelo menos 30 amostras de 60 segundos e 6 matrizes completas de 5 minutos. [VERIFIED: autorização do usuário 2026-07-13]
   - Decisão: imagens, volumes, dumps, checksums e unit files de rollback permanecem preservados por no mínimo 7 dias; não há descarte automático. [VERIFIED: autorização do usuário 2026-07-13]
   - Decisão: monitoramento de 24h é pós-operação recomendado e documentado, não gate bloqueante do retirement. [VERIFIED: autorização do usuário 2026-07-13]

2. **RESOLVED — qual modelo ativo deve representar cada smoke pago?**
   - What we know: non-stream, stream e Responses são obrigatórios; provider errors precisam ser classificados. [VERIFIED: user constraint]
   - Decisão: Phase 29 GO registra os modelos aprovados e Phase 30 consome essa lista sem hard-code temporal; ausência da lista é NO-GO. [VERIFIED: fail-closed dependency]

3. **RESOLVED — quais backends antigos serão aposentados?**
   - Router/Redis Podman: gerar inventário T-0 live, derivar allowlist explícita e exigir igualdade antes de stop/disable/remove; nunca usar wildcard destrutivo. [VERIFIED: autorização de retirement autônomo]
   - PostgreSQL 17 host: a auditoria live confirmou ATS, Horistic, GBrain, Omni Fleet, cinco mappings PgBouncer e outras databases no cluster. Manter `postgresql@17-main` active/enabled, retirar somente `DBRouterAiAtius` como source e proibir stop/disable nesta fase. [VERIFIED: auditoria live 2026-07-13]
   - Em ambos os ramos, nunca apagar `/var/lib/postgresql/17/main`, databases, dumps, imagens, volumes ou unit files; retenção mínima de 7 dias. [VERIFIED: correção live]
   - Após PASS checksummed do soak, o agente executa retirement autonomamente até o fim, sem checkpoint humano. [VERIFIED: autorização do usuário 2026-07-13]

## Environment Availability

| Dependência | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| k3s/kubectl | Services/soak | ✓ | 1.35.5+k3s1 | — |
| Apache | edge | ✓ | 2.4.58 | restore vhost |
| PgBouncer | DB switch | ✓ | 1.25.2 | restore config/reload |
| Podman router/Redis | rollback da aplicação/cache | ✓ | 4.9.3 | artifacts preservados |
| PostgreSQL 17 host | fonte e rollback DB | ✓ | 17 | `postgresql@17-main`, `127.0.0.1:8745` |
| systemd | lifecycle | ✓ | 255 | — |
| curl/Python/jq | smoke/evidence | ✓ | 8.5.0/3.12.3/1.7 | — |
| CLIAnything k3s backend | operação DB | gate Phase 29 | não existe nesta worktree (`host|podman` only) | bloquear Phase 30 |

[VERIFIED: environment audit + code]

**Missing dependencies with no fallback:** artifact `GO` da Phase 29 e backend k3s do CLIAnything validado. [VERIFIED: CONTEXT]

**Missing dependencies with fallback:** nenhuma; NodePort/Ingress não são fallback autorizado. [VERIFIED: locked decision]

## Validation Architecture

### Test Framework

| Property | Value |
|---|---|
| Framework | Bash gates + Python HTTP contract tests + kubectl/systemd checks [VERIFIED: repo] |
| Config file | nenhum dedicado; scripts `k3s-router-*` [VERIFIED: repo] |
| Quick run command | `bash -n scripts/k3s-router-*.sh && python3 -m py_compile tools/clianything.py scripts/smoke-*.py` [VERIFIED: tooling] |
| Full suite command | preflight Phase 29 GO → dry-run/candidate validation → shadow smoke → DB switch → public cutover → soak sampler [ASSUMED] |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| PHASE-22-CUTOVER-ROLLBACK | PgBouncer/Apache reversíveis | integration/live | `scripts/k3s-router-cutover.sh` + `scripts/k3s-router-rollback.sh` | ❌ Wave 0 |
| PHASE-20-GO-ONLY-V1-MODELS | shape/auth/model fields | HTTP contract | public smoke models | ⚠️ parcial em `k3s-router-smoke.sh` |
| PHASE-25-CLIENT-SMOKE-VALIDATION | non-stream/stream/Responses | integration | `scripts/k3s-router-public-smoke.sh` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** syntax/static tests e fixture de configuração; nenhuma mutação live. [ASSUMED]
- **Per wave merge:** preflight/candidate validation read-only. [ASSUMED]
- **Phase gate:** public smoke completo + soak checksummed PASS de 30 minutos + rollback evidence; retirement autônomo em seguida, sem checkpoint humano. [RESOLVED: autorização do usuário 2026-07-13]

### Wave 0 Gaps

- [ ] `scripts/k3s-router-cutover.sh` — duas etapas guarded e evidence sanitizada.
- [ ] `scripts/k3s-router-public-smoke.sh` — health/models/non-stream/stream/Responses.
- [ ] `scripts/k3s-router-soak.sh` — baseline/deltas/critérios.
- [ ] `scripts/k3s-router-rollback.sh` — Apache depois PgBouncer, smoke completo.
- [ ] `scripts/k3s-router-podman-retire.sh` — retirada allowlisted de router/Redis Podman.
- [ ] `scripts/k3s-router-host-pg-preserve.sh` — inventário de dependências e prova `KEEP_SERVICE` do PostgreSQL 17 host.
- [ ] testes unitários com fixtures para garantir `3003` intacto e apenas uma entrada PgBouncer alterada.

[VERIFIED: repo gap analysis]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | yes | token do Vault somente em memória/processo; smoke fail-closed. [VERIFIED: AGENTS.md] |
| V3 Session Management | yes | preservar secrets/config da Phase 29; validar login/session sem logar cookie. [ASSUMED] |
| V4 Access Control | yes | `/v1/models` sem token continua 401; autenticado 200. [VERIFIED: script/contract] |
| V5 Input Validation | yes | allowlists de path/entry/line count e JSON/SSE assertions. [VERIFIED: recommended pattern] |
| V6 Cryptography | yes | TLS permanece no Apache; não alterar certificados/keys. [VERIFIED: vhost + scope] |

### Known Threat Patterns for k3s/Apache/PgBouncer

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| Segredo em logs/evidence | Information Disclosure | redaction, sem `set -x`, não persistir bodies/headers sensíveis. [VERIFIED: AGENTS.md] |
| Config swap amplo | Tampering/DoS | exact counts, allowlist, checksum, candidate, configtest. [VERIFIED: inventory] |
| Backend DB misto | Tampering/Integrity | RELOAD + WAIT_CLOSE + SHOW DATABASES sanitizado + query nova. [CITED: https://www.pgbouncer.org/usage.html] |
| Service sem endpoints | DoS | EndpointSlice Ready + host connection preflight. [CITED: https://kubernetes.io/docs/tasks/debug/debug-application/debug-service/] |
| Rollback destruído por cleanup | DoS | stop/disable only; preservar data dir, databases, imagens, volumes, dumps e unit files. [VERIFIED: user constraint] |
| Cluster PG17 compartilhado parado | DoS/Tampering | inventário completo de databases, mappings PgBouncer e workloads; manter service se houver qualquer dependência. [VERIFIED: correção live] |

## Sources

### Primary (HIGH confidence)

- `30-CONTEXT.md`, todos os artefatos Phase 29, `phase29-diskpressure-audit.md`, ROADMAP/PROJECT/STATE, scripts `k3s-router-*`, `tools/clianything.py`, docs operacionais. [VERIFIED: codebase reads]
- `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf` e `/etc/pgbouncer/pgbouncer.ini` read-only; somente metadados não sensíveis foram extraídos. [VERIFIED: live inspection 2026-07-12]
- Host CLI versions e systemd active state. [VERIFIED: live inspection 2026-07-12]

### Secondary (MEDIUM confidence)

- https://httpd.apache.org/docs/2.4/configuring.html — configtest.
- https://httpd.apache.org/docs/2.4/stopping.html — graceful restart.
- https://httpd.apache.org/docs/2.4/mod/mod_proxy.html — ProxyPassReverse.
- https://kubernetes.io/docs/concepts/services-networking/service/ — ClusterIP.
- https://kubernetes.io/docs/tasks/debug/debug-application/debug-service/ — EndpointSlices.
- https://www.pgbouncer.org/usage.html — RELOAD, WAIT_CLOSE, SHOW DATABASES.
- https://github.com/systemd/systemd/blob/main/docs/FAQ.md — enablement/start semantics.

### Tertiary (LOW confidence)

- A1 está resolvida por autorização explícita; A2–A5 continuam sujeitas à validação automatizada live, sem checkpoint humano.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versões e estado live verificados.
- Architecture: HIGH — configuração exata, dependencies e ordem de cutover confirmadas.
- Pitfalls: HIGH — derivados de gaps concretos e semântica oficial; thresholds de soak são LOW até aprovação.

**Research date:** 2026-07-13
**Valid until:** 2026-07-20 para estado live; 2026-08-12 para padrões estáveis.
