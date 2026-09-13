# Engineering Harness Pessoal — Especificação Arquitetural e Estudo de Viabilidade

## 1. Visão Geral & Filosofia de Engenharia

O **Engineering Harness Pessoal** é um ambiente autônomo de desenvolvimento de software, concebido para acelerar a criação, manutenção e evolução de ferramentas, produtos e projetos pessoais com máxima velocidade e alavancagem de IA.

Ao contrário de soluções corporativas que priorizam comitês de governança, restrições contratuais e políticas defensivas, o Engineering Harness Pessoal é orientado a:
1. **Velocidade Extrema & Autonomia (Full YOLO Mode)**: Execução direta sem telas de confirmação burocráticas, com automação ponta a ponta;
2. **Dogfooding Radical**: O harness é utilizado para desenvolver, testar e refatorar a sua própria base de código desde o primeiro dia de vida;
3. **Agnosticismo de Agentes**: Orquestração intercambiável dos melhores runtimes do mercado (Antigravity `agy`, Claude Code, Codex CLI, OpenCode);
4. **Memória de Longo Prazo e Auto-Evolução**: Aprendizado cumulativo via **GBrain** pessoal e consolidação automática de padrões de arquitetura (**Architecture Dream**);
5. **Isolamento Absoluto do Mundo Corporativo**: Separação física e lógica estrita contra qualquer vazamento de credenciais, repositórios ou dados entre vida pessoal e empresarial.

---

## 2. Princípios de Isolamento e Separação Pessoal vs. Corporativo

```text
┌──────────────────────────────────────┐       ┌──────────────────────────────────────┐
│     QA Harness Corporativo           │       │    Engineering Harness Pessoal       │
├──────────────────────────────────────┤       ├──────────────────────────────────────┤
│ - Escopo: Empresas & Contratos       │       │ - Escopo: Projetos Pessoais & Lab    │
│ - Governança: InfoSec, SOC2, CISO    │       │ - Governança: Autonomia Ágil         │
│ - Secrets: Vaults Empresariais       │       │ - Secrets: HashiCorp Vault Pessoal   │
│ - Memória: GBrain Corporativo        │       │ - Memória: GBrain Local / Obsidian   │
│ - Runtime: Claude Code Homologado    │       │ - Runtime: agy / Claude / Codex YOLO │
└──────────────────────────────────────┘       └──────────────────────────────────────┘
                  ▲                                               ▲
                  │                                               │
                  └────────────[ BARREIRA HERMÉTICA ]─────────────┘
                     - Sem compartilhamento de memória
                     - Sem reuso de chaves ou sessões
                     - Sem contaminação de repositórios
```

### Regras de Ouro de Segurança:
- **Zero Cross-Contamination**: O Engineering Harness nunca consulta nem armazena nós de grafos com IDs ou tags de clientes corporativos.
- **Sessões Isoladas de Terminal**: Execução sob contas locais e ambientes Virtuais (Python uv, Bun, Podman) segregados.
- **Armazenamento de Notas**: Toda documentação gerada pelo Engineering Harness é arquivada no cofre pessoal do Obsidian (`~/GitHub/obsidian-vault/AiSecondBrain`).

---

## 3. Arquitetura em Monorepo Modular

O projeto adota estrutura de monorepo gerenciado por **Bun** ou **pnpm workspaces**, garantindo reutilização de bibliotecas compartilhadas com fronteiras estritas de pacotes:

```text
engineering-harness/
├── apps/
│   ├── cli/                    # CLI de terminal ultra-responsivo para fluxo diário
│   └── desktop/                # Shell Electron / Tauri para cockpit visual e grafos
├── packages/
│   ├── core/                   # Orquestrador central e máquina de estados de workflows
│   ├── runtime-sdk/            # Contrato comum de agentes com streaming e IPC
│   ├── skill-engine/           # Carregador, validador e executor de skills versionadas
│   ├── memory-sdk/             # Cliente de integração com GBrain e Second Brain
│   ├── dream-engine/           # Motores de consolidação (Memory, Skill, Architecture)
│   └── ui-components/          # Componentes visuais compartilhados (React 19 / Tailwind)
├── adapters/
│   ├── antigravity/            # Adapter para runtime oficial agy
│   ├── claude-code/            # Adapter para Claude Code CLI headless
│   └── codex/                  # Adapter para OpenAI Codex CLI
└── profiles/
    ├── software-engineering/   # Refatorações, scaffolding, PRs e git worktrees
    ├── devops/                 # Infra local, Podman, cgroups, systemd, WireGuard
    ├── ai-integration/         # Provedores de modelos, proxies, SSE, tool-calling
    └── research/               # Síntese bibliográfica, experimentação e spikes
```

