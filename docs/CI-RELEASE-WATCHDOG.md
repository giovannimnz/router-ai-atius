# CI release watchdog

The release and Electron workflows build both frontend variants through
`scripts/ci-build-frontends.sh`. The script runs `bun install --frozen-lockfile`,
builds `web/default` and `web/classic`, mirrors `web/default/dist` to `web/dist`
for Electron compatibility, and fails if the expected `index.html` artifacts are
missing.

Use the watchdog when a GitHub Actions run should be followed until success:

```bash
scripts/gh-actions-watchdog.sh --run-id 28721172212 --repo giovannimnz/router-ai-atius
```

Use the release checker for the whole tag:

```bash
scripts/check-release-actions.sh v0.12.15.2 --repo giovannimnz/router-ai-atius
```

For a tag workflow, the latest run for the tag can be resolved automatically:

```bash
scripts/gh-actions-watchdog.sh \
  --repo giovannimnz/router-ai-atius \
  --tag v0.12.15.2 \
  --workflow "Build Electron App"
```

The watchdog prints the failed jobs and failed steps, then runs
`gh run rerun --failed` until the configured attempt limit is reached. Override
the defaults with `--max-attempts`, `--poll-seconds`, `GH_WATCHDOG_MAX_ATTEMPTS`,
or `GH_WATCHDOG_POLL_SECONDS`.

The release checker scans all runs for the tag, calls the watchdog for retryable
failures, and stops with the failed log when it sees deterministic failures such
as missing scripts or missing registry credentials.

If a run was created from an old tag or commit with a broken workflow file,
rerunning it will keep using that old workflow. In that case, push a corrected
commit and run `workflow_dispatch`, create a new patch tag, or intentionally
move the tag only after confirming that rewriting the published tag is allowed.
