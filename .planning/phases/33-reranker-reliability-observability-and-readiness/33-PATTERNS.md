# Phase 33: Reranker reliability, observability, and readiness - Pattern Map

**Mapped:** 2026-08-14
**Files analyzed:** 35 new/modified files or generated file groups
**Analogs found:** 35 / 35
**Graphify:** fresh at `bc09737`; focused queries returned no nodes, so assignments below come from focused source reads and the fresh `GRAPH_REPORT.md`

## Scope Notes

- Explicit paths come from `33-CONTEXT.md`, `33-RESEARCH.md`, `33-UI-SPEC.md`, and `33-VALIDATION.md`.
- `controller/channel_readiness.go`, the two smoke scripts, and `docs/ATIUS-RERANKER-RELIABILITY.md` are implied names for required new surfaces; the planner may rename them while preserving the assigned patterns.
- Cross-repo files under `/home/ubuntu/GitHub/omni-srv-admin` are read-only from this phase map and must be changed only by an explicitly assigned omni task.
- Locale JSON files are generated outputs. They must be changed through `web/default/scripts/add-missing-keys.mjs`, then normalized with `bun run i18n:sync`; never edit them manually.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `pkg/perf_metrics/types.go` | model/utility | event-driven + transform | same file: `Sample`, atomic bucket and API DTOs | exact |
| `pkg/perf_metrics/metrics.go` | service | event-driven + batch transform | same file: `Record`, `QuerySummaryAll`, hot/Redis merge | exact |
| `pkg/perf_metrics/flush.go` | service | batch + database I/O | same file: drain, upsert, restore-on-error | exact |
| `pkg/perf_metrics/metrics_test.go` (new/expand) | test | event-driven + transform | `relay/embedding_handler_test.go` table fixtures; `service/embeddinggovernor/governor.go` exact-once lease | role-match |
| `model/perf_metric.go` | model | CRUD + batch | same file: additive GORM upsert and SUM queries | exact |
| `model/perf_metric_test.go` (new/expand) | test | CRUD + batch | `controller/channel_authz_test.go` deterministic testify fixtures | role-match |
| `middleware/distributor.go` | middleware | request-response | same file: pre-controller selection and 503 abort seam | exact |
| `middleware/distributor_test.go` (new) | test | request-response | `router/channel_router_test.go` compact routing assertions | role-match |
| `controller/relay.go` | controller | request-response + retry/event | same file: retry loop, terminal error, refund/settlement seam | exact |
| `controller/relay_test.go` (new/expand) | test | request-response + retry | `relay/embedding_handler_test.go` injected governor and table cases | role-match |
| `controller/perf_metrics.go` | controller | request-response | same file: bounded query parsing and response envelope | exact |
| `controller/channel_readiness.go` (new) | controller | request-response + bounded polling | `controller/codex_oauth.go:219-249` | role-match |
| `controller/channel_readiness_test.go` (new) | test | request-response + event-driven | `controller/channel_authz_test.go`; `router/channel_router_test.go` | role-match |
| `router/channel-router.go` | route | request-response | same file: permission route table | exact |
| `router/channel_router_test.go` | test | request-response | same file: handler/permission identity assertion | exact |
| `dto/channel_settings.go` | config/model | transform | same file: optional channel settings and validation | exact |
| `relay/rerank_handler.go` | service | request-response | same file: validation → governor lease → adaptor → settlement | exact |
| `relay/channel/api_request.go` | utility/service | request-response | same file: request construction and upstream execution | exact |
| `relay/channel/api_request_test.go` | test | request-response | same file: `httptest` Gin/request fixtures | exact |
| `relay/embedding_handler_test.go` | test | request-response | same file: governed rerank metadata and 20-document cap | exact |
| `web/default/src/features/performance-metrics/types.ts` | model | transform | same file: additive API type boundary | exact |
| `web/default/src/features/performance-metrics/lib/outcome-summary.ts` (new) | utility | transform | `lib/format.ts` pure helpers plus current component aggregation | role-match |
| `web/default/src/features/performance-metrics/lib/outcome-summary.test.ts` (new) | test | transform | required Bun/node:test pure-unit pattern from validation contract | role-match |
| `web/default/src/features/dashboard/components/overview/performance-health-panel.tsx` | component | request-response + transform | same file: TanStack query and panel shell | exact |
| `web/default/src/features/dashboard/components/models/performance-overview.tsx` | component | request-response + transform | same file: compact responsive strip | exact |
| `web/default/scripts/add-missing-keys.mjs` | utility/config | file-I/O + transform | same file: all-locale atomic sorted writes | exact |
| `web/default/src/i18n/locales/{en,zh,fr,ja,ru,vi,pt}.json` (generated) | config | file-I/O | `add-missing-keys.mjs:55-79` | role-match |
| `k8s/router-ai-atius/configmap.yaml` | config | deployment config | same file: governor alias/ceiling env | exact |
| `scripts/smoke-reranker-readiness.py` (new) | test/utility | request-response | `scripts/smoke-embeddings.py` | role-match |
| `.planning/config.json` | config | lifecycle control | same file: Graphify enablement; disable unsafe asynchronous auto-update | exact |
| `scripts/phase33-post-execution-closeout.sh` (new) | utility/release gate | Git + file-I/O + governed batch | no repository analog; follow real GSD handler outputs and `podman-admin.sh` | no-analog |
| `scripts/test-phase33-post-execution-closeout.sh` (new) | test | disposable Git/worktree fixtures | omni `test_release_preflight.py` fail-closed fixtures, adapted to shell/Git | role-match |
| `.githooks/post-commit` (new) | Git hook | commit event | no repository analog; exact canonical GSD commit subjects are the interface | no-analog |
| `.githooks/pre-push` (new) | Git hook | push boundary | `.git/hooks/pre-push.sample` only for stdin shape; authorization contract is new | no-analog |
| `docs/ATIUS-RERANKER-RELIABILITY.md` (new) | config/docs | file-I/O | `.planning/debug/reranker-errors-last-24h.md` and omni guard doc | role-match |
| `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/sync.yaml` | config | batch | same file: protected paths + ordered post-sync gates | exact |
| `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/projects/atius-router/UPSTREAM-SYNC-GUARDS.md` | config/docs | batch | same file: invariant + protected paths + checks | exact |
| `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/cli/fork_sync/core/release_preflight.py` | service/utility | file-I/O + batch | same file: conservative violation collectors and structured result | exact |
| `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/cli/tests/test_release_preflight.py` | test | file-I/O + batch | same file: temporary repo fixtures and fail-closed assertions | exact |
| `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync/bin/deploy.sh` | utility/config | batch + deployment | same file: preflight → governed build → inspect → restart → health | exact |

