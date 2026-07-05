---
gsd_state_version: 1.0
milestone: v2.13
milestone_name: Router DB/catalog recovery on canonical host DB
current_phase: 24
status: executing
stopped_at: Phase 24 runtime fully standardized on DBRouterAiAtius after PgBouncer cleanup and provider validation
last_updated: "2026-07-05T10:37:22.474Z"
last_activity: 2026-07-05
last_activity_desc: Phase 25 execution started
progress:
  total_phases: 27
  completed_phases: 2
  total_plans: 17
  completed_plans: 11
  percent: 7
---

# STATE.md — atius-ai-router

## Current Position

Phase: 25 (embedding-governor-auto-workload-inference) — EXECUTING
Plan: 3 of 3
**Milestone:** v2.13 — router DB/catalog recovery on canonical host DB (current)
**Phase:** 24
**Status:** Ready to execute
**Last activity:** 2026-07-05 — Phase 25 execution started

## What Was Done

### 2026-06-04 — v1.6 ✅ DONE + v1.8 ✅ DONE

**v1.6 — Internacionalização PT-BR** (closed):

- Frontend `web/default/src/i18n/locales/pt.json` — 3910 chaves, 100% de en.json
- Backend `i18n/locales/pt.yaml` — 227 chaves, 100% de en.yaml
- `web/default/src/i18n/languages.ts` — pt registrado, fallback en
- Tests: `pt-fallback.test.ts`, `normalize-interface-language.test.ts`
- PR upstream #2 mergeado em 2026-05-31 (commit `e06abacb7`)
- Validado em runtime: log do new-api mostra
  `i18n initialized with languages: zh-CN, zh-TW, en, pt`

**v1.8 — Podman Migration** (closed, reconciled 2026-06-29):

- Runtime production is already rootless Podman in
  `/home/ubuntu/GitHub/containers/router-ai-atius`.

- User systemd source of truth: `container-router-ai-atius.service`.
- Production pod: `atius-ai-router`; containers: `router-ai-atius`, `postgres`,
  `redis`, infra pause.

- Canonical `/v1/` path is full-Go on `127.0.0.1:3000`; no Python
  `model-detailed` container participates in the active relay path.

- Dev stack source is `podman-compose.yml`; `make dev-api`,
  `make dev-api-rebuild`, and `make reset-setup` use `podman compose`.

- `docs/PODMAN.md` is the current Podman runbook and
  `scripts/podman-validate.sh` is the lightweight config gate.

- `Dockerfile`, `Dockerfile.dev`, and `.dockerignore` remain OCI/upstream build
  surfaces and are valid with Podman/Buildah.

### Pending operational work (not committed)

- **v2.12 Phase 7 ready for handoff** — branch local `feat/pt-native` contains exatamente:
  `i18n/i18n.go`, `i18n/locales/pt.yaml`,
  `web/default/src/i18n/config.ts`,
  `web/default/src/i18n/languages.ts`,
  `web/default/src/i18n/locales/pt.json`
  No commit/push yet by design.

- **Podman config/docs reconciliation** — completed 2026-06-29 in this
  checkout. Remaining Docker references are upstream/legacy compatibility or
  OCI build terminology, not the active production path.

- **Limpar backup tag** `backup/before-squash-20260604` depois de
  confirmar produção estável por ≥ 7 dias.

## Architecture Discovered

```
Apache (router.atius.com.br:443)
├── /v1/*          → router-ai-atius Go backend: 127.0.0.1:3000/v1/*
├── /api/*         → router-ai-atius Go backend: 127.0.0.1:3000/api/*
├── /health        → router-ai-atius Go backend: 127.0.0.1:3000/api/status
├── /login         → router-ai-atius Go backend: 127.0.0.1:3000/sign-in
├── /logoff        → router-ai-atius Go backend: 127.0.0.1:3000/logout
└── /              → router-ai-atius Go backend: 127.0.0.1:3000/ (SPA)

Runtime (rootless Podman, current host):
router-ai-atius        Go AI gateway       local 127.0.0.1:3000
postgres               PostgreSQL          pod-internal
redis                  Redis               pod-internal

Pod:     atius-ai-router
DB:      DBRouterAiAtius
Unit:    container-router-ai-atius.service
Runbook: docs/PODMAN.md
```

