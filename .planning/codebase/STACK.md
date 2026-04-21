# STACK.md - Technology Stack

## Visão Geral

Este projeto é uma instância de **NewAPI** — um gateway LLM (Large Language Model) e sistema de gerenciamento de ativos de IA, baseado no OneAPI original por JustSong, mantido por QuantumNous.

## Runtime & Infraestrutura

| Tecnologia | Versão | Uso |
|---|---|---|
| Docker | N/A | Containerização completa |
| Docker Compose | Plugin (`docker compose`) + standalone (`docker-compose`) | Orquestração de serviços |
| PostgreSQL | 15-alpine | Banco de dados principal |
| Linux | Host Ubuntu | Sistema operacional do servidor |

## Aplicação Principal

| Componente | Imagem Docker | Descrição |
|---|---|---|
| NewAPI | `calciumion/new-api:latest` | Gateway LLM com UI administrativa |
| PostgreSQL | `postgres:15-alpine` | Banco de dados relacional |

## Linguagens de Script

| Linguagem | Uso |
|---|---|
| **Bash** | Scripts de operacionalização (start, management, backup, disk-health, reload, recreate) |
| **Python** | Scripts de integração (sync de channels, normalização de modelos, testes) |

## provedores de LLM Integrados

| Provider | Modelos | Endpoint Base |
|---|---|---|
| **DeepSeek** | DeepSeek-R1, DeepSeek-V3.2-Exp | `https://api.deepseek.com` |
| **MiniMax** | MiniMax-M2.7, MiniMax-M2.5 | `https://api.minimax.io` |
| **Qwen (Alibaba)** | Qwen3-Max, Qwen3-VL-Plus, Qwen3-Coder-Plus | API externa |
| **Kimi (Moonshot)** | Kimi-K2-Instruct-0905 | API externa |

## Modelo de Catálogo (preços)

| Modelo | Input ($/1M) | Output ($/1M) | Contexto | Max Output |
|---|---|---|---|---|
| Qwen3-VL-Plus | $0.137 | $0.409 | 256K | 32K |
| Qwen3-Coder-Plus | $0.65 | $3.25 | 1M | 64K |
| Kimi-K2-Instruct-0905 | $0.60 | $2.50 | 256K | 64K |
| Qwen3-Max | $0.78 | $3.90 | 256K | 32K |
| DeepSeek-R1 | $0.55 | $2.19 | 128K | 32K |
| DeepSeek-V3.2-Exp | $0.51 | $2.04 | 128K | 64K |
| MiniMax-M2.7 | $0.30 | $1.20 | 245K | 50K |
| MiniMax-M2.5 | $0.30 | $1.20 | 245K | 50K |

## Middleware & Integrações

| Componente | Tipo | Descrição |
|---|---|---|
| **model-detailed** | Middleware Python | Enriquecimento de metadata (DeepSeek + MiniMax) no `GET /v1/models` |
| **Search-Engine** | Middleware Python | Proxy de busca com rate limiting |
| **SearXNG** | Motor de busca | Instância self-hosted para search |
| **Whisper.cpp** | STT | Transcrição de áudio local |
| **Bruno CLI** | Testes | Testes automatizados para endpoints MiniMax |

## Endpoints de API

| Endpoint | Formato | Auth |
|---|---|---|
| `/v1/models` | OpenAI-compatible | Bearer token |
| `/v1/chat/completions` | OpenAI-compatible | Bearer token |
| `/v1/model/info` | LiteLLM-compatible | Bearer token |
| `/api/status` | Health | Público |
| `/health` | Health | Público |
| `/api/user/login` | UI Admin | Sessão |
| `/api/channel/` | UI Admin | Sessão admin |
| `/api/option/` | UI Admin | Sessão admin |

## Networking

| Serviço | Porta Host | Porta Container | Protocolo |
|---|---|---|---|
| NewAPI | — | 3000 | HTTP |
| model-detailed | 3300 | 3001 | HTTP |
| PostgreSQL | 8746 | 5432 | TCP |

## Domínio Público

- **URL**: `https://router.atius.com.br`
- **Local**: `http://localhost:3300`

## Observações

- A aplicação NewAPI é **closed-source** neste deployment (imagem Docker pré-construída)
- Não há código-fonte local da aplicação — apenas scripts de operação e integração
- O banco de dados PostgreSQL é usado para persistência de channels, tokens, options e configurações
- O projeto usa redes Docker: `newapi-internal` (interna) e `atius-shared` (externa, para integração com outros serviços)
