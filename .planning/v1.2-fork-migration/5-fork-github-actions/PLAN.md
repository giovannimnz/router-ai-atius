# Phase Plan: GitHub Actions CI/CD

**Slug:** `fork-github-actions`
**Milestone:** v1.2
**Status:** `pending`
**Depends on:** Phase 2 (sync script) and Phase 3 (version bump)

## Objetivo

Configurar GitHub Actions workflows para sync automático e releases.

## Workflows a Criar

### 5.1 — `.github/workflows/sync.yml`

Disparado semanalmente ou via schedule.

```yaml
name: Weekly Upstream Sync

on:
  schedule:
    - cron: '0 3 * * 1'  # Toda segunda-feira 3:00 UTC
  workflow_dispatch:  # Permitir trigger manual

jobs:
  sync:
    runs-on: ubuntu-latest
    if: ${{ github.repository == 'giovannimnz/atius-ai-router' }}

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Git
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"

      - name: Add upstream remote
        run: |
          git remote add upstream https://github.com/QuantumNous/new-api.git
          git fetch upstream --prune

      - name: Run sync script
        run: ./scripts/sync-fork.sh --strategy theirs

      - name: Create PR if changes
        if: github.event_name == 'workflow_dispatch'
        run: |
          gh pr create --title "chore: sync with upstream" --body "Automated sync"
```

### 5.2 — `.github/workflows/release.yml`

Disparado em push de tags `v*`.

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    if: ${{ github.repository == 'giovannimnz/atius-ai-router' }}

    steps:
      - uses: actions/checkout@v4

      - name: Build and push Docker image
        run: |
          docker build -t ghcr.io/giovannimnz/atius-ai-router:${{ github.ref_name }} .
          docker push ghcr.io/giovannimnz/atius-ai-router:${{ github.ref_name }}

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v1
        with:
          files: VERSION
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Arquivos a Criar

| Arquivo | Conteúdo |
|---------|----------|
| `.github/workflows/sync.yml` | Workflow de sync semanal |
| `.github/workflows/release.yml` | Workflow de release em tag |

## Configuração Adicional

### packages write permission for Docker push

No repositório GitHub, ir em Settings > Actions > General > Workflow permissions
e habilitar "Read and write permissions".

## Dependencies

- Phase 2 (fork-sync-script)
- Phase 3 (fork-version-bump)

## Tempo Estimado

1-2 horas (inclui configuração GitHub)

## Criteria de Completion

- [ ] `.github/workflows/sync.yml` existe e é válido YAML
- [ ] `.github/workflows/release.yml` existe e é válido YAML
- [ ] `if: github.repository == 'giovannimnz/atius-ai-router'` presente em ambos
- [ ] Workflowsdisparam em push (testar com branch vazia ou tag test)
