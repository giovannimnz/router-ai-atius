---
status: testing
phase: 21-feat-pt-native-pr
source:
  - 21-01-SUMMARY.md
  - 21-02-SUMMARY.md
  - 21-03-SUMMARY.md
  - 21-04-SUMMARY.md
  - 21-05-SUMMARY.md
started: "2026-07-05T03:43:00-03:00"
updated: "2026-07-05T10:42:59-03:00"
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

number: 1
name: Lane limpa de implementação upstream
expected: |
  A lane de implementação da Fase 21 é `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`, baseada no commit atual de `upstream/main` `5fc35e28a253bd5a5656c177aea1bd121231398f`, e o diff contém apenas arquivos nativos de localização PT, wiring, testes e relatórios de locale. Nenhum arquivo de planning, runtime, provider, banco de dados, Podman ou conteúdo exclusivo do fork aparece no diff da branch upstream.
awaiting: user response

## Tests

### 1. Lane limpa de implementação upstream
expected: A lane de implementação da Fase 21 é `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`, baseada no commit atual de `upstream/main` `5fc35e28a253bd5a5656c177aea1bd121231398f`, e o diff contém apenas arquivos nativos de localização PT, wiring, testes e relatórios de locale. Nenhum arquivo de planning, runtime, provider, banco de dados, Podman ou conteúdo exclusivo do fork aparece no diff da branch upstream.
result: [pending]

### 2. Comportamento de idioma português no backend
expected: O backend reconhece `pt`, `pt-BR` e `pt_BR` como português, retorna traduções em português para mensagens como parâmetros inválidos, mantém fallback para inglês em idiomas não suportados e preserva paridade completa de chaves e placeholders com o locale backend em inglês.
result: [pending]

### 3. Experiência em português no frontend default
expected: Em `web/default`, `Português` aparece nas opções compartilhadas de idioma da interface, a seleção usa os fluxos existentes do seletor de idioma/preferência de perfil, `pt`, `pt-BR` e `pt_BR` normalizam para `pt`, e o `pt.json` do default fica com zero chaves faltando, extras ou não traduzidas, mantendo paridade de placeholders.
result: pass
reported: "Finalizado então? Ainda nao apresenta portugues, será q precisa rebuildar o container podman? Screenshot de `https://router.atius.com.br/keys` mostra seletor de idioma sem Português."
severity: major
fixed: "2026-07-05T10:42:59-03:00"
evidence:
  - "Imagem `ghcr.io/giovannimnz/router-ai-atius:latest` rebuildada via `./scripts/podman-admin.sh build` com limite `cpuset=0-1`/`cpu_quota=200000`."
  - "Smoke temporário local inicializou `i18n initialized with languages: zh-CN, zh-TW, en, pt` e serviu bundle contendo `Português`."
  - "Produção reiniciada via `systemctl --user restart container-router-ai-atius.service`; container `router-ai-atius` roda image ID `19e8cb4c2676d635cc484c8f5d65fd2c6416afe17a2011952694b290d94ab115`."
  - "Produção registrou `i18n initialized with languages: zh-CN, zh-TW, en, pt` às 2026-07-05 10:42:08-03."
  - "`https://router.atius.com.br/keys?codex_pt_validate=202607051043` serve `/static/js/index.41bf9f4d01.js`, com ocorrências de `Português` e `code:\"pt\"`."

### 4. Experiência em português no frontend classic
expected: Em `web/classic`, `Português` aparece no seletor de idioma do header e nas preferências de perfil, a seleção usa os caminhos existentes de alteração/persistência, `pt`, `pt-BR` e `pt_BR` normalizam para `pt`, e o `pt.json` do classic tem paridade completa de chaves e placeholders com o inglês.
result: [pending]

### 5. Prontidão do handoff upstream
expected: O handoff identifica a issue `#2924`, a PR aberta `#5801` e as PRs históricas fechadas `#5238`/`#5245`; o rascunho do corpo da PR usa o template upstream, inclui disclosure de assistência por IA, lista evidências de validação local e evita conteúdo operacional sensível ou exclusivo do fork.
result: [pending]

## Summary

total: 5
passed: 1
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps

- truth: "Em `web/default`, `Português` aparece nas opções compartilhadas de idioma da interface, a seleção usa os fluxos existentes do seletor de idioma/preferência de perfil, `pt`, `pt-BR` e `pt_BR` normalizam para `pt`, e o `pt.json` do default fica com zero chaves faltando, extras ou não traduzidas, mantendo paridade de placeholders."
  status: fixed
  reason: "User reported: Finalizado então? Ainda nao apresenta portugues, será q precisa rebuildar o container podman? Screenshot de `https://router.atius.com.br/keys` mostra seletor de idioma sem Português."
  severity: major
  test: 3
  root_cause: "Produção rodava imagem `ghcr.io/giovannimnz/router-ai-atius:latest` criada em 2026-07-05 08:53:28-03 sem os arquivos PT da Fase 21; o log antigo inicializava i18n apenas com zh-CN, zh-TW e en, e os assets públicos não continham `Português`."
  artifacts:
    - path: "web/default/src/i18n/config.ts"
      issue: "Corrigido: runtime inclui `pt` em `supportedLngs`."
    - path: "web/default/src/i18n/languages.ts"
      issue: "Corrigido: runtime expõe `Português` nas opções compartilhadas."
  fixed_by:
    - "Alterações PT portadas da worktree `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream` para a branch runtime do fork."
    - "Backend, frontend default e frontend classic validados localmente."
    - "Imagem Podman rebuildada e produção reiniciada com limites de CPU/memória do wrapper."
    - "HTML e bundle públicos validados após restart."
  debug_session: "inline-2026-07-05-runtime-pt-missing"
