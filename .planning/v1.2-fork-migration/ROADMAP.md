# v1.2 — Fork Migration & CLI-Anything Integration

## Objetivo

Estabelecer workflow de fork formal com sync automation, versionamento, e CLI-Anything para gerenciar o NewAPI e infraestrutura Atius via agente.

## Fases

| # | Fase | Slug | Status |
|---|------|------|--------|
| 1 | Git Setup & Remotes | `fork-git-setup` | `pending` |
| 2 | Fork Sync Script | `fork-sync-script` | `pending` |
| 3 | Version Bump Script | `fork-version-bump` | `pending` |
| 4 | FORK.md Documentation | `fork-fork-md` | `pending` |
| 5 | GitHub Actions CI/CD | `fork-github-actions` | `pending` |
| 6 | CLI-Anything: NewAPI Management | `cli-anything-newapi` | `pending` |

## Dependencies

- Fase 1 não tem dependências
- Fase 2 depende de Fase 1
- Fase 3 depende de Fase 1
- Fase 4 depende de Fases 2 e 3
- Fase 5 depende de Fases 2 e 3
- Fase 6 é independente (pode rodar em paralelo após Fase 1)

## Executar em Paralelo

Após Fase 1:
- Fases 2 e 3 podem rodar em paralelo
- Fase 6 pode iniciar (análise do codebase NewAPI)
- Fase 5 espera Fases 2 e 3

## Critério de Verificação

Ao final, deve estar operacional:
- `git remote -v` mostra origin + upstream
- `./scripts/sync-fork.sh --dry-run` funciona
- `./scripts/version-bump.sh --check` funciona
- `FORK.md` documenta todas modificações
- GitHub Actions disparam em push
- `cli-anything-newapi --help` funciona
- Agente consegue: listar models, ver channels, fazer restart de containers

---

## Fase 1: Git Setup & Remotes

**Slug:** `fork-git-setup`
**Dir:** `.planning/v1.2-fork-migration/1-fork-git-setup/`
**Status:** `pending`

Ver `.planning/v1.2-fork-migration/1-fork-git-setup/PLAN.md`

---

## Fase 2: Fork Sync Script

**Slug:** `fork-sync-script`
**Dir:** `.planning/v1.2-fork-migration/2-fork-sync-script/`
**Status:** `pending`

Ver `.planning/v1.2-fork-migration/2-fork-sync-script/PLAN.md`

---

## Fase 3: Version Bump Script

**Slug:** `fork-version-bump`
**Dir:** `.planning/v1.2-fork-migration/3-fork-version-bump/`
**Status:** `pending`

Ver `.planning/v1.2-fork-migration/3-fork-version-bump/PLAN.md`

---

## Fase 4: FORK.md Documentation

**Slug:** `fork-fork-md`
**Dir:** `.planning/v1.2-fork-migration/4-fork-fork-md/`
**Status:** `pending`

Ver `.planning/v1.2-fork-migration/4-fork-fork-md/PLAN.md`

---

## Fase 5: GitHub Actions CI/CD

**Slug:** `fork-github-actions`
**Dir:** `.planning/v1.2-fork-migration/5-fork-github-actions/`
**Status:** `pending`

Ver `.planning/v1.2-fork-migration/5-fork-github-actions/PLAN.md`

---

## Fase 6: CLI-Anything: NewAPI Management

**Slug:** `cli-anything-newapi`
**Dir:** `.planning/v1.2-fork-migration/6-cli-anything-newapi/`
**Status:** `pending`

Ver `.planning/v1.2-fork-migration/6-cli-anything-newapi/PLAN.md`

---

## Criteria

- [ ] `git remote -v` mostra origin (router-ai-atius) e upstream (QuantumNous/new-api)
- [ ] `git fetch upstream` funciona
- [ ] `git fetch origin` funciona
- [ ] `sync-fork.sh --dry-run` executa sem erro
- [ ] `version-bump.sh --check` mostra versão atual
- [ ] `FORK.md` existe e está completo
- [ ] `.github/workflows/sync.yml` existe e é válido
- [ ] `.github/workflows/release.yml` existe e é válido
- [ ] CLI-Anything para NewAPI gerado e instalável
- [ ] `cli-anything-newapi --help` funciona
- [ ] `cli-anything-newapi channel list` funciona
- [ ] Agente consegue executar operações via CLI

## Last Updated
2026-04-21
