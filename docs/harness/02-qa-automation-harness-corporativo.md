# QA Automation Harness Corporativo — Especificação Arquitetural e Estudo de Viabilidade

## 1. Visão Geral & Enquadramento Corporativo

O **QA Automation Harness Corporativo** é uma solução desktop e CLI orientada a engenharia de qualidade de software para médias e grandes empresas.
Ele se diferencia radicalmente de assistentes de chat ou extensões genéricas de IDE: não se trata de uma coleção de prompts, mas de um **mecanismo determinístico de engenharia de testes**, atuando como um "Spec Kit Executável" com governança estrita, memória corporativa de longo prazo e execução autônoma dentro de guardrails inquebráveis.

### 1.1 Missão do Produto
- Automatizar o ciclo completo de testes (front-end com Playwright, back-end com REST/gRPC e contratos OpenAPI);
- Garantir 100% de aderência à **Constitution Corporativa** (regras inegociáveis de código, segurança e arquitetura);
- Eliminar testes frágeis (*flaky tests*) através de auto-correção baseada em evidências forenses;
- Reter conhecimento organizacional entre sprints por meio de um **Sistema de Memória Escopada** com ciclos de consolidação (*Dream Mode*).

---

## 2. Viabilidade Corporativa, InfoSec & Governança

Ambientes corporativos impõem restrições rigorosas de segurança da informação, auditoria e conformidade regulatória. Para que o Harness seja adotado sem atrito com equipes de Security/CISO, ele incorpora por design os seguintes pilares:

### 2.1 Matriz de Riscos de InfoSec & Mitigações

| Requisito Corporativo | Risco Potencial | Arquitetura de Mitigação no Harness |
| :--- | :--- | :--- |
| **Vazamento de Segredos e Credenciais** | Envio de chaves de API, tokens de acesso ou senhas de banco nos prompts para LLMs. | **Sanitização Pré-Voo**: Interceptor local usando regex estrito e heurísticas de entropia que mascara segredos (`<SECRET_MASKED>`) antes de qualquer payload sair da máquina. Todas as credenciais de teste são injetadas em runtime via HashiCorp Vault ou variáveis de ambiente de CI. |
| **Proteção de Dados Pessoais (LGPD / GDPR)** | Ingestão de dados reais de clientes (PII) em logs de teste ou na base de conhecimento. | **Política Zero-PII**: O Harness exige e gera dados sintéticos via Faker/Mock generators para testes. Qualquer identificador pessoal detectado em payloads de API é barrado no gate de Intake. |
| **Aprovação de Software (Whitelist/Homologação)** | Bloqueio de instalação por ferramentas de EDR (CrowdStrike, Defender for Endpoint) ou firewall corporativo. | **InfoSec Approval Pack**: Distribuição com SBOM assinado (Software Bill of Materials em formato CycloneDX), binários assinados digitalmente via certificado corporativo, ausência de telemetria oculta e código open/internal source auditável. |
| **Inspeção SSL e Proxies Corporativos** | Falha de conexão HTTPS com agentes ou APIs devido a proxies como Zscaler ou BlueCoat. | Suporte nativo a variáveis `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` e injeção de Root CA corporativa personalizada (`NODE_EXTRA_CA_CERTS`). |
| **Propriedade Intelectual & Código** | Uso de código do cliente para treinamento de modelos de terceiros. | Runtimes corporativos autorizados com cláusula contratual explícita de **Zero Data Retention (ZDR)** e opt-out de treinamento (ex: Claude for Enterprise, Azure OpenAI, GCP Vertex AI). |

---

## 3. Runtimes Homologáveis & Adapter Pattern

O Harness opera através de uma interface agnóstica (`AgentRuntime`). Cada runtime declara suas capacidades operacionais em tempo de inicialização, permitindo que a organização selecione o motor aprovado para sua política de segurança.

```text
                           ┌───────────────────────────┐
                           │    AgentRuntime Contract  │
                           └─────────────┬─────────────┘
                                         │
                 ┌───────────────────────┼───────────────────────┐
                 ▼                       ▼                       ▼
      ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐
      │  ClaudeCodeRuntime  │ │  AntigravityRuntime │ │    LocalEngine      │
      │  (Headless CLI /    │ │  (agy oficial /     │ │  (vLLM / Ollama     │
      │   Enterprise API)   │ │   GCP Homologado)   │ │   Offline Airgap)   │
      └─────────────────────┘ └─────────────────────┘ └─────────────────────┘
```

### 3.1 Contrato da Interface `AgentRuntime`

