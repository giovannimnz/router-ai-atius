# Phase 06: Login Explícito + Armazenamento Licença (SDK-02) - Context

**Gathered:** 2026-06-06
**Status:** Ready for planning

## Phase Boundary

Implementar o fluxo explícito de autenticação do Codex SDK no admin, com persistência própria de credenciais e leitura pelo sidecar da Phase 05. O objetivo desta fase é decidir como o admin autentica, onde a licença vive, como ela chega ao `data/codex/license.json`, e como o sistema reage quando a licença SDK está ausente ou inválida — sem fallback silencioso para credenciais do host.

## Implementation Decisions

### Superfície de auth
- **D-01:** A autenticação SDK vive numa rota dedicada global: `/admin/codex-auth`. Não é um fluxo inline por canal.
- **D-02:** O drawer do canal Codex mantém só status + link para `/admin/codex-auth`. Nada de fluxo completo de OAuth dentro do drawer.
- **D-03:** A página `/admin/codex-auth` terá 2 blocos na mesma tela: (a) colar authorization code / callback URL do OAuth; (b) importar JSON manual.
- **D-04:** A página mostra status completo da licença e ações administrativas: `email`, `account_id`, `expired`, `last_refresh`, `source`, botão de refresh manual, e download/export completo da credencial.
- **D-05:** A navegação da página será por atalho dentro de `Channels` + rota dedicada. Não precisa entrada top-level separada no menu.

### Dono da credencial
- **D-06:** A source of truth operacional continua por canal. Cada canal Codex mantém sua própria `key` OAuth JSON.
- **D-07:** `data/codex/license.json` não é a fonte primária. Ele é um cache auxiliar que espelha um canal Codex primário.
- **D-08:** Pode existir mais de um canal com `backend=sdk`.
- **D-09:** O canal primário que espelha para `data/codex/license.json` é escolhido manualmente. O facto de um canal usar `backend=sdk` não o torna primário automaticamente.

### Defaults aplicados por timeout
- **D-10:** Reload do sidecar usa cache em memória com detecção por `mtime` e reload automático quando `data/codex/license.json` muda. (timeout — best-judgment default: evita IO a cada request e não exige botão manual extra para o sidecar reaprender a licença.)
- **D-11:** Se um canal `backend=sdk` não tiver licença válida/primária disponível, o comportamento é hard fail com erro explícito de configuração; nunca fallback silencioso para `relay`. A UI pode avisar/bloquear, mas o router não troca backend por conta própria. (timeout — best-judgment default: alinha com login explícito + zero fallback silencioso.)

### Claude's Discretion
Detalhes internos de implementação podem variar desde que preservem D-01..D-11, em especial:
- Onde armazenar o ponteiro do canal primário (setting global, option, ou metadata equivalente)
- Como serializar `source`/`primary_channel_id` no status retornado à UI
- Como acoplar o refresh manual da página ao refresh já existente por canal

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope
- `.planning/PROJECT.md` — premissas de v2.14, incluindo login explícito e proibição de reuso silencioso de `~/.codex/auth.json`
- `.planning/REQUIREMENTS.md` — definição formal de SDK-02
- `.planning/ROADMAP.md` § Phase 06 — meta, scope e verification da fase
- `.planning/STATE.md` — ordem de execução e dependência em relação à Phase 05

### Prior phase dependency
- `.planning/phases/05-sidecar-python-http-bridge-sdk-01/05-CONTEXT.md` — decisões já travadas sobre sidecar, SSE e volume `./data/codex`
- `.planning/phases/05-sidecar-python-http-bridge-sdk-01/05-SUMMARY.md` — arquivos realmente criados na fundação do sidecar
- `service/codex_sdk.go` — chamadas do router Go para `codex-sidecar`
- `docker-compose.yml` — serviço `codex-sidecar` e volume `./data/codex:/app/data`

### Existing Codex auth / refresh behavior
- `controller/codex_oauth.go` — fluxo atual de OAuth e geração de key JSON
- `controller/codex_usage.go` — refresh sob demanda quando usage pega 401/403
- `controller/channel.go` — endpoint manual de refresh por canal
- `service/codex_oauth.go` — authorization flow, token exchange, refresh token logic
- `service/codex_credential_refresh.go` — refresh persistido no `channel.key`
- `service/codex_credential_refresh_task.go` — auto-refresh batch existente por canal
- `relay/channel/codex/oauth_key.go` — shape canônico do OAuth JSON

### Frontend integration points
- `web/default/src/features/channels/api.ts` — APIs existentes de start/complete OAuth, usage e refresh
- `web/default/src/features/channels/components/dialogs/codex-oauth-dialog.tsx` — UX atual de OAuth dialog por canal
- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx` — bloco atual de `Codex Authorization` no drawer
- `dto/channel_settings.go` — `CodexBackend` (`relay|sdk`) já existente

## Existing Code Insights

### Reusable Assets
- `controller/codex_oauth.go`: já resolve PKCE + code exchange e produz JSON com `access_token`, `refresh_token`, `account_id`, `email`, `expired`, `last_refresh`.
- `service/codex_credential_refresh.go` + `service/codex_credential_refresh_task.go`: já existe refresh manual e automático por canal; Phase 06 deve reaproveitar isso, não reimplementar OAuth refresh do zero.
- `web/default/src/features/channels/components/dialogs/codex-oauth-dialog.tsx`: o fluxo de instruções e UX de colar callback já está pronto e pode virar bloco reutilizável da nova página.
- `web/default/src/features/channels/api.ts`: endpoints frontend de OAuth/refresh já existem e podem ser adaptados/estendidos para a nova superfície global.

### Established Patterns
- OAuth Codex no código atual é channel-centric: credencial fica em `channel.key` e refresh também.
- Sidecar já monta `./data/codex` como volume persistente; logo `data/codex/license.json` encaixa no desenho atual sem novo storage externo.
- O projeto já diferencia `backend=relay|sdk` em `dto/channel_settings.go`; Phase 06 precisa complementar o fluxo de credencial, não reinventar o switching.

### Integration Points
- Nova rota/página admin `/admin/codex-auth` precisa conversar com o bloco atual de Channels sem duplicar responsabilidade.
- O espelhamento para `data/codex/license.json` precisa acontecer a partir de um canal primário explícito.
- O sidecar precisa reagir a mudanças do arquivo `license.json` sem restart manual do container.
- O planner deve considerar como representar `primary_channel_id` / `source` / validade na API de status da nova página.

## Specific Ideas

- Reaproveitar o conteúdo do `CodexOAuthDialog` como bloco da nova página, em vez de manter dois fluxos completos concorrentes.
- O drawer de canal pode virar só um painel de observabilidade leve: status atual + link para a página global.

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 06-Login Explícito + Armazenamento Licença (SDK-02)*
*Context gathered: 2026-06-06*
