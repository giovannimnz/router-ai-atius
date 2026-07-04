---
phase: 21-feat-pt-native-pr
artifact: translation-inventory
date: "2026-07-04"
status: draft-from-readonly-audit
---

# Phase 21 Translation Inventory

This artifact exists to preserve Giovanni's existing PT-BR translation work while preventing old fork/runtime changes from entering a clean upstream PR.

## Baseline

- Upstream repo: `QuantumNous/new-api`
- Current validated upstream baseline: `1ae757475f9e8dad4ffedf89b3e707756fe8ecf9`
- Current upstream key counts: backend `en.yaml` has 228 message IDs; default frontend `en.json` has 4978 `translation` keys; classic frontend `en.json` has 3831 `translation` keys.
- Final locale code: `pt`
- Accepted variants: `pt`, `pt-BR`, `pt_BR` normalized to `pt`

## Reuse Order

1. Backend: use `feat/pt-native-i18n-clean:i18n/locales/pt.yaml` or `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean/i18n/locales/pt.yaml` as the primary source. Read-only audit found 228/228 backend keys aligned with current English and no placeholder mismatch.
2. Default frontend: use the linked branch `giovannimnz/router-ai-atius:feat/pt-native-i18n-clean` (`cd8cb89bb72b1f5551a9f7536f104498ddfb4d75`) as the primary source for every current upstream key it covers. Recheck against current `upstream/main` found 4655/4978 current English keys covered, 323 missing, 20 extras, 0 placeholder mismatches, and 228 same-as-English values to classify.
3. Default frontend gap supplement: use `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean/web/default/src/i18n/locales/pt.json` for the 323 current keys missing from the linked branch. The 4655 overlapping current keys are identical between the linked branch and the external worktree, and the external worktree covers 4978/4978 current upstream keys with 11 extras to remove.
4. Classic frontend: no complete classic `pt.json` source was found. Reuse default frontend PT strings only when the English source text is identical. Read-only audit found only 28 obvious exact text matches; the remaining classic-specific keys need translation/review.
5. Historical `pt`/`pt-BR` artifacts: use commits `728bb2e2`, `3f9209e0`, and `05accaf9` only as fallback wording references. They are stale against current upstream key sets.
6. Glossary/style: use `5f0453fb:docs/TRANSLATION-PT-BR.md` only as a wording/glossary reference, not as a parity source.

## Linked Branch Coverage Evidence

The linked branch contains substantial default-frontend screen/menu coverage and must not be treated as a minor fallback source:

| Area keyword | Matching PT keys in linked branch |
|---|---:|
| Dashboard | 23 |
| Settings | 86 |
| Channel | 167 |
| User | 182 |
| Model | 407 |
| Token | 153 |
| Redemption | 45 |
| Pricing | 45 |
| Logs | 24 |
| System | 55 |
| API | 177 |
| Provider | 60 |
| Notification | 6 |
| Task | 11 |
| Menu | 4 |

The final default `pt.json` should therefore be built as an overlay:

1. start from current `upstream/main:web/default/src/i18n/locales/en.json` key order;
2. fill matching keys from the linked branch first;
3. fill only the 323 linked-branch gaps from `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean`;
4. remove extras from both sources;
5. classify same-as-English values before claiming complete translation coverage.

## Source Classification

| Source | Useful files | Fit | Classification |
|---|---|---|---|
| Current worktree | no active PT locale files | base only | Read current EN bases only. |
| Linked branch `giovannimnz/router-ai-atius:feat/pt-native-i18n-clean` / `FETCH_HEAD` | `i18n/locales/pt.yaml`, `web/default/src/i18n/locales/pt.json` | backend 228/228; default 4655/4978 current keys, 323 missing, 20 extras, 0 placeholder mismatch | Primary backend source and primary default source for all covered keys. |
| `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean` | `i18n/locales/pt.yaml`, `web/default/src/i18n/locales/pt.json` | backend 228/228; default 4978/4978 with 11 extras; overlapping default values identical to linked branch | Gap supplement for default source after removing extras. |
| `728bb2e2` | historical `pt.json`, `pt.yaml` | stale; default has hundreds of missing keys | Fallback wording only. |
| `3f9209e0`, `05accaf9` | historical `pt-BR.json`, `pt-BR.yaml` | stale; many extras/missing keys | Fallback wording only. |
| `5f0453fb` | `docs/TRANSLATION-PT-BR.md` | glossary/style guide | Wording reference only. |

## Unsafe Reuse Rules

- Do not cherry-pick whole old branches or PR commits.
- Do not copy root `i18n/pt.yaml`; backend upstream-native path is `i18n/locales/pt.yaml`.
- Do not copy `.planning/`, Graphify, Obsidian, Podman, runtime DB, provider/router/governor, or deployment docs into the upstream implementation branch.
- Do not reuse fork/brand strings such as `Atius`, `router-ai-atius`, `/home/ubuntu`, or runtime host paths.
- Protected identity: preserve upstream project and organization names as literal identity text when the source says `New API`, `new-api`, or `QuantumNous`; do not translate those identities to `Nova API` or rename them.
- Review `same_as_key` values before declaring 100% coverage. Some are legitimate brand/code literals, but English UI sentences must not remain as fallback.
- Placeholder mismatches are blockers. Preserve `{{...}}`, `{{.Max}}`, markdown/code fragments, URLs, API/model names, and JSON examples exactly.

## Branch Contract

The final upstream PR branch may contain only native language files, native wiring, and narrowly justified validation tests/checks. Planning artifacts stay outside the final upstream PR branch:

- `.planning/phases/21-feat-pt-native-pr/21-TRANSLATION-INVENTORY.md`
- `.planning/phases/21-feat-pt-native-pr/*-SUMMARY.md`
- `.planning/phases/21-feat-pt-native-pr/21-UPSTREAM-HANDOFF.md`
- Graphify data
- Obsidian notes
- temporary PR body/comment files

## Open Checks For Execution

- Re-run key counts against the clean implementation worktree after `upstream/main` is fetched.
- Re-run default/frontend counts against the execution lane because old local worktree counts can differ from `upstream/main`; the authoritative counts above are from `upstream/main` on 2026-07-04.
- Re-fetch the linked branch before execution: `git fetch origin feat/pt-native-i18n-clean --prune` and record `FETCH_HEAD`.
- Re-check PR #5801 before creating a PR because its scope may change.
- Re-check issue #2924 and related PRs #5238/#5245 for upstream context.
