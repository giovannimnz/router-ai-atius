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
- On merge conflict, `strategy=theirs` means upstream wins and uses
  `git checkout --theirs .`; `strategy=ours` keeps the fork side.
- After any upstream merge, restore protected fork paths before committing the
  fork version bump.

## Local guard

Run:

```bash
scripts/check-upstream-sync-workflow.sh
```

The guard fails if the workflow regresses to fetching upstream tags or if the
merge-strategy mapping is inverted again.
