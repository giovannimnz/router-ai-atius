# Phase 21: feat-pt-native-pr - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md; this log preserves the alternatives considered.

**Date:** 2026-06-26
**Phase:** 21-feat-pt-native-pr
**Areas discussed:** Clean PR source, commit strategy, upstream PR scope
**Migration note:** Discussion was originally captured under legacy Phase 8 and moved to Phase 21 on 2026-06-26.

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

---

## Replanning: Native Upstream Language Parity

**Date:** 2026-07-04
**Prompt:** Giovanni asked to validate the current `QuantumNous/new-api` upstream language pattern and replan PT-BR as a fully native language implementation, implemented locally first and kept upstream-contributable.

**Outcome:** The plan no longer treats the old clean commit as the final scope contract. It now uses current `upstream/main` as the source of truth and includes backend, default frontend, and classic frontend native language surfaces.

**Clarification:** "Remove custom i18n" does not mean deleting upstream `i18n/`; it means avoiding fork-only translation mechanisms and adding `pt` through the existing upstream-native `i18n/` and frontend i18next patterns.

**Replanning Review:** After subagent review, the original single broad plan was split into five focused plans: translation inventory/clean lane, backend, default frontend, classic frontend, and upstream handoff. The revised plan adds explicit parity/placeholder/normalization checks, issue/PR duplicate search including #2924 and #5801, failing path/leak checks, and a value-level reuse inventory so existing PT-BR translations are reused without cherry-picking polluted fork branches.

**Linked Branch Revalidation:** Giovanni asked to re-check `https://github.com/giovannimnz/router-ai-atius/tree/feat/pt-native-i18n-clean`. The branch resolves to `cd8cb89bb72b1f5551a9f7536f104498ddfb4d75`, has backend PT 228/228, and covers 4655/4978 current default frontend keys with many screen/menu translations. The plan now treats this linked branch as the primary default source for covered keys and uses `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean` only to fill the 323 linked-branch gaps.
