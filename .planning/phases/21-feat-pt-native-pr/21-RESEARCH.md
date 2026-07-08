---
phase: 21-feat-pt-native-pr
research_date: "2026-07-04"
baseline: "QuantumNous/new-api upstream/main 1ae757475f9e8dad4ffedf89b3e707756fe8ecf9"
status: complete
---

# Phase 21 Research: Native PT-BR Language Support

## Research Question

How should Brazilian Portuguese be implemented so it is 100% native to current `QuantumNous/new-api` upstream, with no fork-only i18n layer, and still safe to submit upstream later?

## Summary

Current upstream already has three native language systems:

1. Backend Go package `i18n/` using `nicksnyder/go-i18n/v2` and embedded YAML files.
2. Default React frontend `web/default/src/i18n/` using i18next and flat JSON locale files.
3. Classic React frontend `web/classic/src/i18n/` using i18next and its own flat JSON locale files.

Therefore, the correct plan is not to delete `i18n/`. The correct plan is to delete or avoid any **custom** translation mechanism and add `pt` through the existing upstream-native surfaces.

## Evidence From Upstream

### Backend

Files present in `upstream/main`:

- `i18n/i18n.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `i18n/locales/zh-TW.yaml`
- `middleware/i18n.go`

Observed pattern:

- `//go:embed locales/*.yaml`
- explicit load list in `Init()`
- localizer map keyed by constants
- `normalizeLang` handles supported language prefixes
- `SupportedLanguages()` returns the supported codes
- unsupported languages fall back to `en`

Native PT-BR means:

- add `i18n/locales/pt.yaml`
- add `LangPt = "pt"`
- load `locales/pt.yaml`
- pre-create `localizers[LangPt]`
- normalize `pt`, `pt-BR`, and `pt_BR` to `pt`
- include `pt` in `SupportedLanguages()`

Do **not** add root-level `i18n/pt.yaml`.

### Default Frontend

Files present in `upstream/main`:

- `web/default/src/i18n/config.ts`
- `web/default/src/i18n/languages.ts`
- `web/default/src/i18n/static-keys.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/_reports/_sync-report.json`
- `web/default/scripts/sync-i18n.mjs`

Observed pattern:

- locale JSON files are flat under `translation`
- `config.ts` statically imports each locale and includes it in `resources`
- `supportedLngs` is an explicit array
- `languages.ts` owns the UI language list
- `LanguageSwitcher` and `LanguagePreferencesCard` consume `INTERFACE_LANGUAGE_OPTIONS`
- `sync-i18n.mjs` normalizes key ordering and reports missing/extras/untranslated counts
- `static-keys.ts` lists dynamic/static translation keys that may not be found through `t('...')` extraction and should be treated as coverage input

Native PT-BR means:

- add `web/default/src/i18n/locales/pt.json`
- import/add `pt` in `config.ts`
- add `pt` to `supportedLngs`
- add `{ code: 'pt', label: 'Português' }` to `INTERFACE_LANGUAGE_OPTIONS`
- ensure `normalizeInterfaceLanguage('pt-BR')` returns `pt`
- update `sync-i18n.mjs` untranslated detection to include `pt` alongside other Latin-script locales if needed
- verify static keys are covered by `pt.json` wherever they are present in the upstream base locale
- run `bun run i18n:sync` until `_sync-report.json` reports `pt` with all zeros

### Classic Frontend

Files present in `upstream/main`:

- `web/classic/src/i18n/i18n.js`
- `web/classic/src/i18n/language.js`
- `web/classic/src/i18n/locales/en.json`
- `web/classic/src/i18n/locales/zh-CN.json`
- `web/classic/src/i18n/locales/zh-TW.json`
- `web/classic/src/i18n/locales/zh.json`
- `web/classic/src/i18n/locales/fr.json`
- `web/classic/src/i18n/locales/ru.json`
- `web/classic/src/i18n/locales/ja.json`
- `web/classic/src/i18n/locales/vi.json`
- `web/classic/src/components/layout/headerbar/LanguageSelector.jsx`
- `web/classic/src/components/settings/personal/cards/PreferencesSettings.jsx`

