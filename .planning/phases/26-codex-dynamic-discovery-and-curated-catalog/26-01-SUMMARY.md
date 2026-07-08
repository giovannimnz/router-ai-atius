---
phase: 26-codex-dynamic-discovery-and-curated-catalog
plan: "01"
subsystem: codex-catalog
tags: [codex, catalog, discovery, scheduler, validation]
requires:
  - phase: 24-router-db-catalog-recovery-and-canonical-host-db
    provides: Canonical Go-owned `/v1/models` contract and Codex provider baseline
  - phase: 25-embedding-governor-auto-workload-inference
    provides: Current build/test runtime discipline and guarded validation pattern
provides:
  - Dynamic account-aware Codex discovery with local persistence and fallback snapshot
  - Candidate state machine with validation gate before public promotion
  - Promoted Codex metadata overlay for curated `/v1/models`
  - Daily `04:00` Codex catalog sync task with local default-model policy preserved
affects: [controller, service, model, dto, scheduler, operator-validation]
tech-stack:
  added: [Go persistence models, Codex discovery service, scheduler task]
  patterns:
    - "Run Go validation inside `./scripts/podman-admin.sh profile-run` but use real toolchain binaries inside the shell."
    - "Keep Codex discovery asynchronous; `/v1/models` remains local and deterministic."
key-files:
  created:
    - model/codex_catalog.go
    - service/codex_catalog.go
    - service/codex_catalog_task.go
    - service/codex_catalog_test.go
    - .planning/phases/26-codex-dynamic-discovery-and-curated-catalog/26-01-SUMMARY.md
  modified:
    - controller/codex_fetch_models.go
    - controller/channel.go
    - controller/channel_upstream_update.go
    - controller/model.go
    - controller/codex_fetch_models_test.go
    - controller/model_list_test.go
    - dto/pricing.go
    - main.go
    - model/main.go
key-decisions:
  - "Codex discovery now queries `/backend-api/codex/models?client_version=...` using the active OAuth/account context, but request-time `/v1/models` never calls upstream."
  - "Promotion is gated by a minimal live Responses request that must return only `Ok`."
  - "The public catalog keeps local policy precedence, including default model `gpt-5.4` and denylisting `gpt-5.4-1m` / `gpt-5.5-1m`."
patterns-established:
  - "When global build guards wrap `go`/`gcc`, invoke `profile-run` outside and real binaries (`/usr/local/go/bin/go`, `/usr/bin/gcc`) inside."
  - "Use an isolated `GOCACHE` for guarded Go builds on this host to avoid host cleanup races against `~/.cache/go-build`."
requirements-completed:
  - PHASE-26-LOCAL-CURATED-V1-MODELS
  - PHASE-26-DYNAMIC-CODEX-DISCOVERY
  - PHASE-26-MULTI-SOURCE-ENRICHMENT
  - PHASE-26-CANDIDATE-PROBE-GATE
  - PHASE-26-CODEX-METADATA-ENRICHMENT
  - PHASE-26-DAILY-SCHEDULED-SYNC
  - PHASE-26-DEFAULT-MODEL-GUARD
coverage:
  - id: D1
    description: Admin/model-update discovery uses account-aware Codex upstream lookup with local fallback snapshot
    requirement: PHASE-26-DYNAMIC-CODEX-DISCOVERY
    verification:
      - kind: unit
        ref: "controller.TestFetchDynamicCodexModelIDsUsesAccountAwareDiscovery"
        status: pass
    human_judgment: false
  - id: D2
    description: Promoted Codex entries enrich `/v1/models` with context_window, endpoint preference, and deterministic local metadata
    requirement: PHASE-26-CODEX-METADATA-ENRICHMENT
    verification:
      - kind: unit
        ref: "controller.TestListModelsAppliesPromotedCodexCatalogMetadata"
        status: pass
      - kind: unit
        ref: "service.TestMergeCodexCatalogMetadataPrefersOverrideAndKeepsContextWindowGroup"
        status: pass
    human_judgment: false
  - id: D3
    description: Daily scheduler and default-model policy are wired without breaking package/root compilation under the guarded build profile
    requirement: PHASE-26-DAILY-SCHEDULED-SYNC
    verification:
      - kind: unit
        ref: "service.TestNextCodexCatalogSyncDelay"
        status: pass
      - kind: unit
        ref: "service.TestPrioritizeCodexDefaultModel"
        status: pass
      - kind: other
        ref: "`go test . -run '^$' -count=1 -vet=off` under `profile-run`"
        status: pass
    human_judgment: false
duration: 1 session
completed: 2026-07-08
status: complete
---

# Phase 26 Plan 01 Summary

**The Codex catalog pipeline now discovers models dynamically from the active account, persists candidate state locally, validates candidates with a live `Ok` probe, and only then promotes them into the deterministic Go-owned catalog.**

## Performance

- **Duration:** 1 session
- **Completed:** 2026-07-08
- **Tasks:** 6 workstreams in one implementation plan
- **Files modified:** 9
- **Files created:** 5

## Accomplishments

