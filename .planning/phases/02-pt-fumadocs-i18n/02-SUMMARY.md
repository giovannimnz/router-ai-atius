# Phase 02: pt Fumadocs i18n — SUMMARY.md

**Status:** Complete
**Date:** 2026-06-05
**Phase:** 02
**Branch:** feat/pt-fumadocs (via refs/new-api-docs-v1)

## What was built

Added `pt` locale to the Fumadocs documentation site (Next.js 16 + Fumadocs Core 16). All 294 MDX/MD files translated to PT-BR, 203 API ref files preserved in EN. Site rebranded to "Atius AI Router".

## Key Decisions

| Decision | Choice | Reason |
|---|---|---|
| D-01 Locale code | `pt` | Follows en/zh/ja pattern (2-letter) |
| D-02 Scope | All 313 EN files | Mirror consistency |
| D-03 Method | Python + mmx CLI | Keys already configured |
| D-04 Work location | refs/new-api-docs-v1 | Deps installed, git ready |
| D-05 API ref | EN (copied, not translated) | Auto-generated, would break |
| D-06 Branding | atius-router-docs-rebrand.sh | 8 changes applied |

## Files Created/Modified

| File | Change |
|---|---|
| `src/lib/i18n.ts` | Add 'pt' to languages |
| `next.config.mjs` | Regex extendido, trailingSlash |
| `scripts/translate-docs.ts` | Add pt to LANGUAGES |
| `scripts/translate-en-to-pt.py` | NEW — batch translate wrapper |
| `content/docs/pt/` | 294 files PT-BR |
| Various branding files | 8 rebrand changes |
| `Dockerfile` | Created from template |
| `.dockerignore` | NEW |

## Build Fixes Applied
- 37 frontmatter fixes
- 3 image path fixes
- 1 import fix
- 4 frontmatter title translations

## Verification
- [x] `bun run build` — 2655 pages, exit 0
- [x] Docker image rebuilt
- [x] Container running
- [x] All PT titles verified
- [x] Atius AI Router branding confirmed
- [x] Logo, GitHub links, metadata correct
- [ ] hreflang missing pt (non-blocking)
