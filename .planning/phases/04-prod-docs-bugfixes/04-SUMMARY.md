---
status: complete
phase: 04
completed: 2026-06-06
---

# Phase 04: Prod Docs Bugfixes — Summary

## What was built

Three post-deploy fixes for the Atius AI Router docs site, all proven via curl + chromium/CDP visual validation:

1. **D-02 (Apache pt proxy):** Added `ProxyPass /pt/ http://127.0.0.1:3003/pt/` to the
   SSL vhost. Without it, `/pt/docs/` fell through to the catch-all `/` and was
   served as the Go SPA 404 page (x-new-api-version header). Now returns
   `x-powered-by: Next.js` + `x-nextjs-cache: HIT` like the other 3 locales.

2. **D-03 (Atius logo alias):** Added `Alias /assets/atius-logo.svg` and
   `Alias /assets/atius-logo.png` pointing to `/var/www/atius/atius-logo.{svg,png}`
   (already existed on disk). The Fumadocs header references these paths
   directly. Apache now serves 200 OK with proper Content-Type.

3. **D-03b (Bonus — `/_next/` proxy, the real CSS bug):** Discovered during
   validation: `/_next/static/chunks/*.css` was being routed to the Go SPA
   catch-all, returning text/html instead of text/css. Added
   `ProxyPass /_next/ http://127.0.0.1:3003/_next/`. CSS rule count went
   from 2 (broken) to 392 (full Fumadocs).

4. **D-01 (Lang switcher order):** Reordered `INTERFACE_LANGUAGE_OPTIONS`
   in `web/default/src/i18n/languages.ts` to alphabetical by code:
   en, fr, ja, pt, ru, vi, zh. typecheck + build pass clean.

## Key Decisions

| ID  | Decision | Choice | Reason |
|-----|----------|--------|--------|
| D-01 | Lang switcher order | Alphabetical by code | UX standard, user request |
| D-02 | Next.js docs routing | Expose via Apache proxy /pt/ | All 4 locales consistent |
| D-03 | Atius logo 404 | Alias /assets/atius-logo.{svg,png} | Files already on disk, just missing Apache rule |
| D-03b | CSS chunks 404 | ProxyPass /_next/ → 3003 | Without it, every CSS file returns SPA HTML |
| D-04 | CDN cache invalidation | Documented as external (CF) | User must purge CF; origin is now correct |
| D-05 | Visual validation | MANDATORY gate (mmx vision) | Phase 04 revalidation: green |
| D-06 | Browser validation tool | chrome-devtools raw WS (CDP) | 2.4MB JS bundle needs raw WS, MCP times out |

## Files Created/Modified

- `web/default/src/i18n/languages.ts` — alphabetical lang order
- `.planning/phases/04-prod-docs-bugfixes/04-SUMMARY.md` — this file
- `.planning/scripts/validate-pt-docs.py` — chromium/CDP validation script
- `.planning/phase-04-screenshots/pt-docs-after-fix.png` — broken state
- `.planning/phase-04-screenshots/pt-docs-after-_next-fix.png` — green state
- `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf` — infra patch
  (3 surgical edits: /pt/ proxy, /assets/ aliases, /_next/ proxy)
  (backup: `router.atius.com.br-le-ssl.conf.bak-pre-phase04-20260606-021042`)

## Verification

- [x] All 4 locales (/en/ /pt/ /ja/ /zh/) return `x-powered-by: Next.js`
- [x] `/_next/static/chunks/*.css` returns `text/css` (was `text/html` from Go SPA)
- [x] `/assets/atius-logo.svg` returns 200 OK (15086 bytes, image/svg+xml)
- [x] CSS rule count: 2 → 392 (full Fumadocs layout)
- [x] `<html lang>` attribute correct per locale
- [x] `<title>` PT-BR: "Início Rápido | Atius AI Router"
- [x] Sidebar + header + main + TOC (3-col Fumadocs layout) present
- [x] Visual: mmx vision confirms "completamente estilizada, layout Fumadocs"
- [x] D-01 typecheck + build green (`bun run typecheck && bun run build`)
- [x] Apache configtest: "Syntax OK"
- [x] No regression: en/ja/zh still work
- [ ] **DEFERRED:** Cloudflare cache purge for old /pt/docs/ 404 and /assets/atius-logo.svg 404
      entries. Origin is correct; new requests will succeed. User action: manual
      Cloudflare purge or wait for TTL (~24h). Documented in 21.03-Decisoes-Arquitetura/.

## What I would do differently

- The CONTEXT.md (gathered 2026-06-05) only identified 3 bugs but missed the
  `/_next/` proxy gap — the CSS bug was visible but the root cause was not
  stated. Discovered during re-validation, not in research. Future phases:
  always include network-level request inspection in the initial visual audit.
- D-04 (CF cache) should be a Phase 05 task or a one-liner in the deploy
  checklist, not a "user can deal with it" footnote. Filed for M099 if
  recurring.
