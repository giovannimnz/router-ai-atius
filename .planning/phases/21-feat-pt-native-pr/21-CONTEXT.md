# Phase 21: feat-pt-native-pr - Context

**Gathered:** 2026-06-26
**Replanned:** 2026-07-04
**Status:** Ready for native PT-BR implementation planning
**Migration note:** This context was first captured under legacy Phase 8 and moved to Phase 21 on 2026-06-26 so the work follows the current post-Phase-20 sequence.

<domain>
## Phase Boundary

This phase implements Brazilian Portuguese first in this fork, using only the native language extension points that exist in current `QuantumNous/new-api` upstream. If the local result is accepted, the same diff should be suitable for a clean upstream PR with no fork-specific runtime, provider, Graphify, GSD, Obsidian, Podman, DB, catalog, or Atius branding changes.

The phrase "remove custom i18n" means remove or avoid any fork-only translation mechanism. It does **not** mean deleting upstream `i18n/` directories. Current upstream still uses:

- backend `i18n/` with `go-i18n` and embedded YAML files;
- default frontend `web/default/src/i18n/` with i18next and flat JSON files;
- classic frontend `web/classic/src/i18n/` with i18next and flat JSON files.

</domain>

<decisions>
## Implementation Decisions

### Upstream Baseline
- **D-01:** Use current `QuantumNous/new-api` `upstream/main` as the implementation baseline. The baseline validated during replanning was `1ae757475f9e8dad4ffedf89b3e707756fe8ecf9` on 2026-07-04.
- **D-02:** Revalidate `upstream/main` before execution if any time has passed; do not assume the old clean commit `cd8cb89bb72b1f5551a9f7536f104498ddfb4d75` is still complete.
- **D-03:** The older clean branch remains useful as translation source material only. It is no longer the final scope contract.

### Native Language Scope
- **D-04:** Backend Portuguese must be added through upstream-native `i18n/locales/pt.yaml` plus the minimal `i18n/i18n.go` wiring (`LangPt`, file load, pre-created localizer, normalize, supported languages).
- **D-05:** Do not add root-level `i18n/pt.yaml`. PR #5801 currently does that and is therefore related but not equivalent to this phase's full native implementation.
- **D-06:** Default frontend Portuguese must be added through `web/default/src/i18n/locales/pt.json`, `web/default/src/i18n/config.ts`, and `web/default/src/i18n/languages.ts`, plus sync-script support for `pt` untranslated detection if needed.
- **D-07:** Classic frontend Portuguese must also be added for 100% native parity, because upstream still ships and wires `web/classic` language support independently.
- **D-08:** Existing language pickers already iterate upstream language lists in default UI, but classic UI has explicit language selector/preference lists. Add `pt` there using the same pattern as `fr`, `ru`, `ja`, and `vi`.

### Coverage and Quality
- **D-09:** PT-BR coverage target is 100% parity with upstream base locale key sets for backend, default frontend, and classic frontend.
- **D-10:** Default frontend `bun run i18n:sync` must report `missingCount=0`, `extrasCount=0`, and `untranslatedCount=0` for `pt`.
- **D-11:** Preserve placeholders such as `{{count}}`, ICU plural suffixes, JSON examples, URLs, model names, brand names, and protected project identity strings.
- **D-12:** Add tests only where they protect observable language behavior. Do not add reward-hacking tests that merely count files or private implementation details.
- **D-13:** Reuse existing PT-BR translations before creating any new translation. Current fork files, the previous clean PT lane, and historical `pt`/`pt-BR` artifacts are translation sources, not final scope contracts.
- **D-14:** For classic frontend, first reuse matching default-frontend PT strings for identical English keys, then translate only the remaining classic-specific gaps.
- **D-15:** Do not cherry-pick or copy whole historical branches. Reuse translation values through an inventory/mapping, because historical branches include unrelated fork/runtime/planning changes.
- **D-16:** Same-as-English values must be classified as legitimate brand/code literals or unresolved translation gaps before claiming 100% coverage.

