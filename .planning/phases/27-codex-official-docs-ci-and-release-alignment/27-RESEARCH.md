# Phase 27 Research - codex-official-docs-ci-and-release-alignment

**Date:** 2026-07-08  
**Status:** Ready for execution

## Official findings

### `codex exec` and non-interactive automation

From the official Codex non-interactive docs:

- `codex exec` is the supported non-interactive CLI path for scripts and CI.
- Default sandbox is read-only; automation should explicitly choose the least privilege needed.
- `CODEX_API_KEY` is supported only in `codex exec`.
- For GitHub Actions, OpenAI recommends `openai/codex-action` instead of manually installing the CLI and passing API keys to shell steps.

### API key auth versus ChatGPT-managed auth

From the official docs:

- API keys are the recommended default for automation.
- ChatGPT-managed auth in CI/CD is an advanced/private-only path.
- `~/.codex/auth.json` must be treated like a password.
- The advanced `auth.json` persistence flow must not be used for public or open-source repositories.
- The supported pattern is to let Codex refresh `auth.json` itself and persist the updated file, not to hand-roll token refresh logic in jobs.

### GitHub Action contract

From the official Codex GitHub Action docs:

- `openai/codex-action@v1` installs the CLI, starts the Responses API proxy when given an API key, and runs `codex exec`.
- Key workflow inputs include `prompt`/`prompt-file`, `codex-args`, `model`, `effort`, `sandbox`, `output-file`, `codex-home`, and `safety-strategy`.
- Official examples show:
  - `actions/checkout@v5`
  - `persist-credentials: false` in PR review examples
  - least privilege
  - `safety-strategy: drop-sudo`
- Security guidance explicitly says:
  - prefer trusted triggers
  - sanitize prompt inputs
  - keep `drop-sudo` unless Windows forces `unsafe`
  - run Codex late in the job

### Docs MCP

From the official MCP/connectors docs:

- OpenAI Docs MCP / official docs are a valid first-party path for current documentation lookup.
- MCP/connectors guidance belongs to the Responses API and current OpenAI docs surface, not to bespoke repo-only notes.

## Repo findings

- `.github/workflows/sync.yml` already uses `openai/codex-action@v1` for failure analysis.
- That step currently encodes reasoning tuning through `codex-args` and `-c model_reasoning_effort=...`, while the official action exposes a first-class `effort` input.
- `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` documents runtime/provider behavior but has no dedicated Phase 27 official CI/auth/release section.
- `docs/CI-RELEASE-WATCHDOG.md` describes GitHub Actions rerun mechanics but not the official Codex automation boundary.
- Release notes PT-BR are already produced in `.github/workflows/release.yml`, which aligns with the fork’s PT-BR requirement.

## Recommended execution

1. Add a dedicated PT-BR doc for official Codex CI/auth/release alignment.
2. Link it from the main operator manual and the CI watchdog doc.
3. Update `sync.yml` to use the official `effort` input directly.
4. Validate:
   - workflow YAML parses
   - docs contain the official guidance and repo-specific mapping
   - Phase 27 planning artifacts show complete execution

## Gain assessment

### High gain

- reduces drift between fork docs and current OpenAI guidance
- removes future temptation to invent unsupported auth-refresh workflows in CI
- leaves a repo-local PT-BR operator bridge to the official docs

### Medium gain

- makes the existing `sync.yml` Codex step easier to compare against upstream docs

### Low gain

- no direct end-user runtime change

## Recommendation

Proceed.

Phase 27 should be completed now because Phase 26 already created the local curated contract, and the remaining gap is documentation and CI alignment rather than product behavior.
