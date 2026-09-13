# Estudo de Viabilidade Técnica, Compatibilidade e Arquitetura Comparada — Família de Harnesses & Runtimes Atius

## 1. Visão Executiva & Contexto Geral

Este documento consolida a análise técnica profunda, a matriz de compatibilidade e o plano de ação estratégico para três iniciativas complementares, porém rigorosamente segregadas:

1. **Router AI Atius + Channel Antigravity (`agy`)**: Integração de baixo nível no backend Go do Atius para orquestrar contas e runtimes Antigravity em containers Podman rootless com segurança anti-banimento e otimização semântica via Headroom.
2. **QA Automation Harness Corporativo**: Solução desktop (Electron/Node) e CLI focada em engenharia de qualidade corporativa com Playwright, governança inegociável (*QA Constitution*), memória escopada via GBrain e consolidação contínua (*Dream Mode*).
3. **Engineering Harness Pessoal**: Ambiente de desenvolvimento autônomo e genérico para projetos pessoais, fundamentado em dogfooding contínuo, orquestração multi-agente (`agy`, Claude Code, Codex CLI) e consolidação arquitetural (*Architecture Dream*).

---

## 2. Comparativo Dimensional dos Três Pilares

| Dimensão | 1. Channel Antigravity (Atius) | 2. QA Harness Corporativo | 3. Engineering Harness Pessoal |
| :--- | :--- | :--- | :--- |
| **Finalidade Primária** | Gateway de API e proxy de inferência com abstração de processos. | Engenharia de testes e qualidade para produtos empresariais. | Desenvolvimento de software, criação de ferramentas e automações pessoais. |
| **Ambiente Alvo** | Cluster / Servidor Atius (`atius-srv-1`, Podman rootless). | Máquinas de desenvolvedores/QAs corporativos e CI/CD corporativo. | Máquina de desenvolvimento pessoal (Linux Ubuntu, WSL2). |
| **Stack Tecnológica** | Go 1.22+, Gin, GORM, Podman, Cgroups v2. | Electron, Node.js 22 LTS, React 19, Playwright, Tailwind. | Monorepo Bun/pnpm, TypeScript, CLI Node/Bun, Electron opcional. |
| **Protocolo de Entrada** | OpenAI Compatible (`/v1/chat/completions` com SSE). | Desktop UI / CLI de testes (`qa-harness run / dream`). | Terminal CLI First (`eng-harness run / plan`) + Cockpit Desktop. |
| **Runtimes Suportados** | Exclusivamente Antigravity CLI oficial (`agy`). | Claude Code Headless, Local vLLM/Ollama, Mock Engine. | Antigravity (`agy`), Claude Code, Codex CLI, OpenCode. |
| **Governança & Risco** | Alto foco em segurança de conta Google e zero scraping. | Conformidade rigorosa (InfoSec, SOC2, Zero-PII, SBOM, Proxies). | Autonomia máxima (100% Full YOLO Mode, auto-evolução). |
| **Mecanismo de Memória** | Histórico canônico no PostgreSQL + compressão Headroom. | GBrain Corporativo com escopo hierárquico estrito. | GBrain Pessoal + Obsidian Second Brain (`AiSecondBrain`). |
| **Ciclo de Consolidação** | N/A (foco em streaming e baixa latência de resposta). | *Memory Dream* (fatos de QA) e *Skill Dream* (testes estáveis). | *Memory Dream*, *Skill Dream* e *Architecture Dream*. |

---

## 3. Análise de Viabilidade Técnica e Operacional

### 3.1 Viabilidade do Channel Antigravity no Atius
- **Viabilidade Técnica**: **ALTA**. O uso de Podman rootless isolando o binário oficial em modo headless elimina o atrito de engenharia reversa de APIs privadas e protege contra banimentos. A comunicação via Unix Domain Sockets ou stdio pipe possui sobretaxa de latência desprezível (< 15ms).
- **Viabilidade de Recursos**: **ALTA**. Com o cgroup v2 respeitando a regra máxima de CPU do host (teto de 20% do total em `atius-srv-1`, ou seja, 0.8 vCPU no servidor de 4 núcleos), o pool pode manter até 3 instâncias em warm state consumindo menos de 700MB de RAM agregados.
- **Principal Risco & Mitigação**: Risco de suspensão de contas Google caso haja rajadas de tráfego fora do padrão. Mitigação: Concurrency Governor em Go impondo delays humanos e distribuição balanceada entre contas no pool.

