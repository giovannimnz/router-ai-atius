# Phase 25: embedding-governor-auto-workload-inference - Pattern Map

**Mapped:** 2026-07-05
**Files analyzed:** 9 likely modified files
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `service/embeddinggovernor/governor.go` | service | request-response, queue/backpressure, batch | `service/embeddinggovernor/governor.go` | exact |
| `service/embeddinggovernor/governor_test.go` | test | request-response, queue/backpressure, batch | `service/embeddinggovernor/governor_test.go` | exact |
| `relay/embedding_handler.go` | relay/controller | request-response, transform | `relay/embedding_handler.go`; validation analog `relay/helper/valid_request.go` | exact + role-match |
| `relay/embedding_handler_test.go` | test | request-response, captured hook | `relay/embedding_handler_test.go` | exact |
| `dto/embedding.go` | dto/utility | transform, metadata extraction | `dto/embedding.go` | exact |
| `dto/embedding_test.go` | test | transform, metadata privacy | `dto/embedding_test.go` | exact |
| `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | docs/config | operational request-response | `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` | exact |
| `scripts/smoke-embeddings.py` | utility | CLI + request-response | `scripts/smoke-embeddings.py`; smoke analog `scripts/smoke-provider-consolidation.py` | exact + role-match |
| `tests/test_clianything.py` | test | utility helper regression | `tests/test_clianything.py` | exact |

## Pattern Assignments

### `service/embeddinggovernor/governor.go` (service, request-response + queue/backpressure)

**Analog:** `service/embeddinggovernor/governor.go`

**Imports pattern** (lines 3-14):
```go
import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)
```

**Default config pattern** (lines 16-24):
```go
const (
	defaultModels                   = "embedding-gte-v1"
	defaultBatchModels              = ""
	defaultInitialConcurrency       = 2
	defaultMinConcurrency           = 1
	defaultMaxConcurrency           = 3
	defaultBatchConcurrency         = 1
	defaultBatchInputCountThreshold = 4
	defaultBatchInputCharsThreshold = 12000
)
```

Apply Phase 25 here: change/normalize `defaultBatchInputCountThreshold` to `2`; add an `AutoWorkload` config field and `EMBEDDING_GOVERNOR_AUTO_WORKLOAD` env read near the existing threshold envs.

**Metadata-only request pattern** (lines 34-42):
```go
// Request carries only routing metadata. It must never include embedding input text.
type Request struct {
	Model       string
	ChannelID   int
	ChannelName string
	Workload    string
	InputCount  int
	InputChars  int
}
```

Do not add raw input, request body, bearer token, or Authorization-derived fields to this struct.

**Env load pattern** (lines 168-178):
```go
cfg := Config{
	Enabled:                  envBool("EMBEDDING_GOVERNOR_ENABLED", true),
	Models:                   parseCSVSet(envString("EMBEDDING_GOVERNOR_MODELS", defaultModels)),
	BatchModels:              parseCSVSet(envString("EMBEDDING_GOVERNOR_BATCH_MODELS", defaultBatchModels)),
	InitialConcurrency:       envInt("EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY", defaultInitialConcurrency),
	MinConcurrency:           envInt("EMBEDDING_GOVERNOR_MIN_CONCURRENCY", defaultMinConcurrency),
	MaxConcurrency:           envInt("EMBEDDING_GOVERNOR_MAX_CONCURRENCY", defaultMaxConcurrency),
	BatchConcurrency:         envInt("EMBEDDING_GOVERNOR_BATCH_CONCURRENCY", defaultBatchConcurrency),
	BatchInputCountThreshold: envInt("EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD", defaultBatchInputCountThreshold),
	BatchInputCharsThreshold: envInt("EMBEDDING_GOVERNOR_BATCH_INPUT_CHARS_THRESHOLD", defaultBatchInputCharsThreshold),
}
```

Add `AutoWorkload: envBool("EMBEDDING_GOVERNOR_AUTO_WORKLOAD", true)` adjacent to these workload fields.

**Model scope pattern** (lines 424-433):
```go
func (g *Governor) applies(model string) bool {
	if !g.cfg.Enabled {
		return false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	return g.cfg.Models[model]
}
```

Phase 25 helper should expose this behavior without changing semantics, for example `IsGovernedModel(model string)` on `Governor` or package-level wrapper. Unknown models must still no-op.

**Classifier priority pattern** (lines 435-450):
```go
func (g *Governor) isBatch(req Request) bool {
	workload := strings.ToLower(strings.TrimSpace(req.Workload))
	if workload == "batch" || workload == "bulk" {
		return true
	}
	if workload == "interactive" || workload == "realtime" {
		return false
	}
	if g.cfg.BatchInputCountThreshold > 0 && req.InputCount >= g.cfg.BatchInputCountThreshold {
		return true
	}
	if g.cfg.BatchInputCharsThreshold > 0 && req.InputChars >= g.cfg.BatchInputCharsThreshold {
		return true
	}
	return g.cfg.BatchModels[strings.TrimSpace(req.Model)]
}
```

Preserve this order: explicit valid header first, metadata thresholds second, configured batch model set last. If `AutoWorkload=false` is added, it should disable only the metadata inference step, not explicit header compatibility.

**Normalization pattern** (lines 745-775):
```go
func normalizeConfig(cfg Config) Config {
	if cfg.Models == nil {
		cfg.Models = parseCSVSet(defaultModels)
	}
	if cfg.BatchModels == nil {
		cfg.BatchModels = parseCSVSet(defaultBatchModels)
	}
	// ...
	if cfg.BatchInputCountThreshold <= 0 {
		cfg.BatchInputCountThreshold = defaultBatchInputCountThreshold
	}
	if cfg.BatchInputCharsThreshold <= 0 {
		cfg.BatchInputCharsThreshold = defaultBatchInputCharsThreshold
	}
```

Use this normalization style for invalid threshold values and any new boolean default.

**Split feedback pattern** (lines 363-410):
```go
g.latencyEWMA = blendDuration(g.latencyEWMA, latency)
if batch {
	g.batchLatencyEWMA = blendDuration(g.batchLatencyEWMA, latency)
} else {
	g.interactiveLatencyEWMA = blendDuration(g.interactiveLatencyEWMA, latency)
}
// ...
if batch {
	g.batchCompleted++
} else {
	g.interactiveCompleted++
}
```

Keep batch and interactive accounting separated when changing classifier logic.

---

### `service/embeddinggovernor/governor_test.go` (test, request-response + queue/backpressure)

**Analog:** `service/embeddinggovernor/governor_test.go`

**Imports/test stack pattern** (lines 3-14):
```go
import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

Use `require` for setup/fatal checks and `assert` for value checks.

**Unknown model no-op pattern** (lines 16-24):
```go
func TestAcquireNoopsForNonGovernedModel(t *testing.T) {
	g := New(testConfig())

	lease, reject := g.Acquire(context.Background(), Request{Model: "gpt-5.4"})

	require.Nil(t, reject)
	assert.Nil(t, lease)
	assert.Equal(t, 0, g.Snapshot().Running)
}
```

Extend or add helper tests so `IsGovernedModel("embedding-gte-v1") == true` and unknown models remain false/no-op.

**Safe defaults pattern** (lines 26-45):
```go
cfg := LoadConfigFromEnv()

assert.Equal(t, 1, cfg.MinConcurrency)
assert.Equal(t, 2, cfg.InitialConcurrency)
assert.Equal(t, 3, cfg.MaxConcurrency)
assert.Equal(t, 1, cfg.BatchConcurrency)
assert.True(t, cfg.Models["embedding-gte-v1"])
assert.False(t, cfg.Models["embedding-gte-v1-batch"])
assert.Empty(t, cfg.BatchModels)
```

Add assertions for `AutoWorkload == true` and `BatchInputCountThreshold == 2`.

**Table-driven classifier pattern** (lines 47-92):
```go
tests := []struct {
	name string
	req  Request
	want bool
}{
	{
		name: "small unlabeled interactive request stays interactive",
		req: Request{
			Model:      "embedding-gte-v1",
			InputCount: 1,
			InputChars: 640,
		},
		want: false,
	},
}

for _, tc := range tests {
	t.Run(tc.name, func(t *testing.T) {
		assert.Equal(t, tc.want, g.isBatch(tc.req))
	})
}
```

Update the count-threshold test from `4` to `2`; add explicit no-header single string, no-header two-item array, and char-threshold cases.

**Header override pattern** (lines 94-152):
```go
{
	name: "batch header forces batch below thresholds",
	req: Request{
		Model:      "embedding-gte-v1",
		Workload:   "batch",
		InputCount: 1,
		InputChars: 320,
	},
	want: true,
},
{
	name: "interactive header wins over count threshold",
	req: Request{
		Model:      "embedding-gte-v1",
		Workload:   "interactive",
		InputCount: 8,
		InputChars: 24000,
	},
	want: false,
},
```

Keep `batch|bulk` and `interactive|realtime` compatibility. Add an invalid-header case that falls through to metadata inference.

**Snapshot privacy pattern** (lines 375-417):
```go
payload, err := common.Marshal(g.Snapshot())
require.NoError(t, err)

raw := strings.ToLower(string(payload))
assert.Contains(t, raw, "\"interactive_average_latency_ms\"")
assert.Contains(t, raw, "\"batch_average_latency_ms\"")
assert.NotContains(t, raw, "input_count")
assert.NotContains(t, raw, "input_chars")
assert.NotContains(t, raw, "workload")
assert.NotContains(t, raw, "authorization")
assert.NotContains(t, raw, "token")
assert.NotContains(t, raw, "secret")
```

Re-run this pattern if new helper/snapshot fields are added.

**Test fixture pattern** (lines 722-752):
```go
func testConfig() Config {
	return Config{
		Enabled:            true,
		Models:             parseCSVSet(defaultModels),
		BatchModels:        parseCSVSet(defaultBatchModels),
		InitialConcurrency: 2,
		MinConcurrency:     1,
		MaxConcurrency:     3,
		BatchConcurrency:   1,
		QueueLimit:         8,
		BatchQueueLimit:    8,
	}
}

func testConfigWith(update func(*Config)) Config {
	cfg := testConfig()
	update(&cfg)
	return cfg
}
```

Set Phase 25 defaults in this fixture if classifier tests should use default threshold `2`.

---

### `relay/embedding_handler.go` (relay/controller, request-response + transform)

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
)
```

Use `common.Marshal`/`common.DeepCopy`; do not add direct `encoding/json` marshal/unmarshal in relay business code.

**Test seam pattern** (line 22):
```go
var acquireEmbeddingGovernor = embeddinggovernor.Acquire
```

Keep this package-local hook for relay tests. New classification/cap tests should capture the request through this seam.

**Public model + metadata boundary pattern** (lines 24-35):
```go
embeddingReq, ok := info.Request.(*dto.EmbeddingRequest)
if !ok {
	return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected *dto.EmbeddingRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}
publicModelName := embeddingReq.Model
inputStats := embeddingReq.GetInputStats()

request, err := common.DeepCopy(embeddingReq)
```

The governor must receive the public model alias before upstream model mapping, plus only numeric input stats.

**Acquire pattern** (lines 78-90):
```go
lease, reject := acquireEmbeddingGovernor(c.Request.Context(), embeddinggovernor.Request{
	Model:       publicModelName,
	ChannelID:   c.GetInt("channel_id"),
	ChannelName: c.GetString("channel_name"),
	Workload:    c.GetHeader("X-Embedding-Workload"),
	InputCount:  inputStats.InputCount,
	InputChars:  inputStats.InputChars,
})
if reject != nil {
	if reject.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int(reject.RetryAfter.Seconds())))
	}
	return types.NewErrorWithStatusCode(fmt.Errorf("%s", reject.Message), types.ErrorCode(reject.Code), reject.StatusCode, types.ErrOptionWithSkipRetry())
}
```

Phase 25 relay tests should prove no-header requests arrive here with sufficient metadata for `ClassifyWorkload`, or that the request already carries a resolved workload before `Acquire`.

**Finish pattern** (lines 92-130):
```go
governorStartedAt := time.Now()
finishGovernor := func(success bool, statusCode int) {
	if lease == nil {
		return
	}
	lease.Finish(success, statusCode, time.Since(governorStartedAt))
	lease = nil
}
// ...
if httpResp.StatusCode != http.StatusOK {
	finishGovernor(false, httpResp.StatusCode)
	newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
	service.ResetStatusCode(newAPIError, statusCodeMappingStr)
	return newAPIError
}
// ...
finishGovernor(true, http.StatusOK)
```

Do not bypass lease finish paths when inserting cap validation.

**Fail-closed validation analog** (`relay/helper/valid_request.go` lines 97-115):
```go
func GetAndValidateEmbeddingRequest(c *gin.Context, relayMode int) (*dto.EmbeddingRequest, error) {
	var embeddingRequest *dto.EmbeddingRequest
	err := common.UnmarshalBodyReusable(c, &embeddingRequest)
	if err != nil {
		logger.LogError(c, fmt.Sprintf("getAndValidateTextRequest failed: %s", err.Error()))
		return nil, types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	if embeddingRequest.Input == nil {
		return nil, fmt.Errorf("input is empty")
	}
	// ...
	return embeddingRequest, nil
}
```

If planner chooses fail-closed TEI cap, use this style: return an invalid-request error before upstream dispatch. Prefer checking only governed local TEI scope, not all embedding providers, unless explicitly intended.

**Current OpenAI embedding gap** (`relay/channel/openai/adaptor.go` lines 359-360):
```go
func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}
```

The OpenAI-compatible TEI path currently passes arrays through unchanged. Do not assume the adaptor enforces max client batch size.

**Provider conversion analog for optional sub-batching** (`relay/channel/minimax/embedding.go` lines 53-75):
```go
openAIResponse := &dto.OpenAIEmbeddingResponse{
	Object: "list",
	Data:   make([]dto.OpenAIEmbeddingResponseItem, 0, len(response.Vectors)),
	Model:  model,
	Usage: dto.Usage{
		PromptTokens: response.TotalTokens,
		TotalTokens:  response.TotalTokens,
	},
}
for index, vector := range response.Vectors {
	openAIResponse.Data = append(openAIResponse.Data, dto.OpenAIEmbeddingResponseItem{
		Object:    "embedding",
		Index:     index,
		Embedding: vector,
	})
}
```

This is only a response-shape analog. There is no close analog for transparent TEI sub-batch request splitting plus response/usage recomposition in the relay.

---

### `relay/embedding_handler_test.go` (test, request-response + captured hook)

**Analog:** `relay/embedding_handler_test.go`

**Imports pattern** (lines 3-20):
```go
import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/embeddinggovernor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

