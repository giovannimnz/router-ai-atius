---
phase: 27-codex-official-docs-ci-and-release-alignment
plan: "01"
subsystem: codex-docs-ci
tags: [codex, docs, ci, auth, release]
requires:
  - phase: 26-codex-dynamic-discovery-and-curated-catalog
    provides: Curated local Codex contract and dynamic discovery baseline
provides:
  - PT-BR runbook for official Codex CI/auth/release guidance
  - Workflow alignment for `openai/codex-action`
  - Repo-level separation between API-key automation and advanced ChatGPT-managed auth
affects: [docs, github-actions, planning, operator-runbooks]
tech-stack:
  added: [PT-BR runbook for official Codex CI/auth/release alignment]
  patterns:
    - "Official OpenAI/Codex docs are the source of truth for CI/auth behavior."
    - "GitHub Actions should prefer first-class `openai/codex-action` inputs over bespoke CLI config shims."
key-files:
  created:
    - .planning/phases/27-codex-official-docs-ci-and-release-alignment/27-CONTEXT.md
    - .planning/phases/27-codex-official-docs-ci-and-release-alignment/27-RESEARCH.md
    - .planning/phases/27-codex-official-docs-ci-and-release-alignment/27-01-PLAN.md
    - .planning/phases/27-codex-official-docs-ci-and-release-alignment/27-01-SUMMARY.md
    - docs/CODEX-CI-AUTH-RELEASE.md
  modified:
    - .github/workflows/sync.yml
    - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md
    - docs/CI-RELEASE-WATCHDOG.md
key-decisions:
  - "GitHub Actions Codex usage in this fork should keep `openai/codex-action@v1` and align with its first-class `effort` input."
  - "API key auth remains the default automation path; ChatGPT-managed auth stays private-runner-only and must never be normalized for public/open-source workflows."
  - "PT-BR operator output remains local fork policy even when the authoritative source is English official documentation."
patterns-established:
  - "When Phase 27-style guidance is updated, the operator manual should link to a dedicated Codex CI/auth/release runbook instead of embedding all policy inline."
requirements-completed:
  - PHASE-27-OFFICIAL-DOCS-FIRST
  - PHASE-27-CODEX-CI-AUTH
  - PHASE-27-PTBR-RELEASE-OPS-DOCS
coverage:
  - id: D1
    description: Local docs point to official OpenAI/Codex references and capture the correct CI/auth split
    requirement: PHASE-27-OFFICIAL-DOCS-FIRST
    verification:
      - kind: other
        ref: "rg checks for `codex exec`, `openai/codex-action`, `auth.json`, and `Docs MCP` across changed docs"
        status: pass
    human_judgment: false
  - id: D2
    description: Existing GitHub Actions Codex usage aligns with official action inputs
    requirement: PHASE-27-CODEX-CI-AUTH
    verification:
      - kind: other
        ref: "PyYAML parse of `.github/workflows/sync.yml` and `.github/workflows/release.yml`"
        status: pass
    human_judgment: false
  - id: D3
    description: Phase artifacts and top-level planning show Phase 27 as executable and then complete
    requirement: PHASE-27-PTBR-RELEASE-OPS-DOCS
    verification:
      - kind: other
        ref: "`node \"$HOME/.codex/gsd-core/bin/gsd-tools.cjs\" query init.execute-phase 27` after summary"
        status: pass
    human_judgment: false
duration: 1 session
completed: 2026-07-08
status: complete
---

# Phase 27 Plan 01 Summary

**The fork now has an explicit PT-BR runbook for official Codex CI/auth/release behavior, and the existing Codex GitHub Actions usage is aligned with the official action contract.**

## Performance

- **Duration:** 1 session
- **Completed:** 2026-07-08
- **Tasks:** 3 workstreams
- **Files modified:** 3
- **Files created:** 5

## Accomplishments

- Added `docs/CODEX-CI-AUTH-RELEASE.md` as the dedicated PT-BR runbook for official Codex CI/auth/release guidance.
- Linked the new runbook from `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` and `docs/CI-RELEASE-WATCHDOG.md`.
- Updated `.github/workflows/sync.yml` to use the official `effort` input for `openai/codex-action@v1` instead of encoding reasoning effort through an internal config shim in `codex-args`.
- Materialized the missing Phase 27 planning artifacts (`27-CONTEXT`, `27-RESEARCH`, `27-01-PLAN`) and completed the phase with a summary.

## Task Commits

No commit was created in this run.

## Decisions Made

- API key remains the default automation path.
- ChatGPT-managed auth stays documented as an advanced private-runner-only pattern.
- Official OpenAI/Codex docs are now explicit source of truth for this topic inside the repo.

## Deviations from Plan

No product/runtime deviation. The only procedural deviation was that Phase 27 had no plan directory at execution start, so the missing planning artifacts were created before execution.

## Issues Encountered

- `init.execute-phase 27` initially returned `phase_dir: null` and `plan_count: 0`; this was corrected by creating the phase planning artifacts before execution.
- `ROADMAP.md` also had stale closeout markers for Phases 24 and 25, which were reconciled as part of the broader closeout sweep.

## Next Phase Readiness

With Phase 27 complete, the `v2.13` milestone is ready to be treated as closed. The next open tracks are outside this milestone:

- Phase 21 (`v2.12`) PT-native PR handoff
- Phase 22 k3s migration
- Phase 23 long-context alias validation

## Verification Results

- `python3` + `yaml.safe_load` on `.github/workflows/sync.yml` - PASS
- `python3` + `yaml.safe_load` on `.github/workflows/release.yml` - PASS
- `rg -n "codex exec|openai/codex-action|API key|auth.json|Docs MCP|ChatGPT-managed"` across changed docs and workflow - PASS
- `git diff --check` on changed Phase 27/doc/workflow files - PASS

## Self-Check: PASSED

- Official docs first: yes
- CI/auth alignment with official Codex guidance: yes
- PT-BR-first operator output preserved: yes

---
*Phase: 27-codex-official-docs-ci-and-release-alignment*
*Completed: 2026-07-08*
