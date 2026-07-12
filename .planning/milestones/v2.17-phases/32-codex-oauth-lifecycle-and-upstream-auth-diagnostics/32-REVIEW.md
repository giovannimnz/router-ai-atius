---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
reviewed: 2026-07-12T04:44:21Z
depth: deep
files_reviewed: 13
files_reviewed_list:
  - controller/channel.go
  - controller/codex_oauth.go
  - controller/codex_oauth_test.go
  - controller/codex_usage.go
  - dto/channel_settings.go
  - relay/responses_handler.go
  - relay/responses_handler_test.go
  - router/channel-router.go
  - service/codex_catalog.go
  - service/codex_credential_refresh.go
  - service/codex_credential_refresh_test.go
  - service/codex_oauth.go
  - types/error.go
findings:
  critical: 6
  warning: 3
  info: 0
  total: 9
status: resolved
findings_open: 0
---

# Phase 32: Code Review Report

**Reviewed:** 2026-07-12T04:44:21Z
**Depth:** deep
**Files Reviewed:** 13
**Status:** resolved after remediation

## Summary

The backend implementation does not yet satisfy the 32-01 health-persistence and upstream-auth separation contract. Six correctness/security defects block commit. `service/codex_catalog.go` was evaluated only at its staged hunks, as requested.

## Narrative Findings (AI reviewer)

## Critical Issues

### CR-01: Invalidated credentials are still reported as authenticated

**Classification:** BLOCKER
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_credential_refresh.go:97`
**Issue:** `Authenticated` is derived only from access-token/account identity and local expiry. Stored `auth_failed`, `token_invalidated`, or `RequiresRegeneration` health never clears it; malformed/missing expiry also remains authenticated. The test at `service/codex_credential_refresh_test.go:84` locks in this contradictory state. This violates the plan's requirement that future local expiry is never proof of validity.
**Fix:** Derive authenticated health from both local credential fields and persisted probe/upstream-auth state; return false for auth-failed/regeneration-required state and invalid or expired expiry metadata. Update the test to require false for `token_invalidated`.

### CR-02: Catalog refresh retries with the stale access token

**Classification:** BLOCKER
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_catalog.go:337` (staged index hunk; current worktree line 386)
**Issue:** After `RefreshCodexChannelCredential` succeeds, the staged code discards the returned key/channel and calls `doCodexDiscoveryRequest` with the original `channel`, whose `Key` still contains the rejected token. Discovery therefore repeats the same upstream-auth failure.
**Fix:** Use the refreshed channel/key for the retry, or reload the channel after persistence and pass that fresh object.

### CR-03: Usage endpoints expose raw upstream error bodies and fail to classify common 401/403 paths

**Classification:** BLOCKER
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/controller/codex_usage.go:118`
**Issue:** Classification runs only when a refresh token exists and refresh itself fails. A 401/403 with no refresh token, or a second 401/403 after successful refresh, falls through to lines 161-172, where the untrusted upstream body is returned verbatim in `data`. This both loses the Codex upstream-auth code and creates an information-disclosure path.
**Fix:** Classify every final Codex 401/403, persist sanitized health, and return only bounded/sanitized structured fields; never return the raw upstream body on auth failures.

### CR-04: Chat-completions Codex failures remain generic

**Classification:** BLOCKER
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/relay/responses_handler.go:129`
**Issue:** Normalization was added only inside `ResponsesHelper`. The non-stream/stream chat-completions handlers and chat-completions-via-responses path still return `RelayErrorHandler` output directly, so Phase 32's required internal-auth versus upstream-Codex-auth distinction is absent on those public paths.
**Fix:** Apply one Codex-only normalization/health-recording helper at every relay error boundary, including `TextHelper` and `chatCompletionsViaResponses`, while leaving Router middleware authentication errors untouched.

### CR-05: A rotated refresh token can be lost while usage reports success

**Classification:** BLOCKER
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/controller/codex_usage.go:132`
**Issue:** Marshal and database-update errors are ignored, caches are reset, and the request retries with the in-memory refreshed token. If OAuth rotated the refresh token but persistence failed, the endpoint can return success while the database retains an invalidated credential, causing permanent failure on the next request.
**Fix:** Treat marshal/database persistence as mandatory before retrying; return a sanitized failure and do not claim success when the rotated credential was not durably stored.

### CR-06: Core health persistence failures are silently discarded

**Classification:** BLOCKER
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_credential_refresh.go:164`
**Issue:** Failed probes ignore the health-write error and build metadata from the already-mutated in-memory channel, falsely implying persistence. Successful refresh and regeneration similarly ignore `ClearCodexCredentialAuthIssue` failures at `service/codex_credential_refresh.go:381` and `controller/codex_oauth.go:277`, allowing stale `requires_regeneration` state after a successful credential replacement.
**Fix:** Propagate health-write/clear errors (or return an explicit partial-success state), reload persisted metadata before responding, and add failure-injection tests for database write errors.

## Warnings

### WR-01: Relay/catalog errors overwrite explicit probe history

**Classification:** WARNING
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_credential_refresh.go:215`
**Issue:** `RecordCodexCredentialIssue` stamps `LastProbeAt` and `LastProbeStatus=auth_failed` for relay, usage, refresh, and catalog events. Metadata can therefore claim that an explicit probe occurred when it did not, contrary to the endpoint contract. It also replaces the complete prior health object.
**Fix:** Store upstream-auth observation fields separately from probe fields and merge into existing health instead of replacing it.

### WR-02: Sanitized upstream descriptions are parsed but never propagated

**Classification:** WARNING
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/service/codex_oauth.go:351`
**Issue:** `ErrorDescription` is sanitized and stored, but `Error()`, classification, persisted health, and normalized responses discard it. This misses the plan's operator-diagnostic requirement to preserve sanitized upstream code and description.
**Fix:** Carry a bounded sanitized description through the issue/metadata response without including raw bodies or token material.

### WR-03: Tests validate helpers, not the required route matrix, and assert the false-health behavior

**Classification:** WARNING
**File:** `/home/ubuntu/GitHub/containers/router-ai-atius/relay/responses_handler_test.go:16`
**Issue:** The relay test calls the normalizer directly rather than `ResponsesHelper`; there is no coverage for Codex-only branching, health persistence, non-Codex preservation, chat-completions paths, usage/catalog propagation, or Router API-key separation. The metadata test explicitly expects `Authenticated=true` during invalidation.
**Fix:** Add hermetic handler-level tests for all required relay/usage/catalog paths and reverse the invalidated-health expectation.

---

## Remediation Outcome

All six blockers and three warnings were addressed during the execution loop.
The final focused Go suite passed for `controller`, `service`, `relay` and
`relay/channel/codex`; `32-VERIFICATION.md` records the route/runtime evidence.
No finding from this review remains open.

_Reviewed: 2026-07-12T04:44:21Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: deep_
