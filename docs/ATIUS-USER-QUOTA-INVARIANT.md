# Atius user quota invariant

## Contract

A valid personal token is never rejected by this router because user, wallet,
account, or subscription balance is exhausted or negative. Subscription
absence or exhaustion always falls through to wallet accounting, including
`subscription_only` and subscriptions with `allow_wallet_overflow=false`.

Only these limits may reject a request:

- an explicit per-token quota;
- an upstream provider quota, rate limit, entitlement, or billing response.

Wallet and subscription values remain accounting and notification data. They
must not be used as request-admission decisions.

## Versioned guard

Run the audit before every backend/image build and after every upstream sync:

```bash
scripts/atius-user-quota-guard.sh
scripts/fork-sync-guard.sh user-quota-audit
```

The default mode is `audit`. An explicit, idempotent repair is available for a
clean compatible baseline:

```bash
scripts/atius-user-quota-guard.sh repair
```

Repair uses `patches/atius-user-quota-unlimited.patch`, creates a temporary
backup first, applies only when `git apply --check` succeeds, and audits again.
Source drift fails closed and requires a manual semantic port.

## Verification

1. Run the guard audit.
2. Run focused billing regression tests through `scripts/podman-admin.sh
   profile-run` with one Go build job.
3. Build through `scripts/podman-admin.sh build`; direct Podman/Go builds are
   not permitted on this host.
4. Before a production restart, create a rollback image tag and record the
   active image ID/digest without exposing credentials.
5. After restart, verify the selected image ID and repeat the original
   authenticated request against both `127.0.0.1:3000` and
   `https://router.atius.com.br`.
6. Confirm the request reaches upstream or succeeds; a local
   `insufficient_user_quota` or `quota_not_enough` response is a failed gate.

This document does not authorize a deploy or restart. Those actions require a
separate operational approval and rollback/readback procedure.
