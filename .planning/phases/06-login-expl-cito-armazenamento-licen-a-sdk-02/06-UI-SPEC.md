---
phase: 06
slug: login-expl-cito-armazenamento-licen-a-sdk-02
status: approved
shadcn_initialized: true
preset: existing-atuius-default
created: 2026-06-06
source: /gsd-ui-phase 06
---

# Phase 06 — UI Design Contract

> Visual and interaction contract for `/admin/codex-auth`, generated inline because `gsd-ui-researcher` and `gsd-ui-checker` resolved to empty agent skills in Hermes CLI.

---

## Scope

Phase 06 adds a dedicated admin page for explicit Codex SDK authentication and license storage.

Locked UI surfaces:

1. `/admin/codex-auth` authenticated admin route.
2. Channels drawer Codex block becomes status + link only.
3. Same page supports both credential creation paths:
   - OAuth authorization code / callback URL flow.
   - JSON import for `{access_token, refresh_token, account_id}` credentials.
4. Same page supports lifecycle actions:
   - Select manual primary Codex SDK channel.
   - Refresh credential manually.
   - Export/download full credential JSON.
   - Display complete status: email, account_id, expired, last_refresh, source.

Out of scope:

- No hidden fallback to host `~/.codex/auth.json`.
- No automatic primary selection when multiple SDK channels exist.
- No extra sidebar shortcut beyond the admin route entry.
- No new visual system or third-party registry blocks.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | shadcn-compatible local primitives already present in `web/default/src/components/ui` |
| Preset | existing project theme, no new preset |
| Component library | Base UI + local shadcn-style primitives |
| Styling | Tailwind CSS, `cn()` for dynamic classes, CSS variables from `src/styles/theme.css` |
| Icon library | lucide-react, matching existing Codex OAuth dialog usage |
| Font | `Public Sans` via `--font-body` / `--font-sans` |
| Layout shell | existing `AuthenticatedLayout` + `SectionPageLayout` |
| Data state | React Query + existing `api` axios instance |
| i18n | `useTranslation()` in React components; all visible strings wrapped with `t()` |

Required existing primitives:

- `Button`
- `Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardContent`
- `Alert`, `AlertDescription`
- `Badge`
- `Input`
- `Textarea`
- `Label`
- `Select` or existing combobox/select pattern
- `Skeleton`
- `Separator`
- `Tooltip` only for compact metadata hints

Do not introduce new UI dependencies.

---

## Route and Page Contract

| Item | Contract |
|------|----------|
| Route path | `/admin/codex-auth` |
| Route file | `web/default/src/routes/_authenticated/admin/codex-auth.tsx` |
| Feature root | `web/default/src/features/codex-auth/` |
| Feature entry | `web/default/src/features/codex-auth/index.tsx` |
| Access | admin-only; redirect to `/403` if `auth.user.role < ROLE.ADMIN` |
| Page title | `Codex Authorization` |
| Page subtitle | `Manage explicit Codex SDK credentials without reading host credentials.` |
| Drawer link copy | `Manage Codex authorization` |

Page structure:

1. Header row:
   - title + subtitle left.
   - right actions: `Refresh status`, `Export credential`.
2. Status summary card full width:
   - selected primary channel.
   - credential source.
   - account email.
   - account ID.
   - expired state.
   - last refresh.
   - sidecar sync state for `data/codex/license.json`.
3. Two-column credential input grid on desktop; stacked on mobile:
   - OAuth code/callback block.
   - Import JSON block.
4. SDK channels table/list card:
   - shows Codex channels using `backend=sdk`.
   - manual primary selection action.
   - status badge per channel.
5. Help / safety alert:
   - explains that channel key is source of truth and `license.json` is only a cache mirror.

---

## Spacing Scale

Declared values must stay in Tailwind multiples of 4px.

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | icon/text gaps, badge icon gap |
| sm | 8px | inline button groups, metadata row gap |
| md | 16px | card internal vertical rhythm, form field spacing |
| lg | 24px | card padding and page section gaps |
| xl | 32px | desktop grid gap between OAuth/import columns |
| 2xl | 48px | major page section separation only if needed |
| 3xl | 64px | not used in this phase |

Required Tailwind patterns:

