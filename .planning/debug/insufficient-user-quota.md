---
slug: insufficient-user-quota
status: resolved
trigger: "POST autenticado para /v1/chat/completions retorna 403 insufficient_user_quota; incidente recorrente apesar de correção anterior. Credencial redigida."
created: 2026-08-14T12:10:00-03:00
updated: 2026-08-14T13:26:00-03:00
---

# Debug: bloqueio recorrente por user quota no Atius Router

## Symptoms

- expected: tokens pessoais válidos nunca são bloqueados por saldo ou user quota.
- actual: `POST https://router.atius.com.br/v1/chat/completions` com `gpt-5.4` retorna HTTP 403.
- error: `code=insufficient_user_quota`; saldo interno exibido como negativo.
- timeline: incidente equivalente documentado em 2026-07-30; voltou em 2026-08-14.
- reproduction: chamada pública autenticada, prompt mínimo e modelo `gpt-5.4`.

## Constraints

- Não registrar tokens, keys ou valores de segredos.
- Preservar mudanças preexistentes na working tree compartilhada.
- Builds e suítes pesadas somente por `scripts/podman-admin.sh`, com teto de 20% da CPU total.
- Provar o caminho público, não apenas localhost.
- Consultar e atualizar repo, skills, GBrain HTTP e Obsidian HTTP.

## Current Focus

- hypothesis: resolved — o build clean omitia o patch não commitado e não havia proteção integral dos admission paths.
- test: concluído — lifecycle de repair, testes Go, build limpo, preflight Omni, deploy e smokes local/público.
- expecting: requests pessoais continuam funcionando com wallet negativa, enquanto token explícito e provider permanecem limites independentes.
- next_action: monitorar os guards automáticos nos próximos upstream syncs; nenhuma ação corretiva pendente.

reasoning_checkpoint:
  hypothesis: "O patch wallet correto nunca entrou no artefato clean e outros fluxos ainda usam saldo/subscription como autorização; isso causa o 403 recorrente e deixa variantes equivalentes possíveis."
  confirming_evidence:
    - "O request live falhou com a string do wallet guard antigo após o start da imagem atual, enquanto essa string está ausente no dirty source."
    - "Os arquivos do patch têm mtime de 2026-07-30 e seguem uncommitted; Obsidian registra explicitamente ausência de deploy/restart."
    - "Auditoria completa encontrou blockers adicionais em PreWssConsumeQuota, dois caminhos Midjourney e fallback de subscription."
  falsification_test: "Um binário construído do source corrigido que ainda retorna erro local de account/subscription/user balance, ou focused tests/audit que encontrem esses blockers, refuta a completude da correção."
  fix_rationale: "Remover decisões por saldo, tornar toda indisponibilidade de subscription um fallback para wallet e versionar um patch auditável corrige a causa funcional e impede novos builds/syncs de omitirem silenciosamente o invariant."
  blind_spots: "Por ordem do parent não haverá deploy/restart nem smoke público nesta etapa; a validação end-to-end live permanecerá como checkpoint obrigatório."
- reasoning_checkpoint: histórico Obsidian confirma patch local sem deploy e testes sem green run.
- tdd_checkpoint: false

## Evidence

- timestamp: 2026-08-14T13:33:00-03:00
  source: scripts/podman-admin.sh
  finding: `compose-up`, `run-container` e `prod-restart` agora executam `audit_user_quota_invariant` antes de iniciar/promover runtime; os caminhos chamam apenas `audit`, nunca `repair`.
- timestamp: 2026-08-14T13:36:00-03:00
  source: gofmt + git diff --check
  finding: os sete arquivos Go da correção/testes foram formatados; a checagem de whitespace focada terminou sem erro.

- timestamp: 2026-08-14T12:10:00-03:00
  source: Obsidian HTTP
  finding: incidente de 2026-07-30 aplicou patch em `service/pre_consume_quota.go` e `service/billing_session.go`, mas não fez deploy/restart produtivo.
- timestamp: 2026-08-14T12:10:00-03:00
  source: runtime redigido
  finding: pod e backend estão saudáveis; request autenticado reportado pelo usuário falha antes do upstream com `insufficient_user_quota`.
- timestamp: 2026-08-14T12:16:00-03:00
  source: Graphify
  finding: grafo existe, está fresh no commit atual `25a4dd2` e contém 37891 nós; a query combinada de quota não retornou nós, portanto o roteamento será refinado com code intelligence e busca textual focada.
- timestamp: 2026-08-14T12:16:00-03:00
  source: runtime/code diff fornecido
  finding: o request reportado consta nos logs depois do start da imagem ativa e prova execução do wallet guard antigo; o código Go local remove esse guard, mas mantém emissores em branches de subscription de `service/billing_session.go`.
- timestamp: 2026-08-14T12:18:00-03:00
  source: GBrain HTTP
  finding: code graph remoto não está built/ready para `ErrorCodeInsufficientUserQuota` e a busca semântica não retornou páginas; portanto não fornece call graph confiável nesta sessão.
