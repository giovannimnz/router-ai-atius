# Phase 21: feat-pt-native-pr - Context

**Gathered:** 2026-06-26
**Status:** Ready for planning
**Migration note:** This context was first captured under legacy Phase 8 and moved to Phase 21 on 2026-06-26 so the work follows the current post-Phase-20 sequence.

<domain>
## Phase Boundary

This phase prepares a clean upstream handoff for Brazilian Portuguese support:
push a clean branch to Giovanni's fork, close the polluted PR #5245 with a replacement note, and open a new focused PR against `QuantumNous/new-api`.

The upstream PR must contain only Brazilian Portuguese localization and the minimum wiring required for that locale to be selectable and functional. It must not carry Atius fork runtime customizations, Graphify/GSD artifacts, router/provider/governor work, local docs, or unrelated upstream drift.

</domain>

<decisions>
## Implementation Decisions

### Clean PR Source
- **D-01:** Use `feat/pt-native-i18n-clean` as the canonical source lane for the new upstream PR.
- **D-02:** Treat commit `cd8cb89bb72b1f5551a9f7536f104498ddfb4d75` (`feat: add Portuguese localization`) as the operational source commit. Its direct file list is the expected source set for planning.
- **D-03:** Do not trust a broad `origin/main...HEAD` diff from the clean worktree as the final PR scope by itself. That diff currently includes unrelated upstream drift; planning must reconstruct or filter the PR so the final review diff matches the PT-BR scope.

### Commit Strategy
- **D-04:** The upstream PR should be one single commit, not a preserved local history stack.
- **D-05:** The commit message can use a standard upstream-friendly form such as `feat: add Brazilian Portuguese localization`, unless the planner finds a stronger repo-local convention.

### Upstream PR Scope
- **D-06:** Include Brazilian Portuguese locale files plus the minimum i18n wiring required for the language to appear and work.
- **D-07:** Exclude `.planning/`, local Atius docs, fork-specific router/channel/model/governor changes, production runtime notes, protected-path/fork-sync docs, and any unrelated UI/backend changes.
- **D-08:** The expected PT-BR source set from the clean commit is:
  - `i18n/i18n.go`
  - `i18n/locales/pt.yaml`
  - `web/default/scripts/sync-i18n.mjs`
  - `web/default/src/i18n/config.ts`
  - `web/default/src/i18n/languages.ts`
  - `web/default/src/i18n/locales/_reports/_sync-report.json`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/pt.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/vi.json`
  - `web/default/src/i18n/locales/zh.json`

### PR Hygiene
- **D-09:** Use the upstream PR template at `.github/PULL_REQUEST_TEMPLATE.md`; do not replace it with an ad hoc body.
- **D-10:** Because current git user `giovannimnz <munizgiovanni@hotmail.com>` is not one of the historical core upstream authors observed in local git history, the PR body should state that the contribution was AI-assisted when appropriate.
- **D-11:** Close polluted PR #5245 with a concise comment pointing maintainers to the replacement clean PR. Exact wording is planner discretion.

### the agent's Discretion
The planner may choose the safest mechanical route to produce the final branch: cherry-pick with cleanup, branch from upstream `main` and apply only the source commit's PT-BR changes, or another equivalent method. The invariant is the final PR diff, not the local intermediate commands.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### GSD Scope
- `.planning/ROADMAP.md` - Phase 21 goal and boundary: clean replacement PR for `feat-pt-native`.
- `.planning/STATE.md` - Historical v2.12 handoff notes. Treat as context, not as a source of truth when it conflicts with the decisions above.

### Upstream PR Rules
- `AGENTS.md` - Project rules, especially protected upstream identity and PR disclosure/template requirements.
- `.github/PULL_REQUEST_TEMPLATE.md` - Required upstream PR body structure.

### I18n Implementation Rules
- `web/default/AGENTS.md` - Frontend i18n and Bun/typecheck expectations.
- `i18n/i18n.go` - Backend locale registration point touched by the clean PT-BR commit.
- `web/default/src/i18n/config.ts` - Frontend i18next config touched by the clean PT-BR commit.
- `web/default/src/i18n/languages.ts` - Frontend language list touched by the clean PT-BR commit.
- `web/default/scripts/sync-i18n.mjs` - Locale sync script touched by the clean PT-BR commit.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Clean worktree: `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean` is on branch `feat/pt-native-i18n-clean` and was clean when context was gathered.
- Source commit: `cd8cb89bb72b1f5551a9f7536f104498ddfb4d75` adds PT-BR with 13 changed files and 5137 insertions / 11 deletions.

### Established Patterns
- Backend i18n lives under `i18n/` with YAML locale files.
- Default frontend i18n lives under `web/default/src/i18n/` with flat JSON locale files.
- Frontend scripts use Bun conventions from `web/default/AGENTS.md`.
- The PR template requires human-reviewed summary, duplicate PR/issue check, focused scope, local validation, and no secrets.

### Integration Points
- The branch currently has only `origin` pointing to `https://github.com/giovannimnz/router-ai-atius.git`; planning should ensure a valid `QuantumNous/new-api` upstream remote or equivalent base before producing the PR branch.
- The final PR diff should be checked against the actual upstream base, not just the fork's local `origin/main`.
- The current main repo worktree is dirty with unrelated Phase 20 and fork customization work; Phase 21 execution should happen in the clean PT-native worktree or another isolated branch/worktree.

</code_context>

<specifics>
## Specific Ideas

Giovanni explicitly chose:

- canonical source lane: `feat/pt-native-i18n-clean`;
- one single commit for upstream review;
- scope: PT-BR localization plus minimum wiring only.

</specifics>

<deferred>
## Deferred Ideas

None. The discussion stayed within Phase 21 scope.

</deferred>

---

*Phase: 21-feat-pt-native-pr*
*Context gathered: 2026-06-26*