## Pattern Assignments

### 1. Outcome taxonomy, exact-once recording, persistence, and API

**Applies to:**

- `pkg/perf_metrics/types.go`
- `pkg/perf_metrics/metrics.go`
- `pkg/perf_metrics/flush.go`
- `pkg/perf_metrics/metrics_test.go`
- `model/perf_metric.go`
- `model/perf_metric_test.go`
- `controller/perf_metrics.go`

**Imports and package boundaries** — copy the existing separation, not a parallel metrics stack (`pkg/perf_metrics/metrics.go:3-15`):

```go
import (
    "context"
    "math"
    "sort"
    "sync"
    "time"

    "github.com/QuantumNous/new-api/common"
    "github.com/QuantumNous/new-api/model"
    relaycommon "github.com/QuantumNous/new-api/relay/common"
)
```

Keep the fixed outcome enum, raw counters, completeness/schema fields, `Sample`, hot `atomicBucket`, Redis fields, DB fields, query shapes, and frontend response fields aligned. Preserve legacy `success_rate`, `avg_latency_ms`, `avg_tps`, and `models` additively.

**Current sample-to-bucket seam** (`pkg/perf_metrics/types.go:89-105`):

```go
func (b *atomicBucket) add(sample Sample) {
    b.requestCount.Add(1)
    if sample.Success {
        b.successCount.Add(1)
    }
    if sample.LatencyMs > 0 {
        b.totalLatencyMs.Add(sample.LatencyMs)
    }
    if sample.OutputTokens > 0 && sample.GenerationMs > 0 {
        b.outputTokens.Add(sample.OutputTokens)
        b.generationMs.Add(sample.GenerationMs)
    }
}
```

Extend this single seam with seven terminal counters and `latencySampleCount`. The invariant is:

```text
terminal_count == success_count
  + client_validation_count
  + governor_rejection_count
  + routing_unavailable_count
  + transport_timeout_count
  + upstream_http_failure_count
  + invalid_upstream_response_count
```

**Exact-once pattern** (`service/embeddinggovernor/governor.go:186-190,425-431`):

```go
type Lease struct {
    g    *Governor
    once sync.Once
}

func (l *Lease) Finish(success bool, statusCode int, latency time.Duration) {
    if l == nil || l.g == nil {
        return
    }
    l.once.Do(func() {
        l.g.finish(l.batch, success, statusCode, latency)
    })
}
```

