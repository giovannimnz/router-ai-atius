package relay

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
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"
)

func TestEmbeddingHelperPassesGovernorRequestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	first := strings.Repeat("a", 24)
	second := strings.Repeat("b", 40)

	originalAcquire := acquireEmbeddingGovernor
	t.Cleanup(func() {
		acquireEmbeddingGovernor = originalAcquire
	})

	tests := []struct {
		name           string
		workloadHeader string
		input          any
		wantWorkload   string
		wantCount      int
		wantChars      int
	}{
		{
			name:           "header batch array keeps explicit workload metadata",
			workloadHeader: "batch",
			input:          []string{first, second},
			wantWorkload:   "batch",
			wantCount:      2,
			wantChars:      len(first) + len(second),
		},
		{
			name:         "no header single string keeps empty workload metadata",
			input:        first,
			wantWorkload: "",
			wantCount:    1,
			wantChars:    len(first),
		},
		{
			name:         "no header array keeps empty workload metadata",
			input:        []string{first, second},
			wantWorkload: "",
			wantCount:    2,
			wantChars:    len(first) + len(second),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
			if tc.workloadHeader != "" {
				c.Request.Header.Set("X-Embedding-Workload", tc.workloadHeader)
			}
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
			common.SetContextKey(c, constant.ContextKeyChannelId, 77)
			common.SetContextKey(c, constant.ContextKeyChannelName, "Local TEI - GTE Embeddings")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "embedding-gte-v1")

			info := &relaycommon.RelayInfo{
				OriginModelName: "embedding-gte-v1",
				Request: &dto.EmbeddingRequest{
					Model: "embedding-gte-v1",
					Input: tc.input,
				},
			}

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

			err := EmbeddingHelper(c, info)

			require.NotNil(t, err)
			assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
			assert.Equal(t, constant.ChannelTypeOpenAI, info.ChannelType)
			assert.Equal(t, "3", recorder.Header().Get("Retry-After"))
			assert.Equal(t, "embedding_governor_queue_full", string(err.GetErrorCode()))
			assert.Equal(t, "embedding-gte-v1", captured.Model)
			assert.Equal(t, 77, captured.ChannelID)
			assert.Equal(t, "Local TEI - GTE Embeddings", captured.ChannelName)
			assert.Equal(t, tc.wantWorkload, captured.Workload)
			assert.Equal(t, tc.wantCount, captured.InputCount)
			assert.Equal(t, tc.wantChars, captured.InputChars)
			assert.NotContains(t, err.Error(), first)
			assert.NotContains(t, err.Error(), second)
		})
	}
}

func TestEmbeddingHelperRejectsGovernedInputAboveTEICap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	makeArray := func(n int) []string {
		items := make([]string, 0, n)
		for i := 0; i < n; i++ {
			items = append(items, strings.Repeat(string(rune('a'+i)), 4))
		}
		return items
	}

	tests := []struct {
		name           string
		model          string
		workloadHeader string
		input          any
		wantStatus     int
		wantCode       string
		wantAcquire    bool
	}{
		{
			name:        "governed request above cap fails closed",
			model:       "embedding-gte-v1",
			input:       makeArray(5),
			wantStatus:  http.StatusBadRequest,
			wantCode:    string(types.ErrorCodeInvalidRequest),
			wantAcquire: false,
		},
		{
			name:           "interactive header cannot bypass governed cap",
			model:          "embedding-gte-v1",
			workloadHeader: "interactive",
			input:          makeArray(5),
			wantStatus:     http.StatusBadRequest,
			wantCode:       string(types.ErrorCodeInvalidRequest),
			wantAcquire:    false,
		},
		{
			name:        "unknown model keeps existing no-op governor behavior",
			model:       "text-embedding-3-small",
			input:       makeArray(5),
			wantStatus:  http.StatusTooManyRequests,
			wantCode:    "embedding_governor_queue_full",
			wantAcquire: true,
		},
	}

	originalAcquire := acquireEmbeddingGovernor
	t.Cleanup(func() {
		acquireEmbeddingGovernor = originalAcquire
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
			if tc.workloadHeader != "" {
				c.Request.Header.Set("X-Embedding-Workload", tc.workloadHeader)
			}
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
			common.SetContextKey(c, constant.ContextKeyChannelId, 77)
			common.SetContextKey(c, constant.ContextKeyChannelName, "Local TEI - GTE Embeddings")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, tc.model)

			info := &relaycommon.RelayInfo{
				OriginModelName: tc.model,
				Request: &dto.EmbeddingRequest{
					Model: tc.model,
					Input: tc.input,
				},
			}

			acquireCalled := false
			acquireEmbeddingGovernor = func(ctx context.Context, req embeddinggovernor.Request) (*embeddinggovernor.Lease, *embeddinggovernor.Reject) {
				acquireCalled = true
				return nil, &embeddinggovernor.Reject{
					StatusCode: http.StatusTooManyRequests,
					Code:       "embedding_governor_queue_full",
					Message:    "synthetic governor reject",
					RetryAfter: 3 * time.Second,
				}
			}

			err := EmbeddingHelper(c, info)

			require.NotNil(t, err)
			assert.Equal(t, tc.wantStatus, err.StatusCode)
			assert.Equal(t, tc.wantCode, string(err.GetErrorCode()))
			assert.Equal(t, tc.wantAcquire, acquireCalled)
			if tc.model == "embedding-gte-v1" {
				assert.Contains(t, err.Error(), "at most 4 input items")
			}
			for _, item := range tc.input.([]string) {
				assert.NotContains(t, err.Error(), item)
			}
		})
	}
}
