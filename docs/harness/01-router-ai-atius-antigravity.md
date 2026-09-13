# Router AI Atius + Antigravity (agy) Runtime Channel

## 1. Sumário Executivo & Visão Estratégica

O objetivo deste projeto é integrar o runtime oficial do **Antigravity (`agy`)** como um canal de backend nativo de primeira classe dentro do **Router AI Atius** (base Go / `new-api`), viabilizando:
1. Orquestração multi-contas totalmente isolada e legítima, sem risco de suspensão de contas Google;
2. Exposição de um endpoint compatível com o protocolo OpenAI (`/v1/chat/completions`) com streaming Server-Sent Events (SSE);
3. Gerenciamento inteligente de instâncias via Podman Rootless com contenção estrita de recursos (CPU/RAM cgroup v2);
4. Otimização semântica e compressão de contexto via middleware **Headroom**, desacoplando o histórico canônico do armazenamento local transitório do runtime.

---

## 2. Princípios Inegociáveis & Análise de Riscos de Conta

### 2.1 Por que contas foram suspensas no passado?
A análise de incidentes anteriores envolvendo o ecossistema Google/Gemini revelou causas determinantes:
- **Scraping de tokens internos**: ferramentas não-oficiais extraíam tokens OAuth ou cookies da sessão web e disparavam requisições HTTP diretas contra endpoints internos/privados (`https://*.corp.google.com`, gRPC-Web privados de interfaces internas).
- **Incoerência de assinaturas TLS/HTTP/Fingerprinting**: tráfego de API vindo de cURL ou bibliotecas HTTP padrão simulando ser um browser ou cliente de desktop sem JA3/JA4/HTTP2 fingerprint legítimo.
- **Requisições concorrentes anômalas**: dezenas de prompts simultâneos despachados pela mesma credencial em frações de segundo sem comportamento de leitura humana.

### 2.2 Princípios de Segurança e Legitimação
- **Fronteira Única de Autenticação**: o binário oficial `agy` executa a autenticação e o handshake de segurança. Nenhuma chave de sessão, token OAuth ou credencial é extraída ou transmitida pelo Atius fora do ambiente do próprio runtime.
- **Zero Endpoints Privados**: o Atius não emula chamadas de rede do Google; ele interage exclusivamente com a interface local (CLI/stdio/socket) fornecida pelo binário oficial.
- **Isolamento Hermético (1 Conta = 1 Container = 1 Storage)**: cada conta reside em um container rootless isolado, com seu próprio diretório de trabalho, variáveis de ambiente, cookies e cache local. Não há volume compartilhado entre identidades.
- **Uso Legítimo de Contas Distintas**: múltiplos runtimes servem para segregação de ambientes, limites administrativos e continuidade operacional, nunca para contornar quotas abusivamente.
- **Rate-Limiting e Cooldowns Orgânicos**: o pool aplica espaçamento temporal adaptativo entre requisições, simulando cadência operacional segura.

---

## 3. Arquitetura do Sistema

```text
               ┌────────────────────────────────────────┐
               │         Cliente OpenAI / SDK           │
               │   (POST /v1/chat/completions stream)   │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │            Router AI Atius             │
               │         (Go Backend / Gin Engine)      │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │          Middleware Headroom           │
               │  - Poda e compressão de mensagens      │
               │  - Desduplicação de histórico          │
               │  - Cache de prefixo semântico          │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │       Channel Antigravity (Go)         │
               │  - Conversor OpenAI DTO <-> Agy        │
               │  - Transformador de Stream SSE         │
               └───────────────────┬────────────────────┘
                                   │
                                   ▼
               ┌────────────────────────────────────────┐
               │        Runtime Account Pool            │
               │  - Health checks & Circuit Breaker     │
               │  - Seleção por afinidade / carga       │
               │  - Concurrency Governor (cgroup cap)   │
               └───────┬──────────────┬──────────────┬──┘
                       │              │              │
         ┌─────────────┘              │              └─────────────┐
         ▼                            ▼                            ▼
┌───────────────────┐        ┌───────────────────┐        ┌───────────────────┐
│ Container Podman  │        │ Container Podman  │        │ Container Podman  │
│  [Conta A - Prod] │        │  [Conta B - QA]   │        │ [Conta C - Lab]   │
│                   │        │                   │        │                   │
│ - Base Image read │        │ - Base Image read │        │ - Base Image read │
│ - Volume: vol_a   │        │ - Volume: vol_b   │        │ - Volume: vol_c   │
│ - agy oficial     │        │ - agy oficial     │        │ - agy oficial     │
│ - Unix Socket IPC │        │ - Unix Socket IPC │        │ - Unix Socket IPC │
└───────────────────┘        └───────────────────┘        └───────────────────┘
```

---

## 4. Modelagem e Estruturas de Dados (Go Backend)

### 4.1 Abstração de Runtime no Atius
Ao contrário de canais convencionais baseados em HTTP REST (`channel.BaseURL` + `channel.Key`), o canal Antigravity requer uma abstração de orquestração de processos locais ou IPC.