## Phase Status (v1.6 — closed)

| Phase | Status | Notes |
|-------|--------|-------|
| Frontend PT-BR translation | ✅ done | 3910 chaves, 100% cobertura |
| Backend i18n PT-BR | ✅ done | 227 chaves, 100% cobertura |
| DB: set Language=pt | ✅ done | System name "Atius Router" no main |
| Branch: feat/portuguese-translation | ✅ merged | PR #2 upstream |
| Upstream PR | ✅ merged | 2026-05-31 |

## Phase Status (v1.8 — closed)

| Phase | Status | Notes |
|-------|--------|-------|
| Podman compose file | ✅ done | `podman-compose.yml` dev stack |
| User systemd runtime | ✅ done | `container-router-ai-atius.service` owns production backend |
| Makefile dev targets | ✅ done | `make dev-api`, `make dev-api-rebuild`, `make reset-setup` use Podman |
| Validation script | ✅ done | `scripts/podman-validate.sh` |
| Documentation | ✅ done | `docs/PODMAN.md` |
| Production cutover | ✅ done | runtime is Podman/full-Go in `/home/ubuntu/GitHub/containers` |

## Blocker

| Blocker | Priority | Notes |
|---------|----------|-------|
| Nenhum técnico | — | Pronto pra v1.7 |

## Milestones

| Version | Goal | Status |
|---------|------|--------|
| v1.0 | Initial Setup | ✅ |
| v1.1 | DeepSeek Enrichment | ✅ |
| v1.2 | Fork Migration | ✅ |
| v1.3 | Testing Infrastructure | ✅ |
| v1.4 | Model Aliases | ✅ |
| v1.5 | API Unification & Model Listing | ✅ |
| v1.6 | Internacionalização PT-BR | ✅ done 2026-06-04 |
| v1.7 | Documentação PT-BR | deferred (lower priority) |
| v1.8 | Podman Migration | ✅ done; reconciled 2026-06-29 |
| v1.9 | GHCR Deploy | pending |
| v2.0 | Podman Migration (legacy name) | ✅ superseded by v1.8 |
| v2.10 | MiniMax Anthropic | ✅ done 2026-05-31 |
| v2.12 | pt-native upstream sync | 🚧 in progress — Phase 7 local done; Phase 8 moved to Phase 21; Phase 21 ready for planning |

## Next actions

1. **Plan/execute v2.12 Phase 21 — feat-pt-native-pr**:
   - Commitar o working tree atual de `feat/pt-native` com 1 commit limpo
   - Push branch novo pro fork (`giovannimnz/router-ai-atius`)
   - Fechar PR #5245 poluído com comentário
   - Abrir PR novo limpo contra `QuantumNous/new-api`
2. **Execute v2.13 Phase 24 — router DB/catalog recovery**:
   - 24-01: freeze runtime truth, fresh backups, source ranking
   - 24-02: create canonical host DB target and restore transformed catalog
   - 24-03: reconcile `OpenAI - Codex`, DeepSeek, MiniMax disabled state, and governor-facing docs/tests ✅ completed locally without commits due to pre-dirty target files
   - 24-04: cut over runtime/docs/CLI to the canonical DB name and validate rollback ✅ executed and finalized on 2026-07-04

3. **Podman runtime guardrail**:
   - Keep production lifecycle on `systemctl --user restart container-router-ai-atius.service`
   - Keep dev/runtime checks on `podman-compose.yml` + `scripts/podman-validate.sh`
   - Treat `docker-compose*.yml` as upstream/legacy compatibility unless a future
     phase explicitly removes or renames them.

4. **Limpar backup tag** `backup/before-squash-20260604` (≥ 7 dias prod estável)

## Cross-references (Obsidian)

