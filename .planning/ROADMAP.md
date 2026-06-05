# Atius AI Router — Roadmap

## v2.12 — pt Native i18n Integration ✅ Complete

Goal: Integrate Portuguese locale into the entire stack — backend Go (new-api), frontend React/i18next, AND Fumadocs docs site. Zero custom code, only registration points.

### Phase 01: pt Locale Registration ✅ (2026-06-05)

Register `pt` locale in new-api's native i18n (Go + React/i18next). 5 native registration points.

| File | Result |
|---|---|
| `i18n/i18n.go` | LangPt + pt.yaml loading + localizer + normalizeLang + SupportedLanguages |
| `i18n/locales/pt.yaml` | 228 keys backend PT-BR |
| `web/default/src/i18n/config.ts` | import pt + resources + supportedLngs |
| `web/default/src/i18n/languages.ts` | opção "Português" |
| `web/default/src/i18n/locales/pt.json` | 4521 keys frontend PT-BR |

**Branch:** `feat/pt-native` — 3 commits (planning + implementation + tracking)

### Phase 02: pt Fumadocs i18n ✅ (2026-06-05)

Add `pt` to the Fumadocs docs site (upstream QuantumNous/new-api-docs-v1, will propagate to fork via fork-sync).

| File | Change |
|---|---|
| `src/lib/i18n.ts` | Add 'pt' to `languages: ['en', 'zh', 'ja', 'pt']` |
| `next.config.mjs` | Extend `:lang(en\|zh\|ja)` regex → `(en\|zh\|ja\|pt)` |
| `scripts/translate-docs.ts` | Add `pt` to LANGUAGES |
| `scripts/translate-en-to-pt.py` | **NEW** Python wrapper for en→pt batch translation using mmx CLI |
| `content/docs/pt/` | 294 files PT-BR (203 API ref + 80 NL docs + 11 seed) |
| Docker | Image `localhost/router-ai-atius-docs:local` rebuilt + container restarted |

**Result:** `/pt/docs/`, `/pt/docs/skills/` → 200 OK, PT-BR content live in production.

### Phase 03: PT Docs Bugfixes ✅ (2026-06-05)

Fix 2 bugs found during Phase 02 browser validation.

| Bug | Root Cause | Fix |
|---|---|---|
| hreflang missing pt | `layout.tsx` alternates.languages static literal | Add `pt: '/pt'` |
| /{lang}/docs/guide/ 404 | No `guide/index.mdx` in any locale | 4 landing pages (en/zh/ja/pt) |

**Files:** 1 edit (`layout.tsx`) + 4 new (`guide/index.mdx`) — 1 commit.

---

## Architecture Note

The router-ai-atius stack has **3 i18n systems** — all now support `pt`:

| App | Framework | i18n mechanism | PT Status |
|---|---|---|---|
| Backend (new-api) | Go | `go-i18n` with YAML | ✅ 228 keys |
| Frontend (new-api SPA) | React | i18next + language detector | ✅ 4521 keys |
| Docs (Fumadocs) | Next.js | URL prefix + MDX per locale | ✅ 294 files |

All follow native pattern — only registration points, zero custom code.

---

## Next

- [ ] Push `feat/pt-native` for router-ai-atius (pending approval)
- [ ] Push upstream for new-api-docs-v1 PT changes (pending)
- [ ] Monitor Cloudflare cache for PT docs full propagation
- [ ] Classic frontend pt support (optional — not active in prod)