**Gin relay setup pattern** (lines 22-44):
```go
gin.SetMode(gin.TestMode)

recorder := httptest.NewRecorder()
c, _ := gin.CreateTestContext(recorder)
c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
c.Request.Header.Set("X-Embedding-Workload", "batch")
common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
common.SetContextKey(c, constant.ContextKeyChannelId, 77)
common.SetContextKey(c, constant.ContextKeyChannelName, "Local TEI - GTE Embeddings")
common.SetContextKey(c, constant.ContextKeyOriginalModel, "embedding-gte-v1")

request := &dto.EmbeddingRequest{
	Model: "embedding-gte-v1",
	Input: []string{first, second},
}
```

Add cases without setting `X-Embedding-Workload`: single string should be interactive metadata; two-item array should be batch metadata.

**Acquire capture pattern** (lines 46-60):
```go
originalAcquire := acquireEmbeddingGovernor
t.Cleanup(func() {
	acquireEmbeddingGovernor = originalAcquire
})

var captured embeddinggovernor.Request
acquireEmbeddingGovernor = func(ctx context.Context, req embeddinggovernor.Request) (*embeddinggovernor.Lease, *embeddinggovernor.Reject) {
	captured = req
	return nil, &embeddinggovernor.Reject{
		StatusCode: http.StatusTooManyRequests,
		Code:       "embedding_governor_queue_full",
		Message:    "synthetic governor reject",
		RetryAfter: 3 * time.Second,
	}
}
```