- Page wrapper: `space-y-6`.
- Header: `flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between`.
- Credential grid: `grid gap-6 lg:grid-cols-2`.
- Card content: `space-y-4`.
- Form group: `space-y-2`.
- Button groups: `flex flex-wrap items-center gap-2`.
- Status metadata grid: `grid gap-3 sm:grid-cols-2 lg:grid-cols-4`.

Exceptions: none.

---

## Typography

| Role | Size | Weight | Line Height | Usage |
|------|------|--------|-------------|-------|
| Body | 14px (`text-sm`) | 400 | 1.5 | descriptions, helper text, form body |
| Label | 14px (`text-sm`) | 500 | 1.25 | field labels, metadata labels |
| Metadata | 12px (`text-xs`) | 400/500 | 1.33 | channel IDs, timestamps, source labels |
| Heading | 20px (`text-xl`) | 600 | 1.25 | card titles when outside `SectionPageLayout.Title` |
| Page title | existing `SectionPageLayout.Title` | existing | existing | page title only |
| Monospace credential preview | 12px (`text-xs`) | 400 | 1.5 | redacted JSON preview / account ID snippets |

Rules:

- Use `text-muted-foreground` for descriptions and metadata labels.
- Do not render full tokens inline in normal text. Credential displays must be redacted unless inside explicit export/download flow.
- Account ID can be shown fully if it is not a secret; tokens never shown fully on the page body.

---

## Color

Use existing theme tokens. Do not hardcode hex values.

| Role | Token | Usage |
|------|-------|-------|
| Dominant (60%) | `bg-background`, `text-foreground` | page shell |
| Secondary (30%) | `bg-card`, `border-border`, `bg-muted` | cards, metadata tiles, empty states |
| Accent (10%) | `bg-primary text-primary-foreground`, `text-info`, `text-success`, `text-warning` | primary action + status semantics only |
| Destructive | `destructive` button variant / `text-destructive` | invalid credential, delete/clear only |
| Success | `text-success`, success badge variant if available | credential valid, refresh succeeded |
| Warning | `text-warning`, warning alert | expired/near-expiry, sidecar not synced |
| Neutral | `text-muted-foreground` | unknown/missing metadata |

Accent reserved for:

- Primary CTA `Start authorization`.
- Success/expired/unknown status badges.
- Focus rings inherited from components.

Never use accent for every button. Secondary actions (`Refresh status`, `Export credential`, `Import JSON`) use `variant='outline'` or `variant='secondary'`.

---

## Component Contracts

### Status summary card

Required states:

- Loading: skeleton rows for metadata tiles.
- No primary channel: warning alert + CTA `Select primary channel` disabled until SDK channels load.
- Valid credential: success badge `Valid`.
- Expired credential: warning/destructive badge `Expired` + CTA `Refresh credential`.
- Missing license mirror: warning line `license.json not synced`.
- API error: alert with retry action.

Required metadata tiles:

1. `Primary channel`: channel name + `#id`.
2. `Source`: `channel.key` or `license.json mirror` label.
3. `Account`: email or `Unknown`.
4. `Account ID`: account_id or `Missing`.
5. `Expired`: `Yes` / `No` / `Unknown`.
6. `Last refresh`: relative timestamp + absolute tooltip/title.

### OAuth block

Required actions:

- `Start authorization` opens provider URL in new tab when available.
- `Copy authorization link` appears after start.
- Callback URL input label: `Callback URL`.
- Completion CTA: `Generate credential`.

Required helper copy:

- `Paste the full redirect URL after completing Codex login. It may point to localhost; that is expected.`

Validation:

- Disable completion when callback URL is empty.
- Show field-level error for invalid/missing code/state.
- Never auto-read local host auth files.

### Import JSON block

Required controls:

- Textarea for pasted JSON.
- Optional upload button for `.json` file input.
- CTA `Import credential JSON`.
- Preview summary after parse: account_id, email, token expiry if available.

Validation:

- JSON must parse client-side before submit.
- Must include `access_token`, `refresh_token`, `account_id`.
- Show exact missing fields without printing token values.

### SDK channels list

Required fields:

- Channel name.
- Channel ID.
- Status enabled/disabled.
- `backend=sdk` detected from channel settings.
- Credential state summary.
- Primary marker.

Required action:

- `Set as primary` writes `data/codex/license.json` from selected channel key through backend API.

Manual selection is locked: do not auto-pick first valid channel.

### Channel drawer Codex block

Contract:

