# Phase 06: Login Explícito + Armazenamento Licença (SDK-02) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-06
**Phase:** 06-Login Explícito + Armazenamento Licença (SDK-02)
**Areas discussed:** Superfície de auth, Dono da credencial

---

## Superfície de auth

| Option | Description | Selected |
|--------|-------------|----------|
| Página global só da licença SDK | Superfície dedicada para OAuth/import/status do SDK | ✓ |
| Página global + drawer | Fluxo completo em dois lugares | |
| Drawer por canal | Cada canal cuida da própria auth inline | |

**User's choice:** só pra licença do SDK
**Notes:** user travou que o fluxo de auth não deve viver espalhado por canal.

| Option | Description | Selected |
|--------|-------------|----------|
| Só status + link para /admin/codex-auth | Drawer vira atalho/observabilidade | ✓ |
| Nada de auth no drawer | Remove qualquer referência | |
| Status + ações rápidas | Atalho + utilidades extras | |

**User's choice:** Só status + link para /admin/codex-auth
**Notes:** mantém discoverability sem duplicar o fluxo.

| Option | Description | Selected |
|--------|-------------|----------|
| Mesma página com 2 blocos: OAuth code + importar JSON | Fluxo completo numa tela | ✓ |
| Tabs separadas na mesma página | Separa os modos | |
| Só OAuth code na UI; JSON por API/CLI | UI mínima | |

**User's choice:** Mesma página com 2 blocos: OAuth code + importar JSON
**Notes:** import manual de JSON continua requisito visível na UI.

| Option | Description | Selected |
|--------|-------------|----------|
| Status compacto | email + expiry básico | |
| Status completo | email, account_id, expiry, last_refresh, source | |
| Status completo + botão manual de refresh | status detalhado + ação admin | ✓ |

**User's choice:** Status completo + botão manual de refresh
**Notes:** status da licença precisa ser administrável, não só informativo.

| Option | Description | Selected |
|--------|-------------|----------|
| Não exportar; só status | Sem retorno da credencial | |
| Mostrar JSON mascarado + copiar sob ação explícita | Exposição parcial | |
| Permitir download/export completo | Export integral da credencial | ✓ |

**User's choice:** Permitir download/export completo
**Notes:** decisão sensível; planning precisa preservar isso como superfície admin-only.

| Option | Description | Selected |
|--------|-------------|----------|
| Entrada própria no menu admin | item top-level | |
| Atalho dentro de Channels + rota dedicada | usa contexto existente de canais | ✓ |
| Só rota dedicada, sem item novo de menu | escondido da navegação | |

**User's choice:** Atalho dentro de Channels + rota dedicada
**Notes:** descoberta contextual, sem inflar navegação global.

---

## Dono da credencial

| Option | Description | Selected |
|--------|-------------|----------|
| `license.json` fonte única | canais leem o estado global | |
| Arquivo + espelho no `channel.key` | duplo write com arquivo mandando | |
| Cada canal continua com sua própria key; arquivo global é só cache auxiliar | channel-centric com espelho global | ✓ |

**User's choice:** Cada canal continua com sua própria key; arquivo global é só cache auxiliar
**Notes:** isso preserva o desenho atual por canal e reduz migração estrutural.

| Option | Description | Selected |
|--------|-------------|----------|
| Arquivo global espelha um canal primário | 1 canal publica o cache global | ✓ |
| Arquivo global gerido separado | storage paralelo sem canal dono | |
| Arquivo global nem precisa existir | sidecar lê direto do canal | |

**User's choice:** Arquivo global espelha um canal Codex marcado como primário
**Notes:** o arquivo existe por compatibilidade com o sidecar, não como source of truth.

| Option | Description | Selected |
|--------|-------------|----------|
| Escolha manual no `/admin/codex-auth` | admin define o primário | |
| Primeiro canal válido vira primário | convenção automática | |
| Canal com `backend=sdk` ativo vira primário | backend decide ownership | ✓ |

**User's choice:** Canal com `backend=sdk` ativo vira primário
**Notes:** resposta inicial do user. Em seguida, o user também travou que, com múltiplos `sdk`, o primário é escolhido manualmente — então planning deve tratar a seleção do primário como explícita quando houver ambiguidade.

| Option | Description | Selected |
|--------|-------------|----------|
| Só 1 canal `sdk` por vez | sem concorrência | |
| Pode ter vários; o mais recente salvo vira primário | heurística temporal | |
| Pode ter vários; escolhe manualmente qual é o primário | controle explícito | ✓ |

**User's choice:** Pode ter vários; escolhe manualmente qual é o primário
**Notes:** esta resposta refina a anterior: `backend=sdk` habilita participação, mas o primário final precisa de escolha manual quando houver múltiplos.

---

## Claude's Discretion

- Reload do sidecar: defaultado por timeout para `cache + mtime + reload automático`.
- Falha de licença no backend `sdk`: defaultado por timeout para `hard fail sem fallback silencioso para relay`.

## Deferred Ideas

None.