Use this to stop before upstream dispatch and assert classification inputs.

**Assertions pattern** (lines 62-77):
```go
err := EmbeddingHelper(c, info)

require.NotNil(t, err)
assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
assert.Equal(t, "3", recorder.Header().Get("Retry-After"))
assert.Equal(t, "embedding-gte-v1", captured.Model)
assert.Equal(t, "batch", captured.Workload)
assert.Equal(t, 2, captured.InputCount)
assert.Equal(t, len(first)+len(second), captured.InputChars)
assert.NotContains(t, err.Error(), first)
assert.NotContains(t, err.Error(), second)
```

For no-header tests, assert `captured.Workload == ""` unless implementation resolves workload before acquire. Keep no-raw-text assertions.

---

### `dto/embedding.go` (dto/utility, transform)

**Analog:** `dto/embedding.go`

**DTO optional scalar pattern** (lines 23-35):
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

Keep optional upstream scalar request fields as pointers with `omitempty`.

**Metadata stats pattern** (lines 37-66):
```go
type EmbeddingInputStats struct {
	InputCount int
	InputChars int
}

func (r *EmbeddingRequest) GetInputStats() EmbeddingInputStats {
	if r == nil {
		return EmbeddingInputStats{}
	}

	inputs := r.ParseInput()
	stats := EmbeddingInputStats{InputCount: len(inputs)}
	for _, input := range inputs {
		stats.InputChars += utf8.RuneCountInString(input)
	}
	return stats
}
```

