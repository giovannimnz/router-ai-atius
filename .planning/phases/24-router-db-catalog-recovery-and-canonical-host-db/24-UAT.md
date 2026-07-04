---
status: diagnosed
phase: 24-router-db-catalog-recovery-and-canonical-host-db
source: 24-01-SUMMARY.md, 24-02-SUMMARY.md, 24-03-SUMMARY.md, 24-04-SUMMARY.md
started: 2026-07-04T16:59:28-03:00
updated: 2026-07-04T18:35:17-03:00
---

## Current Test

[testing complete - 1 diagnosed runtime issue]

## Tests

### 1. Podman admin static checks
expected: `scripts/podman-admin.sh` and `scripts/podman-validate.sh` parse cleanly, pass shellcheck, and the standalone compose validator passes.
result: pass
evidence:
  - `bash -n scripts/podman-admin.sh scripts/podman-validate.sh`
  - `shellcheck scripts/podman-admin.sh scripts/podman-validate.sh`
  - `./scripts/podman-validate.sh`

### 2. Runtime Podman cgroup caps
expected: router, Redis, and Postgres remain capped at 2 CPUs, cpuset `0-1`, memory max `11987M`, and no memory swap.
result: pass
evidence:
  - `./scripts/podman-admin.sh verify`
  - `cpu.max=200000 100000`
  - `cpuset.cpus.effective=0-1`
  - `memory.max=12569280512`
  - `memory.swap.max=0`

### 3. User systemd resource profile
expected: long-running units and admin profile enforce at most 50% of the 4-vCPU host and roughly 50% memory.
result: pass
evidence:
  - `systemctl --user show container-router-ai-atius.service container-redis.service container-postgres.service -p ActiveState -p SubState -p CPUQuotaPerSecUSec -p MemoryHigh -p MemoryMax -p MemorySwapMax -p TasksMax --no-pager`
  - `ActiveState=active`
  - `CPUQuotaPerSecUSec=2s`
  - `MemoryHigh=11312037888`
  - `MemoryMax=12569280512`
  - `MemorySwapMax=0`
  - `TasksMax=8192`

### 4. Resource bypass guardrails
expected: env overrides, direct compose/make overrides, and direct Podman build/run flags cannot bypass the 50% cap.
result: pass
evidence:
  - `PODMAN_ADMIN_HOST_CPUS=100 PODMAN_ADMIN_CPUS=20 ./scripts/podman-admin.sh limits; test $? -eq 2`
  - `PODMAN_ADMIN_MEMORY_MAX=999999M PODMAN_ADMIN_MEMORY_SWAP=999999M podman compose -f podman-compose.yml config | rg -n 'mem_limit|memswap_limit|999999'`
  - `make dev-api PODMAN_COMPOSE='podman compose'; test $? -eq 2`
  - previously validated: `PODMAN_ADMIN_CPUS=3`, `build --memory=20G`, `profile-run -- podman build --help`, and `omni-run builds -- podman build --help` were rejected.

### 5. Disposable container path
expected: containers launched through the admin wrapper inherit the same cgroup limits.
result: pass
evidence:
  - `./scripts/podman-admin.sh run-container --rm --entrypoint sh docker.io/library/redis:7-alpine -c '...'`
  - `cpu.max=200000 100000`
  - `cpuset=0-1`
  - `memory.max=12569280512`
  - `memory.swap.max=0`

### 6. Public `/v1/models` schema contract
expected: public model items expose `pricing.input` and `pricing.output`, do not expose `pricing.unit`, `input_price`, `output_price`, `quota_type`, `enable_groups`, or `supported_endpoint_type_labels`.
result: pass
evidence:
  - `/usr/local/go/bin/go test ./controller ./service/modelcatalog ./setting/ratio_setting`
  - `/usr/local/go/bin/go test ./controller -run 'TestListModels(PayloadShapeAndPublicFields|CodexContractAfterPhase24Restore)$' -count=1`
  - `/usr/local/go/bin/go test ./service/modelcatalog -run TestModelCatalogEntryKeepsPricingProvenanceInternal -count=1`

### 7. GPT/Codex pricing contract
expected: Codex model pricing ratios match the verified configured values and stale stored ratios fall back to code defaults.
result: pass
evidence:
  - `/usr/local/go/bin/go test ./setting/ratio_setting -run 'TestCodexPublishedPricingRatios|TestCodexPricingFallsBackToCodeDefaultsWhenStoredRatiosAreStale' -count=1`

### 8. OpenAPI and CLIAnything docs parity
expected: public docs no longer advertise removed fields and `docs/openapi/relay.json` remains valid JSON.
result: pass
evidence:
  - `python3 -m json.tool docs/openapi/relay.json >/tmp/router-ai-atius-relay-openapi.json`
  - `rg '"supported_endpoint_type_labels"|"input_price"|"output_price"|usd_per_1m_tokens|"unit"' docs/openapi/relay.json docs/CLIANYTHING.md` returned no matches.

### 9. Python CLI and static smoke tests
expected: CLIAnything tests pass and smoke scripts fail closed when `ATIUS_ROUTER_TOKEN` is not set.
result: pass
evidence:
  - `python3 -m pytest tests/test_clianything.py scripts/test_long_context_aliases_static_test.py -q`
  - `41 passed, 1 skipped`
  - `unset ATIUS_ROUTER_TOKEN; python3 scripts/smoke-provider-consolidation.py; test $? -eq 2`
  - `unset ATIUS_ROUTER_TOKEN; python3 scripts/smoke-embeddings.py; test $? -eq 2`

