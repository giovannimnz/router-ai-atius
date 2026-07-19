---
gsd_state_version: 1.0
milestone: v2.17
milestone_name: — Codex OAuth lifecycle and upstream auth diagnostics
current_phase: null
status: Awaiting next milestone
stopped_at: Phases 29 and 30 complete; runtime public on k3s
last_updated: "2026-07-19T20:21:06Z"
last_activity: 2026-07-19
last_activity_desc: v2.16 closed after restore, shadow, public cutover, and soak validation
progress:
  total_phases: 1
  completed_phases: 1
  total_plans: 4
  completed_plans: 4
  percent: 100
---

# STATE.md — atius-ai-router

## Current Position

Phase: Milestones v2.16 and v2.17 complete
Plan: —
Status: Awaiting next milestone
Last activity: 2026-07-19 — Phases 29/30 completed; public runtime moved to k3s

## What Was Done

### 2026-07-19 — v2.16 DONE + Codex metadata canonicalized

- Restore final de `DBRouterAiAtius` no PostgreSQL k3s validado por contagens exatas das tabelas criticas.
- Shadow stack e cutover Apache concluidos para o Service `router-ai-atius` (`10.43.102.221:3000`).
- App, PostgreSQL e Redis estao fixados em `atius-srv-1`, com `500m` de request/limit por pod e PVCs `local-path`/`Retain`.
- Podman foi parado, nao removido, e permanece como rollback operacional.
- Catalogo Codex usa OAuth ativo como fonte primaria: se houver contexto e saida usa ambos; se houver apenas contexto, complementa somente a saida pela documentacao oficial OpenAI.
- `/v1/responses` non-stream do Codex agora converte o stream SSE upstream para JSON terminal e passou em runtime publico.

### 2026-06-04 — v1.6 ✅ DONE + v1.8 ✅ DONE

**v1.6 — Internacionalização PT-BR** (closed):

- Frontend `web/default/src/i18n/locales/pt.json` — 3910 chaves, 100% de en.json
- Backend `i18n/locales/pt.yaml` — 227 chaves, 100% de en.yaml
- `web/default/src/i18n/languages.ts` — pt registrado, fallback en
- Tests: `pt-fallback.test.ts`, `normalize-interface-language.test.ts`
- PR upstream #2 mergeado em 2026-05-31 (commit `e06abacb7`)
- Validado em runtime: log do new-api mostra
  `i18n initialized with languages: zh-CN, zh-TW, en, pt`

**v1.8 — Podman Migration** (historical, reconciled 2026-06-29; rollback since 2026-07-19):

- Rootless Podman remains installed and stopped as rollback in
  `/home/ubuntu/GitHub/containers/router-ai-atius`; it is no longer the public runtime.

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

- **v2.12 Phase 21 handoff preserved remotely** — branch remota
  `origin/feat/phase21-pt-native-upstream` contains the clean PT-native
  upstream handoff commit. The old local `feat/pt-native` integration branch
  was backed up and removed during Phase 28.

- **Podman config/docs reconciliation** — completed 2026-06-29 in this
  checkout. Remaining Docker references are upstream/legacy compatibility or
  OCI build terminology, not the active production path.

- **Limpar backup tag** `backup/before-squash-20260604` depois de
  confirmar produção estável por ≥ 7 dias.

## Architecture Discovered

