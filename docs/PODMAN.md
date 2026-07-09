# router-ai-atius — Notas de Migração Podman

Este doc captura lições + workarounds aplicados na migração
Docker → Podman do `router-ai-atius` (precedente 2026-06-04, re-baselined 2026-06-12).

Plano de migração para k3s, ainda sem cutover público executado:

- `docs/K3S-MIGRATION.md`

Podman continua sendo a fonte de verdade da produção atual até existir um
summary de cutover k3s com smoke público e rollback validados.

## Contexto

- **Servidor:** ATIUS-SRV-1 (10.1.1.1 / 137.131.190.161)
- **Runtime origem:** Docker 29.3.0
- **Runtime destino:** Podman 3.4.4 (rootless, user=ubuntu uid=1000)
- **Stack:** 4 user containers (router-ai-atius, model-detailed, redis, postgres) + 1 infra pause (k8s.gcr.io/pause:3.5)
- **Data migração inicial:** 2026-06-04
- **Re-baseline:** 2026-06-12 (compose versionado + recovery script)

## Decisões arquiteturais

### 1. Pod único vs containers soltos

Os 4 containers compartilham **network namespace** (atrás do mesmo
Apache proxy, falando entre si via `localhost:5432`/`localhost:6379`)
+ lifecycle (subir juntos, cair juntos).

**Decisão:** POD único `atius-ai-router` com `network_mode=host` (via
infra pause container do podman).

**Alternativa rejeitada:** containers soltos + custom network
(podman 3.4 tem bug em DNS inter-container que quebrou 2026-06-04
com paperclip-atius-db).

### 2. Bind mounts vs named volumes

- `/home/ubuntu/GitHub/containers/router-ai-atius/data` → `/data` (bind)
- `/home/ubuntu/GitHub/containers/router-ai-atius/logs` → `/app/logs` (bind)
- `pgdata` (named volume podman) → `/var/lib/postgresql/data`

**Decisão:** bind mounts para dados que precisam ser visíveis/backupáveis
pelo host (data, logs). Named volume para Postgres (otimização de I/O
do driver overlay/Volumes).

### 3. Imagens

| Container | Imagem origem (Docker) | Imagem destino (Podman) | Justificativa |
|-----------|------------------------|--------------------------|---------------|
| router-ai-atius | `calciumion/new-api:latest` (upstream) | `ghcr.io/giovannimnz/router-ai-atius:latest` (fork rebrand) | Fork rebranded v2.11+ (2026-06-04) |
| model-detailed | (não estava no compose original) | `localhost/router-ai-atius-model-detailed:latest` (built local) | Adicionado 2026-06-11; hotfix atual usa bind de `/home/ubuntu/GitHub/containers/router-ai-atius/runtime/model-detailed/` |
| postgres | `postgres:15` | `docker.io/library/postgres:15-alpine` | Idêntico, só normalizado |
| redis | `redis:latest` | `docker.io/library/redis:7-alpine` | Pin de major version (era `latest`, agora `7-alpine`) |

**Workaround `docker save | podman load`** foi aplicado para todas
as 4 imagens durante a migração inicial 2026-06-04.

### Deploy da imagem do fork

O caminho atual de deploy do `router-ai-atius` é GHCR -> Podman user unit:

```bash
scripts/pull-and-restart.sh latest
```

Esse script puxa `ghcr.io/giovannimnz/router-ai-atius:latest`, reinicia
`container-router-ai-atius.service` com `systemctl --user` e valida health local.
Para um tag versionado, use `scripts/pull-and-restart.sh vX.Y.Z`; o script
retaga o mesmo digest para `:latest`, que é o tag consumido pela unit.

Autocorreções do deploy:

- Se o restart falhar por storage stale do pod rootless, com erro em
  `userdata/shm`, o script recria `pod-atius-ai-router.service`, sobe Redis e
  Postgres, e tenta o router uma vez novamente.
- Se o router falhar na inicialização com `cached plan must not change result
  type` / `SQLSTATE 0A000`, o script reinicia `pgbouncer` uma vez via
  `sudo -n systemctl restart pgbouncer` e reinicia o router. Esse caminho limpa
  planos preparados velhos após migrations; o código do fork também mantém
  `PrepareStmt=false` para PostgreSQL para evitar recorrência.

## Workarounds aplicados

### 1. Network namespace compartilhado

Containers `router-ai-atius` e `model-detailed` rodam com
`net_mode: container:<infra-id>`, o que permite que os
**port bindings no pod infra** (`3000:3000`, `3001:3001`, `3300:3001`)
sejam visíveis a partir do host e compartilhados entre os containers.

Sem isso, o Apache proxy `router.atius.com.br` quebraria.

### 2. Podman-compose 1.6.0 — limitação de `pods:` block

`podman-compose` versão 1.6.0 (instalada) **suporta** `pods:` block
(traduz para `podman pod create`), MAS o parser YAML falha em alguns
casos edge-case (ver P3b em `multi-server-podman-migration` skill).

**Workaround:** se `podman-compose up -d` falhar com erro de parser,
usar `bash scripts/recreate-pod.sh` (manual `podman run` por container).

### 3. Senhas em env

Compose file tem placeholders `***` em vez de senhas reais. As
senhas ficam em:
- `podman inspect <ctr>` → campo `Env` (legível pelo root)
- **NÃO** em git, vault, logs públicos

## Lição aprendida

> **NUNCA** `docker system prune` ou `podman system prune` em prod
> sem backup verificado primeiro. Precedente: 2026-06-07 perdeu
> imagens de `model-detailed` por prune, restore parcial do GDrive
> salvou 80%. Ver `61-Incidents/2026-06-07-router-ai-atius-prune-sem-backup-previo.md`.

## Verificações pós-deploy

- [x] `podman pod inspect atius-ai-router` → State=Running, NumContainers=5
- [x] `curl http://localhost:3000/api/status` → 200
- [x] `curl http://localhost:3001/health` → 200
- [x] `curl https://router.atius.com.br/api/status` → 200
- [x] `podman logs router-ai-atius` sem erros fatais
- [x] `podman logs model-detailed-hotfix` workers=4 healthy
- [x] Postgres `pg_isready -h localhost` → accepting connections
- [x] Redis `PING` → PONG
- [x] Sem containers Docker ativos com nome `router-ai-atius*` (`docker ps | grep router-ai-atius` → vazio)

## Próximos passos

- [x] Push multi-arch de `ghcr.io/giovannimnz/router-ai-atius:<tag>` e
  `:latest` pelo workflow `docker-build.yml`
- [x] Usar `container-router-ai-atius.service` / `pod-atius-ai-router.service`
  para restart gerenciado no runtime Podman rootless
- [ ] Adicionar `omni srv1-ops podman status` ao omni-srv-admin CLI (já existe esboço)
- [ ] Vault log do dia 2026-06-12 com SHA + container IDs preservados
