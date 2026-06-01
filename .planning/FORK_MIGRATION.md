# FORK_MIGRATION.md — router-ai-atius

**Fork of:** [QuantumNous/new-api](https://github.com/QuantumNous/new-api.git)
**Fork URL:** https://github.com/giovannimnz/router-ai-atius.git
**Created:** 2026-04-21

## Why This Fork

This fork adapts the NewAPI gateway for the Atius domain infrastructure. It provides:
- AI/LLM routing via NewAPI (calciumion/new-api Docker image)
- PostgreSQL-backed channel and quota management
- Custom middleware for model metadata enrichment
- CLI tools for agent-native management

## Local Modifications

### Protected Files (Never Overwritten)

These files are protected during upstream sync and must be manually maintained:

| File/Directory | Purpose | Rationale |
|----------------|---------|-----------|
| `integration/middleware/model_detailed.py` | Model metadata enrichment middleware | Custom logic for Atius model catalog |
| `.planning/` | Planning docs and phase tracking | Atius-specific roadmap and GSD workflow |
| `FORK_MIGRATION.md` | This file | Documents fork identity |

### Restored Files (Re-applied After Merge)

| File | Purpose | Rationale |
|------|---------|-----------|
| `docker-compose.yml` | Container orchestration | Atius-specific container config, ports, volumes |

## Versioning

Fork uses semantic versioning with fork suffix: `X.Y.Z.N`

- **X.Y.Z**: Upstream NewAPI base version (from git tags)
- **N**: Fork-specific suffix (incremented on each sync when base unchanged)

Example: `0.5.2.1` means NewAPI base `0.5.2` + first fork build

Version is stored in `VERSION` file and tagged as `vX.Y.Z.N`.

## Sync Workflow

### Manual Sync

```bash
./scripts/sync-fork.sh                    # Full sync with upstream
./scripts/sync-fork.sh --strategy ours    # Prefer local changes on conflict
./scripts/sync-fork.sh --dry-run          # Preview without changes
```

### GitHub Actions (Automated)

- **Weekly Sync**: `.github/workflows/sync.yml` — runs every Monday 3:00 UTC
- **Release**: `.github/workflows/release.yml` — triggers on `v*` tags

## Upstream Tracking

- **Upstream:** `https://github.com/QuantumNous/new-api.git`
- **Remote name:** `upstream`
- **Sync strategy:** `-X theirs` (prefer upstream on conflict)

## Sync Algorithm

```
1. git remote add upstream https://github.com/QuantumNous/new-api.git
2. git fetch upstream
3. git checkout main
4. git pull origin main --rebase
5. git merge upstream/main -X theirs --no-edit
6. git checkout HEAD -- <protected files>
7. git checkout HEAD -- docker-compose.yml
8. ./scripts/version-bump.sh
9. git push origin main
```

## Restore Commands

If protected files get overwritten during merge:

```bash
# Restore model_detailed.py
git checkout HEAD -- integration/middleware/model_detailed.py

# Restore planning
git checkout HEAD -- .planning/

# Restore this file
git checkout HEAD -- FORK_MIGRATION.md
```

## Troubleshooting

### Merge conflicts

```bash
git status  # See conflicting files
git add <resolved files>
git commit
```

### Protected files overwritten

Run restore commands above, then:
```bash
git add -A
git commit -m "chore: restore fork overrides after upstream merge"
git push origin main
```

### Version conflicts

Check `VERSION` file and verify against upstream tag:
```bash
git tag -l "v*" | sort -V | tail -5
git show upstream/main:VERSION
```

## Notes

- This fork intentionally omits NewAPI web UI, payment, and other non-infrastructure features
- The local version is minimalist — just the proxy + middleware + PostgreSQL backend
- GitHub CLI (`gh`) not yet authenticated — PR creation requires manual auth
