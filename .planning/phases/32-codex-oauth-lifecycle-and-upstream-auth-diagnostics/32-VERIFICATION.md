---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
status: blocked
verified: 2026-07-12
score: 5/6 requirements complete
---

# Phase 32 Verification

| Requirement | Status | Evidence |
|---|---|---|
| UI Codex single endpoint | PASS | UI especifica type 57; smoke 4/4, typecheck e build verdes |
| OAuth regenerate | BLOCKED | Fluxo implementado; live sem login ChatGPT e Vault selado; channel 5 ainda sem refresh token |
| Credential health | PASS | Metadata e probe live 200; `authenticated=true`, `last_probe_status=ok`, `requires_regeneration=true` coerente |
| Upstream auth errors | PASS | Taxonomia e testes; negativo interno 401 sem marcador upstream |
| Fork-sync guard | PASS | Checker verde; `omni-srv-admin` commit `9dd574597` |
| Validation/docs/ship | PASS | Testes Go, UI, build 20%, deploy, smokes publicos, docs, commits e pushes concluidos |

## Provas live

- Imagem: `ec0f29ea91546d4bfa70b3e71aba8c01eace165ac0aaddf75b91c186d3c3123b`.
- Catalogo Codex: 6 abilities, incluindo `gpt-5.6-sol` e `gpt-5.6-terra`.
- `/v1/models`: 200 local e publico, 9 modelos, raiz `data`.
- `gpt-5.6-sol` chat non-stream: 200.
- `gpt-5.4-mini` chat stream: 200.
- `gpt-5.6-terra` Responses stream: 200.
- `clianything status --strict`: PASS.
- Go: controller/service/relay/codex PASS.

## Conclusao

O incidente `401 token_invalidated` nao esta ativo e todos os contratos de API estao saudaveis. A fase permanece bloqueada somente no handoff OAuth humano necessario para substituir o fallback por credencial Router-owned renovavel.
