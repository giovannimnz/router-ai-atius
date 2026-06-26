# Phase 8: feat-pt-native-pr - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md; this log preserves the alternatives considered.

**Date:** 2026-06-26
**Phase:** 8-feat-pt-native-pr
**Areas discussed:** Clean PR source, commit strategy, upstream PR scope

---

## Clean PR Source

| Option | Description | Selected |
|--------|-------------|----------|
| `feat/pt-native-i18n-clean` | Best for upstream. Uses the already separated lane and reduces risk of leaking local Atius changes. | yes |
| `feat/pt-native` | More direct if the current branch should be reused exactly, with higher contamination risk. | |
| New branch from upstream `main` with only PT files ported | Most rigid and predictable, but more work. | |

**User's choice:** `1` - use `feat/pt-native-i18n-clean`.
**Notes:** The selected lane is canonical, but the final PR diff must still be filtered/reconstructed if needed so it contains only the PT-BR upstream scope.

---

## Commit Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| One single commit | Best for upstream review: smaller surface, direct review, easy revert. | yes |
| Few commits by theme | Keeps some structure, but can add review noise. | |
| Preserve existing commits | Keeps local traceability, but highest risk of dirty history. | |

**User's choice:** `1` - one single commit.
**Notes:** The recommended upstream-style commit message is `feat: add Brazilian Portuguese localization`, subject to planner validation against repo conventions.

---

## Upstream PR Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Only PT-BR upstream | Only the files strictly needed to add/adjust Brazilian Portuguese. | |
| PT-BR + minimum integration wiring | PT-BR plus indispensable wiring so the locale appears and works. | yes |
| PT-BR + tests/validation files | Adds tests/scripts directly needed to prove i18n behavior. | |

**User's choice:** `2` - PT-BR plus minimum integration wiring.
**Notes:** The PR must not include Atius fork runtime customizations, `.planning/`, Graphify/GSD artifacts, router provider/governor work, production docs, or unrelated upstream drift.

---

## the agent's Discretion

- Exact mechanical branch creation strategy, as long as the final PR diff is clean.
- Exact close-comment wording for polluted PR #5245.
- Exact local validation command set, guided by changed files and upstream scripts.

## Deferred Ideas

None.