Use the same `sync.Once`/CAS semantics for a request-scoped terminal recorder. Copy only immutable, non-sensitive model/group/start metadata into it; never retain a `*gin.Context` for asynchronous completion.

**Flush failure handling** (`pkg/perf_metrics/flush.go:26-57`):

```go
drained := bucket.drain()
if drained.requestCount == 0 {
    deleteOldEmptyBucket(k, key)
    return true
}
err := model.UpsertPerfMetric(&model.PerfMetric{ /* every drained counter */ })
if err != nil {
    bucket.addCounters(drained)
    common.SysError(fmt.Sprintf("failed to flush perf metric bucket model=%s group=%s bucket=%d: %s", k.model, k.group, k.bucketTs, err.Error()))
    return true
}
```

All new counters must be drained, restored, serialized to Redis, parsed from Redis, and upserted together. A partial extension silently loses outcomes.

**Cross-DB GORM upsert** (`model/perf_metric.go:29-48`):

```go
return DB.Clauses(clause.OnConflict{
    Columns: []clause.Column{{Name: "model_name"}, {Name: "group"}, {Name: "bucket_ts"}},
    DoUpdates: clause.Assignments(map[string]interface{}{
        "request_count": gorm.Expr("perf_metrics.request_count + ?", metric.RequestCount),
        "success_count": gorm.Expr("perf_metrics.success_count + ?", metric.SuccessCount),
    }),
}).Create(metric).Error
```

Add integer columns through GORM and extend the `SUM(...)` selects (`model/perf_metric.go:81-115`). Do not introduce JSONB or database-specific operators. Historical rows need an explicit writer/schema completeness marker; zeros created by migration are not proof of classified historical outcomes.

**Controller response/error envelope** (`controller/perf_metrics.go:14-35`):

```go
result, err := perfmetrics.QuerySummaryAll(hours, activeGroups)
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{
        "success": false,
        "message": err.Error(),
    })
    return
}
c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
```

Return an additive `schema_version`, complete `overall` raw totals, and per-model raw totals in `data`; keep the legacy envelope. Compute overall from summed counts/totals, never from model percentages.

**Testing pattern** (`relay/embedding_handler_test.go:217-279`): inject a function seam, restore it with `t.Cleanup`, capture structured metadata, and use `require` for setup/fatal assertions plus `assert` for value checks. Add table cases for every origin/status mapping, double `Finish`, retry fail-then-success, invariant sum, uneven weighted volumes, latency denominator, Redis drain/restore, and historical partial buckets.

---

### 2. Pre-relay routing outcome, relay classification, and rerank retry budget

**Applies to:**

- `middleware/distributor.go`
- `middleware/distributor_test.go`
- `controller/relay.go`
- `controller/relay_test.go`

**Pre-controller routing seam** (`middleware/distributor.go:128-153`):

```go
channel, selectGroup, err = service.CacheGetRandomSatisfiedChannel(&service.RetryParam{
    Ctx: c, ModelName: modelRequest.Model, TokenGroup: usingGroup,
    RequestPath: c.Request.URL.Path, Retry: common.GetPointer(0),
})
if err != nil {
    abortWithOpenAiMessage(c, http.StatusServiceUnavailable, message, types.ErrorCodeModelNotFound)
    return
}
if channel == nil {
    abortWithOpenAiMessage(c, http.StatusServiceUnavailable,
        i18n.T(c, i18n.MsgDistributorNoAvailableChannel, ...),
        types.ErrorCodeModelNotFound)
    return
}
```

Initialize the request recorder after model/group resolution and finalize `routing_unavailable` immediately before both 503 exits. Validation/model parsing failures before selection finalize `client_validation` only when the recorder has sufficient normalized metadata. Every exit must converge on the same idempotent recorder.

**Existing retry lifecycle** (`controller/relay.go:181-237`):

```go
retryParam := &service.RetryParam{Ctx: c, TokenGroup: relayInfo.TokenGroup,
    ModelName: relayInfo.OriginModelName, RequestPath: c.Request.URL.Path,
    Retry: common.GetPointer(0)}
for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
    relayInfo.RetryIndex = retryParam.GetRetry()
    channel, channelErr := getChannel(c, relayInfo, retryParam)
    if channelErr != nil { newAPIError = channelErr; break }
    newAPIError = relayHandler(c, relayInfo)
    if newAPIError == nil { relayInfo.LastError = nil; return }
    relayInfo.LastError = newAPIError
    if !shouldRetry(c, newAPIError, common.RetryTimes-retryParam.GetRetry()) { break }
}
```