```go
package antigravity

import (
	"context"
	"io"
	"time"
)

type AccountStatus string

const (
	StatusReady     AccountStatus = "ready"
	StatusBusy      AccountStatus = "busy"
	StatusDegraded  AccountStatus = "degraded"
	StatusCooldown  AccountStatus = "cooldown"
	StatusOffline   AccountStatus = "offline"
)

type RuntimeAccount struct {
	ID             string            `json:"id"`
	AccountLabel   string            `json:"account_label"`
	ContainerName  string            `json:"container_name"`
	SocketPath     string            `json:"socket_path"`
	Status         AccountStatus     `json:"status"`
	CurrentLoad    int               `json:"current_load"`
	MaxConcurrency int               `json:"max_concurrency"`
	ConsecutiveErr int               `json:"consecutive_errors"`
	CooldownUntil  time.Time         `json:"cooldown_until"`
	LastActiveAt   time.Time         `json:"last_active_at"`
	Metadata       map[string]string `json:"metadata"`
}

type RuntimePool interface {
	Acquire(ctx context.Context, req *RuntimeRequest) (*RuntimeAccount, func(), error)
	HealthCheck(ctx context.Context, accountID string) error
	RecordMetric(accountID string, success bool, latency time.Duration)
	Snapshot() []RuntimeAccount
}
```

### 4.2 Conversão de DTOs e Protocolo
O relay traduz o payload OpenAI (`dto.GeneralOpenAIRequest`) para o contrato de execução do `agy`:

```go
type AgyExecutionRequest struct {
	Prompt          string            `json:"prompt"`
	SystemPrompt    string            `json:"system_prompt,omitempty"`
	ConversationID  string            `json:"conversation_id,omitempty"`
	Model           string            `json:"model"`
	Stream          bool              `json:"stream"`
	WorkingDir      string            `json:"working_dir"`
	Environment     map[string]string `json:"environment,omitempty"`
}

type AgyEventChunk struct {
	Type      string `json:"type"`       // "text_delta", "tool_call", "done", "error"
	Content   string `json:"content"`
	Tokens    int    `json:"tokens"`
	Timestamp int64  `json:"timestamp"`
}
```

O stream de saída é serializado no formato padrão:
```text
data: {"id":"chatcmpl-agy-123","object":"chat.completion.chunk","created":1741830000,"model":"gemini-3.8-pro","choices":[{"index":0,"delta":{"content":"exemplo"},"finish_reason":null}]}
```

---

## 5. Infraestrutura Podman Rootless & Isolamento Operacional

### 5.1 Especificação da Imagem Base
A imagem base contém apenas o runtime mínimo necessário para executar o `agy` em modo headless:
- Base: `debian:bookworm-slim` ou `ubuntu:24.04-minimal`
- Binário oficial `agy` em `/usr/local/bin/agy`
- Dependências de sistema: `ca-certificates`, `curl`, `jq`, glibc atualizada
- Usuário não-root: `uid=10001(agy)` `gid=10001(agy)`
- Root filesystem montado como `read-only` (`--read-only`)

### 5.2 Volumes e Persistência por Conta
Cada conta possui volume isolado montado exclusivamente em `/home/agy`:
```bash
podman volume create atius-agy-vol-account-a
podman volume create atius-agy-vol-account-b
```
Diretórios internos do volume:
- `/home/agy/.config/antigravity`: dados de autenticação e sessão oficial
- `/home/agy/workspace`: diretório scratch temporário (limpo periodicamente)
- `/home/agy/ipc`: socket Unix para comunicação com o host

### 5.3 Contenção Estrita de Recursos (CPU Guardrail)
Seguindo a política máxima do host (`RG_PROFILE_BUILDS_CPU_TOTAL_PCT=20`):
- Em servidor com 4 vCPUs, a quota de CPU por container não ultrapassa `0.5 CPU` (`500m`), e o agregado do pool respeita o teto de 20% total do host.
- Comando canônico de subida de container:
```bash
podman run -d \
  --name atius-agy-runtime-a \
  --restart unless-stopped \
  --read-only \
  --cpus 0.5 \
  --cpuset-cpus 0 \
  --memory 1024m \
  --memory-reservation 512m \
  --pids-limit 128 \
  --security-opt no-new-privileges \
  --user 10001:10001 \
  -v atius-agy-vol-account-a:/home/agy:U,rw \
  -v /run/atius/agy-sockets:/ipc:rw \
  localhost/antigravity-runtime:latest \
  --headless-daemon --socket /ipc/account-a.sock
```

---

## 6. O Papel do Middleware Headroom

O runtime `agy` não deve ser tratado como um banco de dados de conversas.
A dependência do SQLite interno do `agy` gera fragilidades severas:
1. Formatos de banco internos podem mudar entre versões do CLI;
2. Se uma conta atingir quota e for necessário fallback para a Conta B, o histórico ficaria preso no SQLite da Conta A;
3. O histórico cumulativo sem otimização satura rapidamente a janela de contexto e aumenta latência/custo.