- `61-Incidents/2026-06-04-router-atius-503-new-api-crash` — fix do 503
- `61-Incidents/2026-06-04-podman-cherry-pick-main` — cherry-pick inicial
- `61-Incidents/2026-06-04-podman-full-rebrand-v211` — rebrand completo
- `61-Incidents/2026-06-04-podman-latest-tag-strategy` — `:latest` decision
- `61-Incidents/2026-06-04-podman-pre-rebrand-cleanup` — docker-compose + PROJECT.md
- `61-Incidents/2026-06-04-podman-push-to-origin` — push final
- `61-Incidents/2026-06-04-translation-pt-br-status` — pt-BR 100% verificado

---
*Last updated: 2026-06-17 21:48 -0300 after v2.12 Phase 7 local execution on feat/pt-native*

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Phase phase-20 P02 | 14 min | 6 tasks | 15 files |
| Phase 24 P02 | 8 min | 3 tasks | 4 files |
| Phase 24 P03 | 12 min | 3 tasks | 4 files |
| Phase 25 P01 | 14 min | 2 tasks | 2 files |
| Phase 25 P02 | 10 min | 2 tasks | 2 files |

## Decisions

- [Phase ?]: Use existing GET /v1/models as the only public Go catalog endpoint — Avoids a second source of truth and satisfies the corrected Phase 20 contract.
- [Phase ?]: Use api_format=anthropic and Anthropic headers for model-list intent — Lets Go serve Anthropic-selected model lists under the same root data-only payload contract.
- [Phase ?]: Keep pricing provenance internal to JSON output — pricing_source and pricing_estimated are useful internally but must not leak from public /v1/models.
- [Phase 24]: Candidate DB build stays dry-run by default and requires explicit source/target confirmations.
- [Phase 24]: Transformed catalog restore injects the Codex credential only from a secure runtime variable instead of git.
- [Phase 24]: newapi remains intact as rollback holdback throughout Phase 24 Plan 24-02.
- [Phase 24]: Plan 24-03 keeps `gpt-5.4` as the default long-context Codex model and removes final-state `-1m` alias expectations from code, tests, and docs.
- [Phase 24]: Plan 24-03 documents DeepSeek as the single active consolidated provider and MiniMax as restored but disabled in the final state.
- [Phase 24]: Plan 24-03 preserves `embedding-gte-v1` as the only governed public embedding alias with `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` unchanged.

## Accumulated Context

### Roadmap Evolution

- Phase 22 added: k3s migration preflight and cutover plan for router-ai-atius. Phase 21 (`feat-pt-native-pr`) remains a separate PT-native upstream PR handoff. Podman remains the current production source of truth until Phase 22 shadow/cutover gates pass.
- Phase 23 added: long-context alias validation for `gpt-5.5-1m` and `gpt-5.4-1m`. This is an operational validation track for progressive reasoning/context tests up to approximately 1M tokens. It is independent of Phase 21 and blocked on deploying the alias pricing fix before accepting production UAT evidence.
- Phase 24 added: router DB/catalog recovery and canonical host DB restoration. This phase owns the post-2026-07-02 runtime drift: canonical host PostgreSQL/PgBouncer path, full `OpenAI - Codex` catalog recovery, DeepSeek recovery, MiniMax consolidated-but-disabled recovery, and preservation of the Go embedding governor path. Phase 21 remains parked, not deleted.

### Active execution note

- Phase 24 execution finalized the live cutover on `2026-07-04`: runtime points only to `DBRouterAiAtius` via PgBouncer, the legacy `newapi` mapping was removed from PgBouncer, `embedding-gte-v1` validates at `768` dims, `gpt-5.4` validates via Codex after reloading channel 5 from `~/.codex/auth.json`, DeepSeek validates after key replacement, and MiniMax was disabled in channels/abilities and no longer appears in authenticated `/v1/models`. Phase 21 remains parked, not deleted.

## Session

**Last session:** 2026-07-04T05:00:19.031Z
**Stopped at:** Phase 24 runtime fully standardized on DBRouterAiAtius after PgBouncer cleanup and provider validation
**Resume file:** .planning/phases/24-router-db-catalog-recovery-and-canonical-host-db/24-04-PLAN.md
