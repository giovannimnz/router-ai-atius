# STATE.md - router-ai-atius

## Current Position

**Milestone:** v1.6 — Internacionalização PT-BR
**Phase:** Not started (planning)
**Status:** Defining requirements
**Last activity:** 2026-06-02 — Vendored new-api-docs-v1 reference (quick task 260601-wnh)

## What Was Done

_Milestone v1.6 started. Planning i18n PT-BR scope._

## Architecture Discovered

```
Apache (router.atius.com.br:9444)
├── /docs          → model-detailed:3300/docs
├── /openapi.json  → model-detailed:3300/openapi.json
├── /v1/*          → model-detailed:3300/v1/*
├── /api/*         → new-api:3000/api/*
└── /              → new-api:3000/ (SPA)

Containers:
model-detailed  FastAPI middleware  port 3300:host
new-api         Go router          port 3301:host
db-newapi       PostgreSQL 15      port 5432:internal
```

## Phase Status (v1.6)

| Phase | Status | Notes |
|-------|--------|-------|
| Frontend PT-BR translation | pending | 3914 keys in en.json |
| Backend i18n PT-BR | pending | Go i18n (nicksnyder/go-i18n/v2) |
| DB: set Language=pt | pending | Options table |
| Branch: feat/portuguese-translation | pending | For upstream PR |

## Blocker

| Blocker | Priority | Notes |
|---------|----------|-------|
| None | — | Ready to start |

## Milestones

| Version | Goal | Status |
|---------|------|--------|
| v1.0 | Initial Setup | ✅ |
| v1.1 | DeepSeek Enrichment | ✅ |
| v1.2 | Fork Migration | ✅ |
| v1.3 | Testing Infrastructure | ✅ |
| v1.4 | Model Aliases | ✅ |
| v1.5 | API Documentation Site | ✅ |
| v1.6 | Internacionalização PT-BR | in progress |
| v1.7 | Documentação PT-BR | pending |
| v1.8 | Podman Migration | pending |

## Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260601-wnh | Clone QuantumNous/new-api-docs-v1 into docs/new-api-docs-v1 and add it to .gitignore | 2026-06-02 | a53a57941 | [260601-wnh-clone-quantumnous-new-api-docs-v1-into-d](./quick/260601-wnh-clone-quantumnous-new-api-docs-v1-into-d/) |
