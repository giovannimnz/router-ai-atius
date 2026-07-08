# Branch hygiene - PT native e planning lanes

Data de referência: 2026-07-08.

Este documento existe para encerrar a ambiguidade entre branches, worktrees e
trilhas de planning que se acumularam durante a limpeza do PR PT-BR e das fases
24-27.

## Diagnóstico curto

Hoje existem múltiplas linhas locais relacionadas a PT-BR:

- `feat/pt-native`
- `feat/phase21-pt-native-upstream`
- `feat/brazilian-portuguese-localization`
- `feat/pt-native-i18n-clean` (remoto/local quando presente)

Além disso, existe um worktree local chamado `main`, mas ele está atrasado em
relação a `origin/main` e não deve ser tratado como fonte autoritativa.

## Fonte de verdade por trilha

### Main do fork

Fonte de verdade:

- `origin/main`

Não confiar automaticamente no worktree local:

- `/home/ubuntu/GitHub/containers/router-ai-atius-main-exec`

Motivo:

- esse worktree ficou muito atrás de `origin/main`
- ele também carrega mudanças locais fora do fluxo atual

### Phase 21 handoff upstream

Fonte de verdade operacional:

- worktree `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`
- branch `feat/phase21-pt-native-upstream`

Estado esperado:

- branch baseada em `upstream/main`
- mudanças PT-BR ainda locais/uncommitted até o handoff final
- usar esta lane para:
  - commit limpo
  - push para o fork
  - PR novo contra `QuantumNous/new-api`

### Branch de referência de tradução

Fonte de referência histórica/material:

- worktree `/home/ubuntu/GitHub/containers/router-ai-atius-pt-native-clean`
- branch `feat/brazilian-portuguese-localization`
- branch remota/local `feat/pt-native-i18n-clean` quando existir

Uso correto:

- consultar cobertura já pronta
- reaproveitar textos PT-BR
- comparar gaps

Uso incorreto:

- não usar como branch final de handoff upstream sem revalidar escopo/diff

### Branch `feat/pt-native`

Status:

- branch de integração/planning local
- não é a lane limpa de upstream handoff

Problema:

- acumulou planning, docs, fases 24-27 e outras mudanças do fork
- está muito distante de `upstream/main`
- não serve como base limpa para a `Phase 21`

Regra:

- não abrir PR upstream a partir de `feat/pt-native`
- não usar `feat/pt-native` como “verdade” da tradução limpa

## Conclusões práticas

1. `Phase 21` não está “aberta” por falta de implementação; ela está em estado
   de **handoff**.
2. `Phase 22` e `Phase 23` continuam realmente não executadas.
3. A confusão vinha de mistura entre:
   - branch de integração/planning (`feat/pt-native`)
   - lane limpa de upstream (`feat/phase21-pt-native-upstream`)
   - branch/worktree de referência de tradução (`feat/brazilian-portuguese-localization` e `feat/pt-native-i18n-clean`)
   - worktree local `main` desatualizado

## Regra daqui para frente

- `origin/main` = referência do fork
- `feat/phase21-pt-native-upstream` = handoff limpo da `Phase 21`
- `feat/pt-native` = branch de integração/planning local, não de PR upstream
- `feat/brazilian-portuguese-localization` / `feat/pt-native-i18n-clean` = referência de tradução reaproveitável

Antes de qualquer push/PR da `Phase 21`, validar:

```bash
git -C /home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream status --short --branch
git -C /home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream diff --name-status upstream/main...HEAD
git -C /home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream diff --check upstream/main...HEAD
```
