# Phase 32 UI-SPEC — Codex OAuth lifecycle

## Objetivo de UI

Substituir a experiencia generica de credenciais do channel type `57` por uma interface operacional especifica para `OpenAI - Codex`, reduzindo risco de operador editar `base_url`/key errados e deixando claro quando a credencial precisa de refresh, regeneration ou break-glass.

## Superficies Afetadas

- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx`
- `web/default/src/features/channels/api.ts`
- `web/default/src/features/channels/constants.ts`
- `web/default/src/i18n/locales/pt.json`

## Contrato Visual

- Preservar o design system atual do drawer: cards `border-border/60`, `bg-muted/10`, `Alert`, `Badge`, `Button`, `FormDescription`.
- Nao introduzir nova paleta, fonte ou layout fora do padrao existente.
- O bloco Codex deve ficar dentro da secao de credenciais, mas com titulo proprio `Credencial OAuth Codex`.
- Usar badges simples:
  - `Autenticado`
  - `Precisa regenerar`
  - `Sem refresh_token`
  - `Probe OK`
  - `Erro upstream`
- Erros de upstream auth devem usar alerta amber/red sem mostrar token, request body ou cabecalhos.

## Copy PT-BR

Strings obrigatorias:

- `Credencial OAuth Codex`
- `Atualizar credencial`
- `Regenerar credencial`
- `Gerar nova credencial OAuth propria do Router`
- `Esta credencial nao possui refresh_token e nao pode ser renovada automaticamente. Regenerar e a correcao definitiva.`
- `A expiracao local ainda esta no futuro, mas o ultimo probe upstream falhou. Regenerar a credencial.`
- `Cole a URL final do callback ou o par code#state.`
- `Os tokens nunca serao exibidos nesta tela.`
- `Fallback temporario: access_token do Codex CLI, sem refresh_token e sem autorrenovacao.`

## Comportamento

### Tipo diferente de 57

- Manter comportamento atual.

### Tipo 57

- Nao renderizar o campo generico `Base URL`.
- Nao renderizar a descricao generica de endpoint/proxy nem a orientacao sobre `/v1`.
- Nao renderizar o textarea generico `API Key *`.
- Nao renderizar reveal/copy da key atual.
- Renderizar card de metadata OAuth sanitizada.
- Renderizar `Atualizar credencial` quando houver channel id.
- Renderizar `Regenerar credencial` quando houver channel id.
- Se nao houver channel id, mostrar que regeneracao exige canal salvo/canonico e nao oferecer batch creation.

## Fluxo Regenerar Credencial

1. Usuario clica `Regenerar credencial`.
2. UI chama start endpoint e recebe `authorize_url`.
3. UI abre authorize URL em nova aba/janela.
4. Usuario autentica no Brave/Chrome ja logado em ChatGPT/OpenAI.
5. Browser termina em `http://localhost:1455/auth/callback?...`; se nao houver listener local, a URL da barra ainda deve ser usada.
6. UI modal aceita URL final completa ou `code#state`.
7. UI chama complete endpoint.
8. UI recarrega metadata e mostra status sem tokens.

## Acessibilidade e Estados

- Botoes devem ter texto visivel, nao apenas icone.
- Loading state deve usar `Loader2` e desabilitar a acao correspondente.
- Erros devem aparecer em toast e no card quando retornados pela API.
- Nenhum estado deve exigir copiar token para clipboard.

## Regressao Proibida

- O texto PT-BR "URL base da API personalizada..." nao pode aparecer no bloco Codex.
- O texto "Deixe vazio para usar o padrao" nao pode aparecer no bloco Codex.
- `API Key *`, `Current key`, `Reveal key` e `Copy` nao podem aparecer para type `57`.

## Validacao UI

- Teste ou smoke DOM deve provar que type `57` nao contem `Base URL`, `API Key *`, `Current key`, `Reveal key` ou copy de key.
- Teste ou smoke DOM deve provar que type `57` contem `Atualizar credencial` e `Regenerar credencial`.
- `bun run i18n:sync`, `bun run typecheck` e `bun run build` devem rodar via `./scripts/podman-admin.sh profile-run`.
