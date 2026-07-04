# Phase 20: go-native-model-router - Pattern Map

**Mapped:** 2026-06-26
**Files analyzed:** 5 likely modified files
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `service/embeddinggovernor/governor.go` | service | request-response + event-driven lease finish | `service/embeddinggovernor/governor.go` | exact |
| `service/embeddinggovernor/governor_test.go` | test | request-response + state-machine/concurrency | `service/embeddinggovernor/governor_test.go` | exact |
| `relay/embedding_handler.go` | relay integration | request-response + upstream HTTP I/O | `relay/embedding_handler.go` | exact |
| `dto/embedding.go` | dto / utility | transform | `dto/embedding.go` | exact |
| `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | docs / config | operational config + validation | `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | exact |

## Pattern Assignments

### `service/embeddinggovernor/governor.go` (service, request-response + event-driven lease finish)

**Analog:** `service/embeddinggovernor/governor.go`

**Imports pattern** (lines 3-11):

```go
import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
```

**Protected defaults pattern** (lines 13-23):

```go
const (
	defaultModels             = "embedding-pt-v1,embedding-pt-v1-batch"
	defaultBatchModels        = "embedding-pt-v1-batch"
	defaultInitialConcurrency = 2
	defaultMinConcurrency     = 1
	defaultMaxConcurrency     = 3
	defaultBatchConcurrency   = 1
	defaultQueueLimit         = 128
	defaultBatchQueueLimit    = 512
	defaultSuccessWindow      = 8
)
```

Planner notes:
- Keep daily defaults `min=1`, `initial=2`, `max=3`; do not make `4` automatic.
- Add new adaptive defaults as constants near this block.
- Any new threshold must have a safe fallback and deterministic normalization.

**No-input-text invariant** (lines 25-31):

```go
// Request carries only routing metadata. It must never include embedding input text.
type Request struct {
	Model       string
	ChannelID   int
	ChannelName string
	Workload    string
}
```

Planner notes:
- If adding request-local signals, use metadata-only fields such as `InputCount`, coarse `InputChars`, `InputCharsBucket`, or `ClientWorkload`.
- Never add raw `[]string`, raw text, token, Authorization header, or provider secret fields to `Request`, `Governor`, `Snapshot`, logs, docs, or tests.

**Config + env parsing pattern** (lines 40-59, 118-139):

```go
type Config struct {
	Enabled                  bool
	Models                   map[string]bool
	BatchModels              map[string]bool
	InitialConcurrency       int
	MinConcurrency           int
	MaxConcurrency           int
	BatchConcurrency         int
	QueueLimit               int
	BatchQueueLimit          int
	InteractiveTimeout       time.Duration
	BatchTimeout             time.Duration
	Cooldown                 time.Duration
	SlowRequestDuration      time.Duration
	BatchSlowRequestDuration time.Duration
	LatencyTarget            time.Duration
	ScaleUpMinInterval       time.Duration
	ScaleDownIdle            time.Duration
	SuccessWindow            int
}
```