- timestamp: 2026-08-14T12:18:00-03:00
  source: Obsidian HTTP
  finding: a busca encontrou o incidente canônico `ideaverse/61-Incidents/2026-07-30-hermes-home-s20-403-preconsumo-gpt54.md`, que registra wallet overdraft permitido, guards restritos a subscription sem overflow, token guard preservado e ausência de restart/deploy após o patch.
- timestamp: 2026-08-14T12:20:00-03:00
  source: Obsidian HTTP
  finding: leitura integral confirma que o patch tocou `service/pre_consume_quota.go`, `service/billing_session.go` e adicionou teste de overdraft; a suíte não obteve green run por problema antigo no wrapper de compilador, e não houve restart/deploy produtivo.
- timestamp: 2026-08-14T12:20:00-03:00
  source: Codex memory quick-pass
  finding: o registro do hotfix de catálogo indica que `container-router-ai-atius.service` usa imagem rootless Podman e que apenas substituir imagem/binário alterou o comportamento live; mudanças de DB/options não substituíram código antigo.
- timestamp: 2026-08-14T12:23:00-03:00
  source: project skills + Codex rollout memory
  finding: os cinco project skills foram inspecionados e nenhum acrescenta regra de quota; o rollout confirma risco de `pull-and-restart.sh latest` reintroduzir binário remoto antigo após hotfix local, por isso deploy deve preservar rollback e validar imagem/digest após restart.
- timestamp: 2026-08-14T12:23:00-03:00
  source: IJFW integration
  finding: nenhuma tool `ijfw_memory_prelude` está exposta no runtime MCP atual; investigação continua com Graphify fresh, GBrain HTTP, Obsidian HTTP e memória Codex.
- timestamp: 2026-08-14T12:28:00-03:00
  source: CLIAnything read-only
  finding: pod e backend estão saudáveis; `OpenAI - Codex` está ativo com sete abilities, eliminando indisponibilidade geral/provider disabled como causa do 403 local.
- timestamp: 2026-08-14T12:28:00-03:00
  source: source diff + full file reads
  finding: o working tree remove exatamente dois guards wallet (`userQuota <= 0` e `userQuota-preConsumedQuota < 0`) de ambos os caminhos antigos; `NewBillingSession.tryWallet` agora só lê quota e delega a `WalletFunding`, enquanto emissores restantes observados são de falha de subscription.
- timestamp: 2026-08-14T12:28:00-03:00
  source: regression test
  finding: `service/billing_session_wallet_overdraft_test.go` cobre wallet negativo, fallback subscription→wallet permitido, bloqueio de subscription sem overflow e preservação do bloqueio por token insuficiente com assertions determinísticas.
- timestamp: 2026-08-14T12:31:00-03:00
  source: GBrain HTTP code intelligence
  finding: `code_blast`, `code_callers` e `code_flow` foram executados antes de qualquer source edit, mas o source code graph remoto retornou `not_found/not_built`; o blast radius foi então derivado por busca local focada.
- timestamp: 2026-08-14T12:31:00-03:00
  source: local call-site search
  finding: os entry points `controller/relay.go` e `relay/relay_task.go` chamam `PreConsumeBilling`, que chama `NewBillingSession`; `PreConsumeQuota` legado não possui caller textual. Os únicos emissores atuais são subscription (linhas 219/244) e o teste que os valida.
- timestamp: 2026-08-14T12:36:00-03:00
  source: funding/caller reads
  finding: `WalletFunding.PreConsume` e `Settle` delegam diretamente a `model.DecreaseUserQuota` sem saldo guard; o fluxo síncrono chama `PreConsumeBilling` antes do upstream e o task flow usa o mesmo entry point, confirmando o blast radius do patch.
- timestamp: 2026-08-14T12:37:00-03:00
  source: focused test attempt
  finding: o wrapper abortou antes do build com `build-cpu-guard: real command not found for go`; nenhum teste executou e a hipótese funcional permanece não testada.
- timestamp: 2026-08-14T12:42:00-03:00
  source: focused test retry
  finding: retry com `/usr/local/go/bin/go`, `CC=/usr/bin/gcc`, `GOMAXPROCS=1` e `-p 1` iniciou sob `podman-admin.sh profile-run`; após cerca de 3 minutos o processo permanecia ativo (`Sl`) sem output, ainda sem resultado funcional.
- timestamp: 2026-08-14T12:42:00-03:00
  source: build/release evidence supplied
  finding: quota source/test files têm mtime de 2026-07-30 e seguem uncommitted; a imagem construída em 2026-08-14 contém string do guard ausente no dirty checkout, provando que o build omitiu mudanças uncommitted.
- timestamp: 2026-08-14T12:49:00-03:00
  source: focused test result
  finding: os quatro testes executaram e falharam em 0.099s antes da lógica de wallet; SQLite gerou `SELECT ... FROM tokens WHERE  = ?` em `model/token.go:282`, retornando `pre_consume_token_quota_failed`. O teste funcional permanece inconclusivo por fixture de DB inválida.
