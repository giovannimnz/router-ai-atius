---
phase: 32
slug: codex-oauth-lifecycle-and-upstream-auth-diagnostics
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-12
audited: 2026-07-12T13:58:00-03:00
---

# Phase 32 - Validation Strategy

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go test; Bun node:test + React server rendering; live HTTP smokes |
| **Config file** | `go.mod`, `web/default/package.json`, `web/default/tsconfig.app.json` |
| **Quick run command** | `./scripts/podman-admin.sh profile-run -- bun test web/default/src/features/channels/components/codex/codex-credential-panel.test.tsx` |
| **Full suite command** | `GOCACHE` isolado + `go test ./controller ./service ./relay ./relay/channel/codex -count=1`; frontend test/typecheck/build pelo wrapper |
| **Estimated runtime** | ~20 min com cache Go frio e teto de 20% CPU |

## Sampling Rate

- Depois de mudanca backend Codex: suite Go focada com cache isolado.
- Depois de mudanca UI type 57: teste do painel/boundary + typecheck.
- Antes de transicao: build frontend, health, probe, refresh e smokes live.
- Nenhum comando usa watch mode.

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command / Evidence | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|------------------------------|-------------|--------|
| 32-01-01 | 01 | 1 | PHASE-32-CODEX-OAUTH-REGENERATE | PKCE/session/token disclosure | State/verifier e metadata sanitizada; tokens nao retornam | unit + live | `controller/codex_oauth_test.go`; regenerate/probe/refresh live | yes | green |
| 32-01-02 | 01 | 1 | PHASE-32-CODEX-CREDENTIAL-HEALTH | stale/future expiration | Probe e upstream auth participam de `authenticated`/regeneration | unit + live | `service/codex_credential_refresh_test.go`; probe/refresh live | yes | green |
| 32-01-03 | 01 | 1 | PHASE-32-CODEX-UPSTREAM-AUTH-ERRORS | auth confusion/raw body | Router auth e Codex upstream auth permanecem distintos e sanitizados | unit | `relay/responses_handler_test.go`; suite controller/service/relay | yes | green |
| 32-02-01 | 02 | 2 | PHASE-32-CODEX-UI-SINGLE-ENDPOINT | token/base URL exposure | Type 57 remove campos/efeitos genericos, usa locale PT-BR real, erro inline e popup seguro | unit + typecheck + build | `codex-credential-panel.test.tsx` 5/5; typecheck/build PASS | yes | green |
| 32-03-01 | 03 | 3 | PHASE-32-FORK-SYNC-GUARD | upstream overwrite | Paths e contrato Codex protegidos no fork-sync | static/integration | checker e dry-run; `omni-srv-admin` `9dd574597` | yes | green |
| 32-04-01 | 04 | 4 | PHASE-32-VALIDATION-DOCS-SHIP | secret leakage/false closure | Backup, docs sem segredo, smokes e Git remoto comprovados | live/integration | `32-VERIFICATION.md`; `origin/main=c3a45e917` | yes | green |

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. No new test dependency
or stub is required.

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Login/consentimento no browser para callback OAuth | PHASE-32-CODEX-OAUTH-REGENERATE | OpenAI login, consent and PKCE callback require a human authenticated browser | Use `Regenerar credencial`, complete consent, then validate sanitized metadata/probe/refresh |
| Negative upstream auth in production | PHASE-32-CODEX-UPSTREAM-AUTH-ERRORS | Deliberately invalidating the newly regenerated live credential is unsafe | Use deterministic `httptest` coverage; exercise live only during a real incident or isolated disposable channel |

## Validation Audit 2026-07-12

| Metric | Count |
|--------|-------|
| Requirements mapped | 6 |
| Automated coverage green | 6 |
| Missing automated references | 0 |
| Manual safety checks | 2 |

## Validation Sign-Off

- [x] All tasks have automated verification or existing infrastructure.
- [x] Sampling continuity has no three consecutive tasks without automation.
- [x] Wave 0 has no missing references.
- [x] No watch-mode flags.
- [x] CPU-heavy commands ran through the 20% wrapper.
- [x] `nyquist_compliant: true` set in frontmatter.

**Approval:** approved 2026-07-12