```go
func LoadConfigFromEnv() Config {
	cfg := Config{
		Enabled:                  envBool("EMBEDDING_GOVERNOR_ENABLED", true),
		Models:                   parseCSVSet(envString("EMBEDDING_GOVERNOR_MODELS", defaultModels)),
		BatchModels:              parseCSVSet(envString("EMBEDDING_GOVERNOR_BATCH_MODELS", defaultBatchModels)),
		InitialConcurrency:       envInt("EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY", defaultInitialConcurrency),
		MinConcurrency:           envInt("EMBEDDING_GOVERNOR_MIN_CONCURRENCY", defaultMinConcurrency),
		MaxConcurrency:           envInt("EMBEDDING_GOVERNOR_MAX_CONCURRENCY", defaultMaxConcurrency),
		BatchConcurrency:         envInt("EMBEDDING_GOVERNOR_BATCH_CONCURRENCY", defaultBatchConcurrency),
		QueueLimit:               envInt("EMBEDDING_GOVERNOR_QUEUE_LIMIT", defaultQueueLimit),
		BatchQueueLimit:          envInt("EMBEDDING_GOVERNOR_BATCH_QUEUE_LIMIT", defaultBatchQueueLimit),
		InteractiveTimeout:       envDuration("EMBEDDING_GOVERNOR_INTERACTIVE_TIMEOUT", 30*time.Second),
		BatchTimeout:             envDuration("EMBEDDING_GOVERNOR_BATCH_TIMEOUT", 10*time.Minute),
		Cooldown:                 envDuration("EMBEDDING_GOVERNOR_COOLDOWN", 10*time.Minute),
		SlowRequestDuration:      envDuration("EMBEDDING_GOVERNOR_SLOW_REQUEST_DURATION", 2*time.Minute),
		BatchSlowRequestDuration: envDuration("EMBEDDING_GOVERNOR_BATCH_SLOW_REQUEST_DURATION", 10*time.Minute),
		LatencyTarget:            envDuration("EMBEDDING_GOVERNOR_LATENCY_TARGET", 90*time.Second),
		ScaleUpMinInterval:       envDuration("EMBEDDING_GOVERNOR_SCALE_UP_MIN_INTERVAL", 30*time.Second),
		ScaleDownIdle:            envDuration("EMBEDDING_GOVERNOR_SCALE_DOWN_IDLE", 10*time.Minute),
		SuccessWindow:            envInt("EMBEDDING_GOVERNOR_SUCCESS_WINDOW", defaultSuccessWindow),
	}
	return normalizeConfig(cfg)
}
```

Planner notes:
- Add env vars in `LoadConfigFromEnv`, normalize them in `normalizeConfig`, and document them in the manual.
- Use existing helpers `envBool`, `envInt`, `envDuration`, `envString`, and `parseCSVSet`; do not add a new config package.

**Snapshot fields pattern** (lines 61-82, 240-264):

```go
type Snapshot struct {
	Enabled              bool      `json:"enabled"`
	CurrentConcurrency   int       `json:"current_concurrency"`
	MinConcurrency       int       `json:"min_concurrency"`
	MaxConcurrency       int       `json:"max_concurrency"`
	BatchConcurrency     int       `json:"batch_concurrency"`
	Running              int       `json:"running"`
	RunningBatch         int       `json:"running_batch"`
	WaitingInteractive   int       `json:"waiting_interactive"`
	WaitingBatch         int       `json:"waiting_batch"`
	CooldownUntil        time.Time `json:"cooldown_until,omitempty"`
	ConsecutiveSuccesses int       `json:"consecutive_successes"`
	Completed            uint64    `json:"completed"`
	Failed               uint64    `json:"failed"`
	Slow                 uint64    `json:"slow"`
	AverageLatencyMs     int64     `json:"average_latency_ms"`
	LastSuccessAt        time.Time `json:"last_success_at,omitempty"`
	LastFailureAt        time.Time `json:"last_failure_at,omitempty"`
	LastScaleAt          time.Time `json:"last_scale_at,omitempty"`
	PeakRunning          int       `json:"peak_running"`
	PeakWaiting          int       `json:"peak_waiting"`
}
```

```go
func (g *Governor) Snapshot() Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	return Snapshot{
		Enabled:              g.cfg.Enabled,
		CurrentConcurrency:   g.currentConcurrency,
		MinConcurrency:       g.cfg.MinConcurrency,
		MaxConcurrency:       g.cfg.MaxConcurrency,
		BatchConcurrency:     g.cfg.BatchConcurrency,
		Running:              g.running,
		RunningBatch:         g.runningBatch,
		WaitingInteractive:   g.waitingInteractive,
		WaitingBatch:         g.waitingBatch,
		CooldownUntil:        g.cooldownUntil,
		ConsecutiveSuccesses: g.successes,
		Completed:            g.completed,
		Failed:               g.failed,
		Slow:                 g.slow,
		AverageLatencyMs:     g.latencyEWMA.Milliseconds(),
		LastSuccessAt:        g.lastSuccessAt,
		LastFailureAt:        g.lastFailureAt,
		LastScaleAt:          g.lastScaleAt,
		PeakRunning:          g.peakRunning,
		PeakWaiting:          g.peakWaiting,
	}
}
```

