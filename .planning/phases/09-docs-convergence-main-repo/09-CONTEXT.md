# Phase 09: docs-convergence-main-repo - Context
**Gathered:** 2026-06-07
**Status:** Ready for planning
**Source:** user directive + production incidents + current fork-sync/runtime state

<domain>
## Phase Boundary
Convergir o docs site do Atius Router para dentro do repo principal `router-ai-atius`, parando de depender do runtime/source checkout em `/home/ubuntu/docker/Atius/atius-router-docs`.

Entregável desta fase:
- desenho final da árvore integrada em `router-ai-atius/docs/atius-router-docs/`
- plano executável de cutover com rollback
- plano de gestão pelo `~/GitHub/omni-srv-admin`
- regra explícita para o destino do remote separado `atius-router-docs`

Fora do escopo imediato:
- executar a remoção definitiva do remote sem backup
- reescrever conteúdo MDX completo
- mexer no milestone ativo v2.14 Codex além do necessário para evitar colisão operacional
</domain>

<decisions>
## Implementation Decisions

### Repository topology
- D-01: O source canônico da documentação deve viver dentro de `router-ai-atius` no path `docs/atius-router-docs/`.
- D-02: O mecanismo alvo é `git submodule` para manter identidade própria do docs codebase sem continuar tratando o runtime como repo standalone solto no filesystem.

### Branding and UX
- D-03: A documentação integrada deve permanecer Atius-first, com logo SVG válida, PT-BR preservado e sem vazamento visível de branding `New API` nas superfícies públicas do fork.
- D-04: A preferência é continuar usando SVG em todos os logos; PNG só entra como fallback técnico se um consumer quebrar com SVG.

### Operations and ownership
- D-05: `~/GitHub/omni-srv-admin` será a autoridade operacional para sync, rebrand, patch, deploy, rollback e eventual arquivamento do remote separado.
- D-06: O cutover não pode depender de `next dev` como estado final. O plano precisa fechar um runtime de produção estável.

### Safety and rollout
- D-07: O remote separado `atius-router-docs` só pode ser arquivado/removido depois de cutover validado, rollback documentado e automação substituta confirmada.
- D-08: O fluxo precisa preservar as rotas `/pt/`, `/pt/docs/`, `/pt/docs/skills/` e `/en/` sem regressão visual ou de asset.

### Claude's Discretion
- Estrutura exata do wrapper de runtime (`systemd`, container, ou ambos durante transição)
- Nome final do projeto no `omni-srv-admin` (`atius-router-docs` legado vs novo slug convergido)
- Estratégia de rollout em 1 ou 2 passos, desde que rollback fique explícito
</decisions>

<canonical_refs>
## Canonical References
**Downstream agents MUST read these before planning or implementing.**

### Current incidents and live state
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/60-LOGS/64-Worklogs-Agrupados/2026-06-07-atius-router-docs-pt-en-layout-fix.md` — estado atual, runtime temporário, rotas validadas
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/61-Incidents/2026-06-07-atius-router-docs-invalid-svg-logo.md` — causa raiz da logo quebrada e fix aplicado
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/61-Incidents/2026-06-07-atius-router-docs-podman-build-no-space.md` — por que o container/build de produção ficou inviável no estado atual
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/61-Incidents/2026-06-04-docs-via-new-api-docs-v1.md` — origem do cutover para `new-api-docs-v1`

### Current repo and infra state
- `/home/ubuntu/docker/Atius/router-ai-atius/.planning/ROADMAP.md` — phase source of truth
- `/home/ubuntu/docker/Atius/router-ai-atius/.planning/REQUIREMENTS.md` — DOCS-01/02/03
- `/home/ubuntu/docker/Atius/router-ai-atius/.planning/PROJECT.md` — arquitetura atual e constraints do fork
- `/home/ubuntu/docker/Atius/router-ai-atius/21.03-Decisoes-Arquitetura/2026-06-06-apache-proxy-nextjs-docs-licoes-phase-04.md` — lições de Apache + Next.js docs
- `/home/ubuntu/.config/systemd/user/atius-router-docs.service` — runtime temporário atual (`next dev`)
- `/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf` — roteamento ativo para docs em `127.0.0.1:3003`

### omni / fork-sync ownership
- `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router-docs/sync.yaml` — protected_paths/globs atuais
- `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router-docs/scripts/fork-sync-docs.sh` — pipeline atual de sync/build/deploy
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/61-Incidents/2026-06-04-fork-sync-atius-router-docs.md` — setup do fork-sync do docs standalone
- `/home/ubuntu/GitHub/obsidian-vault/ideaverse/20-PROJETOS/21-PROJETOS-ATIVOS/omni-srv-admin/fork-sync-submodule.md` — padrão de ownership do submodule fork-sync
</canonical_refs>

<specifics>
## Specific Ideas
- Target path preferido pelo user: `~/docker/Atius/router-ai-atius/docs/atius-router-docs`
- Quer encerrar o modelo de repo/runtime separado
- Quer docs como parte real de Atius Router
- Quer gestão e patches centralizados em `~/GitHub/omni-srv-admin`
- Quer validar visualmente com chrome-devtools como gate real
</specifics>

<deferred>
## Deferred Ideas
- Remoção física/definitiva do repo standalone local só depois de rollback e cutover estável
- Higienização completa de qualquer referência interna a `New API` fora das superfícies públicas pode virar follow-up se não bloquear o cutover principal
</deferred>

---
*Phase: 09-docs-convergence-main-repo*
*Context gathered: 2026-06-07 via inline planning path*
