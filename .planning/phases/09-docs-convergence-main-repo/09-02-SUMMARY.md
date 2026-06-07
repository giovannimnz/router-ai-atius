---
phase: 09
plan: 09-02
status: completed
date: 2026-06-07
commits:
  - docs(phase-09): add docs runtime deploy helper
  - docs(phase-09): bump docs submodule runtime helpers
key-files:
  created:
    - docs/atius-router-docs/deploy/docs-runtime.md
    - docs/atius-router-docs/scripts/restart-atius-router-docs.sh
  modified:
    - docs/atius-router-docs
---

# Phase 09-02: Runtime repoint to `docs/atius-router-docs/`

## Status

Completed.

## What was built

The docs runtime was repointed from the legacy standalone checkout to the integrated submodule path `docs/atius-router-docs/`. The host user service now serves from the new tree, Apache comments were aligned to the integrated docs path, and the docs submodule gained deployment/runtime helpers for repeatable restart and production startup.

## Changes applied

### Host runtime

- `/home/ubuntu/.config/systemd/user/atius-router-docs.service`
  - `WorkingDirectory` now points to `/home/ubuntu/docker/Atius/router-ai-atius/docs/atius-router-docs`
  - runtime command now uses Bun on port `3003`
- `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf`
  - docs comments now reference `docs/atius-router-docs`
  - no behavioral routing change was needed because the proxy already targets `127.0.0.1:3003`

### Docs submodule helpers

- `docs/atius-router-docs/deploy/docs-runtime.md`
  - documents the canonical runtime, restart commands, and build notes
- `docs/atius-router-docs/scripts/restart-atius-router-docs.sh`
  - one-command `systemctl --user` restart/status helper

## Verification

- `systemd-analyze verify /home/ubuntu/.config/systemd/user/atius-router-docs.service` exited `0`
- `apache2ctl configtest` returned `Syntax OK`
- `systemctl --user restart atius-router-docs.service` brought the service back up as `active (running)`
- `curl -I http://127.0.0.1:3003/pt/` returned the expected `308` redirect for the slashless locale route
- `curl -I http://127.0.0.1:3003/en/` and `/pt/docs/skills/` behaved consistently with the docs runtime
- `bun install` and `bun run build` succeeded in the docs submodule before the restart

## Notes

- The docs submodule now has local-only runtime artifacts (`node_modules`, `.next`) required for the production start command; they were not committed.
- Unrelated dirty files in `.planning/` were preserved and left untouched.