Reuse this for classifier input. If TEI cap enforcement needs count only, use `GetInputStats().InputCount`; do not pass raw input into governor.

**Input normalization pattern** (lines 78-97):
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

If Phase 25 adds helper(s) for max input count, keep them based on this normalized view.

---

### `dto/embedding_test.go` (test, transform + metadata privacy)

**Analog:** `dto/embedding_test.go`

**Imports pattern** (lines 3-10):
```go
import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

**Nil stats pattern** (lines 12-19):
```go
func TestEmbeddingInputStatsNilInput(t *testing.T) {
	req := &EmbeddingRequest{}

	stats := req.GetInputStats()

	assert.Equal(t, 0, stats.InputCount)
	assert.Equal(t, 0, stats.InputChars)
}
```

**Table + privacy pattern** (lines 21-68):
```go
tests := []struct {
	name      string
	input     any
	wantCount int
	wantChars int
}{
	{
		name:      "string slice",
		input:     []string{short, medium},
		wantCount: 2,
		wantChars: len(short) + len(medium),
	},
}

for _, tc := range tests {
	t.Run(tc.name, func(t *testing.T) {
		req := &EmbeddingRequest{Input: tc.input}

		stats := req.GetInputStats()

		require.NotNil(t, req.GetTokenCountMeta())
		assert.Equal(t, tc.wantCount, stats.InputCount)
		assert.Equal(t, tc.wantChars, stats.InputChars)

		rendered := fmt.Sprintf("%#v", stats)
		assert.NotContains(t, rendered, short)
		assert.NotContains(t, rendered, medium)
		assert.NotContains(t, rendered, long)
	})
}
```

Add any cap/helper tests here only if the implementation lives in `dto/embedding.go`.

---

### `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md` (docs/config, operational request-response)

**Analog:** `docs/MANUAL-OPERACAO-ROUTER-AI-ATIUS.md`

**Governor contract pattern** (lines 165-187):
```markdown
## Governor de embeddings Go-native

