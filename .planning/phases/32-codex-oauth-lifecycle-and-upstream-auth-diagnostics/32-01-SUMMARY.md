---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
plan: 32-01
status: complete
completed: 2026-07-12
---

# 32-01 Summary - Backend OAuth e diagnosticos

O backend do channel type 57 agora expoe metadata sanitizada, refresh, probe e regeneracao OAuth separada. Falhas `token_invalidated`, `refresh_token_invalidated`, `invalid_api_key` e 401/403 upstream recebem classificacao Codex propria, sem confusao com API key interna do Router e sem vazar tokens.

O relay e o catalogo tambem foram alinhados ao contrato upstream atual: `stream=true`, input em lista, SSE sem `Content-Type`, buffering para clientes non-stream e versionamento do contrato de validacao.

Validacao: `go test ./controller ./service ./relay ./relay/channel/codex -count=1` passou sob `cpus=0.8`.
