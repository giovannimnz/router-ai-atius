# Legacy Planning Archive - 2026-07-09

Arquivamento estrutural para reduzir dívida do `.planning/` sem perder histórico.

Itens movidos para fora de `.planning/phases`:

- `phase-1-router-anthropic-channels`
- `phase-2-claude-models-endpoint`
- `phase-3-model-unification`
- `phase-4-session-fix`
- `phase-cjk-strip`
- `v1.7-cjk-strip-filter`

Item movido para fora da raiz de `.planning`:

- `FORK_MIGRATION.md`

Motivo:

- esses artefatos são históricos/legacy
- não seguem mais o padrão canônico `NN-name`
- estavam degradando `validate.health`
- o histórico foi preservado por movimento, não por deleção
