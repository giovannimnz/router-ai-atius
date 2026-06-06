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

## Current Milestone: v2.14 — Codex SDK Transformer

**Goal:** Usar assinatura Codex Pro (plano 100 USD) como módulo transformer
dentro do router-ai-atius, expondo o Codex SDK programaticamente com
visibilidade de saldo/usage — sem quebrar o canal Codex tipo 57 existente.

**Premissas (constraints that MUST hold):**

1. **Aditivo, não substitutivo.** O transformer novo coexiste com o relay
   HTTP Codex tipo 57 atual. Nenhum canal existente pode quebrar.
2. **fork-sync safe.** Mudanças no código Go e frontend devem sobreviver
   a sync de upstream. Scripts/adapters externos (Python/TS SDK) podem
   viver fora da árvore Go principal.
3. **Login explícito (estilo Hermes).** O transformer SEMPRE pede input do
   admin para autenticar: OAuth code colado no dashboard ou token manual.
   Nunca reutiliza credenciais automaticamente do filesystem do host.
   Mesmo comportamento do Hermes Agent: `hermes login` → pede código.
4. **Scope contido.** v2.14 é só Codex SDK transformer + saldo. Features
   não-Codex ou outros providers ficam pra v2.15+.

**Target features (4):**

- **SDK-01 — Transformer module:** Novo módulo que converte requests do
  router em chamadas ao Codex SDK (Python `openai-codex` como primeira
  runtime), em vez do relay HTTP puro `/backend-api/codex/responses`.
- **SDK-02 — Login explícito + armazenamento próprio:** O transformer SEMPRE
  força input do admin para autenticar: (a) colar authorization code do
  OAuth flow no campo do dashboard, ou (b) importar JSON manual com
  `{access_token, refresh_token, account_id}`. Resultado armazenado em
  `data/codex/license.json` com refresh automático depois de autenticado.
  Nunca reutiliza `~/.codex/auth.json` do host automaticamente. Mesmo
  comportamento do Hermes Agent: login explícito → armazena → usa.
- **SDK-03 — Usage/saldo endpoint:** Expor dados de pricing/usage do
  account Codex no admin dashboard e via endpoint REST. Mostrar consumo
  do plano Pro (100 USD), renovação, histórico.
- **SDK-04 — Fallback coexistence:** O transformer novo coexiste com o
  canal Codex tipo 57 atual. Admin pode escolher qual backend usar
  (SDK vs relay HTTP) via config de canal.

**Out of scope (explicitly excluded for v2.14):**

- Outros provedores SDK (Claude, Gemini, etc.) — Codex only
- TypeScript SDK runtime (v2.14 é Python-first)
- Multi-key / multi-account Codex
- Faturamento próprio do router via Codex — é só relay/transformer

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

2026-06-06 — Milestone v2.14 started (Codex SDK Transformer)
