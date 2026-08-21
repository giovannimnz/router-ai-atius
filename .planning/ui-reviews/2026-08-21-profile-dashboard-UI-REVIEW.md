# Retroactive UI Review - /profile and /dashboard/overview

**Audited:** 2026-08-21
**Baseline:** Abstract 6-pillar standards; no applicable `UI-SPEC.md` was found
**Scope:** Current worktree corrections plus the supplied dark-theme evidence
**Screenshots:** 4 supplied plus 10 post-fix headless captures in `runtime/evidence/ui-20260821/`
**Verdict:** **APPROVED AFTER REMEDIATION**

The min-content and nowrap causes were corrected and then exercised in a dark-theme viewport matrix. All 10 route/viewport checks passed with zero document overflow and all monitored CTAs inside their viewport and card boundaries. The fetch-error, copy, typography, and heading-semantic warnings identified below were also remediated.

## Post-remediation validation

- `/profile` and `/dashboard/overview` passed at 375x812, 500x900, 768x1024, 1280x800, and 1440x900.
- The automated assertion measured `scrollWidth - clientWidth = 0` for every render.
- The two 2FA actions and all four recommended actions remained fully contained.
- 2FA, API-key, and model fetch failures now render explicit retryable error states instead of false disabled/ready states.
- PT-BR uses `Desativar 2FA`, action descriptions use readable `text-sm`, and the 2FA title is a semantic heading.
- Focused lint and frontend typecheck completed successfully.

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 4/4 | Action copy is concise and punctuation is normalized. |
| 2. Visuals | 4/4 | Post-fix captures prove responsive containment in the full viewport matrix. |
| 3. Color | 3/4 | Semantic dark-theme tokens and restrained accents work, but nested surfaces depend on subtle tonal separation. |
| 4. Typography | 4/4 | Hierarchy is clear and essential action descriptions now use `text-sm`. |
| 5. Spacing | 4/4 | Automated measurements prove containment and zero horizontal overflow. |
| 6. Experience Design | 4/4 | Destructive, loading, failure, and retry states are explicit and accessible. |

**Overall: 23/24**

**Verdict rationale:** The only remaining point is a non-blocking recommendation to confirm subtle dark-surface contrast on a calibrated physical display. No functional, containment, state, or accessibility warning remains open in the audited scope.

---

## Remediated Priorities

1. **Prove the overflow correction after the patch** - Capture dark-theme `/profile` and `/dashboard/overview` at 375x812, 500x900, 768x1024, 1280x800, and 1440x900. Assert `document.documentElement.scrollWidth <= document.documentElement.clientWidth` and verify each CTA/action bounding box remains inside its card. Do not substitute `overflow-x-hidden` for containment.
2. **Render explicit failed-to-load states** - Expose the 2FA status error instead of falling back to "disabled", and use `isPending`/`isError` for the API-key and model queries before presenting route/auth/model signals or a ready-to-copy request.
3. **Reduce copy pressure and improve readable semantics** - Translate `Disable 2FA` as `Desativar 2FA`, normalize description punctuation, use at least `text-sm` for essential mobile action descriptions, and render the 2FA card title as a semantic heading.

---

## Evidence Separation

| Evidence | Classification | What it establishes |
|----------|----------------|---------------------|
| S1, 518x413 | **CAPTURE-PROVEN** | The destructive 2FA CTA is clipped by the right card edge in the captured dark-theme state. |
| S2, 500x538 | **CAPTURE-PROVEN** | Recommended-action rows and descriptions extend beyond the parent card's right edge. |
| S3, 1661x1025 | **CAPTURE-PROVEN** | Desktop hierarchy is visible, but the recommended-action content remains clipped and a page-level horizontal scrollbar is present. |
| S4, 1661x1025 | **DUPLICATE EVIDENCE** | Byte-identical to S3; it does not independently increase viewport coverage. |
| Current JSX/CSS classes | **CODE-CONFIRMED** | `min-w-0`, `w-full`, wrapping, and delayed row layout directly address the diagnosed min-content overflow mechanism. |
| Post-fix appearance | **AUTOMATION-PROVEN** | Ten dark-theme route/viewport renders show zero document overflow and all monitored actions contained. |

---

## Initial Findings (Historical, All Closed)

The findings below describe the pre-remediation audit that drove the fixes. Their open-state wording is retained as historical evidence; the post-remediation validation and 23/24 score above are authoritative.

### Pillar 1: Copywriting (3/4)

**WARNING - CAPTURE-PROVEN / CODE-CONFIRMED:** The source CTA is concise (`Disable 2FA`), but PT-BR expands it to `Desativar autenticação de dois fatores`. S1 shows this label clipped in the captured implementation. Wrapping now prevents the same mechanical failure, but the translation still adds avoidable visual pressure. Use `Desativar 2FA` while retaining the full consequence in the confirmation dialog. Evidence: `web/default/src/features/profile/components/two-fa-card.tsx:152`, `web/default/src/i18n/locales/pt.json:1364`.

