---
slug: profile-dashboard-overflow
status: fixed
trigger: "Analisar e corrigir todos os bugs visuais das telas /profile e /dashboard/overview mostrados nas capturas anexadas."
created: 2026-08-21T05:00:00-03:00
updated: 2026-08-21T07:25:00-03:00
---

# Debug: overflow responsivo no perfil e dashboard

## Symptoms

- expected: cards, textos e botoes devem permanecer integralmente visiveis em desktop e mobile, sem overflow horizontal, truncamento acidental ou scroll interno indevido.
- actual: no card de autenticacao em duas etapas, os dois botoes de acao extrapolam a largura do card; no painel de acoes recomendadas, cards e descricoes sao cortados pela lateral direita e aparece scroll interno.
- error: nao ha erro de JavaScript informado; a falha e visual e reproduzida nas capturas em tema escuro.
- timeline: observado em producao em 2026-08-21; estado anterior sem regressao visual nao foi datado.
- reproduction: abrir https://router.atius.com.br/profile e https://router.atius.com.br/dashboard/overview em viewport estreito ou na composicao desktop exibida nas capturas.

## Constraints

- Preservar a linguagem visual existente do frontend default.
- Corrigir a causa responsiva, sem apenas esconder overflow ou truncar conteudo relevante.
- Validar tema escuro, desktop e mobile.
- Preservar todas as alteracoes preexistentes da working tree compartilhada.
- Builds e testes pesados devem usar o wrapper com limite de 20% da CPU total.

## Current Focus

- resolved: true
- conclusion: "o overflow vinha de min-content implicito e dos defaults `whitespace-nowrap`/`shrink-0` do Button em colunas estreitas; wrappers sem `min-w-0` ampliavam a trilha. Wrap, shrink e breakpoints foram corrigidos e a matriz headless passou 10/10."
- production_state: "a imagem `prod-20260821-ui-embedding-ha` serve o mesmo bundle validado, sem Devtools em producao e com as duas rotas respondendo HTTP 200."
- next_action: none

## Evidence

- `web/default/src/components/ui/button.tsx` aplica `inline-flex`, `shrink-0` e `whitespace-nowrap` por padrao a todo `Button`.
- `web/default/src/features/profile/components/two-fa-card.tsx` colocava os botoes de acao em linha desde `sm` (`sm:flex-row`) e com `flex-1`, mesmo dentro da coluna lateral estreita do perfil; como os botoes herdavam `whitespace-nowrap`, os labels longos nao podiam quebrar linha nem encolher.
- `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx` renderizava `QuickActionItem` sem `w-full`, sem `min-w-0` e sem sobrescrever `whitespace-nowrap`; o botao ficava orientado a largura do proprio conteudo, e as descricoes nao podiam quebrar linha corretamente.
- Os wrappers de grade/motion do perfil e do overview tambem nao garantiam `min-w-0` nas colunas/cards laterais, o que permitia que o tamanho minimo do conteudo empurrasse a largura da trilha e produzisse corte lateral/scroll.

## Eliminated

- Hipotese de erro global de tema escuro: nao ha classe especifica de dark mode causando overflow; o problema vem de layout e largura minima.
- Hipotese de regressao de JavaScript/runtime: nao ha evidencia de erro JS; os sintomas saem diretamente das classes responsivas e defaults de `Button`.

## Resolution

- root_cause: "Componentes de CTA reutilizavam o `Button` com defaults de `whitespace-nowrap`/`shrink-0` em colunas estreitas, enquanto alguns wrappers de grid/flex nao tinham `min-w-0`; isso impedia quebra de linha e encolhimento, gerando overflow horizontal e corte lateral."
- fix: "No perfil, a area de acoes do 2FA passou a empilhar botoes por padrao, so volta a linha em `2xl`, e os botoes agora aceitam quebra de linha. No overview, `QuickActionItem` passou a ocupar `w-full`, aceitar quebra de linha e nao truncar descricao. Tambem adicionei `min-w-0` aos wrappers laterais relevantes no perfil e no overview."
- verification:
  - "Inspecao estatica do layout confirma que os pontos antes bloqueados por `whitespace-nowrap` agora permitem wrap e shrink."
  - "`./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun run typecheck'` concluiu com sucesso em 2026-08-21."
  - "Lint focal dos arquivos alterados concluiu sem erros; o uso anterior de indice como chave no preview tambem foi removido."
  - "QA headless em tema escuro passou em 10/10 combinacoes: `/profile` e `/dashboard/overview` a 375x812, 500x900, 768x1024, 1280x800 e 1440x900."
  - "Em todas as combinacoes, `document.documentElement.scrollWidth - clientWidth` foi zero e os CTAs monitorados permaneceram dentro do viewport e de seus cards."
  - "Resultados e capturas pos-correcao foram preservados em `runtime/evidence/ui-20260821/`."
  - "A imagem final de producao `localhost/router-ai-atius:prod-20260821-ui-embedding-ha` (`5f8a7b71461e`) serve `/profile` e `/dashboard/overview` com HTTP 200; o bundle validado na matriz headless nao recebeu alteracoes visuais posteriores."
- files_changed:
  - "web/default/src/features/profile/index.tsx"
  - "web/default/src/features/profile/components/two-fa-card.tsx"
  - "web/default/src/features/profile/hooks/use-two-fa.ts"
  - "web/default/src/features/dashboard/components/overview/overview-dashboard.tsx"
  - "web/default/src/i18n/locales/en.json"
  - "web/default/src/i18n/locales/pt.json"
  - ".planning/debug/profile-dashboard-overflow.md"
