# Phase 32 — UI Review

**Audited:** 2026-07-12
**Baseline:** `32-UI-SPEC.md`
**Screenshots:** not captured — code-only audit; browser/Playwright unavailable by task constraint. Although `localhost:3000` returned HTTP 200, no runtime visual or interaction inspection was performed.
**Re-audited:** 2026-07-12 — all three original priority findings are resolved in current code.

## Runtime Validation Limitations

The audit cannot confirm computed contrast, light/dark rendering, responsive wrapping at desktop/tablet/mobile widths, clipping or overflow in the drawer, focus trapping, keyboard traversal, screen-reader announcements, popup behavior, or the appearance of loading/error states. Findings below are based on static React, Tailwind, i18n, and test analysis only.

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 4/4 | Required PT-BR operational copy is present, including `Autenticado`, and the panel test now consumes the real `pt.json` resource. |
| 2. Visuals | 3/4 | Hierarchy and visible actions are clear in code, but nested credential cards and uniformly destructive failure badges weaken prioritization. |
| 3. Color | 3/4 | Semantic variants are mostly used, but the fallback alert hardcodes amber light/dark colors instead of a reusable semantic warning treatment. |
| 4. Typography | 3/4 | The implementation stays to two sizes and two weights, but runtime legibility and long PT-BR wrapping are unverified. |
| 5. Spacing | 3/4 | The spacing scale is compact and consistent, but nested `p-4` cards and `space-y-*` usage diverge from local composition guidance. |
| 6. Experience Design | 3/4 | Lifecycle failures now update the inline card and refetch metadata, and blocked popups stop the flow with actionable copy; runtime interaction remains unverified. |

**Overall: 19/24**

---

## Original Priority Findings — Re-audit Status

1. **RESOLVED — Real PT-BR `Authenticated` translation.** `pt.json` now contains `Authenticated: Autenticado` (`pt.json:486`), the key is statically retained (`static-keys.ts:107-109`), and the panel smoke imports `ptTranslations.translation` and asserts `Autenticado` (`codex-credential-panel.test.tsx:23-35`, `codex-credential-panel.test.tsx:93-102`).
2. **RESOLVED — Inline lifecycle errors and metadata reconciliation.** `codexCredentialActionError` is passed into the panel (`channel-mutate-drawer.tsx:3078-3088`), while refresh, probe, and regeneration-complete failure paths set it, invalidate channel/credential queries, and emit a toast (`channel-mutate-drawer.tsx:1338-1392`, `channel-mutate-drawer.tsx:1434-1459`).
3. **RESOLVED — Blocked OAuth popup handling.** `openOAuthAuthorizationWindow` reports a null popup (`codex-regenerate-dialog.tsx:41-52`); the drawer throws actionable localized feedback and reaches `setCodexRegenerateDialogOpen(true)` only after a successful open (`channel-mutate-drawer.tsx:1406-1428`). The helper's blocked/success results are covered by the fifth panel test (`codex-credential-panel.test.tsx:105-119`).

## Remaining Top 3 Priority Fixes

1. **Run browser-based responsive and interaction UAT** — code-only evidence cannot prove popup behavior, focus management, keyboard traversal, light/dark contrast, or 375 px wrapping — capture desktop/tablet/mobile states when Playwright or a browser is available.
2. **Replace the hardcoded amber warning palette with a semantic warning treatment** — manual light/dark colors remain harder to maintain and validate — introduce or reuse a warning Alert variant/token while preserving the UI-SPEC's amber intent.
3. **Remove the nested credential-card treatment** — the Codex `p-4` bordered card still sits inside another `p-4` bordered card — flatten one surface or let the outer credentials section own the frame and spacing.

---

## Detailed Findings

### Pillar 1: Copywriting (4/4)

