# Phase Plan: Version Bump Script

**Slug:** `fork-version-bump`
**Milestone:** v1.2
**Status:** `pending`
**Depends on:** `fork-git-setup` (Phase 1)

## Objetivo

Criar script que gerencia versionamento do fork baseado no upstream, usando sufixo incremental.

## Version Scheme

Formato: `X.Y.Z.N`
- `X.Y.Z` = versão base do upstream (ex: `2.1.3`)
- `N` = sufixo incremental do fork (ex: `1`, `2`, `3`)

Exemplos:
- `2.1.3.1` — primeiro release do fork sobre upstream `2.1.3`
- `2.1.3.2` — segundo release, upstream ainda `2.1.3`
- `2.2.0.1` — upstream mudou para `2.2.0`, sufixo resetado

## Passos

### 3.1 — Criar version-bump.sh

NewAPI é Go-based (usa Docker image), então não tem `package.json`. Usar:
- Git tags no formato `vX.Y.Z.N`
- Arquivo `VERSION` no root para tracking local

```bash
#!/usr/bin/env bash
# version-bump.sh — Bump atius-ai-router version

# 1. Parse arguments: --check (dry run)
# 2. Fetch upstream tags
# 3. Get latest upstream tag (X.Y.Z format)
# 4. Read current fork version from VERSION file or latest tag
# 5. Compare base versions
#    - Se mudou: suffix = 1
#    - Se igual: suffix++
# 6. Se dry-run, mostrar resultado
# 7. Se real: criar tag vX.Y.Z.N, atualizar VERSION file
```

### 3.2 — Definir VERSION file

```bash
# /home/ubuntu/docker/ai-apps/new-api/VERSION
2.1.3.1
```

Formato: apenas a versão, uma linha.

### 3.3 — Implementar lógica de parse

```bash
parse_version() {
    local version="$1"
    # Se tem sufixo (.N), extrair base e suffix
    if [[ "$version" =~ ^([0-9]+\.[0-9]+\.[0-9]+)\.([0-9]+)$ ]]; then
        base="${BASH_REMATCH[1]}"
        suffix="${BASH_REMATCH[2]}"
    else
        base="$version"
        suffix="0"
    fi
    echo "$base $suffix"
}
```

### 3.4 — Comparar com upstream

```bash
get_upstream_version() {
    # Opção 1: git describe --tags upstream/main
    # Opção 2: GitHub API para latest release
    # Opção 3: Parse docker image tag do upstream

    # Para NewAPI (calciumion/new-api:latest), verificar:
    # https://hub.docker.com/r/calciumion/new-api/tags
    curl -s "https://hub.docker.com/v2/repositories/calciumion/new-api/tags/latest" | jq -r .name
}
```

### 3.5 — Criar tag e atualizar VERSION

```bash
bump_version() {
    local new_version="$1"
    echo "$new_version" > VERSION
    git add VERSION
    git commit -m "chore: bump version to $new_version"
    git tag -a "v$new_version" -m "Version $new_version"
}
```

## Arquivos a Criar

| Arquivo | Conteúdo |
|---------|----------|
| `scripts/version-bump.sh` | Script principal de version bump |
| `VERSION` | Arquivo com versão atual |

## Comandos para Verificação

```bash
# Testar parse
./scripts/version-bump.sh --check

# Verificar tag atual
git describe --tags 2>/dev/null || cat VERSION

# Verificar todas as tags
git tag -l 'v*'
```

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Upstream não tem tags | Usar Docker Hub API para detectar versão |
| VERSION file desatualizado | Sempre ler de VERSION file, não de git describe |
| Conflito de tags | Usar formato `vX.Y.Z.N` (com 'v' prefix) |

## Dependencies

- Phase 1 (fork-git-setup)

## Tempo Estimado

30-60 minutos

## Criteria de Completion

- [ ] `scripts/version-bump.sh` existe e é executável
- [ ] `--check` mostra versão atual sem modificar
- [ ] `VERSION` file existe com versão válida
- [ ] `git tag -l 'v*'` mostra tags no formato correto
- [ ] `version-bump.sh` (sem --check) cria tag e committa
