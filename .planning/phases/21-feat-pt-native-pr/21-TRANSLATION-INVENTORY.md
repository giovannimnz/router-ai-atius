---
phase: 21-feat-pt-native-pr
artifact: translation-inventory
date: "2026-07-04"
status: local-execution-complete
---

# Phase 21 Translation Inventory

This artifact exists to preserve Giovanni's existing PT-BR translation work while preventing old fork/runtime changes from entering a clean upstream PR.

## Baseline

- Upstream repo: `QuantumNous/new-api`
- Current validated upstream baseline: `5fc35e28a253bd5a5656c177aea1bd121231398f`
- Implementation worktree: `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`
- Implementation branch: `feat/phase21-pt-native-upstream`
- Current upstream key counts: backend `en.yaml` has 231 message IDs; default frontend `en.json` has 4978 `translation` keys; classic frontend `en.json` has 3831 `translation` keys.
- Final locale code: `pt`
- Accepted variants: `pt`, `pt-BR`, `pt_BR` normalized to `pt`

## Codex Execution Lane

- `implementation_worktree`: `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`
- `implementation_branch`: `feat/phase21-pt-native-upstream`
- `upstream_main_commit`: `5fc35e28a253bd5a5656c177aea1bd121231398f`
- `created_from`: `upstream/main`
- `preflight_status`: `clean on 2026-07-05 after fast-forward to upstream/main; implementation completed locally with validation evidence recorded in 21-0x-SUMMARY.md`

Plans `21-02`, `21-03`, `21-04`, and implementation-diff checks in `21-05` must run from `implementation_worktree`. The dirty planning checkout remains the place for GSD summaries, reviews, Graphify output, and local handoff artifacts.

## Reuse Order

1. Backend: use `feat/pt-native-i18n-clean:i18n/locales/pt.yaml` or `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean/i18n/locales/pt.yaml` as the primary source. Execution on 2026-07-05 reused the clean source, then machine-translated only the 3 new upstream backend keys introduced after the old baseline, keeping placeholder parity.
2. Default frontend: use the linked branch `giovannimnz/router-ai-atius:feat/pt-native-i18n-clean` as the primary source for every current upstream key it covers. Execution on 2026-07-05 filled 4489 keys from that linked branch.
3. Default frontend gap supplement: use `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean/web/default/src/i18n/locales/pt.json` for current keys missing from the linked branch. Execution on 2026-07-05 filled 86 keys from the external clean worktree and translated the remaining 403 upstream-current gaps/unsafe carryovers with placeholder-protected machine translation plus same-as-English review.
4. Classic frontend: no complete classic `pt.json` source was found. Execution on 2026-07-05 reused 718 default-PT values where the classic English source text matched a default English string exactly, then translated the remaining 3113 classic values with placeholder-protected machine translation plus same-as-English review.
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

## Execution Outcome

- Backend implementation completed in `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream` with native `pt` wiring, `i18n/locales/pt.yaml`, and behavior/parity tests.
- Default frontend implementation completed with `pt.json`, `config.ts`, `languages.ts`, and updated `sync-i18n.mjs` reporting `pt.missingCount=0`, `pt.extrasCount=0`, `pt.untranslatedCount=0`.
- Classic frontend implementation completed with `pt.json`, `i18n.js`, `language.js`, and both existing language controls exposing `Português`.
- Reviewed same-as-English allowlists live in `.planning/phases/21-feat-pt-native-pr/21-SAME-AS-ENGLISH-LITERALS.json`.

## Remaining Open Checks

- Re-check PR #5801 immediately before any upstream PR submission; as of 2026-07-05 it is still open and still changes only `i18n/pt.yaml`.
- Re-check issue #2924 and related PRs #5238/#5245 immediately before submission; as of 2026-07-05 issue #2924 is open and PRs #5238/#5245 are closed historical context only.
- Decide whether to fix or waive inherited frontend gates before submission: `web/default lint`, `web/classic i18n:lint`, and `web/classic lint` currently fail on pre-existing unrelated repo issues documented in summaries.
