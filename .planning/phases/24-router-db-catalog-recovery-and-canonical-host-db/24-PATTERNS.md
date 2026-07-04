---
status: planned
phase: 24-router-db-catalog-recovery-and-canonical-host-db
created: 2026-07-04
updated: 2026-07-04
---

# Phase 24 Patterns — Router DB/Catalog Recovery

## Planning Shape To Reuse

- Use multi-wave planning when the work mixes runtime, DB, catalog, and docs.
- Keep recovery split into:
  - freeze and backup
  - candidate restore and transformation
  - provider/catalog reconciliation
  - cutover and rollback validation
- Preserve one clear source of truth per wave:
  - runtime truth
  - backup truth
  - known-good catalog snapshot
  - final validation truth

## Recovery Patterns To Reuse

### Pattern 1: Backup Before Mutation

- Take a fresh full DB dump before touching the active database.
- Take a catalog-only export for `channels`, `models`, `abilities`, and `tokens`.
- Verify every dump with `pg_restore -l` or equivalent before proceeding.

### Pattern 2: Transform Snapshot, Do Not Replay Blindly

- Use known-good SQL snapshots as templates.
- Apply user-mandated transforms before restore:
  - skip `gpt-5.4-1m`
  - skip `gpt-5.5-1m`
  - skip `text-embedding-3-small`
  - skip `text-embedding-3-large`
  - keep DeepSeek active
  - keep MiniMax restored but disabled

### Pattern 3: Preserve Live Non-Catalog Data

- Do not overwrite current `users`, `tokens`, `logs`, or recent operational rows with an older full dump unless rollback is required.
- Treat current live DB as the source of truth for mutable operational data.
- Treat the 2026-07-01 catalog snapshots as the source of truth for the missing router catalog.

### Pattern 4: Full-Go Runtime Remains Canonical

- Do not reintroduce Python/model-detailed ownership for `/v1/models` or embeddings.
- Validate every restored provider/model through the Go path only.
- Keep `embedding-gte-v1` under the Go governor path.

## Validation Patterns To Reuse

- `bin/clianything status --strict`
- `bin/clianything providers --all`
- authenticated `GET /v1/models`
- authenticated `POST /v1/chat/completions`
- authenticated `POST /v1/embeddings`
- `python3 scripts/smoke-embeddings.py`
- focused Go tests for catalog/routing/policy files

## Files Phase 24 Should Touch

- `.planning/ROADMAP.md`
- `.planning/STATE.md`
- `.planning/REQUIREMENTS.md`
- `.planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/*`
- `tools/clianything.py`
- `tests/test_clianything.py`
- `service/modelcatalog/catalog.go`
- `controller/model_list_test.go`
- `setting/ratio_setting/model_ratio.go`
- `docs/PODMAN.md`
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`
- `docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`
- live operational targets:
  - `/home/ubuntu/.config/systemd/user/container-router-ai-atius.service`
  - PgBouncer config / host PostgreSQL DBs

## Anti-Patterns To Avoid

- Restoring only embeddings and calling the recovery complete.
- Repointing runtime to a different DB without a catalog diff and validation gate.
- Blindly replaying old SQL that reintroduces forbidden `-1m` aliases or Codex embeddings rows.
- Keeping split active MiniMax/DeepSeek channels in the final state.
- Changing DB name/path without retaining rollback access to the previous DB target.
