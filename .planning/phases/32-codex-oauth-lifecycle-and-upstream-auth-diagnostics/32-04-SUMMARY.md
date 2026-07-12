---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
plan: 32-04
status: complete
completed: 2026-07-12
requirements_completed:
  - PHASE-32-VALIDATION-DOCS-SHIP
---

# 32-04 Summary - Validacao live e fechamento OAuth

## Resultado

O callback Authorization Code + PKCE foi concluido no channel 5. A credencial
live deixou de ser `codex-cli-hotfix` e passou a ser `router_owned`, com
`refresh_token` proprio do Router e renovacao automatica disponivel.

## Provas live

- metadata antes do refresh: `authenticated=true`, `has_refresh_token=true`,
  `requires_regeneration=false`;
- probe upstream: `success=true`, `last_probe_status=ok`;
- refresh manual: `success=true`, `last_refresh=2026-07-12T12:47:27-03:00` e
  `expires_at=2026-07-22T12:47:27-03:00`;
- `/v1/models` local e publico: HTTP 200, raiz somente `data`, sem campos
  internos, com `gpt-5.5`, `gpt-5.6-sol` e `gpt-5.6-terra` preservados;
- chat non-stream, chat stream e Responses stream: HTTP 200 local e publico;
- token interno invalido: HTTP 401 de auth interna do Router, sem marcador de
  auth upstream Codex;
- `clianything status --strict`: PASS;
- containers nao-infra: limite de `0.800 CPU` mantido.
- warning generico de Base URL no type 57 legado removido por boundary testavel;
- teste do painel/boundary: 4/4; typecheck: PASS; build Rsbuild: PASS, todos
  executados pelo wrapper de 20% CPU.

## Seguranca operacional

- backup live anterior ao refresh:
  `backups/clianything/20260712_124648_phase32_live_pre_refresh_channels.sql`;
- a validacao administrativa usou access token root aleatorio e efemero;
- o token efemero foi removido ao final por cleanup garantido;
- nenhum callback, authorization code, access token ou refresh token foi
  impresso ou documentado.

## Descoberta adicional

O `clianything` em modo host consulta o PostgreSQL local por default, enquanto o
runtime usa o DSN canonico `10.11.1.11:6432/DBRouterAiAtius`. Validacoes live de
credencial devem resolver o `SQL_DSN` do container ou configurar explicitamente
o target do CLI, sempre sem imprimir a senha.

O audit de integracao tambem confirmou as cinco rotas Codex consumidas pela UI,
o ciclo PKCE -> probe -> refresh, a taxonomia de auth upstream, a preservacao do
catalogo e as guardas do fork-sync. O negativo upstream permanece coberto por
testes deterministas; nao se invalida deliberadamente uma credencial live recem
regenerada apenas para produzir evidencia operacional negativa.

## Conclusao

Os seis requisitos da Phase 32 passaram. Nao ha dependencia de Headroom e
nenhuma instalacao/configuracao de Headroom foi feita.
