---
phase: "09-docs-convergence-main-repo"
status: complete
method: inline
created: 2026-06-07
updated: 2026-06-07
---

# Research — Phase 09 Docs Convergence Main Repo

## Question
Como migrar o docs site para dentro do `router-ai-atius` sem perder:
- branding Atius
- PT-BR
- upstream sync
- runtime estável
- ownership operacional no `omni-srv-admin`

## Findings

### 1. Source topology atual está quebrada conceitualmente
- runtime/source atual: `/home/ubuntu/docker/Atius/atius-router-docs`
- isso força dependência operacional fora do repo principal
- hoje o repo principal já contém artefatos docs em `docs/` e `integration/middleware/docs/openapi.json`, mas não o docs site Next.js completo
- `router-ai-atius` ainda não tem `.gitmodules`

### 2. Ownership operacional já vive no omni
`omni-srv-admin` já contém toda a inteligência crítica do docs standalone:
- `modules/fork-sync/projects/atius-router-docs/sync.yaml`
- `modules/fork-sync/projects/atius-router-docs/atius-router-docs-rebrand.sh`
- `modules/fork-sync/projects/atius-router-docs/scripts/fork-sync-docs.sh`
- `pt-content/docs/pt/**`

Conclusão: o plano não precisa inventar uma nova autoridade. Precisa só redirecionar essa autoridade para a nova topologia.

### 3. Runtime atual não é aceitável como estado final
Estado atual do live fix:
- `~/.config/systemd/user/atius-router-docs.service`
- `WorkingDirectory=/home/ubuntu/docker/Atius/atius-router-docs`
- `ExecStart=... next dev -p 3003 -H 127.0.0.1`

Isto resolveu urgência, mas não é estado de produção.

### 4. Apache já está path-based, não repo-based
Apache só conhece endpoint/porta:
- docs via `127.0.0.1:3003`
- assets críticos e locale routing já foram estabilizados

Conclusão: mover o source path é viável sem refazer o modelo de roteamento inteiro. O corte real está em:
- checkout path
- unit file / container config
- automação do omni

### 5. Logo/asset mostrou por que path + cache precisam ser explícitos
A quebra de `/assets/atius-logo.svg` revelou 2 lições:
- asset inválido pode sobreviver atrás de cache mesmo com fix local
- qualquer migração precisa incluir cache-bust, purge e validação visual real

### 6. Submodule é coerente com o pedido, mas exige disciplina operacional
Vantagens do submodule em `docs/atius-router-docs/`:
- mantém histórico próprio do docs codebase
- deixa claro que docs ainda têm ciclo de sync específico
- permite ao repo principal fixar SHA conhecido

Custos:
- update extra (`git submodule update --init --recursive`)
- scripts do omni precisam operar no path novo
- deploy local precisa garantir checkout do submodule antes do build/serve

### 7. Remote separado não deve ser removido no primeiro passo
Mesmo que o target seja parar de tratá-lo como repo standalone de runtime:
- o remote ainda é a âncora natural do submodule
- deletar cedo demais destrói rollback fácil
- fase deve terminar com regra explícita: arquivar/remover só após cutover estável + rollback validado

## Recommended Strategy

### Recommendation
Migrar em 3 movimentos:
1. **Convergência de source**
   - adicionar `docs/atius-router-docs/` como submodule no `router-ai-atius`
   - portar runtime para ler desse path
2. **Convergência operacional**
   - reescrever `omni-srv-admin` para apontar para o novo path
   - separar sync/build/deploy/rollback por passos claros
3. **Convergência de governança**
   - depois do live validado, decidir se o remote separado vira:
     - repo arquivado + ainda submodule
     - repo ativo privado
     - espelho transitório até subtree futura

## Validation Architecture

### Pre-cutover
- backup do unit file atual
- backup do Apache vhost
- backup do checkout docs standalone atual
- snapshot da árvore do submodule e do SHA fixado

### During cutover
- `git submodule status`
- `systemctl --user status` ou container status do runtime novo
- `curl -I` em:
  - `/pt/`
  - `/pt/docs/`
  - `/pt/docs/skills/`
  - `/en/`
- asset checks:
  - `/assets/atius-logo.svg`
  - `/assets/atius-logo.svg?v=<cache-bust>`

### Visual gate
chrome-devtools / browser validation obrigatória:
- `naturalWidth > 0` para logo
- PT-BR em `/pt/` e `/pt/docs/skills/`
- sem regressão de TOC/sidebar/header/footer

## Risks
- mover para submodule sem ajustar deploy local → runtime sobe sem checkout
- manter `next dev` por tempo demais → produção frágil
- apagar remote separado cedo → rollback ruim
- deixar scripts do omni antigos apontando para path velho → drift operacional

## Outcome for planning
A fase deve ser planejada em 3 planos:
1. topology + cutover contract
2. repo/runtime migration
3. omni ownership + remote decommission rule
