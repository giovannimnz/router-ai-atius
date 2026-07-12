# Phase 32 Plan Review

## Resultado

PASS WITH EXECUTION RISKS.

## Verificacoes

- Goal-backward: os quatro planos cobrem backend, UI, docs/fork-sync e validacao live ate commit/push.
- Requirements: todos os IDs `PHASE-32-*` estao cobertos.
- Nyquist: o plano inclui positivo, negativo interno, negativo upstream, build, runtime e docs.
- CPU guardrail: todos os comandos pesados planejados usam `./scripts/podman-admin.sh`.
- Secret hygiene: nenhum passo imprime token; backup e Vault sao tratados como operacionais.
- Fork-sync: a fase inclui protecao no repo e no `omni-srv-admin`.

## Riscos a Controlar na Execucao

- O fluxo OAuth depende do redirect `localhost:1455`; se o browser nao estiver no mesmo host ou se DevTools nao expuser a URL, usar fallback manual de callback.
- Criar health em `channel.setting` exige merge cuidadoso para nao sobrescrever settings existentes.
- O negativo upstream live nao deve corromper channel 5; preferir mock/unit ou channel temporario com backup.
- A etapa de Graphify pode tocar arquivos `.planning/graphs` que ja estavam dirty antes desta fase; separar staging para nao misturar mudancas antigas.

## Recomendacao

Executar na ordem 32-01, 32-02, 32-03, 32-04. Nao pular a regeneration propria do Router, porque o hotfix access-token-only expira em `2026-07-17T11:04:04Z` e nao possui autorrenovacao.
