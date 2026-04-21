# STRUCTURE.md - Directory Layout & Module Organization

## Visão Geral

O projeto é organizado como um **deployment Docker** com scripts de operacionalização em Bash e scripts de integração em Python. O código da aplicação NewAPI está embutido na imagem Docker — não há código-fonte local do gateway.

## Árvore de Diretórios

```
/home/ubuntu/docker/ai-apps/new-api/
│
├── docker-compose.yml           # Compose principal (standalone)
├── .env                         # Variáveis de ambiente (app + DB)
│
├── README.md                    # Documentação principal
├── SUMMARY.md                   # Resumo do projeto
├── DEVELOPMENT_GUIDE.md         # Guia de desenvolvimento
├── FILES_LOCATION.md            # Localização de arquivos
│
├── start.sh                     # Inicialização da aplicação
├── management.sh                # Menu interativo de gerenciamento
├── reload-newapi.sh             # Recria container com force-recreate
├── recreate-all.sh              # Recriação completa (com perda de dados)
├── backup-restore.sh            # Backup e restauração de dados
├── disk-health.sh               # Monitoramento de disco e limpeza
│
├── data/                        # Persistência (bind mount)
│   ├── logs/                    # Logs da aplicação
│   ├── postgres_data/           # Dados do PostgreSQL
│   └── one-api.db               # Legacy SQLite (possivelmente não usado)
│
├── integration/                 # Scripts e configs de integração
│   ├── docker-compose.yml       # Compose com redes avançadas
│   ├── .env                     # Variáveis unificadas (app + middleware)
│   │
│   ├── scripts/                 # Scripts de integração
│   │   ├── sync_deepseak_channels.py      # Sync channels DeepSeek
│   │   ├── sync_openrouter_channels.py    # Sync channels OpenRouter
│   │   ├── sync_iflow_channel_keys.py     # Sync chaves iFlow
│   │   ├── normalize_models_real_only.py  # Normaliza modelos
│   │   ├── test_all_models.sh             # Testa todos os modelos
│   │   ├── update_api_keys.sh             # Atualiza API keys
│   │   ├── update_newapi_safe.sh          # Update seguro
│   │   ├── verify_stack.sh                # Verifica saúde do stack
│   │   └── backup_integration_state.sh    # Backup estado
│   │
│   ├── searxng/
│   │   └── settings.yml         # Configuração do SearXNG
│   │
│   ├── whisper.cpp/             # Whisper.cpp (STT)
│   │
│   ├── search-engine/           # Middleware de busca
│   │
│   ├── backups/                 # Backups de integração
│   ├── Models_gsd.json          # Catálogo de modelos
│   └── GUIA_MUDANCA_CHAVES_API.md  # Guia de chaves de API
│
├── .planning/                   # Planejamento GSD
│   └── codebase/                # Documentos de mapeamento
│       ├── STACK.md
│       ├── INTEGRATIONS.md
│       ├── ARCHITECTURE.md
│       ├── STRUCTURE.md
│       ├── CONVENTIONS.md
│       ├── TESTING.md
│       └── CONCERNS.md
│
├── .artifacts/                  # Artefatos de build
│   └── browser/
│
├── .bg-shell/                   # Shell background
│   └── manifest.json
│
└── .gsd/                        # GSD workflow
    └── notifications.jsonl
```

## Módulos e Responsabilidades

### Módulo Core (raiz)

