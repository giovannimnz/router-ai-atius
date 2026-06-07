---
phase: "09"
phase_name: docs-convergence-main-repo
status: passed
date: 2026-06-07
must_have_count: 14
must_have_passed: 14
must_have_failed: 0
requirements_mapped: [DOCS-01, DOCS-02, DOCS-03]
---

# Phase 09: Docs Convergence Main Repo — Verification

## Verdict

**✅ PASSED** — All 14 must-haves verified. The docs source topology, runtime repoint, and omni-srv-admin ownership transfer are complete and functional.

## Must-Have Results

| Plan | Check | Status |
|------|-------|--------|
| 09-01 | .gitmodules exists with submodule entry | ✅ |
| 09-01 | docs/atius-router-docs is a valid submodule (via .git/modules) | ✅ |
| 09-01 | ADR exists with rollback and threat model | ✅ |
| 09-01 | README has docs convergence note | ✅ |
| 09-02 | systemd unit WorkingDirectory points to submodule path | ✅ |
| 09-02 | systemd unit uses bun run start (production mode) | ✅ |
| 09-02 | Apache comments reference docs/atius-router-docs | ✅ |
| 09-02 | Apache config syntax valid | ✅ |
| 09-02 | Runtime active and serving | ✅ |
| 09-03 | fork-sync-docs.sh targets submodule path | ✅ |
| 09-03 | fork-sync-docs.sh uses systemctl restart (not podman) | ✅ |
| 09-03 | sync.yaml has deploy_type: systemd-user | ✅ |
| 09-03 | Manual version bumped to 2 | ✅ |
| 09-03 | Remote governance decision documented | ✅ |

## Requirement Coverage

| REQ-ID | Description | Status |
|--------|-------------|--------|
| DOCS-01 | Docs source integrado ao repo principal como submodule | ✅ |
| DOCS-02 | Cutover runtime/deploy sem repo standalone (Apache + systemd) | ✅ |
| DOCS-03 | Gestão via omni-srv-admin + destino do remote separado | ✅ |

## Notes

- One must-have check showed `❌` for `docs/atius-router-docs/.git` because submodules store their git metadata in the parent `.git/modules/` directory. Verified manually: the submodule resolves HEAD correctly.
- No code changes were made to Go or frontend files. This phase was purely about docs topology, runtime configuration, and operational ownership.
