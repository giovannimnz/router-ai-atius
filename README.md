# router-ai-atius — podman deployment (AT IUS AI Router)

Estado do deployment do `router-ai-atius` no ATIUS-SRV-1 após migração
Docker → Podman (Phase 17 do projeto omni-srv-admin).

Governanca/admin local: `omni-srv-admin`, em
`/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/`.

## Status

- **Migrado:** 2026-06-04 (precedente, recriado 2026-06-11)
- **Re-baselined:** 2026-06-12 (compose file versionado aqui)
- **Container runtime:** podman 4.9.3 (rootless, network=host via pod)
- **Pod name:** `atius-ai-router` (id: `9ffed7fe58c8239c5cbc8b3c254842f9558a680379a7c8329234888b01d349e3`)
- **Containers no pod:** 4 user + 1 infra pause = **5 containers total**
- **Docker (legado):** nenhum container router-ai-atius ativo (intencional)

## Containers

| Nome | Imagem | Função | Porta (host) |
|------|--------|--------|--------------|
| `router-ai-atius` | `ghcr.io/giovannimnz/router-ai-atius:latest` | Backend Go (new-api) | 3000 |
| `model-detailed-hotfix` | `localhost/router-ai-atius-model-detailed:latest` | FastAPI model proxy (SSO proxy) | 3001/3300 |
| `postgres` | `docker.io/library/postgres:15-alpine` | DB (DBRouterAiAtius) | (interno) |
| `redis` | `docker.io/library/redis:7-alpine` | Cache | (interno) |
| `<pod-id>-infra` | `localhost/podman-pause:4.9.3-0` | Network namespace holder (podman) | n/a |

## Network / Portas

O pod usa **network=host** (todos containers compartilham net ns via
infra pause). Bind mounts do **infra container**:
- `3000:3000` → `router-ai-atius` (UI/API)
- `3001:3001` → `model-detailed-hotfix` (FastAPI)
- `3300:3001` → alias Apache proxy → `model-detailed-hotfix`

Domínios públicos (via Apache):
- `https://router.atius.com.br/` → `127.0.0.1:3000`
- `https://docs.router.atius.com.br/` → mesmo upstream
- (sem proxy separado pro model-detailed ainda)

## Volumes críticos

| Tipo | Source (host) | Destination (ctr) | Conteúdo |
|------|---------------|-------------------|----------|
| bind | `/home/ubuntu/GitHub/containers/router-ai-atius/data` | `/data` | codex-home, logs, migrations, pg_backup, scalar, static |
| bind | `/home/ubuntu/GitHub/containers/router-ai-atius/logs` | `/app/logs` | oneapi-*.log, auto-sync-deploy.log |
| volume | `pgdata` (podman) | `/var/lib/postgresql/data` | Postgres data |

## Verificação

```bash
# Status do pod
podman pod inspect atius-ai-router
# Esperado: State=Running, NumContainers=5

# Containers
podman ps --filter pod=atius-ai-router
# Esperado: 5 linhas (infra, postgres, redis, router-ai-atius, model-detailed-hotfix)

# Backend responde
curl -sI http://localhost:3000/api/status | head -1
# Esperado: HTTP/1.1 200 OK

# Model-detailed responde
curl -sI http://localhost:3001/health | head -1
# Esperado: HTTP/1.1 200 OK

# Domínio público
curl -sI https://router.atius.com.br/api/status | head -1
# Esperado: HTTP/2 200
```

## CLIAnything

O deploy agora tem um CLI operacional para gestao sem frontend:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
bin/clianything status
bin/clianything coverage --strict
bin/clianything providers --all
bin/clianything resources
```

Principais garantias:

- `create`, `update` e `delete` sao dry-run por padrao.
- Escrita real exige `--execute`.
- Antes de qualquer escrita real, o CLI faz backup data-only da tabela em
  `backups/clianything/`.
- Campos sensiveis sao redigidos por padrao.
- `coverage --strict` valida 158 endpoints administrativos documentados contra
  `tools/clianything_endpoints.json`.
- `endpoint`, `channel`, `model`, `option`, `ratio`, `token`, `log`, `task` e
  `vendor` cobrem acoes do frontend/API sem depender do navegador.

Manuais:

- `docs/CLIANYTHING.md`
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`
- `docs/PROVIDERS-HERMES-CODEX.md`

## Restart / Recovery

Container restart é gerenciado pelo systemd unit `pod-atius-ai-router.service`
(gerado por `podman generate systemd --new --files`).

```bash
# Reiniciar pod inteiro
systemctl --user restart pod-atius-ai-router.service

# Reiniciar 1 container específico
podman restart router-ai-atius

# Logs
podman logs router-ai-atius --tail 50
podman logs model-detailed-hotfix --tail 50
```

## Recriar do zero (apenas se pod sumir)

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
podman-compose up -d
# OU (se podman-compose bug, ver script manual abaixo)
bash scripts/recreate-pod.sh
```

O script `scripts/recreate-pod.sh` faz o equivalente manual
(`podman pod create` + 4x `podman run --pod`).

## Histórico

- **2026-06-04** — Migração inicial Docker → Podman (5 containers).
  Vault: `60-LOGS/2026-06-04-atius-router-podman-cutover.md` (precedente).
- **2026-06-11** — Recriação do pod durante Phase 17 (rollback jenkins não
  afetou router-ai-atius). Compose atualizado para refletir imagem
  `ghcr.io/giovannimnz/router-ai-atius:latest` (substituiu
  `calciumion/new-api:latest`).
- **2026-06-12** — `podman-compose.yml` versionado neste diretório como
  source of truth pra re-deploy. Adicionado `model-detailed` (estava
  rodando no pod mas não estava no compose file original).
- **2026-06-12** — `docs/PODMAN.md` com notas de migração + lessons learned.

## References

- Vault: `20-PROJETOS/21-PROJETOS-ATIVOS/atius-router/router-ai-atius-panorama-fork-2026-06-01.md`
- Vault: `60-LOGS/2026-06-11-podman-migration-3srv-parcial.md` (Phase 17 partial)
- Vault: `61-Incidents/2026-06-07-router-ai-atius-prune-sem-backup-previo.md` (LIÇÃO)
- Phase plan: `/home/ubuntu/.planning/phases/17-podman-migration-3srv/17-01-PLAN.md`
- Source compose atual: `/home/ubuntu/GitHub/containers/router-ai-atius/podman-compose.yml`
- Admin config: `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/`
- Fork GHCR: `https://github.com/giovannimnz/router-ai-atius`
