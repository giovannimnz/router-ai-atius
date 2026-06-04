# Phase 7: feat-pt-native-branch - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-04
**Phase:** 07-feat-pt-native-branch
**Areas discussed:** Working Tree Strategy, Conflict Resolution Strategy, Coverage Validation, Commit Strategy

---

## Working Tree Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Stash + pop | `git stash push -u` pre-flight, `git stash pop` post-validation | ✓ |
| Ignore | Leave `podman-compose.yml` modified + `integration/docs/` untracked, work around them | |
| Treat | Fold the dirty work into the new branch (mixes fork-specific with PT-only) | |

**User's choice:** [timeout — best-judgment default: Stash + pop]
**Notes:** Stash is the safe option — preserves fork work, isolates Phase 7 scope. If stash conflicts, abort and ask.

## Conflict Resolution Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Hunk-by-hunk patch | `git show main:file` for sources, `patch` tool for i18n.go with 4 known hunks | ✓ |
| Full file replacement | `cp` for all 5 files, manual re-edit if needed | |
| 3-way merge | Use `git merge-file` with base/upstream/ours | |

**User's choice:** [timeout — best-judgment default: Hunk-by-hunk patch]
**Notes:** For i18n.go, patch is the cleanest — 4 hunks are known and specified in PLAN.md Task 04. For locale files, copy is fine (no upstream version exists, so no merge needed).

## Coverage Validation

| Option | Description | Selected |
|--------|-------------|----------|
| jq + python yaml | `jq '.translation \| keys \| length'` + `yaml.safe_load` count | ✓ |
| bun i18n:sync | Use upstream's i18n tooling (run `bun run i18n:sync`) | |
| Manual check | eyeball the JSON files | |

**User's choice:** [timeout — best-judgment default: jq + python yaml]
**Notes:** `i18n:sync` is a fork-specific script, not present in upstream, not trustworthy for cross-fork validation. jq + yaml is universal.

## Commit Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| 1 squash | Single commit, 5 files, atomic change | ✓ |
| 5 separate commits | One commit per file, more auditable history | |
| 2 commits (Go + frontend) | i18n.go + pt.yaml as "backend", pt.json + config.ts + languages.ts as "frontend" | |

**User's choice:** [timeout — best-judgment default: 1 squash]
**Notes:** Phase 7 does NOT commit. Phase 8 does (commit + push + PR). Squash is the cleanest PR shape for upstream maintainers.

---

## Claude's Discretion

- Commit message body (recommended: minimal, no body, just `feat: add Portuguese (pt) language`)
- Order of hunk application in i18n.go (recommended: bottom-up to avoid offset shifts)
- Whether to delete the `feat/portuguese-translation-clean` branch from the fork before opening the new PR (default: keep, document in PR body)

## Deferred Ideas

- PR #5245 closure message tone/content — Phase 8, not Phase 7
- Branch disposal strategy post-merge — Phase 8
- Auto-update CLAUDE.md / agent context with `feat/pt-native` branch info — future, not in scope
