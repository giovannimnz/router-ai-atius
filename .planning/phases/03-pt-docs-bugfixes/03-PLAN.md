---
phase: 03
wave: 1
depends_on: []
files_modified:
  - src/app/[lang]/layout.tsx
  - content/docs/en/guide/index.mdx
  - content/docs/zh/guide/index.mdx
  - content/docs/ja/guide/index.mdx
  - content/docs/pt/guide/index.mdx
autonomous: true
must_haves:
  - hreflang shows pt
  - /en/docs/guide/ → 200 (via redirect)
  - /zh/docs/guide/ → 200
  - /ja/docs/guide/ → 200
  - /pt/docs/guide/ → 200
  - bun run build passes
---

# Phase 03: PT Docs Bugfixes — PLAN.md

**Phase:** 03
**Status:** Ready for execution
**Date:** 2026-06-05

## Plan 01: hreflang pt alternates (layout.tsx)

### Context
`src/app/[lang]/layout.tsx:96-101` has `alternates.languages` as a static
object. `pt` was added to `i18n.languages` in Phase 02 but this literal
was not updated.

### Task 01: Add pt to alternates

<read_first>
- src/app/[lang]/layout.tsx (lines 96-101)
</read_first>

<acceptance_criteria>
- `alternates.languages` contains `pt: '/pt'` alongside en, zh, ja
- `bun run build` passes (typecheck + build)
- `curl http://127.0.0.1:3003/pt/docs/ | grep -c 'hrefLang="pt"'` > 0
</acceptance_criteria>

<action>
In src/app/[lang]/layout.tsx, add `pt` to the static alternates.languages map:

```ts
alternates: {
  languages: {
    en: '/en',
    zh: '/zh',
    ja: '/ja',
    pt: '/pt',
  },
},
```

No other changes needed — `i18n.languages` already includes `pt`.
</action>

---

## Plan 02: Guide index redirect (all 4 locales)

### Context
`/en/docs/guide/` returns 404 because the `guide/` directory has no
`index.mdx` — only sub-pages (home, about, document, pricing, wiki/).
The `meta.json` uses `"root": true` + `"../index"` as first page.

### Task 02: Create guide/index.mdx for all locales

<read_first>
- content/docs/en/guide/meta.json
- content/docs/en/guide/home.mdx (for title reference)
- src/lib/i18n.ts (for available locales)
</read_first>

<acceptance_criteria>
- `content/docs/en/guide/index.mdx` exists with Redirect
- `content/docs/zh/guide/index.mdx` exists with Redirect
- `content/docs/ja/guide/index.mdx` exists with Redirect
- `content/docs/pt/guide/index.mdx` exists with Redirect (title in PT)
- `bun run build` passes
- `/en/docs/guide/` → 200 (resolved via Redirect or content)
- `/pt/docs/guide/` → 200
</acceptance_criteria>

<action>
Create 4 index.mdx files, one per locale directory:

1. **EN:** `content/docs/en/guide/index.mdx`
```mdx
---
title: User Guide
description: Quick start and basic tutorials
---

import { Redirect } from 'fumadocs-ui/components/redirect';

<Redirect href="/en/docs" />
```

2. **ZH:** `content/docs/zh/guide/index.mdx`
Same pattern, href="/zh/docs"

3. **JA:** `content/docs/ja/guide/index.mdx`
Same pattern, href="/ja/docs"

4. **PT:** `content/docs/pt/guide/index.mdx`
```mdx
---
title: Guia do Usuário
description: Início rápido e tutoriais básicos
---

import { Redirect } from 'fumadocs-ui/components/redirect';

<Redirect href="/pt/docs" />
```

All 4 files redirect to their respective docs home (`/{lang}/docs`).
</action>
