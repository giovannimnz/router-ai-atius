---
phase: 21
slug: feat-pt-native-pr
status: approved
shadcn_initialized: true
preset: web/default/components.json style=base-nova
created: 2026-07-04
---

# Phase 21 - UI Design Contract

> Visual and interaction contract for native PT-BR language support in the default and classic frontends. This phase changes existing language options, locale resources, and language persistence behavior only; it does not redesign product screens.

---

## Design System

| Property | Value |
|----------|-------|
| Tool | shadcn initialized in `web/default`; none for `web/classic` |
| Preset | `base-nova`, neutral base, CSS variables |
| Component library | `web/default`: Base UI wrappers via existing `@/components/ui/*`; `web/classic`: existing Semi UI components |
| Icon library | Existing icons only; keep current project icon libraries and do not add a new package |
| Font | `web/default`: existing `Public Sans` theme stack; `web/classic`: existing Lato/Helvetica/Microsoft YaHei stack |

---

## Spacing Scale

| Token | Value | Usage |
|-------|-------|-------|
| xs | 4px | Existing menu offsets, icon alignment, primitive side offsets |
| sm | 8px | Compact menu/select item padding and gaps |
| md | 16px | Form row spacing and standard content gaps |
| lg | 24px | Existing settings/card section spacing |
| xl | 32px | Existing larger section gaps only |
| 2xl | 48px | Not introduced by this phase |
| 3xl | 64px | Not introduced by this phase |

Exceptions: keep existing control sizes, including the default header language trigger, default select trigger, and classic Semi button/select sizing. Phase 21 must not introduce new spacing tokens or layout primitives.

---

## Typography

| Role | Size | Weight | Line Height |
|------|------|--------|-------------|
| Helper/description | 12px | 400 | 1.5 |
| Menu/select/body label | 14px | 400 | 1.5 |
| Card heading | 18px | 600 | 1.2 |
| Large card heading breakpoint | 20px | 600 | 1.2 |

Allowed weights: `400` regular and `600` semibold only for new or changed visible language entries. Do not add responsive font scaling. `Português` must fit in existing menu/select widths without custom typography.

---

## Color

| Role | Value | Usage |
|------|-------|-------|
| Dominant (60%) | Default: `var(--background)`; Classic: `var(--semi-color-bg-0)` | App background and stable surfaces |
| Secondary (30%) | Default: `var(--card)`, `var(--popover)`, `var(--muted)`; Classic: `var(--semi-color-bg-overlay)`, `var(--semi-color-fill-0)` | Existing cards, dropdowns, selectors |
| Accent (10%) | Default: `var(--accent)` / `var(--accent-foreground)`; Classic: existing selected-language classes | Selected/focused language item only |
| Destructive | existing theme tokens | Not used by this phase |

Accent reserved for: existing selected state and focus/hover affordances already used by the language selector/preference components. Do not introduce new palette values.

---

## Copywriting Contract

| Element | Copy |
|---------|------|
| Primary CTA / action | `Select language` |
| Header language trigger sr-only | `Change language` |
| Language label | `Português` |
| Locale code | `pt` |
| Accepted variants | `pt`, `pt-BR`, `pt_BR` normalize to `pt` where normalization exists |
| Empty state heading | Not applicable |
| Empty state body | Not applicable |
| Error state | Use existing save/update failure strings and translate them in PT-BR; no new error UI |
| Destructive confirmation | Not applicable |

All translated strings must preserve placeholders, ICU plural suffixes, markdown/code fragments, URLs, API/model names, and protected project identity strings exactly as required by `AGENTS.md` and Phase 21 context.

---

## Interaction Contract

- Default frontend language switcher must show `Português` through `INTERFACE_LANGUAGE_OPTIONS`.
- Default frontend profile language preferences must consume the same `INTERFACE_LANGUAGE_OPTIONS`; no duplicate PT label list should be invented.
- Default i18next config must register `pt` in `resources` and `supportedLngs`.
- Classic header language selector must add `Português` using the same option pattern as `fr`, `ru`, `ja`, and `vi`.
- Classic profile language preferences must add `Português` using the existing option-list pattern.
- Classic `i18n.js` must register the `pt` resource.
- Selecting `Português` changes the active language immediately and persists through the existing `/api/user/self` preference path.
- Failed preference persistence must keep existing rollback behavior: restore previous language and show existing error feedback.
- Keyboard, focus, hover, selected, and disabled states must remain provided by the existing Base UI/Semi primitives.
- Persisted or requested `pt-BR` and `pt_BR` values must resolve to `pt` where the upstream language normalizer handles locale variants.
- Primary visual anchor remains the existing language selector/menu or profile language select; hierarchy is current trigger/select -> selected `Português` row -> existing save/error feedback.
- No custom PT-only UI flow, banner, settings page, modal, or onboarding affordance is allowed.

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| shadcn official | none | not required |
| `@ai-elements` existing registry in `components.json` | none | not used by this phase |
| third-party additions | none | blocked unless separately vetted |

No registry components are introduced by this phase.

---

## Checker Sign-Off

- [x] Dimension 1 Copywriting: PASS
- [x] Dimension 2 Visuals: PASS
- [x] Dimension 3 Color: PASS
- [x] Dimension 4 Typography: PASS
- [x] Dimension 5 Spacing: PASS
- [x] Dimension 6 Registry Safety: PASS

**Approval:** approved 2026-07-04 after UI checker validation
