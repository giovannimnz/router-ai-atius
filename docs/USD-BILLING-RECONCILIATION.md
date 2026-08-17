# USD Billing Reconciliation

## Contract

Dollar-cost prices are USD per one million tokens. The canonical quota formula is:

```text
quota = round((input_tokens * input_usd_per_1m
             + output_tokens * output_usd_per_1m)
             / 1_000_000
             * quota_per_unit
             * group_ratio)
```

Production uses `quota_per_unit=500000`, `quota_display_type=USD`, and
`usd_exchange_rate=1`. Therefore every displayed `$1.00` is exactly USD 1.00.

The one-time marker records the first closed hourly boundary after the last
corrected request. Dashboard money before that boundary comes from reconciled
consume logs; money at and after the boundary continues to come from the durable
`quota_data` aggregate. This bounded split avoids rewriting incomplete historical
aggregates while ensuring a future log-write failure cannot remove new billed
usage from the dashboard. The flow/topology endpoint remains entirely on
`quota_data` because logs do not retain `node_name`.

## One-Time V1 Repair

`ATIUS_DOLLAR_COST_RECONCILIATION_V1_ENABLED=true` enables the one-time repair.
It is disabled by default and requires the main and log tables to be co-located.

The migration:

- fixes only logs whose frozen billing metadata proves the missing per-million divisor;
- never reprices with current model prices;
- excludes tiered billing and web, file, audio, or image surcharges;
- applies wallet counter changes only for corrected logs with `billing_source=wallet`
  and positive user, token, and channel identifiers;
- changes counters by the proven delta instead of replacing absolute totals;
- aborts on subscription or any other unsupported source and leaves
  missing-source counters unchanged;
- fixes a bounded `max_log_id` watermark and records an idempotency marker;
- records an audited closed-hour cutoff and does not rebuild or delete
  `quota_data`.

Production rollout must set both audited expectations:

```text
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS=<exact count>
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_DELTA=<exact quota delta>
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_LOGS=<exact count>
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_DELTA=<exact quota delta>
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_LOGS=<exact count>
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_DELTA=<exact quota delta>
ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_CUTOFF=<exact epoch-hour boundary>
```

Any count, delta, malformed metadata, underflow, overflow, or entity mismatch
rolls back the whole transaction and prevents startup.

## Controlled Rollout

1. Drain every writer that can use the database, including shadow deployments.
2. Take and verify a PostgreSQL backup before changing counters or logs.
3. Reconcile known asynchronous counter drift against wallet logs under a
   separate audited transaction. Preserve `quota + used_quota` and
   `remain_quota + used_quota` invariants.
4. Clear the dedicated persistent Redis database while writers are stopped so
   no stale user or token balance survives the database transaction.
5. Start one master with the enable flag and exact expected values.
6. Verify the marker, watermark, corrected log count, total delta, user/token/
   channel invariants, and absence of still-inflated target logs.
7. Start normal traffic, remove the one-time enable flag, and validate a new
   request against the canonical formula.
8. Confirm `/api/data/self`, `/api/data/`, and the overview dashboard totals
   match the bounded sum of historical consume logs plus post-cutoff
   `quota_data`.

Rollback restores the verified database backup and the previous application
image while all writers remain stopped. Do not attempt a partial inverse update.

## Reranker Alias

The canonical local reranker is `reranker-gte-v1`. Startup migration also
normalizes the legacy alias in logs and `quota_data`. Performance queries map
legacy and canonical rows to `reranker-gte-v1` and aggregate overlapping buckets,
so the dashboard exposes one model without a concurrent destructive data merge.
