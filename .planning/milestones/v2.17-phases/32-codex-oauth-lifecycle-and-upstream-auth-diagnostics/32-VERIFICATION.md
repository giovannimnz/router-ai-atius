---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
status: passed
verified: 2026-07-12
verified_at: 2026-07-12T14:08:00-03:00
score: 6/6 requirements complete
---

# Phase 32 Verification

| Requirement | Status | Evidence |
|---|---|---|
| UI Codex single endpoint | PASS | UI especifica type 57; smoke 4/4, typecheck e build verdes |
| OAuth regenerate | PASS | Callback PKCE concluido live; channel 5 `router_owned`, com `has_refresh_token=true` e renovacao propria do Router |
| Credential health | PASS | Metadata, probe e refresh live 200; `authenticated=true`, `last_probe_status=ok`, `requires_regeneration=false` |
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
- Regeneracao concluida em `2026-07-12T12:38:03-03:00`.
- Probe upstream concluido com `success=true` e `last_probe_status=ok`.
- Refresh manual concluido em `2026-07-12T12:47:27-03:00`; nova expiracao `2026-07-22T12:47:27-03:00`.
- Chat non-stream, chat stream e Responses stream repetidos local e publicamente: HTTP 200.
- Token interno invalido repetido publicamente: HTTP 401 classificado como auth interna do Router.
- Containers nao-infra continuam limitados a `0.800 CPU`.
- Boundary do warning Base URL type 57: teste 4/4, typecheck PASS e build Rsbuild PASS sob wrapper de 20% CPU.
- Integration checker: cinco fluxos E2E wired; nenhum export/route orfao. O negativo upstream destrutivo live foi substituido pela cobertura deterministica ja validada.
- Nyquist: `32-VALIDATION.md` compliant, 6/6 requisitos com cobertura automatizada; suite Go controller/service/relay/codex PASS com cache isolado.
- Security: `32-SECURITY.md`, 14/14 threats closed, `threats_open=0`.
- UI review: 19/24 code-only; as tres prioridades foram corrigidas; teste 5/5, typecheck e build PASS.

## Conclusao

O incidente `401 token_invalidated` nao esta ativo, o fallback foi substituido por uma credencial OAuth Router-owned renovavel e todos os contratos de API permanecem saudaveis. A Phase 32 esta concluida.