Estado atualizado em 2026-06-26:

- O governor de embeddings roda dentro do proprio processo Go do router; nao ha sidecar, middleware Python, container adicional ou rota `model-detailed` no caminho canonico.
- Implementacao principal: `service/embeddinggovernor/` e integracao em `relay/embedding_handler.go`.
- Escopo default: somente `embedding-gte-v1`. Outros modelos passam pelo relay normal sem fila do governor.
- `embedding-gte-v1` e o unico alias publico governado; `EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1` nao muda durante a recuperacao/catalog restore.
- Envelope automatico protegido: `min=1`, `initial=2`, `max=3`, `batch_concurrency=1`, fila interativa `128`, fila batch `512`, timeout interativo `30s`, timeout batch `10m`, cooldown `10m`. O valor `4` continua reservado para override/manual turbo window; nao faz parte da escala automatica diaria.
- Classificacao de workload e metadata-only. Ordem de precedencia:
  1. `X-Embedding-Workload: batch|bulk|interactive|realtime`;
  2. thresholds locais derivados do request (`InputCount >= 4` ou `InputChars >= 12000`).
```

Update threshold docs to `InputCount >= 2`; document `X-Embedding-Workload` as optional override, not a client requirement.

**Env block pattern** (lines 190-215):
```bash
EMBEDDING_GOVERNOR_ENABLED=true
EMBEDDING_GOVERNOR_MODELS=embedding-gte-v1
EMBEDDING_GOVERNOR_BATCH_MODELS=
EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY=2
EMBEDDING_GOVERNOR_MIN_CONCURRENCY=1
EMBEDDING_GOVERNOR_MAX_CONCURRENCY=3
EMBEDDING_GOVERNOR_BATCH_CONCURRENCY=1
EMBEDDING_GOVERNOR_QUEUE_LIMIT=128
EMBEDDING_GOVERNOR_BATCH_QUEUE_LIMIT=512
```

Add `EMBEDDING_GOVERNOR_AUTO_WORKLOAD=true` and `EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD=2` near the other workload envs.

**TEI cap evidence pattern** (lines 226-236):
```markdown
- `GBRAIN_EMBED_PROVIDER_BATCH_SIZE=4` foi o sub-batch que passou em slug que falhava; o TEI tambem registrou que o backend nao suporta batch client acima de `4`.
- O ajuste que mudou o envelope operacional foi: `max_client_batch_size=4`, probes mais tolerantes, health monitorado com timeout de `30s`, pod novo `1/1`, `restarts=0`, limite `3 CPU / 12Gi`.
- Resultado operacional final desta iteracao: `min=1`, `initial=2`, `max=3` continuam como baseline diario; `4` segue restrito a janela manual/turbo.
```

Preserve max client batch size `4` as an invariant; clarify whether router rejects larger arrays or sub-batches them.

**Validation command pattern** (lines 238-259):
```bash
/usr/local/go/bin/go test ./service/embeddinggovernor -count=1
/usr/local/go/bin/go test ./common ./controller ./service/modelcatalog ./relay/common ./service/embeddinggovernor ./relay -count=1

