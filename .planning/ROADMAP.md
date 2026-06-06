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

### Phase 04: Prod Docs Bugfixes ✅ (2026-06-06)

3 post-deploy fixes after user reported "site is broken, never tested properly":

| Bug | Root Cause | Fix |
|---|---|---|
| `/pt/docs/` served as Go SPA 404 | Apache missing `ProxyPass /pt/` (en/zh/ja had it, pt didn't) | Add `/pt/` ProxyPass → 3003 |
| `/assets/atius-logo.svg` 404 | No Apache alias for Fumadocs header path | Add `Alias /assets/atius-logo.{svg,png}` |
| CSS unstyled (Times New Roman) | `/_next/static/chunks/*.css` fell to Go catch-all (text/html) | Add `ProxyPass /_next/` → 3003 |
| Lang switcher not alphabetical | zh/en/fr/ru/ja/vi/pt order | Alphabetical: en, fr, ja, pt, ru, vi, zh |

**Files:** 1 commit (`f78631367`) — `web/default/src/i18n/languages.ts` + planning + screenshots + validation script.
**Infra (out of repo):** Apache vhost patch (3 surgical edits, configtest OK).
**Deferred to user:** Cloudflare cache purge for stale 404 entries (origin is now correct, new requests succeed).

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

## v2.13 — Post-i18n Hardening 🔵 Active

Goal: Close the 4 v2.12 deferred items and add infrastructure to prevent
regression. Native-only, fork-sync-safe, no scope creep.

### Phase 05: Cloudflare Cache Purge Automation
- CLI script `purge-cf.sh` accepting paths as args
- Token in vault, not in repo
- Optional post-deploy hook in deploy script
- Idempotent (no-op if no stale entries)

### Phase 06: Apache Proxy + Visual Validation Toolkit
- `apx-smoke.sh` — curl-based check for 4 locales + `/_next/` + `/assets/atius-logo.*`
- `validate-spa.py` — chromium + CDP raw WS + mmx vision (reusable, not phase-specific)
- Both live in `.planning/scripts/`, run manually pre/post deploy

### Phase 07: Classic Frontend PT Support
- Register `pt` in `web/classic/src/i18n/{i18n,language}.js`
- Translate `pt.json` keys (mirror default frontend pattern)
- Code-only — classic not in production, completes pt coverage matrix
- Validation: `bun run typecheck && bun run build` in `web/classic/`

### Phase 08: v2.13 Verification + Roadmap Audit
- Run apx-smoke + validate-spa on full prod
- Visual confirm 4/4 locales still styled, logos loaded, lang order ok
- Audit deferred items closed, file new M099 if any residual
- Update STATE.md + ROADMAP.md → ready for v2.13 complete-milestone

---

## Next

- [ ] Push `feat/pt-native` for router-ai-atius (pending approval)
- [ ] Push upstream for new-api-docs-v1 PT changes (pending)
- [ ] Cloudflare cache purge (manual, or wait TTL ~24h)
- [ ] Run Phase 05: CF purge CLI
- [ ] Run Phase 06: APX + VIS toolkit
- [ ] Run Phase 07: Classic PT
- [ ] Run Phase 08: Verification + audit
