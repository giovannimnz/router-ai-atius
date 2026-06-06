# STATE.md

**Project:** Atius AI Router
**Current milestone:** v2.12 — pt Native i18n Integration
**Status:** ✅ Complete (all phases done — 4/4)

## Progress

| Phase | Status | Date |
|---|---|---|
| 01 — pt Locale Registration (Go + React SPA) | ✅ Complete | 2026-06-05 |
| 02 — pt Fumadocs i18n (Docs) | ✅ Complete | 2026-06-05 |
| 03 — PT Docs Bugfixes (hreflang + guide) | ✅ Complete | 2026-06-05 |
| 04 — Prod Docs Bugfixes (Apache + logo + lang order) | ✅ Complete | 2026-06-06 |

## Summary

All 4 systems of the v2.12 milestone are complete. The `pt` locale is now registered in:

1. **Backend Go** (go-i18n): 5 registration points + 228 key YAML
2. **Frontend SPA** (i18next): 4 registration points + 4521 key JSON
3. **Docs** (Fumadocs): 3 config points + 294 MDX files
4. **fork-sync**: pt-content mirror updated, protected_globs active
5. **Production routing** (Apache vhost): 4 locales consistent, /_next/ proxied to 3003
6. **Visual validation**: mmx vision confirms "completamente estilizada" Fumadocs layout

All changes follow the native i18n infrastructure of each platform. Zero custom code.

## Last Activity

2026-06-06: Phase 04 complete. Apache patch in /etc/apache2/sites-enabled/, repo
commit f78631367. 4/4 locales return `x-powered-by: Next.js`. CSS rule count
2 → 392. Visual validation: PT-BR Fumadocs fully styled.

## Pending (user action)

- Cloudflare cache purge for stale `/pt/docs/` (x-new-api-version) and
  `/assets/atius-logo.svg` (404) entries. Origin Apache is now correct;
  new requests will succeed. Old cached entries expire naturally in ~24h
  or via manual CF purge.
- Push `feat/pt-native` to fork origin (waiting for user approval).
- Push upstream for new-api-docs-v1 PT changes (waiting).
