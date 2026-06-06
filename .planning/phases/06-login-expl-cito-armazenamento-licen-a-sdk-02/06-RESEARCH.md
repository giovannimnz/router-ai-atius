---
phase: 06
phase_name: login-expl-cito-armazenamento-licen-a-sdk-02
status: complete
created: 2026-06-06
method: inline-orchestrator
---

# Phase 06 — Research

Inline research performed after `/gsd-ui-phase 06` because `gsd-phase-researcher`, `gsd-planner`, and `gsd-plan-checker` resolved to empty agent skill payloads in Hermes CLI.

## Inputs Read

- `.planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-CONTEXT.md`
- `.planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-UI-SPEC.md`
- `.planning/ROADMAP.md`
- `.planning/STATE.md`
- `controller/codex_oauth.go`
- `controller/codex_usage.go`
- `service/codex_credential_refresh.go`
- `service/codex_credential_refresh_task.go`
- `relay/channel/codex/adaptor.go`
- `relay/channel/codex/oauth_key.go`
- `integration/codex-sidecar/main.py`
- `router/api-router.go`
- `web/default/src/features/channels/api.ts`
- `web/default/src/features/channels/components/dialogs/codex-oauth-dialog.tsx`
- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx`
- `web/default/src/features/channels/types.ts`
- `web/default/src/components/layout/lib/sidebar-view-registry.ts`
- `web/default/src/styles/theme.css`
- `web/default/src/styles/index.css`
- `web/default/package.json`

## Existing Backend Patterns

### Codex OAuth

`controller/codex_oauth.go` already supports:

- `POST /api/channel/codex/oauth/start` — global flow, returns `authorize_url`.
- `POST /api/channel/codex/oauth/complete` — global flow, returns generated credential JSON as `data.key`.
- `POST /api/channel/:id/codex/oauth/start` — channel-scoped flow.
- `POST /api/channel/:id/codex/oauth/complete` — channel-scoped flow, writes JSON to `channel.key`.

The existing completion path already builds a JSON credential with:

- `access_token`
- `refresh_token`
- `account_id`
- `last_refresh`
- `expired`
- `email`
- `type: codex`

All JSON calls use `common.Marshal` / `common.Unmarshal`, matching AGENTS.md.

### Refresh

`service/codex_credential_refresh.go` parses `channel.key`, refreshes via `RefreshCodexOAuthTokenWithProxy`, writes the refreshed JSON back to `model.Channel.key`, and resets caches when requested.

`service/codex_credential_refresh_task.go` already auto-refreshes Codex channel credentials every 10 minutes when expiry is within 24h. Phase 06 should extend this so the selected primary channel mirror is also updated after refresh when applicable.

### Sidecar License

`integration/codex-sidecar/main.py` currently has an SDK-02 placeholder:

- tries repo/container license path;
- falls back to `~/.codex/auth.json`;
- initializes SDK once at startup.

Phase 06 must replace this with the locked decision:

- no host fallback;
- only `data/codex/license.json` from the app mirror;
- reload by `mtime`;
- `backend=sdk` hard-fails if license is missing/invalid.

Container path note: `podman-compose.yml` from Phase 05 maps `./data/codex:/app/data`, so inside the sidecar the license should be `/app/data/license.json`. The app-level canonical path remains `data/codex/license.json`.

### Routing

`router/api-router.go` has Codex channel routes under `channelRoute := apiRouter.Group("/channel")` with `middleware.AdminAuth()`.

Good fit for page-level auth routes:

- `GET /api/channel/codex/auth/status`
- `POST /api/channel/codex/auth/import`
- `POST /api/channel/codex/auth/primary`
- `POST /api/channel/codex/auth/refresh`
- `GET /api/channel/codex/auth/export`

Keep route registration before generic/ambiguous channel operations where possible, but `/:id` is an exact one-segment route and does not consume `/codex/auth/status`.

### Primary Channel Persistence

`model/option.go` supports generic persisted options through `model.UpdateOption`, `model.UpdateOptionsBulk`, `common.OptionMap`, and `InitOptionMap` defaults.

Recommended new option key:

- `CodexSDKPrimaryChannelID`

Default value should be `""` in `InitOptionMap()`. This avoids a new table and remains cross-DB compatible.

## Existing Frontend Patterns

### UI Stack

Frontend uses:

- React 19 + TypeScript.
- Bun scripts.
- TanStack Router file routes.
- React Query + axios `api` instance.
- Base UI/local shadcn-style primitives.
- Tailwind CSS tokens from `src/styles/theme.css`.
- `useTranslation()` for visible UI copy.

### Routes

Authenticated routes live under `web/default/src/routes/_authenticated/` and use `createFileRoute('/_authenticated/...')`.

To expose `/admin/codex-auth`, create:

- `web/default/src/routes/_authenticated/admin/codex-auth.tsx`
- route path string `createFileRoute('/_authenticated/admin/codex-auth')`

Use the same admin guard pattern as `routes/_authenticated/channels/index.tsx`.

### Layout

Channel page uses `SectionPageLayout`. Phase 06 should use the same shell:

- `SectionPageLayout.Title`
- `SectionPageLayout.Actions`
- `SectionPageLayout.Content`

### Existing Codex UI

`codex-oauth-dialog.tsx` already implements:

- start authorization;
- copy authorization link;
- paste callback URL;
- generate credential;
- toasts;
- disabled/loading states.

Phase 06 can reuse/adapt this logic into page sections instead of a drawer dialog.

`channel-mutate-drawer.tsx` currently has a type 57 Codex auth block around lines 1988-2047. Phase 06 should replace the embedded OAuth dialog with compact status + link to `/admin/codex-auth`.

### i18n

`web/default/AGENTS.md` requires all visible copy through `useTranslation()` / `t()`. New constants or static labels must be discoverable by the i18n sync script. Validation command should include:

- `bun run i18n:sync`
- `bun run typecheck`

## Risks

1. Sidecar mtime reload may require reinitializing the SDK handle safely. If `openai-codex` does not support runtime credential reload, implement an explicit teardown/recreate path guarded by a lock.
2. Writing `data/codex/license.json` must be atomic to avoid sidecar reading partial JSON.
3. Full credential export is sensitive. It must require admin auth and explicit user click, and tokens must not be printed in normal page content.
4. Channel `backend=sdk` hard-fail means Phase 05 fallback behavior must be removed intentionally.
5. The existing auto-refresh task refreshes channel keys but not the mirror; primary mirror sync must happen after refresh for the selected primary channel.

## Recommended Implementation Shape

- Add backend service layer first (`service/codex_license.go`) so controller and refresh task share parsing, status, mirror write, export, and primary-channel logic.
- Add controller routes next (`controller/codex_auth.go`) and wire in `router/api-router.go`.
- Update sidecar license loading/mirror reload after backend mirror writer exists.
- Build frontend feature after API contract is stable.
- Update drawer last, keeping old channel key editing intact.

## Verification Commands

Backend:

```bash
go test ./service ./relay/channel/codex ./controller
go build ./...
python3 -m py_compile integration/codex-sidecar/main.py
```

Frontend:

```bash
cd web/default
bun run typecheck
bun run i18n:sync
bun run build
```

Runtime/smoke when stack is available:

```bash
curl -sS http://localhost:3000/api/channel/codex/auth/status
podman compose restart codex-sidecar
podman compose logs codex-sidecar --tail=80
```

Visual validation:

- Open `/admin/codex-auth`.
- Compare page against `06-UI-SPEC.md` with browser vision.
- Open channel drawer for Codex type 57 and confirm only compact status + link remains.
