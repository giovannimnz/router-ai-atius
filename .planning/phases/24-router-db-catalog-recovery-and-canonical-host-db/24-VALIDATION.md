---
phase: 24
phase_slug: router-db-catalog-recovery-and-canonical-host-db
created: 2026-07-04
status: planned
---

# Phase 24 Validation Strategy

## Validation Architecture

Phase 24 has to prove four things:

1. the recovery source is correct;
2. the canonical host DB name/cutover is safe;
3. the router catalog is fully restored with the requested transforms;
4. runtime/docs/CLI all agree after cutover.

## Required Gates

| Gate | Type | Evidence |
|---|---|---|
| Graphify | preflight | `graphify status` fresh before and after planning/execution |
| Runtime baseline | preflight | `bin/clianything status --strict`, `providers --all`, current DB counts |
| Backup | data | fresh full dump + catalog-only dump + `pg_restore -l` verification |
| Source selection | data | diff between live DB, 2026-07-03 pre-fix dump, and 2026-07-01 catalog snapshots |
| Candidate DB | restore | candidate DB exists, counts sane, channel/model/ability rows match transformed target |
| Catalog transform | review | no `gpt-5.4-1m`, no `gpt-5.5-1m`, no `text-embedding-3-*`, no channel 5 `-1m` mapping |
| Runtime cutover | manual | router unit points to canonical host DB name via PgBouncer |
| Public API | runtime | authenticated `/v1/models`, `/v1/chat/completions`, `/v1/embeddings` pass |
| Provider policy | runtime | DeepSeek one active channel; MiniMax one restored-but-disabled channel |
| Governor | runtime | `embedding-gte-v1` still governed and returns 768 dims |
| Docs parity | docs | `PODMAN.md`, manual, and Codex docs reflect final DB path/catalog state |

## Blockers

- No DB mutation without fresh validated backups.
- No runtime cutover without PgBouncer mapping for the final canonical DB name.
- No phase completion if `OpenAI - Codex` is still absent from the live catalog.
- No phase completion if DeepSeek is not restored active.
- No phase completion if MiniMax is restored active instead of restored-but-disabled.
- No phase completion if `embedding-gte-v1` fails after catalog restore.

## Success Markers

- `channel 5 = OpenAI - Codex` exists again in the active DB.
- `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`, and `gpt-5.3-codex-spark` reappear in active catalog/routing.
- `gpt-5.4-1m` and `gpt-5.5-1m` are absent.
- `text-embedding-3-small` and `text-embedding-3-large` are absent.
- DeepSeek V4 Flash and Pro are active.
- MiniMax consolidated channel exists but is disabled, and related models are disabled.
- `embedding-gte-v1` remains active and governed.
- runtime uses host PgBouncer plus final canonical DB name.