- timestamp: 2026-08-14T12:49:00-03:00
  source: fork/WSS audit
  finding: `scripts/fork-sync-guard.sh` protege apenas identidade/remotes/push/workflow e Rule 10 não lista quota files; `PreWssConsumeQuota` possui guard separado `userQuota < quota`, chamado pelo handler OpenAI realtime, portanto não causa o chat completion reportado mas viola o invariant mais amplo em realtime.
- timestamp: 2026-08-14T12:53:00-03:00
  source: test fixture inspection
  finding: `service/task_billing_test.go::TestMain` abre SQLite e chama `common.SetDatabaseTypes`, mas não chama `model.InitLogDB`; `model/main.go` mostra que `commonKeyCol` só é preenchido por `initCol`, invocado pelo initializer público.
- timestamp: 2026-08-14T13:04:00-03:00
  source: focused test retry after commonKeyCol fix
  finding: wallet overdraft e token-limit cases passaram; os dois casos de subscription chegaram mais longe e falharam apenas porque a fixture não migrou `subscription_pre_consume_records`, confirmando a primeira correção da fixture e isolando o próximo setup necessário.
- timestamp: 2026-08-14T13:18:00-03:00
  source: integral source patch
  finding: foram removidas leituras de `GetUserQuota` e decisões por saldo dos admission paths wallet, WSS e Midjourney; subscription_only, ausência, exhaustion e flag strict agora convergem para wallet, enquanto explicit token quota permanece bloqueante.
- timestamp: 2026-08-14T13:18:00-03:00
  source: static audit
  finding: `rg` confirma zero caller/constante `ErrorCodeInsufficientUserQuota` e zero `GetUserQuota`, `userQuota`, `quota_not_enough` ou mensagem equivalente nos quatro runtime files; ambos `atius-user-quota-guard.sh audit` e `fork-sync-guard.sh user-quota-audit` passam.
- timestamp: 2026-08-14T13:22:00-03:00
  source: clean-worktree lifecycle attempt 1
  finding: pre-audit falhou corretamente e listou todos os blockers do HEAD baseline, mas `repair` não iniciou porque `audit()` encerrou o processo em vez de apenas retornar non-zero; worktree temporário foi limpo pelo trap.

## Eliminated

- hypothesis: existe um segundo guard de user quota no fluxo wallet do código Go local atual.
  evidence: leitura integral de `service/pre_consume_quota.go` e `service/billing_session.go` mostra que os emissores remanescentes estão no tratamento/reserva de subscription; `tryWallet` não retorna mais `ErrorCodeInsufficientUserQuota` por saldo.
  timestamp: 2026-08-14T12:28:00-03:00

## Resolution

- root_cause: "O invariant de saldo ilimitado foi implementado em 2026-07-30 apenas no dirty checkout; builds clean omitiram o patch e a imagem live continuou com wallet guards antigos. Fluxos WSS/Midjourney e subscription fallback também preservaram decisões locais por saldo/subscription, e não existia guard versionado para detectar a regressão antes de build/push."
- fix: "Aplicado e implantado: removidos balance admission guards em wallet/WSS/MJ; subscription sempre cai para wallet; mantidos apenas token/upstream limits; adicionados testes, patch canônico, auto-repair e gates de build/sync/deploy no Router e omni-srv-admin."
- verification:
  - "Focused Go tests: PASS para wallet negativa, subscription strict/only/ausente, WSS negativo com token unlimited e token finito insuficiente."
  - "Clean worktree: baseline audit bloqueou; repair aplicou; audit passou; segundo repair foi no-op."
  - "Omni preflight: zero violações, 8/8 artefatos protegidos e post_sync inicia com repair."
  - "Imagem 3de044df43073346c363af3c4a6346acaaf2e93c196650991087c35f4710bbab ativa; rollback 6fb96b572ab28290b66f7343abd55afd5c666a90f6da7867895862e9c85b1916 preservado."
  - "POST local e público para gpt-5.4 retornaram HTTP 200 e conteúdo Y; logs confirmam relay 200 sem insufficient_user_quota."
- files_changed:
  - service/task_billing_test.go
  - service/billing_session_wallet_overdraft_test.go
  - service/pre_consume_quota.go
  - service/billing_session.go
  - service/quota.go
  - relay/mjproxy_handler.go
  - types/error.go
  - scripts/atius-user-quota-guard.sh
  - patches/atius-user-quota-unlimited.patch
  - scripts/fork-sync-guard.sh
  - scripts/ci-build-backend.sh
  - scripts/podman-admin.sh
  - scripts/check-upstream-sync-workflow.sh
  - docs/ATIUS-USER-QUOTA-INVARIANT.md
  - docs/API.md
  - AGENTS.md
  - .planning/debug/insufficient-user-quota.md