Add a rerank-specific decision before the generic `shouldRetry`: maximum one repeat (`RetryIndex == 0`), allowlisted transport errors or safe 5xx only. Never retry client/local 4xx, governor rejection, routing failure, upstream 4xx/429, or invalid response. Each attempt reacquires and finishes its governor lease; only the final request records one terminal outcome and settles/refunds once.

**Current terminal recording seam to replace/bridge** (`controller/relay.go:239-248`):

```go
if newAPIError != nil {
    gopool.Go(func() {
        perfmetrics.RecordRelaySample(relayInfo, false, 0)
    })
}
```

Do not pass mutable `relayInfo`/Gin state into an async closure. Finalize synchronously or capture a sanitized immutable recorder/sample before scheduling persistence.

Classify by origin plus structured error type/code/status, not status alone. Keep a pure table-tested classifier shared by middleware/controller terminal paths.

---

### 3. TEI validation, bounded context, governor accounting, and response integrity

**Applies to:**

- `dto/channel_settings.go`
- `relay/rerank_handler.go`
- `relay/channel/api_request.go`
- `relay/channel/api_request_test.go`
- `relay/embedding_handler_test.go`

**Validation before governor** (`relay/rerank_handler.go:30-42`):

```go
rerankReq, ok := info.Request.(*dto.RerankRequest)
if !ok {
    return types.NewErrorWithStatusCode(..., http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}
if embeddinggovernor.IsGovernedModel(publicModelName) && len(rerankReq.Documents) > 20 {
    return types.NewErrorWithStatusCode(
        fmt.Errorf("%s supports at most %d documents per request", publicModelName, 20),
        types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}
```

Preserve the deterministic 20-document cap and pre-governor rejection. Document client chunks of at most 20 with `globalIndex = chunkOffset + result.index`; do not implement transparent Router reranking across chunks.

**Governor per-attempt lifecycle** (`relay/rerank_handler.go:100-126,142-155`):

```go
lease, reject := acquireRerankGovernor(c.Request.Context(), embeddinggovernor.Request{/* metadata only */})
if reject != nil { /* Retry-After + skip retry */ }
finishGovernor := func(success bool, statusCode int) {
    if lease == nil { return }
    lease.Finish(success, statusCode, time.Since(governorStartedAt))
    lease = nil
}
resp, err := adaptor.DoRequest(c, info, requestBody)
if err != nil {
    finishGovernor(false, http.StatusInternalServerError)
    return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
}
```

Keep this acquisition/finalization inside each attempt. A retry must create a fresh lease and fresh outbound context.

**Outbound context seam** (`relay/channel/api_request.go:307-334`):

```go
fullRequestURL, err := a.GetRequestURL(info)
if err != nil { return nil, fmt.Errorf("get request url failed: %w", err) }
req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)
if err != nil { return nil, fmt.Errorf("new request failed: %w", err) }
// headers and overrides
resp, err := doRequest(c, req, info)
```

Extend this with an explicit outbound context and `http.NewRequestWithContext`. For rerank, derive `context.WithTimeout(c.Request.Context(), configuredTimeout)` per attempt and always cancel it. Do not overwrite `c.Request` with the child context; attempt 2 must receive a new live child.

If the planner accepts the research assumption, add an optional positive `rerank_timeout_seconds` to `dto.ChannelSettings` (`dto/channel_settings.go:9-20`) with a 60s fallback. The duration remains a planning decision, not a locked context decision.

**Response validation and project JSON wrapper** (`relay/channel/advancedcustom/tei_rerank.go:68-98`):

```go
responseBody, err := io.ReadAll(resp.Body)
if err != nil {
    return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
}
var teiResults []teiRerankResult
if err := common.Unmarshal(responseBody, &teiResults); err != nil {
    return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
}
if result.Index < 0 || info == nil || result.Index >= len(info.Documents) {
    return nil, types.NewOpenAIError(fmt.Errorf("TEI rerank result index out of range: %d", result.Index), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
}
```

Map malformed body/index to `invalid_upstream_response`; map completed non-2xx to `upstream_http_failure`; distinguish context deadline/transport from upstream status.

**Existing cap test to extend** (`relay/embedding_handler_test.go:281-321`): preserve the fixture and add 20 accepted, 21 rejected before governor, parent cancellation observed by `httptest.Server`, timeout, fresh retry context, max one safe retry, no retry for 429/4xx/malformed body, and one terminal sample.

---

### 4. Authenticated, bounded channel readiness

**Applies to:**

- `controller/channel_readiness.go`
- `controller/channel_readiness_test.go`
- `router/channel-router.go`
- `router/channel_router_test.go`
- `model/channel_cache.go` (called, not necessarily modified)
- `model/channel_satisfy.go` (called, not necessarily modified)

