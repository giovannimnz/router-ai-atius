---
phase: 09
plan: 09-01
status: completed
date: 2026-06-07
commits:
  - docs(phase-09): register docs submodule at docs/atius-router-docs
  - docs(phase-09): document docs convergence path in README
  - docs(phase-09): add docs convergence ADR and rollback contract
key-files:
  created:
    - .gitmodules
    - docs/atius-router-docs
    - 21.03-Decisoes-Arquitetura/2026-06-07-docs-convergence-submodule.md
  modified:
    - README.md
---

# Phase 09-01: Docs source topology

## Status

Completed.

## What was built

The docs source is now registered inside the main repo as a submodule at `docs/atius-router-docs/`, pointing to the current docs fork on `main`. The README now explains the canonical init/update command, and an architecture decision record captures the path move, threat model, and rollback safety for the source move.

## Key decisions

- The canonical docs path is `docs/atius-router-docs/` inside `router-ai-atius`.
- The submodule points to `https://github.com/giovannimnz/new-api-docs-v1` on `main`.
- Developers and automation use `git submodule update --init --recursive`.
- The standalone checkout at `/home/ubuntu/docker/Atius/atius-router-docs` remains only as a migration source until the cutover phases finish.

## Verification

- `git submodule status --recursive` shows `docs/atius-router-docs` at `c031fadf28dd4c571ac2cf2743a82e742c32157a` on `heads/main`.
- `README.md` includes the docs convergence note and init command.
- `21.03-Decisoes-Arquitetura/2026-06-07-docs-convergence-submodule.md` exists with threat model and rollback safety.
- No runtime tests were required; this wave was docs/topology only.

## Notes

- Unrelated dirty files already present in `.planning/` were preserved and not touched.
- The submodule registration and docs notes are committed separately so each task stays atomic.