```typescript
export interface RuntimeCapabilities {
  supportsStreaming: boolean;
  supportsHeadless: boolean;
  supportsSessionResume: boolean;
  supportsMcpServers: boolean;
  supportsSubagents: boolean;
  supportsToolExecution: boolean;
  maxContextTokens: number;
}

export interface SessionInput {
  sessionId: string;
  scope: KnowledgeScope;
  constitutionRules: string[];
  skills: CanonicalSkill[];
  workingDirectory: string;
  env: Record<string, string>;
}

export interface AgentRuntime {
  id: string;
  name: string;
  version: string;
  getCapabilities(): Promise<RuntimeCapabilities>;
  startSession(input: SessionInput): Promise<SessionHandle>;
  sendPrompt(session: SessionHandle, prompt: string): Promise<AsyncIterable<HarnessEvent>>;
  resumeSession(sessionId: string): Promise<SessionHandle>;
  abortSession(sessionId: string): Promise<void>;
}
```

---

## 4. O Workflow Determinístico de QA (14 Estágios)

Para garantir previsibilidade e auditoria, o agente não pula etapas. A execução avança através de uma máquina de estados finita com critérios de passagem (gates) matemáticos:

```text
 [1. Intake] ──► [2. Req Analysis] ──► [3. Context Retrieval] ──► [4. Skill Discovery]
                                                                          │
                                                                          ▼
 [8. Validation] ◄── [7. Implement] ◄── [6. Test Plan] ◄── [5. Test Strategy]
        │
        ▼
 [9. Static Checks] ──► [10. Execution] ──► [11. Evidence] ──► [12. Review]
                                                                     │
                                                                     ▼
                                            [14. Promotion] ◄── [13. Extraction]
```

### Detalhamento dos Gates:
1. **Intake**: Validação da história de usuário, issue no Jira ou PR. Rejeita requisitos ambíguos.
2. **Requirement Analysis**: Mapeamento de regras de negócio e critérios de aceitação (Gherkin/BDD quando aplicável).
3. **Context Retrieval**: Consulta ao **GBrain** com escopo delimitado ao projeto específico.
4. **Skill Discovery**: Identificação das skills canônicas obrigatórias para a tarefa (ex: `playwright.auth-session`, `playwright.table-grid`).
5. **Test Strategy**: Decisão entre testes de unidade, integração de API ou E2E visual.
6. **Test Plan**: Documento formal contendo matriz de cobertura e casos de teste positivos/negativos/edge cases.
7. **Implementation**: Geração dos arquivos de teste seguindo rigorosamente o Page Object Model.
8. **Validation**: Execução em modo isolado local para checar sintaxe e importações.
9. **Static Checks**: Execução de ESLint, TypeScript check (`tsc --noEmit`) e validação da Constitution.
10. **Execution**: Rodada completa com Playwright em modo headless multi-browser.
11. **Evidence Collection**: Gravação de trace files (`trace.zip`), vídeos em WebP, screenshots de assertions e logs HAR de rede.
12. **Review**: Auto-avaliação do agente comparando o resultado obtido com o plano original.
13. **Memory/Skill Extraction**: Identificação de novos padrões de tela, APIs modificadas ou workarounds descobertos.
14. **Promotion Candidate**: Submissão do conhecimento para o ciclo noturno do *Dream Mode*.

---

## 5. Playwright Engineering Constitution

A **Constitution** é o conjunto de regras inegociáveis que o motor aplica automaticamente sobre qualquer código gerado:

### 5.1 Regras de Seletores (Resiliência Absoluta)
- **Proibido**: Seletores hierárquicos frágeis (`div > div:nth-child(2) > span`), seletores acoplados a classes utilitárias de CSS (`.flex.items-center.p-4`).
- **Obrigatório**:
  1. Accessible Roles: `page.getByRole('button', { name: 'Confirmar' })`
  2. Text Labels: `page.getByLabel('E-mail')`
  3. Placeholder/Text: `page.getByPlaceholder('Digite sua senha')`
  4. Atributo Corporativo Canônico: `page.getByTestId('submit-order-button')`

### 5.2 Regras de Sincronização & Waits
- **Proibido**: `page.waitForTimeout(...)` ou sleeps arbitrários em qualquer circunstância.
- **Obrigatório**: Web-first assertions nativas com auto-retry:
  ```typescript
  await expect(page.getByRole('alert')).toHaveText('Operação realizada com sucesso');
  ```

### 5.3 Arquitetura de Page Objects
- Cada tela ou componente de domínio possui sua classe correspondente herdando de uma `BasePage`.
- As classes de teste nunca chamam `page.locator()` diretamente; elas utilizam métodos de negócio expressivos (`await checkoutPage.finalizePurchase()`).

---

## 6. Sistema de Memória Escopada & Integração com GBrain

