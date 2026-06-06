# PROJECT.md — Atius AI Router

**Project:** Atius AI Router (router-ai-atius fork of QuantumNous/new-api)
**Repository:** giovannimnz/router-ai-atius
**Owner:** giovannimnz (munizgiovanni@hotmail.com)
**Production:** https://router.atius.com.br
**Last updated:** 2026-06-06

## Core Value

A unified AI API gateway (40+ upstream providers) that looks, feels, and operates
in Brazilian Portuguese where it matters, while staying 100% in sync with the
upstream QuantumNous/new-api codebase via fork-sync.

## What This Is

- An Atius-branded fork of the new-api AI gateway
- Go backend (port 3030) + React/i18next SPA + Next.js/Fumadocs docs site
- Apache reverse proxy in front of all 3 services
- Cloudflare CDN in front of Apache
- 7-language UI: en, fr, ja, pt, ru, vi, zh (pt added v2.12)
- 4-language docs: en, ja, pt, zh (pt added v2.12)
- Classic (legacy Semi Design) frontend — code-only PT support (not in prod)

## What This Is NOT

- Not a SaaS — single-tenant, self-hosted
- Not a multi-tenant provider — single owner, multiple internal users
- Not a UI framework — uses QuantumNous React/i18next unmodified
- Not a translation service — translations are static, generated offline
- Not a CI/CD system — deploys are manual with `podman` + `systemctl`

## Architecture (current)

```
[Cloudflare CDN]
        |
[Apache vhost 443 — router.atius.com.br-le-ssl.conf]
   |
   |--> /v1/, /health, /docs/json      → model-detailed (3300)
   |--> /docs/, /{en,pt,ja,zh}/, /_next/ → Next.js docs (3003)
   |--> /api/, /login, /logoff, /      → new-api Go SPA (3030)
   |--> /assets/atius-logo.{svg,png}   → /var/www/atius/
   |
[Docker / Podman containers, PM2-managed]
```

## Validated Capabilities (post-v2.12)

- 4 locales consistent routing on Apache (D-02 fixed 2026-06-06)
- 2 logos accessible (D-03 fixed 2026-06-06)
- CSS chunks reach Next.js (D-03b fixed 2026-06-06)
- Lang switcher alphabetical (D-01 fixed 2026-06-06)
- mmx vision validates "completamente estilizada" Fumadocs layout
- 392 CSS rules, 3-col layout, light theme, sidebar + header + TOC
- Bun-managed React 19 + Rsbuild frontend
- fork-sync CLI for protected Globs during upstream sync

## Tech Stack

- **Backend:** Go 1.22+, Gin, GORM v2
- **Frontend:** React 19, TypeScript, Rsbuild, Base UI, Tailwind, i18next
- **Docs:** Next.js 15+, Fumadocs, MDX per locale
- **DB:** SQLite/MySQL/PostgreSQL (all 3 supported, currently SQLite prod)
- **Cache:** Redis + in-memory
- **Infra:** Apache 2.4, Cloudflare, Podman, PM2
- **i18n:** go-i18n (backend) + i18next (frontend) + Fumadocs (docs)

## Current Milestone: v2.13 — Post-i18n Hardening

**Goal:** Close the v2.12 deferred items and add the missing infrastructure
to prevent the same class of bugs (Next.js asset proxy gaps, Cloudflare
stale cache, visual regressions) from recurring.

**Premissas (constraints that MUST hold):**

1. **Native infrastructure only.** Every fix must follow the existing native
   pattern. Zero custom code. If the upstream QuantumNous/new-api has a way
   to do it, use that way. (Same principle as v2.12.)
2. **fork-sync safe.** All changes must survive `fork-sync` from upstream.
   No edits to upstream-only paths. Use `protected_globs` if needed.
3. **No scope creep.** v2.13 is exclusively about closing the 4 deferred
   items from v2.12. New features, new locales, new design = v2.14+.

**Target features (4 total, derived strictly from v2.12 SUMMARY deferred list):**

- **CF-01 — Cloudflare cache purge automation:** CLI script + deploy hook
  to purge `/pt/docs/*` and `/assets/atius-logo.svg` stale entries. Token
  stored in vault, not in repo. Triggered manually + on deploy.
- **VIS-01 — Visual validation gate:** Reusable `validate-spa.py` script
  (chromium + CDP raw WS + mmx vision) that can be invoked post-deploy
  to catch unstyled layouts, broken images, and 404 SPA bodies. Lives in
  `.planning/scripts/`. User runs it manually after each deploy.
- **APX-01 — Apache proxy smoke test:** curl-based check that verifies
  all 4 locales return `x-powered-by: Next.js`, `/_next/static/chunks/*.css`
  returns `text/css`, `/assets/atius-logo.{svg,png}` returns 200. Catches
  the "added a new locale but forgot the proxy" class of bug. Lives in
  `.planning/scripts/`, run before each deploy.
- **CLS-01 — Classic frontend PT support:** Register `pt` in
  `web/classic/src/i18n/i18n.js` + `web/classic/src/i18n/language.js`,
  translate `pt.json` keys (mirror default frontend pattern). Code-only
  — classic frontend not deployed, but completes the pt coverage matrix.

**Out of scope (explicitly excluded for v2.13):**

- Multi-tenant / SaaS mode
- CI/CD pipeline (no GitHub Actions, no deploy automation beyond the script hooks)
- Translation service integration (DeepL, etc.) — offline-generated only
- Classic frontend deployment (code-only support, no Apache routing)
- Mobile apps
- Visual validation as blocking CI gate (advisory only in v2.13)

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via /gsd-transition):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via /gsd-complete-milestone):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

## Key Decisions (chronological)

| Date | Decision | Reason |
|---|---|---|
| 2026-02-17 | Fork giovannimnz/router-ai-atius 100% Atius (Zentrius assets moved to ~/Imagens/zentrius.*) | Brand separation |
| 2026-06-04 | Bun > npm for Node/Next.js projects (per user) | Speed |
| 2026-06-05 | pt locale = native infrastructure only (5 reg points) | No custom code, sync-friendly |
| 2026-06-05 | Fumadocs MDX per locale (URL prefix /pt/) | Native Fumadocs pattern |
| 2026-06-06 | Apache ProxyPass /_next/ for Next.js assets | CSS chunks 404 without it |
| 2026-06-06 | chromium + CDP raw WS for SPA validation | MCP chrome-devtools times out on 2.4MB JS |
| 2026-06-06 | v2.13 = ONLY 4 v2.12-deferred items, no scope creep | User: "manter 100% dentro das premissas" |

## Out of Scope

- Multi-tenant / SaaS — single-tenant only
- Payment integration — no Stripe/etc.
- Mobile native apps — web only
- Translation service integration (DeepL, Google) — offline-generated only
- Classic frontend production deployment — code-only support, no deploy
- CI/CD automation beyond deploy hooks (no GitHub Actions, no blocking gates)

## Last updated

2026-06-06 — Milestone v2.13 started (4 v2.12-deferred items)
