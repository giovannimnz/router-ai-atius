---
phase: 21
reviewers: [codex]
failed_reviewers: [claude]
reviewed_at: 2026-07-04T20:22:48-03:00
plans_reviewed:
  - 21-01-PLAN.md
  - 21-02-PLAN.md
  - 21-03-PLAN.md
  - 21-04-PLAN.md
  - 21-05-PLAN.md
convergence:
  cycles: 3
  focused_closure: true
  current_high: 0
  current_actionable: 0
---

# Cross-AI Plan Review - Phase 21

## Reviewer Availability

`claude` was installed but failed the non-interactive review preflight with:

```text
Invalid API key - Fix external API key
```

`codex exec --ephemeral` was available and completed the plan review in an isolated process. The local Codex session is the active runtime, so this is not a different vendor model, but it is still a separate CLI review process with the full review prompt and repository access.

## Cycle 1 - Codex Review

**Result:** `CYCLE_SUMMARY: current_high=0 current_actionable=2`

### Strengths

- Clean-lane discipline is explicit in `21-01`: fresh `upstream/main`, clean branch/worktree, and `.planning` kept outside the future PR branch.
- Backend plan targets the current native `i18n/locales/*.yaml` embed/localizer pattern.
- Default and classic frontend plans target their independent native i18next surfaces.
- PR hygiene is covered by duplicate search, template use, leak checks, and path allowlist.

### Actionable Concerns

1. `21-03` and `21-04` placeholder checks only compared `{{...}}`, which could miss `${...}` and `{name}` fragments in frontend locale strings.
2. `21-02` described `testify/require` and `testify/assert` as conditional, while repository policy requires them for new or substantially rewritten Go backend tests.

### Resolution

- `21-03` and `21-04` now require placeholder parity across `{{...}}`, `${...}`, and `{name}` token families, plus same-as-English fallback detection.
- `21-02` now requires `github.com/stretchr/testify/require` and `github.com/stretchr/testify/assert` explicitly.

## Cycle 2 - Codex Review

**Result:** `CYCLE_SUMMARY: current_high=0 current_actionable=2`

### Confirmed Resolved

- Cycle 1 placeholder-token coverage was incorporated into both frontend plans.
- Cycle 1 backend `testify` requirement was incorporated.

### Actionable Concerns

1. Same-as-English fallback checks needed a reviewed classification/allowlist for legitimate brand/code literals such as API, OpenAI, GitHub, and protected identity strings.
2. `21-05` leak checks used bare `Bearer`, which could false-fail legitimate user-facing locale text.

### Resolution

- `21-01` now produces `21-SAME-AS-ENGLISH-LITERALS.json` with reviewed key arrays for `web/default` and `web/classic`.
- `21-03` and `21-04` now fail only unclassified equal English/PT values outside that reviewed allowlist.
- `21-05` task-level leak checks now match token-shaped Bearer secrets, not bare `Bearer` text.

## Cycle 3 - Codex Review

**Result:** `CYCLE_SUMMARY: current_high=0 current_actionable=1`

### Confirmed Resolved

- The same-as-English literal allowlist and frontend parity checks were accepted.

### Remaining Concern

`21-05` task-level leak checks were fixed, but the aggregate `<verification>` block still used bare `Bearer`.

### Resolution

The aggregate `21-05` verification block now uses the same token-shaped patterns:

```text
Authorization:\s*Bearer\s+|Bearer\s+[A-Za-z0-9._~+/-]{12,}
```

## Focused Closure Review

**Result:** `CYCLE_SUMMARY: current_high=0 current_actionable=0`

The focused reviewer checked the remaining `21-05` Bearer concern and confirmed:

```text
## Current HIGH Concerns
None.

## Current Actionable Non-HIGH Concerns
None.
```

## Consensus Summary

### Agreed Strengths

- Phase 21 is scoped narrowly to native PT-BR i18n implementation and upstream PR handoff.
- The plan set covers backend, default frontend, classic frontend, validation, duplicate search, leak checks, and PR template hygiene.
- Review concerns were incorporated into executable plan tasks and verification commands rather than left as prose.

### Agreed Concerns

None remain unresolved after focused closure.

### Divergent Views

None material to execution readiness. The only reviewer limitation is operational: `claude` could not run because the configured external API key is invalid.

