# Phase 31 Context

## Objective

Resolve the current `.planning/` health debt without losing useful history.

## Current Health Warnings To Address

- `W002`: `STATE.md` references legacy phase 20 semantics not declared in the
  current roadmap parser output
- `W005`: legacy directories outside `NN-name` convention
- `W006`: roadmap references to phases `5`, `6`, and `8` with no directory on disk
- `W019`: `.planning/FORK_MIGRATION.md` as non-canonical root artifact
- `I001`: multiple legacy `PLAN.md` files without summaries

## Constraints

- preserve historical context that still matters
- do not rewrite recent completed phases 21-28
- prefer archiving/moving over destructive deletion

## Success Definition

`validate.health` should improve materially, ideally to `healthy`, or to a
smaller, well-understood set of warnings that are explicitly justified.
