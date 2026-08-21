---
name: router-pt-br-guardian
description: >-
  Specialist for Atius Router PT-BR preservation. Use when PT-BR frontend/backend
  translation, locale registration, language normalization, upstream-sync protection,
  or post-deploy PT-BR verification is requested. Covers router source files,
  classic/default frontend locale wiring, sync workflow disablement, and live bundle
  smoke for https://router.atius.com.br.
---

# Router PT-BR Guardian

Use this skill for PT-BR work in `router-ai-atius`.

## Scope

- `web/default/src/i18n/*`
- `web/classic/src/i18n/*`
- PT-BR locale strings
- `.github/workflows/sync.yml`
- live PT-BR verification on `https://router.atius.com.br`

## Rules

- For `web/default/src/i18n/locales/*.json`, do not hand-edit JSON.
- Write locale changes through `web/default/scripts/add-missing-keys.mjs`.
- Run `bun run i18n:sync` after locale writes.
- Keep `pt-BR` / `pt_BR` normalized to `pt` in router frontend.
- Preserve classic frontend PT visibility too.
- Treat broken upstream auto-sync as translation risk; verify workflow is disabled when requested.

## Verification

Primary gate:

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
./scripts/smoke-pt-br-i18n.sh
```

The smoke must prove:

- source registers `pt`
- source normalizes `pt-BR` / `pt_BR`
- critical PT strings exist
- sync workflow is not scheduled
- public bundle serves PT-capable assets

## Critical files

- `web/default/src/i18n/config.ts`
- `web/default/src/i18n/languages.ts`
- `web/default/src/i18n/locales/pt.json`
- `web/classic/src/i18n/i18n.js`
- `web/classic/src/i18n/language.js`
- `web/classic/src/components/layout/headerbar/LanguageSelector.jsx`
- `web/classic/src/components/settings/personal/cards/PreferencesSettings.jsx`
- `.github/workflows/sync.yml`
- `scripts/smoke-pt-br-i18n.sh`