- Remove embedded full OAuth dialog behavior from drawer.
- Keep compact status block for type 57 only.
- Show status summary if editing an existing channel.
- Include link/button to `/admin/codex-auth` with copy `Manage Codex authorization`.
- Drawer must still allow editing `key`; page is convenience/auth management, not sole source of truth.

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Page title | `Codex Authorization` |
| Page subtitle | `Manage explicit Codex SDK credentials without reading host credentials.` |
| Primary CTA | `Start authorization` |
| OAuth completion CTA | `Generate credential` |
| Import CTA | `Import credential JSON` |
| Manual refresh CTA | `Refresh credential` |
| Export CTA | `Export credential` |
| Select primary CTA | `Set as primary` |
| Drawer link | `Manage Codex authorization` |
| Empty state heading | `No Codex SDK channel selected` |
| Empty state body | `Create or select a Codex channel using backend=sdk, then set it as the primary license source.` |
| Error state | `Credential update failed. Review the error, keep your current channel key unchanged, and retry.` |
| Expired state | `Credential expired. Refresh it or import a newer JSON credential.` |
| Destructive confirmation | `Clear credential mirror: this removes data/codex/license.json but does not modify channel keys.` |

All copy must be i18n-wrapped with `t()`.

---

## Interaction and State Contract

Required API shape from frontend perspective:

- `GET /api/channel/codex/auth/status`
  - returns primary channel, candidate SDK channels, credential status, mirror status.
- `POST /api/channel/codex/oauth/start`
  - returns authorization URL.
- `POST /api/channel/codex/oauth/complete`
  - accepts callback URL/code and selected channel ID or primary target.
- `POST /api/channel/codex/import`
  - accepts parsed credential JSON and selected channel ID.
- `POST /api/channel/codex/primary`
  - accepts channel ID and writes mirror from that channel key.
- `POST /api/channel/codex/refresh`
  - refreshes selected/primary credential.
- `GET /api/channel/codex/export`
  - downloads full credential JSON.

If implementation reuses existing per-channel endpoints, the page API wrapper must still present this page-level contract to components.

React Query contract:

- Query key: `['codex-auth', 'status']`.
- Invalidate status after import, OAuth complete, primary set, refresh, mirror clear.
- Mutations must use existing toast/error handling pattern.
- No token values stored in Zustand/localStorage.

---

## Accessibility Contract

- Every input has a visible `Label` and matching `id`.
- JSON textarea uses `spellCheck={false}` and `autoComplete='off'`.
- Buttons have stable text; icon-only buttons need `aria-label`.
- Status badges have adjacent text, not color-only meaning.
- Loading states disable only the relevant action, not the full page unless initial load.
- Export/download action requires explicit click.
- Keyboard path: header actions → OAuth block → import block → channels list.

---

## Responsive Contract

| Viewport | Contract |
|----------|----------|
| Mobile | single column; actions wrap; metadata tiles stack; no horizontal table overflow for channels list |
| Tablet | two metadata columns; credential blocks may stack until `lg` |
| Desktop | OAuth/import two-column grid; channels list full width below |

Use card/list layout for channels instead of a dense table on small screens if needed.

---

## Security and Privacy Contract

- Never display full `access_token` or `refresh_token` in normal page content.
- Export/download is the only full-token reveal path and must be explicit.
- Clipboard actions only copy authorization URL, not tokens, unless the user clicks export/download.
- Backend remains source of truth for channel key mutations.
- `channel.key` remains the canonical credential store.
- `data/codex/license.json` is only the selected primary channel mirror/cache for sidecar reload by `mtime`.
- Manual primary choice is required before writing mirror when multiple SDK channels exist.

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| Local `components/ui` | Button, Card, Alert, Badge, Input, Textarea, Label, Select/Combobox, Skeleton, Separator | not required |
| Third-party registry | none | not allowed for this phase |

---

## Checker Sign-Off

- [x] Dimension 1 Copywriting: PASS — all required visible strings are specified and i18n-wrapped.
- [x] Dimension 2 Visuals: PASS — page uses existing admin layout, cards, metadata tiles, and status badges.
- [x] Dimension 3 Color: PASS — theme tokens only; no hardcoded colors.
- [x] Dimension 4 Typography: PASS — project font and explicit text roles locked.
- [x] Dimension 5 Spacing: PASS — 4px-multiple Tailwind scale locked.
- [x] Dimension 6 Registry Safety: PASS — no new dependencies or third-party registry blocks.

**Approval:** approved 2026-06-06