```
Apache (router.atius.com.br:443)
├── app/API/health/login → k3s Service 10.43.102.221:3000
└── docs                 → local 127.0.0.1:3003

Runtime (k3s, namespace router-ai-atius, node atius-srv-1):
router-ai-atius            Go AI gateway       1 replica, 500m
router-ai-atius-postgres   PostgreSQL 17        StatefulSet, 500m
router-ai-atius-redis      Redis                Deployment, 500m

DB:       DBRouterAiAtius
Rollback: container-router-ai-atius.service (inactive, preserved)
Runbook:  docs/K3S-MIGRATION.md
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
| v2.12 | pt-native upstream sync | 🚧 in progress — Phase 21 executed locally; clean upstream handoff still pending |
| v2.13 | Router DB/catalog recovery on canonical host DB | ✅ done 2026-07-08 |
| v2.14 | Branch hygiene and mainline reconciliation | ✅ done 2026-07-08 |
| v2.15 | K3s transition and deferred runtime validation | ✅ done 2026-07-09 (preparation package only; no public cutover) |
| v2.16 | K3s shadow, cutover, and planning hygiene | ✅ done 2026-07-19 — Phases 29/30/31 complete |
| v2.17 | Codex OAuth lifecycle and upstream auth diagnostics | ✅ done 2026-07-12 — Router-owned OAuth, probe, refresh e smokes completos |

## Next actions

1. **Optional handoff v2.12 Phase 21 — feat-pt-native-pr**:
   - Usar `origin/feat/phase21-pt-native-upstream`, não `feat/pt-native`
   - Validar o diff contra `upstream/main`
   - Abrir PR novo limpo contra `QuantumNous/new-api` somente se/quando aprovado
2. **K3s runtime guardrail**:
   - operar pelo namespace `router-ai-atius` e validar sempre `500m` por pod
   - manter `container-router-ai-atius.service` parado e preservado durante o soak estendido
   - executar backup por `scripts/k3s-router-backup.sh` antes de mudancas de dados
3. **Limpar backup tag** `backup/before-squash-20260604` apos confirmar que ainda nao e necessaria.

## Cross-references (Obsidian)

- `61-Incidents/2026-06-04-router-atius-503-new-api-crash` — fix do 503
- `61-Incidents/2026-06-04-podman-cherry-pick-main` — cherry-pick inicial
- `61-Incidents/2026-06-04-podman-full-rebrand-v211` — rebrand completo
- `61-Incidents/2026-06-04-podman-latest-tag-strategy` — `:latest` decision
- `61-Incidents/2026-06-04-podman-pre-rebrand-cleanup` — docker-compose + PROJECT.md
- `61-Incidents/2026-06-04-podman-push-to-origin` — push final
- `61-Incidents/2026-06-04-translation-pt-br-status` — pt-BR 100% verificado

---
*Last updated: 2026-07-19 17:21 -0300 after Phases 29/30 k3s completion.*

## Performance Metrics

| Phase | Plan | Duration | Notes |
|-------|------|----------|-------|
| Go-native cutover P02 | 14 min | 6 tasks | 15 files |
| Phase 24 P02 | 8 min | 3 tasks | 4 files |
| Phase 24 P03 | 12 min | 3 tasks | 4 files |
| Phase 25 P01 | 14 min | 2 tasks | 2 files |
| Phase 25 P02 | 10 min | 2 tasks | 2 files |
| Phase 25 P03 | 12 min | 2 tasks | 3 files |
| Phase 26 P01 | 1 session | 6 workstreams | dynamic discovery, promotion, scheduler |
| Phase 27 P01 | 1 session | 3 workstreams | official docs, workflow alignment, PT-BR runbook |

## Decisions

- [Go-native cutover]: Use existing GET /v1/models as the only public Go catalog endpoint — Avoids a second source of truth and satisfies the corrected Go-owned catalog contract.
- [Phase ?]: Use api_format=anthropic and Anthropic headers for model-list intent — Lets Go serve Anthropic-selected model lists under the same root data-only payload contract.
- [Phase ?]: Keep pricing provenance internal to JSON output — pricing_source and pricing_estimated are useful internally but must not leak from public /v1/models.
- [Phase 24]: Candidate DB build stays dry-run by default and requires explicit source/target confirmations.
- [Phase 24]: Transformed catalog restore injects the Codex credential only from a secure runtime variable instead of git.
- [Phase 24]: newapi remains intact as rollback holdback throughout Phase 24 Plan 24-02.
- [Phase 24]: Plan 24-03 keeps `gpt-5.4` as the default long-context Codex model and removes final-state `-1m` alias expectations from code, tests, and docs.
- [Phase 24]: Plan 24-03 documents DeepSeek as the single active consolidated provider and MiniMax as restored but disabled in the final state.
- [Phase 24]: Plan 24-03 preserves `embedding-gte-v1` as the only governed public embedding alias with `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` unchanged.
- [Phase 26]: Codex discovery became asynchronous and account-aware, while request-time `/v1/models` remained fully local and deterministic.
- [Phase 26]: Host validation for guarded Go builds must use `profile-run` outside, real toolchain binaries inside, and isolated `GOCACHE`.
- [Phase 27]: Official OpenAI/Codex docs are the source of truth for CI/auth behavior; API keys stay default for automation and ChatGPT-managed auth remains private-runner-only.
- [Repo hygiene]: `origin/main` is the authoritative fork mainline, and the local `/home/ubuntu/GitHub/containers/router-ai-atius` worktree is a clean `main` tracking it.
- [Repo hygiene]: `origin/feat/phase21-pt-native-upstream` is the clean Phase 21 handoff lane. `feat/pt-native` and redundant PT branches were backed up and removed.
- [Phase 28 planning]: v2.14 should own branch/worktree hygiene and mainline reconciliation; v2.15 should own Phases 22 and 23 as deferred platform/runtime work.
- [Phase 22]: k3s migration is now documented and tooled, but public cutover remains manual and blocked by restore/shadow evidence plus current cluster constraints.
- [Phase 23]: the long-context harness is validated locally/static with the 1M cost gate preserved; new paid live runs remain operator-triggered only.
- [Phase 29]: real k3s shadow deployment, restore rehearsal, and go/no-go should live in a new phase, not be treated as already completed work from Phase 22.
- [Phase 30]: public Apache cutover should stay separate from shadow validation and remain rollback-first.
- [Phase 31]: current `.planning/` health debt is real but historical; fix it in a dedicated hygiene phase instead of mixing it into runtime work.
- [Phase 31]: legacy directories and `FORK_MIGRATION.md` were archived into `.planning/milestones/legacy-planning-archive-20260709`, and `validate.health` is now healthy.
- [Phase 32]: Codex channel type `57` needs a dedicated OAuth lifecycle UI; generic `Base URL` and `API Key` surfaces are regressions for this fork.
- [Phase 32]: Future local expiration is not enough to declare a Codex credential valid; upstream auth probe/error state must participate in channel health.
- [Phase 32]: Copying an access token from Codex CLI is break-glass only, not a durable Router credential.
- [Codex metadata]: Active OAuth output is authoritative when present; official OpenAI metadata may fill only a missing output limit and must not overwrite OAuth context.
- [Phase 29]: Single-node `local-path` is accepted for this deployment, with explicit `Retain`, node pinning, tested backup/restore, and no HA claim.
- [Phase 30]: Apache targets the k3s Service; Podman remains stopped and intact as the immediate rollback path during extended soak.

## Accumulated Context

### Roadmap Evolution

- Phase 22 added: k3s migration preflight and cutover plan for router-ai-atius. Phase 21 (`feat-pt-native-pr`) remains a separate PT-native upstream PR handoff. Podman remains the current production source of truth until Phase 22 shadow/cutover gates pass.
- Phase 23 added: long-context alias validation for `gpt-5.5-1m` and `gpt-5.4-1m`. This is an operational validation track for progressive reasoning/context tests up to approximately 1M tokens. It is independent of Phase 21 and blocked on deploying the alias pricing fix before accepting production UAT evidence.
- Phase 24 added: router DB/catalog recovery and canonical host DB restoration. This phase owns the post-2026-07-02 runtime drift: canonical host PostgreSQL/PgBouncer path, full `OpenAI - Codex` catalog recovery, DeepSeek recovery, MiniMax consolidated-but-disabled recovery, and preservation of the Go embedding governor path. Phase 21 remains parked, not deleted.
- Phase 29 added: k3s shadow deployment, restore rehearsal, and explicit go/no-go are separated from the already-closed preparation package of Phase 22.
- Phase 30 added: public k3s cutover and rollback soak are separated from shadow validation and stay blocked on real evidence from Phase 29.
- Phase 31 added: planning-health normalization and legacy archive now have a dedicated place in the roadmap instead of staying as permanent background debt.
- Phase 31 completed on 2026-07-09 by archiving legacy phase directories and moving `FORK_MIGRATION.md` out of the `.planning/` root; `validate.health` now returns `healthy`.
- Phase 32 added: Codex OAuth lifecycle hardening is a new v2.17 incident-driven milestone and does not close or replace the real k3s shadow/cutover work from Phases 29/30.

### Active execution note

- Phase 24 execution finalized the live cutover on `2026-07-04`: runtime points only to `DBRouterAiAtius` via PgBouncer, the legacy `newapi` mapping was removed from PgBouncer, `embedding-gte-v1` validates at `768` dims, `gpt-5.4` validates via Codex after reloading channel 5 from `~/.codex/auth.json`, DeepSeek validates after key replacement, and MiniMax was disabled in channels/abilities and no longer appears in authenticated `/v1/models`. Phase 21 remains parked, not deleted.
- Phase 26 execution finalized on `2026-07-08`: dynamic Codex discovery now reads the active account’s `/backend-api/codex/models`, persists snapshots/candidates locally, gates promotion on a live `Ok` probe, overlays promoted metadata into `/v1/models`, and schedules daily sync at `04:00` without making the public catalog depend on live upstream reads.
- Phase 27 execution finalized on `2026-07-08`: CI/auth/release guidance is now explicitly pinned to official OpenAI/Codex docs, `sync.yml` uses the first-class `effort` input for `openai/codex-action`, PT-BR operator docs capture API-key default automation, and ChatGPT-managed auth remains restricted to trusted private runners.
- Phase 32 execution completed on `2026-07-12`: UI/API/runtime/docs/fork-sync were validated live, the `401 token_invalidated` incident is no longer active, and the temporary fallback was replaced by a Router-owned OAuth credential with refresh token; probe, refresh and local/public smokes passed.
- Phase 29 execution completed on `2026-07-19`: the k3s stateful stack was restored from the effective production DSN, critical table counts matched, shadow health/catalog/embedding/Codex smokes passed, and the decision was GO.
- Phase 30 execution completed on `2026-07-19`: Apache moved to Service `10.43.102.221:3000`, public smokes and non-stream Responses passed after the initial soak, and Podman stayed inactive but preserved for rollback.

## Session

**Last session:** 2026-07-19T17:21:06-03:00
**Stopped at:** Phases 29/30 complete; k3s public runtime validated
**Resume file:** None

## Project Reference

See: `.planning/PROJECT.md` (updated 2026-07-12)

**Core value:** Keep the router operational and upstream-compatible while making every change traceable to a narrow, validated plan.
**Current focus:** Milestones v2.16 and v2.17 complete; optional Phase 21 upstream handoff remains separate.

## Operator Next Steps

- Start the next milestone with $gsd-new-milestone