test -n "$ATIUS_ROUTER_TOKEN" && \
  ATIUS_ROUTER_EMBEDDINGS_BASE_URL=http://127.0.0.1:3000/v1 \
  ATIUS_ROUTER_EMBEDDINGS_MODEL=embedding-gte-v1 \
  ATIUS_ROUTER_EXPECT_EMBEDDING_DIM=768 \
  python3 scripts/smoke-embeddings.py

node /home/ubuntu/.codex/gsd-core/bin/gsd-tools.cjs graphify status
```

Add no-header and optional array-mode smoke command once script supports it.

**Secret hygiene pattern** (lines 261-265):
```markdown
- Para automacao e validacao, recupere `ATIUS_ROUTER_API_KEY` do HashiCorp Vault machine/automation e exporte como `ATIUS_ROUTER_TOKEN` apenas no shell efemero do teste.
- Sem `ATIUS_ROUTER_TOKEN`, o smoke de embeddings deve falhar com `exit 2` antes da rede. Isso e limitacao de ambiente, nao passe livre.
- Se `graphify status` retornar `stale=true` ou `commit_stale=true` num checkout com Graphify habilitado, rebuild e obrigatorio antes de assinar a mudanca.
```

Do not paste tokens or runtime secret values into docs.

---

### `scripts/smoke-embeddings.py` (utility, CLI + request-response)

**Analog:** `scripts/smoke-embeddings.py`

**Defaults pattern** (lines 17-23):
```python
REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_BASE_URL = "http://127.0.0.1:3001/v1"
DEFAULT_MODEL = "embo-01"
DEFAULT_EXPECTED_DIM = 1536
USER_AGENT = os.environ.get("ATIUS_ROUTER_USER_AGENT", "Mozilla/5.0 AtiusRouterSmoke/1.0")
MAX_OUTPUT_CHARS = 180
ACCEPTABLE_UPSTREAM_CODES = {400, 402, 429}
```

For Phase 25, update defaults or docs-driven env mode to `http://127.0.0.1:3000/v1`, `embedding-gte-v1`, expected dimension `768`.

**Secret scrubbing pattern** (lines 38-47):
```python
def _scrub(message: str, secrets: Iterable[str]) -> str:
    scrubbed = message
    for secret in secrets:
        if secret:
            scrubbed = scrubbed.replace(secret, "<redacted>")
    scrubbed = scrubbed.replace("GroupId", "<redacted-group-id>")
    scrubbed = scrubbed.replace("group_id", "<redacted-group-id>")
    scrubbed = scrubbed.replace("Authorization", "<redacted-auth>")
    scrubbed = scrubbed.replace("x-api-key", "<redacted-auth>")
    return _short_text(scrubbed)
```

