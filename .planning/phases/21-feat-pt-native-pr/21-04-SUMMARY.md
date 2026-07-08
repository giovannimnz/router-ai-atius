# 21-04 Summary

## Outcome

- Added `web/classic/src/i18n/locales/pt.json`.
- Registered `pt` in `web/classic/src/i18n/i18n.js`.
- Added `pt` and `pt-BR`/`pt_BR` normalization in `web/classic/src/i18n/language.js`.
- Added `Português` to:
  - `web/classic/src/components/layout/headerbar/LanguageSelector.jsx`
  - `web/classic/src/components/settings/personal/cards/PreferencesSettings.jsx`

## Translation Fill Breakdown

- `718` classic values reused from default PT because the classic English source matched an existing default English source exactly
- `3113` classic values translated during execution

## Validation

- Explicit classic parity check passed:
  - no missing keys
  - no extra keys
  - no placeholder drift across `{{...}}`, `${...}`, and `{name}`
  - no unreviewed same-as-English values outside `21-SAME-AS-ENGLISH-LITERALS.json`
- `bun --eval "normalizeLanguage(...)"` passed for `pt`, `pt-BR`, `pt_BR`
- `bun run build` passed

## Blockers

- `bun run i18n:lint` fails with `106` pre-existing issues in unrelated classic files, including hardcoded strings and interpolation warnings outside the PT diff.
- `bun run lint` fails because Prettier wants formatting updates in `58` existing files outside the PT scope.
