# Phase 24: router-db-catalog-recovery-and-canonical-host-db - Research

**Researched:** 2026-07-04
**Domain:** PostgreSQL/PgBouncer runtime recovery, router catalog restoration, Go-native provider routing
**Confidence:** HIGH

## User Constraints (from CONTEXT.md)

### Locked Decisions

- Recover the full `router-ai-atius` runtime/catalog on the canonical host PostgreSQL path via PgBouncer. `[VERIFIED: 24-CONTEXT.md]`
- This is not only an embeddings repair; it must recover `OpenAI - Codex`, known-good GPT/Codex models, DeepSeek V4 Flash/Pro, `embedding-gte-v1`, provider/channel consolidation invariants, and the Go embedding governor path. `[VERIFIED: 24-CONTEXT.md]`
- Restore `OpenAI - Codex`. `[VERIFIED: 24-CONTEXT.md]`
- Restore DeepSeek provider/channel and models `deepseek-v4-flash` and `deepseek-v4-pro`. `[VERIFIED: 24-CONTEXT.md]`
- Restore MiniMax provider/channel and models, but leave MiniMax disabled in the final state. `[VERIFIED: 24-CONTEXT.md]`
- Preserve `embedding-gte-v1` as the governed public embedding alias. `[VERIFIED: 24-CONTEXT.md]`
- Keep the DB on the host via PgBouncer. `[VERIFIED: 24-CONTEXT.md]`
- Do not recreate `gpt-5.4-1m` or `gpt-5.5-1m`. `[VERIFIED: 24-CONTEXT.md]`
- Do not restore `text-embedding-3-small` or `text-embedding-3-large`. `[VERIFIED: 24-CONTEXT.md]`
- Do not restore channel 5 `channels.model_mapping` entries for `gpt-5.5-1m -> gpt-5.5` or `gpt-5.4-1m -> gpt-5.4`. `[VERIFIED: 24-CONTEXT.md]`
- `gpt-5.4` becomes the default long-context Codex model and is treated as approximately `1050000` max tokens in the intended restored contract. `[VERIFIED: 24-CONTEXT.md]`
- `gpt-5.5` does not get a 1M Codex alias. `[VERIFIED: 24-CONTEXT.md]`
- DeepSeek must end with only one channel and automatic OpenAI/Anthropic routing in Go. `[VERIFIED: 24-CONTEXT.md]`
- MiniMax must end with only one channel and automatic OpenAI/Anthropic routing in Go, with the final MiniMax channel and related models disabled. `[VERIFIED: 24-CONTEXT.md]`
- No secret printing in docs, plans, summaries, logs, or diffs. `[VERIFIED: 24-CONTEXT.md; AGENTS.md]`
- No destructive mutation without a fresh validated backup. `[VERIFIED: 24-CONTEXT.md; AGENTS.md]`
- Preserve the full-Go runtime path, the Go-native embedding governor path, and the host PgBouncer approach. `[VERIFIED: 24-CONTEXT.md]`
- Use local evidence and backups only; no GDrive is required for this phase. `[VERIFIED: user prompt]`

### the agent's Discretion

- CONTEXT.md does not contain a literal `## the agent's Discretion` section. `[VERIFIED: 24-CONTEXT.md]`
- The strategy choice left for research is the safest DB-name/cutover path from current `newapi` to the intended canonical DB identity. `[VERIFIED: 24-CONTEXT.md]`

### Deferred Ideas (OUT OF SCOPE)

- CONTEXT.md does not contain a literal `## Deferred Ideas` section. `[VERIFIED: 24-CONTEXT.md]`
- Reintroducing Python/model-detailed as owner of `/v1/models` or embeddings is out of scope because Phase 20 made Go the canonical owner. `[VERIFIED: REQUIREMENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`
- Recreating `gpt-5.4-1m`, `gpt-5.5-1m`, `text-embedding-3-small`, or `text-embedding-3-large` is out of scope by user constraint. `[VERIFIED: 24-CONTEXT.md]`

## Summary

The runtime is healthy but pointed at the wrong host DB identity: `container-router-ai-atius.service` runs through PgBouncer at `10.1.1.1:6432/newapi`, PgBouncer maps `newapi` to host PostgreSQL `127.0.0.1:8745 dbname=newapi`, and the host cluster does not contain `DBRouterAiAtius`. `[VERIFIED: systemd; pgbouncer config; psql]` The live `newapi` database has current operational data (`users=6`, `tokens=8`, `logs=85435`) and the 2026-07-03 embedding-only repair (`channel 9`, `embedding-gte-v1`), but it is missing `channel 5 = OpenAI - Codex`, all GPT/Codex model rows, and all GPT/Codex abilities. `[VERIFIED: clianything query]`

The best catalog source is the 2026-07-01 `clianything` table snapshots for `channels`, `models`, and `abilities`; the 2026-07-03 `newapi-before.fix.dump` is valuable for rollback/diffing but predates the embedding repair and still lacks the modern Codex catalog. `[VERIFIED: pg_restore; backups/clianything SQL]` The recovery must therefore preserve current `newapi` non-catalog state, apply a transformed 2026-07-01 catalog restore, and explicitly skip the now-forbidden `-1m` aliases and Codex embedding rows. `[VERIFIED: clianything query; 24-CONTEXT.md; backups/clianything SQL]`

**Primary recommendation:** create a new host PostgreSQL database named `DBRouterAiAtius` from a fresh dump of current `newapi`, apply catalog reconciliation there, add a PgBouncer mapping for `DBRouterAiAtius`, repoint the router unit and `clianything` defaults to `DBRouterAiAtius`, restart via user systemd, validate, and keep old `newapi` untouched as rollback until the phase gate passes. `[VERIFIED: STATE.md; container-postgres.service; pgbouncer config; clianything.py]`

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Canonical DB identity and copy/restore | Database / Storage | Ops runtime | PostgreSQL owns persistent router state; PgBouncer and systemd only route the application to it. `[VERIFIED: psql; pgbouncer config; systemd]` |
| Router catalog row recovery | Database / Storage | API / Backend | `channels`, `models`, and `abilities` rows decide available providers/models; Go consumes those rows for routing and `/v1/models`. `[VERIFIED: clianything.py; service/modelcatalog/catalog.go]` |
| Provider consolidation | API / Backend | Database / Storage | Go channel types `35`, `43`, and `57` own routing semantics, while DB rows choose which channel/model pairs are enabled. `[VERIFIED: AGENTS.md; tools/clianything.py; service/modelcatalog/catalog.go]` |
| Embedding governor preservation | API / Backend | Database / Storage | `service/embeddinggovernor` gates `embedding-gte-v1` before upstream dispatch; DB channel/model/ability rows expose the governed public alias. `[VERIFIED: service/embeddinggovernor/governor.go; relay/embedding_handler.go; clianything query]` |
| Cutover and rollback | Ops runtime | Database / Storage | systemd, PgBouncer, and backups determine whether the application can switch DB names and revert without data loss. `[VERIFIED: systemd; pgbouncer config; docs/PODMAN.md]` |