Preserve this for every new error path and output line.

**Payload builder pattern** (lines 113-125):
```python
def build_embedding_payload(
    *,
    model: str,
    input_text: str,
    embedding_type: str | None = None,
    openai_dimensions: int | None = None,
) -> dict[str, Any]:
    payload: dict[str, Any] = {"model": model, "input": input_text}
    if model == "embo-01":
        payload["type"] = embedding_type if embedding_type in {"query", "db"} else "query"
    if openai_dimensions is not None:
        payload["dimensions"] = openai_dimensions
    return payload
```

If adding array/no-header smoke, extend this builder with an env-controlled `input` shape; do not add `X-Embedding-Workload` by default.

**HTTP request pattern** (lines 139-152):
```python
def _request_embeddings(base_url: str, token: str, payload: dict[str, Any]) -> tuple[int, dict[str, Any] | None, str]:
    url = base_url.rstrip("/") + "/embeddings"
    body = json.dumps(payload).encode("utf-8")
    request = Request(
        url,
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "User-Agent": USER_AGENT,
        },
        method="POST",
    )
```

Keep authenticated request behavior but do not print headers/token.

**Config/env pattern** (lines 170-186):
```python
token = _env("ATIUS_ROUTER_TOKEN")
if token is None:
    print(
        "Missing ATIUS_ROUTER_TOKEN; export it to run the embeddings smoke test.",
        file=sys.stderr,
    )
    return 2

base_url = _env("ATIUS_ROUTER_EMBEDDINGS_BASE_URL", DEFAULT_BASE_URL) or DEFAULT_BASE_URL
model = _env("ATIUS_ROUTER_EMBEDDINGS_MODEL", DEFAULT_MODEL) or DEFAULT_MODEL
embedding_type = _env("ATIUS_ROUTER_EMBEDDING_TYPE", "query") or "query"
expected_dim_raw = _env("ATIUS_ROUTER_EXPECT_EMBEDDING_DIM")
```

Add envs here for array/no-header smoke if needed, for example `ATIUS_ROUTER_EMBEDDINGS_INPUT_MODE=array`.

**Response validation pattern** (lines 219-259):
```python
if code != 200:
    detail = _scrub(raw_text, [token, expected_channel_name or ""])
    print(f"embeddings upstream: HTTP {code} {detail}", file=sys.stderr)
    if code in ACCEPTABLE_UPSTREAM_CODES:
        return 0 if accept_upstream_error else 1
    return 1

embedding_rows = data.get("data", [])
if not isinstance(embedding_rows, list) or not embedding_rows:
    print("embeddings error: missing data[0]", file=sys.stderr)
    return 1

dimension = assert_embedding_vector_shape(first.get("embedding"), expected_dim, model)
print(f"embeddings ok: model={model} type={display_type} dimension={dimension}")
```

If array smoke is added, validate all returned rows or at least row count + first vector dimension.

**Role-match analog:** `scripts/smoke-provider-consolidation.py`

**Base URL normalization** (lines 20-29):
```python
def _normalize_base_url(value: str) -> str:
    base_url = value.rstrip("/")
    if base_url.endswith("/v1"):
        return base_url[:-3]
    return base_url

BASE_URL = _normalize_base_url(os.environ.get("ATIUS_ROUTER_BASE_URL", "http://127.0.0.1:3000"))
PUBLIC_BASE_URL = _normalize_base_url(os.environ.get("ATIUS_ROUTER_PUBLIC_BASE_URL", "https://router.atius.com.br"))
```

Use this style if smoke needs to accept public/local bases consistently.

---

### `tests/test_clianything.py` (test, smoke utility helper regression)

**Analog:** `tests/test_clianything.py`