**WARNING - CAPTURE-PROVEN:** Action descriptions use inconsistent terminal punctuation: Channels has a period while API Keys, Usage Logs, and Pricing do not. The mismatch is visible in S2/S3 and weakens editorial consistency. Normalize all four as sentence fragments without periods or all as complete sentences. Evidence: `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:503`, `:536`, `:543`, `:549`.

**PASS - CODE-CONFIRMED:** Labels such as `Regenerate Backup Codes`, `Create API Key`, `Usage Logs`, and `Pricing` describe outcomes instead of using generic `Submit`, `OK`, or `Click Here` copy. Status labels are paired with supporting context and counts. Evidence: `web/default/src/features/profile/components/two-fa-card.tsx:90-120`, `:138-153`; `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:499-553`.

### Pillar 2: Visuals (2/4)

**WARNING - CAPTURE-PROVEN:** S1 loses the right portion of the destructive 2FA action. S2 and S3 show recommended-action rows escaping the card, clipped descriptions, and broken right-edge alignment. S3 also exposes a horizontal scrollbar. These are not cosmetic-only defects: they disrupt scanning, weaken the card boundary, and make controls appear unfinished.

**PASS - CODE-CONFIRMED:** The current worktree addresses the specific visual causes rather than hiding them. The 2FA actions remain stacked until `2xl` and opt into wrapping; quick actions now use `w-full min-w-0 whitespace-normal`; affected motion/grid wrappers now allow children to shrink. Evidence: `web/default/src/features/profile/index.tsx:58`, `:72`; `web/default/src/features/profile/components/two-fa-card.tsx:81-89`, `:137-153`; `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:417-437`, `:619-623`, `:676-690`.

**WARNING - INFERRED RISK:** The fix is only statically verified. The debug record explicitly cites inspection and typecheck, not a rendered post-fix viewport matrix. Because S1-S3 all show the old failure, the visual pillar cannot score above 2 without new evidence. Evidence: `.planning/debug/profile-dashboard-overflow.md:46-53`.

**PASS - CAPTURE-PROVEN:** Outside the overflow, S3 establishes a clear desktop focal point: the setup guide leads, the request preview supports it, recommended actions are secondary, and usage cards form the next tier.

### Pillar 3: Color (3/4)

**PASS - CODE-CONFIRMED / CAPTURE-PROVEN:** Target files contain zero hardcoded hex/RGB colors and zero direct `text-primary`/`bg-primary`/`border-primary` uses. Dark mode uses semantic charcoal, foreground, muted, destructive, success, warning, and focus-ring tokens. S1/S3 show status and destructive accents used sparingly and reinforced by words/icons, not color alone. Evidence: `web/default/src/styles/theme.css:173-201`; `web/default/src/features/profile/components/two-fa-card.tsx:91-113`, `:146-153`.

**WARNING - NEEDS_HUMAN_REVIEW:** Nested dark surfaces rely on a small background-lightness step (`--background` L 0.235, `--card` L 0.285) and a 10% white border. S2/S3 show several adjacent gray outlined boxes with limited tonal differentiation. Text remains legible, but section grouping may soften on low-brightness or lower-quality displays. Validate with contrast tooling and a real display before considering this pillar excellent. Evidence: `web/default/src/styles/theme.css:175-199`.

### Pillar 4: Typography (3/4)

**WARNING - CAPTURE-PROVEN / CODE-CONFIRMED:** Recommended-action descriptions use `text-xs` (12px) for essential explanatory copy. S2 shows that long PT-BR descriptions become dense in a narrow panel even before clipping is considered. Prefer `text-sm` on mobile/narrow cards, or prove 12px meets the product's accessibility target at 100% zoom. Evidence: `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:429-435`.

**WARNING - CODE-CONFIRMED:** The target files use five named display/body sizes (`xs`, `sm`, `lg`, `xl`, `2xl`) plus 9px and 11px arbitrary sizes. The two sub-12px sizes are decorative, hidden from assistive technology, and therefore low risk, but this distribution is broader than the abstract four-size guideline. Evidence: `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:193-209`.

**PASS - CAPTURE-PROVEN / CODE-CONFIRMED:** Weight usage is restrained to medium and semibold, while size and tracking create a recognizable heading, eyebrow, title, and supporting-text hierarchy. S1-S3 show readable primary headings and clear label emphasis. Evidence: `web/default/src/features/profile/components/two-fa-card.tsx:71-77`; `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:627-637`, `:678-684`.

### Pillar 5: Spacing (2/4)

**WARNING - CAPTURE-PROVEN:** The captured state fails the central spacing contract: content must remain within its parent. S1 clips a CTA; S2/S3 let action rows cross the right inset; S3 has horizontal page scroll. This justifies a score of 2 even though gaps and padding elsewhere are visually consistent.

**PASS - CODE-CONFIRMED:** Most spacing follows Tailwind's scale, led by `gap-2`, `gap-3`, `gap-4`, `px-3`, and `p-3`/`p-5`. The new containment classes are applied at the content, card, and grid levels rather than only at the outermost wrapper.

