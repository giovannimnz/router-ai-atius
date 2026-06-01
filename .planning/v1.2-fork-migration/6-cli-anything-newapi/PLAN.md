# Phase Plan: CLI-Anything for NewAPI Management

**Slug:** `cli-anything-newapi`
**Milestone:** v1.2
**Status:** `pending`
**Depends on:** Phase 1 (fork-git-setup) — pode iniciar em paralelo após Fase 1

## Objetivo

Usar CLI-Anything para gerar um CLI que permite gerenciar o NewAPI (channels, models, containers Docker) de forma agent-native, facilitando operações via agente.

## O que é CLI-Anything

Framework que gera automaticamente CLIs para qualquer software. O CLI gerado:
- Tem comandos estruturados com `--json` output para agentes
- Suporta REPL interativo
- Tem --help completo para auto-descoberta
- É instalável via `pip install -e .`

## Referências

- Repo: https://github.com/HKUDS/CLI-Anything
- Plugin: `/home/ubuntu/GitHub/forks/openclaude/` (já tem CLI-Anything instalado)
- SKILL.md do plugin: 参考 `cli_anything/` no repo HKUDS

## Funcionalidades Desejadas para NewAPI CLI

### 6.1 — Docker Management

```bash
# Gerenciar containers
newapi-cli container list
newapi-cli container status
newapi-cli container restart
newapi-cli container logs --follow

# Rebuild
newapi-cli container rebuild
```

### 6.2 — Channel Management

```bash
# Listar channels
newapi-cli channel list
newapi-cli channel info --name deepseek-chat

# CRUD de channels
newapi-cli channel create --name xxx --model xxx --base-url xxx
newapi-cli channel update --name xxx --key new-key
newapi-cli channel delete --name xxx
```

### 6.3 — Model Management

```bash
# Listar models disponíveis
newapi-cli model list

# Ver metadados enriquecidos (via middleware)
newapi-cli model info --name deepseek-chat
```

### 6.4 — API Operations

```bash
# Testar endpoints
newapi-cli api models --json
newapi-cli api chat --model deepseek-chat --prompt "Hello"
```

### 6.5 — Database Operations

```bash
# Ver quotas e usage
newapi-cli db quotas --user atius
newapi-cli db channels --list
```

## Abordagem de Implementação

### Opção A: CLI-Anything Plugin (Recomendado se disponível)

```bash
# Usar o plugin CLI-Anything do OpenClaude
# O plugin já está em /home/ubuntu/GitHub/forks/openclaude/

# Gerar CLI para o NewAPI
# Dependendo de como o OpenClaude fork está configurado, pode usar:
/home/ubuntu/GitHub/forks/openclaude/bin/openclaude --dangerously-skip-permissions \
  --cli-anything /home/ubuntu/docker/ai-apps/new-api
```

**Desafio:** O CLI-Anything no OpenClaude fork pode não estar instalado como plugin. Precisamos verificar.

### Opção B: Gerar CLI Manualmente (Mais controlável)

Criar CLI do zero usando Click, baseado na estrutura NewAPI:

```bash
# Estrutura do projeto
new-api/
├── agent-harness/
│   ├── cli_newapi/
│   │   ├── __init__.py
│   │   ├── cli.py           # Click CLI principal
│   │   ├── docker.py        # Comandos Docker
│   │   ├── channel.py       # Channel management
│   │   ├── model.py         # Model info
│   │   └── api.py           # API testing
│   ├── setup.py
│   └── tests/
│       ├── test_channel.py
│       └── test_docker.py
```

### Opção C: Script hibrido (Mais rápido)

Criar scripts bash simples + Python para operations críticas, sem Click:

```bash
#!/bin/bash
# newapi-cli - CLI para NewAPI

COMMAND="$1"
shift

case "$COMMAND" in
  channel)
    ./scripts/newapi-channels.sh "$@"
    ;;
  container)
    docker "$@"
    ;;
  model)
    curl -s http://localhost:3300/v1/models | jq "$@"
    ;;
esac
```

## Recomendação

**Opção B** (CLI Click manual) é mais robusta e alinhada com o framework CLI-Anything, mas leva mais tempo.

**Opção C** (Scripts) é mais rápida e funcional, ideal para MVP.

Decisão: Começar com **Opção C** (MVP funcional) e iterar para **Opção B**.

## Passos

### 6.1 — Analisar NewAPI

Entender a estrutura do NewAPI:
- Endpoints da API admin
- Estrutura do banco de dados
- Volumes Docker e logging
- Environment variables

