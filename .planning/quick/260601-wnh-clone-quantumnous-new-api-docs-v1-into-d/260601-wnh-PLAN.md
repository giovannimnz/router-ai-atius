# Quick Task 260601-wnh: Clone QuantumNous/new-api-docs-v1 into docs/new-api-docs-v1 and add it to .gitignore

**Mode:** quick
**Status:** Ready for execution

## Task

Clone the upstream documentation repository `QuantumNous/new-api-docs-v1` into a subdirectory of the existing `docs/` folder and ensure the cloned content is excluded from version control.

## Rationale

The existing `docs/` directory contains project-internal documentation (ARCHITECTURE.md, CONFIGURATION.md, etc.) which must not be touched. The new upstream docs are a separate, self-contained git repository with its own version history. Placing it under `docs/new-api-docs-v1/` keeps the project tidy while signaling the source explicitly.

## Plan

### Task 1: Clone repo + gitignore + commit

**Files:**
- `docs/new-api-docs-v1/` (created via `git clone`)
- `.gitignore` (appended `docs/new-api-docs-v1/`)

**Action:**
1. `git clone https://github.com/QuantumNous/new-api-docs-v1.git docs/new-api-docs-v1`
2. Verify the clone produced expected content (`README.md` should exist at the root).
3. Append `docs/new-api-docs-v1/` to `.gitignore` (append only — do not touch the existing upstream/HEAD merge markers or any other line).
4. `git status` to confirm:
   - `docs/new-api-docs-v1/` is NOT staged (gitignore working).
   - `.gitignore` IS staged.
5. Atomic commit: `chore(docs): vendor new-api-docs-v1 reference under docs/new-api-docs-v1 (gitignored)`.

**Verify:**
- `git check-ignore docs/new-api-docs-v1/README.md` returns 0 (path is ignored).
- `git diff --cached --name-only` includes only `.gitignore`.
- `git log -1 --oneline` shows the new commit.
- `ls docs/new-api-docs-v1/README.md` exists locally.

**Done when:** upstream docs are present locally under `docs/new-api-docs-v1/`, the path is gitignored, and the gitignore change is committed atomically.

## Decisions / Alternatives Considered

- **Cloning inside `docs/` (chosen)** vs cloning at the repo root as a sibling folder. The user asked for it inside `docs/`; sibling placement was rejected.
- **Subdirectory `docs/new-api-docs-v1/` (chosen)** vs cloning directly into `docs/` (would overwrite existing project docs). The existing `docs/` contains `ARCHITECTURE.md`, `CONFIGURATION.md`, `openapi.json`, etc. Cloning directly would clobber them. Subdirectory placement is the only safe option.
- **`.gitignore` append (chosen)** vs full rewrite. Append preserves the in-progress `<<<<<<< HEAD / ======= / >>>>>>> upstream/main` merge markers and any other entries without risk.
- **No submodule (chosen)** vs `git submodule add`. Submodule would force a commit pointing at an upstream commit SHA and would be tracked in the parent repo — the user explicitly said "nao afete nada" and "inclua-a no .gitignore", which is incompatible with submodule tracking. Plain clone + gitignore matches the request verbatim.
