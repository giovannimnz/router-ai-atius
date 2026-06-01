# Phase Plan: Fork Git Setup

**Slug:** `fork-git-setup`
**Milestone:** v1.2
**Status:** `pending`

## Objetivo

Configurar git remotes no repo local para estabelecer workflow de fork formal conectando ao GitHub `giovannimnz/router-ai-atius` e ao upstream `QuantumNous/new-api`.

## Passos

### 1.1 — Adicionar remotes

```bash
cd /home/ubuntu/docker/ai-apps

# Adicionar origin (o fork GitHub)
git remote add origin https://github.com/giovannimnz/router-ai-atius.git

# Adicionar upstream (o repo original)
git remote add upstream https://github.com/QuantumNous/new-api.git

# Verificar configuração
git remote -v
```

### 1.2 — Verificar conectividade

```bash
# Testar fetch do upstream
git fetch upstream --prune

# Testar fetch do origin
git fetch origin --prune
```

### 1.3 — Comparar história git

```bash
# Verificar se há relação entre história local e upstream
git log --oneline --graph --all --decorate -20

# Verificar tags do upstream
git tag -l | head -20

# Verificar se história local está à frente ou atrás do upstream
git rev-list --left-right --count HEAD...upstream/main 2>/dev/null || echo "No upstream main yet"
```

### 1.4 — Push inicial para origin (se história compatível)

```bash
# Forçar push do branch main para origin
# ATENÇÃO: isto sobrescreve o remote. Apenas se o fork estiver vazio.
git push -u origin main --force

# Ou criar branch separate se não quiser sobrescrever
# git checkout -b atius-fork
# git push -u origin atius-fork
```

### 1.5 — Configurar git user para fork

Verificar se git user está configurado:
```bash
git config user.name
git config user.email
```

Se não estiver, configurar:
```bash
git config user.name "Giovanni MNZ"
git config user.email "giovannimnz@..."
```

## Arquivos a Criar/Modificar

| Arquivo | Ação | Conteúdo |
|---------|------|----------|
| `.git/config` | Modificar | Adicionar origin e upstream remotes |

## Comandos para Verificação

```bash
# Verificar remotes
git remote -v
# Deve mostrar:
# origin   https://github.com/giovannimnz/router-ai-atius.git (fetch)
# upstream https://github.com/QuantumNous/new-api.git (fetch)

# Verificar fetch
git fetch --all
echo "Exit code: $?"
```

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| Repo origin não está vazio | Usar `--force` se necessário, ou criar branch separado |
|gh` não autenticado | Autenticar com `gh auth login` antes de push |
| História local incompatível com upstream | Manter branch local separado, não fazer force push |

## Dependencies

Nenhuma — esta é a primeira fase.

## Tempo Estimado

15-30 minutos

## Criteria de Completion

- [ ] `git remote -v` mostra origin e upstream corretos
- [ ] `git fetch upstream` funciona sem erro
- [ ] `git fetch origin` funciona sem erro
- [ ] `git log` mostra histórico local
- [ ] Commit inicial feito no remote origin (se fork vazio)
