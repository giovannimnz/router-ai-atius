---
phase: 32-codex-oauth-lifecycle-and-upstream-auth-diagnostics
plan: 32-03
status: complete
completed: 2026-07-12
---

# 32-03 Summary - Operacao e fork-sync

O runbook PT-BR documenta refresh, regeneracao, browser-assisted callback, fallback manual e break-glass via access token do Codex CLI sem copiar `refresh_token`. As guardas do fork-sync preservam rotas, UI, relay e taxonomia de erros Codex.

Validacao: checker do workflow passou e o dry-run do fork-sync classificou os novos paths como protegidos. Commit no `omni-srv-admin`: `9dd574597`.