Para evitar vazamento de dados entre clientes ou projetos corporativos, a memória é estruturada em uma árvore hierárquica estrita:

```text
global/                     (Regras universais de QA, boas práticas Playwright)
  └── client/<client-id>/   (Convenções do cliente, arquitetura geral, stack)
        └── project/<id>/   (Regras de negócio do sistema, rotas, contratos de API)
              └── repo/<id> (Padrões do repositório, branch policies, fixtures)
```

### Regras de Isolamento:
- Uma sessão iniciada em `client/acme-corp/project/banking` tem visibilidade apenas dos nós ancestrais (`global`, `acme-corp`, `banking`).
- **Nenhuma informação** do cliente A pode ser lida, inferida ou vazada para o cliente B.
- O backend de persistência é o **GBrain** corporativo via protocolo MCP HTTP seguro com autenticação por token de curta duração.

---

## 7. O Ciclo Digestivo & Dream Mode

O conhecimento acumulado em testes passa por um processo formal de destilação para evitar inchaço e contradições na base:

### 7.1 Memory Dream (Consolidação de Fatos Declarativos)
Executado em pipelines agendados (ex: toda madrugada):
- **Deduplicação**: Agrupa regras de negócio idênticas reportadas por diferentes testes.
- **Resolução de Conflitos**: Se uma API mudou de `/v1/users` para `/v2/users`, o fato antigo é marcado como `superseded`.
- **Purge de Falhas Efêmeras**: Erros causados por instabilidade momentânea de rede são descartados, retendo apenas padrões sistêmicos.

### 7.2 Skill Dream (Consolidação de Conhecimento Operacional)
- Analisa os relatórios de execução das últimas 50 rodadas de CI.
- Identifica rotinas repetidas (ex: "como contornar um componente complexo de calendário de terceiros").
- Gera uma nova **Skill Canônica**, versionada e acompanhada de testes unitários.
- Submete a nova skill para aprovação humana através do painel desktop.

```text
[Candidato a Skill] ──► [Testes Automáticos] ──► [Revisão do Lead de QA] ──► [Skill Canônica]
```

---

## 8. Arquitetura da Aplicação Desktop (Electron + React)

A interface gráfica é desenhada para transparência total e controle do líder técnico:

### 8.1 Componentes da UI:
- **Grafo de Memória & Skills Interativo**: Visualização baseada em nós conectados mostrando o conhecimento corporativo e a saúde de cada suíte de teste.
- **Constitution Inspector**: Painel que exibe a pontuação de conformidade dos testes com a política da empresa (zero warnings = verde).
- **Execution Replay & Tracing**: Player integrado de traces do Playwright com timeline de rede, console e tela sincronizados.
- **Wiki Corporativa Viva**: Documentação viva gerada automaticamente a partir da memória consolidada, servindo como documentação oficial de negócio e testes.

---

## 9. Cronograma e Plano de Entrega (Fases de Implementação)

| Fase | Escopo Principal | Deliverables | Critério de Sucesso |
| :--- | :--- | :--- | :--- |
| **Fase 1: Core Engine & Constitution** | State machine dos 14 estágios e validador de regras de código. | Pacote `@qa-harness/core` com testes unitários em TypeScript. | Validador barra 100% dos seletores e timeouts proibidos em fixtures de teste. |
| **Fase 2: Playwright Runner & Evidências** | Execução headless do Playwright com coleta padronizada de traces e HAR. | Módulo de execução e reporter de evidências com sanitização. | Execução limpa em Linux/Windows gerando pacote forense completo em falhas. |
| **Fase 3: Runtime Adapters** | Implementação do `ClaudeCodeRuntime` e interface com CLI headless. | Adapter Claude Code corporativo com suporte a streaming de eventos. | Agente completa ciclo de geração e execução de um teste E2E sem intervenção manual. |
| **Fase 4: Integração com GBrain & Memória** | Mapeamento hierárquico de escopos e persistência no grafo de conhecimento. | Conector MCP GBrain com sanitização pré-voo de dados confidenciais. | Memória isolada por cliente validada em testes de isolamento cruzado. |
| **Fase 5: Dream Engine & Pipeline CLI** | Comandos CLI para integração em CI/CD (`qa-harness dream`). | CLI executável em containers de pipeline (GitHub Actions, GitLab CI). | Ciclo noturno roda e consolida 10 novos fatos e 1 skill sem intervenção humana. |
| **Fase 6: Electron Desktop UI & Homologação** | Interface gráfica com visualizador de grafo, player de traces e wiki. | Instaladores desktop assinados e InfoSec Approval Kit completo. | Homologação aprovada em ambiente corporativo de referência. |
