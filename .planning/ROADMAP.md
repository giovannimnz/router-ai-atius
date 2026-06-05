# Atius AI Router — Roadmap

## v2.12 — pt Native i18n Integration

Goal: Integrate Portuguese locale into the upstream new-api native i18n infrastructure — zero custom code, only registration points.

### Phase 01: pt Locale Registration ✅ (2026-06-05)

Register `pt` locale in all native i18n registration points (backend Go + frontend React) following the exact same pattern used by fr, ja, ru, vi locales.

| Arquivo | Mudança |
|---|---|
| `i18n/i18n.go` | LangPt + pt.yaml loading + localizer + normalizeLang + SupportedLanguages |
| `i18n/locales/pt.yaml` | 228 keys backend PT-BR |
| `web/default/src/i18n/config.ts` | import pt + resources + supportedLngs |
| `web/default/src/i18n/languages.ts` | opção "Português" |
| `web/default/src/i18n/locales/pt.json` | 4521 keys frontend PT-BR |

Canonical refs:
- `i18n/i18n.go` — backend locale loading
- `web/default/src/i18n/config.ts` — frontend i18next config
- `web/default/src/i18n/languages.ts` — language options
- `web/default/src/components/language-switcher.tsx` — UI switcher

---

## Up Next

- [ ] Deploy + browser validation — construir Docker image, rodar no servidor, validar visualmente
- [ ] Push `feat/pt-native` para origin