| Arquivo | Responsabilidade |
|---|---|
| `docker-compose.yml` | Definição dos serviços Docker (NewAPI + PostgreSQL) |
| `.env` | Configurações de ambiente (DB creds, locale, SQL_DSN) |
| `start.sh` | Script de inicialização com detecção automática de compose |
| `management.sh` | Menu interativo (13 opções) para gerenciar o ciclo de vida |
| `reload-newapi.sh` | Recria container NewAPI com `--force-recreate` para aplicar `.env` |
| `recreate-all.sh` | Recriação completa com perda de dados |
| `backup-restore.sh` | Menu interativo para backup/restauração de `data/` |
| `disk-health.sh` | Monitoramento de disco com limpeza segura opcional |
| `README.md` | Documentação principal com endpoints, troubleshooting, preços |
| `DEVELOPMENT_GUIDE.md` | Guia para desenvolvedores trabalhar com o projeto |
| `SUMMARY.md` | Resumo executivo do projeto |
| `FILES_LOCATION.md` | Mapa de localização de arquivos |

### Módulo Integration

| Arquivo | Responsabilidade |
|---|---|
| `integration/docker-compose.yml` | Compose avançado com redes `newapi-internal` + `atius-shared` |
| `integration/.env` | Variáveis unificadas (DB + NewAPI + DeepSeek + Middleware) |
| `integration/scripts/*.py` | Scripts Python para sync de channels e normalização |
| `integration/scripts/*.sh` | Scripts Bash para testes, updates e verificações |
| `integration/searxng/settings.yml` | Configuração do motor de busca SearXNG |
| `integration/Models_gsd.json` | Catálogo de modelos com preços e especificações |
| `integration/GUIA_MUDANCA_CHAVES_API.md` | Documentação para rotação de chaves |

### Módulo Data

| Diretório | Responsabilidade |
|---|---|
| `data/logs/` | Logs da aplicação NewAPI |
| `data/postgres_data/` | Dados persistentes do PostgreSQL (bind mount) |
| `data/one-api.db` | Possível legado SQLite (pode ser resíduo de migração) |

## Convenções de Nomenclatura

### Containers

| Container | Convenção |
|---|---|
| `new-api` | Nome direto do serviço |
| `db-newapi` | Prefixo `db-` + nome do serviço |

### Scripts

| Padrão | Exemplo |
|---|---|
| `ação-recurso.sh` | `start.sh`, `reload-newapi.sh`, `disk-health.sh` |
| `ação-recurso.py` | `sync_deepseak_channels.py`, `normalize_models_real_only.py` |
| `verbo_recurso.sh` | `backup-restore.sh`, `test_all_models.sh` |

### Variáveis de Ambiente

| Prefixo | Uso |
|---|---|
| `POSTGRES_*` | Configurações do banco de dados |
| `SQL_*` | String de conexão SQL |
| `DEEPSEAK_*` | Configurações da API DeepSeek |
| `NEWAPI_*` | Configurações específicas do NewAPI |
| `LITELLM_*` | Compatibilidade com ecossistema LiteLLM |
| `TZ`, `LANG`, `LC_*` | Configurações de locale |

## Redes Docker

| Rede | Escopo | Descrição |
|---|---|---|
| `newapi-internal` | Internal | Comunicação entre `new-api` e `db-newapi` |
| `atius-shared` | External | Compartilhada com outros serviços do ecossistema Atius |

## Dependências entre Componentes

```
start.sh / management.sh
  └── docker-compose.yml
       ├── new-api (depends_on: db-newapi)
       │    └── PostgreSQL healthcheck obrigatório
       └── db-newapi

integration/docker-compose.yml
  └── Mesmos serviços + redes avançadas
       └── atius-shared (external)

Scripts de integração (.py)
  └── Dependem de:
       ├── integration/.env (API keys)
       └── NewAPI rodando (API admin acessível)
```

## Arquivos de Configuração por Contexto

| Contexto | Arquivo Primário |
|---|---|
| Deploy standalone | `docker-compose.yml` + `.env` (raiz) |
| Deploy com integração | `integration/docker-compose.yml` + `integration/.env` |
| Operacionalização | Scripts na raiz (`start.sh`, `management.sh`, etc.) |
| Manutenção de channels | `integration/scripts/sync_*.py` |
| Monitoramento | `disk-health.sh`, `integration/scripts/verify_stack.sh` |
