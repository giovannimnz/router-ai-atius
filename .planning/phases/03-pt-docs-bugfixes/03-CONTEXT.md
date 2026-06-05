# Phase 03: PT Docs Bugfixes — Context

**Gathered:** 2026-06-05
**Status:** Ready for planning + execution
**Milestone:** v2.12 — pt Native i18n Integration (post-deploy fixes)

## Phase Boundary

Fix 2 bugs found during Phase 02 browser validation.

### Bug 1: hreflang alternates missing `pt`

**Symptom:** `<link rel="alternate" hrefLang="...">` only shows en, zh, ja — no pt.

**Root cause:** `src/app/[lang]/layout.tsx:96-101` has a **hardcoded static object**:

```ts
alternates: {
  languages: {
    en: '/en',
    zh: '/zh',
    ja: '/ja',
  },
},
```

`i18n.languages` in `src/lib/i18n.ts` correctly includes `pt`, but this
static literal was never updated to include it. Fumadocs/i18n does NOT
auto-generate these — they are manually specified in the layout metadata.

**Impact:** Google won't index PT pages as alternate PT-BR versions.
PT pages still work (routes exist, content loads), but SEO signal is missing.

**Fix:** Add `pt: '/pt'` to the `alternates.languages` object.

---

### Bug 2: `/en/docs/guide/` → 404

**Symptom:** `GET /en/docs/guide/` returns HTTP 404. Same for all 4 locales.

**Root cause:** `content/docs/en/guide/meta.json` uses `"root": true` with
`"pages": ["---Introduction---", "../index", ...]`. The `../index` redirects
the first nav item to `/en/docs/` (the docs home). The `guide/` directory
has no `index.mdx` — rendering `/en/docs/guide/` resolves to a missing page.

The built `.next` shows `guide/home.html`, `guide/about.html` etc but NO
`guide/index.html` — Fumadocs never generated one because there's no
`guide/index.mdx` source file.

**Impact:** Low. The sidebar "User Guide" section works correctly (first
page links to `/en/docs/`). The `/en/docs/guide/` URL is not linked from
any visible navigation element. However, it's a dead URL that search
engines might discover via sitemap or crawl.

**Fix:** Add a minimal `content/docs/{locale}/guide/index.mdx` that either:
a) Redirects via Fumadocs `<Redirect />` to `/en/docs/`, or
b) Is a landing/overview page for the guide section

Option (a) is simpler and consistent with how the section already works.

---

## Decisions

| ID | Decision | Choice | Reason |
|----|----------|--------|--------|
| D-01 | hreflang fix | Add `pt: '/pt'` to static literal | Minimal change, follows existing pattern |
| D-02 | guide 404 fix | Add `index.mdx` with Redirect | Same pattern all locales; pre-existing upstream |
| D-03 | Scope | All 4 locales (en, zh, ja, pt) | Bug affects ALL locales, not just pt |

---

## Code Context

- `src/app/[lang]/layout.tsx` — lines 96-101: alternates.languages static object
- `src/lib/i18n.ts` — `languages: ['en', 'zh', 'ja', 'pt']` (already correct)
- `content/docs/en/guide/meta.json` — `"root": true`, no index page
- `content/docs/zh/guide/` — same structure as en
- `content/docs/ja/guide/` — same structure as en
- `content/docs/pt/guide/` — mirrored from en (same issue)