- **PASS:** `CodexCredentialPanel` renders `t("Authenticated")` (`codex-credential-panel.tsx:113-119`), and the production PT-BR resource now maps it to `Autenticado` (`pt.json:486`), satisfying the required badge copy.
- Required panel, warning, modal, and action strings are present in PT-BR: `Credencial OAuth Codex` (`pt.json:888`), callback guidance (`pt.json:3212`), refresh/regenerate actions (`pt.json:3614`, `pt.json:3625`), missing-refresh warning (`pt.json:4441`), token-safety copy (`pt.json:4560`), and fallback copy (`pt.json:4349`).
- The regression test now imports the real `pt.json` resource and proves `Autenticado`, the main required actions, and forbidden generic controls in static panel markup (`codex-credential-panel.test.tsx:23-35`, `codex-credential-panel.test.tsx:72-103`).
- The popup-blocked message has actionable PT-BR copy (`pt.json:4387`) and a static key entry (`static-keys.ts:107-109`).

### Pillar 2: Visuals (3/4)

- **WARNING:** The Codex panel is a bordered `bg-muted/10 p-4` surface (`codex-credential-panel.tsx:100`) mounted inside another bordered `bg-muted/10 p-4` credential container (`channel-mutate-drawer.tsx:2123`, `channel-mutate-drawer.tsx:3078-3101`). This double-card treatment can dilute the panel's focal hierarchy and consume drawer width, especially on mobile; runtime confirmation is unavailable.
- The code establishes a clear heading, metadata grid, status badge cluster, warnings, and three visible text actions (`codex-credential-panel.tsx:101-132`, `codex-credential-panel.tsx:160-197`, `codex-credential-panel.tsx:228-288`). No action is icon-only.
- Both `requires_regeneration` and any upstream failure use the same destructive badge treatment (`codex-credential-panel.tsx:120-130`). That makes cause and consequence visually equivalent; a warning/outline distinction would improve scanning while preserving destructive emphasis for the required action.
- The dialog has a visible title and description (`codex-regenerate-dialog.tsx:73-83`), satisfying the overlay accessibility composition contract.

### Pillar 3: Color (3/4)

- **WARNING:** Most status UI uses design-system variants (`Badge` secondary/outline/destructive and destructive `Alert`), but the missing-refresh warning hardcodes six amber light/dark classes (`codex-credential-panel.tsx:201`). This duplicates the drawer's ad hoc warning palette and conflicts with the local shadcn guidance to use semantic tokens and avoid manual `dark:` overrides.
- No hex or RGB literals occur in the two Codex components. Accent use is restrained; the primary button is reserved for `Regenerate credential`, while refresh/probe remain outline (`codex-credential-panel.tsx:229-287`).
- `border-border/60` and `bg-muted/10` match the UI-SPEC contract (`codex-credential-panel.tsx:100`). Computed contrast for amber text/background and destructive badges remains unverified without browser rendering.

### Pillar 4: Typography (3/4)

- **WARNING:** Static classes are disciplined—only `text-xs`, `text-sm`, `font-medium`, and `font-semibold` appear in the Codex components—but no runtime evidence proves the dense metadata grid and long PT-BR warnings remain legible or unclipped at 375 px.
- The heading, supporting copy, metadata labels, and values form a coherent hierarchy (`codex-credential-panel.tsx:103-108`, `codex-credential-panel.tsx:161-195`). No new font family or oversized display style was introduced.
- The dialog follows the existing title/body primitives and uses `text-sm`/`text-xs` only for field support copy (`codex-regenerate-dialog.tsx:75-103`).

### Pillar 5: Spacing (3/4)

