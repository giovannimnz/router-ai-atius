---
status: planned
phase: 24-router-db-catalog-recovery-and-canonical-host-db
created: 2026-07-04
updated: 2026-07-04
---

# Phase 24 Context — Router DB/Catalog Recovery On Canonical Host DB

## Goal

Recover the full `router-ai-atius` runtime/catalog on the canonical host PostgreSQL path via PgBouncer, preserving the current host-based DB approach while fixing the post-2026-07-02 drift that left the runtime pointed at an incomplete database.

This phase is not just an embeddings repair. It must recover the whole router catalog/routing surface needed by production:

- `OpenAI - Codex`
- GPT/Codex models that were present in the known-good 2026-07-01 catalog
- DeepSeek V4 Flash and Pro
- `embedding-gte-v1`
- provider/channel consolidation invariants
- the Go embedding governor path

## User-Mandated Final State

### Restore

- Restore the provider/channel for `OpenAI - Codex`.
- Restore DeepSeek provider/channel and models:
  - `deepseek-v4-flash`
  - `deepseek-v4-pro`
- Restore MiniMax provider/channel and models, but leave them disabled in the final state.
- Preserve `embedding-gte-v1` as the governed public embedding alias.
- Keep the DB on the host via PgBouncer.

### Do Not Restore / Do Not Recreate

- Do not recreate:
  - `gpt-5.4-1m`
  - `gpt-5.5-1m`
- Do not restore disabled embeddings catalog rows:
  - `text-embedding-3-small`
  - `text-embedding-3-large`
- Do not restore `channels.model_mapping` entries on channel 5 for:
  - `gpt-5.5-1m -> gpt-5.5`
  - `gpt-5.4-1m -> gpt-5.4`

### GPT/Codex Policy

- `gpt-5.4` becomes the default long-context Codex model and is treated as approximately `1050000` max tokens in the intended restored contract.
- `gpt-5.5` does not get 1M context in the Codex path and must stay without a `-1m` alias.

### Provider Consolidation

- DeepSeek must end with only one channel and automatic routing for OpenAI/Anthropic semantics in Go.
- MiniMax must end with only one channel and automatic routing for OpenAI/Anthropic semantics in Go.
- Final state for MiniMax is restored-but-disabled:
  - channel disabled
  - related models disabled

## Forensic Findings Already Confirmed

- On `2026-07-02 08:11:13 -03`, the live unit `container-router-ai-atius.service` was rewritten and now points to:
  - `SQL_DSN=postgresql://admin:${POSTGRES_PASSWORD}@10.1.1.1:6432/newapi`
- `container-postgres.service` became inactive on `2026-07-02 08:10:10 -03`.
- The live host PostgreSQL at `127.0.0.1:8745` / PgBouncer `10.1.1.1:6432` currently has database `newapi` and does not have `DBRouterAiAtius`.
- The current live `newapi` DB still has logs/users/tokens, but its router catalog is incomplete:
  - active channels now: `1,2,3,4,9`
  - no `channel 5 = OpenAI - Codex`
  - no GPT/Codex rows in `models`
  - no GPT/Codex rows in `abilities`
- The 2026-07-03 repair restored only the minimal embeddings path:
  - `embedding-gte-v1`
  - channel `9`
  - ability for `embedding-gte-v1`
  - GBrain-related token compatibility

## Recovery Sources Identified

### Highest-value local catalog snapshot

- `backups/clianything/20260701_184735_channels.sql`
- `backups/clianything/20260701_184735_models.sql`
- `backups/clianything/20260701_184735_abilities.sql`

This is the best local known-good source for the modern `OpenAI - Codex` + GPT catalog before the drift.

### Additional local DB backups

- `/home/ubuntu/.backups/router-ai-atius-incident-20260703T231027-0300/newapi-before.fix.dump`
- `/home/ubuntu/.backups/router-ai-atius-incident-20260703T231027-0300/newapi-router-catalog-before.sql`
- `/home/ubuntu/.backups/srv1-pgbouncer-newapi-20260613-085834/newapi.pgcustom`
- `data/pg_backup/newapi_backup_20260531_235230.dump`

These are useful for rollback, diffing, and older fallback states, but the 2026-07-01 catalog snapshots are the strongest source for the desired GPT/Codex state.

## Prior Validated Embedding State

The Codex session `019f0612-f723-7dd0-a0f6-76de4b694b51` validated the intended embeddings baseline on 2026-06-26:

- `embedding-gte-v1` was the only active public embedding model.
- `Local TEI - GTE Embeddings` was channel `9`.
- `embedding-gte-v1` had one enabled ability on channel `9`.
- `embo-01`, `text-embedding-3-small`, and `text-embedding-3-large` were present only as disabled/historical rows.
- The Go-native governor path was active with:
  - `EMBEDDING_GOVERNOR_ENABLED=true`
  - `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1`
  - `EMBEDDING_GOVERNOR_BATCH_MODELS=`
- Authenticated `/v1/models` exposed `embedding-gte-v1` and did not expose `embedding-pt-v1` or a public batch alias.
- Public `POST /v1/embeddings` with `embedding-gte-v1` returned `768` dimensions.

This session is important because it distinguishes the desired embeddings baseline from the later partial repair on 2026-07-03. Phase 24 must preserve this 2026-06-26 governed embeddings state while restoring the missing GPT/Codex catalog and canonical DB identity.

## Constraints

- No secret printing in docs, plans, or summaries.
- No destructive mutation without a fresh validated backup.
- No assumption that old docs still describe the live runtime correctly.
- Preserve the full-Go runtime path.
- Preserve the Go-native embedding governor path.
- Preserve the host PgBouncer approach, but rename/migrate the DB to the correct intended name.

## Questions This Phase Must Settle

1. What is the exact canonical final DB name:
   - keep `newapi`
   - or move back to `DBRouterAiAtius`
   - or create a new canonical name and repoint router + PgBouncer consistently

2. Which rows should be restored exactly from 2026-07-01 versus transformed during recovery:
   - `OpenAI - Codex`
   - GPT/Codex rows without `-1m`
   - DeepSeek consolidated active
   - MiniMax consolidated but disabled

3. What is the safest mutation order to avoid partial recovery:
   - DB backup
   - restore to staging/candidate DB or in-place with transaction/script
   - provider/channel/model/ability reconciliation
   - token verification
   - runtime cutover
   - docs reconciliation

## Expected Deliverables

- A plan set that recovers the full router catalog, not just embeddings.
- A DB naming/cutover strategy that keeps the host PgBouncer architecture.
- A restore plan that explicitly skips `gpt-5.4-1m`, `gpt-5.5-1m`, `text-embedding-3-small`, and `text-embedding-3-large`.
- Validation artifacts proving the final runtime state.
