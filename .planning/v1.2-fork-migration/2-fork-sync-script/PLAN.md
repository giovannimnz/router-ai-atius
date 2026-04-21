# Phase Plan: Fork Sync Script

**Slug:** `fork-sync-script`
**Milestone:** v1.2
**Status:** `pending`
**Depends on:** `fork-git-setup` (Phase 1)

## Objetivo

Criar `scripts/sync-fork.sh` que automatiza o merge do upstream com proteção de modificações locais.

## Referência

Template: `/home/ubuntu/GitHub/forks/openclaude/scripts/sync-fork.sh`

## Passos

### 2.1 — Criar diretório scripts

```bash
mkdir -p /home/ubuntu/docker/ai-apps/new-api/scripts
```

### 2.2 — Criar sync-fork.sh

Estrutura do script:

```bash
#!/usr/bin/env bash
# sync-fork.sh — Sync atius-ai-router with upstream NewAPI
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/QuantumNous/new-api.git}"
UPSTREAM_NAME="upstream"
BRANCH="${SYNC_BRANCH:-main}"
STRATEGY="${SYNC_STRATEGY:-theirs}"  # theirs = prefer upstream, ours = prefer fork
DRY_RUN=false

# Parse arguments: --strategy, --dry-run, --branch, -h
# 1. Add/set upstream remote
# 2. Fetch upstream
# 3. Checkout target branch
# 4. Pull from origin
# 5. Merge upstream with -X $STRATEGY
# 6. Restore protected files:
#    - integration/middleware/model_detailed.py (never overwrite)
#    - .planning/ (never overwrite)
#    - docker-compose.yml (re-apply customizations)
#    - .github/workflows/ (preserve fork-specific workflows)
# 7. Version bump
# 8. Push to origin
```

### 2.3 — Definir arquivos protegidos

```bash
# Arquivos que nunca devem ser sobrescritos pelo upstream
PROTECTED_PATTERNS=(
    "integration/middleware/model_detailed.py"
    ".planning/"
)

# Arquivos que precisam de restauração pós-merge
RESTORE_PATTERNS=(
    "docker-compose.yml"
)
```

### 2.4 — Implementar lógica de proteção

```bash
restore_protected_files() {
    # Para cada arquivo protegido, verificar se foi modificado pelo merge
    # Se foi, restaurar da versão do fork (stash ou git checkout HEAD)
    for file in "${PROTECTED_PATTERNS[@]}"; do
        if [[ -f "$file" ]] && git diff --quiet HEAD -- "$file" 2>/dev/null; then
            # Não houve mudança, OK
        else
            # Restaura da versão do fork
            git checkout HEAD -- "$file"
            echo "  Restored protected: $file"
        fi
    done
}
```

### 2.5 — Testar com dry-run

```bash
cd /home/ubuntu/docker/ai-apps/new-api
./scripts/sync-fork.sh --dry-run
```

## Arquivos a Criar

| Arquivo | Conteúdo |
|---------|----------|
| `scripts/sync-fork.sh` | Script principal de sync |
| `scripts/sync-fork.sh.README.md` | Documentação de uso |

## Comandos para Verificação

```bash
# Testar help
./scripts/sync-fork.sh --help

# Testar dry-run
./scripts/sync-fork.sh --dry-run

# Verificar que protected files não são sobrescritos
git status integration/middleware/model_detailed.py
```

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Protected files sobrescritos | Script faz `git checkout HEAD -- <file>` após merge |
| Conflitos de merge | Abortar e pedir intervenção manual se merge falhar |
| .planning/no upstream | Marcar como `protected` via padrão, nunca conflita |

## Dependencies

- Phase 1 (fork-git-setup)

## Tempo Estimado

1-2 horas

## Criteria de Completion

- [ ] `scripts/sync-fork.sh` existe e é executável
- [ ] `--help` mostra documentação correta
- [ ] `--dry-run` executa sem erro
- [ ] `--strategy ours` funciona
- [ ] `--strategy theirs` funciona
- [ ] `integration/middleware/model_detailed.py` preservado após sync
- [ ] `.planning/` preservado após sync
