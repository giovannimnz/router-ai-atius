---
name: router-ai-atius
description: Operate or modify the `router-ai-atius` deployment/fork, including provider routing, Antigravity pool, TypeSafe, Alibaba Cloud, Codex/OpenAI/Anthropic channel work, CLIAnything, i18n, docs, sync safety, Podman deployment awareness and fork-specific protected files.
---

# Router AI Atius

Primary paths:

- Deploy/control repo: `/home/ubuntu/GitHub/containers/router-ai-atius`
- Runtime bind data/logs: `/home/ubuntu/GitHub/containers/router-ai-atius/data`, `/home/ubuntu/GitHub/containers/router-ai-atius/logs`
- Legacy/source-like tree may exist at `/home/ubuntu/GitHub/containers/Atius/router-ai-atius`, but verify git health before trusting it

Use this skill when the task touches fork customizations, provider routing, Antigravity channel pool, TypeSafe, Alibaba Cloud, Codex channel integration, OpenAI/Anthropic SDK behavior, CLIAnything, i18n, docs or deployment-sensitive files.

## First commands

```bash
cd /home/ubuntu/GitHub/containers/router-ai-atius
bin/clianything --backend host status
bin/clianything --backend host providers --all
```

> **Atenção Backend DB**: Utilize sempre `--backend host` (ou `CLIANYTHING_DB_BACKEND=host`) ao chamar `bin/clianything`. No host atual, o banco ativo `DBRouterAiAtius` roda via PgBouncer (`10.11.1.11:6432`). O container podman `postgres` é reservado e vazio por design.

## Current guardrails

- Treat `db-data/`, secrets, env files and runtime volumes as off-limits unless explicitly requested
- Never print real values from channel keys, API tokens, passwords, OAuth secrets, Codex auth files, header overrides or `~/.hermes/.env`
- Prefer `bin/clianything --backend host` for DB inspection because it redacts sensitive fields by default
- `clianything create|update|delete` is dry-run unless `--execute`; write mode creates table backup in `backups/clianything/`
- `clianything api|endpoint invoke|call` is dry-run for mutating methods and classified sensitive actions unless `--execute`
- In this personal-use fork, `user_quota` is accounting data and must never authorize or block a request, even when negative. Preserve `scripts/atius-user-quota-guard.sh audit` as the mandatory gate.
- Build Resource Caps: All heavy build, typecheck, and production image commands MUST run through `./scripts/podman-admin.sh` to enforce the 20% host CPU cap (0.8 CPU / 1 job). Never run raw `bun run build`, `podman build` or `docker build` directly.

## Fornecedores, Canais e Ícones

### Esquema de Ícones Híbrido: LobeHub + Internal SVG
1. **LobeHub Standard (`@lobehub/icons`)**:
   - Ícones externos de terceiros usam PascalCase: `Qwen`, `OpenAI`, `DeepSeek`, `Google`, `TypeSafe`, `AtiusLocal`.
2. **Esquema Interno (`Internal.<name>`)**:
   - Fornecedor Antigravity: `Internal.antigravity` (renderiza `web/default/src/components/antigravity-logo.tsx`, SVG monocromático oficial com fundo `#000000`, borda sutil e símbolo branco `#FFFFFF`).
   - Modelos de IA do Fornecedor Antigravity: `Internal.antigravity-color` (renderiza `web/default/src/components/antigravity-color-logo.tsx`, SVG colorido oficial com gradientes e `useId()` dinâmico).
   - Fornecedor Atius Local: `Internal.atius` (renderiza `web/default/src/components/atius-logo.tsx`, SVG monocromático com fundo `#000000`, borda sutil, letra A branca `#FFFFFF` e letra C cinza `#D1D5DB`).
   - Modelos de IA do Fornecedor Atius Local: `Internal.atius-color` (renderiza `web/default/src/components/atius-color-logo.tsx`, SVG colorido oficial com verde `#0f3b25` e dourado `#d2aa2a`).
   - Canal Antigravity (Tipo 60): Renderiza `AntigravityColorLogo` por padrão.
   - Canal Atius Local (Tipo 59): Renderiza `AtiusColorLogo` por padrão.
   - Regra Obrigatória de Monocromático: A coluna "Fornecedor" em `/models` e todas as referências de fornecedor em `/pricing` (cards, colunas, sidebar e detalhes) DEVEM sempre exibir os ícones monocromáticos (`Internal.antigravity` e `Internal.atius`). Os ícones coloridos são reservados exclusivamente para os modelos individuais de IA.
   - Sincronização e utilitários: O script `scripts/sync-vendor-icons.sh` sincroniza os SVGs canônicos a partir de `/home/ubuntu/Imagens/` para `web/default/public/images/` e `web/default/dist/images/`.
   - Mutação via UI: Ao selecionar fornecedor com prefixo `Internal.<nome>`, a interface auto-preenche os modelos vinculados com `Internal.<nome>-color`.

### Agrupamento de Modelos por Esforço de Raciocínio (Reasoning Effort)
- O catálogo de modelos e o canal do Antigravity expõem 1 modelo base canônico por família, sem poluição de múltiplos sufixos (`-low`, `-medium`, `-high`):
  - `gemini-3.8-flash`
  - `gemini-3.7-flash`
  - `gemini-3.6-flash`
  - `gemini-3.1-pro`
  - `claude-sonnet-4-6`
  - `claude-opus-4-6-thinking`
  - `gpt-oss-120b`
- O adaptador Go (`relay/channel/antigravity/adaptor.go`) e o daemon Python (`scripts/agy-daemon.py`) mapeiam o modelo base + o parâmetro `reasoning_effort` (com padrão `high`) para a inferência upstream correspondente. Modelos legados com sufixo explícito continuam aceitos com retrocompatibilidade transparente.

### Acompanhamento de Saldo dos Canais
- **Antigravity (Tipo 60) e Atius Local (Tipo 59)**:
  - Canais internos com cota ilimitada (`channel.unlimited_balance`: `"Este canal é um serviço interno com cota ilimitada"`).
  - O modal e a tabela exibem o Saldo Consumido (`used_quota`) e sinalizam o Saldo Disponível como `Unlimited`.
- **TypeSafe (Tipo 61) e Alibaba Cloud / Qwen (Tipo 17)**:
  - Não possuem API remota de consulta automática de saldo. O backend retorna status `success: true` com mensagem informativa, eliminando erros como "尚未实现" ou falhas HTTP.
  - A interface exibe em grid o **Saldo Consumido (`used_quota`)** e o **Saldo Disponível (`balance`)**, acompanhados de aviso explicativo: *"A consulta automática de saldo não é suportada para este tipo de canal. Tanto o saldo consumido quanto o saldo disponível são acompanhados e exibidos aqui."*

## Antigravity Multi-Account Pool & Load Balancer

O canal Antigravity conecta em `http://10.11.1.11:18088`, operado por `scripts/podman-agy-pool.sh`:
- **Instância 1 (`atius-agy-acc1`)**: Porta 18081, egress IP `137.131.190.161`.
- **Instância 2 (`atius-agy-acc2`)**: Porta 18082, egress IP `137.131.140.20`.
- **Load Balancer (`atius-agy-lb`)**: Porta 18088, distribui requisições em round-robin entre instâncias saudáveis, gerenciando concorrência e evitando bloqueios de IP upstream.

Comandos operacionais do pool:
```bash
./scripts/podman-agy-pool.sh status
./scripts/podman-agy-pool.sh restart
```

## Related agents

- `hsd-dev-agent`
- `gsd-debugger`
- `gsd-code-reviewer`