Observed pattern:

- `supportedLanguages` is explicit in `language.js`
- `i18n.js` imports and registers each locale explicitly
- top-bar `LanguageSelector` hardcodes each language option
- profile preferences hardcode `languageOptions`
- normalization handles Chinese variants specially and exact-matches other supported codes

Native PT-BR means:

- add `web/classic/src/i18n/locales/pt.json`
- import/add `pt` in `i18n.js`
- add `pt` to `supportedLanguages`
- normalize `pt-BR` and `pt_BR` to `pt`
- add `Português` to both classic language selector and profile preference options

## Duplicate/PR Research

During replanning, upstream PR #5801 was open:

- title: `Add Portuguese translations for various messages`
- files: `i18n/pt.yaml`
- additions: 279

This is related but not equivalent to the target implementation because:

- upstream backend expects locale files under `i18n/locales/*.yaml`
- #5801 does not wire backend language constants/normalization
- #5801 does not add default frontend PT
- #5801 does not add classic frontend PT

Execution must re-check #5801 before preparing an upstream PR.

Upstream issue #2924 was also open:

- title: `Portuguese translation`
- state: open during replanning

Closed PRs #5238 and #5245 were inspected as historical context. They are not usable as final scope because they included fork/runtime/planning changes alongside PT translation work.

## Translation Reuse Research

Read-only inventory on 2026-07-04 found:

- primary backend source: `feat/pt-native-i18n-clean:i18n/locales/pt.yaml` or `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean/i18n/locales/pt.yaml`, aligned 228/228 keys with no placeholder mismatch;
- primary default frontend source for covered keys: linked branch `giovannimnz/router-ai-atius:feat/pt-native-i18n-clean` at `cd8cb89bb72b1f5551a9f7536f104498ddfb4d75`, with 4655/4978 current upstream English keys covered, 323 missing, 20 extras, 0 placeholder mismatch, and 228 same-as-English values to classify;
- default frontend gap supplement: `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean/web/default/src/i18n/locales/pt.json`, whose 4655 overlapping current keys match the linked branch exactly and which covers the 323 linked-branch gaps;
- classic frontend: no complete `web/classic/src/i18n/locales/pt.json` source found; only a small set of exact source-text matches can be reused from default PT;
- historical commits `728bb2e2`, `3f9209e0`, and `05accaf9` are stale wording references only;
- `5f0453fb:docs/TRANSLATION-PT-BR.md` is glossary/style guidance only.

Reuse rule: copy translation values by matching current upstream keys/source text and placeholder sets. For the default frontend, prefer linked-branch translations first for its 4655 matching current keys, then fill only the linked-branch gaps from the external clean worktree. Do not cherry-pick old branches or copy files wholesale when they contain stale keys, extras, fork branding, or runtime/planning changes.

## Validation Strategy

Minimum validation after implementation:

- `git diff --name-status upstream/main...HEAD` shows only native language files, wiring files, and justified tests/scripts.
- no `i18n/pt.yaml` exists.
- backend YAML key parity: `pt.yaml` vs `en.yaml`.
- backend placeholder parity: `pt.yaml` vs `en.yaml`.
- default frontend `bun run i18n:sync` leaves a clean worktree and `_sync-report.json` reports `pt` zeros.
- default frontend JSON key and placeholder parity checks pass.
- classic frontend key parity: `pt.json` vs `en.json`.
- classic frontend placeholder parity checks pass.
- `/usr/local/go/bin/go test ./i18n` or `go test ./i18n` passes.
- `bun run typecheck` from `web/default/` passes.
- relevant `bun run lint`/build checks pass or blockers are documented.
- failing leak checks over both code diff and PR/comment drafts find no `.planning`, Graphify, Obsidian, Podman, provider/governor, runtime DB, secrets, or Atius-only content.

## Research Conclusion

Proceed with five focused implementation plans:

1. create a clean local implementation branch/worktree from current `upstream/main` and produce the translation inventory;
2. add PT-BR natively to backend;
3. add PT-BR natively to default frontend;
4. add PT-BR natively to classic frontend;
5. validate final diff and prepare upstream handoff only after local approval.