**Controller pattern** (`controller/codex_oauth.go:219-249`):

```go
channelID, err := strconv.Atoi(c.Param("id"))
if err != nil {
    common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
    return
}
ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
defer cancel()
meta, err := service.ProbeCodexChannelCredential(ctx, channelID)
if err != nil {
    c.JSON(http.StatusOK, gin.H{
        "success": false,
        "message": common.MaskSensitiveInfo(err.Error()),
        "data": meta,
    })
    return
}
c.JSON(http.StatusOK, gin.H{"success": true, "message": "ok", "data": meta})
```

Readiness should validate channel 10, enabled status, exact alias, ability, Advanced Custom `/v1/rerank` route, and group; call `model.InitChannelCache()` synchronously; then bounded-poll the normal selection path until it selects channel 10. Return only `ready`, `channel_id`, `model`, `ability_ready`, `route_ready`, `cache_ready`, `reason_code`, `attempts`, and `elapsed_ms`.

**Atomic cache rebuild** (`model/channel_cache.go:26-40,78-97`): build new maps off-lock, then swap all three maps under `channelSyncLock`. Do not add sleeps as readiness proof.

**Ability/readback pattern** (`model/channel_satisfy.go:8-30`): use `IsChannelEnabledForGroupModel` after sync; normal selection itself must still be the final readiness proof.

**Auth route pattern** (`router/channel-router.go:19-35,39-53`):

```go
channelRoute := apiRouter.Group("/channel")
channelRoute.Use(middleware.AdminAuth())
for _, route := range channelPermissionRoutes {
    channelRoute.Handle(route.method, route.path,
        middleware.RequirePermission(route.permission), route.handler)
}
// readiness route must use authz.ChannelOperate
```

**Permission test pattern** (`router/channel_router_test.go:40-50`): assert method, path, `authz.ChannelOperate`, and handler pointer identity. Controller tests must cover missing/disabled channel, missing alias, missing ability, wrong route, stale cache then success, timeout/cancel, and unauthorized caller.

---

### 5. Weighted React dashboard, states, accessibility, and i18n

**Applies to:**

- `web/default/src/features/performance-metrics/types.ts`
- `web/default/src/features/performance-metrics/lib/outcome-summary.ts`
- `web/default/src/features/performance-metrics/lib/outcome-summary.test.ts`
- `web/default/src/features/dashboard/components/overview/performance-health-panel.tsx`
- `web/default/src/features/dashboard/components/models/performance-overview.tsx`
- `web/default/scripts/add-missing-keys.mjs`
- generated locale JSON files

**Imports/query pattern** (`performance-health-panel.tsx:19-34,57-64`):

```tsx
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'

const metricsQuery = useQuery({
  queryKey: ['perf-metrics-summary', 24],
  queryFn: () => getPerfMetricsSummary(24),
  staleTime: 60 * 1000,
  retry: false,
})
```

Keep the query key/window and cached-data behavior. Add existing `Alert`, `Button`, `Badge`, `Tooltip`, and `Empty` wrappers; do not add packages or charts.

**Panel shell to preserve** (`performance-health-panel.tsx:93-106`):

```tsx
<section className='bg-card h-full overflow-hidden rounded-2xl border shadow-xs'>
  <div className='flex items-center gap-2 border-b px-4 py-3 sm:px-5'>
    <HeartPulse className='text-muted-foreground/60 size-4 shrink-0' aria-hidden='true' />
    <h3 className='text-sm font-semibold'>{t('Performance health')}</h3>
  </div>
  <div className='space-y-3 p-4 sm:p-5'>...</div>
</section>
```

Associate the section with its heading via `aria-labelledby`. Render four KPI cells (2 columns below `sm`, 4 from `sm`) followed by the seven fixed-order outcome rows and up to six model rows. Preserve responsive wrapping and no horizontal scroll.

**Current anti-pattern to remove** (`performance-health-panel.tsx:41-55,71-86`; `performance-overview.tsx:48-82`): both components average `success_rate`, `avg_latency_ms`, and `avg_tps` across model rows. Replace both with one pure helper that sums raw counters/totals and divides once.

Required helper semantics:

```ts
requestSuccess = success_count / terminal_count
serviceDenominator = success_count + routing_unavailable_count
  + transport_timeout_count + upstream_http_failure_count
  + invalid_upstream_response_count
serviceAvailability = success_count / serviceDenominator
averageLatency = total_latency_ms / latency_sample_count
throughput = output_tokens / (generation_ms / 1000)
```

