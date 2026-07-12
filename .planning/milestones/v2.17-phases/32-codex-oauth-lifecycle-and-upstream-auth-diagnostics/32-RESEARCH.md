# Phase 32 Research — Codex OAuth lifecycle and upstream auth diagnostics

## Pergunta

Como transformar o hotfix de `token_invalidated` do channel 5 `OpenAI - Codex` em uma implementacao duravel, com UI correta, credencial propria do Router, diagnostico rapido e protecao contra regressao por upstream sync?

## Evidencias Relevantes

### Incidente 2026-07-10

- Sessao `019f3ef0-bf50-7e93-9834-af51f040c1db` e Obsidian `60-LOGS/2026-07-10-router-ai-atius-codex-token-invalidated-hotfix.md`.
- O metadata de endpoint estava correto: `openai-response` para `/v1/responses` e `openai` para `/v1/chat/completions`.
- A falha real era OAuth upstream do channel 5: `access_token` invalidado e `refresh_token` invalidado.
- O hotfix copiou apenas `access_token` valido do Codex CLI; isso evita mexer no refresh do CLI, mas expira em `2026-07-17T11:04:04Z` e nao autorrenova.

### Backend OAuth

- `controller/codex_oauth.go` ja possui `StartCodexOAuthForChannel` e `CompleteCodexOAuthForChannel`.
- Essas funcoes validam channel type `57`, criam state/verifier PKCE, trocam code por token, extraem `account_id` e gravam OAuth JSON no `channels.key`.
- `router/channel-router.go` expoe apenas `POST /api/channel/:id/codex/refresh`; as rotas start/complete por channel nao estao registradas.
- `service/codex_oauth.go` troca authorization code e refresh token no endpoint OAuth, mas hoje perde detalhes do erro upstream e retorna mensagens genericas como `codex oauth refresh failed: status=400`.
- `service/codex_credential_refresh.go` exige `refresh_token`; logo o hotfix atual access-token-only nunca sera renovado por esse caminho.
- `service/codex_credential_refresh_task.go` roda a cada 10 minutos e renova quando faltam menos de 24h, mas nao faz probe de validade upstream antes da expiracao local.

### Runtime Codex

- `relay/channel/codex/adaptor.go` usa o OAuth JSON como `info.ApiKey`, exige `access_token` e `account_id`, injeta `Authorization`, `chatgpt-account-id`, `OpenAI-Beta` e `originator`.
- `relay/responses_handler.go` chama `service.RelayErrorHandler` para status != 200. O erro upstream passa como OpenAI error generico; nao ha normalizacao Codex-specific.
- `service/error.go` preserva erro estruturado OpenAI quando o corpo tem `{"error":{...}}`; isso e bom para preservar `token_invalidated`, mas insuficiente para separar causa operacional upstream versus auth interna do Router.
- `controller/codex_usage.go` tenta refresh em 401/403 quando ha `refresh_token`, mas ignora o `refreshErr` se o refresh falhar e mostra somente status upstream generico.
- `service/codex_catalog.go` tambem tenta refresh em discovery/probe, mas se o refresh falha retorna o erro original, sem registrar health visivel.

### UI de channel

- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx` ainda mostra o bloco `API Access` completo para `currentType === 57`.
- A condicao do `Base URL` generico e `![3, 8, 22, 36, 45].includes(currentType)`, entao tipo `57` ainda recebe `Base URL` e o copy "Do not add /v1".
- O `FormField name='key'` tambem renderiza para tipo `57`, com label `API Key *`, textarea, reveal/copy current key e descricoes genericas.
- Ha apenas `Refresh credential` em `currentType === 57`; nao ha `Regenerate credential`.
- `web/default/src/features/channels/api.ts` possui `refreshCodexCredential`, mas nao possui APIs para metadata, start/complete regeneration ou probe.
- `web/default/src/features/channels/constants.ts` ainda mapeia type `57` como `ChatGPT Subscription (Codex)`, apesar dos requirements exigirem `OpenAI - Codex` como label canonico.
- `web/default/src/i18n/locales/pt.json` ja contem `Refresh credential` -> `Atualizar credencial`, mas nao contem strings para `Regenerar credencial`, health/probe ou erro upstream auth.

### Fork-sync

- `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/sync.yaml` ja protege `controller/codex_*.go`, `service/codex_*.go`, `relay/channel/codex/`, `router/api-router.go`, `web/default/src/features/channels/`, `docs/` e `.planning/`.
- A etapa de CI em `.github/workflows/sync.yml` adiciona alguns extras dinamicamente, mas nao lista explicitamente `web/default/src/features/channels/`; precisa conferir a configuracao efetiva para nao depender so do arquivo externo.
- `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/UPSTREAM-SYNC-GUARDS.md` lista Codex/Responses/embeddings, mas ainda nao descreve o contrato UI/OAuth regeneration/erro upstream auth.

## Conclusao de Implementacao

### Backend recomendado

1. Reaproveitar `StartCodexOAuthForChannel` e `CompleteCodexOAuthForChannel`, mas registrar rotas admin sob `/api/channel/:id/codex/regenerate/start` e `/api/channel/:id/codex/regenerate/complete`.
2. Adicionar `GET /api/channel/:id/codex/credential` para retornar metadata sanitizada:
   - `authenticated`
   - `has_refresh_token`
   - `expires_at`
   - `last_refresh`
   - `account_id`
   - `email`
   - `last_probe_at`
   - `last_probe_status`
   - `last_upstream_auth_error`
   - `requires_regeneration`
3. Adicionar `POST /api/channel/:id/codex/probe` ou incorporar probe no metadata com flag explicita. Preferencia: endpoint separado para evitar chamada upstream involuntaria em todo load do drawer.
4. Criar erro tipado para OAuth upstream em `service/codex_oauth.go`, preservando `status`, `error`, `error_description` e `body_preview` sanitizado.
5. Criar classificador Codex upstream auth:
   - `token_invalidated`
   - `refresh_token_invalidated`
   - `invalid_api_key`
   - 401/403 sem codigo claro
6. Para relay, normalizar erro Codex em `relay/responses_handler.go` apos `RelayErrorHandler` quando `info.ApiType == constant.APITypeCodex` ou `info.ChannelType == constant.ChannelTypeCodex`.
7. Para admin refresh/usage/catalog, retornar/registrar mensagens operacionais claras quando o refresh token esta ausente ou invalidado.
8. Persistir health nao secreto em campo JSON ja existente, preferencialmente `channels.setting`/settings, para evitar migration cross-DB.

### UI recomendada

1. Criar `isCodexChannel = currentType === 57`.
2. Excluir `57` da renderizacao de Base URL generico.
3. Nao renderizar o `FormField name='key'` generico para Codex.
4. Renderizar card Codex OAuth especifico com:
   - status OAuth
   - expiration
   - refresh token presente/ausente
   - last refresh
   - last probe
   - ultimo erro upstream
   - `Atualizar credencial`
   - `Regenerar credencial`
5. A acao `Regenerar credencial` deve abrir modal com:
   - botao para iniciar OAuth e abrir authorize URL em nova aba
   - campo para colar URL final/codigo quando o browser cair em `localhost:1455/auth/callback`
   - instrucoes PT-BR para fluxo Brave/Chrome logado
   - aviso de que tokens nunca sao exibidos
6. Break-glass access-token-only deve ser documentado fora do fluxo feliz e sinalizado como temporario/sem autorrenovacao.

### Validacao recomendada

- Testes Go unitarios:
  - parse/classificacao de erro OAuth `refresh_token_invalidated`
  - metadata sanitizada sem tokens
  - rotas start/complete/probe/refresh com channel type errado, missing refresh_token e sucesso mockado
  - relay Codex 401 `token_invalidated` vira erro upstream-auth especifico
- Testes frontend:
  - type `57` nao renderiza `Base URL`
  - type `57` nao renderiza `API Key *`/reveal current key
  - type `57` renderiza `Atualizar credencial` e `Regenerar credencial`
- Build/test via CPU wrapper:
  - `./scripts/podman-admin.sh profile-run -- bash -lc 'GOCACHE=$(mktemp -d) go test ./controller ./service ./relay ./relay/channel/codex -count=1'`
  - `./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun run i18n:sync && bun run typecheck && bun run build'`
- Smoke live:
  - backup SQL do channel 5 antes de qualquer escrita
  - regenerar credencial propria do Router
  - validar refresh manual
  - validar `/v1/chat/completions` non-stream e stream
  - validar `/v1/responses` stream
  - validar `/v1/models`
  - validar API key interna invalida separadamente de token upstream invalido

## Riscos

- O redirect OAuth do client Codex usa `http://localhost:1455/auth/callback`; o Browser pode terminar em erro de conexao se nao houver listener local, mas a URL final ainda contem `code` e `state` para completar no backend.
- Automatizar captura via Chrome DevTools depende do Brave/Chrome estar rodando com alvo acessivel; deve ser conveniencia operacional, nao requisito unico.
- Salvar health no `setting` do channel evita migration, mas precisa preservar settings existentes e nao sobrescrever configuracoes do operador.
- Probes upstream demais podem consumir quota ou ativar rate limit; devem ser manuais, agendados com parcimonia, ou reutilizar discovery/probe ja existente.
