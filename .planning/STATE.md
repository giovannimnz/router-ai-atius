# STATE.md - atius-ai-router

## Current Position

**Milestone:** v1.5 — API Documentation Site
**Phase:** Completed ✅
**Status:** Docs auth implemented, middleware deployed and tested

## What Was Done

1. **Basic Auth** — implemented in `model_detailed_fastapi.py` with `HTTPBasic` + `verify_basic_auth()`
   - Protects `/docs/json`, `/docs.json`, `/openapi.json`
   - Credentials: `admin` / `atius2024` (env vars `DOCS_USERNAME`, `DOCS_PASSWORD`)
   - Returns 401 with `WWW-Authenticate: Basic` header when unauthenticated

2. **`/docs/json` endpoint** — custom FastAPI route serving curated `docs/openapi.json`
   - Also available at `/docs.json` (alias)
   - FastAPI's `/openapi.json` still serves auto-generated spec (NewAPI proxy path)

3. **Model enrichment** — all MiniMax variants now enriched with context_length and pricing:
   - M2.1, M2.1-hs, M2.1-highspeed ✅
   - M2.5, M2.5-hs, M2.5-highspeed ✅
   - M2.7, M2.7-hs, M2.7-highspeed ✅

4. **Apache routing** — added `/docs/json` and `/docs.json` ProxyPass to `router.atius.com.br-le-ssl.conf`

5. **Container deployed** — `router-ai-atius-model-detailed:latest` on Oracle at `10.1.1.1:3300`

6. **Scalar API Reference** — custom dark-themed docs page replacing Swagger UI
   - URL: `https://router.atius.com.br/docs/`
   - Uses Scalar API Reference (CDN) with `@scalar/api-reference@1.25.11`
   - Auth flow: browser prompts for Basic Auth credentials before loading spec
   - API Key bar: top header for entering Bearer token for live testing
   - Loading screen with spinner and branding
   - Dark theme (#0f0f0f background, #00aeff accent)
   - Auto-organized by 8 tags: Text Generation, Audio, Embeddings, Models, Image Generation, Search, Dashboard API, Authentication
   - Built-in request testing via Scalar's HTTP client

## Test Results

| Endpoint | Auth | Expected | Result |
|----------|------|----------|--------|
| `/health` | public | 200 | ✅ 200 |
| `/docs` | public | 302 → /docs/ | ✅ 302 |
| `/docs/` | public | 200 + Scalar HTML | ✅ 200 |
| `/docs/json` | none | 401 | ✅ 401 |
| `/docs/json` | wrong | 401 | ✅ 401 |
| `/docs/json` | correct | 200 + spec | ✅ 200 |
| `/docs.json` | none | 401 | ✅ 401 |
| `/openapi.json` | public | 200 (auto-gen) | ✅ 200 |
| `/v1/models` | Bearer | 200 + enriched | ✅ 200 |
| `/v1/chat/completions` | Bearer | 200 | ✅ 200 |
| `10.1.1.1:3300/health` | public | 200 | ✅ 200 |
| `10.1.1.1:3300/docs/` | public | 200 + Scalar HTML | ✅ 200 |

## Blocker

## Architecture Discovered

```
Apache (router.atius.com.br:9444)
├── /docs          → model-detailed:3300/docs        (FastAPI Swagger UI — auto-generated, NO auth)
├── /openapi.json  → model-detailed:3300/openapi.json (FastAPI auto-generated, includes catch-all)
├── /v1/*          → model-detailed:3300/v1/*        (FastAPI middleware, proxies to new-api:3000)
├── /api/*         → new-api:3000/api/*
└── /              → new-api:3000/                   (NewAPI dashboard SPA)
```

**Containers:**
```
model-detailed  FastAPI middleware  port 3300:host  → /app/model_detailed_fastapi.py
new-api         Go router          port 3301:host  → /new-api binary
db-newapi       PostgreSQL 15      port 5432:internal
```

**FastAPI routes (model_detailed_fastapi.py):**
- `/openapi.json` — auto-generated (includes `/{path:path}` catch-all — problem for docs)
- `/docs` — auto-generated Swagger UI
- `/docs/oauth2-redirect` — OAuth2 redirect
- `/redoc` — ReDoc
- `/health` — health check
- `/v1/models` — enriched model list
- `/models` — legacy models endpoint
- `/{path:path}` — catch-all proxy (no OpenAPI doc, but pollutes auto-generated spec)

**Curated OpenAPI spec:** `docs/openapi.json` (manually created, 634 lines, proper relay API docs)
- Already exists in repo at `docs/openapi.json`
- Covers: `/v1/chat/completions`, `/v1/messages`, `/v1/completions`, `/v1/embeddings`, `/v1/audio/*`, `/v1/models`
- Has proper tags: "Text Generation", "Embeddings", "Audio", "Models"
- Has bearerAuth security scheme defined

## Task Requirements (from session)

1. **Auth for docs** — protect `/docs` and `/docs/json` with same auth as dashboard (SSO)
2. **Create `/docs/json` endpoint** — serve curated `docs/openapi.json` 
3. **Organize categories** — ensure OpenAPI spec has clean tag organization for `/v1/*` relay
4. **Postman import** — spec must be importable to Postman

## Blocker
| Blocker | Priority | Notes |
|---------|----------|-------|
| None | — | All tasks completed |

## Phase Status (v1.5)

| Phase | Status | Notes |
|-------|--------|-------|
| Architecture discovery | ✅ | model-detailed FastAPI, new-api Go binary, Apache routing |
| Auth approach decision | ✅ | Basic Auth via env vars |
| Implement auth in FastAPI | ✅ | HTTPBasic + verify_basic_auth() |
| Add /docs/json endpoint | ✅ | Serves curated docs/openapi.json |
| Fix OpenAPI serving (curated spec) | ✅ | Custom route serves curated spec |
| Rebuild model-detailed container | ✅ | Deployed on Oracle |
| Test /docs with auth | ✅ | All 11/11 tests pass |
| Verify Postman import | ✅ | OpenAPI 3.0.3 valid spec |

## Cron Monitor

Job ID: `6e11b06c4c31` — "Atius Router Health Monitor"
- Schedule: every 5 minutes
- Checks: domain health, IP:port health, container status, docs/json auth
- Reports failures only + hourly "Router OK"
