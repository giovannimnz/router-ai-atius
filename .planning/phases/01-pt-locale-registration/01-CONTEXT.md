# Phase 01: pt Locale Registration - Context

**Gathered:** 2026-06-05
**Status:** Ready for planning

## Phase Boundary

Register `pt` (Portuguese) locale in all native i18n registration points of the new-api codebase — backend Go (go-i18n) and frontend React (i18next). Follow the EXACT same pattern used by existing locales: `fr`, `ja`, `ru`, `vi`. Zero custom code, zero new abstractions.

This phase delivers:
- `pt` locale selectable in the language switcher dropdown
- Backend API messages translated to PT-BR (all 279 keys)
- Frontend UI rendering in PT-BR when `pt` is selected
- All existing infrastructure unchanged (language detection, user preference persistence, fallback chain)

## Implementation Decisions

### D-01: Locale Code
- **`pt`** (2-letter ISO 639-1, matches fr/ja/ru/vi pattern)
- config.ts `load: 'languageOnly'` collapses pt-BR → pt automatically
- Backend `normalizeLang("pt-BR")` prefix-matches to LangPt

### D-02: Backend Translation Scope
- **All 279 keys from en.yaml** — translated in first commit
- Covers: common, auth, token, redemption, user, channel, model, vendor, group, quota, subscription, payment, topup, checkin, passkey, 2fa, rate_limit, setting, deployment, performance, ability, oauth, distributor, custom_oauth
- go-i18n auto-fallbacks to en for any missing keys

### D-03: Frontend Locale Source
- **Reuse existing pt.json from stash** (`stash@{0}` on branch `portuguese-translation-clean`)
- File contains 3910 keys with 1907 already translated
- Adapt to match current en.json key set (the stash may be from an older version)
- Run `bun run i18n:sync` to reconcile key count

### D-04: Validation
- **typecheck + build + browser validation**
- Go: `go build` must succeed with pt.yaml embedded
- Frontend: `bun run typecheck` + `bun run build` must pass
- Browser: chrome-devtools navigation to confirm language switcher shows "Português" and UI renders in PT-BR

## Canonical References

### Registration Points (MUST read before implementing)
- `i18n/i18n.go` — backend locale loading, Init(), normalizeLang(), SupportedLanguages()
- `web/default/src/i18n/config.ts` — frontend i18next config, resources, supportedLngs
- `web/default/src/i18n/languages.ts` — INTERFACE_LANGUAGE_OPTIONS array
- `web/default/src/components/language-switcher.tsx` — dropdown UI component
- `web/default/src/features/profile/components/language-preferences-card.tsx` — /profile settings

### Infrastructure (read for context)
- `i18n/keys.go` — all 332 message key constants
- `middleware/i18n.go` — language detection middleware
- `dto/user_settings.go` — UserSetting.Language field

### Pattern Locales
- `i18n/locales/en.yaml` — 279 keys (source of truth for backend)
- `web/default/src/i18n/locales/en.json` — source of truth for frontend
- `web/default/src/i18n/locales/fr.json` — example of a non-en locale registration

## Existing Code Insights

### Reusable Assets
- `normalizeLang()` in i18n.go — already handles prefix matching, just add `pt` case
- `LanguageDetector` in config.ts — auto-detects browser language
- `LanguageSwitcher` component — dropdown already iterates INTERFACE_LANGUAGE_OPTIONS
- `LanguagePreferencesCard` — /profile select already persists to backend

### Established Patterns
- Backend locale files: YAML format, key = dotted path (`common.invalid_params`)
- Frontend locale files: flat JSON, key = English source string
- Registration: import → resources → supportedLngs → INTERFACE_LANGUAGE_OPTIONS (4-step pattern)
- Detection chain: UserSetting → Accept-Language header → default `en`

### Integration Points
- `i18n/i18n.go::Init()` — add `pt.yaml` to file list + create localizer
- `i18n/i18n.go::normalizeLang()` — add `pt` prefix case
- `i18n/i18n.go::SupportedLanguages()` — add LangPt
- `web/default/src/i18n/config.ts` — import pt + add to resources + supportedLngs
- `web/default/src/i18n/languages.ts` — add `{ code: 'pt', label: 'Português' }`

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 01-pt-locale-registration*
*Context gathered: 2026-06-05*