### 3.2 Viabilidade do QA Harness Corporativo
- **Viabilidade Técnica**: **ALTA**. O ecossistema Playwright é maduro, possui APIs robustas de tracing e auto-waiting. O contrato `AgentRuntime` permite trocar o motor de IA sem reescrever as suítes de testes.
- **Viabilidade de InfoSec**: **ALTA COM CONTROLES**. A inclusão de SBOM, sanitização pré-voo de segredos e suporte nativo a proxies corporativos (Zscaler/BlueCoat com custom CAs) atende aos critérios padrão de comitês de segurança corporativa.
- **Principal Risco & Mitigação**: Rejeição de código gerado por falta de padronização. Mitigação: A *QA Constitution* opera como um linter semântico e arquitetural estrito, bloqueando seletores frágeis e sleeps arbitrários antes da execução.

### 3.3 Viabilidade do Engineering Harness Pessoal
- **Viabilidade Técnica**: **MUITO ALTA**. Implementado sobre um monorepo modular em TypeScript, permitindo entrega incremental: começa como um CLI no terminal integrado com `agy` e evolui organicamente para um cockpit com Electron.
- **Viabilidade de Recursos**: **MUITO ALTA**. A arquitetura CLI-First garante inicialização quase instantânea e baixo consumo de recursos, funcionando de maneira suave no ambiente Linux atual.
- **Principal Risco & Mitigação**: Contaminação cruzada de contexto com o trabalho corporativo. Mitigação: Barreira física e lógica inegociável — credenciais, vaults, bancos e arquivos de memória nunca compartilham o mesmo host ou diretório.

---

## 4. Matriz de Compatibilidade Cruzada

```text
[ Cliente / Usuário ]
          │
          ├───► Se for Demanda Corporativa ──────► QA Automation Harness Corporativo
          │                                              │
          │                                              ├──► Claude Code Headless (Homologado)
          │                                              └──► GBrain Corporativo (Escopo Cliente)
          │
          ├───► Se for Desenvolvimento Pessoal ──► Engineering Harness Pessoal
          │                                              │
          │                                              ├──► agy / Claude / Codex (YOLO Mode)
          │                                              └──► GBrain Local / Obsidian Vault
          │
          └───► Se for Chamada de API Gateway ───► Router AI Atius
                                                         │
                                                         └──► Channel Antigravity (Podman Pool)
```

### Regras de Intercâmbio:
1. **Atius como Fornecedor de API**: Tanto o QA Harness quanto o Engineering Harness podem consumir modelos através do Router AI Atius usando a interface padrão `/v1/chat/completions`.
2. **Isolamento de Memória**: O GBrain utilizado para armazenar os grafos do cliente corporativo é física e logicamente segregado do GBrain pessoal.
3. **Compartilhamento de Código**: O `packages/runtime-sdk` do Engineering Harness pode servir como biblioteca base para o adapter do QA Harness, desde que auditado contra inclusão de dependências restritivas.

---

## 5. Planejamento Estratégico Consolidado

```text
2026-Q3                       2026-Q4                       2027-Q1
───┬─────────────────────────────┬─────────────────────────────┬──────────►
   │ [Atius Channel Antigravity] │                             │
   │  - PoC Headless & Podman    │                             │
   │  - Adaptor Go & Pool        │ - Headroom Middleware       │
   │  - Soak test & Produção     │ - Observabilidade / Métricas│
   │                             │                             │
   │ [Engineering Harness]       │                             │
   │  - Monorepo & CLI Core      │ - Multi-Agente (Claude/Codex│
   │  - Integração GBrain/Vault  │ - Architecture Dream        │ - Desktop Cockpit
   │  - Dogfooding diário        │ - Auto-Evolução Controlada  │ - Modulos Públicos
   │                             │                             │
   │                             │ [QA Harness Corporativo]    │
   │                             │  - Core State Machine       │ - InfoSec Approval
   │                             │  - Playwright Constitution  │ - Electron UI
   │                             │  - Conector GBrain Escopado │ - Dream Pipelines
```

### Prioridades Imediatas (Próximos Passos):
1. **PoC de Transporte do `agy` no Atius**: Validar no Linux o comportamento não-interativo do binário oficial `agy` e documentar a estrutura de sockets IPC.
2. **Scaffolding do Monorepo do Engineering Harness**: Inicializar o repositório modular pessoal com Bun workspaces para início imediato do dogfooding.
3. **Formalização da QA Constitution**: Escrever o arquivo canônico de regras de qualidade do Playwright que servirá de especificação para o validador do QA Harness.
