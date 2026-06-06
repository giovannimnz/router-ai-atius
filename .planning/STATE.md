---
gsd_state_version: 1.0
milestone: v2.14
milestone_name: Codex SDK Transformer
status: planning
last_updated: "2026-06-06T21:36:06.066Z"
last_activity: 2026-06-06
progress:
  total_phases: 4
  completed_phases: 1
  total_plans: 1
  completed_plans: 1
  percent: 25
---

# STATE.md

**Project:** Atius AI Router
**Current milestone:** v2.14 — Codex SDK Transformer
**Status:** Ready to plan

## Current Position

Phase: 06
Plan: Not started
Status: Phase 06 context gathered
Last activity: 2026-06-06 — Phase 06 context gathered

## v2.12 Progress (CLOSED ✅)

| Phase | Status | Date |
|---|---|---|
| 01 — pt Locale Registration (Go + React SPA) | ✅ Complete | 2026-06-05 |
| 02 — pt Fumadocs i18n (Docs) | ✅ Complete | 2026-06-05 |
| 03 — PT Docs Bugfixes (hreflang + guide) | ✅ Complete | 2026-06-05 |
| 04 — Prod Docs Bugfixes (Apache + logo + lang order) | ✅ Complete | 2026-06-06 |

## v2.14 Progress (ACTIVE 🔵)

| Phase | Status | Date |
|---|---|---|
| 05 — Sidecar Python + HTTP Bridge (SDK-01) | ⏳ Not Started | — |
| 06 — Login Explícito + Armazenamento Licença (SDK-02) | ⏳ Not Started | — |
| 07 — Dashboard Usage/Saldo (SDK-03) | ⏳ Not Started | — |
| 08 — Channel Coexistence + Validação (SDK-04) | ⏳ Not Started | — |

## ⚡ EXECUTION ORDER

🟡 1. **Phase 05** (sidecar Python) ← primeiro (foundation — sem sidecar nada funciona)
🟡 2. **Phase 06** (login + licença) ← depende do sidecar existente (SDK precisa autenticar)
🟢 3. **Phase 07** (usage dashboard) ← paralelizável com 06 (só precisa do wham/usage endpoint)
🟢 4. **Phase 08** (coexistence + validação) ← último, integra tudo

**Porquê esta ordem:**

- Phase 05 antes: o sidecar é o core. Sem ele, não tem o que autenticar nem testar.
- Phase 06 depende de 05: o fluxo de login escreve `data/codex/license.json` que o sidecar lê.
- Phase 07 é parcialmente independente: o endpoint `wham/usage` já existe no relay HTTP.
  Pode ser desenvolvido em paralelo com 06 se quiser.

- Phase 08 fecha: valida coexistência relay/sdk, testa fluxo completo.

## Summary

v2.12 (pt i18n) shipped ✅. v2.14 (Codex SDK Transformer) adds a Python sidecar
that bridges the router Go → Codex SDK, with explicit login (Hermes-style),
usage dashboard, and channel coexistence. 4 phases, 4 requirements, 1:1 mapping.

## Last Activity

2026-06-06: v2.14 milestone started. REQUIREMENTS.md + ROADMAP.md created.
4 phases (05-08) defined. Phase 05 (Sidecar Python) is next.
