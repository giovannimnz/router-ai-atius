---
phase: 06
phase_name: login-expl-cito-armazenamento-licen-a-sdk-02
wave: 1
depends_on:
  - 05
files_modified:
  - service/codex_license.go
  - service/codex_credential_refresh.go
  - service/codex_credential_refresh_task.go
  - controller/codex_auth.go
  - router/api-router.go
  - model/option.go
  - integration/codex-sidecar/main.py
  - web/default/src/routes/_authenticated/admin/codex-auth.tsx
  - web/default/src/features/codex-auth/index.tsx
  - web/default/src/features/codex-auth/api.ts
  - web/default/src/features/codex-auth/types.ts
  - web/default/src/features/codex-auth/components/*
  - web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx
  - web/default/src/features/channels/api.ts
  - web/default/src/features/channels/types.ts
autonomous: false
requirements_addressed: [SDK-02]
context: .planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-CONTEXT.md
ui_contract: .planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-UI-SPEC.md
research: .planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-RESEARCH.md
must_haves:
  - `/admin/codex-auth` existe e é admin-only
  - login inicial é explícito, com OAuth callback/code ou import JSON
  - tokens renovam automaticamente quando possível, sem pedir novo código
  - `channel.key` permanece source of truth
  - `data/codex/license.json` é espelho/cache do canal primário escolhido manualmente
  - sidecar recarrega licença por `mtime`
  - `backend=sdk` hard-fails sem fallback silencioso para `~/.codex/auth.json`
  - drawer do canal Codex mostra só status + link para `/admin/codex-auth`
---

# Phase 06: Login Explícito + Armazenamento Licença (SDK-02) — Plan

Goal: entregar fluxo explícito de autenticação Codex SDK no admin, persistindo credenciais em `channel.key`, espelhando o canal primário manual em `data/codex/license.json`, e removendo fallback silencioso do sidecar para credenciais do host.

## Locked Decisions

- D-01: página dedicada `/admin/codex-auth`; drawer do canal Codex só status + link.
- D-02: dois blocos na mesma página: OAuth callback/code e import JSON.
- D-03: status completo: email, account_id, expired, last_refresh, source.
- D-04: refresh manual + export/download completo por ação explícita.
- D-05: `channel.key` é source of truth.
- D-06: `data/codex/license.json` é espelho/cache auxiliar do canal primário manual.
- D-07: múltiplos canais `backend=sdk` são permitidos; primário é escolha manual.
- D-08: sidecar recarrega por `mtime`.
- D-09: `backend=sdk` hard-fails se licença estiver ausente/inválida.
- D-10: nunca usar fallback silencioso para `~/.codex/auth.json`.
- UI-SPEC: usar `SectionPageLayout`, cards, status badges, tokens de tema, i18n, e zero dependências novas.

## Push Policy

| Operation | Authorization |
|---|---|
| Commit local de docs/plano | Auto |
| Commit local de implementação | Auto depois de tests relevantes |
| `git push origin <branch>` | Não faz parte desta phase-plan; pedir autorização explícita |
| Deploy/restart produção | Não faz parte desta phase-plan; pedir autorização explícita |

## Threat Model

<threat_model>

Sensitive assets:

- `access_token`
- `refresh_token`
- `id_token`
- `data/codex/license.json`
- `channel.key`

Controls required:

- Nunca renderizar tokens completos na página, exceto export/download explícito.
- Export route admin-only, no cache, content-disposition attachment.
- Mirror write atômico: write temp + chmod 0600 + rename.
- No logs com tokens.
- No localStorage/Zustand para credenciais.
- Backend validates imported JSON before writing channel key or mirror.

</threat_model>

## Tasks

### Task 01: Backend service for Codex license status and mirror

<read_first>
- `.planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-CONTEXT.md`
- `.planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-UI-SPEC.md`
- `service/codex_credential_refresh.go`
- `relay/channel/codex/oauth_key.go`
- `model/option.go`
- `model/channel.go`
</read_first>

<acceptance_criteria>
- `service/codex_license.go` exists.
- Service can parse and validate Codex credential JSON using `common.Unmarshal`.
- Service can list Codex channels where channel type is `constant.ChannelTypeCodex` and `codex_backend == "sdk"`.
- Service can read/write `CodexSDKPrimaryChannelID` through `model.UpdateOption` / `common.OptionMap`.
- Service writes `data/codex/license.json` atomically from the selected primary channel key.
- File permissions for mirror are `0600` when supported by filesystem.
- Unit-testable functions avoid Gin dependencies.
</acceptance_criteria>

<action>
Create `service/codex_license.go` with these responsibilities:

1. Types:
   - `CodexLicenseStatus`
   - `CodexSDKChannelStatus`
   - `CodexLicenseMirrorStatus`
   - `CodexCredentialInput`
2. Constants:
   - `CodexSDKPrimaryChannelIDOption = "CodexSDKPrimaryChannelID"`
   - `CodexLicensePath = "data/codex/license.json"`
3. Functions:
   - `ParseCodexCredential(raw string) (*CodexOAuthKey, error)` reusing existing struct semantics.
   - `NormalizeCodexCredential(raw string) (string, *CodexOAuthKey, error)` to validate/import and return canonical JSON.
   - `ListCodexSDKChannels() ([]CodexSDKChannelStatus, error)` using GORM, no raw SQL unless unavoidable.
   - `GetCodexPrimaryChannelID() (int, error)`.
   - `SetCodexPrimaryChannelID(channelID int) error`.
   - `BuildCodexLicenseStatus() (*CodexLicenseStatus, error)`.
   - `WriteCodexLicenseMirrorFromChannel(channelID int) (*CodexLicenseMirrorStatus, error)`.
   - `ClearCodexLicenseMirror() error` only if needed by UI copy; otherwise leave out.
4. Atomic write pattern:
   - `os.MkdirAll(filepath.Dir(path), 0700)`.
   - Write to temp file in same directory.
   - `common.Marshal` for canonical output.
   - `os.Chmod(tmp, 0600)` best-effort.
   - `os.Rename(tmp, final)`.
5. Do not log raw credentials.
</action>

### Task 02: Backend API endpoints for `/admin/codex-auth`

<read_first>
- `router/api-router.go` lines around `channelRoute`
- `controller/codex_oauth.go`
- `controller/channel.go` patterns for API responses
- `common` API response helpers
- `service/codex_license.go` from Task 01
</read_first>

<acceptance_criteria>
- New page-level routes are registered under `/api/channel/codex/auth/*` and protected by existing `channelRoute.Use(middleware.AdminAuth())`.
- `GET /api/channel/codex/auth/status` returns primary channel, SDK channel candidates, credential status, and mirror status.
- `POST /api/channel/codex/auth/import` validates JSON and writes it to the selected channel key.
- `POST /api/channel/codex/auth/primary` manually selects primary channel and writes mirror.
- `POST /api/channel/codex/auth/refresh` refreshes selected/primary channel credential and resyncs mirror when applicable.
- `GET /api/channel/codex/auth/export` downloads full credential JSON for selected/primary channel only by explicit request.
- Existing per-channel OAuth routes keep working.
</acceptance_criteria>

<action>
Create/extend `controller/codex_auth.go` (or append to `controller/codex_oauth.go` only if project style prefers fewer files):

1. Request DTOs:
   - `codexAuthImportRequest { ChannelID int; Credential string }`
   - `codexAuthPrimaryRequest { ChannelID int }`
   - `codexAuthRefreshRequest { ChannelID int }`
2. Handlers:
   - `GetCodexAuthStatus`
   - `ImportCodexCredential`
   - `SetCodexPrimaryChannel`
   - `RefreshCodexPrimaryCredential`
   - `ExportCodexCredential`
3. Register routes in `router/api-router.go` near existing Codex routes:
   - `channelRoute.GET("/codex/auth/status", controller.GetCodexAuthStatus)`
   - `channelRoute.POST("/codex/auth/import", controller.ImportCodexCredential)`
   - `channelRoute.POST("/codex/auth/primary", controller.SetCodexPrimaryChannel)`
   - `channelRoute.POST("/codex/auth/refresh", controller.RefreshCodexPrimaryCredential)`
   - `channelRoute.GET("/codex/auth/export", controller.ExportCodexCredential)`
4. Reuse existing channel-scoped OAuth endpoints from the page when target channel is known:
   - `POST /api/channel/:id/codex/oauth/start`
   - `POST /api/channel/:id/codex/oauth/complete`
5. For import and OAuth completion, update channel key with GORM and call:
   - `model.InitChannelCache()`
   - `service.ResetProxyClientCache()`
   - mirror sync if imported channel is primary.
6. Use `common.Marshal` / `common.Unmarshal`; do not import `encoding/json` for marshal/unmarshal operations.
</action>

### Task 03: Persist primary option and sync auto-refresh mirror

<read_first>
- `model/option.go`
- `service/codex_credential_refresh.go`
- `service/codex_credential_refresh_task.go`
- `main.go` Codex refresh startup
</read_first>

<acceptance_criteria>
- `model.InitOptionMap()` includes `CodexSDKPrimaryChannelID` default `""`.
- Manual refresh of primary channel updates both `channel.key` and `data/codex/license.json`.
- Auto-refresh task updates `data/codex/license.json` after refreshing the selected primary channel.
- Non-primary channel refresh does not overwrite mirror.
- No cross-DB-specific SQL added.
</acceptance_criteria>

<action>
1. Add `common.OptionMap["CodexSDKPrimaryChannelID"] = ""` to `model.InitOptionMap()`.
2. Extend refresh flow:
   - After `RefreshCodexChannelCredential`, check whether refreshed channel ID matches `GetCodexPrimaryChannelID()`.
   - If yes, call `WriteCodexLicenseMirrorFromChannel(channelID)`.
   - Log only channel ID and metadata; never token contents.
3. Keep refresh failure non-fatal for other channels. For primary mirror sync failure, return API error on manual refresh and log warning on background refresh.
</action>

### Task 04: Sidecar license-only auth and `mtime` reload

<read_first>
- `integration/codex-sidecar/main.py`
- `podman-compose.yml` codex-sidecar volume
- `.planning/phases/06-login-expl-cito-armazenamento-licen-a-sdk-02/06-CONTEXT.md` D-08/D-09/D-10
</read_first>

<acceptance_criteria>
- Sidecar uses only `/app/data/license.json` inside container.
- No fallback to `~/.codex/auth.json` remains.
- Missing/invalid license returns a clear hard-fail for SDK requests.
- Sidecar checks `mtime` before SDK requests and reloads/reinitializes SDK if the file changed.
- Reload path is concurrency-safe.
- `python3 -m py_compile integration/codex-sidecar/main.py` passes.
</acceptance_criteria>

<action>
Modify `integration/codex-sidecar/main.py`:

1. Replace `_load_license()` with a single-path loader:
   - container path: `/app/data/license.json`
   - optional env override: `CODEX_LICENSE_PATH`, default `/app/data/license.json`
2. Remove `os.path.expanduser("~/.codex/auth.json")` fallback.
3. Add global license state:
   - `_license_mtime`
   - `_license_lock`
   - `_license_loaded`
4. Add `ensure_license_loaded()` called at app startup and before each `/v1/codex/run` and `/v1/codex/thread` request.
5. If file missing/invalid, raise HTTP 503 with message `Codex license not configured`.
6. If `mtime` changed, close existing SDK handle if present and create a new SDK handle.
7. Keep health endpoint returning mirror state without token values:
   - `license_present`
   - `license_mtime`
   - `license_loaded`
</action>

### Task 05: Frontend API/types/hooks for Codex Auth page

<read_first>
- `web/default/src/features/channels/api.ts`
- `web/default/src/features/channels/types.ts`
- `web/default/src/lib/api.ts`
- `web/default/AGENTS.md`
- `06-UI-SPEC.md`
</read_first>

<acceptance_criteria>
- New `web/default/src/features/codex-auth/api.ts` wraps all page-level endpoints.
- New `types.ts` defines typed status, SDK channel, mirror status, and mutation payloads without `any`.
- React Query keys are stable and namespaced under `['codex-auth', ...]`.
- No credential values are stored in localStorage/Zustand.
- `bun run typecheck` passes after frontend tasks.
</acceptance_criteria>

<action>
Create `web/default/src/features/codex-auth/`:

1. `types.ts`:
   - Zod or explicit TypeScript types for API response payloads.
   - Include `CodexAuthStatus`, `CodexSDKChannel`, `CodexMirrorStatus`, `CodexCredentialSummary`.
2. `api.ts`:
   - `getCodexAuthStatus()` → `GET /api/channel/codex/auth/status`
   - `importCodexCredential(channelId, credential)` → `POST /api/channel/codex/auth/import`
   - `setCodexPrimaryChannel(channelId)` → `POST /api/channel/codex/auth/primary`
   - `refreshCodexCredential(channelId?)` → `POST /api/channel/codex/auth/refresh`
   - `exportCodexCredential(channelId?)` → `GET /api/channel/codex/auth/export`, blob/download handling in UI action
   - `startCodexOAuthForChannel(channelId)` can reuse `/api/channel/:id/codex/oauth/start`
   - `completeCodexOAuthForChannel(channelId, input)` can reuse `/api/channel/:id/codex/oauth/complete`
3. Use existing `api` axios instance and channel action error style (`skipBusinessError` / `skipErrorHandler`) where relevant.
</action>

### Task 06: Build `/admin/codex-auth` page from UI-SPEC

<read_first>
- `06-UI-SPEC.md`
- `web/default/src/features/channels/components/dialogs/codex-oauth-dialog.tsx`
- `web/default/src/features/channels/index.tsx`
- `web/default/src/routes/_authenticated/channels/index.tsx`
- `web/default/src/components/ui/*` primitives listed in UI-SPEC
</read_first>

<acceptance_criteria>
- `/admin/codex-auth` loads for admin users.
- Non-admin users redirect to `/403`.
- Page has header, status summary, OAuth block, import JSON block, SDK channels list, and safety alert.
- OAuth flow can write generated credential directly to selected channel.
- Import flow validates JSON client-side before submit and shows missing fields without printing token values.
- Primary channel selection is manual.
- Export/download requires explicit click.
- All visible strings use `t()`.
- Layout follows `06-UI-SPEC.md` spacing/color/typography contracts.
</acceptance_criteria>

<action>
1. Add route file:
   - `web/default/src/routes/_authenticated/admin/codex-auth.tsx`
   - `createFileRoute('/_authenticated/admin/codex-auth')`
   - same admin role guard as channels route.
2. Add feature entry:
   - `web/default/src/features/codex-auth/index.tsx`
3. Suggested components:
   - `codex-auth-status-card.tsx`
   - `codex-auth-oauth-card.tsx`
   - `codex-auth-import-card.tsx`
   - `codex-auth-channel-list.tsx`
4. Use `SectionPageLayout`:
   - title `Codex Authorization`
   - actions `Refresh status`, `Export credential`
5. Use React Query mutations and invalidate `['codex-auth', 'status']` after changes.
6. Keep token values redacted in page body. For JSON import preview, show account/email/expiry only.
</action>

### Task 07: Replace Codex drawer auth with status + link

<read_first>
- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx` around Codex block
- `web/default/src/features/channels/components/dialogs/codex-oauth-dialog.tsx`
- `06-UI-SPEC.md` drawer contract
</read_first>

<acceptance_criteria>
- Channel drawer no longer opens the full Codex OAuth dialog.
- Type 57 drawer shows compact Codex status/help text.
- Drawer includes link/button to `/admin/codex-auth` with copy `Manage Codex authorization`.
- Drawer still allows editing/saving `key` manually.
- No regression for non-Codex channel types.
</acceptance_criteria>

<action>
1. Remove `CodexOAuthDialog` usage from drawer if it is no longer needed there.
2. Replace `Authorize` button with a typed router `Link` or `Button asChild` to `/admin/codex-auth`.
3. If editing an existing Codex channel, display minimal channel credential summary if available from parsed form/key state:
   - account_id present/missing
   - expired present/missing
   - no token values.
4. Keep helper copy: `Codex channels use an OAuth JSON credential as the key. Manage authorization on the dedicated Codex Authorization page.`
</action>

### Task 08: i18n and static copy sync

<read_first>
- `web/default/AGENTS.md` section 3.1
- `web/default/src/i18n/static-keys.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
</read_first>

<acceptance_criteria>
- New visible strings are discoverable by `bun run i18n:sync`.
- `bun run i18n:sync` completes without corrupting existing locale files.
- No direct visible English string bypasses `t()` in new components.
</acceptance_criteria>

<action>
1. Run `cd web/default && bun run i18n:sync` after adding page strings.
2. If sync misses constants, add keys to `src/i18n/static-keys.ts` according to `web/default/AGENTS.md`.
3. Do not hand-edit generated locale noise unless needed to fix broken JSON.
</action>

### Task 09: Backend and frontend validation

<read_first>
- `06-CONTEXT.md`
- `06-UI-SPEC.md`
- `06-RESEARCH.md`
</read_first>

<acceptance_criteria>
- `go test ./service ./relay/channel/codex ./controller` passes or failures are documented with exact unrelated/pre-existing reason.
- `go build ./...` passes.
- `python3 -m py_compile integration/codex-sidecar/main.py` passes.
- `cd web/default && bun run typecheck` passes.
- `cd web/default && bun run build` passes.
- `cd web/default && bun run i18n:sync` has expected diff only.
</acceptance_criteria>

<action>
Run commands:

```bash
go test ./service ./relay/channel/codex ./controller
go build ./...
python3 -m py_compile integration/codex-sidecar/main.py
cd web/default && bun run i18n:sync
cd web/default && bun run typecheck
cd web/default && bun run build
```

If a command fails, fix the implementation. If the failure is unrelated/pre-existing, document exact file/error in execution log before proceeding.
</action>

### Task 10: Runtime smoke and visual validation

<read_first>
- `podman-compose.yml`
- `06-UI-SPEC.md`
- `web/default/src/routes/_authenticated/admin/codex-auth.tsx`
</read_first>

<acceptance_criteria>
- `/admin/codex-auth` visually matches `06-UI-SPEC.md`.
- Channel drawer type 57 visually shows compact status + link only.
- `GET /api/channel/codex/auth/status` returns JSON under admin session/cookie.
- Sidecar health reports license state without host fallback.
- If stack is unavailable, this task records exact blocker and leaves automated build/type gates passing.
</acceptance_criteria>

<action>
1. If local stack is running, open `/admin/codex-auth` in browser and inspect with browser vision against `06-UI-SPEC.md`.
2. Open channel drawer for a Codex channel and inspect with browser vision.
3. Smoke backend endpoint with authenticated/admin context when available.
4. Restart sidecar only if user has already authorized local dev restart in this execution session; otherwise document as deferred runtime validation.
</action>

---

## Verification Checklist

- [ ] `06-UI-SPEC.md` exists and is approved.
- [ ] `service/codex_license.go` implements parse/status/mirror/primary logic.
- [ ] `CodexSDKPrimaryChannelID` option exists.
- [ ] API routes under `/api/channel/codex/auth/*` work.
- [ ] Sidecar no longer falls back to `~/.codex/auth.json`.
- [ ] Sidecar reloads mirror on `mtime` change.
- [ ] `/admin/codex-auth` exists and is admin-only.
- [ ] Drawer has status + link only.
- [ ] Tokens are never shown in normal UI.
- [ ] Export is explicit download only.
- [ ] `go build ./...` passes.
- [ ] `bun run typecheck` passes.
- [ ] `bun run build` passes.
- [ ] Visual validation completed or documented as blocked.

## Notes

- Phase 05 plan mentioned fallback to `~/.codex/auth.json` for development. Phase 06 intentionally removes that fallback per D-09/D-10.
- `data/codex/license.json` is not a second source of truth. If channel key and mirror diverge, the selected primary channel wins and mirror must be rewritten from it.
- Multiple SDK channels are valid. The UI must not auto-pick a primary in multi-channel state.