### 10. Frontend production builds
expected: both embedded frontends build successfully for the Go binary embed paths.
result: pass
evidence:
  - `./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && /home/ubuntu/.bun/bin/bun run build && cd ../classic && /home/ubuntu/.bun/bin/bun run build'`

### 11. Podman production image build
expected: production image builds through `scripts/podman-admin.sh build`, with ARM64 target args and the 2-core admin profile.
result: pass
evidence:
  - `./scripts/podman-admin.sh build --build-arg TARGETOS=linux --build-arg TARGETARCH=arm64 -f Dockerfile -t localhost/router-ai-atius:validation-20260704-vault .`
  - `podman image inspect localhost/router-ai-atius:validation-20260704-vault --format 'id={{.Id}} os={{.Os}} arch={{.Architecture}} size={{.Size}}'`
  - `id=eca8b8eedef82f6c78cd9483416a3c0e7acd74ddac30499283194ed1d413c3b2 os=linux arch=arm64 size=250009750`

### 12. Go package suite
expected: operational Go packages pass when runtime data, backups, web assets, and electron directories are excluded from package discovery.
result: pass
evidence:
  - `PKGS=$(find . -path ./data -prune -o -path ./backups -prune -o -path ./runtime -prune -o -path ./web -prune -o -path ./electron -prune -o -name "*.go" -printf "%h\n" | sort -u); /usr/local/go/bin/go test $PKGS`
  - raw `go test ./...` is not a valid repo-wide gate because it enters `data/postgres_data` and `backups/manual-20260701T134746-codex-public-api-mode`.

### 13. Runtime health
expected: local services are running, backend health is reachable, unauthenticated `/v1/models` returns 401, and DB counts are readable.
result: pass
evidence:
  - `./scripts/podman-admin.sh cli status --strict`
  - `curl https://router.atius.com.br/v1/models` returned HTTP 401 with `Invalid token`.
  - `curl http://127.0.0.1:3000/v1/models` returned HTTP 401 with `Invalid token`.

### 14. Authenticated live `/v1/models` payload
expected: authenticated live `GET /v1/models` returns HTTP 200 and the public payload has no removed fields.
result: issue
severity: major
evidence:
  - HashiCorp Vault path `kv/atius/srv1/shell-exports/home-ubuntu-merged` provided `ATIUS_ROUTER_API_KEY`; the value was mapped to `ATIUS_ROUTER_TOKEN` only in the ephemeral validation shell and was not printed.
  - `curl -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" http://127.0.0.1:3000/v1/models` returned HTTP 200 with `count=7`, but `removed_fields_present=true`.
  - `curl -H "Authorization: Bearer $ATIUS_ROUTER_TOKEN" https://router.atius.com.br/v1/models` returned HTTP 200 with `count=7`, but `removed_fields_present=true`.
  - Removed fields still present in both local and public runtime responses: `enable_groups`, `input_price`, `output_price`, `pricing.unit`, `quota_type`, `supported_endpoint_type_labels`.
  - Active runtime container: `image=ghcr.io/giovannimnz/router-ai-atius:latest`, `image_id=079481f584d19335c9cb5fc7071ba14bbcce541a2424d39ecfac26c8283eae57`, `started=2026-07-04 15:29:25 -0300`.
diagnosis: source/tests and the newly built image contain the expected contract, but the active runtime is still serving an older `ghcr.io/giovannimnz/router-ai-atius:latest` image. A deploy/promote/restart of the validated image is still required before live `/v1/models` can pass.

### 15. Authenticated live embeddings smoke
expected: authenticated public `POST /v1/embeddings` with `embedding-gte-v1` returns OpenAI-shaped embeddings with dimension `768`.
result: pass
evidence:
  - HashiCorp Vault path `kv/atius/srv1/shell-exports/home-ubuntu-merged` provided `ATIUS_ROUTER_API_KEY`; the value was mapped to `ATIUS_ROUTER_TOKEN` only in the ephemeral validation shell and was not printed.
  - `ATIUS_ROUTER_EMBEDDINGS_BASE_URL=https://router.atius.com.br/v1 ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 python3 scripts/smoke-embeddings.py`
  - `embeddings ok: model=embedding-gte-v1 type=openai dimension=768`

## Summary

total: 15
passed: 14
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: authenticated live `GET /v1/models` returns HTTP 200 and the public payload has no removed fields
  status: failed
  severity: major
  root_cause: the active user systemd unit still runs `ghcr.io/giovannimnz/router-ai-atius:latest` image id `079481f584d19335c9cb5fc7071ba14bbcce541a2424d39ecfac26c8283eae57`, while the validated local image is `localhost/router-ai-atius:validation-20260704-vault` id `eca8b8eedef82f6c78cd9483416a3c0e7acd74ddac30499283194ed1d413c3b2`.
  missing:
    - promote or deploy the validated image to the production unit image reference
    - restart `container-router-ai-atius.service`
    - rerun the authenticated `/v1/models` field probe using `ATIUS_ROUTER_API_KEY` from HashiCorp Vault