### Solução: Histórico Canônico no Atius + Compressão via Headroom
1. **Histórico Canônico**: O Atius mantém a árvore canônica de mensagens no PostgreSQL.
2. **Compressão Semântica**: Antes de enviar para o pool, o **Headroom**:
   - Poda tool calls transitórias antigas que já tiveram seus resultados consolidados;
   - Remove system prompts redundantes em turnos subsequentes;
   - Realiza sumarização incremental de blocos de contexto antigos;
   - Mantém um prefixo limpo e padronizado, maximizando o reuso de cache do Gemini.
3. **Migração Transparente entre Contas**: Se a Conta A sinalizar `429 Too Many Requests`, o Atius seleciona a Conta B no pool, e o Headroom despacha o contexto otimizado de forma transparente para a nova instância, mantendo a experiência fluida sem perda de continuidade.

---

## 7. Matriz de Compatibilidade e Viabilidade Técnica

| Dimensão | Estado Atual | Viabilidade Técnica | Ações Necessárias |
| :--- | :--- | :--- | :--- |
| **Execução Headless** | Suportada via CLI flags (`--dangerously-skip-permissions`, modos de execução não-interativos) | **Alta** (Comprovada em automações locais) | Definir protocolo de transporte preferencial (Unix socket vs stdio pipe bidirecional). |
| **Streaming de Resposta** | Suportado (emissão de chunks no stdout) | **Alta** | Criar parser de eventos do `agy` para `chat.completion.chunk` OpenAI. |
| **Tool Calling / Function Calling** | Nativo no `agy` | **Média/Alta** | Mapear chamadas de ferramentas do Atius para o executor do `agy` ou delegar totalmente ao `agy`. |
| **Isolamento de Contas** | Múltiplos profiles no CLI vs Containers Podman | **Alta via Podman** | Podman rootless elimina qualquer colisão de cookies/config no filesystem. |
| **Desempenho & Latência** | Cold start ~1-2s; warm socket < 50ms | **Alta** | Manter processos em estado warm no pool para evitar custo de boot por prompt. |
| **Consumo de Memória** | ~80MB a 250MB por container ativo | **Alta** | Teto de 3 containers ativos simultâneos cabe com folga no host de 16GB RAM. |
| **Compatibilidade de Arquitetura** | Linux ARM64 e AMD64 | **Alta** | Binários do Google Cloud Code / Antigravity possuem compilações ARM64 nativas para Linux. |

---

## 8. Plano de Implementação em Fases (Roadmap)

### Fase 1: Análise e Validação de Transporte do `agy`
- Inspecionar a interface não-interativa do `agy` (stdio vs socket IPC).
- Medir latência de resposta, consumo de memória e formato dos eventos emitidos.
- Critério de saída: Script Go de prova de conceito enviando prompt e recebendo stream SSE do binário local.

### Fase 2: Construção da Imagem Base e Sandbox Podman
- Criação do `Dockerfile.antigravity-runtime` minimalista e sem privilégios.
- Validação do volume isolado para profile de autenticação.
- Script de inicialização idempotente com limits de cgroup v2.
- Critério de saída: Container rodando rootless com permissões limitadas e comunicação via Unix Domain Socket no `/run/atius/agy-sockets/`.

### Fase 3: Desenvolvimento do Adaptor no Router AI Atius
- Adição do `ChannelTypeAntigravity` no enum de canais (`constant/channel_type.go`).
- Criação do pacote `relay/channel/antigravity/` com `Adaptor`, `Converter` e `Streamer`.
- Suporte a modelos virtuais no catálogo: `gemini-3.8-pro-agy`, `gemini-3.8-flash-agy`.
- Critério de saída: Requisição `curl http://localhost:3000/v1/chat/completions` respondendo com sucesso via runtime local.

### Fase 4: Runtime Account Pool e Concurrency Governor
- Implementação de `RuntimePool` gerenciando múltiplas instâncias de contas.
- Health checking com detecção precoce de `quota_exhausted` ou `session_invalid`.
- Circuito de fallback automático para a próxima conta disponível.
- Critério de saída: Testes unitários comprovando rotação de conta sem falha visível para o cliente.

### Fase 5: Integração do Middleware Headroom
- Interceptação de mensagens antes do acquire do pool.
- Poda de contexto e sanitização de dados sensíveis.
- Desduplicação de histórico conversacional.
- Critério de saída: Redução de pelo menos 35% no volume de tokens processados por turno longo.

### Fase 6: Homologação, Observabilidade e Hardening
- Exportação de métricas Prometheus (tempo de resposta, taxa de erro por conta, uptime do pool).
- Testes de carga sob limite de CPU do host (respeitando o cap de 20%).
- Validação da blindagem de conta (nenhuma suspensão ou anomalia detectada em 14 dias de soak test contínuo).
- Critério de saída: Produção homologada no cluster Atius.