Planner notes:
- Add split metrics here if implementation splits batch/interactive EWMA.
- Snapshot field names should stay snake_case JSON and contain only aggregate metadata.
- Add tests asserting snapshots do not contain input text or secrets.

**Acquire / queue / cooldown pattern** (lines 169-238):

```go
func (g *Governor) Acquire(ctx context.Context, req Request) (*Lease, *Reject) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !g.applies(req.Model) {
		return nil, nil
	}

	batch := g.isBatch(req)
	timeout := g.cfg.InteractiveTimeout
	if batch {
		timeout = g.cfg.BatchTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
```

```go
	for {
		if ctx.Err() != nil {
			g.decrementWaiterLocked(batch)
			return nil, rejectFromContext(ctx.Err(), timeout)
		}
		if wait := g.cooldownWaitLocked(g.clock()); wait > 0 {
			g.mu.Unlock()
			waitForCooldown(ctx, wait)
			g.mu.Lock()
			continue
		}
		if g.canStartLocked(batch) {
			g.decrementWaiterLocked(batch)
			g.running++
			if batch {
				g.runningBatch++
			}
			g.observePeaksLocked()
			return &Lease{g: g, batch: batch}, nil
		}
		g.cond.Wait()
	}
}
```

Planner notes:
- Keep all state mutations under `g.mu`.
- Preserve `context.WithTimeout` around queue waits.
- If classifying large unlabeled arrays as batch/heavy, do it before timeout selection.

**Finish / adaptive state pattern** (lines 267-324):

```go
func (l *Lease) Finish(success bool, statusCode int, latency time.Duration) {
	if l == nil || l.g == nil {
		return
	}
	l.once.Do(func() {
		l.g.finish(l.batch, success, statusCode, latency)
	})
}
```

```go
	if statusCode >= http.StatusInternalServerError {
		success = false
	}
	slowRequestDuration := g.cfg.SlowRequestDuration
	if batch && g.cfg.BatchSlowRequestDuration > 0 {
		slowRequestDuration = g.cfg.BatchSlowRequestDuration
	}
	if slowRequestDuration > 0 && latency >= slowRequestDuration {
		success = false
		g.slow++
	}
```

Planner notes:
- Put status-class policy here or in a small helper close to `finish`.
- Desired follow-up: ordinary client `4xx` should not downscale; `429`, `5xx`, timeout, connection failure, and slow sustained infrastructure pressure should.
- If split EWMA is added, update per-workload latency before `canIncreaseLocked`.

**Workload classification pattern** (lines 337-346):

```go
func (g *Governor) isBatch(req Request) bool {
	workload := strings.ToLower(strings.TrimSpace(req.Workload))
	if workload == "batch" || workload == "bulk" {
		return true
	}
	if workload == "interactive" || workload == "realtime" {
		return false
	}
	return g.cfg.BatchModels[strings.TrimSpace(req.Model)]
}
```

Planner notes:
- Preserve explicit header override.
- Add request-local fallback after header handling and before configured batch model fallback.

**Env helper pattern** (lines 576-618):

```go
func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
```

---

### `service/embeddinggovernor/governor_test.go` (test, request-response + state-machine/concurrency)

**Analog:** `service/embeddinggovernor/governor_test.go`

**Imports / testify pattern** (lines 3-11):

