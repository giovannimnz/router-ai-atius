# Atius AI Router — Roadmap

## v2.12 — pt Native i18n Integration

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

### Phase 02: pt Fumadocs i18n (Ready for planning)

Add `pt` to the Atius-branded Fumadocs site (fork `giovannimnz/new-api-docs-v1`).

| File | Change |
|---|---|
| Fork: `src/lib/i18n.ts` | Add 'pt' to `languages: ['en', 'zh', 'ja', 'pt']` |
| Fork: `next.config.mjs` | Extend `:lang(en\|zh\|ja)` regex → `(en\|zh\|ja\|pt)` |
| Fork: `scripts/translate-docs.ts` | Add `pt` to LANGUAGES, source = en, update GLOSSARY |
| Fork: `content/docs/pt/` | 313 MDX files (11 already exist in fork-sync seed) |
| Fork: `.github/workflows/translate.yml` | Add pt trigger path |
| Fork-sync: `pt-content/` | Mirror the 313 translated files (already has `protected_globs: ["content/docs/pt/**"]`) |

---

## Architecture Note

The router-ai-atius stack has **3 i18n systems** that all need `pt`:

| App | Framework | i18n mechanism |
|---|---|---|
| Backend (new-api) | Go | `go-i18n` with YAML |
| Frontend (new-api SPA) | React | i18next + language detector |
| Docs (Fumadocs) | Next.js | URL prefix + MDX per locale |

Phase 01 covers backend + frontend. Phase 02 covers docs. Both follow the native pattern — only registration points, no custom code.

---

## Up Next

- [ ] Phase 02: pt Fumadocs i18n (12 tasks — see PLAN.md)
- [ ] Push `feat/pt-native` for router-ai-atius
- [ ] Push `feat/pt-fumadocs` for the fork docs repo
- [ ] Production deploy of both
