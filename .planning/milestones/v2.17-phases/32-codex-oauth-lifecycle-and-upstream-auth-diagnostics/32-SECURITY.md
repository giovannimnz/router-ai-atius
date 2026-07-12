---
phase: 32
slug: codex-oauth-lifecycle-and-upstream-auth-diagnostics
status: verified
threats_open: 0
asvs_level: 1
block_on: high
register_authored_at_plan_time: true
created: 2026-07-12
---

# Phase 32 - Security

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Admin browser -> Router admin API | Authenticated operator invokes metadata, probe, refresh and PKCE regeneration | callback input, sanitized metadata, action result |
| Router -> OpenAI auth/Codex upstream | Router exchanges/refreshes OAuth and probes account-scoped models | OAuth token material and upstream status |
| Router -> PostgreSQL | Channel key and non-secret health are persisted separately | secret credential JSON; sanitized health metadata |
| Router relay -> API client | Internal token auth precedes Codex relay and upstream error normalization | request/response data and sanitized error code |
| Frontend build -> browser | Type 57 UI must not bundle or render secret values | static assets and sanitized DTO types |
| Fork sync -> fork-owned paths | Upstream merge can overwrite Codex customizations | source, tests and operational docs |

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Evidence | Status |
|-----------|----------|-----------|----------|-------------|---------------------|--------|
| T-32-01-S1 | Spoofing | Codex admin routes | high | mitigate | `AdminAuth`, channel permissions/type validation and session-bound PKCE state/verifier | closed |
| T-32-01-T1 | Tampering | channel health writes | high | mitigate | `dto.ChannelSettings` merge and regression coverage preserve unrelated settings | closed |
| T-32-01-R1 | Repudiation | lifecycle failures | medium | mitigate | sanitized status/code/reason persisted in credential health and surfaced in metadata | closed |
| T-32-01-I1 | Information Disclosure | DTO/logs/tests | high | mitigate | metadata omits token fields; bounded/sanitized errors; secret-free tests/docs | closed |
| T-32-01-D1 | Denial of Service | probe/discovery | medium | mitigate | explicit probe, context timeout, no probe on metadata read, mocked negative tests | closed |
| T-32-01-E1 | Elevation of Privilege | refresh/regenerate | high | mitigate | sensitive-write/operate/read permissions and channel type validation before writes | closed |
| T-32-01-SC | Tampering | dependencies | low | accept | no dependency change; Bun hydration used existing frozen lockfile | closed |
| T-32-02-S1 | Spoofing | regeneration modal | high | mitigate | saved channel context, HTTPS authorize URL and backend PKCE state validation | closed |
| T-32-02-T1 | Tampering | type 57 rendering | medium | mitigate | `isCodexChannelType` gates generic fields; Base URL warning boundary is unit-tested | closed |
| T-32-02-R1 | Repudiation | UI lifecycle actions | medium | mitigate | UI uses backend success/error and reloads sanitized metadata; no invented local success | closed |
| T-32-02-I1 | Information Disclosure | DOM/clipboard/storage | high | mitigate | no token DTO fields, reveal/copy controls or durable callback storage; 4/4 markup tests | closed |
| T-32-02-D1 | Denial of Service | UI actions | medium | mitigate | explicit actions with disabled/loading state and no automatic probe on drawer load | closed |
| T-32-02-E1 | Elevation of Privilege | sensitive UI actions | high | mitigate | actions disabled without permission or saved channel; backend enforces permissions | closed |
| T-32-02-SC | Tampering | package installs | high | mitigate | no package/lock change; existing React/Bun/node:test stack only | closed |

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-32-01 | T-32-01-SC | Existing locked dependencies remain a supply-chain dependency; Phase 32 introduced no package version or new dependency. | project policy | 2026-07-12 |

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-12 | 14 | 14 | 0 | Codex GSD L1 audit |

## Evidence

- `router/channel-router.go` permission matrix;
- `controller/codex_oauth.go` PKCE session/state and sanitized completion;
- `service/codex_credential_refresh.go` metadata/health/probe/refresh;
- `relay/codex_auth_error.go` Codex-only upstream auth normalization;
- `web/default/src/features/channels/components/codex/` sanitized panel/dialog;
- `codex-credential-panel.test.tsx` type 57 boundary and secret-free markup;
- `32-VALIDATION.md` green Go/frontend/live matrix;
- fork-sync guard commit `9dd574597`.

## Sign-Off

- [x] All threats have a disposition.
- [x] Accepted risk is documented.
- [x] `threats_open: 0` confirmed.
- [x] `status: verified` set in frontmatter.

**Approval:** verified 2026-07-12
