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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gin-gonic/gin"
)

func TestEmbeddingHelperPassesGovernorRequestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	first := strings.Repeat("a", 24)
	second := strings.Repeat("b", 40)

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
	info := &relaycommon.RelayInfo{
		OriginModelName: "embedding-gte-v1",
		Request:         request,
	}

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

	err := EmbeddingHelper(c, info)

	require.NotNil(t, err)
	assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	assert.Equal(t, constant.ChannelTypeOpenAI, info.ChannelType)
	assert.Equal(t, "3", recorder.Header().Get("Retry-After"))
	assert.Equal(t, "embedding_governor_queue_full", string(err.GetErrorCode()))
	assert.Equal(t, "embedding-gte-v1", captured.Model)
	assert.Equal(t, 77, captured.ChannelID)
	assert.Equal(t, "Local TEI - GTE Embeddings", captured.ChannelName)
	assert.Equal(t, "batch", captured.Workload)
	assert.Equal(t, 2, captured.InputCount)
	assert.Equal(t, len(first)+len(second), captured.InputChars)
	assert.NotContains(t, err.Error(), first)
	assert.NotContains(t, err.Error(), second)
}