## Project Constraints (from AGENTS.md)

- Use PT-BR with Giovanni unless requested otherwise. `[VERIFIED: AGENTS.md]`
- Use Graphify before GSD planning/broad codebase consultation when `.planning/config.json` has Graphify enabled. `[VERIFIED: AGENTS.md; .planning/config.json]`
- Do not overwrite or revert dirty worktree changes made by others. `[VERIFIED: AGENTS.md]`
- Do not print or commit secrets. `[VERIFIED: AGENTS.md]`
- Use Go layered architecture: Router -> Controller -> Service -> Model. `[VERIFIED: AGENTS.md]`
- Use `common/json.go` wrappers for JSON marshal/unmarshal in business code. `[VERIFIED: AGENTS.md]`
- Preserve SQLite, MySQL, and PostgreSQL compatibility for application DB code; this phase's operational SQL is PostgreSQL-host-specific and must stay out of portable app migrations unless cross-DB fallbacks are designed. `[VERIFIED: AGENTS.md]`
- Backend tests should protect behavior and use `testify/require` and `testify/assert` for new or substantially rewritten Go backend tests. `[VERIFIED: AGENTS.md; go.mod]`
- Preserve fork-specific Go-native routing customizations, including Go-owned `/v1/models`, consolidated MiniMax/DeepSeek channels, Codex embeddings behavior, Go embedding governor, and runtime-directory `.dockerignore` exclusions. `[VERIFIED: AGENTS.md]`
- Do not modify protected upstream project identity/branding. `[VERIFIED: AGENTS.md]`

## Exact Current Drift State

| Surface | Current Evidence | Impact |
|---------|------------------|--------|
| Router DB DSN | Running unit and container env use `postgresql://admin:<redacted>@10.1.1.1:6432/newapi`. `[VERIFIED: container-router-ai-atius.service; podman inspect redacted]` | Runtime writes to host PgBouncer database name `newapi`, not the intended canonical name. |
| PgBouncer mapping | `/etc/pgbouncer/pgbouncer.ini` maps `newapi = host=127.0.0.1 port=8745 dbname=newapi`; no `DBRouterAiAtius` mapping exists. `[VERIFIED: pgbouncer config]` | Repointing the router to `DBRouterAiAtius` will fail until PgBouncer has a matching mapping. |
| Host DB names | Host PostgreSQL contains `newapi` but not `DBRouterAiAtius`. `[VERIFIED: psql]` | The intended DB must be created/copied before cutover. |
| Runtime Postgres container | `container-postgres.service` is inactive and its unit still declares `POSTGRES_DB=DBRouterAiAtius`. `[VERIFIED: systemd; container-postgres.service]` | Production must not fall back to Podman Postgres; the unit is historical/runtime-drift evidence, not the active DB path. |
| Live DB counts | `newapi` has `channels=5`, `models=14`, `abilities=18`, `tokens=8`, `users=6`, `logs=85435`. `[VERIFIED: clianything query]` | Current operational identity/log/token data must be preserved; do not overwrite it with an older full dump. |
| Live channels | Active channel IDs are `1`, `2`, `3`, `4`, `9`. `[VERIFIED: clianything providers --all]` | Split MiniMax Anthropic channels are still active; final state must not keep split active routes. |
| Live MiniMax | `id=1` is `MiniMax - Token Plan`, type `35`, active; `id=3` and `id=4` are active type `14` split Anthropic channels. `[VERIFIED: clianything providers --all]` | Must consolidate to one MiniMax channel and leave MiniMax disabled in final state. |
| Live DeepSeek | `id=2` is type `43`, active, named `DeepSeek API`, with `deepseek-v4-flash,deepseek-v4-pro`. `[VERIFIED: clianything providers --all]` | Correct model set and type exist, but final naming/remarks should be reconciled to consolidated `DeepSeek`. |
| Live Codex | No `OpenAI - Codex`, no channel `5`, no `gpt-*` model rows, and no channel 5 abilities exist. `[VERIFIED: clianything query]` | Must restore channel 5 and allowed GPT/Codex rows from 2026-07-01 snapshots. |
| Live Codex embeddings | No `text-embedding-3-small` or `text-embedding-3-large` rows exist. `[VERIFIED: clianything query]` | Correct for final constraints; do not restore those rows. |
| Live local embeddings | `channel 9` is active, type `1`, named `Local TEI - GTE Embeddings`, model `embedding-gte-v1`, with enabled ability tag `local-tei`. `[VERIFIED: clianything embeddings; clianything query]` | Preserve this path during catalog restore. |
| Governor env | Container env has `EMBEDDING_GOVERNOR_ENABLED=true`, `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1`, and empty `EMBEDDING_GOVERNOR_BATCH_MODELS`. `[VERIFIED: podman inspect redacted]` | Matches the intended governed public alias contract. |
| Public unauthenticated model-list | `GET http://127.0.0.1:3000/v1/models` returns HTTP `401` without token. `[VERIFIED: curl]` | Healthy auth boundary; authenticated model-list validation still requires a token. |
| Available auth env | `ATIUS_ROUTER_TOKEN` and `ATIUS_ROUTER_ADMIN_TOKEN` are unset in this shell. `[VERIFIED: shell env check]` | Authenticated validation must be a phase gate, not a completed research check. |

## Exact Known-Good State From 2026-07-01 Catalog Snapshots

The 2026-07-01 snapshots were dumped by PostgreSQL `15.18` and cover only the catalog tables `channels`, `models`, and `abilities`. `[VERIFIED: backups/clianything/20260701_184735_*.sql]`

### Channels Snapshot

| ID | Type | Snapshot Name | Snapshot Status | Snapshot Models | Final Phase 24 Disposition |
|----|------|---------------|-----------------|-----------------|----------------------------|
| `1` | `35` | `MiniMax` | enabled | `MiniMax-M3,MiniMax-M2.7-highspeed,MiniMax-M2.7` | Restore as the single consolidated MiniMax channel, but set final `status=2` and disable related abilities/models per user request. `[VERIFIED: channels snapshot; 24-CONTEXT.md]` |
| `2` | `43` | `DeepSeek` | enabled | `deepseek-v4-pro,deepseek-v4-flash` | Restore/reconcile as the single active DeepSeek channel. `[VERIFIED: channels snapshot; 24-CONTEXT.md]` |
| `5` | `57` | `OpenAI - Codex` | enabled | `gpt-5.5,gpt-5.5-1m,gpt-5.4,gpt-5.4-1m,gpt-5.4-mini,gpt-5.3-codex-spark` | Restore channel 5, but remove `-1m` models from `channels.models` and clear/omit the `-1m` `model_mapping`. `[VERIFIED: channels snapshot; 24-CONTEXT.md]` |
| `9` | `1` | `Local TEI - GTE Embeddings` | enabled | `embedding-gte-v1` | Preserve as-is; mapping remains `embedding-gte-v1 -> text-embeddings-inference`. `[VERIFIED: channels snapshot; clianything embeddings]` |

### Models Snapshot

