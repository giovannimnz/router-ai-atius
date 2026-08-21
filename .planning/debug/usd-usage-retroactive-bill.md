---
status: investigating
trigger: "Full. Fale em pt-BR. Quanto o valor do uso, seja nas ultimas 24h, e anterior, etc... Em $ que o valor monetário deve ser de paridade 1:1 com o USD, como ficou quanto a tu ajustar, e também gostaria que ao entender a lógica e realizar os ajustes devidos baseados nos valores dos modelos e etc, ajustasse o consumo e afins, tudo retroativo também, consulte skills tbm se tem quanto isso para ajustar 100% até bater 100% passado e valide ajuste para consistir corretamente quanto futuro também."
created: 2026-08-17T16:31:00-03:00
updated: 2026-08-17T17:18:00-03:00
---

# USD usage retroactive billing

## Symptoms

- expected: Every monetary usage value is real USD at 1:1 parity; last 24 hours, prior periods, channel counters, logs, and future settlements reconcile exactly from model prices.
- actual: Current correctness is unproven. Prior evidence showed channel 11 post-cutover quota, removed-channel historical counters, excluded test quota, and a reranker fallback ratio of 37.5 while catalog pricing advertises 75 USD per million input tokens.
- errors: No explicit runtime error. Suspected semantic/accounting inconsistency between pricing configuration, log quota, aggregate counters, and dashboard period semantics.
- timeline: Historical usage predates consolidation into channel 11; reranker ID and OpenRouter pricing were changed on 2026-08-17.
- reproduction: Compare production options, model pricing, quota-per-unit, usage logs, channel/user/token counters, and dashboard/API totals for rolling 24h and earlier periods.

## Current Focus

- hypothesis: Confirmed. Legacy ratio-mode quota accounting is being read as if it were canonical USD, while historical dashboards/counters are split across non-authoritative derived tables and post-cutover channel IDs.
- test: Read-only SQL against production DBRouterAiAtius plus code inspection of quota/model pricing conversion paths.
- expecting: Root cause and safe fix direction only; no production mutation in this pass.
- next_action: Stop after recording root cause and proposed safe fix.
- reasoning_checkpoint: Historical truth must be reconstructed from immutable logs plus dated pricing rules, not from `used_quota`, current channel counters, or partially populated `quota_data`.
- tdd_checkpoint: false

## Evidence

- Code: legacy quota unit is not USD. `common/constants.go` sets `QuotaPerUnit = 500 * 1000.0`, with comment `$0.002 / 1K tokens`.
- Code: ratio-mode consume billing writes internal quota units directly. `service/text_quota.go` ratio branch computes `summary.Quota = round((prompt/completion adjusted tokens) * modelRatio * groupRatio)`.
- Code: public catalog pricing converts ratio-mode to USD/M by doubling the ratio. `service/modelcatalog/catalog.go` `PublicTokenPrices()` uses `inputPrice := pricing.ModelRatio * 2`.
- Code: billing usage endpoint is lifetime-based, not period-based. `controller/billing.go:GetUsage()` reads `token.UsedQuota` or `user.UsedQuota` and converts it to display currency; it does not accept or apply any start/end timestamps.
- Code: channel/user counters are derived side effects, not a ledger. `model/user.go` and `model/channel.go` only do `used_quota = used_quota + quota`.
- Production option state: `options` has `ModelRatio` and `USDExchangeRate=1`; there is no `ModelBillingExpr`/tiered USD contract configured for the affected models.
- Production option state: `ModelRatio` explicitly contains `embedding-gte-v1 = 0.035`; `reranker-gte-multilingual-v1` and `reranker-gte-v1` have no explicit entry.
- Code: missing model ratios fall back to `37.5` in self-use mode. `setting/ratio_setting/model_ratio.go:GetModelRatio()` returns `37.5` when no explicit ratio exists and self-use mode is enabled.
- Production consume-log evidence for reranker fallback:
  - Sample logs on 2026-08-14 and 2026-08-16 for `reranker-gte-multilingual-v1` show `quota/prompt_tokens ~= 37.5` (examples: `18 -> 675`, `28 -> 1050`, `31 -> 1163`, `39 -> 1463`).
  - This matches the fallback `modelRatio=37.5` exactly after rounding.
  - Because public catalog code multiplies `modelRatio * 2`, the same internal ratio is advertised as `75 USD / 1M tokens`. The system is therefore mixing two representations of the same model price: internal ratio units in logs/counters and derived USD/M in catalog/public pricing.
- Production channel cutover evidence:
  - Embedding history exists on channel `9` from `2026-07-03 23:14` through `2026-08-17 00:58`, total `137848` quota across `3006` consume logs.
  - Reranker history exists on channel `10` from `2026-08-14 11:35` through `2026-08-16 22:32`, total `11150` quota across `19` consume logs.
  - Channel `11` begins only at `2026-08-17 01:23`, with `9566` total consume-log quota through `2026-08-17 16:28`.
  - Therefore channel `11` cannot represent full historical embedding/reranker lifetime usage.
- Production counter drift evidence:
  - `channels.id=11` currently has `used_quota=9555`.
  - The sum of channel-11 consume logs is `9566`.
  - Even post-cutover, `channels.used_quota` is not an exact ledger (`delta = -11`).
