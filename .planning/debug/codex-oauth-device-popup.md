---
name: codex-oauth-device-popup
slug: codex-oauth-device-popup
status: converged_local_pending_uat
trigger: "O login do channel OpenAI - Codex deveria priorizar device auth; ao regenerar, o popup e reportado como bloqueado mesmo habilitado e o modal com campo para colar o callback nao aparece. Investigar regressao e recorrencia de refresh_token_invalidated."
trigger_source: user-reported
created: 2026-07-13
updated: 2026-07-13T03:26:00-03:00
symptoms:
  expected: "Regenerar credencial inicia exclusivamente device auth, exibe URL e codigo para login e nunca oferece um fallback mutante por PKCE."
  actual: "A UI tenta abrir uma janela de autorizacao, acusa popup bloqueado mesmo com popups permitidos e nao apresenta o modal/campo de callback manual."
  error_messages:
    - "O navegador bloqueou a janela de autorizacao. Permita pop-ups para este site e clique novamente em Regenerar credencial."
    - "probe_status=auth_failed"
    - "upstream_status=401"
    - "refresh_token_invalidated"
  timeline:
    - "Phase 32 foi fechada em 2026-07-12 como OAuth Authorization Code + PKCE com fallback manual."
    - "Falha observada no runtime publico em 2026-07-13."
  reproduction:
    - "Abrir o channel OpenAI - Codex (type 57), entrar em Credenciais e clicar Regenerar credencial."
hypothesis: "O fechamento da Phase 32 implementou Authorization Code + PKCE como unico fluxo e tornou o modal dependente do sucesso sincrono de window.open; isso removeu device auth e escondeu o fallback manual justamente quando o popup falha. A credencial router_owned tambem pode estar competindo com outra sessao Codex pelo mesmo refresh token rotativo."
next_action: "Concluir testes focados locais; depois, fora desta tarefa, executar UAT humano e validacao MySQL/PostgreSQL antes de qualquer claim cross-database."
reasoning_checkpoint: "Reviewer findings convergidos em codigo; build, live, deploy, UAT e integracao MySQL/PostgreSQL nao foram executados nesta tarefa."
files_changed:
  - service/codex_oauth.go
  - service/codex_device_auth_test.go
  - service/codex_credential_refresh.go
  - service/codex_credential_refresh_test.go
  - controller/codex_oauth.go
  - router/channel-router.go
  - web/default/src/features/channels/api.ts
  - web/default/src/features/channels/types.ts
  - web/default/src/features/channels/components/codex/codex-regenerate-dialog.tsx
  - web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx
  - web/default/src/features/channels/components/codex/codex-credential-panel.test.tsx
  - model/codex_oauth_operation.go
  - model/main.go
  - docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md
evidence:
  - timestamp: 2026-07-13T02:27:00-03:00
    type: screenshot
    finding: "Drawer mostra refresh_token_invalidated, probe 401 e alerta de popup bloqueado; modal de callback manual ausente."
  - timestamp: 2026-07-13T03:00:00-03:00
    type: graphify
    finding: "Fluxo roteado para controller/codex_oauth.go e componentes Codex/channel drawer do frontend."
  - timestamp: 2026-07-13T03:01:00-03:00
    type: official_docs
    finding: "Codex App Server documenta chatgptDeviceCode como fluxo apropriado quando o cliente controla a cerimonia ou o callback de browser e fragil."
  - timestamp: 2026-07-13T03:05:00-03:00
    type: git_forensics
    finding: "b4f9dc335 passou a tratar window.open(..., noopener,noreferrer) retornando null como popup bloqueado e lancar erro antes de abrir o modal; MDN confirma que noopener faz o retorno ser null."
  - timestamp: 2026-07-13T03:08:00-03:00
    type: runtime
    finding: "O 401 token_invalidated ocorreu antes do primeiro refresh manual; nao houve refresh local anterior nos logs. A frota usa o mesmo account_id e atius-srv-3 renovou sua sessao apos a credencial do Router, tornando rotacao externa a hipotese dominante para este incidente."
  - timestamp: 2026-07-13T13:00:00-03:00
    type: reviewer_convergence
    finding: "Refresh persiste upstream_started antes da call; device persiste exchange_started antes do exchange; retomadas ambiguas terminalizam uncertain_requires_regeneration e nao repetem segredos one-time."
  - timestamp: 2026-07-13T13:00:00-03:00
    type: scope
    finding: "PKCE removido da UI; endpoints legacy respondem 410 antes de sessao, exchange ou channel write; operation model migra no startup e request path apenas verifica a tabela."
eliminated: []
root_cause: "Device auth nunca foi implementado porque Phase 32 escolheu apenas Authorization Code + PKCE. Depois, b4f9dc335 usou o retorno de window.open com noopener/noreferrer como detector de bloqueio e lancou antes de abrir o modal, embora esse contrato retorne null mesmo com a aba aberta. Em paralelo, refreshes locais nao eram serializados e copias divergentes da mesma conta na frota aumentavam o risco de invalidacao de tokens rotativos."
fix: "Device authorization SQL cancelavel/fenced virou o unico fluxo mutante. Refresh e exchange gravam pre-stage duravel; qualquer janela ambigua termina em uncertain_requires_regeneration sem replay. PKCE legacy ficou fail-closed. Polling de erro transitório preserva a operacao."
verification: "Gofmt via profile-run; Go service e router focados PASS; Bun Codex 7/7; tsgo PASS. Build, live, deploy, UAT, Redis real e integracao MySQL/PostgreSQL nao foram executados."
tags:
  - codex
  - oauth
  - device-code
  - popup
  - refresh-token
severity: high
impact: "Administradores nao conseguem regenerar de modo confiavel a credencial Codex e o channel fica indisponivel apos invalidacao upstream."
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "Authorization Code + PKCE foi tratado como unico fluxo; o modal manual depende do retorno nao-nulo de window.open; device auth nao foi implementado; refresh token pode estar sendo rotacionado fora do Router."
  confirming_evidence:
    - "32-CONTEXT.md escolheu explicitamente Authorization Code + PKCE."
    - "Capturas mostram erro de popup e ausencia do modal manual."
    - "Docs oficiais atuais recomendam device-code quando callback e fragil."
  falsification_test: "Se o frontend abrir o modal antes de window.open e o backend possuir endpoints device-code completos, a hipotese principal e falsa."
  fix_rationale: "Usar somente device-code SQL fenced elimina o segundo caminho mutante e permite cancelar/expirar sem channel write tardio."
  blind_spots: "Sem prova nesta tarefa em MySQL/PostgreSQL, Redis externo, processo realmente morto ou runtime live; os testes injetam as fronteiras de falha localmente."

## Evidence

## Eliminated

## Resolution
