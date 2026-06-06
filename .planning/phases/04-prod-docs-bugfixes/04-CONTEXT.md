# Phase 04: Prod Docs Bugfixes — Context

**Gathered:** 2026-06-05
**Status:** Research complete, planning
**Milestone:** v2.12 — pt Native i18n (post-deploy fixes round 2)

## Visual Validation Results (mmx vision, chrome-devtools)

User feedback: site is broken, never actually opened or tested properly. Full
visual audit via Playwright/Chromium + mmx vision API confirms it.

### EN `/en/docs/` — NOTA 1/10

**Catastrophic CSS failure:**
- Logo SVG breaks: 2/2 images broken (src=`/assets/atius-logo.svg` → 404)
- CSS rules loaded: 2 (expected 1000+)
- CSS variables missing: `--fd-nav-height`, `--fd-sidebar-width`, `--fd-layout-offset`
- HTML lang: empty string
- Main element: padding=0 (broken)
- HTML class: `light` (no dark mode applied)
- Sidebar: structurally present but unstyled
- GitHub icon: massive, no max-width, dominates viewport
- Nav links: "Quick StartProject Introduction" — glued together
- Visual: Times New Roman fallback, looks like 1995 HTML

### PT `/pt/docs/` — 100% BROKEN

- HTTP 200 but body is 404 SPA page
- Title: "Atius Router" (no localized title)
- HTML lang: "en" (wrong)
- No sidebar, no nav, no footer
- 0 links, 0 images
- Just: "Oops! Página não encontrada!" in PT-BR
- CDN serves New-API Go backend (x-oneapi-request-id header) instead of Next.js

## Root Cause Analysis

### Bug 1: `https://router.atius.com.br/assets/atius-logo.svg` → 404

Apache only proxies:
- `/docs/` → model-detailed (port 3399)
- `/` (catch-all) → New-API Go (port 3301)
- `/v1/`, `/health` → model-detailed
- No proxy for Next.js docs (port 3003)

The `/en/docs/` and `/pt/docs/` pages work because Cloudflare has a CACHE
of old Next.js responses. `/pt/docs/` was never cached as Next.js — so it
falls through to the catch-all (New-API Go), which returns 404.

The Next.js docs are being served by... what? The cf-ray header suggests
Cloudflare is reaching an origin. Let me check.

### Bug 2: /pt/docs/ is not Next.js

Header inspection:
- `x-powered-by: Next.js` for `/en/docs/`
- `x-new-api-version: v2.11.0-rebrand.20260602` for `/pt/docs/`
- `x-oneapi-request-id: ...` for `/pt/docs/` (proves it's the Go backend)

`/en/docs/` reaches Next.js via Cloudflare cache (HIT).
`/pt/docs/` was never cached, falls through to Apache, which routes to Go.

### Bug 3: CSS not loaded

When accessed via origin (localhost:3003), CSS works fine. The visual
broken-ness is the result of Cloudflare caching OLDER pages that pointed
to different CSS chunk filenames. After my Phase 02 build, the chunk
hashes changed but CDN still serves old HTML referencing old CSS files.

### Bug 4: Language switcher not alphabetical

Current order in SPA: `zh, en, fr, ru, ja, vi, pt` (not alphabetical)
Current order in docs: `['en', 'zh', 'ja', 'pt']` (3 first are sorted,
but pt is at end, zh after en)

Should be: `en, fr, ja, pt, ru, vi, zh` (alphabetical by code)

## Decisions

| ID | Decision | Choice | Reason |
|----|----------|--------|--------|
| D-01 | Lang switcher order | Alphabetical by code | User request, UX standard |
| D-02 | Next.js docs routing | Expose via Apache proxy | Both en/pt/ja/zh must route consistently |
| D-03 | Atius logo 404 | Move to `/var/www/atius/` (already exists) | Apache has Alias for this |
| D-04 | CDN cache invalidation | Purge everything on each deploy | Prevents stale chunk hashes |
| D-05 | Visual validation | **MANDATORY gate** with mmx vision | All future deployments must pass |
| D-06 | Browser validation tool | Always chrome-devtools (Chromium) | Per user preference, MM-M3 has no native vision |
