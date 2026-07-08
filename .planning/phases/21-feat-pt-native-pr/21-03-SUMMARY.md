# 21-03 Summary

## Outcome

- Added `web/default/src/i18n/locales/pt.json`.
- Registered `pt` in `web/default/src/i18n/config.ts`.
- Added `Português` and `pt` normalization to `web/default/src/i18n/languages.ts`.
- Updated `web/default/scripts/sync-i18n.mjs` so `pt` participates in untranslated detection.

## Translation Fill Breakdown

- `4489` keys reused from the linked branch `feat/pt-native-i18n-clean`
- `86` keys reused from `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean`
- `403` keys translated/repaired during execution

## Validation

- `bun run i18n:sync` generated `_sync-report.json` with `pt.missingCount=0`, `pt.extrasCount=0`, `pt.untranslatedCount=0`
- Explicit parity check passed:
  - no missing keys
  - no extra keys
  - no placeholder drift across `{{...}}`, `${...}`, and `{name}`
  - no unreviewed same-as-English values outside `21-SAME-AS-ENGLISH-LITERALS.json`
- `bun --eval "normalizeInterfaceLanguage(...)"` passed for `pt`, `pt-BR`, `pt_BR`
- `bun run typecheck` passed
- `bun run build` passed

## Blocker

- `bun run lint` fails on pre-existing upstream lint debt outside the PT diff. Example categories:
  - unrelated `typescript/no-import-type-side-effects`
  - unrelated `eslint/no-nested-ternary`
  - unrelated `react/no-array-index-key`
  - unrelated `unicorn/prefer-string-replace-all`