Return a discriminated complete/partial result or `null` per unverifiable KPI. Zero denominator renders `—`; a complete zero class with `terminal_count > 0` renders `0 / 0,00%`; missing fields render `Dados parciais`. Legacy model success may render per-model only and must not enter weighted overall.

**Compact strip pattern** (`performance-overview.tsx:111-170`): preserve the bordered flex-wrap shell and add service availability beside request success; top badges remain request-success only with an explicit accessible label/tooltip.

**Pure test contract:** use Bun/node:test with uneven volumes (prove result differs from simple mean), 8/9 client validation (`88,89%` request success and `100,00%` service availability), one routing failure, zero denominator, complete zero class, missing/legacy fields, latency sample denominator, and no `NaN`/`Infinity`.

**i18n write pattern** (`web/default/scripts/add-missing-keys.mjs:55-79`):

```js
for (const [locale, trans] of Object.entries(newKeys)) {
  const filePath = path.join(LOCALES_DIR, `${locale}.json`)
  const json = JSON.parse(await fs.readFile(filePath, 'utf8'))
  // apply values
  json.translation = Object.fromEntries(
    Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
  )
  await fs.writeFile(filePath, stableStringify(json), 'utf8')
}
```

Put every new English source key and all seven supported locale values in `newKeys`; run the script, then governed `bun run i18n:sync`. Do not hand-edit locale JSON. Preserve protected project/license headers.

---

### 6. Runtime config and sanitized functional smoke

**Applies to:**

- `k8s/router-ai-atius/configmap.yaml`
- `scripts/smoke-reranker-readiness.py`
- `docs/ATIUS-RERANKER-RELIABILITY.md`

**Config analog** (`k8s/router-ai-atius/configmap.yaml:22-30`): keep the exact aliases and reconcile static ceilings to the canonical unbounded values:

```yaml
EMBEDDING_GOVERNOR_ENABLED: "true"
EMBEDDING_GOVERNOR_MODELS: "embedding-gte-v1,reranker-gte-multilingual-v1"
EMBEDDING_GOVERNOR_MAX_CONCURRENCY: "0"
EMBEDDING_GOVERNOR_BATCH_CONCURRENCY: "0"
```

This is config/source persistence only; Phase 33 does not authorize the deferred k3s public cutover.

**Smoke script safety pattern** (`scripts/smoke-embeddings.py:26-47,153-181,184-239`): read token/base/model from environment, scrub token/auth identifiers, bound HTTP timeout, handle HTTP/URL errors, and print concise status/model/count only. The reranker smoke must exercise:

- authenticated readiness for channel 10 + alias + ability + route;
- synthetic valid request with at most 20 documents;
- deterministic 21-document 4xx before governor;
- candidate, local live, and public live base URLs;
- no token, query, document, request body, or provider body in retained output.

The runbook should copy the existing incident evidence style: state the invariant, governed commands, expected sanitized output, candidate digest/source label checks, old digest capture, rollback steps, and post-rollback rerank proof.

---

### 7. Omni fork-sync protection, candidate gate, promotion, and rollback

**Applies to (cross-repo):**

- `modules/fork-sync/projects/atius-router/sync.yaml`
- `modules/fork-sync/projects/atius-router/UPSTREAM-SYNC-GUARDS.md`
- `modules/fork-sync/cli/fork_sync/core/release_preflight.py`
- `modules/fork-sync/cli/tests/test_release_preflight.py`
- `modules/fork-sync/bin/deploy.sh`

**Protected-path pattern** (`sync.yaml:18-66,134-137`): append the four Phase 33 paths required by research — `pkg/perf_metrics/`, `model/perf_metric.go`, `controller/perf_metrics.go`, and `middleware/distributor.go` — and add focused reranker guard/smoke commands to ordered `post_sync`. Preserve the current guard-first ordering.

**Guard documentation pattern** (`UPSTREAM-SYNC-GUARDS.md:25-27,56-103,147-166`): document behavior first, list every carrier path, then list executable post-sync checks and explicit fail-closed language. Update the reranker bullet to include terminal taxonomy/readiness/context/retry persistence, channel 10, both governor aliases, and canonical `0/0` ceilings.

**Static violation collector** (`release_preflight.py:442-500`):

```python
def _atius_router_user_quota_violations(repo: Path) -> list[tuple[Path, str]]:
    violations = []
    for rel in REQUIRED_FILES:
        if not (repo / rel).is_file():
            violations.append((Path(rel), "required ... file is missing"))
    # deterministic marker/value checks
    return violations
```

Add an equivalent reranker collector with fixed required files, source markers, exact aliases, channel/config expectations, candidate source/capability label, immutable digest, and focused-test evidence. Feed violations through `_add_issue` and add a success check only when the collector is empty (`release_preflight.py:693-709`). Never scan or emit credential values.

