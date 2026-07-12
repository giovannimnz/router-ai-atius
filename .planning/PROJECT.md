# router-ai-atius

## What This Is

`router-ai-atius` is Giovanni's operational fork of `QuantumNous/new-api`, an AI API gateway/proxy with Go backend routing, provider adapters, billing/catalog behavior, and two frontend surfaces. This planning tree tracks fork-specific production work while preserving the ability to prepare narrow upstream-ready contributions when the target is `QuantumNous/new-api`.

## Core Value

Keep the router operational and upstream-compatible while making every change traceable to a narrow, validated plan.

## Requirements

### Validated

- Native Go `/v1/models` routing, provider catalog controls, Codex OAuth routing, and governed `embedding-gte-v1` are established fork behaviors that must not regress.
- PT-BR language work has prior local evidence, but Phase 21 must revalidate it against current `upstream/main` before any upstream PR handoff.
- Phase 32: o channel Codex type `57` usa UI dedicada, OAuth Router-owned com `refresh_token`, probe upstream, refresh automatico e erros de auth upstream distintos da API key interna do Router.

### Active

- [ ] Phase 28: create a safety-first branch/worktree hygiene pass before any destructive local cleanup.
- [ ] Phase 28: promote a single canonical PT-native upstream handoff branch and retire ambiguous local PT lanes.
- [ ] Phase 21: hand off the already-implemented PT-BR native lane through one clean upstream PR branch.
- [ ] Phase 21: keep the upstream PR candidate free of `.planning`, Graphify, Obsidian, runtime, DB/catalog, Podman, provider-routing, or Atius-only content.
- [ ] Phase 24 follow-up state: preserve the canonical router DB/catalog recovery decisions already recorded in `STATE.md`.

### Out of Scope

- Replacing upstream `i18n/` directories — current upstream uses those as native language mechanisms.
- Shipping fork/runtime/provider changes in the Phase 21 upstream contribution path — those belong to separate fork phases.
- Opening an upstream PR without Giovanni's approval after local validation.

## Context

The project uses Go 1.22+, Gin, GORM, React frontends under `web/default` and `web/classic`, Bun for default frontend scripts, and GSD planning under `.planning/`. Phase 21 is intentionally narrow: local first PT-BR native language implementation, then optional clean upstream handoff.

The main checkout may be dirty with unrelated fork/runtime work. Phase 21 implementation must start from a clean worktree or branch based on current `upstream/main`.

## Constraints

- **Project policy**: Preserve protected upstream project and organization identifiers from `AGENTS.md`.
- **Database/provider fork safety**: Do not disturb router DB/catalog/provider customizations while preparing Phase 21.
- **Frontend workflow**: Use Bun for `web/default` scripts and keep classic/default i18n systems independently valid.
- **PR hygiene**: Use `.github/PULL_REQUEST_TEMPLATE.md`; disclose AI assistance when required by repository policy.
- **Secret hygiene**: Do not print, commit, or store bearer tokens or provider credentials.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Phase 21 uses current `QuantumNous/new-api` `upstream/main` as baseline | Prevents stale or contaminated PT-BR work from becoming the implementation contract | Implemented locally; pending handoff |
| Phase 21 adds `pt` through existing backend/default/classic i18n surfaces | Matches upstream-native architecture and avoids fork-only translation layers | Implemented locally; pending handoff |
| Phase 21 planning artifacts stay outside any upstream PR branch | Keeps the potential upstream contribution reviewable and narrow | Implemented locally; pending handoff |
| Codex is the active GSD runtime in this checkout | Local skills and agents are installed under `~/.codex` | Pending execution |
| PT-native upstream handoff must converge on a single canonical remote branch | Prevents branch drift and accidental PR creation from polluted local integration branches | Planned |
| `origin/main` is the only trustworthy fork mainline | Local `main` worktrees may drift and must be recreated from remote truth when hygiene is required | Planned |
| Codex type `57` nao expoe Base URL/API Key genericos | Existe um unico endpoint Codex e material OAuth nao deve aparecer na UI | Validated in Phase 32 |
| Credencial Codex valida exige probe e capacidade de renovacao | Expiracao local futura nao prova autenticidade upstream nem autorrenovacao | Validated in Phase 32 |
| Access token do Codex CLI e somente break-glass | Copiar o refresh token compartilharia rotacao e poderia quebrar o CLI | Replaced by Router-owned OAuth in Phase 32 |
| Validacao live deve confirmar o DSN efetivo do runtime | Bancos homonimos local e canonico podem produzir evidencia divergente | Validated in Phase 32 |

---
*Last updated: 2026-07-12 after Phase 32 OAuth lifecycle completion.*
