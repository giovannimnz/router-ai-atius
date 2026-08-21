# Debug: gpt-5.4-mini com taxa de sucesso 0%

status: resolved
opened: 2026-08-21
resolved: 2026-08-21
surface: dashboard/overview

## Sintoma

O painel de producao exibia `gpt-5.4-mini` com taxa de sucesso de 0% nas
ultimas 24 horas.

## Evidencias

- Tres falhas originais no channel `5`, `ChatGPT - Codex`, rota
  `/v1/messages`, todas HTTP 401.
- Taxonomia normalizada: `codex_upstream_auth_failed`.
- O refresh automatico retornava `refresh_token_invalidated`.
- A credencial do banco tinha `last_refresh=2026-08-15`; o Codex Desktop tinha
  uma credencial da mesma conta renovada em 2026-08-21.
- Antes da correcao, Chat Completions, Messages e Responses falharam com 401.

## Causa raiz

O channel do Router e o Codex Desktop compartilhavam uma copia da mesma cadeia
de refresh token. Quando o Desktop rotacionou a credencial, a copia salva no
channel ficou invalidada. O problema afetava o channel Codex inteiro; o painel
destacou apenas `gpt-5.4-mini` porque era o unico modelo com trafego no periodo
da falha.

## Correcao

- Credencial do channel `5` regenerada a partir do `~/.codex/auth.json` atual,
  depois de confirmar a mesma conta sem expor tokens.
- Channel marcado com `codex_credential_source=external_file`.
- Auto-refresh interno passa a ignorar credenciais gerenciadas externamente.
- Sincronizador idempotente instalado com path unit e timer de 5 minutos.
- Formulario de canais passa a preservar metadados desconhecidos de `setting`,
  evitando remover ownership/health ao editar o channel.

## Validacao

- Chat Completions: HTTP 200.
- Claude Messages: HTTP 200.
- Responses: HTTP 200.
- Sete sucessos e zero erros desde a correcao.
- Bucket atual: 4/4, 100%.
- Janela de 24h efetiva: 7/13, 53,85%; os seis 401 historicos permanecem ate expirarem da
  janela, sem reescrever telemetria passada.
- Teste Go `TestShouldAutoRefreshCodexChannel`: passou.
- Teste frontend de preservacao de setting: passou.
- Typecheck e build Rsbuild: passaram.

## Producao

- Imagem: `localhost/router-ai-atius:prod-20260821-gpt54-mini-auth-sync`.
- Image ID: `ae5839459fea595aa4d8ebc7aded6c8006251a529bfa039aa76741a24e1365c8`.
- Rollback: `localhost/router-ai-atius:rollback-pre-gpt54-mini-auth-sync-20260821`.
