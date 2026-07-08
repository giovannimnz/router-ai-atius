---
phase: 28-branch-hygiene-and-mainline-reconciliation
plan: "03"
type: summary
status: complete
completed_at: "2026-07-08T17:59:56-03:00"
requirements:
  - PHASE-28-MAINLINE-RECONCILIATION
---

# 28-03 Summary - selective mainline reconciliation

## Result

Created and validated the clean reconciliation branch
`reconcile/v2.14-mainline-clean` from `origin/main`, then fast-forwarded
`origin/main` from `d743533c9` to `3c55d7b11`.

The reconciliation deliberately avoided a wholesale merge of `feat/pt-native`.
The PT implementation lane remains preserved separately on
`origin/feat/phase21-pt-native-upstream`.

## Included

- Phase 24 and 25 recovery/governor code, docs, scripts, and planning.
- Phase 26 dynamic Codex catalog reconciliation.
- Phase 27 Codex CI/auth/release docs and workflow alignment.
- Phase 28 branch hygiene planning and docs.
- Phase 21 handoff docs only, not PT implementation files.
- Phase 22/23 planning moved to future v2.15 context.

## Excluded

- `feat/pt-native` as a merge source.
- PT-BR implementation files for the upstream handoff.
- Backups, pycache, node_modules, frontend dist, and runtime caches.
- Broad deletions inherited from stale local branches.

## Validation

- `go test ./controller ./service ./service/openaicompat ./model -count=1 -timeout 600s -vet=off`
- `go test . -run "^$" -count=1 -timeout 600s -vet=off`
- `go test . ./controller ./dto ./model ./relay ./service ./service/embeddinggovernor ./service/modelcatalog ./service/openaicompat -count=1 -timeout 600s -vet=off`
  - Root package initially failed until frontend assets were built.
  - All listed non-root packages passed.
- `scripts/ci-build-frontends.sh v2.14-reconcile-validation`
- `python3 -m py_compile tools/clianything.py scripts/smoke-provider-consolidation.py scripts/smoke-embeddings.py scripts/test_long_context_aliases_static_test.py`
- `bash -n scripts/test-long-context-aliases.sh scripts/smoke-docs-links.sh`
- `python3 -m unittest discover -s tests -p 'test_clianything*.py'`
- `python3 scripts/test_long_context_aliases_static_test.py`
- `scripts/smoke-docs-links.sh`
- workflow YAML parse for `.github/workflows/sync.yml` and `.github/workflows/release.yml`
- `node "$HOME/.codex/gsd-core/bin/gsd-tools.cjs" graphify status`
  - Fresh before the reconciliation push: `commit_stale=false`.

## Remote State

- `origin/main` fast-forwarded to `3c55d7b11`.
- Wave 4 can now reset/recreate the local main worktree and remove stale local
  branches/worktrees.
