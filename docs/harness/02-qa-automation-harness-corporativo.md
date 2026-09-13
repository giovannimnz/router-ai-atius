# QA Automation Harness Corporativo

## Objetivo

Criar um aplicativo executável, voltado a automação de QA, especialmente front-end e back-end, com forte foco em Playwright, workflows padronizados, políticas corporativas, skills canônicas, memória de longo prazo e suporte a diferentes runtimes de agentes.

Este projeto é separado do Atius.

Uso pretendido: ambiente corporativo.

## Conceito central

O Harness não deve ser apenas uma coleção de prompts.

Ele deve funcionar como uma camada determinística de engenharia de testes, semelhante a um Spec Kit especializado em QA.

```text
Usuário
  |
  v
QA Harness Desktop
  |
  +--> Workflow Engine
  +--> Policy / Constitution
  +--> Skill Engine
  +--> Memory Engine
  +--> Runtime Adapter
  |
  +--> Claude Code headless
  +--> Antigravity/agy para uso permitido
```

## Aplicativo desktop

Direção sugerida:

- Node.js no core;
- Electron como aplicação desktop;
- UI desacoplada da lógica de runtime;
- adapters para diferentes agentes/runtimes.

Possível evolução futura: considerar Tauri se footprint virar prioridade.

## Runtime Adapter

O Harness deve expor um contrato comum.

Exemplo conceitual:

```ts
interface AgentRuntime {
  id: string;
  capabilities(): Promise<RuntimeCapabilities>;
  startSession(input: SessionInput): Promise<SessionHandle>;
  send(session: SessionHandle, event: HarnessEvent): Promise<RuntimeEventStream>;
  resume(sessionId: string): Promise<SessionHandle>;
  cancel(sessionId: string): Promise<void>;
}
```

Implementações iniciais:

```text
ClaudeCodeRuntime
AntigravityRuntime
```

Possíveis futuras:

```text
CodexRuntime
OpenCodeRuntime
CustomRuntime
```

## Capabilities

Não reduzir todos os runtimes ao menor denominador comum.

Cada adapter declara capacidades:

- streaming;
- headless;
- resume;
- remote;
- browser/tool execution;
- MCP;
- filesystem;
- subagents;
- structured output;
- session persistence.

Assim, o Harness pode habilitar recursos dinamicamente.

## Workflow de QA

Exemplo:

```text
1. Intake
2. Requirement Analysis
3. Context Retrieval
4. Skill Discovery
5. Test Strategy
6. Plan
7. Implementation
8. Validation
9. Static checks
10. Execution
11. Evidence Collection
12. Review
13. Memory/Skill Extraction
14. Promotion Candidate
```

Cada estágio deve possuir gates explícitos.

## QA Constitution

Conjunto de regras obrigatórias do ambiente.

Exemplos:

- estrutura padrão de projeto;
- padrão de selectors;
- Page Object / Screenplay / fixture policy;
- naming conventions;
- requisitos de segurança;
- regras de secrets;
- regras de pipeline;
- política de retries;
- padrões de reports;
- linting;
- cobertura mínima;
- critérios para flaky tests;
- padrões de API testing;
- padrões de evidência.

O runtime executa dentro dessas regras.

## Skills como conceito canônico

Toda ação importante deve consultar skills antes de improvisar.

Uma skill representa **como fazer algo**.

Exemplos:

```text
playwright.create-page-object
playwright.map-dynamic-element
playwright.api-auth
playwright.pipeline-template
playwright.wait-strategy
playwright.retry-policy
api.karate.authorization-validation
api.contract-testing
```

## Memória vs Skill

Memória:

> O que sabemos.

Skill:

> Como fazemos.

O planejamento deve consultar ambos antes da execução.

## GBrain como base de memória

Direção registrada:

- adotar o GBrain como base do mecanismo de memória;
- evitar recriar do zero grafo, consolidação, dreaming e purge;
- criar adapters/integração entre Harness e GBrain;
- construir uma camada visual própria sobre o grafo;
- tornar lifecycle e escopo visíveis.