**WARNING - INFERRED RISK:** The responsive composition still depends on fixed custom tracks (`21rem`, `22rem`, and a `360px` profile side-column minimum). These values are reasonable at their declared breakpoints but have not been rendered against browser chrome, translated PT-BR content, scrollbar width, or intermediate viewport sizes. Evidence: `web/default/src/features/profile/index.tsx:58`; `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:619`, `:623`, `:756`.

**WARNING - INFERRED RISK:** `Main` clips outer overflow while `/profile` creates its own scrolling region. This is intentional, but an overflow assertion must check both the document and the inner scroll container so a regression is not merely hidden. Evidence: `web/default/src/components/layout/components/main.tsx:25-35`; `web/default/src/features/profile/index.tsx:50-52`.

### Pillar 6: Experience Design (2/4)

**PASS - CODE-CONFIRMED:** 2FA has a skeleton while loading. Destructive disablement opens a confirmation dialog, requires a code and an explicit acknowledgement, disables actions while pending, and reports failures. Motion respects reduced-motion preference, and text links/buttons retain focus-visible ring styles. Evidence: `web/default/src/features/profile/components/two-fa-card.tsx:54-65`; `web/default/src/features/profile/components/dialogs/two-fa-disable-dialog.tsx:52-80`, `:94-167`; `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:280-331`; `web/default/src/components/ui/button.tsx:25-39`.

**WARNING - CODE-CONFIRMED RISK:** `useTwoFA` initializes to a valid-looking disabled state, logs fetch errors only to the console, then removes loading. A failed status request can therefore tell the user that 2FA is disabled when its state is actually unknown. Return an error state and render a retry action instead. Evidence: `web/default/src/features/profile/hooks/use-two-fa.ts:29-52`.

**WARNING - CODE-CONFIRMED RISK:** Overview queries collapse unsuccessful responses to empty arrays and do not render query-level loading/error states. The request example simultaneously falls back to `gpt-4o-mini`, so a key can make the preview appear ready before the user's model list is known. Gate readiness on successful model/key queries and show retryable inline feedback. Evidence: `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx:476-492`, `:562-599`.

**WARNING - CODE-CONFIRMED A11Y:** `CardTitle` renders a `div`, and the 2FA title does not provide an explicit heading role or element. Visual hierarchy is present, but heading navigation cannot discover this section reliably. Render it through a semantic heading while preserving styling. Evidence: `web/default/src/components/ui/card.tsx:54-64`; `web/default/src/features/profile/components/two-fa-card.tsx:71-77`.

**WARNING - CAPTURE-PROVEN:** In S1-S3, clipped actions and horizontal scrolling increase interaction cost, especially for keyboard, zoom, and touch users. Current source likely reduces this risk, but only post-fix interaction and screenshot evidence can close it.

---

## Completed Validation Matrix

| Check | Expected Result |
|-------|-----------------|
| `/profile`, dark, 375/500/768/1440 widths | Both 2FA actions fully visible; labels may wrap; no horizontal scroll. |
| `/dashboard/overview`, dark, 375/500/768/1280/1440 widths | Every recommended-action row remains inside its card; descriptions wrap naturally; no horizontal scroll. |
| Browser zoom at 200% | Reading order and action access remain intact without two-dimensional scrolling. |
| Keyboard-only pass | All CTAs and links show focus, dialogs trap/restore focus, and no clipped target is required. |
| API failure simulation | 2FA, API keys, and models show unknown/error plus retry, never a false disabled/ready state. |
| Reduced-motion pass | Decorative animation stops while content and focus behavior remain unchanged. |

---

## Files Audited

- `web/default/src/features/profile/index.tsx`
- `web/default/src/features/profile/components/two-fa-card.tsx`
- `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx`
- `.planning/debug/profile-dashboard-overflow.md`
- `web/default/src/features/profile/hooks/use-two-fa.ts` (supporting state audit)
- `web/default/src/features/profile/components/dialogs/two-fa-disable-dialog.tsx` (supporting destructive-flow audit)
- `web/default/src/components/ui/button.tsx` (supporting interaction/layout audit)
- `web/default/src/components/ui/card.tsx` (supporting semantics audit)
- `web/default/src/components/page-transition.tsx` (supporting reduced-motion/layout audit)
- `web/default/src/components/layout/components/main.tsx` (supporting overflow audit)
- `web/default/src/components/status-badge.tsx` (supporting status/accessibility audit)
- `web/default/src/i18n/locales/pt.json` (supporting copy audit)
- `web/default/src/styles/theme.css` (supporting dark-theme audit)
- Supplied attachments S1-S4; S3 and S4 have identical SHA-256 `5bdb40edc5c7694f83f733cad76f79acf05c0b4e084e2bccf37b91760fdb9aaa`

## Recommendation Count

- Priority fixes: 3
- Additional warnings: 8
- Blockers: 0
