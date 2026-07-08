---
phase: 24-router-db-catalog-recovery-and-canonical-host-db
plan: "03"
subsystem: api
tags:
  - codex
  - catalog
  - docs
  - pricing
  - governor
dependency_graph:
  requires:
    - 24-02-SUMMARY.md
    - 24-CONTEXT.md
    - 24-RESEARCH.md
  provides:
    - final Codex no-alias contract
    - final provider/channel policy docs
    - explicit governor preservation wording
  affects:
    - Phase 24 Plan 04
    - controller/model_list_test.go
    - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md
    - docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md
tech_stack:
  added: []
  patterns:
    - final-state docs override historical validation notes
    - representative model-list fixtures mirror restored DB contract
key_files:
  created:
    - .planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/24-03-SUMMARY.md
  modified:
    - controller/model_list_test.go
    - setting/ratio_setting/model_ratio.go
    - docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md
    - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md
key_decisions:
  - "The final restored Codex contract does not publish `gpt-5.4-1m` or `gpt-5.5-1m`; `gpt-5.4` remains the default long-context model in docs and tests."
  - "DeepSeek is documented as the single active consolidated provider, while MiniMax is restored but disabled in the final state."
  - "Governor preservation stays explicit in docs with `embedding-gte-v1` as the only governed public embedding alias and `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` unchanged."
patterns-established:
  - "When runtime history contradicts final restore state, mark the old alias/provider evidence as historical instead of treating it as current contract."
requirements-completed:
  - PHASE-24-CATALOG-RESTORE
  - PHASE-24-PROVIDER-CONSOLIDATION
  - PHASE-24-EMBEDDING-GOVERNOR-PRESERVE
metrics:
  started_at: 2026-07-04T04:47:00Z
  completed_at: 2026-07-04T04:59:24Z
status: complete
---

# Phase 24 Plan 03: Final Codex contract without `-1m` aliases, restored provider policy, and explicit governor preservation

The public contract now treats `gpt-5.4` as the default long-context Codex model, removes final-state expectations for `gpt-5.4-1m` and `gpt-5.5-1m`, documents DeepSeek active plus MiniMax restored-but-disabled, and keeps `embedding-gte-v1` as the only governed public embedding alias.

## Outcomes

- Updated [controller/model_list_test.go](/home/ubuntu/GitHub/containers/router-ai-atius/controller/model_list_test.go) so the representative `/v1/models` fixture and assertions match the restored final contract: no `gpt-5.4-1m` or `gpt-5.5-1m`, `gpt-5.4` present as the long-context default surface, and public payload still hides internal pricing provenance fields.
- Updated [setting/ratio_setting/model_ratio.go](/home/ubuntu/GitHub/containers/router-ai-atius/setting/ratio_setting/model_ratio.go) to remove hardcoded ratio/completion entries for the forbidden `-1m` aliases while preserving the base Codex model pricing.
- Updated [OPENAI-CODEX-PROVIDER-1M-CONTEXT.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md) so current-state sections describe the final Phase 24 contract and label the 2026-07-01 alias material as historical evidence only.
- Updated [MANUAL-OPERACAO-ROUTER-AI-ATIUS.md](/home/ubuntu/GitHub/containers/router-ai-atius/docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md) so the operational policy explicitly says `OpenAI - Codex` active, DeepSeek consolidated and active, MiniMax consolidated but disabled, and the Go governor path remains `embedding-gte-v1` only.

## Verification

`rg -n "gpt-5.4|gpt-5.5|1m" service/modelcatalog/catalog.go controller/model_list_test.go setting/ratio_setting/model_ratio.go docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`
Result: passed. The remaining `-1m` matches are historical notes in the Codex doc plus explicit "must not exist" statements; test and ratio code no longer require the aliases.

`rg -n "OpenAI - Codex|DeepSeek|MiniMax|restaurad|desabilitad|embedding-gte-v1" docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`
Result: passed. The manual now states `OpenAI - Codex` active, DeepSeek active/consolidated, MiniMax restored-but-disabled, and `embedding-gte-v1` preserved.

`rg -n "EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1|only governed public embedding alias|embedding-gte-v1" docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`
Result: passed. Both docs keep `embedding-gte-v1` explicit and preserve `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1`.

`PATH=/usr/local/go/bin:$PATH go test ./controller ./setting/ratio_setting -run 'TestListModelsPayloadShapeAndPublicFields|TestListModelsRepresentativeOrder|TestListModelsCodexContractAfterPhase24Restore' -count=1`
Result: passed.

`node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify build .` followed by `graphify status`
Result: build command returned the usual `spawn_agent` envelope, but `graphify status` remained `stale: true` and `commit_stale: true` at commit `956701a`. This was recorded as a toolchain/runtime limitation and did not block the narrow file-level verification.

## Commits

None. The scoped files were already dirty before this plan and a clean task commit would have bundled unrelated pre-existing hunks. Per the user constraint, the plan was left as working-tree changes only.

## Decisions Made

- Keep `gpt-5.4` as the documented long-context default without reintroducing any public `-1m` alias in code, tests, or docs.
- Treat the 1M alias material as historical evidence instead of deleting it wholesale, so rollback/forensics remain available without contradicting the final restored contract.
- Make governor preservation explicit in both docs so Phase 24-04 validation can assert `embedding-gte-v1` and `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` directly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Graphify rebuild does not refresh status in this checkout**
- **Found during:** verification
- **Issue:** `graphify build .` returns a `spawn_agent` envelope, but `graphify status` stays at the old `built_at_commit` and remains `commit_stale: true`.
- **Fix:** Continued with focused file reads, `rg`, and targeted `go test` verification instead of trusting Graphify freshness.
- **Files modified:** none
- **Verification:** repeated `graphify status` after `graphify build .`
- **Committed in:** none

**Total deviations:** 1 auto-fixed/worked-around
**Impact on plan:** No scope creep. The workaround only affected verification routing and did not change runtime code.

## Issues Encountered

- The target files were pre-dirty before execution, so atomic per-task commits were intentionally skipped to avoid capturing unrelated hunks.

## Next Phase Readiness

- Plan 24-04 can now validate cutover/runtime state against the final doc contract instead of the older alias experiment.
- Remaining risk: docs still preserve historical `-1m` evidence by design, so validators should distinguish "historical 2026-07-01" sections from "contrato final restaurado" sections.

## Self-Check: PASSED

- Found `controller/model_list_test.go`, `setting/ratio_setting/model_ratio.go`, `docs/OPENAI-CODEX-PROVIDER-1M-CONTEXT.md`, `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`, and `24-03-SUMMARY.md`.
- No task commits were expected because commit creation was intentionally skipped due to pre-dirty target files.