```bash
# Explorar API admin
curl -s http://localhost:3300/api/v1/info 2>/dev/null | jq .
curl -s http://localhost:3300/api/v1/channels 2>/dev/null | jq .

# Ver estrutura do banco
docker exec db-newapi psql -U atius -d atius -c "\dt"
```

### 6.2 — Criar estrutura CLI

```bash
mkdir -p /home/ubuntu/docker/ai-apps/new-api/agent-harness
mkdir -p /home/ubuntu/docker/ai-apps/new-api/agent-harness/cli_newapi
mkdir -p /home/ubuntu/docker/ai-apps/new-api/agent-harness/tests
mkdir -p /home/ubuntu/docker/ai-apps/new-api/agent-harness/scripts
```

### 6.3 — Criar CLI com Click

Criar `cli_newapi/cli.py` com Click:

```python
#!/usr/bin/env python3
import click
import subprocess
import requests
import json

@click.group()
@click.pass_context
def cli(ctx):
    """router-ai-atius CLI — NewAPI management for agents"""
    ctx.ensure_object(dict)
    ctx.obj['base_url'] = 'http://localhost:3300'

# Docker commands
@cli.group()
def container():
    """Docker container management"""
    pass

@container.command('list')
def container_list():
    """List NewAPI containers"""
    result = subprocess.run(['docker', 'ps', '--filter', 'name=new-api', '--format', 'json'],
                          capture_output=True, text=True)
    for line in result.stdout.splitlines():
        print(line)

# Channel commands
@cli.group()
def channel():
    """Channel management"""
    pass

@channel.command('list')
@click.pass_context
def channel_list(ctx):
    """List all channels"""
    r = requests.get(f"{ctx.obj['base_url']}/api/v1/channels")
    if r.ok:
        data = r.json()
        click.echo(json.dumps(data, indent=2))
    else:
        click.echo(f"Error: {r.status_code}", err=True)

# ... mais comandos
```

### 6.4 — Criar setup.py

```python
from setuptools import setup, find_packages

setup(
    name='cli-anything-newapi',
    version='0.1.0',
    packages=find_packages(),
    install_requires=['click', 'requests'],
    entry_points={
        'console_scripts': [
            'newapi-cli=cli_newapi.cli:cli',
        ],
    },
)
```

### 6.5 — Criar SKILL.md

Para agentes descobrirem o CLI:

```markdown
---
name: newapi-cli
description: CLI for managing Atius NewAPI gateway (channels, models, containers)
---

# newapi-cli

CLI for managing Atius NewAPI LLM gateway.

## Installation
pip install -e agent-harness/

## Commands

### container
Docker container management.
- `newapi-cli container list` — List containers
- `newapi-cli container restart` — Restart container
- `newapi-cli container logs` — View logs

### channel
Channel management.
- `newapi-cli channel list` — List all channels
- `newapi-cli channel info --name <name>` — Get channel details
- `newapi-cli channel create --name <name> --model <model> --base-url <url>` — Create channel

### model
Model information.
- `newapi-cli model list` — List available models
- `newapi-cli model info --name <name>` — Get model metadata

## JSON Output
Use `--json` flag for machine-readable output.
```

## Arquivos a Criar

| Arquivo | Conteúdo |
|---------|----------|
| `agent-harness/cli_newapi/__init__.py` | Package init |
| `agent-harness/cli_newapi/cli.py` | Click CLI principal |
| `agent-harness/cli_newapi/docker.py` | Docker commands |
| `agent-harness/cli_newapi/channel.py` | Channel commands |
| `agent-harness/cli_newapi/model.py` | Model commands |
| `agent-harness/setup.py` | Package setup |
| `agent-harness/SKILL.md` | Skill documentation |
| `agent-harness/tests/test_channel.py` | Unit tests |
| `agent-harness/tests/test_docker.py` | Unit tests |

## Dependencies

- Phase 1 (fork-git-setup) — necessário para ter remotes configurados se quisermos usar o plugin CLI-Anything
- Esta fase pode rodar em paralelo com Fases 2 e 3

## Tempo Estimado

2-4 horas (MVP funcional)

## Criteria de Completion

- [ ] `agent-harness/` existe com estrutura CLI-Anything
- [ ] `pip install -e agent-harness/` funciona
- [ ] `newapi-cli --help` funciona
- [ ] `newapi-cli container list` funciona
- [ ] `newapi-cli channel list` funciona
- [ ] `newapi-cli model list --json` funciona
- [ ] `SKILL.md` existe e é válido
- [ ] Agente consegue usar o CLI para operações básicas