```go
import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

**Env default test pattern** (lines 23-39):

```go
func TestLoadConfigUsesDailySafeDefaults(t *testing.T) {
	t.Setenv("EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_MIN_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_MAX_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_TIMEOUT", "")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_SLOW_REQUEST_DURATION", "")

	cfg := LoadConfigFromEnv()

	assert.Equal(t, 1, cfg.MinConcurrency)
	assert.Equal(t, 2, cfg.InitialConcurrency)
	assert.Equal(t, 3, cfg.MaxConcurrency)
	assert.Equal(t, 1, cfg.BatchConcurrency)
	assert.Equal(t, 10*time.Minute, cfg.BatchTimeout)
	assert.Equal(t, 10*time.Minute, cfg.BatchSlowRequestDuration)
}
```

Planner notes:
- Extend this test when adding env vars.
- Use `t.Setenv` and assert fallback behavior for invalid/empty values.

**Queue/concurrency test pattern** (lines 41-77):

```go
first, reject := g.Acquire(context.Background(), Request{Model: "embedding-pt-v1"})
require.Nil(t, reject)
require.NotNil(t, first)

waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
waiting := make(chan *Lease, 1)
waitingReject := make(chan *Reject, 1)
go func() {
	lease, reject := g.Acquire(waitCtx, Request{Model: "embedding-pt-v1"})
	waiting <- lease
	waitingReject <- reject
}()
require.Eventually(t, func() bool {
	return g.Snapshot().WaitingInteractive == 1
}, time.Second, 10*time.Millisecond)
```

Planner notes:
- Use `require.Eventually` only to observe real goroutine state.
- Avoid sleeps as assertions except when testing cooldown with tight, deterministic bounds.

**Clock-controlled adaptive test pattern** (lines 166-199):

```go
now := time.Unix(100, 0)
g := New(testConfigWith(func(cfg *Config) {
	cfg.InitialConcurrency = 2
	cfg.MaxConcurrency = 3
	cfg.MinConcurrency = 1
	cfg.SuccessWindow = 2
	cfg.Cooldown = time.Minute
}))
g.clock = func() time.Time {
	return now
}
```

```go
now = now.Add(time.Minute + time.Nanosecond)
finishSequential(t, g, true, http.StatusOK)
finishSequential(t, g, true, http.StatusOK)
assert.Equal(t, 2, g.Snapshot().CurrentConcurrency)
```

Planner notes:
- Prefer injected `g.clock` over wall-clock waiting for scale-up/down, cooldown, and health hysteresis tests.

**Fixture helper pattern** (lines 218-252):

```go
func finishSequential(t *testing.T, g *Governor, success bool, statusCode int) {
	t.Helper()
	lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-pt-v1"})
	require.Nil(t, reject)
	require.NotNil(t, lease)
	lease.Finish(success, statusCode, time.Millisecond)
}

func testConfig() Config {
	return Config{
		Enabled:             true,
		Models:              parseCSVSet(defaultModels),
		BatchModels:         parseCSVSet(defaultBatchModels),
		InitialConcurrency:  2,
		MinConcurrency:      1,
		MaxConcurrency:      3,
		BatchConcurrency:    1,
		QueueLimit:          8,
		BatchQueueLimit:     8,
		InteractiveTimeout:  time.Second,
		BatchTimeout:        time.Second,
		Cooldown:            time.Minute,
		SlowRequestDuration: 0,
		LatencyTarget:       0,
		ScaleUpMinInterval:  0,
		ScaleDownIdle:       0,
		SuccessWindow:       20,
	}
}
```

Planner notes:
- New tests should be deterministic table tests for: input-count classification, split EWMA, status classification, snapshot no-text invariant, and optional health hysteresis if included.
- Follow AGENTS.md: `require` for setup/fatal assertions, `assert` for non-fatal value checks.

---

### `relay/embedding_handler.go` (relay integration, request-response + upstream HTTP I/O)

**Analog:** `relay/embedding_handler.go`

**Imports pattern** (lines 3-20):

```go
import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/embeddinggovernor"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)
```

**Request validation / copy / mapping pattern** (lines 22-50):

```go
embeddingReq, ok := info.Request.(*dto.EmbeddingRequest)
if !ok {
	return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected *dto.EmbeddingRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}
publicModelName := embeddingReq.Model