---

## 4. O Sistema de Runtimes Multi-Agente

O módulo `runtime-sdk` fornece a abstração necessária para despachar tarefas para o melhor modelo disponível de acordo com a característica da demanda:

```typescript
export type AgentType = 'antigravity' | 'claude-code' | 'codex' | 'opencode';

export interface RuntimeCapabilities {
  supportsCodeExecution: boolean;
  supportsHeadlessStreaming: boolean;
  supportsSubagentSpawning: boolean;
  supportsMcpTools: boolean;
  preferredWorkload: 'coding' | 'refactor' | 'architecture' | 'debugging';
}

export interface ExecutionContext {
  taskDescription: string;
  cwd: string;
  yoloMode: boolean;
  activeProfile: string;
  allowedTools: string[];
  systemInstructions?: string;
}

export interface AgentRuntime {
  id: AgentType;
  getCapabilities(): Promise<RuntimeCapabilities>;
  execute(ctx: ExecutionContext): Promise<AsyncIterable<ExecutionEvent>>;
  abort(executionId: string): Promise<void>;
}
```

### 4.1 Características de Cada Runtime Integrado:
- **Antigravity (`agy`)**: Motor primário para tarefas complexas de coding e automação em ambiente local. Utiliza o binário oficial em modo headless com flags `--dangerously-skip-permissions` para execução autônoma contínua.
- **Claude Code**: Excepcional para raciocínio analítico, leitura abrangente de repositórios e refatorações cirúrgicas de código TypeScript/Go.
- **Codex CLI**: Ideal para automações de infraestrutura e tarefas rápidas de terminal sob o wrapper `codex --yolo`.
- **OpenCode / Ollama Local**: Motor de fallback offline para operações locais que não dependem de conexão externa.

---

## 5. Dogfooding & O Ciclo de Auto-Evolução Controlada

Um diferencial estrutural do Engineering Harness é a sua capacidade de propor melhorias em sua própria arquitetura:

```text
 ┌───────────────────────────────────────────────────────────┐
 │               Ciclo de Auto-Evolução                      │
 └─────────────────────────────┬─────────────────────────────┘
                               │
                               ▼
 ┌───────────────────────────────────────────────────────────┐
 │ 1. Detecção de Atrito ou Oportunidade                      │
 │    - Análise de repetição de comandos ou erros frequentes │
 └─────────────────────────────┬─────────────────────────────┘
                               │
                               ▼
 ┌───────────────────────────────────────────────────────────┐
 │ 2. Formulação de Proposta de Mudança                      │
 │    - Criação de RFC e branch isolada auto-evolve/*        │
 └─────────────────────────────┬─────────────────────────────┘
                               │
                               ▼
 ┌───────────────────────────────────────────────────────────┐
 │ 3. Implementação e Bateria de Testes                      │
 │    - Typecheck, ESLint, testes de unidade e benchmarks    │
 └─────────────────────────────┬─────────────────────────────┘
                               │
                               ▼
 ┌───────────────────────────────────────────────────────────┐
 │ 4. Gate Humano de Promoção (1 Clique / 1 Comando)         │
 │    - Exibição do diff sintético para aprovação final      │
 └───────────────────────────────────────────────────────────┘
```

### Regras do Auto-Upgrade Seguro:
1. O agente **nunca** comita alterações em si mesmo diretamente na branch `main`;
2. As mudanças são desenvolvidas em uma worktree ou branch efêmera;
3. Antes de ser apresentada ao desenvolvedor, a versão atualizada roda uma suíte de testes de auto-validação comprovando que nenhuma funcionalidade existente quebrou;
4. Um comando simples de CLI (`eng-harness self-upgrade apply`) realiza o fast-forward merge e reinicia o serviço/processo suavemente.

