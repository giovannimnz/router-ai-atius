# Phase 33: Reranker reliability, observability, and readiness - Research

**Researched:** 2026-08-14
**Domain:** Go relay outcome telemetry, channel readiness, bounded TEI inference, weighted React dashboard, and governed release persistence
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- Preservar `request_success_rate`, mas nunca apresentá-la sozinha como saúde do TEI.
- Adicionar taxonomia explícita para client/local validation, governor, routing, transport/timeout, upstream e success.
- Medir falhas sem canal antes da seleção do relay, com uma única amostra terminal por request.
- Calcular indicadores gerais a partir dos contadores subjacentes, de forma ponderada.
- Exigir sync/readiness do cache/ability antes do smoke de um canal recém-criado ou reativado.
- Manter o limite de 20 documentos; orientar sub-batching no cliente preservando índices.
- Propagar context/deadline ao request TEI e limitar retry a uma única repetição apenas para transporte/5xx explicitamente seguros.
- Não promover candidato sem o reranker governado completo, canal 10 e os dois aliases do governor.

### the agent's Discretion

O `33-CONTEXT.md` não contém uma seção de discricionariedade do agente. [VERIFIED: 33-CONTEXT.md]

### Deferred Ideas (OUT OF SCOPE)

- Remover ou enfraquecer o embedding/rerank governor.
- Fazer sub-batching transparente no Router sem design específico de recomposição, ranking e billing.
- Fazer cutover público k3s das Phases 29/30.
- Alterar identidade protegida do upstream.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PHASE-33-METRICS-OUTCOME-TAXONOMY | Preserve the existing request success rate as a request-outcome metric, not as the sole backend-health signal; classify at least client/local validation, governor rejection, routing availability, transport/timeout, upstream HTTP failure, invalid upstream response, and success; persist enough aggregate dimensions without sensitive content; keep SQLite, MySQL, and PostgreSQL compatibility. | Schema aditivo, enum fixo, invariant de soma, agregação GORM e versionamento parcial. [VERIFIED: REQUIREMENTS.md and codebase inspection] |
| PHASE-33-ROUTING-AVAILABILITY | Requests that fail because no eligible channel exists must contribute before the relay loop; one terminal request produces exactly one terminal outcome sample; distinguish routing failure from TEI/provider failure. | Recorder terminal request-scoped iniciado no distributor e finalizado em routing, relay error ou settlement. [VERIFIED: REQUIREMENTS.md and codebase inspection] |
| PHASE-33-WEIGHTED-DASHBOARD | Do not calculate overall health as a simple average; weight request success and service availability from underlying counts; expose client/governor/routing/upstream classes; preserve compatibility or use an explicit additive/versioned contract. | Contrato `overall` aditivo, helpers puros de agregação e estados zero/partial do UI-SPEC aprovado. [VERIFIED: REQUIREMENTS.md and 33-UI-SPEC.md] |
| PHASE-33-CHANNEL-READINESS | Provisioning or re-enabling must force/await cache sync and poll authenticated readiness; prove channel 10, reranker alias and ability are consistent; fail closed, bounded, observable and rollback-safe. | Endpoint admin bounded que força `InitChannelCache` e sonda a seleção normal do distributor. [VERIFIED: REQUIREMENTS.md and codebase inspection] |
| PHASE-33-TEI-TIMEOUT-RETRY | Reranker outbound requests inherit cancellation/deadline and a bounded inference timeout even with `RELAY_TIMEOUT=0`; retry at most once for explicitly safe transport/5xx only; do not bypass governor accounting or duplicate settlement/metrics. | Context por tentativa, `http.NewRequestWithContext`, policy rerank-specific e accounting por tentativa/terminal. [VERIFIED: REQUIREMENTS.md] [CITED: https://pkg.go.dev/net/http@go1.25.3] |
| PHASE-33-RERANK-BATCH-CONTRACT | Preserve the 20-document cap; return deterministic client-facing 4xx above it and document client sub-batches with stable index recomposition; transparent Router sub-batching remains out of scope without a separate design. | Manter validação pré-governor e documentar chunks com recomposição de índice, sem sub-batching Router. [VERIFIED: REQUIREMENTS.md and codebase inspection] |
| PHASE-33-FORK-SYNC-PERSISTENCE | Never promote without governed reranker source, channel 10 and exact governor aliases; preserve baseline `deb39c92d`; protect the four specified additional files; fail preflight before cutover on missing markers/env/tests/candidate contents. | Protected paths completos, preflight estático, label/digest, candidate smoke e rollback por digest no omni-srv-admin. [VERIFIED: REQUIREMENTS.md and cross-repo inspection] |
| PHASE-33-TEST-DOC-LIVE-EVIDENCE | Add deterministic Go/frontend tests; run heavy checks through the CPU wrapper; validate local and authenticated public paths with rollback and no sensitive content; update Router/omni/GBrain/Obsidian/Graphify evidence. | Mapa Nyquist, comandos governados, evidência sanitizada e readback Router/omni/GBrain/Obsidian/Graphify. [VERIFIED: REQUIREMENTS.md and repository policy] |
</phase_requirements>

## Summary

O problema não é apenas uma fórmula incorreta no frontend. O caminho atual representa qualquer término como um booleano `Success`, registra o erro final somente dentro de `controller/relay.go`, e seleciona canal no middleware antes de o controller executar. Por isso um `503 No available channel` encerra a request no distributor sem passar pelo ponto de registro atual, enquanto um `400` local de mais de 20 documentos entra como falha genérica. O dashboard então calcula médias simples de percentuais por modelo e perde tanto o peso por volume quanto a origem do outcome. [VERIFIED: codebase inspection]

A solução deve introduzir um outcome terminal fechado e idempotente por request, persistir contadores aditivos por bucket, fornecer um agregado `overall` derivado das somas brutas e manter os campos legados. O mesmo desenho deve carregar `latency_sample_count`, pois `total_latency_ms` sem denominador explícito não permite distinguir ausência de latência de latência zero. Linhas históricas anteriores ao novo schema devem ser apresentadas como parciais, nunca inferidas como zero nas novas classes. [VERIFIED: codebase inspection] [VERIFIED: 33-UI-SPEC.md]

Readiness e release precisam ser funcionais, não temporais: depois de criar/reativar o canal, force o rebuild atômico do cache, consulte a mesma seleção usada pelo distributor até que canal, alias e ability concordem, e só então execute smoke. Antes do cutover, valide o source candidate, focused tests, OCI label/digest e um rerank autenticado isolado; preserve o digest anterior e faça rollback automático se o smoke falhar. [VERIFIED: codebase and cross-repo inspection]

**Primary recommendation:** implementar primeiro o contrato de outcome terminal + schema/API aditivos, depois readiness/timeout/retry e UI ponderada, fechando com guards de candidato e rollback no `omni-srv-admin`. [VERIFIED: dependency analysis]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Classificação terminal exatamente uma vez | API / Backend | Database / Storage | A request atravessa middleware, relay, retries e settlement; o backend é o único tier que observa o outcome final e pode impor idempotência. [VERIFIED: codebase inspection] |
| Falha no-channel | API / Backend middleware | Database / Storage | A indisponibilidade é conhecida em `middleware/distributor.go`, antes do controller, e precisa alcançar a mesma persistência agregada. [VERIFIED: codebase inspection] |
| Buckets e contadores de performance | Database / Storage | API / Backend | GORM persiste buckets por model/group/time; o backend agrega hot bucket, Redis e DB. [VERIFIED: codebase inspection] |
| Overall ponderado e contrato compatível | API / Backend | Browser / Client | O backend deve fornecer somas brutas/overall; o cliente pode recalcular apenas quando todas as linhas possuem os contadores. [VERIFIED: 33-UI-SPEC.md] |
| Dashboard semântico | Browser / Client | API / Backend | React renderiza labels, partial/empty/error e fórmulas aprovadas a partir do contrato aditivo. [VERIFIED: 33-UI-SPEC.md] |
| Channel readiness | API / Backend | Database / Storage | Readiness deve reconciliar row, ability, route cache e a seleção real protegida por auth admin. [VERIFIED: codebase inspection] |
| Deadline e retry TEI | API / Backend relay | External TEI service | Cada outbound attempt herda cancelamento, recebe timeout próprio e passa pelo governor. [CITED: https://pkg.go.dev/net/http@go1.25.3] |
| Cap e sub-batching | Browser / API client | API / Backend | O Router preserva o cap e o índice de cada request; o chamador divide a lista e recompõe índices estáveis. [VERIFIED: 33-CONTEXT.md] |
| Candidate gate e rollback | Release / Operations | API / Backend | O omni-srv-admin deve provar conteúdo e comportamento da imagem antes de promover/reiniciar, mantendo o digest anterior. [VERIFIED: cross-repo inspection] |

## Project Constraints (from AGENTS.md)

- Qualquer build, typecheck, suite pesada, container build ou indexação deve executar pelo `scripts/podman-admin.sh` sob o limite de 20% da CPU total; neste host de 4 vCPUs isso significa quota de 0,8 CPU. [VERIFIED: AGENTS.md; `verify-profile` readback]
- O backend segue Router → Controller → Service → Model; a implementação deve preservar essa separação. [VERIFIED: AGENTS.md]
- Marshal/unmarshal de negócio deve usar `common/json.go`; chamadas diretas a `encoding/json` são proibidas. [VERIFIED: AGENTS.md]
- Toda alteração de persistência precisa funcionar em SQLite, MySQL >= 5.7.8 e PostgreSQL >= 9.6, preferindo GORM e evitando SQL específico sem fallback. [VERIFIED: AGENTS.md]
- O frontend usa React 19, TypeScript, Rsbuild, Base UI, Tailwind e Bun; não adicionar package, chart library, route ou tema para esta fase. [VERIFIED: AGENTS.md] [VERIFIED: 33-UI-SPEC.md]
- Campos escalares opcionais enviados a upstream devem usar ponteiro com `omitempty` para preservar zero explícito. [VERIFIED: AGENTS.md]
- O cap TEI de quatro itens para embeddings e o cap de vinte documentos para rerank permanecem; esta fase não cria sub-batching transparente no Router. [VERIFIED: AGENTS.md] [VERIFIED: 33-CONTEXT.md]
- `EMBEDDING_GOVERNOR_MAX_CONCURRENCY=0` e `EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=0` representam ausência de ceiling estático; scale continua governado por feedback, queue, cooldown, health, latency e capacity telemetry. [VERIFIED: AGENTS.md]
- Testes Go novos/substancialmente reescritos devem usar `testify/require` para setup/fatal e `testify/assert` para verificações não fatais, cobrindo contrato observável e não coverage gaming. [VERIFIED: AGENTS.md]
- Não modificar/remover identidade protegida `new-api` ou `QuantumNous`. [VERIFIED: AGENTS.md]
- Não reintroduzir sidecar/Python para rerank/embedding; preservar o caminho Go-native, canal 10, Advanced Custom TEI e governor. [VERIFIED: AGENTS.md]
- Nenhuma telemetria/evidência pode registrar prompt, documento, token, credencial, segredo ou corpo completo de erro. [VERIFIED: AGENTS.md] [VERIFIED: 33-CONTEXT.md]
- Graphify é obrigatório antes/depois de mudanças de planning/código; navegador, se necessário para evidência, deve permanecer headless. Para o closeout terminal desta fase, `graphify.auto_update=false`: indexação assíncrona/default-branch não aplica o wrapper CPU20 nem prova o HEAD final. [VERIFIED: AGENTS.md; `.planning/config.json`; GSD/Graphify lifecycle inspection]

## Standard Stack

### Core

| Library / Facility | Version | Purpose | Why Standard |
|--------------------|---------|---------|--------------|
| Go standard `context` + `net/http` | Go module declares 1.25.1 | Propagar cancelamento/deadline e limitar cada attempt outbound. | `Request.Context` e `NewRequestWithContext` controlam a vida inteira da request/response outbound. [CITED: https://pkg.go.dev/net/http@go1.25.3] |
| Gin | 1.9.1 | Middleware, auth, handlers e request-scoped recorder. | Já é o HTTP framework do Router; não criar um segundo pipeline. [VERIFIED: go.mod and codebase inspection] |
| GORM | 1.25.2 | Schema aditivo, upsert e consultas SUM cross-DB. | Já persiste `PerfMetric` e suporta AutoMigrate aditivo/Migrator. [VERIFIED: go.mod and codebase inspection] [CITED: https://gorm.io/docs/migration.html] |
| Existing `pkg/perf_metrics` | repository-local | Hot buckets, Redis mirror, flush e summary. | Estender todas as representações juntas evita telemetria paralela inconsistente. [VERIFIED: codebase inspection] |
| Existing `service/embeddinggovernor` | repository-local | Admission/accounting por attempt TEI. | O reranker já adquire e finaliza lease por tentativa; retry não deve contornar esse caminho. [VERIFIED: codebase inspection] |
| React + TanStack Query | React 19 catalog; Query 5.101.2 | Dashboard, cache, refetch e error state. | A query atual já usa `retry: false`; refetch pode manter dados previamente carregados. [VERIFIED: package.json and codebase inspection] [CITED: https://tanstack.com/query/v5/docs/framework/react/reference/useQuery] |
| Base UI / existing shadcn wrappers | Base UI 1.6.0 | Alert, Button, Badge, Tooltip, Skeleton e Empty. | Contrato aprovado proíbe novo kit visual. [VERIFIED: package.json] [VERIFIED: 33-UI-SPEC.md] |

### Supporting

| Library / Facility | Version | Purpose | When to Use |
|--------------------|---------|---------|-------------|
| `github.com/stretchr/testify` | 1.11.1 | Assertions determinísticas Go. | Todos os testes backend novos/substancialmente reescritos. [VERIFIED: go.mod and AGENTS.md] |
| Bun test | 1.3.14 installed | Testar helpers TypeScript puros com `node:test`/Bun sem package novo. | Agregação ponderada, zero denominator e partial series. [VERIFIED: environment probe and existing test inspection] |
| `scripts/podman-admin.sh` | repository-local | Cgroup CPU/memory containment. | Todo heavy check, typecheck, build e image build. [VERIFIED: AGENTS.md; `verify-profile` readback] |
| Existing channel cache + ability model | repository-local | Rebuild atômico e seleção normal. | Readiness de canal novo/reativado. [VERIFIED: codebase inspection] |
| OCI image labels/digests | Podman 4.9.3 installed | Vincular source candidate ao artefato e preservar rollback. | Release preflight, candidate verification e cutover. [VERIFIED: environment probe and cross-repo inspection] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Contadores fixos por outcome | JSON blob de outcomes | Blob complica SUM cross-DB, migrations e queries; colunas inteiras fixas refletem uma taxonomia locked e são mais auditáveis. [VERIFIED: requirements analysis] |
| Endpoint `overall` + contadores por modelo | Média de percentuais no browser | Média simples viola peso por volume e não consegue distinguir missing de zero. [VERIFIED: 33-UI-SPEC.md] |
| Readiness pela seleção normal | Sleep fixo ou somente consulta DB | Sleep não prova refresh; row ativa não prova ability/route cache nem seleção. [VERIFIED: codebase inspection] |
| Context por attempt | `http.Client.Timeout` global | O timeout global pode estar desabilitado por `RELAY_TIMEOUT=0` e não fornece granularidade do canal rerank. [VERIFIED: codebase inspection] |
| Candidate smoke antes do cutover | `/api/status` depois do restart | Health genérico não prova que o reranker foi compilado/configurado nem evita downtime de candidato inválido. [VERIFIED: cross-repo inspection] |

**Installation:** nenhuma instalação externa é necessária. Use somente dependências já presentes. [VERIFIED: dependency audit]

**Version verification:** as versões acima foram lidas de `go.mod`, `web/default/package.json` e do ambiente instalado; não há upgrade/install de registry nesta fase, portanto publish date e “latest” não são critérios de planejamento. [VERIFIED: manifest and environment inspection]

## Package Legitimacy Audit

Não aplicável: esta fase não instala packages externos e deve reutilizar exclusivamente stdlib e dependências já travadas no repo. [VERIFIED: dependency audit]

**Packages removed due to [SLOP] verdict:** none. [VERIFIED: dependency audit]

**Packages flagged as suspicious [SUS]:** none. [VERIFIED: dependency audit]

## Architecture Patterns

### System Architecture Diagram

```text
Authenticated client request
          |
          v
Token/auth middleware ---- auth/validation reject ---> CLIENT_VALIDATION
          |
          v
Distributor: model + group + path
    |                      \
    | selected              `-- no eligible channel ---> ROUTING_UNAVAILABLE
    v
RerankHelper: validate <=20 documents
    |                      \
    | valid                 `-- invalid payload -------> CLIENT_VALIDATION
    v
Governor Acquire
    |                      \
    | lease                 `-- rejected --------------> GOVERNOR_REJECTION
    v
Attempt 1: child context + bounded timeout -> TEI
    | success / terminal error             \
    |                                      `-- safe transport/5xx? -- yes --> Attempt 2 once
    |                                                                      |
    +----------------------------- final semantic classification <---------+
                                      |
                                      v
                         Request-scoped terminal recorder
                         (CompareAndSwap / finalize once)
                                      |
                                      v
            hot bucket + Redis fields + GORM bucket upsert
                                      |
                                      v
              additive API: overall + models + legacy fields
                                      |
                                      v
      React: four weighted KPIs + seven outcomes + top models

Provision/reactivate channel
          |
          v
DB channel 10 + model ability + Advanced Custom route
          |
          v
InitChannelCache synchronously
          |
          v
Bounded poll through normal distributor selection
    | ready                                  | timeout/mismatch
    v                                        v
candidate authenticated rerank smoke      FAIL CLOSED
    |
    v
promote labeled digest / cutover ---- smoke failure ---> rollback old digest
```

O recorder terminal é um objeto por request, mas sua amostra deve copiar somente metadata imutável e não sensível; não retenha `*gin.Context` em closures assíncronas porque Gin pode reutilizar o contexto após o handler. [VERIFIED: codebase inspection]

### Recommended Project Structure

```text
pkg/perf_metrics/
├── types.go                 # fixed outcome enum, Sample, counters, API shapes
├── metrics.go               # exact-once recorder integration and weighted summary
├── flush.go                 # every counter drained/flushed together
└── metrics_test.go          # invariant and weighted aggregation
model/
└── perf_metric.go           # additive columns, upsert and SUM queries
middleware/
└── distributor.go           # initialize/finalize routing-unavailable outcome
controller/
├── relay.go                 # final relay outcome and rerank retry policy
├── perf_metrics.go          # additive/versioned response
└── channel*.go              # authenticated readiness orchestration
relay/
├── rerank_handler.go        # cap, per-attempt timeout, governor accounting
└── channel/api_request.go   # NewRequestWithContext from explicit outbound context
web/default/src/features/performance-metrics/
├── types.ts                 # additive raw-count contract
└── lib/outcome-summary.ts   # pure weighted/partial calculations
omni-srv-admin/modules/fork-sync/
├── projects/atius-router/sync.yaml
├── UPSTREAM-SYNC-GUARDS.md
└── cli/release_preflight.py # source/env/marker/test/image gates
```

Cada arquivo acima mapeia para um seam já existente; novos arquivos devem ser limitados ao helper/test puro quando isso reduz duplicação entre os dois painéis React. [VERIFIED: codebase inspection]

### Pattern 1: Closed terminal outcome with exact-once finalize

**What:** substituir o booleano `Success` por um enum fechado com sete valores e uma finalização idempotente. `request_count` continua sendo `terminal_count`; `success_count` continua representando success para compatibilidade. [VERIFIED: requirements analysis]

**When to use:** em todo ponto que encerra a request: validação local, governor, no-channel, erro final após retry e sucesso após settlement. [VERIFIED: codebase inspection]

**Example:**

```go
// Source: project pattern; exact API name is implementation-defined.
type Outcome uint8

const (
    OutcomeSuccess Outcome = iota
    OutcomeClientValidation
    OutcomeGovernorRejection
    OutcomeRoutingUnavailable
    OutcomeTransportTimeout
    OutcomeUpstreamHTTPFailure
    OutcomeInvalidUpstreamResponse
)

func (r *TerminalRecorder) Finish(outcome Outcome, latency time.Duration) bool {
    if !r.finished.CompareAndSwap(false, true) {
        return false
    }
    Record(Sample{Model: r.model, Group: r.group, Outcome: outcome, LatencyMs: latency.Milliseconds()})
    return true
}
```

Teste a invariante persistida `terminal_count == sum(all seven outcome counts)` em hot bucket, Redis representation, flush e DB summary. [VERIFIED: requirements analysis]

### Pattern 2: Additive schema with explicit completeness

**What:** adicionar seis failure counters, `latency_sample_count` e um marcador de writer/schema ao bucket existente, devolver `schema_version`, `overall` e raw counters/totals por model, preservando `success_rate`, `avg_latency_ms`, `avg_tps` e `models`. Um agregado só é completo quando **cada bucket contribuinte** carrega o marcador v2 e, em cada bucket, a soma dos sete outcomes é igual a `request_count`; a soma global isolada não prova completude. [VERIFIED: codebase inspection] [VERIFIED: 33-UI-SPEC.md]

**When to use:** durante migração sem quebrar consumidores existentes nem reinterpretar buckets históricos. [CITED: https://gorm.io/docs/migration.html]

**Recommended response shape:**

```json
{
  "schema_version": 2,
  "overall": {
    "terminal_count": 100,
    "success_count": 90,
    "client_validation_count": 3,
    "governor_rejection_count": 1,
    "routing_unavailable_count": 2,
    "transport_timeout_count": 1,
    "upstream_http_failure_count": 2,
    "invalid_upstream_response_count": 1,
    "total_latency_ms": 72000,
    "latency_sample_count": 96,
    "output_tokens": 24000,
    "generation_ms": 120000,
    "complete": true
  },
  "models": []
}
```

O payload contém apenas agregados; não inclua error body, URL com credencial, token, query, document ou prompt. [VERIFIED: requirements and security analysis]

### Pattern 3: Origin-aware classification, not status-only classification

**What:** classificar pelo ponto de decisão e tipo do erro, não somente por HTTP status. Um `400` produzido pelo Router é client/local validation; uma resposta HTTP 4xx do TEI é upstream HTTP failure/semantic upstream failure conforme o contrato, mas nunca transport timeout. [VERIFIED: codebase inspection] [VERIFIED: requirements analysis]

**When to use:** ao mapear `NewAPIError`, governor error, distributor abort, client transport error e adaptor conversion error. [VERIFIED: codebase inspection]

Mantenha uma função pura `ClassifyTerminalOutcome(origin, errorType, status)` com table tests; não espalhe switches divergentes pelos handlers. [VERIFIED: architecture analysis]

### Pattern 4: Fresh bounded context per rerank attempt

**What:** derivar `context.WithTimeout(c.Request.Context(), configuredTimeout)` em cada attempt e enviar esse contexto explicitamente a `DoApiRequest`; o helper genérico cria a request com `http.NewRequestWithContext`. [CITED: https://pkg.go.dev/context] [CITED: https://pkg.go.dev/net/http@go1.25.3]

**When to use:** somente na inferência rerank nesta fase; não substitua permanentemente `c.Request` pelo child context porque o controller reutiliza RelayInfo/Gin request no retry seguinte. [VERIFIED: codebase inspection]

**Example:**

```go
// Source: https://pkg.go.dev/net/http@go1.25.3
attemptCtx, cancel := context.WithTimeout(c.Request.Context(), timeout)
defer cancel()

req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, fullRequestURL, body)
if err != nil {
    return nil, err
}
return client.Do(req)
```

O contexto outbound controla conexão, envio da request e leitura do response body; o cancel deve sempre executar. [CITED: https://pkg.go.dev/net/http@go1.25.3]

### Pattern 5: Rerank-specific retry budget

**What:** aplicar no máximo `RetryIndex == 0 -> one retry` apenas para transport errors e 5xx explicitamente allowlisted. Não retry em local 4xx, governor, routing/no-channel, upstream 4xx, invalid response ou 429. [VERIFIED: requirements analysis]

**When to use:** no controller retry loop quando `RelayModeRerank` estiver ativo. Cada attempt readquire/finalize uma governor lease; somente o outcome final atualiza perf metrics e settlement/refund continua uma vez por request. [VERIFIED: codebase inspection]

### Pattern 6: Selection-based channel readiness

**What:** endpoint admin protegido por `authz.ChannelOperate` que carrega channel/ability, valida Advanced Custom route, chama `model.InitChannelCache()`, e faz poll bounded da mesma `CacheGetRandomSatisfiedChannel` usada pelo distributor até selecionar canal 10 para o alias/path/group esperados. [VERIFIED: codebase inspection]

**When to use:** imediatamente após create, re-enable ou alteração relevante de setting/model do reranker e antes de qualquer smoke. [VERIFIED: requirements analysis]

O response deve conter somente `ready`, `channel_id`, `model`, `ability_ready`, `route_ready`, `cache_ready`, `reason_code`, `attempts` e `elapsed_ms`; timeout/cancelamento retorna não-ready e mantém rollback. [VERIFIED: security and operability analysis]

### Pattern 7: Candidate-before-promotion release gate

**What:** separar quatro gates: source preflight → focused tests → artifact identity → isolated functional candidate smoke. Somente depois atualizar `latest`/reiniciar produção. [VERIFIED: cross-repo inspection]

**When to use:** toda sincronização upstream, rebuild ou deploy do Router. [VERIFIED: AGENTS.md]

O Dockerfile final contém apenas o binário, portanto procurar `tei_rerank.go` dentro da camada final não prova source inclusion. Grave source commit/capability como OCI label, verifique label+digest, e execute readiness/rerank autenticado no candidato; salve/tague o digest atual antes da promoção. [VERIFIED: Dockerfile and deploy-script inspection]

### Anti-Patterns to Avoid

- **Finalizar no attempt:** retries passam a contar duas requests; finalize apenas depois da policy encerrar. [VERIFIED: requirements analysis]
- **Usar apenas HTTP status:** mistura client validation, governor, routing e upstream. [VERIFIED: codebase inspection]
- **Tratar missing counter como zero:** falsifica saúde histórica e elimina o estado partial aprovado. [VERIFIED: 33-UI-SPEC.md]
- **Calcular overall a partir de percentuais:** produz média simples/duplamente arredondada e ignora volume. [VERIFIED: 33-UI-SPEC.md]
- **Sleep fixo para cache:** não demonstra que channel, ability e route realmente entraram na seleção. [VERIFIED: codebase inspection]
- **Alterar `c.Request` para um contexto já expirado:** envenena o segundo attempt. [VERIFIED: codebase inspection]
- **Retry amplo pelo global `RetryTimes`:** pode retry 4xx/429/semantic error e amplificar carga. [VERIFIED: codebase inspection]
- **Promover `latest` antes do candidate smoke:** destrói a referência simples de rollback e expõe imagem incompleta. [VERIFIED: cross-repo inspection]
- **Adicionar gráfico/package para sete outcomes:** viola o contrato visual e aumenta surface sem necessidade. [VERIFIED: 33-UI-SPEC.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cancelamento HTTP | Goroutine/timer que fecha conexão manualmente | `context.WithTimeout` + `http.NewRequestWithContext` | A stdlib controla o ciclo completo e integra cancelamento do inbound. [CITED: https://pkg.go.dev/net/http@go1.25.3] |
| Migração cross-DB | DDL por database ou JSONB outcome blob | GORM AutoMigrate/Migrator + colunas inteiras aditivas | O projeto suporta três engines e já AutoMigrate `PerfMetric`. [VERIFIED: codebase inspection] [CITED: https://gorm.io/docs/migration.html] |
| Agregação overall | Média de percentuais/model TPS | SUM dos raw counters/totals, uma divisão final | Preserva peso e denominador. [VERIFIED: 33-UI-SPEC.md] |
| Readiness | Sleep/retry shell sem condição semântica | Rebuild atômico + poll da seleção normal | Prova o mesmo caminho que receberá tráfego. [VERIFIED: codebase inspection] |
| UI state/cache | Fetch loop e retry custom | TanStack Query `retry:false`, `refetch`, cached data | O stack já expõe os estados necessários. [CITED: https://tanstack.com/query/v5/docs/framework/react/reference/useQuery] |
| Componentes visuais | Novo chart/alert/badge | Wrappers existentes `@/components/ui/*` | O UI-SPEC fixa shell e componentes. [VERIFIED: 33-UI-SPEC.md] |
| JSON de negócio | `encoding/json.Marshal/Unmarshal` direto | `common.Marshal`, `common.Unmarshal`, `common.DecodeJson` | Regra obrigatória do repo. [VERIFIED: AGENTS.md] |
| Rerank batching | Sub-batching/re-ranking transparente no Router | 4xx determinístico + chunks cliente <=20 | Recomposição/ranking/billing cross-batch não foi aprovada. [VERIFIED: 33-CONTEXT.md] |
| Rollback | Rebuild do commit anterior durante incidente | Digest/tag imutável preservado antes do cutover | Rollback precisa continuar disponível mesmo se registry/source mudar. [VERIFIED: release analysis] |

**Key insight:** os pontos difíceis desta fase são invariantes distribuídas entre middleware, retries, persistence, frontend e release; reutilizar primitives existentes mantém uma única fonte de verdade em cada seam. [VERIFIED: architecture analysis]

## Common Pitfalls

### Pitfall 1: Retry double-counts a terminal request

**What goes wrong:** uma falha transitória e o sucesso seguinte geram duas amostras e distorcem taxa/volume. [VERIFIED: requirements analysis]

**Why it happens:** recorder chamado dentro de cada attempt em vez de na transição terminal do request. [VERIFIED: architecture analysis]

**How to avoid:** CAS/idempotency guard por request e teste com fail-first/succeed-second. [VERIFIED: architecture analysis]

**Warning signs:** `terminal_count` cresce mais que requests de acesso; soma das classes diverge; um request_id aparece em duas finalizações de teste. [VERIFIED: test design]

### Pitfall 2: No-channel remains invisible

**What goes wrong:** disponibilidade mostra 100% mesmo quando o distributor retorna 503. [VERIFIED: incident evidence and codebase inspection]

**Why it happens:** `middleware/distributor.go` aborta antes de `controller.Relay`. [VERIFIED: codebase inspection]

**How to avoid:** iniciar recorder após model/group resolution e finalizar routing-unavailable antes do abort. [VERIFIED: architecture analysis]

**Warning signs:** access log tem `No available channel`, mas `routing_unavailable_count` permanece zero. [VERIFIED: validation design]

### Pitfall 3: Historical buckets look healthy by default

**What goes wrong:** colunas novas com default zero fazem rows antigas parecerem completas. [VERIFIED: migration analysis]

**Why it happens:** schema version/completeness não acompanha o bucket. [VERIFIED: migration analysis]

**How to avoid:** persistir `outcome_schema_version=2` em cada write novo e marcar aggregate/model partial se qualquer bucket contribuinte não tiver o marcador v2 **ou** se sua soma dos sete outcomes diferir de `request_count`. Não use apenas a soma agregada, pois buckets incompletos podem se compensar. [VERIFIED: architecture analysis]

**Warning signs:** janela de 24h muda de partial para completa sem todos os buckets pós-deploy. [VERIFIED: validation design]

### Pitfall 4: Latency aggregate uses the wrong denominator

**What goes wrong:** dividir por terminal_count inclui outcomes sem latency de relay; média fica artificialmente baixa. [VERIFIED: 33-UI-SPEC.md]

**Why it happens:** schema atual tem total de latência, mas não `latency_sample_count`. [VERIFIED: codebase inspection]

**How to avoid:** persistir/retornar o denominador explícito e testar requests sem sample de latency. [VERIFIED: architecture analysis]

**Warning signs:** agregar um no-channel altera latência média mesmo sem request upstream. [VERIFIED: validation design]

### Pitfall 5: Timeout child leaks into retry

**What goes wrong:** attempt 2 falha imediatamente com deadline exceeded. [VERIFIED: codebase flow analysis]

**Why it happens:** handler substitui `c.Request` ou guarda o child context no RelayInfo sem reset. [VERIFIED: codebase flow analysis]

**How to avoid:** criar/cancelar child novo por attempt e limpar/encapsular o campo outbound. [CITED: https://pkg.go.dev/context]

**Warning signs:** retry duration ~0 ms depois do primeiro timeout. [VERIFIED: validation design]

### Pitfall 6: Retry bypasses governor or settlement invariant

**What goes wrong:** segundo attempt não consome lease ou a request é cobrada/registrada duas vezes. [VERIFIED: requirements analysis]

**Why it happens:** retry implementado dentro do adaptor fora do controller/governor lifecycle. [VERIFIED: architecture analysis]

**How to avoid:** manter retry no loop existente, readquirir/finalizar lease por attempt e manter settlement/outcome somente terminal. [VERIFIED: codebase inspection]

**Warning signs:** governor active count fica preso, duas logs de settlement ou duas perf samples. [VERIFIED: validation design]

### Pitfall 7: Readiness proves DB but not routing

**What goes wrong:** channel row e ability existem, mas cache/route ainda não selecionam canal 10. [VERIFIED: incident evidence]

**Why it happens:** provisioning escreve DB fora do controller ou o smoke precede o refresh periódico. [VERIFIED: codebase and incident inspection]

**How to avoid:** chamar sync explícito e poll da mesma seleção do distributor. [VERIFIED: architecture analysis]

**Warning signs:** direct channel test passa e `/v1/rerank` retorna no-channel. [VERIFIED: validation design]

### Pitfall 8: Candidate source gate cannot see the final image

**What goes wrong:** preflight acha o source file no checkout, mas a imagem foi construída de outro commit/context. [VERIFIED: Dockerfile/release inspection]

**Why it happens:** final image contém binário, não source; tag `latest` é mutável. [VERIFIED: Dockerfile/release inspection]

**How to avoid:** label source commit/capability, validar digest, rodar candidate smoke isolado e só então promover. [VERIFIED: release analysis]

**Warning signs:** label não casa com commit preflight ou smoke rerank só é executado após restart live. [VERIFIED: validation design]

### Pitfall 9: Configmap protection preserves drift

**What goes wrong:** fork guard mantém aliases corretos, porém perpetua static governor ceilings `3/1` incompatíveis com o contrato canônico `0/0`. [VERIFIED: `k8s/router-ai-atius/configmap.yaml` and AGENTS.md inspection]

**Why it happens:** proteção de arquivo sem marker/value assertions. [VERIFIED: cross-repo analysis]

**How to avoid:** reconciliar os valores a `0/0` nesta fase e testar alias + ceilings no preflight, sem executar cutover k3s. [VERIFIED: repository policy]

**Warning signs:** configmap protegido contém `EMBEDDING_GOVERNOR_MAX_CONCURRENCY=3` ou batch `=1`. [VERIFIED: codebase inspection]

### Pitfall 10: Evidence leaks rerank content

**What goes wrong:** traces, error body, curl transcript ou screenshot guardam documents/token. [VERIFIED: threat analysis]

**Why it happens:** debug captura request/response integral. [VERIFIED: threat analysis]

**How to avoid:** fixtures sintéticos não sensíveis, headers redigidos, status/count/latency only e artifacts revisados antes de persistir. [CITED: https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md]

**Warning signs:** `Authorization`, `documents`, provider response body ou token aparece em log/evidence. [VERIFIED: security requirements]

## Code Examples

Verified patterns from official sources and existing project code:

### Outbound request inherits a bounded context

```go
// Source: https://pkg.go.dev/net/http@go1.25.3
ctx, cancel := context.WithTimeout(parent, timeout)
defer cancel()

req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
if err != nil {
    return nil, err
}
resp, err := client.Do(req)
```

### Additive GORM migration

```go
// Source: https://gorm.io/docs/migration.html
if err := DB.AutoMigrate(&PerfMetric{}); err != nil {
    return err
}
```

GORM AutoMigrate cria colunas/índices/constraints ausentes e não remove colunas não usadas; esta fase deve ser somente aditiva e não executar cleanup destrutivo no rollback. [CITED: https://gorm.io/docs/migration.html]

### Weighted aggregation

```ts
// Source: 33-UI-SPEC.md
const serviceDenominator =
  counts.success_count +
  counts.routing_unavailable_count +
  counts.transport_timeout_count +
  counts.upstream_http_failure_count +
  counts.invalid_upstream_response_count

const serviceAvailability =
  serviceDenominator === 0
    ? null
    : (counts.success_count / serviceDenominator) * 100
```

O helper deve retornar `null`/discriminated partial para denominador zero ou campo ausente; o component converte isso em `—`, não em zero. [VERIFIED: 33-UI-SPEC.md]

### Refetch without automatic retry

```tsx
// Source: https://tanstack.com/query/v5/docs/framework/react/reference/useQuery
const query = useQuery({
  queryKey: ['performance-metrics', 'summary', 24],
  queryFn: fetchSummary,
  retry: false,
})

<Button onClick={() => query.refetch()} disabled={query.isFetching}>
  {t('Reload metrics')}
</Button>
```

### Stable client-side index recomposition

```ts
// Source: phase contract; Router remains capped at 20 documents.
for (let offset = 0; offset < documents.length; offset += 20) {
  const chunk = documents.slice(offset, offset + 20)
  const response = await rerank({ query, documents: chunk })
  results.push(...response.results.map((item) => ({
    ...item,
    index: offset + item.index,
  })))
}
```

Esse exemplo preserva identidade/índice global, mas não autoriza o Router a produzir um ranking global transparente nem muda billing/accounting. [VERIFIED: 33-CONTEXT.md]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Boolean `success` por relay | Taxonomia terminal fechada + exact-once request recorder | Phase 33 | Separa erro do cliente, política, routing, transport e upstream. [VERIFIED: phase requirements] |
| Média simples de `success_rate`, latency e TPS por modelo | Agregado de raw counts/totals com completeness | Phase 33 | Remove viés por volume e evita zero inventado. [VERIFIED: 33-UI-SPEC.md] |
| Channel test após espera/cache eventual | Sync explícito + poll da seleção autenticada | Phase 33 | Provisioning deixa de depender de timing. [VERIFIED: phase requirements] |
| `http.NewRequest` sem inbound context | Child timeout por attempt + `NewRequestWithContext` | Phase 33 | Cancellation/deadline funciona mesmo com `RELAY_TIMEOUT=0`. [VERIFIED: codebase inspection] [CITED: https://pkg.go.dev/net/http@go1.25.3] |
| Retry global genérico | Budget rerank de uma repetição allowlisted | Phase 33 | Evita retry de 4xx/governor/429 e duplicação. [VERIFIED: phase requirements] |
| Source guard + health genérico após restart | Labeled candidate digest + readiness/rerank smoke pré-promoção + rollback digest | Phase 33 | Imagem incompleta falha antes do cutover. [VERIFIED: cross-repo analysis] |

**Deprecated/outdated:**

- `simpleAverage` para overall em `PerformanceHealthPanel`/`PerformanceOverview`: substituir por helper de soma ponderada. [VERIFIED: codebase inspection]
- Boolean-only `RecordRelaySample(info, success, ...)`: manter wrapper temporário somente se necessário para compatibilidade interna, mas novos caminhos devem finalizar outcome explícito. [VERIFIED: codebase inspection]
- Fixed sleep como readiness: substituir por condição bounded e autenticada. [VERIFIED: phase requirements]
- `/api/status` como única prova de release: manter como health genérico, adicionar proof funcional rerank. [VERIFIED: cross-repo inspection]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Nenhum valor numérico fixo é escolhido na pesquisa. O fallback é selecionado na execução por evidência sanitizada de latência/SLO/deadline, e `rerank_timeout_seconds` positivo no channel sempre o substitui. [RESOLVED] | Open Questions (RESOLVED) / Plan 33-07 | A execução deve falhar antes do código se os inputs não produzirem um bound seguro. |

## Open Questions (RESOLVED)

1. **Qual valor numérico deve ser o fallback de inferência rerank? — RESOLVED**
   - Decision: Plan 33-07 registra `observed_p99_seconds`, `observed_max_seconds`, `documented_slo_seconds` e `outer_deadline_seconds`, calcula `N = ceil_to_5(max(15, 2*p99+5, observed_max+5, slo+5))` e aceita N somente quando `N <= 120` e `N <= outer_deadline_seconds-5`. Sem amostras/SLO/deadline coerentes, a fase para antes de alterar relay. [VERIFIED: validation design]
   - Runtime contract: `rerank_timeout_seconds > 0` em `ChannelSettings` substitui N por channel; ausente/zero usa o N selecionado e documentado por Plan 33-07, nunca um literal 60 assumido. Parent cancellation continua prevalecendo. [VERIFIED: phase requirements]

2. **Como representar buckets históricos durante a janela de transição? — RESOLVED**
   - Decision: cada write novo carrega `outcome_schema_version=2`. Overall/model só é completo se todos os buckets contribuintes têm marcador v2 e, individualmente, `sum(seven outcome counters)==request_count`. Qualquer bucket sem marcador ou com invariante quebrada torna a janela partial; não há backfill inferido. Legacy-all-success e janelas mistas são regressões obrigatórias. [CITED: https://gorm.io/docs/migration.html] [VERIFIED: architecture analysis]
   - Verification split: fórmulas exatas, todos os sete contadores, totais model-to-overall, ambos os denominadores zero e o fixture 8/9 são provados imediatamente no candidato imutável com storage SQLite descartável e somente buckets v2. O gate live de 24 horas sempre valida tipos/marcadores; aceita `complete=false` com KPIs derivados indisponíveis e UI `Dados parciais`, ou `complete=true` com fórmulas exatas. Nenhum dos dois caminhos infere classes legacy, altera métricas de produção ou espera 24 horas. [VERIFIED: plan-review resolution]

3. **Onde o provisioning externo que criou/reativou canal 10 deve chamar readiness? — RESOLVED**
   - Decision: o consumidor canônico é o endpoint autenticado `POST /api/channel/:id/readiness`, protegido por `authz.ChannelOperate`. O fluxo de provisioning/re-enable e o `scripts/smoke-reranker-readiness.py` devem chamá-lo após persistência/cache sync e antes de qualquer rerank smoke; resposta não-ready impede smoke/promoção e restaura/desabilita o channel conforme o fluxo. [VERIFIED: architecture analysis]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| `scripts/podman-admin.sh` | Heavy Go/frontend checks and builds | ✓ | repo-local; profile verified | none; mandatory. [VERIFIED: environment probe] |
| CPU cgroup profile | 20% host cap | ✓ | `cpu.max=80000 100000`; 4 vCPU host | none; stop if verification fails. [VERIFIED: `verify-profile` readback] |
| Podman | Candidate image, label/digest, rollback | ✓ | 4.9.3 | none for production image path. [VERIFIED: environment probe] |
| Bun | Frontend tests/i18n/typecheck/build | ✓ | 1.3.14 | no npm fallback; Bun is project standard. [VERIFIED: environment probe and AGENTS.md] |
| Node.js | GSD/Graphify and frontend tooling | ✓ | v24.13.1 | Bun for repo frontend scripts where supported. [VERIFIED: environment probe] |
| Python | omni-srv-admin preflight tests/scripts | ✓ | 3.12.3 | none documented. [VERIFIED: environment probe] |
| jq | Sanitized JSON assertions in smoke scripts | ✓ | 1.7 | Python JSON parsing if necessary. [VERIFIED: environment probe] |
| curl | Local/authenticated public smoke | ✓ | installed | Go/Python HTTP client only if script requires structured assertions. [VERIFIED: environment probe] |
| Go toolchain | Backend tests/build | ✓ via governed absolute path expected | module declares Go 1.25.1; shell `go` wrapper cannot locate real command | invoke `/usr/local/go/bin/go` only inside `profile-run`; verify before Wave 0. [VERIFIED: environment and repo inspection] |
| TEI reranker service | Functional candidate/live smoke | runtime-dependent | two healthy pods in incident evidence | fail closed; do not promote without reachable authenticated candidate path. [VERIFIED: incident evidence] |
| Channel 10 / DB ability | Readiness and smoke | runtime-dependent | production invariant | readiness endpoint must prove it at execution time. [VERIFIED: 33-CONTEXT.md] |

**Missing dependencies with no fallback:** nenhum tool local ausente; TEI/channel/runtime availability continua um gate live obrigatório. [VERIFIED: environment audit]

**Missing dependencies with fallback:** shell `go` wrapper não resolve diretamente; use o toolchain absoluto dentro do governed profile. [VERIFIED: environment audit]

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Backend framework | Go `testing` + `testify` 1.11.1. [VERIFIED: go.mod and existing tests] |
| Frontend framework | Existing Bun/`node:test` pure-unit pattern; no Vitest dependency required. [VERIFIED: package and test inspection] |
| Config file | Go modules via `go.mod`; Bun via `web/default/package.json`; no new test config. [VERIFIED: codebase inspection] |
| Quick backend run | `./scripts/podman-admin.sh profile-run -- /usr/local/go/bin/go test ./pkg/perf_metrics ./middleware ./relay ./relay/channel ./relay/channel/advancedcustom -run 'Test.*(Outcome|Routing|Rerank|Context|Deadline|Retry|TEI)' -count=1` [VERIFIED: repository test layout] |
| Quick frontend run | `./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun test src/features/performance-metrics/lib/outcome-summary.test.ts'` [VERIFIED: repository tooling] |
| Full phase backend | `./scripts/podman-admin.sh profile-run -- /usr/local/go/bin/go test ./pkg/perf_metrics ./model ./middleware ./controller ./relay ./relay/channel ./relay/channel/advancedcustom ./service/embeddinggovernor -count=1` [VERIFIED: repository test layout] |
| Full frontend | `./scripts/podman-admin.sh profile-run -- bash -lc 'cd web/default && bun run typecheck && bun test src/features/performance-metrics'` [VERIFIED: AGENTS.md and package scripts] |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PHASE-33-METRICS-OUTCOME-TAXONOMY | Sete outcomes, invariant de soma, flush/upsert/query cross-DB. | unit/integration | Governed Go test `./pkg/perf_metrics ./model -run 'Test.*(Outcome|PerfMetric)'`. [VERIFIED: test design] | ❌ Wave 0: expand/create focused tests |
| PHASE-33-ROUTING-AVAILABILITY | No-channel registra routing uma vez; retry não duplica. | middleware/controller integration | Governed Go test `./middleware ./controller -run 'Test.*(NoChannel|SingleTerminal|Retry)'`. [VERIFIED: test design] | ❌ Wave 0 |
| PHASE-33-WEIGHTED-DASHBOARD | Weighted request/service/latency/TPS; labels; zero/partial. | pure unit + component contract | Governed `bun test .../outcome-summary.test.ts`; existing component lint/typecheck. [VERIFIED: test design] | ❌ Wave 0 |
| PHASE-33-CHANNEL-READINESS | DB/ability/cache/route consistency, bounded timeout, admin auth. | service/router integration | Governed Go test `./model ./controller ./router -run 'Test.*ChannelReadiness'`. [VERIFIED: test design] | ❌ Wave 0 |
| PHASE-33-TEI-TIMEOUT-RETRY | Parent cancel, per-attempt deadline, fresh retry context, max one safe retry, leases. | unit/integration with `httptest.Server` | Governed Go test `./relay ./relay/channel ./service/embeddinggovernor -run 'Test.*(Deadline|Context|RerankRetry|Lease)'`. [VERIFIED: test design] | 🟡 Existing files, missing cases |
| PHASE-33-RERANK-BATCH-CONTRACT | 20 accepted; 21 returns deterministic 4xx before governor; indices stable. | unit/integration | Governed Go test `./relay ./relay/channel/advancedcustom -run 'Test.*(RerankDocumentLimit|TEIRerank)'`. [VERIFIED: existing test inspection] | 🟡 Existing partial coverage |
| PHASE-33-FORK-SYNC-PERSISTENCE | Protected paths/markers/env; candidate label/digest; pre-cutover fail. | Python unit + shell dry-run | Governed omni preflight pytest plus `release_preflight.py --candidate ...`; exact wrapper from omni AGENTS. [VERIFIED: cross-repo inspection] | ❌ Wave 0 |
| PHASE-33-TEST-DOC-LIVE-EVIDENCE | Local/public auth smoke, redaction, rollback, docs/readback. | headless smoke/manual release gate | Sanitized readiness + rerank probes against candidate, local live and public live. [VERIFIED: phase requirements] | ❌ Wave 0 scripts/runbook assertions |

### Required deterministic cases

- Table-test every origin/error mapping, including local 400, governor reject, no-channel 503, connection error, deadline, upstream 500, upstream 429, malformed response and success. [VERIFIED: requirements analysis]
- Call `Finish` twice and assert one sample; run fail-first/succeed-second and assert one success terminal plus two governor attempt completions. [VERIFIED: exact-once design]
- Seed model rows with very different volumes and prove overall differs from simple average. [VERIFIED: 33-UI-SPEC.md]
- Assert zero denominator renders `—`; complete zero class renders `0 / 0,00%`; missing field yields `Dados parciais`. [VERIFIED: 33-UI-SPEC.md]
- Cancel parent context and assert `httptest.Server` observes cancellation; timeout attempt 1 and assert attempt 2 receives a fresh non-expired context only when policy permits. [CITED: https://pkg.go.dev/net/http@go1.25.3]
- Create readiness fixtures for channel missing, disabled, alias absent, ability absent, wrong route, stale cache then sync success, timeout and unauthorized caller. [VERIFIED: readiness design]
- Assert protected paths include all four new files and config preflight requires exact two aliases plus canonical `0/0` ceilings. [VERIFIED: AGENTS.md and cross-repo inspection]

### Sampling Rate

- **Per task commit:** quick command for the touched tier through `podman-admin.sh`. [VERIFIED: AGENTS.md]
- **Per wave merge:** full phase backend plus frontend pure tests/typecheck through `podman-admin.sh`. [VERIFIED: AGENTS.md]
- **Phase gate:** all focused suites green, candidate readiness/rerank smoke green, rollback artifact present, then local and authenticated public live evidence. [VERIFIED: phase requirements]

### Wave 0 Gaps

- [ ] `pkg/perf_metrics/metrics_test.go` — outcome enum, exact-once, invariant, weighted raw summary and historical partial behavior. [VERIFIED: test gap inspection]
- [ ] `model/perf_metric_test.go` additions — additive upsert/SUM fields using the project DB test fixture. [VERIFIED: test gap inspection]
- [ ] `middleware/distributor_test.go` or nearest existing middleware fixture — pre-relay no-channel sample. [VERIFIED: test gap inspection]
- [ ] `controller/relay_test.go` additions — retry terminal semantics and rerank budget. [VERIFIED: test gap inspection]
- [ ] `controller/channel_readiness_test.go` and router permission assertion — authenticated bounded readiness. [VERIFIED: test gap inspection]
- [ ] `relay/channel/api_request_test.go` additions — context cancellation/deadline propagation. [VERIFIED: existing test inspection]
- [ ] `relay/embedding_handler_test.go` additions — cap, policy, governor/settlement invariants. [VERIFIED: existing test inspection]
- [ ] `web/default/src/features/performance-metrics/lib/outcome-summary.test.ts` — weighted, zero, partial and legacy behavior. [VERIFIED: test gap inspection]
- [ ] omni-srv-admin release preflight tests — protected files, marker/env drift, label/digest and fail-before-cutover. [VERIFIED: cross-repo test gap inspection]
- [ ] sanitized candidate/live smoke script or assertions — channel 10 readiness, valid <=20 rerank, deterministic 21-doc 4xx, no payload/token in artifacts. [VERIFIED: phase requirements]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes | Reuse token auth for `/v1/rerank` and existing admin authentication for readiness; no anonymous readiness. [VERIFIED: codebase inspection] |
| V3 Session Management | yes | Reuse existing dashboard/session middleware; this phase adds no session primitive. [VERIFIED: codebase inspection] |
| V4 Access Control | yes | Readiness route requires `authz.ChannelOperate`; metric endpoint preserves current permission boundary. [VERIFIED: router inspection] |
| V5 Input Validation | yes | Validate model, channel ID, route, positive timeout and <=20 documents server-side; never trust dashboard/script input. [VERIFIED: requirements analysis] |
| V6 Cryptography | no new crypto | Reuse existing TLS/auth/token facilities; never implement encryption/signatures here. [VERIFIED: scope analysis] |
| V16 Security Logging and Error Handling | yes | Record outcome/count/reason code only; protect sensitive data and return generic errors without stack/secret/full upstream body. [CITED: https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md] |

### Known Threat Patterns for Go relay + operational telemetry

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthorized readiness reveals channel topology or triggers cache churn | Information Disclosure / Denial of Service | `authz.ChannelOperate`, bounded poll, rate/concurrency control, generic reason codes. [VERIFIED: threat analysis] |
| Metric cardinality/memory amplification via arbitrary model/group | Denial of Service | Reuse normalized selected model/group, bounded labels and existing bucket retention; do not add error message/path as dimension. [VERIFIED: codebase and threat inspection] |
| Document/token leakage in logs/evidence | Information Disclosure | Aggregate-only schema, synthetic fixtures, header/body redaction, artifact review. [CITED: https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md] |
| Retry amplification against degraded TEI | Denial of Service | max one retry, allowlisted transport/5xx only, fresh governor lease and bounded timeout. [VERIFIED: requirements analysis] |
| Error/body injection into dashboard or logs | Tampering / Information Disclosure | fixed enum/reason codes, no full upstream body, React text rendering, common JSON wrapper. [CITED: https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md] |
| Candidate substitution/tag drift | Tampering | immutable digest, source commit/capability OCI label, pre-promotion functional smoke and rollback digest. [VERIFIED: release threat analysis] |
| Metrics conceal availability failure | Repudiation | terminal sum invariant, routing pre-relay counter, version/completeness marker and cross-check against sanitized access-log counts. [VERIFIED: observability analysis] |

## Release and Rollback Evidence Contract

1. **Source gate:** fail if candidate commit lacks `relay/channel/advancedcustom/tei_rerank.go`, governed handler/tests, channel settings test, configmap aliases or baseline-equivalent markers. [VERIFIED: phase requirements]
2. **Fork-sync gate:** add the four missing protected paths to `sync.yaml` and guard docs; add post-sync focused tests and exact marker/env assertions. [VERIFIED: cross-repo inspection]
3. **Artifact gate:** build only through governed wrapper; label image with source commit and reranker capability; verify label and immutable digest before any tag promotion. [VERIFIED: AGENTS.md and release analysis]
4. **Runtime candidate gate:** run isolated candidate, prove authenticated readiness for channel 10/alias/ability/route and a synthetic valid rerank; a generic status endpoint is insufficient. [VERIFIED: requirements analysis]
5. **Promotion:** preserve/tag current live digest, then promote candidate digest/restart. [VERIFIED: rollback analysis]
6. **Post-cutover:** validate local authenticated path and authenticated public path, taxonomy counters, no-channel metric test where safely isolated, and UI labels/calculation. [VERIFIED: phase requirements]
7. **Rollback:** on readiness/rerank/public failure, restore previous immutable digest, restart, revalidate generic health plus rerank functional path, and retain sanitized failure evidence. [VERIFIED: phase requirements]
8. **Consumed terminal Graphify closeout:** install versioned `post-commit`/`pre-push` hooks with exact local `core.hooksPath=.githooks` readback.
   Each canonical `docs(33-16):...` or later `docs(phase-33):...` commit performs only a sub-10-second atomic PASS invalidation to mode-0600 PENDING/NO-GO plus HEAD; PENDING carries no fingerprint and never authorizes. Only bounded, awaited foreground `git rev-parse`/`git show`/`mktemp`/`chmod`/`mv`/`rm` helpers are allowed. Graphify, wrapper, closeout, hash runtime, network, container/build tools and background/orphan work are forbidden.
   Range-aware `pre-push` is the sole synchronous consumer: when exact current PASS is absent/stale, it validates the configured 600-second build timeout, adds 120 seconds, and invokes the CPU-governed closeout once through foreground `timeout` against a detached clean temporary worktree at the pushed HEAD; the push caller permits the same 720 seconds. It then revalidates marker, fingerprint, idle state, disabled auto-update, `built_at_commit`, and all three query counts before network mutation. Validated output and mode-0600 PASS/NO-GO state remain ignored under `graphify-out/` and `.git`. [VERIFIED: canonical 10-second commit timeout; execute-plan/execute-phase commit order; `requirements mark-complete`; `phase.complete`; Graphify CLI]

Não grave tokens, documents, request bodies ou provider bodies nos artifacts; evidência suficiente é commit/digest, channel/model IDs, HTTP class/reason code, counts, elapsed time e pass/fail. [CITED: https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md]

### Decision note — Git lifecycle owns terminal freshness

`execute:post` acontece antes de `phase.complete`, e o fluxo canônico ainda cria commits de completion, todo closure e PROJECT evolution. Uma sync dentro do plano ou no hook de workflow não é terminal; porém, o commit helper canônico encerra `git commit` após 10 segundos, enquanto Graphify leva cerca de 219 segundos. A decisão é separar invalidação de prova: `post-commit` apenas grava PENDING/NO-GO/HEAD rapidamente, com helpers foreground allowlisted e aguardados; `pre-push` calcula o fingerprint completo e executa o closeout terminal em foreground depois de todos os commits e antes da mutação remota. O checkout compartilhado pode estar dirty (`use_worktrees=false`), então essa indexação exclusiva de pre-push usa um worktree temporário detached no commit esperado, nunca limpeza global. O caller de push deve permitir pelo menos 720 segundos. Não existe analog seguro no repo; fixtures descartáveis devem provar post-commit abaixo de 2 segundos, allowlist/counts, modo 0600/atomicidade, zero indexação/rede/container/build e ausência de helper sobrevivente, além de espera/invocation count/retry no pre-push, PASS e todas as falhas antes da instalação. [VERIFIED: GSD workflow/handler source and Git/Graphify local interfaces]

## Sources

### Primary (HIGH confidence)

- Repository code: `pkg/perf_metrics/`, `model/perf_metric.go`, `middleware/distributor.go`, `controller/relay.go`, `relay/rerank_handler.go`, `relay/channel/api_request.go`, channel cache/ability and frontend performance components. [VERIFIED: codebase inspection]
- Phase inputs: `33-CONTEXT.md`, `REQUIREMENTS.md`, approved `33-UI-SPEC.md`, `.planning/debug/reranker-errors-last-24h.md`. [VERIFIED: repository documents]
- Cross-repo source: `omni-srv-admin` fork-sync config/guards, release preflight, deploy and restart scripts. [VERIFIED: cross-repo inspection]
- Operational evidence: GBrain `ops/router-ai-atius/reranker-88-89-debug-2026-08-14` and Obsidian incident note. [VERIFIED: GBrain and Obsidian readback]
- Graphify status at commit `bc09737`: fresh, 37,981 nodes, 78,840 edges, 0 commits behind; focused code reads were used when broad semantic query returned no results. [VERIFIED: Graphify status/query]

### Secondary (MEDIUM confidence)

- Go `net/http` official package documentation — request contexts and `NewRequestWithContext`. [CITED: https://pkg.go.dev/net/http@go1.25.3]
- Go `context` official package documentation — derived timeout/cancel lifecycle. [CITED: https://pkg.go.dev/context]
- GORM official migration documentation — additive AutoMigrate/Migrator behavior. [CITED: https://gorm.io/docs/migration.html]
- TanStack Query official v5 documentation — retry/refetch/query state behavior. [CITED: https://tanstack.com/query/v5/docs/framework/react/reference/useQuery]
- OWASP ASVS 5.0 V16 official controls — logging metadata, sensitive data, validation/control failures, unexpected errors and generic error messages. [CITED: https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x25-V16-Security-Logging-and-Error-Handling.md]

### Tertiary (LOW confidence)

- Timeout sem literal presumido: Plan 33-07 seleciona N por evidência runtime sanitizada, com margem explícita de cinco segundos e override positivo por channel. [RESOLVED]

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — nenhuma dependência nova; versões e seams foram lidos no repo, e APIs de context/migration/query foram conferidas em documentação oficial. [VERIFIED: codebase and official docs]
- Architecture: HIGH — pontos de seleção, retry, governor, settlement, persistence, API, UI e release foram rastreados diretamente. [VERIFIED: codebase and cross-repo inspection]
- Pitfalls: HIGH — incident evidence confirmou no-channel omitido, 20-doc local 400 e simple-average; riscos restantes derivam de fluxos concretos e têm testes propostos. [VERIFIED: incident and codebase inspection]

**Research date:** 2026-08-14

**Valid until:** 2026-09-13 para arquitetura estável; revalidar runtime/channel/image imediatamente antes de execução/deploy. [VERIFIED: volatility assessment]