**Fail-closed result pattern** (`release_preflight.py:778-788,805-826`): return structured `status/errors/warnings/checks`; CLI exit is non-zero whenever `status != success`.

**Test fixture pattern** (`test_release_preflight.py:19-43,388-414,535-552`): create a minimal temporary repo, write all required files with synthetic markers, invoke `run_preflight`, assert the exact error code/path/message, then separately prove `sync.yaml` contains the required protected paths and ordered post-sync gate.

**Current deploy order to change** (`deploy.sh:125-155`):

```bash
"$PREFLIGHT" "${preflight_args[@]}"
run_builds "$BUILD_WRAPPER" build ... -t "$IMAGE_VERSION_REF" -t "$IMAGE_LATEST_REF" ...
run_builds podman push "$IMAGE_VERSION_REF"
run_builds podman push "$IMAGE_LATEST_REF"
run_plain podman image inspect "$IMAGE_VERSION_REF" ...
"$BUILD_WRAPPER" prod-restart
```

Phase 33 must separate candidate from promotion: build a non-live candidate, verify OCI source commit/capability label and immutable digest, start it isolated, run authenticated readiness + rerank smoke, capture the current live digest, and only then update/promote the live tag and restart. Any failed gate restores the preserved digest and revalidates generic health plus functional rerank.

All build/image work stays inside the existing `run_builds` resource profile or the Router `podman-admin.sh` wrapper; never bypass the 20% CPU/0.8 CPU ceiling.

### 8. Git-consumed terminal Graphify closeout

**Sources:** canonical `execute-plan.md` commit `docs(33-16):...`; canonical `execute-phase.md` commits `docs(phase-33): complete phase execution`, todo closure, and PROJECT evolution; `gsd-core/bin/lib/milestone.cjs` requirement checkbox/table transforms; `gsd-core/bin/lib/phase.cjs` ROADMAP/STATE transforms; `graphify --help`; repository `scripts/podman-admin.sh`.

No repo-local closeout hook exists. Implement the lifecycle explicitly:

- `.planning/config.json` keeps `graphify.enabled=true` but sets `graphify.auto_update=false`; the asynchronous/default-branch path is not CPU-governed terminal evidence.
- `post-commit` is a branch-agnostic sub-10-second invalidator for exact `docs(33-16):...` plus `docs(phase-33):...` subjects. It atomically replaces prior PASS with mode-0600 JSON containing only PENDING, NO-GO, and HEAD; PENDING never authorizes and has no fingerprint. Shell builtins plus bounded, awaited foreground `git rev-parse`/`git show`/`mktemp`/`chmod`/`mv`/`rm` are the complete helper allowlist. Graphify, wrapper, closeout, hash runtime, network, container, build, `&`, `nohup`, `setsid`, and `disown` are forbidden, and no helper may survive hook return.
- At pre-push, never require the shared origin checkout to be clean. Snapshot its porcelain, compute the complete-tree SHA-256 fingerprint from `git ls-tree -r -z --full-tree EXPECTED_HEAD`, create a detached clean temporary worktree at that commit, and run its wrapper there. Preserve the shared dirty/untracked user state byte-for-byte.
- Validate the temporary graph's `built_at_commit`, run the three direct `graphify query --graph` probes, atomically publish ignored outputs, then remove the temporary worktree and prove registry/filesystem absence. Original and cleanup failures both remain nonzero.
- `pre-push` parses ref ranges and is the sole synchronous terminal consumer. For a relevant pushed SHA without exact current PASS it validates `graphify.build_timeout`, adds a 120-second margin, and invokes the closeout once through foreground `timeout` with at least 720 seconds; the push caller permits the same interval. It then blocks unless ignored state matches pushed HEAD, tree fingerprint, disabled auto-update, idle lock/status, `built_at_commit`, and three positive query counts. Exact current PASS retries skip reindexing.
- Existing `core.hooksPath` is a trust boundary: unset or exact `.githooks` may proceed; any other value remains untouched and fails installation. Restore the exact prior state if install/readback/tests fail.
- The versioned fixture must drive real hooks and GSD handler-format artifacts through incomplete rc 3, wrong HEAD, tree drift, wrapper failure, stale commit, empty query, cleanup failure, missing/stale pre-push state, dirty-origin PASS, exact-current PASS, post-commit latency below 2 seconds, exact helper allowlist/bounded counts, atomic mode-0600 HEAD-only PENDING, zero Graphify/network/container/build invocation, no surviving background/orphan helper, foreground slow-closeout waiting, invocation counts, and failed-closeout retry.

