# Phase Plan: FORK.md Documentation

**Slug:** `fork-fork-md`
**Milestone:** v1.2
**Status:** `pending`
**Depends on:** Phase 2 (sync script) and Phase 3 (version bump)

## Objetivo

Documentar todas as modificações locais ao fork para referência durante merges futuros.

## Referência

Template: `/home/ubuntu/GitHub/forks/openclaude/FORK.md`

## Estrutura do FORK.md

```markdown
# Fork Technical Documentation — atius-ai-router

## Parent
https://github.com/QuantumNous/new-api

## Fork Remote
https://github.com/giovannimnz/atius-ai-router

## Version
Current: X.Y.Z.N (see VERSION file)

## Modificações Locais

### 1. Model Metadata Enrichment Middleware
**File:** `integration/middleware/model_detailed.py`
**Since:** v1.1 (2026-04-14)

Python middleware proxy que intercepta GET /v1/models e adiciona metadados:
- context_length, max_completion_tokens, pricing
- Modelos: deepseek-chat, deepseek-reasoner, MiniMax-M2.7, MiniMax-M2.5

### 2. Docker Compose Customization
**File:** `docker-compose.yml`
**Since:** v1.0 (2026-04-12)

Customizações:
- CPU limits específicos (new-api: 0.5, model-detailed: 0.1)
- Serviço model-detailed adicionado
- Redes: newapi-internal, atius-shared

### 3. GSD Workflow State
**Dir:** `.planning/`
**Since:** v1.0 (2026-04-12)

Workflow GSD v1 com:
- Milestones v1.0 e v1.1 completos
- Roadmap para v1.2 (fork migration)
- Documentação de arquitetura

## Keeping Fork Updated

### Manual Sync
./scripts/sync-fork.sh [--strategy ours|theirs] [--dry-run]

### Version Bump
./scripts/version-bump.sh [--check]

## Protected Files

Os seguintes arquivos são protegidos pelo sync-fork.sh e não serão sobrescritos:
- `integration/middleware/model_detailed.py`
- `.planning/`
- `docker-compose.yml`
- `.github/workflows/` (fork-specific CI/CD)
```

## Passos

### 4.1 — Criar FORK.md

Copiar estrutura do template e adaptar.

### 4.2 — Documentar cada modificação local

1. `model_detailed.py` — middleware de enriquecimento
2. `docker-compose.yml` — customização docker
3. `.planning/` — estado do workflow GSD

### 4.3 — Adicionar troubleshooting section

```markdown
## Troubleshooting

### Sync resulted in overwritten protected file
Run: ./scripts/sync-fork.sh --dry-run
If protected file was overwritten, it will be restored automatically on next sync.

### Version not updating
Check: cat VERSION
Run: ./scripts/version-bump.sh --check
```

## Dependencies

- Phase 2 (fork-sync-script)
- Phase 3 (fork-version-bump)

## Tempo Estimado

30-60 minutos

## Criteria de Completion

- [ ] `FORK.md` existe no root
- [ ] Todas modificações locais documentadas
- [ ] Seção de sync e versionamento incluída
- [ ] Protected files listados
- [ ] Troubleshooting section presente
