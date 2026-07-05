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
as missing scripts, missing registry credentials, frozen lockfiles, missing
modules, or frontend build/config errors.

Docker image workflows run `scripts/check-dockerfile-assets.sh` before buildx.
That guard catches stale Dockerfile references and verifies that the Dockerfile
uses the workspace root lockfile `web/bun.lock` before the expensive multi-arch
build starts. It also pins the Docker frontend stages to the same Bun version
used by release workflows so `--frozen-lockfile` behaves consistently.

If a run was created from an old tag or commit with a broken workflow file,
rerunning it will keep using that old workflow. In that case, push a corrected
commit and run `workflow_dispatch`, create a new patch tag, or intentionally
move the tag only after confirming that rewriting the published tag is allowed.

When `sync.yml` creates a tag with `GITHUB_TOKEN`, do not rely on `push` tag
events to start release builds. GitHub suppresses most workflow runs caused by
`GITHUB_TOKEN` pushes. The sync workflow should dispatch release, Docker, and
Electron workflows directly after the tag is pushed.

The sync workflow also runs `scripts/ci-build-frontends.sh "v$NEW_TAG"` before
pushing the sync commit and version tag. If `web/default` or `web/classic` is
broken by an upstream merge, the sync run fails before a release tag exists.
