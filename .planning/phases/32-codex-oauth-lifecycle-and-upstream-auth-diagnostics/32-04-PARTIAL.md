---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
plan: 32-04
status: blocked
updated: 2026-07-12
---

# 32-04 Partial - Validacao live e blocker externo

## Concluido

- Backup SQL do channel 5 antes das escritas live.
- Build final `ec0f29ea91546d4bfa70b3e71aba8c01eace165ac0aaddf75b91c186d3c3123b` com `cpu_quota=80000`, `cpuset=0`, `GOMAXPROCS=1` e `GOFLAGS=-p=1`.
- Deploy live e runtime com tres containers nao-infra limitados a `0.800 CPU`.
- Catalogo: 8 candidatos, 6 promovidos; `gpt-5.6-sol`, `gpt-5.6-terra` e `gpt-5.5` ativos; Luna rejeitada por 404; `codex-auto-review` negado.
- `/v1/models` local e publico 200, payload raiz apenas `data` e sem campos internos.
- Chat non-stream, chat stream e Responses stream publicos 200.
- API key interna invalida retorna 401 interno, sem marcador de auth upstream.
- Probe do channel 5 retorna `authenticated=true`, `last_probe_status=ok` e nenhum erro upstream atual.
- Commits e push em `origin/main` concluidos.

## Blocker

A credencial live ainda e o fallback access-token-only, expira em `2026-07-17T11:04:04Z`, nao possui `refresh_token` e continua com `requires_regeneration=true`. O Vault estava selado e o perfil Brave disponivel nao estava autenticado no ChatGPT, impedindo concluir o callback OAuth Router-owned sem interacao humana.

## Acao para desbloquear

Autenticar o Brave no ChatGPT, executar `Regenerar credencial`, concluir o callback PKCE e repetir probe + refresh. Nao configurar Headroom como parte desta fase.