## Hierarquia de conhecimento

A memória deve ser escopada.

Exemplo:

```text
Global
  |
  +--> Cliente
       |
       +--> Programa/Time
            |
            +--> Projeto
                 |
                 +--> Repositório
```

Uma execução em um projeto nunca deve absorver automaticamente conhecimento de outro cliente.

## Escopos de exemplo

```text
global/
client/<client-id>/
team/<team-id>/
project/<project-id>/
repo/<repo-id>/
```

## Alimentação de memória

O Harness observa continuamente:

- decisões;
- padrões;
- erros;
- workarounds;
- convenções;
- documentação;
- resultados;
- alterações de arquitetura;
- aprendizados operacionais.

O agente pode sugerir memória, mas promoções relevantes devem seguir política de validação.

## Sistema digestivo de memória

Modelo conceitual:

```text
Ingestão
 -> Extração
 -> Classificação
 -> Deduplicação
 -> Validação
 -> Consolidação
 -> Canonização
 -> Depreciação
 -> Purge/Archive
```

Nem tudo que entra permanece.

## Dream Mode

O Dream Mode é mais amplo que memória.

Existem pelo menos dois ciclos:

### Memory Dream

Consolida conhecimento declarativo:

- merge de duplicatas;
- reconciliação de conflitos;
- promoção de conhecimento;
- supersede;
- arquivamento;
- descarte.

### Skill Dream

Consolida conhecimento operacional:

- analisa execuções reais;
- identifica caminhos eficientes;
- mede sucesso;
- mede retries;
- mede estabilidade;
- identifica padrões recorrentes;
- propõe nova skill;
- atualiza skill existente;
- promove método mais eficiente;
- marca método antigo como superseded.

## Critérios para promoção de skill

Nunca promover apenas porque uma execução foi mais rápida.

Avaliar:

- taxa de sucesso;
- estabilidade;
- número de retries;
- cobertura;
- legibilidade;
- manutenibilidade;
- aderência à constitution;
- compatibilidade com pipeline;
- segurança;
- custo;
- latência;
- evidência acumulada.

## Lifecycle de skill

```text
candidate
  -> experimental
  -> validated
  -> canonical
  -> superseded
  -> archived
```

## Dreaming via pipeline

O Electron não precisa permanecer aberto.

Executar ciclos agendados em CI/CD:

```bash
qa-harness dream memory
qa-harness dream skills
```

Runner pode ser:

- VM;
- container;
- bare metal;
- runner corporativo.

## Interface visual

A UI deve ser mais útil que uma simples wiki.

Visualizações:

- grafo de memória;
- grafo de skills;
- lineage;
- lifecycle;
- canonical vs superseded;
- dependências;
- origem da informação;
- evidências;
- cliente/projeto/repo;
- saúde da base;
- conflitos;
- candidatos a promoção.

## Wiki

Além do grafo, oferecer uma camada Wiki:

```text
/knowledge
/skills
/policies
/projects
/clients
/executions
/decisions
```

## Uso corporativo

Ponto importante: software não homologado pode ser bloqueado.

O produto deve ser apresentado com:

- finalidade restrita;
- arquitetura documentada;
- política de secrets;
- dados armazenados;
- endpoints acessados;
- SBOM;
- dependências;
- licenças;
- logs;
- mecanismos de auditoria;
- configuração de proxy;
- execução offline quando possível;
- integração com runtime corporativamente permitido.

No ambiente corporativo, Claude Code headless pode ser o runtime principal caso esteja homologado.

## Próximos passos

1. Definir constitution inicial.
2. Definir workflow state machine.
3. Criar contrato `AgentRuntime`.
4. Implementar `ClaudeCodeRuntime`.
5. Criar modelo de skill.
6. Criar skill registry.
7. Integrar GBrain.
8. Criar Dream Mode.
9. Criar UI de grafo + Wiki.
10. Criar CLI para pipeline.
11. Implementar políticas de escopo por cliente/projeto.
12. Preparar documentação de homologação corporativa.