## Shared Patterns

### Authentication and authorization

**Source:** `router/channel-router.go:19-35`
**Apply to:** readiness route

- `middleware.AdminAuth()` on the channel group.
- `middleware.RequirePermission(authz.ChannelOperate)` on readiness.
- Add handler/permission identity coverage to `router/channel_router_test.go`.

### Error classification and privacy

**Sources:** `controller/relay.go:357-400`, `controller/codex_oauth.go:225-241`, `scripts/smoke-embeddings.py:33-47`
**Apply to:** controller, relay, metrics, readiness, smoke, release evidence

- Persist fixed outcome/reason codes and aggregate counts only.
- Use `common.MaskSensitiveInfo`/sanitized metadata for responses and logs.
- Never persist or print tokens, documents, prompts, query text, request/provider bodies, or URLs containing credentials.

### JSON

**Source:** `relay/channel/advancedcustom/tei_rerank.go:75-78`
**Apply to:** Go business code

Use `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`, or `common.DecodeJson`; do not add direct `encoding/json` marshal/unmarshal calls in business code.

### Database compatibility

**Source:** `model/perf_metric.go:29-48,81-115`
**Apply to:** new performance counters/queries

Use additive GORM fields, `clause.OnConflict`, `gorm.Expr`, and existing `commonGroupCol`. Keep SQLite, MySQL, and PostgreSQL compatible.

### Tests

**Sources:** `relay/embedding_handler_test.go:217-321`, `router/channel_router_test.go:40-50`
**Apply to:** all new Go tests

Use deterministic fixtures, `testify/require` for setup/fatal assertions and `testify/assert` for non-fatal values. Test user-visible/cross-module invariants, not implementation-only coverage.

### UI components and states

**Source:** existing `@/components/ui/*` imports and approved `33-UI-SPEC.md`
**Apply to:** both dashboard components

Reuse `Skeleton`, `Empty`, `Alert`, `Button`, `Badge`, and `Tooltip`; preserve light/dark tokens, compact shell, semantic list markup, accessible labels, 44px recovery target, and `retry: false`.

### i18n

**Source:** `.agents/skills/i18n-translate/SKILL.md` and `web/default/scripts/add-missing-keys.mjs`
**Apply to:** every new user-visible label/copy

Continue `t('English source key')`. Add translations via the script for `en`, `zh`, `fr`, `ja`, `ru`, `vi`, and `pt`, then run `bun run i18n:sync` through the CPU wrapper.

### Heavy verification

**Source:** `AGENTS.md` CPU guardrail
**Apply to:** Go suites, Bun test/typecheck/build, image build

Use `./scripts/podman-admin.sh profile-run -- ...` for test/typecheck/build and `./scripts/podman-admin.sh build ...` for images. Do not run direct heavy `go test ./...`, `bun run build`, or `podman build`.

## No Analog Found

The versioned Phase 33 fast-invalidation `post-commit`, synchronous-closeout `pre-push`, and terminal closeout script have no safe repo-local analog. Their closest interfaces are the canonical GSD commit subjects/handler transforms, Git's pre-push stdin protocol, the existing CPU wrapper, and Graphify's `--graph` query path. Do not copy or chain unknown user hooks into the repository. The new outcome taxonomy remains a Phase 33 semantic addition, but its exact-once mechanism should copy `service/embeddinggovernor.Lease.Finish`.

## Planner Warnings

- The 60-second rerank timeout is research-assumed, not locked. Make it a plan decision/checkpoint or choose an evidence-backed value before implementation.
- Do not infer historical outcome classes from legacy zero-valued columns. Mark mixed windows partial.
- Do not count attempts as terminal requests. Governor accounting is per attempt; perf outcome and billing/settlement are per terminal request.
- Do not use fixed sleeps for readiness. Sync and poll the normal selection path.
- Do not tag/push `latest` before isolated candidate smoke and old-digest capture.
- Do not broaden Phase 33 into the deferred k3s public cutover or transparent Router sub-batching.

## Metadata

**Analog search scope:** `pkg/perf_metrics`, `model`, `middleware`, `controller`, `relay`, `router`, `dto`, `web/default/src/features`, `web/default/scripts`, `k8s/router-ai-atius`, `scripts`, `.githooks`, canonical GSD execute/handler code, and `/home/ubuntu/GitHub/omni-srv-admin/modules/fork-sync`

**Primary analog families:** 8 (metrics, routing/relay, TEI, readiness, frontend, smoke/config, release, Git lifecycle)

**Pattern extraction date:** 2026-08-14
