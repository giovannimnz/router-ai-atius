# Podman runtime - router-ai-atius

This is the canonical Podman runbook for this checkout. Production runtime is
managed by rootless Podman plus user systemd from:

```bash
/home/ubuntu/GitHub/containers/router-ai-atius
```

## Current production shape

- Pod: `atius-ai-router`
- Containers: `router-ai-atius`, `redis`, `postgres`, infra pause
- Backend bind: `127.0.0.1:3000`
- Canonical DB path: PgBouncer on `10.1.1.1:6432` -> database `DBRouterAiAtius`
- Public router: `https://router.atius.com.br`
- Runtime source of truth: `container-router-ai-atius.service`
- Image: `ghcr.io/giovannimnz/router-ai-atius:latest`
- Runtime data/log bind paths: `data/` and `logs/` inside this checkout

The canonical `/v1/` path is full-Go. There is no Python `model-detailed`
container in the current production relay path.

## Daily checks

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
scripts/podman-admin.sh limits
scripts/podman-admin.sh status
bin/clianything status
bin/clianything providers --all
podman ps --filter pod=atius-ai-router --format '{{.Names}} {{.Image}} {{.Status}}'
systemctl --user status container-router-ai-atius.service --no-pager
```

Use the user unit for controlled backend restarts:

```bash
systemctl --user restart container-router-ai-atius.service
bin/clianything status
```

Do not use direct `podman restart router-ai-atius` as routine production
operation. The user unit owns the lifecycle and recovery semantics.

## Resource cap policy

This checkout is intentionally capped to at most 50% of this ARM host per
admin program/container. On the current 4-vCPU host that means two CPU cores.

The policy has two layers:

- admin process profile: `router-ai-atius-podman-admin.slice`
- profile CPU quota: `200%` on the current 4-vCPU host, equal to 50% of host CPU
- profile memory cap: `MemoryHigh=45%`, `MemoryMax=50%`, `MemorySwapMax=0`
- Podman CPU set: `0-1`
- Podman CPU max: `2` cores
- Podman CFS period/quota: `100000/200000`
- Podman memory cap: `11987M`, with `10788M` reservation and no extra swap
- Podman build parallelism: `2` jobs max

These caps are enforced in four places:

- live backend user unit `container-router-ai-atius.service`
- live stateful units `container-redis.service` and `container-postgres.service`
- development `podman-compose.yml`
- wrapper `scripts/podman-admin.sh` for compose/build/run operations

Quick checks:

```bash
scripts/podman-admin.sh limits
scripts/podman-admin.sh inspect-limits
scripts/podman-admin.sh verify-runtime-limits
scripts/podman-admin.sh verify-profile
scripts/podman-admin.sh verify-container-cgroups
```

`inspect-limits` and `verify-runtime-limits` check every non-infra container in
the production pod, not just the Go backend. The infra pause container is
ignored because it is not an application workload.

`verify-profile` starts a transient command through the admin `systemd-run`
profile and reads the active cgroup files. `verify-container-cgroups` then reads
the cgroup files from inside `router-ai-atius`, `redis`, and `postgres`.
On this host the expected hard lines are:

```text
cpu.max=200000 100000
cpuset.cpus.effective=0-1
memory.max=12569280512
memory.swap.max=0
```

The wrapper is fail-closed for the resource contract: overrides above 50% are
rejected, direct Podman CPU/memory flags passed by the caller are rejected, and
direct `podman build`, `podman run`, or `podman compose` through `profile-run`
or `omni-run` are rejected. Use the explicit `build`, `run-container`, and
`compose-*` commands so the script can inject native Podman limits.

## Phase 24 cutover status

Cutover applied on `2026-07-04`:

- `container-router-ai-atius.service` now points to
  `postgresql://admin:${POSTGRES_PASSWORD}@10.1.1.1:6432/DBRouterAiAtius`
- PgBouncer now keeps only the canonical runtime mapping:
  - `DBRouterAiAtius -> DBRouterAiAtius`
- the legacy `newapi` alias was removed from PgBouncer after final validation

Validated after cutover:

- `GET /v1/models` authenticated returns `gpt-5.5`, `gpt-5.4`,
  `gpt-5.4-mini`, `gpt-5.3-codex-spark`, `deepseek-v4-flash`,
  `deepseek-v4-pro`, and `embedding-gte-v1`
