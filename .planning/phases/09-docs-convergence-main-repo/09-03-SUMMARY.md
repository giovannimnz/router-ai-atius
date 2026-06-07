---
phase: 09
plan: 09-03
status: completed
date: 2026-06-07
commits:
  - docs(atius-router-docs): phase 09 convergence — submodule target, systemd deploy, remote governance
key-files:
  created:
    - modules/fork-sync/projects/atius-router-docs/remote-governance.md
  modified:
    - modules/fork-sync/projects/atius-router-docs/scripts/fork-sync-docs.sh
    - modules/fork-sync/projects/atius-router-docs/sync.yaml
    - modules/fork-sync/manuals/atius-router-docs.md
---

# Phase 09-03: Ownership transfer to omni-srv-admin + remote governance

## Status

Completed.

## What was built

Transferred the operational ownership of the integrated docs to `omni-srv-admin` and closed the governance decision on the standalone remote.

### Changes in omni-srv-admin

**fork-sync-docs.sh** — Major rewrite:
- Default REPO_PATH changed from standalone `/home/ubuntu/GitHub/forks/AtiusRouterDocs` to submodule path in `router-ai-atius/docs/atius-router-docs/`
- Build step: `podman build` → `bun install && bun run build`
- Deploy step: `podman rm -f / podman-compose up -d` → `systemctl --user restart atius-router-docs.service`
- Healthcheck: `curl https://router.atius.com.br/en/` → `curl http://127.0.0.1:3003/pt/`
- Added submodule reference bump step after sync
- Added submodule init step for first-run scenarios

**sync.yaml** — Added:
- `deploy_type: systemd-user`
- `service_name: atius-router-docs.service`
- `service_port: 3003`
- Comment block documenting Phase 09 structural change

**atius-router-docs.md (manual)** — Expanded with:
- Phase 09 structural change overview with ADR link
- Bootstrap instructions for the submodule
- Build/deploy/rollback commands for systemd runtime
- Troubleshooting section for common failures
- Cache-bust procedure for assets
- Smoke checks pre-deploy checklist
- Remote governance decision section
- Version bumped to 2

**remote-governance.md** — New file documenting:
- Decision to keep the fork remote as submodule origin
- Standalone checkout kept as transitório mirror until end of v2.15
- Removal criteria (2 consecutive sync cycles, production validation, zero functional references)
- Rollback path

## Verification

- All omni-srv-admin changes committed: `a0e5bd5`
- fork-sync-docs.sh syntax checked (bash set -euo pipefail)
- sync.yaml validated as valid YAML
- Manual covers bootstrap, build, deploy, rollback, troubleshooting

## Notes

- The submodule reference in router-ai-atius already pointed to the correct commit with deploy helpers; no additional submodule bump was needed.
- Unrelated dirty files in `.planning/` were preserved.