| Model | Snapshot ID | Snapshot Status | Snapshot Deleted? | Final Phase 24 Disposition |
|-------|-------------|-----------------|-------------------|----------------------------|
| `gpt-5.5` | `14` | enabled | no | Restore active. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `gpt-5.4` | `15` | enabled | no | Restore active and treat as default long-context Codex model. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `gpt-5.4-mini` | `16` | enabled | no | Restore active. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `gpt-5.3-codex-spark` | `17` | enabled | no | Restore active. `[VERIFIED: models snapshot; REQUIREMENTS.md]` |
| `gpt-5.5-1m` | `22` | enabled | no | Do not restore. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `gpt-5.4-1m` | `23` | enabled | no | Do not restore. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `deepseek-v4-flash` | `5` | enabled | no | Restore/reconcile active. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `deepseek-v4-pro` | `6` | enabled | no | Restore/reconcile active. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `MiniMax-M3` | `13` | enabled | no | Restore/reconcile but final disabled. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `MiniMax-M2.7-highspeed` | `2` | enabled | no | Restore/reconcile but final disabled. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `MiniMax-M2.7` | `1` | enabled | no | Restore/reconcile but final disabled. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| Legacy MiniMax `M2.5`, `M2.1`, `*-hs` rows | `3,4,7,8,10,11,12` | mixed enabled/disabled | mostly deleted in snapshot | Keep disabled/deleted if present; do not advertise as active final models. `[VERIFIED: models snapshot; AGENTS.md]` |
| `embo-01` | `18` | disabled | yes | Keep disabled/historical only if needed; do not add to active `channels.models`. `[VERIFIED: models snapshot; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| `text-embedding-3-small` | `19` | disabled | yes | Do not restore. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `text-embedding-3-large` | `20` | disabled | yes | Do not restore. `[VERIFIED: models snapshot; 24-CONTEXT.md]` |
| `embedding-gte-v1` | `21` | enabled | no | Preserve active. `[VERIFIED: models snapshot; clianything query]` |

### Abilities Snapshot

| Ability Set | Snapshot State | Final Phase 24 Disposition |
|-------------|----------------|----------------------------|
| `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, `gpt-5.3-codex-spark` on channel `5` | enabled | Restore enabled. `[VERIFIED: abilities snapshot; 24-CONTEXT.md]` |
| `gpt-5.4-1m`, `gpt-5.5-1m` on channel `5` | enabled | Do not restore. `[VERIFIED: abilities snapshot; 24-CONTEXT.md]` |
| `deepseek-v4-pro`, `deepseek-v4-flash` on channel `2` | enabled | Restore/reconcile enabled. `[VERIFIED: abilities snapshot; 24-CONTEXT.md]` |
| `MiniMax-M3`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.7` on channel `1` | enabled | Restore/reconcile but set disabled for final state. `[VERIFIED: abilities snapshot; 24-CONTEXT.md]` |
| `embo-01` on channel `1` | disabled | Keep disabled if present; do not expose. `[VERIFIED: abilities snapshot; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| `text-embedding-3-small`, `text-embedding-3-large` on channel `5` | disabled | Do not restore. `[VERIFIED: abilities snapshot; 24-CONTEXT.md]` |
| `embedding-gte-v1` on channel `9` | enabled, tag `local-tei` | Preserve enabled. `[VERIFIED: abilities snapshot; clianything query]` |

## Recovery Source Ranking

| Rank | Source | Use | Why |
|------|--------|-----|-----|
| 1 | Fresh Phase 24 backup of current host `newapi` | Source of truth for non-catalog runtime data and rollback | Current DB has latest `users`, `tokens`, `logs`, options, and the embedding-only repair. `[VERIFIED: clianything query]` |
| 2 | `backups/clianything/20260701_184735_channels.sql`, `models.sql`, `abilities.sql` | Source of truth for known-good catalog templates | Contains modern `OpenAI - Codex`, GPT/Codex, DeepSeek, MiniMax consolidated, and `embedding-gte-v1` rows before drift. `[VERIFIED: backups/clianything SQL]` |
| 3 | Current live `newapi` catalog | Preservation source for channel 9 and current sequence/state | Contains the post-fix `embedding-gte-v1` rows and current host identity data. `[VERIFIED: clianything embeddings; clianything query]` |
| 4 | `/home/ubuntu/.backups/router-ai-atius-incident-20260703T231027-0300/newapi-before.fix.dump` | Rollback/diff source only | It is a full custom dump of `newapi` with `channels=4`, `models=13`, `abilities=17`, `users=6`, `tokens=7`, `logs=85390`, but lacks Codex and `embedding-gte-v1`. `[VERIFIED: pg_restore]` |
| 5 | Older local backups listed in CONTEXT.md | Emergency fallback | Older backups may predate full-Go consolidation and require heavier transformation. `[VERIFIED: 24-CONTEXT.md]` |

## Recommended Canonical DB Naming/Cutover Strategy

Use `DBRouterAiAtius` as the intended canonical final DB name. `[VERIFIED: STATE.md; container-postgres.service]` This is an evidence-based recommendation because project state records `DB: DBRouterAiAtius`, the historical Podman Postgres unit declares `POSTGRES_DB=DBRouterAiAtius`, and current docs call `newapi` the live PgBouncer path after drift rather than the intended identity. `[VERIFIED: STATE.md; container-postgres.service; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

Do not run an in-place `ALTER DATABASE newapi RENAME TO DBRouterAiAtius` as the primary path. `[VERIFIED: psql; pgbouncer config]` A rename requires disconnecting active sessions and removes the simplest rollback target; a copy/restore strategy preserves `newapi` intact while validating the candidate. `[VERIFIED: systemd; pgbouncer config; clianything status]`

Recommended order:

1. Take a fresh full custom dump of current `newapi` and a catalog-only dump of `channels`, `models`, and `abilities`; verify `pg_restore -l`, file sizes, checksums, and table counts before any mutation. `[VERIFIED: pg_dump/pg_restore availability; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`
2. Build a candidate host DB named `DBRouterAiAtius` from the fresh `newapi` dump, preferably during a short freeze window or by rehearsal first on a timestamped staging DB. `[VERIFIED: psql; pg_restore]`
3. Apply catalog reconciliation on the candidate DB, not on live `newapi`: insert/reconcile channel `5`, allowed GPT models `14-17`, DeepSeek rows, MiniMax disabled final state, and preserve channel `9`. `[VERIFIED: backups/clianything SQL; 24-CONTEXT.md]`
4. Set `channels_id_seq` to at least `9` and `models_id_seq` to at least the max restored model ID after reconciliation. `[VERIFIED: clianything query; backups/clianything SQL]`
5. Add PgBouncer mapping `DBRouterAiAtius = host=127.0.0.1 port=8745 dbname=DBRouterAiAtius` and reload PgBouncer without removing the old `newapi` mapping. `[VERIFIED: pgbouncer config]`
6. Update the router unit DSN path from `/newapi` to `/DBRouterAiAtius`; keep host `10.1.1.1:6432` and the existing PgBouncer route. `[VERIFIED: container-router-ai-atius.service]`
7. Update `tools/clianything.py` default host DB from `newapi` to `DBRouterAiAtius` after cutover, or require `CLIANYTHING_DB_NAME=DBRouterAiAtius` in validation until code/docs are reconciled. `[VERIFIED: tools/clianything.py]`
8. Restart only through `systemctl --user restart container-router-ai-atius.service`, then run strict DB/catalog/API gates. `[VERIFIED: docs/PODMAN.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`
9. Keep old `newapi` and its PgBouncer mapping as rollback until the Phase 24 verification gate passes; do not drop it in this phase. `[VERIFIED: psql; pgbouncer config; 24-CONTEXT.md]`

## Standard Stack

### Core

| Tool/Library | Version | Purpose | Why Standard |
|--------------|---------|---------|--------------|
| Host PostgreSQL | server accepts on `127.0.0.1:8745`; dump evidence includes PostgreSQL 17.10 for `newapi-before.fix.dump` | Canonical persistent runtime DB | Current production data lives there via PgBouncer. `[VERIFIED: pg_isready; pg_restore]` |
| PgBouncer | `1.25.2` | Transaction-pooling DB entrypoint on `127.0.0.1,10.1.1.1:6432` | Runtime already uses PgBouncer; final state must keep this path. `[VERIFIED: pgbouncer --version; pgbouncer config]` |
| rootless Podman + user systemd | Podman `4.9.3`, systemd `255` | Router lifecycle and restart/cutover control | Docs and units identify user systemd as source of truth. `[VERIFIED: podman --version; systemctl --version; docs/PODMAN.md]` |
| `clianything` | local `bin/clianything` | Safe read-only queries, redacted output, backups, dry-run writes | Existing operational CLI defaults to host DB and provides table backup before writes. `[VERIFIED: tools/clianything.py]` |
| Go router | Go `1.25.1`; module `github.com/QuantumNous/new-api` | API gateway, `/v1/models`, relay, provider adaptors | Full-Go runtime is the canonical API path. `[VERIFIED: go.mod; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |

### Supporting

| Tool/Library | Version | Purpose | When to Use |
|--------------|---------|---------|-------------|
| `pg_dump` / `pg_restore` | `17.10` | Full/cat-only backups, candidate restore, dump inspection | Required before any DB mutation and for rollback. `[VERIFIED: pg_dump --version; pg_restore --version]` |
| `psql` | `18.4` client | DB inventory, PgBouncer/admin validation, table counts | Use for read-only probes and controlled restore checks. `[VERIFIED: psql --version]` |
| `curl` / `jq` | curl `8.5.0`, jq `1.7` | HTTP validation and JSON assertions | Use for unauthenticated and authenticated API gates. `[VERIFIED: curl --version; jq --version]` |
| Python 3 | `3.12.3` | Existing smoke scripts and CLI tests | Use `scripts/smoke-provider-consolidation.py`, `scripts/smoke-embeddings.py`, and `tests/test_clianything.py`. `[VERIFIED: python3 --version; repo files]` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Copy/restore to `DBRouterAiAtius` | In-place rename `newapi` | Rename is shorter but weaker for rollback and requires disconnecting active PgBouncer/router sessions. `[VERIFIED: pgbouncer config; systemd]` |
| Transform 2026-07-01 catalog rows | Replay snapshots blindly | Blind replay would restore forbidden `-1m` aliases, `model_mapping`, text embeddings, and active MiniMax. `[VERIFIED: backups/clianything SQL; 24-CONTEXT.md]` |
| Preserve current `newapi` non-catalog data | Restore the 2026-07-03 full dump over live DB | Full dump is older than live state and lacks embedding repair; overwriting current DB would lose post-dump logs/tokens and still not restore Codex. `[VERIFIED: pg_restore; clianything query]` |

**Installation:** no external package installation is required for Phase 24. `[VERIFIED: environment availability probes]`

## Package Legitimacy Audit

No new external packages are recommended or required for this phase. `[VERIFIED: go.mod; environment availability probes]`

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| N/A | N/A | N/A | N/A | N/A | N/A | No install needed. `[VERIFIED: go.mod]` |

**Packages removed due to [SLOP] verdict:** none. `[VERIFIED: no package installs]`
**Packages flagged as suspicious [SUS]:** none. `[VERIFIED: no package installs]`

## Architecture Patterns

### System Architecture Diagram

```text
Client / SDK / GBrain
  -> Apache / local router port
  -> Go router-ai-atius process
      -> /v1/models owned by Go modelcatalog
      -> /v1/chat/completions, /v1/responses, /v1/messages provider relay
      -> /v1/embeddings
          -> embeddinggovernor gate for public model embedding-gte-v1
          -> Local TEI channel 9 upstream
  -> SQL_DSN through PgBouncer 10.1.1.1:6432
      -> final DBRouterAiAtius host DB on 127.0.0.1:8745
      -> old newapi retained for rollback
```

This data flow reflects the current full-Go API path and the required final host PgBouncer DB path. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; pgbouncer config; systemd]`

### Recommended Project Structure

```text
.planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/
├── 24-RESEARCH.md        # this research artifact
├── 24-01-PLAN.md         # inventory, freeze, backup, source selection
├── 24-02-PLAN.md         # candidate DB create/restore and catalog reconciliation
├── 24-03-PLAN.md         # provider consolidation and embedding governor preservation
└── 24-04-PLAN.md         # cutover, docs reconciliation, validation, rollback
```

This structure matches the roadmap's four planned Phase 24 waves. `[VERIFIED: .planning/ROADMAP.md]`

### Pattern 1: Candidate DB Before Runtime Cutover

**What:** restore current `newapi` into `DBRouterAiAtius`, reconcile catalog there, and switch PgBouncer/systemd only after validation. `[VERIFIED: psql; pgbouncer config]`

**When to use:** any DB identity repair where the current DB has live operational data and a rollback target must remain intact. `[VERIFIED: clianything query; 24-CONTEXT.md]`

**Example:**

```bash
# Source: local PostgreSQL/PgBouncer runtime, redacted/no secrets.
sudo -u postgres pg_dump -p 8745 -Fc -d newapi \
  -f /home/ubuntu/.backups/router-ai-atius-phase24/newapi-pre-cutover.dump
sudo -u postgres pg_restore -l /home/ubuntu/.backups/router-ai-atius-phase24/newapi-pre-cutover.dump >/tmp/newapi-pre-cutover.toc
```

### Pattern 2: Catalog Transform, Not Replay

**What:** use 2026-07-01 rows as templates, then apply Phase 24 transforms: remove `-1m`, skip text embeddings, disable MiniMax, preserve `embedding-gte-v1`. `[VERIFIED: backups/clianything SQL; 24-CONTEXT.md]`

**When to use:** catalog snapshots conflict with newer user constraints. `[VERIFIED: 24-CONTEXT.md]`

**Example:**

```sql
-- Source: 2026-07-01 snapshots, transformed for Phase 24.
-- Do not include gpt-5.4-1m, gpt-5.5-1m, text-embedding-3-small, or text-embedding-3-large.
-- Do not include channel 5 model_mapping entries for -1m aliases.
```

### Anti-Patterns to Avoid

- **Blind snapshot replay:** restores rows the user explicitly forbade. `[VERIFIED: backups/clianything SQL; 24-CONTEXT.md]`
- **Full-dump overwrite of current `newapi`:** risks losing current tokens/logs and does not recover Codex from `newapi-before.fix.dump`. `[VERIFIED: pg_restore; clianything query]`
- **Re-enabling split MiniMax/DeepSeek channels:** violates the Go-native consolidation contract. `[VERIFIED: AGENTS.md; REQUIREMENTS.md]`
- **Falling back to Podman Postgres:** violates the host PgBouncer final-state constraint. `[VERIFIED: 24-CONTEXT.md; systemd]`
- **Deleting old DB before validation:** removes rollback before UAT proves the recovered state. `[VERIFIED: 24-CONTEXT.md]`

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Full DB backup/restore | Custom SQL copier | `pg_dump -Fc` and `pg_restore` | Preserves schema/data semantics and supports TOC inspection. `[VERIFIED: pg_dump/pg_restore]` |
| Table backup before writes | Manual ad hoc copy | `bin/clianything backup <resource>` or `pg_dump --data-only --column-inserts` | Existing CLI already writes table backups and redacts normal output. `[VERIFIED: tools/clianything.py]` |
| Catalog querying | Direct secret-printing SQL selects | `bin/clianything query`, `providers`, `embeddings` | CLI redacts sensitive columns and gives typed inventory. `[VERIFIED: tools/clianything.py]` |
| Provider routing | New sidecar/middleware | Existing Go channel types/adaptors | Full-Go route ownership is a locked fork invariant. `[VERIFIED: AGENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| Embedding throttling | New queue/service | `service/embeddinggovernor` | Governor already enforces model-scoped concurrency and workload classification. `[VERIFIED: service/embeddinggovernor/governor.go; relay/embedding_handler.go]` |

**Key insight:** the hard part is not SQL syntax; it is preserving the right state boundary. `[VERIFIED: clianything query; pg_restore]` Current `newapi` owns live operational state, while 2026-07-01 snapshots own the missing catalog templates. `[VERIFIED: clianything query; backups/clianything SQL]`

## Runtime State Inventory

| Category | Items Found | Action Required |
|----------|-------------|-----------------|
| Stored data | Host PostgreSQL has live `newapi`; no `DBRouterAiAtius`; current `newapi` has `users=6`, `tokens=8`, `logs=85435`, `channels=5`, `models=14`, `abilities=18`. `[VERIFIED: psql; clianything query]` | Fresh full backup, create/copy candidate `DBRouterAiAtius`, apply catalog reconciliation, keep `newapi` for rollback. |
| Live service config | PgBouncer maps only `newapi` for this router DB; router unit DSN points to `/newapi`; docs and `tools/clianything.py` also currently describe/default to `newapi`. `[VERIFIED: pgbouncer config; systemd; docs/PODMAN.md; tools/clianything.py]` | Add `DBRouterAiAtius` PgBouncer mapping, update unit DSN, update CLI default/docs after cutover. |
| OS-registered state | `container-router-ai-atius.service` is active; `container-postgres.service` is inactive but still declares `POSTGRES_DB=DBRouterAiAtius`; `pgbouncer.service` is active. `[VERIFIED: systemd]` | Use user systemd for restart only; do not reactivate Podman Postgres for production. |
| Secrets/env vars | `/home/ubuntu/.config/router-ai-atius/.env` contains only variable names in research output; channel credentials exist inside DB/snapshots and must not be printed. `[VERIFIED: env var name audit; backups/clianything SQL]` | Keep secret-bearing row restores in controlled SQL files/logless execution; redact all outputs. |
| Build artifacts | Runtime directories `backups`, `data`, `logs`, `runtime` are protected from build context by project policy; no compiled artifact needs rename for DB identity. `[VERIFIED: AGENTS.md; docs/PODMAN.md]` | Update docs/runtime config only; no build artifact rewrite needed. |

**Nothing found in category:** no OS-level scheduler or global package install was found as a required DB-name carrier in the required evidence set. `[VERIFIED: systemd; required files]`

## Common Pitfalls

### Pitfall 1: Restoring The Wrong Source

**What goes wrong:** applying `newapi-before.fix.dump` as the recovery source keeps the old incomplete catalog. `[VERIFIED: pg_restore]`
**Why it happens:** the dump is a full local backup and looks authoritative, but it predates the embedding repair and lacks Codex. `[VERIFIED: pg_restore; clianything query]`
**How to avoid:** use it for rollback/diff only; use current `newapi` plus 2026-07-01 catalog snapshots for recovery. `[VERIFIED: pg_restore; backups/clianything SQL]`
**Warning signs:** candidate DB has no channel `5`, no `gpt-*` models, or no `embedding-gte-v1`. `[VERIFIED: clianything query]`

### Pitfall 2: Replaying Forbidden Rows

**What goes wrong:** raw 2026-07-01 snapshots restore `gpt-5.4-1m`, `gpt-5.5-1m`, text embeddings, and channel 5 `model_mapping`. `[VERIFIED: backups/clianything SQL]`
**Why it happens:** snapshots represent known-good then, not the current user-mandated final state. `[VERIFIED: 24-CONTEXT.md]`
**How to avoid:** generate a transformed restore script with an explicit skip list and a post-restore negative query gate. `[VERIFIED: 24-CONTEXT.md]`
**Warning signs:** any query returns rows for `gpt-5.4-1m`, `gpt-5.5-1m`, `text-embedding-3-small`, or `text-embedding-3-large`. `[VERIFIED: 24-CONTEXT.md]`

### Pitfall 3: MiniMax Accidentally Active

**What goes wrong:** raw channel/model/ability restore makes MiniMax active. `[VERIFIED: channels snapshot; abilities snapshot]`
**Why it happens:** the 2026-07-01 source had MiniMax enabled. `[VERIFIED: backups/clianything SQL]`
**How to avoid:** restore MiniMax channel/model records only into disabled final state; disable split channels `3` and `4` and their abilities. `[VERIFIED: 24-CONTEXT.md; clianything providers --all]`
**Warning signs:** `/v1/models` advertises MiniMax or `bin/clianything providers` shows active MiniMax. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

### Pitfall 4: Losing Live Tokens And Logs

**What goes wrong:** restoring an older full dump over live `newapi` loses current token/log deltas. `[VERIFIED: pg_restore; clianything query]`
**Why it happens:** `newapi-before.fix.dump` had `tokens=7`, `logs=85390`, while current `newapi` has `tokens=8`, `logs=85435`. `[VERIFIED: pg_restore; clianything query]`
**How to avoid:** copy current `newapi` first, then reconcile catalog on the copy. `[VERIFIED: clianything query]`
**Warning signs:** counts move backward after restore. `[VERIFIED: pg_restore; clianything query]`

### Pitfall 5: Secret Leakage During Catalog Restore

**What goes wrong:** channel keys/OAuth JSON leak into planning docs, shell logs, or terminal output. `[VERIFIED: backups/clianything SQL; AGENTS.md]`
**Why it happens:** `channels` rows contain provider credentials. `[VERIFIED: backups/clianything SQL]`
**How to avoid:** never paste full channel rows; restore from local files in controlled commands and only output IDs/names/status. `[VERIFIED: tools/clianything.py; AGENTS.md]`
**Warning signs:** output includes `channels.key`, OAuth JSON, bearer tokens, refresh tokens, or API keys. `[VERIFIED: AGENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

## Code Examples

### Read-Only Drift Checks

```bash
# Source: local runtime checks used in research.
bin/clianything providers --all --format json
bin/clianything embeddings --format json
bin/clianything query "select 'channels' as item, count(*) from channels union all select 'models', count(*) from models union all select 'abilities', count(*) from abilities" --format json
sudo -u postgres psql -Atc "select datname from pg_database where datistemplate = false order by datname"
```

### Pre-Mutation Backup Gate

```bash
# Source: PostgreSQL local tools. Run before any production DB mutation.
sudo -u postgres pg_dump -p 8745 -Fc -d newapi \
  -f /home/ubuntu/.backups/router-ai-atius-phase24/newapi-before-phase24.dump
sudo -u postgres pg_dump -p 8745 --data-only --column-inserts -d newapi \
  -t public.channels -t public.models -t public.abilities \
  -f /home/ubuntu/.backups/router-ai-atius-phase24/newapi-catalog-before-phase24.sql
pg_restore -l /home/ubuntu/.backups/router-ai-atius-phase24/newapi-before-phase24.dump >/tmp/newapi-before-phase24.toc
```

### Negative Final-State Queries

```sql
-- Source: Phase 24 constraints.
select model_name from models
where model_name in ('gpt-5.4-1m','gpt-5.5-1m','text-embedding-3-small','text-embedding-3-large');

select id, model_mapping from channels
where id = 5 and (
  model_mapping like '%gpt-5.4-1m%' or model_mapping like '%gpt-5.5-1m%'
);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Python/model-detailed as catalog owner | Go-owned `/v1/models` | Phase 20 / 2026-06-18 docs | DB restore must feed Go catalog, not Python middleware. `[VERIFIED: REQUIREMENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |
| Split MiniMax/DeepSeek channels per protocol | Single provider channel with Go routing semantics | Phase 20 consolidation | Recovery must not leave split active routes. `[VERIFIED: AGENTS.md; tools/clianything.py]` |
| Public `*-batch` embedding alias | One public `embedding-gte-v1` with internal workload classification | 2026-06-26 governor docs | Recovery must preserve `embedding-gte-v1` and `X-Embedding-Workload`, not public batch aliases. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; service/embeddinggovernor/governor.go]` |
| Runtime DB inside Podman Postgres | Host PostgreSQL via PgBouncer | Current live runtime | `container-postgres.service` must not become production traffic path. `[VERIFIED: systemd; pgbouncer config; 24-CONTEXT.md]` |

**Deprecated/outdated:**
- `container-postgres.service` as production DB owner is outdated for this phase. `[VERIFIED: systemd; 24-CONTEXT.md]`
- Active `MiniMax - Anthropic Compatible` and `MiniMax-Highspeed - Anthropic Compatible` channels are legacy split routes. `[VERIFIED: clianything providers --all; AGENTS.md]`
- `gpt-5.4-1m` and `gpt-5.5-1m` are no longer allowed final aliases. `[VERIFIED: 24-CONTEXT.md]`
- `text-embedding-3-small` and `text-embedding-3-large` are no longer allowed final restored rows. `[VERIFIED: 24-CONTEXT.md]`

## Risks And Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Candidate DB misses live token/log deltas | High | Freeze or stop router during final backup/copy; keep `newapi` rollback. `[VERIFIED: clianything query; systemd]` |
| Codex OAuth row restored but token expired | High | Restore channel 5 secret material only from local backup, then run Codex refresh/route validation without printing credentials. `[VERIFIED: channels snapshot; relay/channel/codex/adaptor.go]` |
| Forbidden aliases reappear | High | Negative SQL gates for `gpt-5.4-1m`, `gpt-5.5-1m`, and channel 5 `model_mapping`. `[VERIFIED: 24-CONTEXT.md]` |
| MiniMax appears active in `/v1/models` | Medium | Set MiniMax channel/model/abilities disabled and verify active catalog excludes MiniMax. `[VERIFIED: 24-CONTEXT.md; clianything providers --all]` |
| PgBouncer mapping mismatch | High | Validate `psql` through PgBouncer to `DBRouterAiAtius` before router restart. `[VERIFIED: pgbouncer config]` |
| `clianything` queries old `newapi` after cutover | Medium | Update default DB name or export `CLIANYTHING_DB_NAME=DBRouterAiAtius` in every validation command. `[VERIFIED: tools/clianything.py]` |
| Secret exposure during restore | High | Use redacted inventories only; do not paste raw `channels` rows. `[VERIFIED: AGENTS.md; tools/clianything.py]` |
| Graphify stale after planning artifact changes | Medium | Re-run Graphify status after writing/changing `.planning/` and rebuild if stale/commit-stale. `[VERIFIED: AGENTS.md; .planning/config.json]` |

## Validation Gates Needed Before Any DB Mutation

1. Graphify must be fresh or rebuilt before planning/execution. `[VERIFIED: AGENTS.md; graphify status]`
2. `git status --short` must be reviewed so Phase 24 edits do not overwrite unrelated dirty work. `[VERIFIED: git status]`
3. Current DB target must be re-confirmed: router DSN, PgBouncer mapping, host DB list, and live counts. `[VERIFIED: systemd; pgbouncer config; psql; clianything query]`
4. Create fresh full and catalog-only backups of current `newapi`; verify `pg_restore -l` and table counts before applying any write. `[VERIFIED: pg_dump/pg_restore availability]`
5. Record exact rollback target: old `newapi` DB, old PgBouncer mapping, old router unit DSN, and backup artifact paths. `[VERIFIED: pgbouncer config; systemd]`
6. Generate a transformed catalog restore script and review negative skips for `gpt-5.4-1m`, `gpt-5.5-1m`, `text-embedding-3-small`, and `text-embedding-3-large`. `[VERIFIED: backups/clianything SQL; 24-CONTEXT.md]`
7. Verify candidate `DBRouterAiAtius` offline before router cutover: required rows present, forbidden rows absent, MiniMax disabled, DeepSeek active, Codex active, channel 9 preserved. `[VERIFIED: clianything query; backups/clianything SQL]`
8. Verify PgBouncer can connect to `DBRouterAiAtius` before changing router DSN. `[VERIFIED: pgbouncer config]`
9. Ensure authenticated validation tokens are available in the operator shell or secret store; research shell had `ATIUS_ROUTER_TOKEN` and `ATIUS_ROUTER_ADMIN_TOKEN` unset. `[VERIFIED: shell env check]`
10. Confirm no command output includes `channels.key`, token values, OAuth JSON, or secret env values. `[VERIFIED: AGENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PHASE-24-CANONICAL-HOST-DB | Runtime must use one canonical host PostgreSQL DB through PgBouncer, with intended production DB name and rollback-safe migration. `[VERIFIED: REQUIREMENTS.md]` | Current drift is `newapi`; recommended final is `DBRouterAiAtius` via copy/restore, PgBouncer mapping, unit DSN update, and preserved `newapi` rollback. `[VERIFIED: psql; pgbouncer config; systemd]` |
| PHASE-24-CATALOG-RESTORE | Recover full known-good router catalog subject to user exclusions. `[VERIFIED: REQUIREMENTS.md]` | 2026-07-01 snapshots identify channel/model/ability templates; transform rules skip forbidden aliases/embeddings. `[VERIFIED: backups/clianything SQL; 24-CONTEXT.md]` |
| PHASE-24-PROVIDER-CONSOLIDATION | DeepSeek and MiniMax must end in single Go-owned consolidated channels; MiniMax disabled; DeepSeek V4 Flash/Pro restored. `[VERIFIED: REQUIREMENTS.md]` | Current DB has split MiniMax active channels and DeepSeek rows; final reconciliation disables split MiniMax and restores active DeepSeek channel `2`. `[VERIFIED: clianything providers --all; channels snapshot]` |
| PHASE-24-EMBEDDING-GOVERNOR-PRESERVE | Preserve `embedding-gte-v1` and Go-native governor path. `[VERIFIED: REQUIREMENTS.md]` | Current channel 9/model 21/ability are present; code and env show governor applies to `embedding-gte-v1`. `[VERIFIED: clianything embeddings; service/embeddinggovernor/governor.go; podman inspect redacted]` |
| PHASE-24-CUTOVER-ROLLBACK | Recovery requires backups, strict validation, docs reconciliation, and named rollback path. `[VERIFIED: REQUIREMENTS.md]` | Research defines backup gates, candidate DB, old `newapi` rollback, user systemd restart, strict CLI/API validation, and doc updates. `[VERIFIED: docs/PODMAN.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Host PostgreSQL | DB backup/restore/candidate | yes | server accepts on `127.0.0.1:8745` | None; required. `[VERIFIED: pg_isready]` |
| PgBouncer | Runtime DB path | yes | `1.25.2` | None; final path requires it. `[VERIFIED: pgbouncer --version]` |
| `psql` | DB probes | yes | `18.4` | None; required. `[VERIFIED: psql --version]` |
| `pg_dump` | backups | yes | `17.10` | None; required. `[VERIFIED: pg_dump --version]` |
| `pg_restore` | dump validation/restore | yes | `17.10` | None; required. `[VERIFIED: pg_restore --version]` |
| Podman | Runtime inventory | yes | `4.9.3` | None for runtime. `[VERIFIED: podman --version]` |
| user systemd | Router restart/cutover | yes | `255` | None; docs forbid direct routine `podman restart`. `[VERIFIED: systemctl --version; docs/PODMAN.md]` |
| `bin/clianything` | Catalog inventory/backup | yes | local script | Direct `psql` read-only queries if CLI fails. `[VERIFIED: bin/clianything --help]` |
| Go | Unit tests | yes | `go1.25.1 linux/arm64` | None for Go tests. `[VERIFIED: /usr/local/go/bin/go version]` |
| Python | Smoke/CLI tests | yes | `3.12.3` | None for Python smoke scripts. `[VERIFIED: python3 --version]` |
| `curl` / `jq` | API validation | yes | curl `8.5.0`, jq `1.7` | Python HTTP script if needed. `[VERIFIED: curl --version; jq --version]` |
| Auth tokens | Authenticated API gates | no in this shell | `ATIUS_ROUTER_TOKEN=unset`, `ATIUS_ROUTER_ADMIN_TOKEN=unset` | Operator must provide tokens at execution/validation. `[VERIFIED: shell env check]` |

**Missing dependencies with no fallback:**
- Authenticated router tokens are missing from this shell, blocking authenticated `/v1/models`, `/v1/embeddings`, GPT, and DeepSeek runtime validation in research. `[VERIFIED: shell env check]`

**Missing dependencies with fallback:**
- None found for local backup/restore tooling. `[VERIFIED: environment availability probes]`

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` with `github.com/stretchr/testify v1.11.1`; Python `unittest` for CLI tests. `[VERIFIED: go.mod; tests/test_clianything.py]` |
| Config file | No central Go test config; use package-level `go test` commands. `[VERIFIED: repo file scan]` |
| Quick run command | `/usr/local/go/bin/go test ./service/modelcatalog ./relay/channel/codex ./service/embeddinggovernor ./relay -count=1` `[VERIFIED: repo tests]` |
| Full suite command | `/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./relay/channel/codex ./relay/channel/minimax ./relay/channel/deepseek ./service/embeddinggovernor ./relay -count=1 && python3 -m unittest tests.test_clianything -v` `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md; repo tests]` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| PHASE-24-CANONICAL-HOST-DB | CLI/runtime points at canonical host DB after cutover | integration/manual | `CLIANYTHING_DB_NAME=DBRouterAiAtius bin/clianything status --strict` | yes: `tools/clianything.py` `[VERIFIED: tools/clianything.py]` |
| PHASE-24-CATALOG-RESTORE | `/v1/models` shape/order and internal pricing fields remain protected | unit/integration | `/usr/local/go/bin/go test ./controller ./service/modelcatalog -run 'TestListModels|TestModelCatalog' -count=1` | yes `[VERIFIED: repo tests]` |
| PHASE-24-PROVIDER-CONSOLIDATION | MiniMax/DeepSeek consolidated routing helpers remain protected | unit | `/usr/local/go/bin/go test ./relay/channel/minimax ./relay/channel/deepseek ./relay/common -count=1` | yes `[VERIFIED: repo tests]` |
| PHASE-24-EMBEDDING-GOVERNOR-PRESERVE | Governor gates only `embedding-gte-v1` and preserves workload semantics | unit | `/usr/local/go/bin/go test ./service/embeddinggovernor ./relay -run 'TestGovernor|TestEmbedding' -count=1` | yes `[VERIFIED: repo tests]` |
| PHASE-24-CUTOVER-ROLLBACK | CLI dry-run/backup/status supports safe operational flow | unit/integration | `python3 -m unittest tests.test_clianything -v` | yes `[VERIFIED: repo tests]` |

### Sampling Rate

- **Per task commit:** quick Go packages plus relevant CLI query dry-run. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`
- **Per wave merge:** full suite command above plus `bin/clianything status --strict`. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`
- **Phase gate:** full suite green, authenticated `/v1/models`, authenticated `/v1/embeddings`, representative GPT/Codex and DeepSeek checks, MiniMax negative visibility check, and rollback artifact verification. `[VERIFIED: REQUIREMENTS.md]`

### Wave 0 Gaps

- [ ] Add or script a deterministic catalog diff check for Phase 24 final row invariants. `[VERIFIED: current repo scan]`
- [ ] Add an execution-only checklist to validate PgBouncer `DBRouterAiAtius` mapping before router restart. `[VERIFIED: pgbouncer config]`
- [ ] Ensure auth tokens are available to the executor for live UAT; research shell did not have them. `[VERIFIED: shell env check]`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes | Preserve token/auth tables from current DB; authenticated validation must use env/secret store without printing tokens. `[VERIFIED: clianything query; AGENTS.md]` |
| V3 Session Management | yes | Preserve current `users`, `tokens`, session secret env, and avoid full-dump rollback that loses current tokens. `[VERIFIED: clianything query; env var name audit]` |
| V4 Access Control | yes | Keep `/v1/models` unauthenticated response at `401`; do authenticated checks only with valid tokens. `[VERIFIED: curl; REQUIREMENTS.md]` |
| V5 Input Validation | yes | Use existing Go handlers/tests; no new public input parser is required for DB recovery. `[VERIFIED: repo tests]` |
| V6 Cryptography | yes | Do not hand-roll credential handling; preserve existing channel/OAuth secret material and Codex adaptor behavior. `[VERIFIED: relay/channel/codex/adaptor.go; AGENTS.md]` |

### Known Threat Patterns for Router DB Recovery

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Secret leakage from `channels.key` or OAuth JSON | Information Disclosure | Redacted inventory only; do not paste raw channel rows; restore from local files. `[VERIFIED: AGENTS.md; backups/clianything SQL]` |
| Wrong DB target after PgBouncer cutover | Tampering / Availability | Validate PgBouncer mapping and router DSN before restart; keep old `newapi` rollback. `[VERIFIED: pgbouncer config; systemd]` |
| Reintroducing forbidden public models | Elevation of privilege / Policy bypass | Negative SQL and authenticated `/v1/models` checks for forbidden IDs. `[VERIFIED: 24-CONTEXT.md]` |
| Lost token/log records from stale full restore | Repudiation / Availability | Preserve current `newapi` as source for non-catalog state and rollback. `[VERIFIED: clianything query; pg_restore]` |
| Reintroducing Python/model-detailed owner | Tampering / Availability | Keep full-Go route and provider adaptors as source of truth. `[VERIFIED: REQUIREMENTS.md; docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]` |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No assumption is required for the DB-name recommendation; it is an inference from project state and unit evidence, not a user-provided locked name. `[VERIFIED: STATE.md; container-postgres.service]` | Recommended Canonical DB Naming/Cutover Strategy | If Giovanni intended a different canonical name, the planner must pause before mutation. |

## Open Questions

1. **Confirm final canonical DB name before mutation**
   - What we know: `DBRouterAiAtius` is recorded in STATE and the historical Postgres unit, while current runtime uses drifted `newapi`. `[VERIFIED: STATE.md; container-router-ai-atius.service; container-postgres.service]`
   - What's unclear: whether Giovanni wants the exact mixed-case `DBRouterAiAtius` or a new canonical host DB name. `[VERIFIED: 24-CONTEXT.md]`
   - Recommendation: plan should use `DBRouterAiAtius` but include a human checkpoint before the first DB write. `[VERIFIED: research synthesis]`

2. **Codex OAuth freshness after channel 5 restore**
   - What we know: the snapshot contains channel 5 secret material and the Go adaptor supports Codex responses/embeddings through type `57`. `[VERIFIED: channels snapshot; relay/channel/codex/adaptor.go]`
   - What's unclear: whether restored OAuth material will refresh/pass live upstream validation at execution time. `[VERIFIED: shell env/token limitations]`
   - Recommendation: restore without printing secrets, then run Codex refresh and representative GPT route checks as a hard gate. `[VERIFIED: docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md]`

3. **Disabled historical MiniMax/split rows: keep or delete**
   - What we know: final active design requires one MiniMax channel and MiniMax disabled. `[VERIFIED: 24-CONTEXT.md]`
   - What's unclear: whether historical split channel rows `3` and `4` should remain disabled for audit/log history or be removed later. `[VERIFIED: clianything providers --all]`
   - Recommendation: disable rather than delete during recovery; consider cleanup only after validation. `[VERIFIED: research synthesis]`

## Sources

### Primary (HIGH confidence)

- `.planning/ROADMAP.md` - Phase 24 goal, dependencies, planned waves. `[VERIFIED: codebase read]`
- `.planning/STATE.md` - current phase and historical DB identity. `[VERIFIED: codebase read]`
- `.planning/REQUIREMENTS.md` - Phase 24 requirements and Phase 20 invariants. `[VERIFIED: codebase read]`
- `.planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/24-CONTEXT.md` - user constraints and forensic findings. `[VERIFIED: codebase read]`
- `backups/clianything/20260701_184735_channels.sql`, `models.sql`, `abilities.sql` - known-good catalog snapshots. `[VERIFIED: codebase read]`
- `/home/ubuntu/.backups/router-ai-atius-incident-20260703T231027-0300/newapi-before.fix.dump` - rollback/diff custom dump inspected with `pg_restore`. `[VERIFIED: pg_restore]`
- `/home/ubuntu/.config/systemd/user/container-router-ai-atius.service` and `container-postgres.service` - active/legacy runtime units. `[VERIFIED: systemd unit read]`
- `/etc/pgbouncer/pgbouncer.ini` selected non-secret fields - PgBouncer DB mapping and pool settings. `[VERIFIED: pgbouncer config]`
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` and `docs/PODMAN.md` - operational runtime and validation docs. `[VERIFIED: codebase read]`
- `tools/clianything.py` - CLI DB defaults, redaction, backup, provider and embedding commands. `[VERIFIED: codebase read]`
- `service/embeddinggovernor/governor.go`, `relay/embedding_handler.go`, `service/modelcatalog/catalog.go`, `relay/channel/codex/adaptor.go`, `service/openaicompat/policy.go` - Go runtime behavior. `[VERIFIED: codebase read]`
- Live read-only commands: `clianything`, `psql`, `pg_isready`, `systemctl`, `podman inspect`, `curl`. `[VERIFIED: local command output]`

### Secondary (MEDIUM confidence)

- GBrain query surfaced related router history but did not supply decisive Phase 24 facts used as primary evidence. `[VERIFIED: gbrain query]`

### Tertiary (LOW confidence)

- None used. `[VERIFIED: no web search/no training-only claims]`

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - versions and runtime paths were verified locally. `[VERIFIED: environment availability probes]`
- Architecture: HIGH - Go/PgBouncer/systemd paths were verified from code, docs, and live state. `[VERIFIED: codebase read; systemd; pgbouncer config]`
- Pitfalls: HIGH - based on direct diffs between current DB, 2026-07-01 snapshots, and 2026-07-03 dump. `[VERIFIED: clianything query; pg_restore; backups/clianything SQL]`

**Research date:** 2026-07-04
**Valid until:** 2026-07-11, because runtime DB state and provider credentials can change quickly. `[VERIFIED: current_date; runtime state]`
