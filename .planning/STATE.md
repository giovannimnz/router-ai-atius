---
gsd_state_version: 1.0
milestone: v1.4
milestone_name: — Model Aliases & Token Management ✓
current_phase: 20
status: completed
stopped_at: Completed phase-20-03-PLAN.md
last_updated: "2026-06-26T13:04:51.800Z"
last_activity: 2026-06-26
progress:
  total_phases: 2
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# STATE.md — atius-ai-router

## Current Position

Phase: phase-20 (go-native-model-router) — EXECUTING
Plan: Not started
**Milestone:** v2.12 — pt-native upstream sync (next)
**Phase:** 20
**Status:** Milestone complete
**Last activity:** 2026-06-26

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

**v1.8 — Podman Migration** (closed):

- `podman-compose.yml` — rebrand v2.11 + tag `:latest` canônico
- 4 systemd quadlets: `podman/quadlets/router-ai-atius-*.container`
- 6 scripts operacionais: `podman-{up,down,validate,prepare-images,migrate-from-docker,quadlets-install}.sh`
- `docs/PODMAN.md` (160+ linhas) — referência operacional completa
- `.env.example` + `podman/systemd/router-ai-atius.env.example` — env templates
- `docker-compose.yml` + `docker-compose.dev.yml` alinhados (legacy mantido)
- `.planning/PROJECT.md` rebrand v2.11 + seção Podman
- `./scripts/podman-validate.sh` passa: 4 services v2.11 + spec render OK
- 5 commits pushed em 2026-06-04: `091ef482a`, `cd49cc5f3`, `32c01aa51`,
  `7fd0f455e`, `8fe7e01bb` (squash de `:local` → `:latest`),
  `243df2d48` (pre-rebrand cleanup). Final head: `e6c617f00`.

### Pending operational work (not committed)

- **v2.12 Phase 7 ready for handoff** — branch local `feat/pt-native` contains exatamente:
  `i18n/i18n.go`, `i18n/locales/pt.yaml`,
  `web/default/src/i18n/config.ts`,
  `web/default/src/i18n/languages.ts`,
  `web/default/src/i18n/locales/pt.json`
  No commit/push yet by design.

- **SRV-1 migration to Podman** — quando Giovanni marcar janela
  de manutenção, rodar `./scripts/podman-migrate-from-docker.sh` no
  SRV-1 (137.131.190.161). Sem push do `:v2.11.1-rebrand` pro GHCR
  ainda — Docker local build é suficiente.

- **Limpar backup tag** `backup/before-squash-20260604` depois de
  confirmar produção estável por ≥ 7 dias.

## Architecture Discovered

```
Apache (router.atius.com.br:443)
├── /docs          → router-ai-atius-model-detailed:3300/docs
├── /openapi.json  → router-ai-atius-model-detailed:3300/openapi.json
├── /v1/*          → router-ai-atius-model-detailed:3300/v1/* (relay)
├── /api/*         → router-ai-atius:3030/api/*
├── /login         → router-ai-atius:3030/sign-in
├── /logoff        → router-ai-atius:3030/logout
└── /              → router-ai-atius:3030/ (SPA)

Containers (SRV-1, atualmente em Docker, alvo = Podman):
router-ai-atius               Go AI gateway       port 3030:host → 3000
router-ai-atius-model-detailed FastAPI middleware port 3300:host → 3001
router-ai-atius-db            PostgreSQL 15       port 5432 (internal)
router-ai-atius-redis         Redis 7             port 6379 (internal)

Network: atius-ai-router_internal (rootless podman bridge)
DB:      DBRouterAiAtius
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
| Podman compose file | ✅ done | `podman-compose.yml` v2.11 |
| Systemd quadlets | ✅ done | 4 .container files |
| Helper scripts | ✅ done | 6 scripts (up/down/validate/prepare/migrate/quadlets-install) |
| Validation script | ✅ done | `podman-validate.sh` passa |
| Documentation | ✅ done | `docs/PODMAN.md` |
| SRV-1 cutover | ⏳ pending | Janela de manutenção |

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
| v1.5 | API Documentation Site | ✅ |
| v1.6 | Internacionalização PT-BR | ✅ done 2026-06-04 |
| v1.7 | Documentação PT-BR | deferred (lower priority) |
| v1.8 | Podman Migration | ✅ done 2026-06-04 (code); SRV-1 cutover pending |
| v1.9 | GHCR Deploy | pending |
| v2.0 | Podman Migration (legacy name) | ✅ superseded by v1.8 |
| v2.10 | MiniMax Anthropic | ✅ done 2026-05-31 |
| v2.12 | pt-native upstream sync | 🚧 in progress — Phase 7 local done; Phase 8 pending |

## Next actions

1. **Plan/execute v2.12 Phase 8 — feat-pt-native-pr** (autorização necessária):
   - Commitar o working tree atual de `feat/pt-native` com 1 commit limpo
   - Push branch novo pro fork (`giovannimnz/router-ai-atius`)
   - Fechar PR #5245 poluído com comentário
   - Abrir PR novo limpo contra `QuantumNous/new-api`
2. **SRV-1 Podman cutover** (quando Giovanni marcar):
   - Build/populate `:latest` images
   - Janela de manutenção
   - `./scripts/podman-migrate-from-docker.sh`
   - Smoke test: `curl https://router.atius.com.br/api/status`
3. **Limpar backup tag** `backup/before-squash-20260604` (≥ 7 dias prod estável)

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

## Decisions

- [Phase ?]: Use existing GET /v1/models as the only public Go catalog endpoint — Avoids a second source of truth and satisfies the corrected Phase 20 contract.
- [Phase ?]: Use api_format=anthropic and Anthropic headers for model-list intent — Lets Go serve Anthropic-selected model lists under the same root data-only payload contract.
- [Phase ?]: Keep pricing provenance internal to JSON output — pricing_source and pricing_estimated are useful internally but must not leak from public /v1/models.

## Session

**Last session:** 2026-06-26T11:48:08.881Z
**Stopped at:** Completed phase-20-03-PLAN.md
**Resume file:** None
