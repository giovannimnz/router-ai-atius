---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
reviewed: 2026-07-13T13:00:00-03:00
depth: convergence
status: validated_local
original_findings: 8
resolved_findings: 8
---

# Phase 32: Reviewer Convergence

## Status

Os oito findings do review de 2026-07-13 foram enderecados no escopo OAuth.
Esta convergencia executou testes Go/Bun, typecheck, i18n sync e build frontend
sob o limite de 0.8 CPU. Deploy e live request permanecem no gate das Phases
29/30. MySQL/PostgreSQL externos nao fazem parte da evidencia local.

## Findings

### CR-01: refresh token rotativo

Resolvido por uma operation row SQL fenced. `upstream_started` e persistido em
transacao curta antes da call upstream. O resultado rotacionado, quando obtido,
e selado antes do update final do channel. Timeout, perda de processo ou falha
entre retorno upstream e persistencia terminaliza a operation como
`uncertain_requires_regeneration`; o refresh token antigo nunca e reutilizado.

Claim honesta: upstream e SQL nao sao atomicamente transacionais. A seguranca e
fail-closed na janela ambigua, com regeneracao obrigatoria, nao recuperacao
magica de um resultado que morreu antes de ser persistido.

### CR-02: exchange e fencing

Resolvido com lease renovavel, fence monotonic e update condicional do channel
na mesma transacao que salva o estado final. `exchange_started` e persistido
antes de enviar o authorization code one-time. Falha/timeout/morte nessa janela
vira `uncertain_requires_regeneration`; takeover nao repete o exchange.

### CR-03: cancelamento e expiry

Resolvido com endpoint de cancelamento, `AbortSignal` no poll e verificacao de
status, expiry, owner, fence e lease na transacao que atualiza `channels.key`.
Cancel/expiry/lease lost retornam diretamente do runner e nao tentam uma segunda
transicao com owner vencido.

### WR-01: store compartilhado

Resolvido substituindo fallback runtime por SQL compartilhado. O model
`model.CodexOAuthOperation` entra no `migrateDB` canonico e no fast path. Request
path apenas verifica a existencia da tabela e falha fechado; nao executa
`AutoMigrate` lazy.

### WR-02: erros transitorios

Resolvido com backoff preservando o expiry absoluto. Erro transitorio de store
no poll retorna `retryable`/`retry_after` e nao limpa sessao nem cancela a
operation. Frontend chama cancel apenas para terminal, `cancelled` ou `expired`.
Depois de `exchange_started`, erros nao sao retryable porque repetir o code seria
inseguro.

### WR-03: qualidade da evidencia

Resolvido parcialmente no limite honesto desta tarefa: existem testes
deterministicos SQLite/in-memory para cancel/expiry/fence, takeover de resultado
persistido, crash em `exchange_started`, timeout sem replay e falha injetada
exatamente apos retorno de refresh e antes da persistencia. Nao se declara prova
de MySQL, PostgreSQL, Redis externo ou morte real de processo.

### WR-04: anti-phishing

Resolvido com allowlist exata de `https://auth.openai.com/codex/device`. Origins,
userinfo e paths alternativos nao geram link clicavel. O caminho browser PKCE
foi removido da UI.

### WR-05: claims operacionais

Resolvido no manual e no debug artifact: nao ha claim de atomicidade upstream +
SQL, cross-database, build, live ou deploy. O estado ambiguo e a regeneracao
obrigatoria estao documentados explicitamente.

## PKCE Legacy

Os endpoints `/codex/regenerate/start` e `/codex/regenerate/complete` continuam
registrados somente por compatibilidade e respondem `410 codex_pkce_disabled`
antes de ler body/sessao, chamar upstream ou escrever channel. A UI nao importa
nem chama essas APIs; device flow SQL cancelavel/fenced e o unico caminho
mutante.

## Verification

- `gofmt`: executado via `scripts/podman-admin.sh profile-run`.
- Go focado: `./service` PASS e `./router` PASS.
- Frontend Codex: Bun `7 pass, 0 fail`; `bun run typecheck` PASS.
- Build/live/deploy/commit: nao executados por escopo.
- MySQL/PostgreSQL/Redis externo: nao executados; residual de integracao.
