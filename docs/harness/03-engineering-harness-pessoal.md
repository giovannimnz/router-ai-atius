# Engineering Harness Pessoal

## Objetivo

Criar um Harness pessoal e genérico para desenvolvimento de software, separado do QA Automation Harness corporativo.

Este projeto poderá ser usado para:

- desenvolver o próprio Harness;
- desenvolver ferramentas;
- desenvolver integrações;
- trabalhar em projetos pessoais;
- experimentar diferentes agentes/runtimes;
- criar e evoluir skills de engenharia;
- aplicar dogfooding desde o início.

## Relação com o QA Harness

Os dois projetos podem compartilhar conceitos e eventualmente bibliotecas, porém são produtos distintos.

```text
Engineering Harness
  = pessoal
  = genérico
  = experimental
  = múltiplos domínios

QA Automation Harness
  = corporativo
  = focado em QA
  = Playwright/API/pipeline
  = governança empresarial
```

Não misturar memória, credenciais, políticas ou projetos dos dois ambientes.

## Core genérico

O Engineering Harness deve ser baseado em profiles.

Exemplo:

```text
core/
  runtime/
  workflows/
  skills/
  memory/
  dreaming/
  policies/
  ui/

profiles/
  software-engineering/
  qa/
  devops/
  ai-integration/
  research/
```

QA pode existir como profile pessoal, mas isso não transforma este projeto no QA Harness corporativo.

## Dogfooding

Princípio registrado:

> Usar o próprio Engineering Harness para desenvolver e evoluir o próprio Engineering Harness.

Exemplos:

- criar adapter novo;
- detectar padrão recorrente;
- gerar uma nova skill;
- validar uma nova skill;
- atualizar workflow;
- testar migração;
- registrar decisão arquitetural;
- consolidar conhecimento via Dream Mode.

## Runtime Adapter

Mesmo conceito do QA Harness:

```ts
interface AgentRuntime {
  capabilities(): Promise<RuntimeCapabilities>;
  startSession(input: SessionInput): Promise<SessionHandle>;
  send(session: SessionHandle, event: HarnessEvent): Promise<RuntimeEventStream>;
  resume(sessionId: string): Promise<SessionHandle>;
  cancel(sessionId: string): Promise<void>;
}
```

Runtimes possíveis:

```text
Antigravity / agy
Claude Code
Codex
OpenCode
outros
```

## Electron + Node

Direção inicial:

- Node.js para orchestration/core;
- Electron como desktop shell;
- IPC bem definido;
- workers para tarefas longas;
- CLI compartilhando o mesmo core.

Arquitetura:

```text
Electron UI
   |
   v
Harness Core
   |
   +--> Runtime Adapters
   +--> Skill Engine
   +--> Memory Engine
   +--> Dream Engine
   +--> Workflow Engine
```

## Modo CLI

Muito importante não acoplar tudo ao Electron.

Exemplo:

```bash
eng-harness run
eng-harness plan
eng-harness skills
eng-harness memory
eng-harness dream
eng-harness runtime list
```

Isso permite:

- automações;
- CI;
- VM;
- execução remota;
- cron;
- pipelines.

## Memória

Usar a mesma separação conceitual:

- memória declarativa;
- skills operacionais;
- decisões arquiteturais;
- histórico de execução;
- evidências.

GBrain pode funcionar como backend de memória.

## Skills

Skills devem ser reutilizáveis e versionadas.

Exemplos:

```text
node.create-runtime-adapter
electron.secure-ipc
typescript.refactor-service
github.prepare-pr
docker.optimize-image
podman.rootless-runtime
api.openai-compatible-adapter
```

## Dream Mode

Dois ciclos principais:

```text
Memory Dream
Skill Dream
```

O Skill Dream deve aprender com execução real.

Possível extensão pessoal:

```text
Architecture Dream
```

Responsável por analisar:

- decisões repetidas;
- tech debt;
- padrões arquiteturais;
- componentes duplicados;
- oportunidades de abstração;
- bibliotecas que podem virar módulos comuns.

## Lifecycle de conhecimento

```text
raw
 -> candidate
 -> validated
 -> canonical
 -> superseded
 -> archived
```

## Auto-evolução controlada

O Harness pode sugerir alterações em si mesmo, mas não deve aplicar mudanças estruturais críticas automaticamente sem gate.

Exemplo:

```text
observação
 -> proposta
 -> implementação em branch
 -> testes
 -> benchmark
 -> revisão
 -> promoção
```

## Separação entre dados pessoais e corporativos

Regra forte:

- não compartilhar memória corporativa com o Engineering Harness;
- não sincronizar repositórios corporativos sem autorização;
- não reutilizar secrets;
- não importar automaticamente policies internas;
- não assumir que um runtime permitido no ambiente pessoal é permitido na empresa.

## Possível arquitetura de monorepo

```text
apps/
  desktop/
  cli/

packages/
  core/
  runtime-sdk/
  workflow-engine/
  memory-sdk/
  skill-engine/
  dream-engine/
  ui-components/

adapters/
  antigravity/
  claude-code/
  codex/

profiles/
  software-engineering/
  qa/
  devops/
```

## Prioridade inicial

Primeiro entregar um núcleo pequeno:

1. CLI.
2. Runtime Adapter.
3. Um runtime funcional.
4. Skill registry.
5. GBrain integration.
6. Workflow engine.
7. Dream Mode básico.
8. Electron UI.
9. Grafo/Wiki.
10. Auto-evolução controlada.

## Decisão canônica registrada

O Engineering Harness é pessoal e genérico.

O QA Automation Harness permanece um projeto corporativo distinto.

Ambos podem compartilhar conceitos e bibliotecas, mas não devem compartilhar automaticamente dados, memória, secrets ou governança.