- Production aggregate-table incompleteness:
  - `quota_data.channel_id=10` matches logs exactly (`11150`).
  - `quota_data.channel_id=11` matches logs exactly (`9566`).
  - `quota_data.channel_id=9` does not match logs: `14941` in `quota_data` vs `137848` in consume logs.
  - `quota_data` for channel `9` begins only at `2026-07-12 03:00`, while consume logs begin at `2026-07-03 23:14`.
  - Therefore period dashboards driven by `quota_data` are incomplete for older embedding history unless explicitly backfilled.
- Production test-log evidence:
  - `reranker-gte-multilingual-v1` has `14` consume logs with `content='模型测试'`, all zero-token, contributing `14` quota.
  - `embedding-gte-v1` has `13` consume logs with `content='模型测试'`, contributing `198` quota.
  - Any retroactive rebill that uses consume logs must explicitly filter or classify test traffic; `type=2` alone is insufficient.

## Eliminated

- Hypothesis: channel `11` alone can be used as the authoritative lifetime source for Atius local embeddings usage.
  - Rejected by direct log evidence: historical usage exists on channels `9` and `10` before `2026-08-17 01:23`.
- Hypothesis: `quota_data` is a complete historical source for prior-period embedding usage.
  - Rejected by direct aggregate mismatch on channel `9` (`14941` vs `137848`) and later first-seen timestamp.
- Hypothesis: reranker historical logs already reflect canonical 1:1 USD/M pricing.
  - Rejected by inferred per-token quota ratio (`~37.5`) matching the legacy fallback rather than a canonical USD ledger.

## Resolution

- root_cause: |
    The affected usage surfaces are built on a legacy quota-unit accounting model, not on a canonical USD ledger.

    Internally, ratio-mode consume billing stores `quota` in "quota units" using `modelRatio`, where `QuotaPerUnit=500000` defines the conversion to display USD. Public model pricing then converts ratio-mode to USD/M by multiplying `modelRatio * 2`. For the reranker models, missing explicit ratios triggered the `37.5` self-use fallback, and consume logs were written at that internal ratio while public/catalog pricing represented the same pricing as `75 USD / 1M`. This is a representation mismatch, not a single authoritative USD ledger.

    On top of that, historical aggregates are split across non-authoritative derived surfaces:
    - `users.used_quota` / `tokens.used_quota` / `channels.used_quota` are increment-only side counters;
    - channel history was consolidated from channels `9` and `10` into channel `11` on 2026-08-17;
    - `quota_data` is incomplete for older channel-9 embedding history;
    - `/billing/usage` is lifetime-based and ignores period windows.

    Because of those combined factors, the current system cannot truthfully answer "last 24h / previous period / retroactive USD 1:1" from existing counters alone, and any retroactive rewrite based on them would be incorrect.
- fix: |
    Safe fix direction:

    1. Establish one canonical billing contract per affected model in real USD terms.
       - Move `embedding-gte-v1`, `reranker-gte-multilingual-v1`, and `reranker-gte-v1` off ambiguous legacy ratio-only behavior.
       - Prefer explicit `tiered_expr` or explicit dollar-cost pricing so stored pricing matches published USD/M directly.
       - Remove reliance on the `37.5` fallback for rerankers.

    2. Rebuild historical truth from immutable consume logs, not from counters.
       - Use `logs` as the source ledger.
       - Apply a dated pricing map by model and pricing epoch (at minimum pre/post 2026-08-17 cutover and any reranker rename/pricing-change timestamp used operationally).
       - Explicitly classify and exclude/segregate test traffic (`content='模型测试'`, zero-token test rows, and any other agreed synthetic probes).
       - Compute both corrected USD and corrected quota units; preserve original quota as audit data.

    3. Write corrected history to a new idempotent ledger/backfill target before touching existing counters.
       - Example: a backfill table keyed by log id with fields such as `original_quota`, `corrected_quota`, `corrected_usd`, `pricing_version`, `is_test_traffic`, `backfill_run_id`.
       - Do not overwrite `logs.quota` or current counters until reconciliation reports pass.

    4. Recompute derived aggregates from the reconstructed ledger.
       - Backfill `quota_data` completely for historical windows.
       - Rebuild `channels.used_quota`, `users.used_quota`, and `tokens.used_quota` from authoritative ledger totals if the product still needs lifetime counters.
       - Split lifetime and period-based views explicitly; do not reuse lifetime `used_quota` for 24h/prior-period usage endpoints.

    5. Correct the period API semantics.
       - Keep lifetime usage endpoints clearly labeled lifetime.
       - For 24h / prior-period / dashboard comparisons, query the rebuilt ledger or fully backfilled `quota_data` with timestamps.

    6. Run the backfill as a reversible migration with dry-run reconciliation.
       - Report deltas by model, channel, day, and test/non-test class.
       - Require a zero-surprise review before any production write.
- verification: |
    Root cause confirmed by:
    - code-path inspection of ratio-mode vs USD/M conversion;
    - production consume-log samples inferring `37.5` reranker ratio;
    - production channel-history split across `9`, `10`, `11`;
    - production mismatch between `channels.used_quota`, `logs`, and `quota_data`;
    - production discovery that `/billing/usage` uses lifetime `used_quota` only.

    No production mutation performed in this pass.
- files_changed:
  - .planning/debug/usd-usage-retroactive-bill.md