### Upstream PR Hygiene
- **D-17:** Keep implementation commits upstream-ready: no `.planning/`, Graphify, Obsidian, runtime docs, provider routing, DB/catalog, Podman, or fork-only changes in the code branch.
- **D-18:** Use `.github/PULL_REQUEST_TEMPLATE.md` if a PR is prepared.
- **D-19:** Search upstream PRs/issues for duplicate Portuguese work before opening a PR. During replanning, issue #2924 was open as the Portuguese translation request and PR #5801 was open but only touched `i18n/pt.yaml`; re-check both before PR creation.
- **D-20:** Treat closed PRs #5238 and #5245 as contaminated historical context only. They are evidence for why a replacement clean PR is needed, not a reusable scope.
- **D-21:** Because local git user `giovannimnz <munizgiovanni@hotmail.com>` is not one of the historical core upstream authors observed previously, the PR body should disclose AI assistance when appropriate.
- **D-22:** Leak checks must fail when forbidden fork/planning/runtime/secrets text appears in either the code diff or PR/comment draft text.

</decisions>

<canonical_refs>
## Canonical References

**Downstream execution must read these before editing code.**

### GSD Scope
- `.planning/ROADMAP.md` - Phase 21 goal and boundaries.
- `.planning/REQUIREMENTS.md` - Phase 21 requirements.
- `.planning/STATE.md` - Active project state; treat as context, not as a substitute for upstream source.
- `.planning/phases/21-feat-pt-native-pr/21-RESEARCH.md` - Current upstream validation.
- `.planning/phases/21-feat-pt-native-pr/21-TRANSLATION-INVENTORY.md` - PT-BR reuse source order and unsafe reuse rules.

### Upstream PR Rules
- `AGENTS.md` - Project rules, especially i18n, protected identity, tests, and PR disclosure/template rules.
- `.github/PULL_REQUEST_TEMPLATE.md` - Required PR body structure.
- `web/default/AGENTS.md` - Frontend i18n, Bun, typecheck, lint, and build expectations.

### Backend Native Language
- `i18n/i18n.go` - Backend locale registration, normalization, and supported language list.
- `i18n/locales/en.yaml` - Backend base locale.
- `i18n/locales/zh-CN.yaml`, `i18n/locales/zh-TW.yaml` - Existing backend locale examples.

### Default Frontend Native Language
- `web/default/src/i18n/config.ts` - i18next resources and supported languages.
- `web/default/src/i18n/languages.ts` - user-facing interface language list and normalization.
- `web/default/src/i18n/static-keys.ts` - default frontend dynamic/static translation keys that may not be discovered by regex extraction.
- `web/default/src/i18n/locales/en.json` - default frontend base locale.
- `web/default/scripts/sync-i18n.mjs` - default frontend locale parity and untranslated report tool.
- `web/default/src/components/language-switcher.tsx` and `web/default/src/features/profile/components/language-preferences-card.tsx` - default UI consumers of `INTERFACE_LANGUAGE_OPTIONS`.

### Classic Frontend Native Language
- `web/classic/src/i18n/i18n.js` - classic i18next resource registry.
- `web/classic/src/i18n/language.js` - classic supported languages and normalization.
- `web/classic/src/i18n/locales/en.json` - classic base locale.
- `web/classic/src/components/layout/headerbar/LanguageSelector.jsx` - classic top-bar language selector.
- `web/classic/src/components/settings/personal/cards/PreferencesSettings.jsx` - classic profile language preferences.

</canonical_refs>

<code_context>
## Existing Code Insights

