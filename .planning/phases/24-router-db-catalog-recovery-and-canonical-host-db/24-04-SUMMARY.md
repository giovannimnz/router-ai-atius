---
phase: 24-router-db-catalog-recovery-and-canonical-host-db
plan: "04"
subsystem: runtime-cutover
tags:
  - recovery
  - postgres
  - pgbouncer
  - podman
  - validation
status: complete
---

# Phase 24 Plan 04 Summary

Cutover applied on `2026-07-04` from host PgBouncer `newapi` to
`DBRouterAiAtius`, followed by final cleanup removing the legacy `newapi`
mapping from PgBouncer after validation.

## Runtime changes applied

- Added PgBouncer mapping:
  - `DBRouterAiAtius = host=127.0.0.1 port=8745 dbname=DBRouterAiAtius`
- Removed legacy PgBouncer mapping:
  - `newapi = host=127.0.0.1 port=8745 dbname=newapi`
- Updated `/home/ubuntu/.config/systemd/user/container-router-ai-atius.service`
  to use:
  - `SQL_DSN=postgresql://admin:${POSTGRES_PASSWORD}@10.1.1.1:6432/DBRouterAiAtius`
- Reloaded `pgbouncer.service`
- Reloaded user systemd and restarted `container-router-ai-atius.service`

## Candidate DB build and reconciliation

- Created fresh custom dump from live `newapi`
- Restored it into host DB `DBRouterAiAtius`
- Applied transformed catalog restore for:
  - `OpenAI - Codex`
  - `DeepSeek`
  - `embedding-gte-v1`
- Preserved `newapi` only as offline legacy DB / backup reference
- Adjusted DB ownership/ACL so the app user `admin` can migrate/use
  `DBRouterAiAtius`

## Validation

- `systemctl --user status container-router-ai-atius.service` -> active after
  ACL fix
- authenticated `GET /v1/models` -> `200`
  - present: `gpt-5.5`, `gpt-5.4`, `gpt-5.4-mini`,
    `gpt-5.3-codex-spark`, `deepseek-v4-flash`, `deepseek-v4-pro`,
    `embedding-gte-v1`
  - absent: `gpt-5.5-1m`, `gpt-5.4-1m`
- authenticated `POST /v1/chat/completions` with `gpt-5.4` -> `200` after
  reloading channel 5 from `~/.codex/auth.json`
- authenticated `POST /v1/chat/completions` with `deepseek-v4-flash` and
  `deepseek-v4-pro` -> `200` after replacing the active DeepSeek key
- authenticated `POST /v1/embeddings` with `embedding-gte-v1` -> `200`,
  `768` dimensions
- authenticated `GET /v1/models` after disabling channels/abilities MiniMax ->
  no `MiniMax-*` models returned
- authenticated `POST /v1/chat/completions` with `MiniMax-M3` -> not usable
  (`model_not_found`)

## Residual notes

- The governed public embedding alias remains `embedding-gte-v1`
- Codex/OpenAI embedding routes remain subject to upstream quota/licensing for
  `text-embedding-3-*`; they are intentionally not part of the active public
  exposure

## Rollback

Rollback target is the timestamped Phase 24 backups:

- `/etc/pgbouncer/pgbouncer.ini.phase24-*.bak`
- `/home/ubuntu/.config/systemd/user/container-router-ai-atius.service.phase24-*.bak`

Rollback sequence:

```bash
sudo cp /etc/pgbouncer/pgbouncer.ini.phase24-*.bak /etc/pgbouncer/pgbouncer.ini
cp /home/ubuntu/.config/systemd/user/container-router-ai-atius.service.phase24-*.bak \
  /home/ubuntu/.config/systemd/user/container-router-ai-atius.service
sudo systemctl reload pgbouncer
systemctl --user daemon-reload
systemctl --user restart container-router-ai-atius.service
```