request, err := common.DeepCopy(embeddingReq)
if err != nil {
	return types.NewError(fmt.Errorf("failed to copy request to EmbeddingRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
}

err = helper.ModelMappedHelper(c, info, request)
if err != nil {
	return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
}
```

Planner notes:
- Derive input metadata from `embeddingReq.ParseInput()` before governor `Acquire`.
- Use only counts/coarse size; discard raw strings immediately.
- Keep `publicModelName` for governor scope so aliases remain governed as public models.

**JSON wrapper / outbound body pattern** (lines 52-72):

```go
jsonData, err := common.Marshal(convertedRequest)
if err != nil {
	return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
}

if len(info.ParamOverride) > 0 {
	jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
	if err != nil {
		return newAPIErrorFromParamOverride(err)
	}
}

logger.LogDebug(c, "converted embedding request body: %s", jsonData)
body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
if err != nil {
	return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
}
defer closer.Close()
jsonData = nil
info.UpstreamRequestBodySize = size
```

Planner notes:
- Business JSON marshal/unmarshal must use `common.Marshal` / `common.Unmarshal`.
- Be careful with line 64: converted request body logging can contain embedding text. Do not add governor logs that include `jsonData` or parsed inputs.

**Governor acquire / reject pattern** (lines 75-93):

```go
lease, reject := embeddinggovernor.Acquire(c.Request.Context(), embeddinggovernor.Request{
	Model:       publicModelName,
	ChannelID:   c.GetInt("channel_id"),
	ChannelName: c.GetString("channel_name"),
	Workload:    c.GetHeader("X-Embedding-Workload"),
})
if reject != nil {
	if reject.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int(reject.RetryAfter.Seconds())))
	}
	return types.NewErrorWithStatusCode(fmt.Errorf("%s", reject.Message), types.ErrorCode(reject.Code), reject.StatusCode, types.ErrOptionWithSkipRetry())
}
governorStartedAt := time.Now()
finishGovernor := func(success bool, statusCode int) {
	if lease == nil {
		return
	}
	lease.Finish(success, statusCode, time.Since(governorStartedAt))
	lease = nil
}
```

Planner notes:
- Extend this struct literal with metadata-only fields after updating `embeddinggovernor.Request`.
- Preserve `Retry-After` behavior for queue rejects.
- If adding classification helper, keep it local and testable without upstream network calls.

**Upstream result / status mapping pattern** (lines 96-125):

```go
resp, err := adaptor.DoRequest(c, info, requestBody)
if err != nil {
	finishGovernor(false, http.StatusInternalServerError)
	return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
}

var httpResp *http.Response
if resp != nil {
	httpResp = resp.(*http.Response)
	if httpResp.StatusCode != http.StatusOK {
		finishGovernor(false, httpResp.StatusCode)
		newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}
}
```

Planner notes:
- Existing code marks all non-200 as governor failure. The follow-up should pass status code to a governor-side classifier or classify before `Finish`.
- Preserve `RelayErrorHandler` and `ResetStatusCode` response behavior.

**Supplemental relay analog:** `relay/rerank_handler.go`

**Status mapping + quota post pattern** (lines 81-105):

```go
resp, err := adaptor.DoRequest(c, info, requestBody)
if err != nil {
	return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
}