### Current Upstream Pattern Validated on 2026-07-04
- Backend upstream has only `i18n/locales/en.yaml`, `zh-CN.yaml`, and `zh-TW.yaml`.
- Default frontend upstream has only `web/default/src/i18n/locales/en.json`, `zh.json`, `fr.json`, `ru.json`, `ja.json`, `vi.json`, and `_reports/_sync-report.json`.
- Default frontend also has `web/default/src/i18n/static-keys.ts`; execution should treat it as coverage input for dynamic labels even if the sync script does not currently import it.
- Classic frontend upstream has `en.json`, `fr.json`, `ja.json`, `ru.json`, `vi.json`, `zh.json`, `zh-CN.json`, and `zh-TW.json`.
- Backend `i18n/i18n.go` currently normalizes `zh-*` and `en`; PT support should match this switch style.
- Default `normalizeInterfaceLanguage` collapses `zh*` to `zh` and otherwise accepts codes present in `INTERFACE_LANGUAGE_OPTIONS`.
- Classic `normalizeLanguage` preserves supported codes from `supportedLanguages`; `pt-BR` should normalize to `pt` if `pt` is the supported code.

### Existing Open Upstream Work
- Issue #2924 (`Portuguese translation`) was open during replanning and is the current upstream request for Portuguese translation.
- PR #5801 (`Add Portuguese translations for various messages`) was open during replanning.
- #5801 currently touches only `i18n/pt.yaml`, not `i18n/locales/pt.yaml`, default frontend, or classic frontend.
- Treat #5801 as a duplicate-risk preflight item, not as a blocker, unless it changes scope before execution.
- Closed PRs #5238 and #5245 carried useful PT translation attempts but were contaminated with fork/runtime/planning changes; they should not be reused as PR scope.

### Worktree Safety
- The main checkout is dirty with unrelated Phase 20/24 and runtime work.
- Implementation should happen in a clean worktree or branch based on `upstream/main`.
- Do not revert unrelated dirty files in the main checkout.

### Codex Execution Lane
- This project runs GSD with `runtime=codex` and `workflow.use_worktrees=false`; therefore `gsd-execute-phase 21` must treat this dirty checkout as the planning/control checkout only.
- Plan `21-01` must create or refresh a clean implementation worktree from current `upstream/main`, record the exact baseline commit and `implementation_worktree` path in `21-TRANSLATION-INVENTORY.md`, and verify the lane is clean before any code edits.
- Plans `21-02`, `21-03`, `21-04`, and implementation-diff parts of `21-05` must run all code edits, `git diff upstream/main...HEAD`, builds, and tests from the recorded implementation worktree, not from this planning checkout.
- Planning artifacts, summaries, reviews, Graphify outputs, and handoff notes remain in this planning checkout unless a plan explicitly says otherwise.
- If the recorded implementation worktree is missing, dirty before that plan starts, not based on current `upstream/main`, or not listed in `git worktree list`, stop before edits and repair `21-01`.

### Translation Reuse Sources
- Current fork/local files such as `i18n/locales/pt.yaml`, `web/default/src/i18n/locales/pt.json`, and any `pt-BR` locale files if present.
- Previous clean lane `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean`, especially its PT locale files and the amended clean commit if it still exists.
- Historical v1.6 PT-BR artifacts documented in Obsidian as prior coverage evidence.
- Matching default-frontend PT strings for classic frontend keys with identical English source text.
- Read-only audit and recheck on 2026-07-04 found the linked branch source at 228/228 backend keys and 4655/4978 current default frontend keys, including substantial screen/menu translations; the external clean worktree supplements exactly the 323 current default keys missing from the linked branch. No complete classic frontend PT source was found.
- Translation reuse must be value-level reuse, not branch-level cherry-pick, because old PT work includes extras and unrelated sync changes. For default frontend, prefer linked-branch translation values first, then supplement its current-key gaps from the external clean worktree.

</code_context>

<specifics>
## Specific Ideas

Giovanni explicitly wants:

- validate against current `QuantumNous/new-api` upstream, not the older local plan;
- remove any custom i18n approach;
- implement Portuguese as a fully native language in the same pattern as existing upstream languages;
- implement locally first in this fork;
- keep the result upstream-contributable if approved.

</specifics>

<deferred>
## Deferred Ideas

- Actually opening the upstream PR is deferred until Giovanni approves the local implementation.
- Runtime deployment/UAT is deferred to execution/verification; planning only defines the code and validation path.

</deferred>

---

*Phase: 21-feat-pt-native-pr*
*Context replanned: 2026-07-04*
