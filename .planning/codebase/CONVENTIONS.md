# CONVENTIONS.md - Coding Standards & Conventions

## Visão Geral

Este projeto é predominantemente **operacional** — consiste em scripts Bash para gerenciamento de infraestrutura Docker e scripts Python para integração com a API do NewAPI. Não há código de aplicação local (o NewAPI roda como imagem Docker pré-construída).

## Bash Scripts

### Estrutura Padrão

```bash
#!/bin/bash

# Comentário descritivo do script
# Autor: Sistema de IA
# Descrição: Breve descrição do propósito

set -e  # Sai se algum comando falhar

# Diretório base do script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
```

### Convenções de Nomenclatura

| Elemento | Convenção | Exemplo |
|---|---|---|
| **Arquivos** | `kebab-case.sh` | `disk-health.sh`, `backup-restore.sh` |
| **Variáveis** | `UPPER_SNAKE_CASE` | `SCRIPT_DIR`, `COMPOSE_CMD`, `THRESHOLD` |
| **Funções** | `snake_case` | `show_menu()`, `start_app()`, `backup_data()` |
| **Constantes** | `UPPER_SNAKE_CASE` | `POSTGRES_USER`, `SQL_DSN` |

### Padrões de Script

#### 1. Menu Interativo

Scripts de gerenciamento seguem o padrão `show_menu` + `while true` + `case`:

```bash
show_menu() {
    echo "=================================="
    echo " Título do Menu"
    echo "=================================="
    echo "1. Opção 1"
    echo "2. Opção 2"
    echo "N. Sair"
    echo "=================================="
}

while true; do
    show_menu
    read -p "Escolha uma opção [1-N]: " choice
    case $choice in
        1) action_one ;;
        2) action_two ;;
        N) exit 0 ;;
        *) echo "Opção inválida." ;;
    esac
    read -p "Pressione Enter para continuar..."
done
```

#### 2. Detecção de Docker Compose

Todos os scripts detectam automaticamente o comando compose disponível:

```bash
get_compose_cmd() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    elif command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    else
        echo "Erro: Docker Compose não está disponível no PATH" >&2
        exit 1
    fi
}
```

#### 3. Parse de Argumentos (scripts avançados)

```bash
while [[ $# -gt 0 ]]; do
  case "$1" in
    --threshold) THRESHOLD="$2"; shift 2 ;;
    --cleanup-safe) DO_CLEANUP=true; shift ;;
    *) echo "Uso: $0 [--threshold 95] [--cleanup-safe]" >&2; exit 1 ;;
  esac
done
```

### Tratamento de Erros

| Padrão | Uso |
|---|---|
| `set -e` | Sai imediatamente em caso de erro |
| `2>/dev/null \|\| true` | Suprime erros esperados sem falhar |
| `echo "Erro: ..." >&2` | Mensagens de erro para stderr |
| `exit 1` | Código de saída não-zero para falhas |
| `exit 2` | Código específico para condições críticas (ex: disco cheio) |

### Comentários

- Cabeçalho com **Autor** e **Descrição**
- Comentários inline em português para lógica importante
- Separadores visuais (`# ---`) para seções

## Python Scripts

### Scripts de Integração

| Script | Finalidade |
|---|---|
| `sync_deepseak_channels.py` | Sincroniza channels DeepSeek via API admin |
| `sync_openrouter_channels.py` | Sincroniza channels OpenRouter |
| `sync_iflow_channel_keys.py` | Sincroniza chaves de canais iFlow |
| `normalize_models_real_only.py` | Normaliza catálogo de modelos |

### Convenções Inferidas

| Elemento | Convenção |
|---|---|
| **Arquivos** | `snake_case.py` |
| **Dependências** | `requests` (implícito para chamadas HTTP) |
| **Config** | Lê variáveis de ambiente de `integration/.env` |
| **Autenticação** | Usa `NEWAPI_ADMIN_TOKEN` para chamadas à API admin |

## Docker & Compose

### Nomenclatura de Serviços

| Serviço | Convenção |
|---|---|
| App principal | `new-api` (nome direto) |
| Banco de dados | `db-newapi` (prefixo `db-`) |

### Nomenclatura de Containers

| Container | Convenção |
|---|---|
| `new-api` | Mesmo nome do serviço |
| `db-newapi` | Mesmo nome do serviço |

### Portas

| Serviço | Host | Container |
|---|---|---|
| NewAPI | 3300 | 3000 |
| PostgreSQL | 8746 | 5432 |

### Volumes

| Volume | Tipo | Descrição |
|---|---|---|
| `./data:/data` | Bind mount | Logs e dados da aplicação |
| `./data/postgres_data:/var/lib/postgresql/data` | Bind mount | Dados do PostgreSQL |

## Variáveis de Ambiente

### Hierarquia de Configuração

| Caminho | Prioridade | Uso |
|---|---|---|
| `integration/.env` | Principal | Variáveis unificadas (app + middleware + providers) |
| `.env` (raiz) | Standalone | Variáveis básicas (app + DB) |

### Grupos de Variáveis

| Prefixo | Grupo |
|---|---|
| `POSTGRES_*` | Banco de dados PostgreSQL |
| `SQL_*` | String de conexão SQL |
| `DEEPSEAK_*` | Provider DeepSeek |
| `NEWAPI_*` | Configurações NewAPI |
| `LITELLM_*` | Compatibilidade LiteLLM |
| `MODEL_*`, `RATE_LIMIT_*` | Middleware search-engine |
| `TZ`, `LANG`, `LC_*` | Locale do sistema |

## Documentação

### Arquivos de Documentação

| Arquivo | Conteúdo |
|---|---|
| `README.md` | Documentação principal com endpoints, cURL examples, troubleshooting |
| `DEVELOPMENT_GUIDE.md` | Guia de desenvolvimento com estrutura e opções de dev local |
| `SUMMARY.md` | Resumo executivo |
| `FILES_LOCATION.md` | Mapa de arquivos |
| `integration/GUIA_MUDANCA_CHAVES_API.md` | Guia de rotação de chaves |

### Convenções de Documentação

- **Idioma**: Português (Brasil) para conteúdo principal
- **Formatação**: Markdown com tabelas para dados estruturados
- **Code blocks**: Bash com syntax highlighting
- **Seções**: Headers com `##` e `###`
- **Tabelas**: Usadas extensivamente para preços, endpoints, configurações

## Padrões de Segurança

| Prática | Implementação |
|---|---|
| **Tokens em .env** | API keys definidas em variáveis de ambiente, não em código |
| **SSL Mode** | `sslmode=disable` no DSN (apenas para rede interna Docker) |
| **Aviso de troca** | README alerta para trocar credenciais padrão |
| **Healthcheck** | PostgreSQL com healthcheck antes do NewAPI iniciar |

## Padrões de Operacionalização

| Operação | Script | Método |
|---|---|---|
| Iniciar | `start.sh` | `docker compose up -d` |
| Parar | `management.sh` (opção 2) | `docker compose down` |
| Reiniciar | `management.sh` (opção 3) | `docker compose restart` |
| Aplicar .env | `reload-newapi.sh` | `docker compose up -d --force-recreate` |
| Backup | `backup-restore.sh` | `tar -czf` de `data/` |
| Limpeza | `disk-health.sh --cleanup-safe` | Remove cache e logs antigos |
| Recriar tudo | `recreate-all.sh` | Destrói e recria containers + dados |