statusCodeMappingStr := c.GetString("status_code_mapping")
var httpResp *http.Response
if resp != nil {
	httpResp = resp.(*http.Response)
	if httpResp.StatusCode != http.StatusOK {
		newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}
}
```

**Supplemental outbound-body analog:** `relay/common/outbound_body.go`

**Body storage pattern** (lines 9-30):

```go
// NewOutboundJSONBody wraps the already-marshaled upstream request body into a
// BodyStorage. When disk cache is enabled and the payload exceeds the configured
// threshold, the data is written to a temp file and the original []byte can be
// GC'd, significantly reducing the heap residency while waiting for the
// upstream provider to respond (the dominant cost for large base64 payloads).
func NewOutboundJSONBody(data []byte) (body io.Reader, size int64, closer io.Closer, err error) {
	storage, err := common.CreateBodyStorage(data)
	if err != nil {
		return nil, 0, nil, err
	}
	return common.ReaderOnly(storage), storage.Size(), storage, nil
}
```

---

### `dto/embedding.go` (dto / utility, transform)

**Analog:** `dto/embedding.go`

**DTO field pattern** (lines 22-34):

```go
type EmbeddingRequest struct {
	Model            string   `json:"model"`
	Input            any      `json:"input"`
	Type             string   `json:"type,omitempty"`
	EncodingFormat   string   `json:"encoding_format,omitempty"`
	Dimensions       *int     `json:"dimensions,omitempty"`
	User             string   `json:"user,omitempty"`
	Seed             *float64 `json:"seed,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
}
```

Planner notes:
- If adding optional scalar DTO fields, use pointer types with `omitempty` to preserve explicit zero values.
- This follow-up likely does not need a public DTO change; prefer a private metadata helper if possible.

**ParseInput pattern** (lines 59-78):

```go
func (r *EmbeddingRequest) ParseInput() []string {
	if r.Input == nil {
		return make([]string, 0)
	}
	var input []string
	switch r.Input.(type) {
	case string:
		input = []string{r.Input.(string)}
	case []string:
		input = r.Input.([]string)
	case []any:
		input = make([]string, 0, len(r.Input.([]any)))
		for _, item := range r.Input.([]any) {
			if str, ok := item.(string); ok {
				input = append(input, str)
			}
		}
	}
	return input
}
```

Planner notes:
- Use this to derive `input_count` and coarse size.
- Do not store the returned strings in governor state.
- A helper such as `EmbeddingInputStats()` can live here only if it avoids retaining text and has focused tests.

**Token-count caution pattern** (lines 36-46):

```go
func (r *EmbeddingRequest) GetTokenCountMeta() *types.TokenCountMeta {
	var texts = make([]string, 0)

	inputs := r.ParseInput()
	for _, input := range inputs {
		texts = append(texts, input)
	}

	return &types.TokenCountMeta{
		CombineText: strings.Join(texts, "\n"),
	}
}
```

Planner notes:
- This pattern intentionally carries text for token counting; do not copy it into governor metrics/snapshot code.

**Supplemental provider transform analog:** `relay/channel/minimax/embedding.go`

**Provider conversion using ParseInput** (lines 41-50):

```go
func openAIEmbeddingRequestToMiniMax(request dto.EmbeddingRequest) embeddingRequest {
	embeddingType := defaultMiniMaxEmbeddingType
	if validMiniMaxEmbeddingTypes[request.Type] {
		embeddingType = request.Type
	}
	return embeddingRequest{
		Model: request.Model,
		Texts: request.ParseInput(),
		Type:  embeddingType,
	}
}
```

Planner notes:
- This is correct for upstream provider conversion, not for governor state.
- If adding tests around input stats, use synthetic strings and assert only counts/size buckets.

---

### `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` (docs / config, operational config + validation)

**Analog:** `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`

**Governor status pattern** (lines 153-164):

```markdown
## Governor de embeddings Go-native

Estado atualizado em 2026-06-26:

- O governor de embeddings roda dentro do proprio processo Go do router; nao ha sidecar, middleware Python, container adicional ou rota `model-detailed` no caminho canonico.
- Implementacao principal: `service/embeddinggovernor/` e integracao em `relay/embedding_handler.go`.
- Escopo default: `embedding-pt-v1` e `embedding-pt-v1-batch`. Outros modelos passam pelo relay normal sem fila do governor.
- Default operacional diario: concorrencia inicial `2`, minima `1`, maxima `3`, batch limitado a `1`, fila interativa `128`, fila batch `512`, timeout interativo `30s`, timeout batch `10m`, cooldown `10m`.
- O header opcional `X-Embedding-Workload: batch` classifica a chamada como batch; `interactive`/`realtime` forcam tratamento interativo. Sem header, `embedding-pt-v1-batch` e batch e `embedding-pt-v1` e interativo.
```

Planner notes:
- Update this section when adding automatic input-count classification, split metrics, status classification, or optional health probe.
- Keep explicitly saying no Python/model-detailed/sidecar/container path.

**Env documentation pattern** (lines 166-187):

```bash
EMBEDDING_GOVERNOR_ENABLED=true
EMBEDDING_GOVERNOR_MODELS=embedding-pt-v1,embedding-pt-v1-batch
EMBEDDING_GOVERNOR_BATCH_MODELS=embedding-pt-v1-batch
EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY=2
EMBEDDING_GOVERNOR_MIN_CONCURRENCY=1
EMBEDDING_GOVERNOR_MAX_CONCURRENCY=3
EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=1
EMBEDDING_GOVERNOR_QUEUE_LIMIT=128
EMBEDDING_GOVERNOR_BATCH_QUEUE_LIMIT=512
EMBEDDING_GOVERNOR_INTERACTIVE_TIMEOUT=30s
EMBEDDING_GOVERNOR_BATCH_TIMEOUT=10m
EMBEDDING_GOVERNOR_COOLDOWN=10m
EMBEDDING_GOVERNOR_SLOW_REQUEST_DURATION=2m
EMBEDDING_GOVERNOR_BATCH_SLOW_REQUEST_DURATION=10m
EMBEDDING_GOVERNOR_LATENCY_TARGET=90s
EMBEDDING_GOVERNOR_SCALE_UP_MIN_INTERVAL=30s
EMBEDDING_GOVERNOR_SCALE_DOWN_IDLE=10m
EMBEDDING_GOVERNOR_SUCCESS_WINDOW=8
```

Planner notes:
- Add any new env var to this block in the same naming style.
- Include defaults, operational meaning, and safe failure behavior.

**Operational basis / gaps pattern** (lines 189-201):

```markdown
- Resultado operacional: `min=1`, `initial=2`, `max=3` e o perfil diario correto para o router. O valor `4` e validado apenas como override manual/turbo para janelas controladas, porque pode consumir processamento demais do host mesmo sem derrubar o TEI.
- O fallback tecnico deve continuar sendo `1`: em falha real, timeout, restart, health ruim persistente ou latencia lenta sustentada, o governor segura novos despachos durante cooldown e so reabre com uma carga pequena.
- Gaps para o governor perfeito: classificar automaticamente catch-up/batch por tamanho do input ou token/cliente, dividir arrays de embeddings em sub-batches de ate `4` antes do upstream, separar EWMA de latencia batch vs interativa, e incorporar telemetria do TEI/Kubernetes (`ready`, `restarts`, health timeout >= `30s`) antes de aumentar concorrencia.
```

Planner notes:
- If router-side sub-batching is not implemented in this follow-up, keep it documented as a gap, not implied behavior.
- If optional health probing is implemented, document disabled-by-default behavior and hysteresis.

**Validation command pattern** (lines 203-208):

```bash
go test ./service/embeddinggovernor ./relay -count=1
ATIUS_ROUTER_TOKEN=... python3 scripts/smoke-embeddings.py --base-url http://127.0.0.1:3000/v1
```

Planner notes:
- Prefer `/usr/local/go/bin/go` in plan commands when relying on this checkout's researched environment.
- Keep runtime smoke token redacted.

**Fork-sync protected path pattern** (lines 305-320):

```markdown
- `dto/embedding.go`, `relay/channel/minimax/` e `relay/channel/deepseek/`: roteamento Go-native de embeddings MiniMax e URLs OpenAI/Anthropic por provider unico.
- `relay/embedding_handler.go` e `service/embeddinggovernor/`: governor Go-native de embeddings locais, sem sidecar Python.
- `relay/channel/codex/`: adaptador Codex OAuth com suporte a embeddings.
- `service/codex_*.go`: refresh OAuth e protecao de referencias `shared:codex`.
- `docs/` e `.planning/`: requisitos e manual operacional do fork.
```

Planner notes:
- Any new governor behavior should be reflected in protected-path docs so upstream sync does not regress it.

## Shared Patterns

### Go-Native Ownership

**Source:** `AGENTS.md` lines 171-175 and `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` lines 153-159
**Apply to:** `service/embeddinggovernor/governor.go`, `relay/embedding_handler.go`, docs, tests

```markdown
- The Codex adaptor in `relay/channel/codex/` must continue to support embeddings through `https://api.openai.com/v1/embeddings` without ChatGPT-only headers on that request.
- Local TEI embeddings are governed inside the Go router through `service/embeddinggovernor/` and `relay/embedding_handler.go`. Do not reintroduce Python/model-detailed, a sidecar, or an extra container for this path.
```

### JSON Wrappers

**Source:** `AGENTS.md` lines 57-69 and `relay/embedding_handler.go` lines 52-55
**Apply to:** relay integration and any DTO/helper tests that marshal/unmarshal JSON

```go
jsonData, err := common.Marshal(convertedRequest)
if err != nil {
	return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
}
```

### Testify Backend Tests

**Source:** `AGENTS.md` lines 147-160 and `service/embeddinggovernor/governor_test.go` lines 9-10
**Apply to:** all new or substantially rewritten Go backend tests

```go
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
```

Planner notes:
- Use deterministic tests that assert behavior/contracts.
- Avoid coverage-only tests, random/stress tests, sleeps as primary assertions, and implementation-detail lock-in.

### No Sensitive Payloads

**Source:** `service/embeddinggovernor/governor.go` lines 25-31 and `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` lines 476-480
**Apply to:** governor request/state/snapshot/logs/docs/tests

```go
// Request carries only routing metadata. It must never include embedding input text.
type Request struct {
	Model       string
	ChannelID   int
	ChannelName string
	Workload    string
}
```

Planner notes:
- Tests should use a sentinel fake input string and assert it does not appear in `Snapshot` JSON or exposed/loggable governor structs.
- Do not document real tokens, OAuth JSON, provider secrets, request bodies, or channel keys.

### Relay Error Handling

**Source:** `relay/embedding_handler.go` lines 96-125 and `relay/rerank_handler.go` lines 81-105
**Apply to:** `relay/embedding_handler.go`

```go
if httpResp.StatusCode != http.StatusOK {
	finishGovernor(false, httpResp.StatusCode)
	newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
	// reset status code 重置状态码
	service.ResetStatusCode(newAPIError, statusCodeMappingStr)
	return newAPIError
}
```

Planner notes:
- Do not change client-facing relay error response semantics just to improve governor metrics.
- Add governor classification without bypassing `RelayErrorHandler` / `ResetStatusCode`.

### Env Config Normalization

**Source:** `service/embeddinggovernor/governor.go` lines 493-551 and 576-618
**Apply to:** new threshold/env fields in `Config`

```go
if cfg.SuccessWindow < 1 {
	cfg.SuccessWindow = defaultSuccessWindow
}
return cfg
```

Planner notes:
- Every new env-driven field needs a default, parsing helper, normalization rule, and test.

## No Analog Found

No default likely file lacks an analog. The follow-up should modify existing Go-native files rather than create Python/model-detailed/sidecar paths.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| n/a | n/a | n/a | All likely default files have exact in-repo analogs. |

Optional scope note: if the planner chooses to add a separate disabled-by-default TEI health sampler file, keep it under `service/embeddinggovernor/` and copy config/state/test patterns from `service/embeddinggovernor/governor.go` and `governor_test.go`. Do not add Kubernetes client ownership or a runtime sidecar without a separate explicit decision.

## Metadata

**Analog search scope:** `service/`, `service/embeddinggovernor/`, `relay/`, `relay/common/`, `relay/channel/minimax/`, `dto/`, `docs/`, `controller/*_test.go`
**Files scanned:** focused `rg --files` scan across service/relay/dto/docs/controller tests plus exact `rg` searches for governor, snapshot, env, relay, testify, and ParseInput patterns
**Graphify:** `graphify status` returned fresh graph (`stale=false`, `commit_stale=false`, commit `754feaf`); task-specific graph queries for `embeddinggovernor` returned no nodes, so routing used focused reads and `rg`
**Pattern extraction date:** 2026-06-26
