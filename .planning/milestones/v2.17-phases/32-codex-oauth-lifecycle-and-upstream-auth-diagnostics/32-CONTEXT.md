# Phase 32 Context — Codex OAuth lifecycle and upstream auth diagnostics

## Origem

Em 2026-07-10, a sessao Codex `019f3ef0-bf50-7e93-9834-af51f040c1db` e a nota Obsidian `60-LOGS/2026-07-10-router-ai-atius-codex-token-invalidated-hotfix.md` registraram que o endpoint do channel 5 `OpenAI - Codex` estava correto, mas a credencial OAuth upstream estava invalidada:

- `/v1/chat/completions` e `/v1/responses` falhavam por `401 token_invalidated` vindo do upstream Codex.
- `access_token` e `refresh_token` do channel 5 estavam invalidados; o refresh retornava `refresh_token_invalidated`.
- O hotfix seguro copiou apenas um `access_token` valido do Codex CLI, sem copiar `refresh_token`, para nao quebrar nem rotacionar a auth do Codex CLI.
- O hotfix expira em `2026-07-17T11:04:04Z`; a solucao definitiva e uma credencial OAuth propria do Router, com `refresh_token` independente.

## Problema

O canal Codex ainda apresenta superficies genericas e sinais de saude que induzem erro operacional:

- O editor de canal tipo `57` ainda mostra `Base URL` generico e descricao de endpoint/proxy, mas o Codex do fork tem um unico upstream canonico.
- O editor ainda mostra textarea/reveal de `API Key`, apesar de a credencial real ser OAuth JSON sensivel.
- Existe `Atualizar credencial`, mas nao existe `Regenerar credencial`.
- `Atualizar credencial` depende de `refresh_token`; com o hotfix temporario sem `refresh_token`, ele nunca pode autorrenovar.
- A UI pode parecer saudavel quando `expired` esta no futuro, mesmo se o upstream ja invalidou o token.
- O erro retornado ao cliente ainda parece uma falha OpenAI generica e nao separa auth interna do Router de auth upstream do provider Codex.

## Decisoes

- `OpenAI - Codex` tipo `57` deve ter UX propria: sem `Base URL` generico, sem textarea/reveal de API key, e com status OAuth/health especifico.
- `Atualizar credencial` significa renovar com `refresh_token` existente. `Regenerar credencial` significa iniciar novo Authorization Code + PKCE e gravar uma credencial propria do Router.
- O fluxo de regeneracao deve ser browser-assisted: abrir a URL de autorizacao no Brave/Chrome logado, capturar ou colar a URL final `localhost:1455/auth/callback` com `code` e `state`, completar no backend e gravar `access_token`, `refresh_token`, `account_id`, `email`, `expired` e timestamps sem imprimir segredos.
- A automacao via Chrome DevTools/Brave e desejavel para operador local, mas deve ter fallback manual seguro. O codigo do Router nao deve depender de ferramenta Codex ou de segredo no chat.
- Copiar `access_token` do Codex CLI continua sendo break-glass temporario, sempre sem `refresh_token` e sempre marcado como nao autorrenovavel.
- Validade do canal Codex deve combinar expiracao local, presenca de `refresh_token`, ultimo probe upstream e ultimo erro upstream. Expiracao futura sozinha nao prova saude.
- Erros upstream `token_invalidated`, `refresh_token_invalidated`, `invalid_api_key` ou 401/403 equivalentes devem produzir codigo operacional especifico de upstream auth, distinto de API key invalida do Router.
- As mudancas precisam ficar protegidas no fork-sync local e no ajuste de CI que reescreve protected paths.

## Escopo

- Backend admin API para metadata, refresh, regenerate start/complete e health/probe Codex.
- Servico Codex OAuth com erro tipado preservando `status`, `error`, `error_description` e sem logar tokens.
- Normalizacao de erro do relay para Codex upstream auth invalida.
- UI do drawer de channel para tipo `57`, em PT-BR, com acoes `Atualizar credencial` e `Regenerar credencial`.
- Docs operacionais PT-BR e guardas de fork-sync.
- Testes unitarios, integracao controlada, build com limite de CPU e smoke live apos deploy.

## Fora de Escopo

- Reativar embeddings Codex sem quota/licenca upstream.
- Copiar `refresh_token` do Codex CLI.
- Alterar a identidade protegida upstream `new-api`/`QuantumNous`.
- Fazer cutover k3s das Phases 29/30.

## Validacao Obrigatoria

- `git diff --check` para todos os arquivos alterados.
- Testes Go focados via `./scripts/podman-admin.sh profile-run`, nunca `go test ./...` direto.
- `web/default` i18n/typecheck/build via `./scripts/podman-admin.sh profile-run`.
- Validacao negativa de auth interna do Router com API key invalida.
- Validacao negativa de auth upstream Codex em teste controlado sem quebrar channel 5.
- Validacao positiva local e publica de `/v1/chat/completions` non-stream, `/v1/chat/completions` stream, `/v1/responses` stream e `/v1/models`.
- Backup SQL do channel 5 antes de qualquer escrita live.
- Registro Obsidian e, se aplicavel, GBrain sem segredos.
- Commit e push apenas depois da bateria verde.

## Learning Seed

- Incidente confirmou que endpoint/catálogo correto nao prova credencial valida.
- `expired` no JSON local nao e fonte de verdade quando o upstream invalida token antes do prazo.
- Refresh silencioso sem expor `refresh_token_invalidated` aumenta MTTR.
- UI generica de channel aumenta risco de operador mexer em `base_url`/key errados para Codex.
- O fallback por Codex CLI resolve emergencia, mas cria uma janela de expiracao curta e sem autorrenovacao.

## Proximo Comando Esperado

Depois que esta fase for planejada e revisada: `$gsd-execute-phase 32`.

Depois da execucao e dos SUMMARYs: `$gsd-extract-learnings 32`.