---

## 6. Architecture Dream & Memória Operacional

Além dos ciclos de consolidação de memória de fatos e de skills, o Engineering Harness introduz o **Architecture Dream**:

### 6.1 Objetivos do Architecture Dream
- **Detecção de Código Duplicado Entre Repositórios**: Analisa projetos distintos mantidos no workspace e identifica helpers ou funções utilitárias que foram reinventadas, propondo sua extração para um pacote no monorepo.
- **Identificação de Débito Técnico Recorrente**: Identifica arquivos que sofreram múltiplos hotfixes sucessivos, sugerindo refatorações estruturais ou padrões de projeto mais adequados.
- **Sincronização com o Second Brain**: Gera registros arquiteturais canônicos em formato Markdown no Obsidian (`AiSecondBrain/Architectural-Decisions/`), mantendo o mapa mental sempre alinhado com a realidade do código.

---

## 7. Interfaces de Usuário: CLI Primário + Cockpit Desktop

A arquitetura garante que a produtividade nunca fique refém de uma interface gráfica pesada:

### 7.1 CLI First (Terminal diário)
O CLI é construído com Node.js nativo ou Bun binary standalone para inicialização instantânea (< 30ms):
```bash
# Execução direta de tarefa com o perfil padrão de engenharia
eng-harness run "Refatorar middleware de autenticação para suportar passkeys"

# Planejamento estruturado sem alterar arquivos imediatamente
eng-harness plan "Migrar banco de dados SQLite para PostgreSQL"

# Gestão de skills do ecossistema
eng-harness skills list
eng-harness skills create "docker.multi-stage-optimization"

# Disparo manual de ciclo de consolidação
eng-harness dream architecture --scope "containers/router-ai-atius"
```

### 7.2 Cockpit Desktop (Electron / React 19)
Para análises visuais profundas e acompanhamento de execuções longas:
- **Grafo do Second Brain & Memória**: Navegação interativa nas entidades, conexões e decisões salvas no GBrain;
- **Painel de Runtimes Ativos**: Métricas em tempo real de latência, tokens gerados e saúde de cada agente;
- **Timeline de Auto-Evolução**: Histórico visual de todas as melhorias sugeridas e aplicadas no harness;
- **Central de Skills**: Editor visual de skills com validação de schema e testes integrados.

---

## 8. Plano de Execução e Roadmap em Milestones

| Milestone | Objetivo Técnico | Entregáveis | Critério de Aceitação |
| :--- | :--- | :--- | :--- |
| **M1: CLI Core & Antigravity Adapter** | Estrutura base de monorepo e execução do `agy` headless via CLI. | `apps/cli`, `packages/core`, `adapters/antigravity`. | CLI executa tarefa simples de refatoração ponta a ponta sem falhas. |
| **M2: Skill Engine & Integração GBrain** | Registry de skills versionadas e persistência de memória no GBrain. | `packages/skill-engine`, `packages/memory-sdk`. | Agente consulta e executa skill pré-definida e registra aprendizado na memória. |
| **M3: Suporte Multi-Agente (Claude + Codex)** | Implementação dos adapters para Claude Code e Codex CLI. | `adapters/claude-code`, `adapters/codex`. | Capacidade de alternar o agente executor via flag `--runtime=claude-code`. |
| **M4: Self-Improvement Loop & Architecture Dream** | Criação do mecanismo de auto-evolução e análise transversal de código. | `packages/dream-engine` com rotinas de análise estática. | Harness detecta duplicação de código real e abre branch com proposta de abstração. |
| **M5: Desktop Cockpit Shell** | Interface gráfica rica em Electron com visualizador de grafos e métricas. | `apps/desktop` com React 19, Tailwind CSS e Base UI. | Usuário navega pelo grafo de conhecimento e acompanha execuções em tempo real. |
| **M6: Dogfooding Contínuo & Ecossistema de Skills** | Operação diária completa usando o harness como motor principal de trabalho. | Suíte de 30+ skills canônicas cobrindo stack Go, React, Podman e TypeScript. | 100% dos novos desenvolvimentos de ferramentas pessoais geridos pelo harness. |