- `gpt-5.5-1m` and `gpt-5.4-1m` are absent from the live public catalog
- `POST /v1/embeddings` with `embedding-gte-v1` returns `768` dimensions
- `POST /v1/chat/completions` with `gpt-5.4` returns `200` after reloading
  channel 5 from `~/.codex/auth.json`
- `POST /v1/chat/completions` with `deepseek-v4-flash` and
  `deepseek-v4-pro` returns `200` after updating the active DeepSeek key
- MiniMax no longer appears in authenticated `GET /v1/models`, and
  `MiniMax-M3` returns `model_not_found`

Open blockers after cutover:

- Codex/OpenAI embeddings remain tied to upstream quota/licensing acceptance
  for `text-embedding-3-*`; this does not affect the governed live alias
  `embedding-gte-v1`

Rollback to pre-cleanup state:

```bash
cp /home/ubuntu/.config/systemd/user/container-router-ai-atius.service.phase24-*.bak \
  /home/ubuntu/.config/systemd/user/container-router-ai-atius.service
sudo cp /etc/pgbouncer/pgbouncer.ini.phase24-*.bak /etc/pgbouncer/pgbouncer.ini
sudo systemctl reload pgbouncer
systemctl --user daemon-reload
systemctl --user restart container-router-ai-atius.service
```

## Local development stack

The development compose file is:

```bash
podman-compose.yml
```

It binds the backend to host port `3001` by default to avoid colliding with the
live backend on `3000`.

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
scripts/podman-validate.sh
scripts/podman-admin.sh compose-up
scripts/podman-admin.sh compose-down
```

Use another local port when needed:

```bash
DEV_API_PORT=3002 scripts/podman-admin.sh compose-up
```

The make targets also use Podman:

```bash
make dev-api
make dev-api-rebuild
make reset-setup
make podman-status
make podman-verify
```

`PODMAN_COMPOSE` can be overridden if the host needs a specific wrapper:

```bash
PODMAN_COMPOSE="./scripts/podman-admin.sh compose-raw" make dev-api
```

## Build notes

`Dockerfile`, `Dockerfile.dev`, and `.dockerignore` are OCI build surfaces and
are intentionally kept with those names for upstream/tool compatibility. Podman
and Buildah can build from them directly:

```bash
scripts/podman-admin.sh build-image localhost/router-ai-atius:dev Dockerfile .
scripts/podman-admin.sh build-image localhost/router-ai-atius:dev-local Dockerfile.dev .
scripts/podman-admin.sh build -f Dockerfile -t localhost/router-ai-atius:dev .
scripts/podman-admin.sh run-container --rm alpine:3.20 uname -m
```

These commands automatically run through `router-ai-atius-podman-admin.slice`
and also pass native Podman CPU/memory flags to the build or runtime container.
For arbitrary heavy operations, use the same profile directly:

```bash
scripts/podman-admin.sh profile-run -- sh -c 'cat /sys/fs/cgroup$(awk -F: '\''$2=="" { print $3 }'\'' /proc/self/cgroup)/cpu.max'
```

These filenames do not mean the runtime is Docker.

`omni-run` remains available when the Omni resource governor should wrap a
non-Podman command for comparison with host-wide profiles:

```bash
scripts/podman-admin.sh omni-run builds -- true
```

The local admin profile is the default for this repo; the Podman flags still
limit the build or runtime container itself. For Podman workloads, use this
script's native commands, not `omni-run`.

## Boundaries

- Production runtime must stay on Podman in `/home/ubuntu/GitHub/containers`.
- Do not reintroduce `model-detailed` into the canonical `/v1/` path.
- Do not create a second OpenAI embeddings channel for Codex embeddings.
- Keep runtime directories out of build context: `/backups`, `/data`, `/logs`,
  `/runtime`.
- `docker-compose*.yml` may exist only as upstream/legacy compatibility while
  this fork's operational path uses `podman-compose.yml` and `makefile`.

## Verification gate

Before claiming a Podman config change is ready:

```bash
scripts/podman-validate.sh
scripts/podman-admin.sh verify
scripts/podman-admin.sh verify-profile
scripts/podman-admin.sh verify-container-cgroups
make -n dev-api dev-api-rebuild reset-setup
bin/clianything status
node "$HOME/.Codex/gsd-core/bin/gsd-tools.cjs" graphify status
```
