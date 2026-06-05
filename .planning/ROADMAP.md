# Atius AI Router — Roadmap

## v2.12 — pt Native i18n Integration

Goal: Integrate Portuguese locale into the upstream new-api native i18n infrastructure — zero custom code, only registration points.

### Phase 01: pt Locale Registration

Register `pt` locale in all native i18n registration points (backend Go + frontend React) following the exact same pattern used by fr, ja, ru, vi locales.

Canonical refs:
- `i18n/i18n.go` — backend locale loading
- `web/default/src/i18n/config.ts` — frontend i18next config
- `web/default/src/i18n/languages.ts` — language options
- `web/default/src/components/language-switcher.tsx` — UI switcher