- Added persistent Codex catalog tables for snapshots and candidate state in `model/codex_catalog.go`, wired through `model/main.go` migrations.
- Implemented the core pipeline in `service/codex_catalog.go`: dynamic discovery, metadata merge, local override policy, probe validation, promoted-model persistence, and channel model sync.
- Added a daily `04:00` scheduler in `service/codex_catalog_task.go` and wired it from `main.go`.
- Switched admin Codex fetch-models and channel upstream update logic to use dynamic discovery with local fallback instead of static-only lookup.
- Extended `/v1/models` payload generation to expose promoted Codex `context_window.max_tokens` and `context_window.max_completion_tokens` while preserving the local curated contract.
- Added focused tests for discovery, metadata overlay, scheduler timing, and default-model ordering.

## Task Commits

No commit was created in this run.

## Files Created/Modified

- `model/codex_catalog.go` - new persistent snapshot/candidate state for Codex catalog sync.
- `service/codex_catalog.go` - discovery, merge, validation, promotion, and channel sync logic.
- `service/codex_catalog_task.go` - daily `04:00` sync scheduler.
- `controller/codex_fetch_models.go` - dynamic admin fetch-models path.
- `controller/channel.go` - Codex fetch-models now uses account-aware discovery.
- `controller/channel_upstream_update.go` - Codex upstream update path now consumes promoted/dynamic catalog state.
- `controller/model.go` - overlays promoted Codex metadata into public `/v1/models`.
- `dto/pricing.go` - adds `context_window` to public catalog structs.
- `controller/*_test.go`, `service/codex_catalog_test.go` - focused verification for the new contract.

## Decisions Made

- The only safe request-time source of truth remains local Go catalog state; upstream Codex discovery stays asynchronous.
- Promotion requires a real upstream validation response and does not trust discovery alone.
- Default-model policy remains local even when upstream exposes new models.

## Deviations from Plan

None in product behavior. The only runtime adaptation was validation technique:

- Go validation had to run through `./scripts/podman-admin.sh profile-run` with real toolchain binaries and isolated `GOCACHE`, because the host-wide `build-cpu-guard` wrappers for `go` and `gcc` interfere with direct guarded `go test` invocation.

## Issues Encountered

- Initial validation attempts hit a real import cycle and then a function-signature mismatch in `service/codex_catalog.go`; both were fixed before final verification.
- Host build wrappers rewrote `go`/`gcc` into nested guarded scopes, which broke `cwd`, cgo temp files, and cache paths. The stable pattern was:
  - outer guard: `./scripts/podman-admin.sh profile-run -- bash -lc '...'`
  - inner toolchain: `PATH=/usr/local/go/bin:/usr/bin:/bin`
  - isolated cache: `GOCACHE=/tmp/router-ai-atius-go-cache-phase26`

## Next Phase Readiness

Phase 26 execution is complete at the code-and-test level. The next logical step is Phase 27, which can now assume:

- dynamic Codex discovery exists,
- local candidate/promotion state exists,
- promoted metadata reaches `/v1/models`,
- build validation on this host must respect the guarded pattern documented above.

## Verification Results

- `./scripts/podman-admin.sh profile-run -- bash -lc 'cd /home/ubuntu/GitHub/containers/router-ai-atius && export PATH=/usr/local/go/bin:/usr/bin:/bin && export GOCACHE=/tmp/router-ai-atius-go-cache-phase26 && /usr/local/go/bin/go test ./service -run "^(TestPrioritizeCodexDefaultModel|TestNextCodexCatalogSyncDelay|TestMergeCodexCatalogMetadataPrefersOverrideAndKeepsContextWindowGroup)$" -count=1 -timeout 600s -vet=off'` - PASS
- `./scripts/podman-admin.sh profile-run -- bash -lc 'cd /home/ubuntu/GitHub/containers/router-ai-atius && export PATH=/usr/local/go/bin:/usr/bin:/bin && export GOCACHE=/tmp/router-ai-atius-go-cache-phase26 && /usr/local/go/bin/go test ./controller -run "^(TestFetchDynamicCodexModelIDsUsesAccountAwareDiscovery|TestListModelsAppliesPromotedCodexCatalogMetadata)$" -count=1 -timeout 600s -vet=off'` - PASS
- `./scripts/podman-admin.sh profile-run -- bash -lc 'cd /home/ubuntu/GitHub/containers/router-ai-atius && export PATH=/usr/local/go/bin:/usr/bin:/bin && export GOCACHE=/tmp/router-ai-atius-go-cache-phase26 && /usr/local/go/bin/go test ./model -run "^$" -count=1 -timeout 600s -vet=off'` - PASS
- `./scripts/podman-admin.sh profile-run -- bash -lc 'cd /home/ubuntu/GitHub/containers/router-ai-atius && export PATH=/usr/local/go/bin:/usr/bin:/bin && export GOCACHE=/tmp/router-ai-atius-go-cache-phase26 && /usr/local/go/bin/go test . -run "^$" -count=1 -timeout 600s -vet=off'` - PASS

## Self-Check: PASSED

- Dynamic discovery is no longer static-only.
- `/v1/models` remains local and deterministic.
- Candidate promotion is validation-gated.
- Default model policy stays local.
- The guarded 20%-CPU validation path is now documented and reproducible.

---
*Phase: 26-codex-dynamic-discovery-and-curated-catalog*
*Completed: 2026-07-08*
