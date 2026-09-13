---
status: resolved
trigger: "Ainda ocorrendo: Ocorreu um erro de solicitacao: Unsupported parameter: top_p"
created: 2026-09-01
updated: 2026-09-01
---

# Codex top_p persists

## Symptoms

- Expected: ChatGPT Codex requests succeed through API and Playground without
  unsupported sampling parameters.
- Actual: `Unsupported parameter: top_p` still appears after prior fix was
  committed and opened as PR.
- Error: `Unsupported parameter: top_p`.
- Timeline: persists after the user retried the affected models.
- Reproduction: select a ChatGPT Codex model through API or Playground while
  sampling controls remain enabled.

## Current Focus

- hypothesis: the normal Codex adaptor path is corrected, but a configured
  pass-through bypass can still send the original client JSON including top_p.
- test: compare the running image to PR #11 and trace all Chat/Responses paths.
- expecting: determine whether an older image or an unguarded relay path explains
  the upstream error.
- next_action: if the error recurs, inspect the active SQL_DSN-backed settings for
  global/channel pass-through and capture the request route and request ID.

## Evidence

- timestamp: 2026-09-01T18:34:43-03:00
  source: git
  finding: PR #11 head is `76ef4eba151d22d6491bf93c3fa3e15c664c5c1b`
  (`fix(codex): omit unsupported sampling params`); it is not an ancestor of
  checkout `main`/`origin/main` at `80213e10d`.
- timestamp: 2026-09-01T18:34:43-03:00
  source: runtime container inspection
  finding: the active `router-ai-atius` container is image
  `localhost/router-ai-atius:main-76ef4eba1`, image ID
  `4a55f91cd1ce3032febd525688b786b20e9d0a9e32c87cede044c91dbc9f9701`,
  started 2026-09-01 15:34:53 -03. The PR commit is deployed to this runtime.
- timestamp: 2026-09-01T18:34:43-03:00
  source: relay/channel/codex/adaptor.go at deployed commit
  finding: the PR adds `request.TopP = nil` after normal Responses request
  normalization, alongside the existing removals of `MaxOutputTokens` and
  `Temperature`.
- timestamp: 2026-09-01T18:34:43-03:00
  source: relay paths
  finding: `/v1/chat/completions` for Codex deterministically enters
  `chatCompletionsViaResponses` when pass-through is disabled; conversion copies
  client `top_p` into a Responses DTO, then the deployed Codex adaptor clears it.
  Direct `/v1/responses` follows the same adaptor and is also covered.
- timestamp: 2026-09-01T18:34:43-03:00
  source: relay/compatible_handler.go and relay/responses_handler.go
  finding: both Chat Completions and Responses use the raw original body when
  `PassThroughRequestEnabled` or `ChannelSetting.PassThroughBodyEnabled` is true.
  This bypasses `ConvertOpenAIResponsesRequest`, so `top_p` can reach the Codex
  upstream despite PR #11.
- timestamp: 2026-09-01T18:34:43-03:00
  source: dto/openai_responses_compaction_request.go and responses_handler.go
  finding: `/v1/responses/compact` is not a top_p carrier: its public DTO has no
  TopP field and the handler reconstructs a Responses DTO without TopP before the
  adaptor's compact early return.
- timestamp: 2026-09-01T18:34:43-03:00
  source: public runtime probes
  finding: `https://router.atius.com.br` returns `x-new-api-version:
  1.0.0-rc.16.8.1`; `/api/user/models` requires authentication, so supported
  parameter metadata could not be verified unauthenticated. The application's
  SQL_DSN points somewhere other than the local inspected PostgreSQL data store,
  so the active pass-through flags remain unverified in this read-only session.

## Eliminated

- The active local production container is not an image predating PR #11.
- The normal Codex Chat-to-Responses and direct Responses conversion paths do not
  forward TopP in the deployed PR image.
- Responses compaction cannot preserve a user-supplied top_p through its parsed
  public request DTO.

## Resolution

- root_cause: The reproduced live errors at 15:29 and 15:30 came from the
  pre-fix image `fix-embedding-halfopen-20260828`. Those Chat requests use the
  normal Chat-to-Responses path, which only forwards top_p when the old adaptor
  is running. PR #11 had been built but not deployed.
- fix: PR #11 was merged to main and its production-built image was deployed as
  `localhost/router-ai-atius:main-76ef4eba1` at 15:34:53 -03.
- verification: local and public `/api/status` returned HTTP 200 after restart;
  active image ID is `4a55f91cd1ce`; runtime and container cgroup limits pass;
  no new top_p relay error appeared after cutover. No authenticated synthetic
  request was sent because no user token was read or reused.
- files_changed: relay/channel/codex/adaptor.go, controller/user.go,
  web/default/src/features/playground/, and this debug record.

## Residual Risk

- The global/per-channel pass-through branches still bypass Codex request
  normalization for direct API requests. This was not the cause of the logged
  Playground Chat failure, but should receive a separate hardening patch after
  its active settings are verified.
