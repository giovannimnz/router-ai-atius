# CI Upstream Sync

The scheduled `Sync Upstream + Release` workflow keeps the fork aligned with
`QuantumNous/new-api` and then publishes a fork version tag when `main` changes.

## 2026-07-05 failure

Run `28732178229` failed in `Configure upstream remote` because the workflow
used `git fetch upstream --tags`. The upstream tag `v0.12.15` also exists in
this fork, but points to a different object, so Git rejected the fetch with:

```text
! [rejected] v0.12.15 -> v0.12.15 (would clobber existing tag)
```

Fork release tags and upstream release tags are separate namespaces in practice,
even when the tag names overlap. The sync workflow must not import upstream tags
into local `refs/tags/*`.

## Contract

- Fetch upstream branches with `--no-tags`.
- Detect the latest upstream version with `git ls-remote --tags --refs`.
- Keep fork-owned release tags untouched.
- On merge conflict, `strategy=theirs` means upstream wins. The workflow resolves
  each unmerged path individually, because modify/delete conflicts may not have a
  file on the selected side.
- If a failed merge leaves `.git/index.lock` behind in the ephemeral Actions
  checkout, the workflow removes that stale lock before resolving paths.
- The resolver collects the unmerged path list before checkout/add/rm operations,
  so a diagnostic `git diff` process cannot race with index writes.
- Protected paths are removed from the index/worktree and restored from the
  fork baseline before the merge commit is completed. This avoids pushing an
  intermediate commit or newly-added upstream workflow file that GitHub rejects.
- With `strategy=theirs`, `web/default` is treated as upstream-owned and is
  restored wholesale from `upstream/main` after a merge. This prevents hidden
  non-conflicting fork leftovers from compiling against newer upstream
  frontend contracts.
- After any upstream merge, restore protected fork paths before committing the
  fork version bump.
- `.github/workflows/` is protected as fork-owned. The scheduled workflow runs
  with `GITHUB_TOKEN`, which cannot create or update workflow files from an
  upstream merge unless it has the `workflow` permission.
- A tag pushed by `GITHUB_TOKEN` does not trigger `push`-based workflows. After
  creating the fork tag, the sync workflow dispatches `release.yml`,
  `docker-build.yml`, and `electron-build.yml` explicitly with
  `workflow_dispatch` and `--repo "$GITHUB_REPOSITORY"` so `gh` cannot infer the
  upstream repository after a merge.
- The legacy GHCR workflow uses `workflow_run` against the actual workflow name
  `Sync Upstream + Release` and falls back to `github.token` when `GHCR_TOKEN`
  is not configured.
- Before pushing the sync commit or version tag, the workflow runs
  `scripts/ci-build-frontends.sh "v$NEW_TAG"` after installing Bun `1.3.14`.
  A broken frontend sync now stops inside the sync workflow instead of
  publishing a tag that immediately fails Release, Docker, and Electron.
- Before pushing the sync commit or version tag, the workflow also runs
  `scripts/ci-build-backend.sh "v$NEW_TAG"`. Backend compile errors introduced
  by conflict resolution now fail in the sync workflow before any release tag is
  published.
- The fork keeps `common.RelayIdleConnTimeout` because upstream protected-fetch
  clients and HTTP transports now use that setting during backend compile.
- Fork patch suffixes are calculated by `scripts/next-fork-version.sh`. For the
  same upstream base, `1.0.0-rc.16.5` becomes `1.0.0-rc.16.6`; when the
  upstream base changes, the suffix resets to `.1`.
- GitHub Actions should use `actions/checkout@v5` so scheduled sync runs do not
  emit Node.js 20 deprecation warnings from older checkout actions.

## Local guard

Run:

```bash
scripts/check-upstream-sync-workflow.sh
```

The guard fails if the workflow regresses to fetching upstream tags, loses the
post-tag dispatch calls, points the GHCR workflow at the wrong sync workflow
name, omits the Bun setup/pre-tag frontend/backend builds, omits upstream-owned
`web/default` restoration, resets same-base fork suffixes to `.1`, or inverts
the merge-strategy mapping again.
