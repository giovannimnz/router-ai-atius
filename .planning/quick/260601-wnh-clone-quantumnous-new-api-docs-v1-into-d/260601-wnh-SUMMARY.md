# Quick Task 260601-wnh: Clone QuantumNous/new-api-docs-v1 into docs/new-api-docs-v1 and add it to .gitignore — SUMMARY

**Status:** Complete
**Commits:**
- `b46642942` — `docs(quick-260601-wnh): Clone ... (PLAN + SUMMARY + STATE)`
- `a53a57941` — `chore(docs): vendor new-api-docs-v1 reference under docs/new-api-docs-v1 (gitignored)`
- `<this-commit>` — `.gitignore` ajuste trailing slash (symlink-safe)

**Date:** 2026-06-02

## What Was Done

### 1. Cloned upstream docs

`https://github.com/QuantumNous/new-api-docs-v1.git` (depth=1) — Next.js documentation site for New API.

### 2. Moved out of worktree per user decision

Initial clone landed in `docs/new-api-docs-v1/` (aninhava `.git/` dentro do worktree, frágil). User requested:

- Real location: `~/GitHub/refs/new-api-docs-v1/`
- Worktree entry: `docs/new-api-docs-v1` → symlink → real path
- Convention: `~/GitHub/refs/` becomes the standard location for upstream reference clones (documented in Obsidian at `90-META/94-Sistema-e-Convencoes/94.01-Convencao-Diretorio-Refs.md`).

### 3. `.gitignore` ajuste: trailing slash

Original pattern `docs/new-api-docs-v1/` (with trailing slash) only matches **directories**. The symlink is technically a file in git's view, so the original pattern didn't ignore it — `git status` showed `?? docs/new-api-docs-v1` as untracked. Without the fix, a careless `git add .` would follow the symlink and stage 138M of content.

Changed to `docs/new-api-docs-v1` (no slash) — matches file OR directory, ignores the symlink as an untracked path. Comments added explaining the convention and the slash gotcha.

## Final Layout

```
~/GitHub/refs/new-api-docs-v1/        # real clone (138M, has its own .git/)
       ▲
       │  symlink
       │
/home/ubuntu/docker/Atius/router-ai-atius/
└── docs/new-api-docs-v1 → ~/GitHub/refs/new-api-docs-v1   # ignored
```

## Verification

- `git check-ignore -v docs/new-api-docs-v1` → matches `.gitignore:51:docs/new-api-docs-v1`, exit 0.
- `git status` → clean, no untracked or modified files.
- `git ls-files --others --exclude-standard docs/` → empty.
- `git add --dry-run docs/new-api-docs-v1` → blocked by gitignore (hint: "Use -f").
- `git add -f --dry-run docs/new-api-docs-v1` → would add the symlink itself (not the contents) — git default behavior.
- `ls docs/new-api-docs-v1/README.md` → resolves via symlink to upstream README.
- `du -sh ~/GitHub/refs/new-api-docs-v1` → 138M (unchanged from initial clone).

## Decisions / Alternatives Considered

- **`~/GitHub/refs/` + symlink (chosen)** vs keep in `docs/`. User-mandated: zero aninhamento de `.git/` dentro do worktree. Symlink pattern also enables a single clone to be referenced by multiple projects if needed later.
- **Symlink vs bind mount vs hard link**. Symlink: portable, works across filesystems, IDE-friendly, no permission surprises. Bind mount: requires root. Hard link: doesn't work across filesystems and is confusing with dirs.
- **`.gitignore` pattern without trailing slash (chosen)** vs keep original `/` suffix. Slash suffix would leak the symlink as untracked — verified via `git check-ignore` exit 1 before the fix.
- **Standalone clone vs git submodule**. User said "nao afete nada" — submodule would force a parent-repo commit pinning an upstream SHA. Standalone clone + gitignore is consistent with the user's intent.

## Follow-ups

- None for the router-ai-atius project.
- Convention `~/GitHub/refs/` is now documented in Obsidian. Apply consistently for future upstream reference clones.
- If the user wants a helper script (e.g. `bin/ref-add <url> <short-name>` to clone + symlink in one go), that's a separate quick task.
