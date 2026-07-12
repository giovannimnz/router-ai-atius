---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
plan: 32-02
status: complete
completed: 2026-07-12
---

# 32-02 Summary - UI OAuth Codex

O editor `OpenAI - Codex` nao exibe mais Base URL/API Key genericos. O painel apresenta somente metadata OAuth sanitizada e separa `Atualizar credencial`, `Regenerar credencial` e probe upstream, com fluxo PKCE e fallback por callback manual.

Validacao: smoke da UI 4/4, i18n, typecheck e Rsbuild de producao passaram pelo wrapper de CPU da fase.
