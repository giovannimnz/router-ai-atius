---
title: Docs Convergence to docs/atius-router-docs
status: accepted
date: 2026-06-07
scope: phase-09-docs-convergence-main-repo
---

# Docs Convergence to `docs/atius-router-docs/`

## Context

A documentação do Atius Router era tratada como checkout standalone em `/home/ubuntu/docker/Atius/atius-router-docs`.
A Phase 09 precisa mover o runtime e o fluxo operacional para dentro do repo principal `router-ai-atius` sem perder o source forkado nem o cutover controlado.

## Decision

- O path canônico da docs passa a ser `docs/atius-router-docs/` dentro de `router-ai-atius`.
- O path é registrado como submodule apontando para o fork atual `https://github.com/giovannimnz/new-api-docs-v1` na branch `main`.
- Developers e automações inicializam o tree com `git submodule update --init --recursive`.
- O checkout standalone legado permanece apenas como fonte de migração/rollback até a validação final das fases 09-02 e 09-03.

## Threat Model

- Ameaça: clone/deploy sem submodule inicializado.
- Impacto: runtime sobe sem source, build falha ou serve conteúdo incompleto.
- Mitigação: checklist obrigatório de `git submodule update --init --recursive` + smoke check explícito antes do build/deploy.
- Rollback safety: manter o checkout standalone intacto até o cutover final e restaurar o runtime para o path legado se a validação falhar.

## Consequences

- O runtime pode ser apontado para o path integrado sem depender do checkout standalone como source-of-truth.
- O sync/patch/deploy passa a ter um target estável dentro do repo principal.
- Rollback continua possível voltando o runtime para o path legado enquanto o submodule fica preservado.

## Links

- [[../../README.md]]
- [[../../.planning/phases/09-docs-convergence-main-repo/09-01-PLAN.md]]
- [[../../.planning/phases/09-docs-convergence-main-repo/09-02-PLAN.md]]
- [[../../.planning/phases/09-docs-convergence-main-repo/09-03-PLAN.md]]
