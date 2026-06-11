# Phase 10 — Device Auth + Real Models + OpenAI Codex OAuth Branding

**Status:** Planning
**Requirements:** DEV-01 (Device Auth Upload), DEV-02 (PKCE Callback Paste), DEV-03 (Real Models), DEV-04 (Branding)

---

## Architecture Decision (Critical)

### Why no server-side device auth

Cloudflare bot protection on `auth.openai.com` blocks all non-browser HTTP requests
from our server. The Codex CLI works because it runs on the end-user's machine
where a browser is available for the OAuth verification page.

### The TWO supported auth flows

| Flow | Name | How it works | Default |
|------|------|-------------|---------|
| 1 | **Device Auth Upload** | Admin runs `codex login --device-auth` locally -> uploads `auth.json` -> router parses & saves | YES - PRIMARY |
| 2 | **PKCE Callback Paste** | Current flow: admin opens browser -> logs in -> copies `code` param -> pastes in dashboard | SECONDARY |

### Why Device Auth Upload is the primary

1. Admin sees a short code (e.g. `AEF2-K95ZY`)
2. Enters it at `https://auth.openai.com/codex/device`
3. Gets `auth.json` locally
4. Uploads to router -- zero URL copying, zero callback debugging

---

## Tasks

### T1 -- Rename: Codex OAuth -> OpenAI Codex OAuth

- [ ] `web/default/src/features/channels/constants.ts`: rename type 57 label
- [ ] All i18n keys referencing "Codex" -> prefix with "OpenAI Codex"
- [ ] UI: channel type dropdown, OAuth section header

### T2 -- Backend: Device Auth JSON Upload Endpoint

- [ ] `POST /api/channel/codex/oauth/device/upload` -- accepts file upload or raw JSON
- [ ] Parses `auth.json` format: `{access_token, refresh_token, account_id, email, expires_at}`
- [ ] Validates: access_token non-empty, account_id present
- [ ] Saves credential to channel key field
- [ ] Returns credential info: email, account_id, expires_at, token preview

### T3 -- Frontend: Device Auth Upload UI (Primary)

- [ ] Shows `codex login --device-auth` command for admin to copy
- [ ] Shows verification URL `https://auth.openai.com/codex/device`
- [ ] "Drop auth.json here or click to browse" file upload
- [ ] Or paste JSON manually
- [ ] After upload: shows credential status card
- [ ] "Advanced: PKCE callback method" toggle link

### T4 -- Frontend: PKCE Callback Paste (Secondary)

- [ ] Keep existing PKCE flow behind a toggle/link
- [ ] Default view shows Device Auth Upload

### T5 -- Backend: Real Model Fetching from Codex Upstream

- [ ] Fetch models from upstream using credential
- [ ] Parse model list from upstream response (chatgpt.com/backend-api/codex/responses)
- [ ] Update channel model list automatically

### T6 -- Frontend: Auto-load Models After Auth

- [ ] After successful auth: auto-trigger model fetch
- [ ] Show spinner while loading, then count: "12 models loaded"
- [ ] Populate channel model selection

### T7 -- Integration Testing

- [ ] `go build ./...` passes
- [ ] `bun run typecheck` passes
- [ ] Device auth upload works end-to-end
- [ ] PKCE fallback works
- [ ] Real models populate after auth
- [ ] Channel save retains credential + models

---

## File Inventory

| File | Action | Purpose |
|------|--------|---------|
| `controller/codex_device.go` | NEW | Device auth upload endpoint |
| `service/codex_models.go` | NEW | Real model fetching from upstream |
| `router/api-router.go` | MODIFY | Add device upload route |
| `relay/channel/codex/adaptor.go` | MODIFY | Model list from upstream (not static) |
| `web/.../constants.ts` | MODIFY | Rename to OpenAI Codex OAuth |
| `web/.../codex-oauth-section.tsx` | REWRITE | Device auth primary + PKCE secondary |
| `web/.../channel-mutate-drawer.tsx` | MODIFY | Auto-load models after auth |
| i18n locale files | MODIFY | Add new i18n keys |

---

## Verification

- [ ] `go build ./...` compila
- [ ] `bun run typecheck` 0 errors
- [ ] `bun run build` succeeds
- [ ] Canal OpenAI Codex OAuth -> Device Auth Upload shows `codex login --device-auth` + file upload
- [ ] Upload de auth.json populates credential status
- [ ] PKCE fallback funciona como antes
- [ ] Models sao carregados do upstream real apos auth
- [ ] Dashboard mostra lista de modelos reais do Codex
