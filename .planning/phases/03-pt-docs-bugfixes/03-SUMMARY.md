---
phase: 3
status: complete
completed: 2026-06-05
---

# Phase 03: PT Docs Bugfixes — SUMMARY.md

## What was built

Fixed 2 bugs found during Phase 02 browser validation.

### Bug 1: hreflang alternates missing pt

**Root cause:** `src/app/[lang]/layout.tsx:96-101` had a static alternates.languages object
hardcoded with only en, zh, ja — pt was in `i18n.languages` but not in this literal.

**Fix:** Added `pt: '/pt'` to the alternates map.

### Bug 2: /en/docs/guide/ 404

**Root cause:** The `content/docs/{locale}/guide/` directory had no `index.mdx`, so
resolving the route produced 404. Pre-existing upstream issue across all 4 locales.

**Fix:** Created 4 landing pages:
- `en/guide/index.mdx` — "User Guide" with sidebar links
- `zh/guide/index.mdx` — "用户指南"
- `ja/guide/index.mdx` — "ユーザーガイド"
- `pt/guide/index.mdx` — "Guia do Usuário"

## Files Changed

| File | Change |
|---|---|
| `src/app/[lang]/layout.tsx` | Add `pt: '/pt'` to alternates.languages |
| `content/docs/en/guide/index.mdx` | NEW — landing page |
| `content/docs/zh/guide/index.mdx` | NEW — landing page |
| `content/docs/ja/guide/index.mdx` | NEW — landing page |
| `content/docs/pt/guide/index.mdx` | NEW — landing page |

## Verification

- [x] hreflang shows `pt` in `<link rel="alternate">`
- [x] /en/docs/guide/ → 200
- [x] /zh/docs/guide/ → 200
- [x] /ja/docs/guide/ → 200
- [x] /pt/docs/guide/ → 200
- [x] bun run build → 2661 pages, exit 0
- [x] Docker image rebuilt
- [x] Container deployed