- **WARNING:** The panel and dialog use `space-y-1`/`space-y-2` (`codex-credential-panel.tsx:102`, `codex-regenerate-dialog.tsx:85`) even though the local shadcn styling rule standardizes vertical stacks on flex plus `gap-*`. This introduces two spacing idioms in the same feature.
- **WARNING:** The nested outer credential card and inner Codex card each add `p-4`, border, and background (`channel-mutate-drawer.tsx:2123`, `codex-credential-panel.tsx:100`), likely creating heavier inset than adjacent credential content.
- No arbitrary pixel/rem spacing values appear in the Codex components. The implementation uses a compact scale of `gap-2`, `gap-3`, `gap-4`, `p-4`, and small margin offsets.
- Responsive intent exists through `sm:flex-row` and `sm:grid-cols-2` (`codex-credential-panel.tsx:101`, `codex-credential-panel.tsx:161`), but actual wrapping and touch layout are not runtime-verified.

### Pillar 6: Experience Design (3/4)

- **PASS:** `openOAuthAuthorizationWindow` detects a blocked popup (`codex-regenerate-dialog.tsx:41-52`). The drawer sets an inline/toast error and does not open the completion modal unless the popup succeeds (`channel-mutate-drawer.tsx:1406-1428`).
- **PASS:** Refresh and probe failures set `codexCredentialActionError`, invalidate the detail and credential queries, and show a toast (`channel-mutate-drawer.tsx:1338-1392`). Regeneration-complete failures do the same and return `false` so the dialog remains available (`channel-mutate-drawer.tsx:1434-1459`).
- **PASS:** The action error is prioritized in the panel's inline destructive Alert path (`channel-mutate-drawer.tsx:3078-3088`, `codex-credential-panel.tsx:153-158`), satisfying the UI-SPEC's toast-and-card requirement.
- Loading spinners and mutual action disabling are implemented for refresh, probe, and regeneration (`codex-credential-panel.tsx:229-287`); the callback submit is disabled while empty/completing and clears transient input on close/success (`codex-regenerate-dialog.tsx:51-68`, `codex-regenerate-dialog.tsx:89-120`).
- Type 57 correctly gates metadata fetching and removes generic Base URL/key/reveal/copy paths (`channel-mutate-drawer.tsx:767-775`, `channel-mutate-drawer.tsx:2765-2786`, `channel-mutate-drawer.tsx:2911-3076`). `shouldWarnAboutBaseUrl` also excludes type 57 (`codex-credential-panel.tsx:67-75`).
- **WARNING:** The five static tests cover the popup helper and production PT-BR rendering, but still do not execute the drawer branch end-to-end, verify inline mutation error rendering after a rejected API call, or exercise focus/transient-input behavior in a browser (`codex-credential-panel.test.tsx:84-149`). The reported 5/5 pass is strong static evidence, not runtime UAT.

---

## Registry Safety

Registry audit skipped: `web/default/components.json` declares `@ai-elements`, but `32-UI-SPEC.md` lists no third-party registry blocks installed or used by this phase. The audited Codex components import existing local primitives only.

## Files Audited

- `.planning/phases/32-codex-oauth-lifecycle-and-upstream-auth-diagnostics/32-UI-SPEC.md`
- `.planning/phases/32-codex-oauth-lifecycle-and-upstream-auth-diagnostics/32-CONTEXT.md`
- `.planning/phases/32-codex-oauth-lifecycle-and-upstream-auth-diagnostics/32-01-PLAN.md`
- `.planning/phases/32-codex-oauth-lifecycle-and-upstream-auth-diagnostics/32-02-PLAN.md`
- `.planning/phases/32-codex-oauth-lifecycle-and-upstream-auth-diagnostics/32-01-SUMMARY.md`
- `.planning/phases/32-codex-oauth-lifecycle-and-upstream-auth-diagnostics/32-02-SUMMARY.md`
- `web/default/src/features/channels/components/codex/codex-credential-panel.tsx`
- `web/default/src/features/channels/components/codex/codex-regenerate-dialog.tsx`
- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx`
- `web/default/src/features/channels/components/codex/codex-credential-panel.test.tsx`
- `web/default/src/i18n/static-keys.ts`
- `web/default/src/i18n/locales/pt.json`
- `web/default/src/i18n/locales/_reports/_sync-report.json`
- `web/default/components.json`