**Smoke helper test pattern** (lines 700-727):
```python
def test_smoke_embeddings_helpers_cover_payload_shape_and_redaction(self):
    payload = smoke_embeddings.build_embedding_payload(
        model="embo-01",
        input_text="hello",
        embedding_type="db",
    )
    self.assertEqual(payload["type"], "db")
    self.assertEqual(payload["input"], "hello")

    openai_payload = smoke_embeddings.build_embedding_payload(
        model="text-embedding-3-small",
        input_text="hello",
        openai_dimensions=1536,
    )
    self.assertNotIn("type", openai_payload)
    self.assertEqual(openai_payload["dimensions"], 1536)

    scrubbed = smoke_embeddings._scrub(
        "GroupId=123 Authorization=Bearer abc ATIUS_ROUTER_TOKEN=secret",
        ["secret", "abc"],
    )
    self.assertNotIn("secret", scrubbed)
    self.assertNotIn("abc", scrubbed)
    self.assertNotIn("GroupId", scrubbed)
```

Update this if `build_embedding_payload` supports array input or Phase 25 defaults. Keep redaction assertions.

## Shared Patterns

### Metadata-Only Governor Boundary

**Source:** `dto/embedding.go` lines 55-66 and `relay/embedding_handler.go` lines 31-35, 78-85
**Apply to:** `relay/embedding_handler.go`, `service/embeddinggovernor/governor.go`, tests

```go
publicModelName := embeddingReq.Model
inputStats := embeddingReq.GetInputStats()
// ...
InputCount:  inputStats.InputCount,
InputChars:  inputStats.InputChars,
```

Planner should route only count/chars/header/model/channel metadata into governor code.

### Header Priority Then Auto Inference

**Source:** `service/embeddinggovernor/governor.go` lines 435-450
**Apply to:** classifier helper and governor tests

```go
if workload == "batch" || workload == "bulk" {
	return true
}
if workload == "interactive" || workload == "realtime" {
	return false
}
if g.cfg.BatchInputCountThreshold > 0 && req.InputCount >= g.cfg.BatchInputCountThreshold {
	return true
}
```

Invalid/absent header must fall through. Explicit valid headers keep priority.

### Fail-Closed TEI Cap Option

**Source:** `relay/helper/valid_request.go` lines 97-115; current gap in `relay/channel/openai/adaptor.go` lines 359-360
**Apply to:** `relay/embedding_handler.go` or a narrow helper if planner chooses fail-closed enforcement

```go
if embeddingRequest.Input == nil {
	return nil, fmt.Errorf("input is empty")
}
```

Use this error-return style to reject governed local TEI requests with `InputCount > 4`, before upstream dispatch. Do not enforce globally across all embedding providers unless planner explicitly expands scope.

### Secret Hygiene In Smokes And Docs

**Source:** `scripts/smoke-embeddings.py` lines 38-47; docs lines 261-265
**Apply to:** smoke script, docs, tests

```python
scrubbed = scrubbed.replace("Authorization", "<redacted-auth>")
scrubbed = scrubbed.replace("x-api-key", "<redacted-auth>")
return _short_text(scrubbed)
```

No token or bearer value should be printed, stored, or copied into phase artifacts.

### Testify For Go Backend Tests

**Source:** `service/embeddinggovernor/governor_test.go` lines 11-14 and `relay/embedding_handler_test.go` lines 16-17
**Apply to:** all new Go backend tests in this phase

```go
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
```

Use deterministic table tests with explicit inputs and exact expected outputs.

## No Analog Found

| File / Pattern | Role | Data Flow | Reason |
|----------------|------|-----------|--------|
| Transparent TEI sub-batch request splitting + OpenAI embedding response recomposition | relay helper | request-response, batch, transform | No close existing analog was found. `relay/channel/minimax/embedding.go` maps provider vectors to OpenAI response shape, but no existing relay path splits one OpenAI-compatible embedding request into bounded TEI sub-requests and merges usage/order. Prefer fail-closed cap unless transparent success is explicitly required. |

## Metadata

**Analog search scope:** `dto/`, `relay/`, `service/`, `scripts/`, `docs/`, `tests/`
**Files scanned:** 350 total files in searched dirs; 48 embedding/governor/valid_request/smoke/clianything/adaptor candidates
**Graphify:** status fresh at commit `93a27e3`; task-specific query returned no direct nodes, so `rg` and focused reads were used
**GBrain/memory context:** prior governed `embedding-gte-v1` history confirmed single public alias, Go-native governor route, and internal batch classification
**Pattern extraction date:** 2026-07-05
