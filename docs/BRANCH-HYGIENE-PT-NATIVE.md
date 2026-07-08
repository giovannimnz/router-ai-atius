# Branch hygiene - PT native e planning lanes

Data de referência: 2026-07-08.

Este documento existe para encerrar a ambiguidade entre branches, worktrees e
trilhas de planning que se acumularam durante a limpeza do PR PT-BR e das fases
24-27.

## Estado final após Phase 28

Estado verificado em 2026-07-08:

- worktree local mantida: `/home/ubuntu/GitHub/containers/router-ai-atius`
- branch local mantida: `main`
- branches remotas mantidas:
  - `origin/main`
  - `origin/feat/phase21-pt-native-upstream`
- branches remotas removidas:
  - `origin/feat/pt-native`
  - `origin/feat/pt-native-i18n-clean`

Backup final antes da limpeza:

- `/home/ubuntu/GitHub/containers/router-ai-atius-phase28-wave4-backup-20260708T210137Z`

## Diagnóstico histórico

Antes da Phase 28 existiam múltiplas linhas locais relacionadas a PT-BR:

- `feat/pt-native`
- `feat/phase21-pt-native-upstream`
- `feat/brazilian-portuguese-localization`
- `feat/pt-native-i18n-clean` (remoto/local quando presente)

Além disso, existia uma worktree local chamada `main` atrasada em relação a
`origin/main`.

## Fonte de verdade por trilha

### Main do fork

Fonte de verdade:

- `origin/main`

Worktree local final:

- `/home/ubuntu/GitHub/containers/router-ai-atius`
- branch `main`
- tracking `origin/main`

### Phase 21 handoff upstream

Fonte de verdade operacional remota:

- branch `origin/feat/phase21-pt-native-upstream`

Estado esperado:

- branch baseada em `upstream/main`
- commit limpo PT-BR preservado remotamente
- usar esta lane para:
  - commit limpo
  - push para o fork
  - PR novo contra `QuantumNous/new-api`

### Branch de referência de tradução

Fonte de referência histórica/material:

- backup da Phase 28 quando for preciso recuperar material histórico

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
- `origin/feat/phase21-pt-native-upstream` = handoff limpo da `Phase 21`
- branches locais PT antigas = removidas após backup
- branches remotas PT redundantes = removidas

Antes de qualquer push/PR da `Phase 21`, validar:

```bash
git ls-remote --heads origin feat/phase21-pt-native-upstream
git fetch origin feat/phase21-pt-native-upstream
git diff --name-status upstream/main...origin/feat/phase21-pt-native-upstream
git diff --check upstream/main...origin/feat/phase21-pt-native-upstream
```
