package relay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
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

func TestEmbeddingHelperSplitsGovernedInputAboveTEICap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	makeArray := func(n int) []string {
		items := make([]string, 0, n)
		for i := 0; i < n; i++ {
			items = append(items, strings.Repeat(string(rune('a'+i)), 4))
		}
		return items
	}

	originalChunkExecutor := executeEmbeddingChunkRequest
	originalPostConsume := postEmbeddingConsumeQuota
	t.Cleanup(func() {
		executeEmbeddingChunkRequest = originalChunkExecutor
		postEmbeddingConsumeQuota = originalPostConsume
	})
	postEmbeddingConsumeQuota = func(*gin.Context, *relaycommon.RelayInfo, *dto.Usage, []string) {}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(c, constant.ContextKeyChannelId, 77)
	common.SetContextKey(c, constant.ContextKeyChannelName, "Local TEI - GTE Embeddings")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "embedding-gte-v1")

	inputs := makeArray(5)
	info := &relaycommon.RelayInfo{
		OriginModelName: "embedding-gte-v1",
		Request: &dto.EmbeddingRequest{
			Model: "embedding-gte-v1",
			Input: inputs,
		},
	}

	var capturedChunks [][]string
	executeEmbeddingChunkRequest = func(c *gin.Context, info *relaycommon.RelayInfo, adaptor relaychannel.Adaptor, request *dto.EmbeddingRequest, publicModelName string) (*dto.OpenAIEmbeddingResponse, *dto.Usage, *types.NewAPIError) {
		chunkInputs := request.ParseInput()
		capturedChunks = append(capturedChunks, append([]string(nil), chunkInputs...))
		data := make([]dto.OpenAIEmbeddingResponseItem, 0, len(chunkInputs))
		for idx := range chunkInputs {
			data = append(data, dto.OpenAIEmbeddingResponseItem{
				Object:    "embedding",
				Index:     idx,
				Embedding: []float64{float64(len(capturedChunks)), float64(idx)},
			})
		}
		response := &dto.OpenAIEmbeddingResponse{
			Object: "list",
			Data:   data,
			Model:  request.Model,
			Usage: dto.Usage{
				PromptTokens: len(chunkInputs),
				TotalTokens:  len(chunkInputs),
				InputTokens:  len(chunkInputs),
			},
		}
		return response, &response.Usage, nil
	}

	err := EmbeddingHelper(c, info)

	if err != nil {
		require.FailNowf(t, "unexpected embedding relay error", "message=%q code=%q status=%d chunks=%d", err.Error(), err.GetErrorCode(), err.StatusCode, len(capturedChunks))
	}
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, [][]string{inputs[:4], inputs[4:]}, capturedChunks)

	var response dto.OpenAIEmbeddingResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Len(t, response.Data, 5)
	for idx, item := range response.Data {
		assert.Equal(t, idx, item.Index)
	}
	assert.Equal(t, 5, response.Usage.PromptTokens)
	assert.Equal(t, 5, response.Usage.TotalTokens)
	assert.Equal(t, 5, response.Usage.InputTokens)
}

func TestChunkEmbeddingInputsSplitsAtGovernedCap(t *testing.T) {
	inputs := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}

	chunks := chunkEmbeddingInputs(inputs, maxGovernedTEIInputCount)

	require.Equal(t, [][]string{
		{"a", "b", "c", "d"},
		{"e", "f", "g", "h"},
		{"i"},
	}, chunks)
}

func TestMergeOpenAIEmbeddingResponsesReindexesAndSumsUsage(t *testing.T) {
	merged := mergeOpenAIEmbeddingResponses([]*dto.OpenAIEmbeddingResponse{
		{
			Object: "list",
			Model:  "embedding-gte-v1",
			Data: []dto.OpenAIEmbeddingResponseItem{
				{Object: "embedding", Index: 0, Embedding: []float64{1}},
				{Object: "embedding", Index: 1, Embedding: []float64{2}},
			},
			Usage: dto.Usage{PromptTokens: 2, TotalTokens: 2, InputTokens: 2},
		},
		{
			Object: "list",
			Model:  "embedding-gte-v1",
			Data: []dto.OpenAIEmbeddingResponseItem{
				{Object: "embedding", Index: 0, Embedding: []float64{3}},
			},
			Usage: dto.Usage{PromptTokens: 1, TotalTokens: 1, InputTokens: 1},
		},
	}, "fallback-model")

	require.Equal(t, "embedding-gte-v1", merged.Model)
	require.Len(t, merged.Data, 3)
	assert.Equal(t, 0, merged.Data[0].Index)
	assert.Equal(t, 1, merged.Data[1].Index)
	assert.Equal(t, 2, merged.Data[2].Index)
	assert.Equal(t, 3, merged.Usage.PromptTokens)
	assert.Equal(t, 3, merged.Usage.TotalTokens)
	assert.Equal(t, 3, merged.Usage.InputTokens)
}

func TestRerankHelperPassesGovernorRequestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	query := strings.Repeat("q", 12)
	first := strings.Repeat("a", 24)
	second := strings.Repeat("b", 40)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", nil)
	c.Request.Header.Set("X-Rerank-Workload", "batch")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAdvancedCustom)
	common.SetContextKey(c, constant.ContextKeyChannelId, 78)
	common.SetContextKey(c, constant.ContextKeyChannelName, "Local TEI - GTE Reranker")
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "http://127.0.0.1:31216")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "reranker-gte-v1")
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{{
			IncomingPath: "/v1/rerank",
			UpstreamPath: "/rerank",
			Converter:    dto.AdvancedCustomConverterJinaRerankToTEINative,
			Auth:         &dto.AdvancedCustomRouteAuth{Type: dto.AdvancedCustomAuthTypeNone},
		}}},
	})

	request := &dto.RerankRequest{
		Model:     "reranker-gte-v1",
		Query:     query,
		Documents: []any{first, second},
	}
	info := relaycommon.GenRelayInfoRerank(c, request)

	originalAcquire := acquireRerankGovernor
	t.Cleanup(func() {
		acquireRerankGovernor = originalAcquire
	})

	var captured embeddinggovernor.Request
	acquireRerankGovernor = func(ctx context.Context, req embeddinggovernor.Request) (*embeddinggovernor.Lease, *embeddinggovernor.Reject) {
		captured = req
		return nil, &embeddinggovernor.Reject{
			StatusCode: http.StatusTooManyRequests,
			Code:       "embedding_governor_queue_full",
			Message:    "synthetic governor reject",
			RetryAfter: 3 * time.Second,
		}
	}

	err := RerankHelper(c, info)

	require.NotNil(t, err)
	assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	assert.Equal(t, "3", recorder.Header().Get("Retry-After"))
	assert.Equal(t, "embedding_governor_queue_full", string(err.GetErrorCode()))
	assert.Equal(t, "reranker-gte-v1", captured.Model)
	assert.Equal(t, 78, captured.ChannelID)
	assert.Equal(t, "Local TEI - GTE Reranker", captured.ChannelName)
	assert.Equal(t, "batch", captured.Workload)
	assert.Equal(t, 2, captured.InputCount)
	assert.Equal(t, len(query)+len(first)+len(second), captured.InputChars)
	assert.NotContains(t, err.Error(), query)
	assert.NotContains(t, err.Error(), first)
	assert.NotContains(t, err.Error(), second)
}

func TestRerankHelperRejectsGovernedDocumentCountAboveTEICap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	documents := make([]any, maxGovernedTEIRerankDocuments+1)
	for i := range documents {
		documents[i] = strings.Repeat("x", i+1)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", nil)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAdvancedCustom)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "reranker-gte-v1")

	request := &dto.RerankRequest{
		Model:     "reranker-gte-v1",
		Query:     "consulta",
		Documents: documents,
	}
	info := relaycommon.GenRelayInfoRerank(c, request)

	originalAcquire := acquireRerankGovernor
	t.Cleanup(func() {
		acquireRerankGovernor = originalAcquire
	})
	acquireCalled := false
	acquireRerankGovernor = func(ctx context.Context, req embeddinggovernor.Request) (*embeddinggovernor.Lease, *embeddinggovernor.Reject) {
		acquireCalled = true
		return nil, nil
	}

	err := RerankHelper(c, info)

	require.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, string(types.ErrorCodeInvalidRequest), string(err.GetErrorCode()))
	assert.Contains(t, err.Error(), "at most 20 documents")
	assert.False(t, acquireCalled)
	for _, document := range documents {
		assert.NotContains(t, err.Error(), document)
	}
}

func TestRerankHelperPassesGovernorRequestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	query := strings.Repeat("q", 12)
	first := strings.Repeat("a", 24)
	second := strings.Repeat("b", 40)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", nil)
	c.Request.Header.Set("X-Rerank-Workload", "batch")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAdvancedCustom)
	common.SetContextKey(c, constant.ContextKeyChannelId, 78)
	common.SetContextKey(c, constant.ContextKeyChannelName, "Local TEI - GTE Reranker")
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, "http://127.0.0.1:31216")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "reranker-gte-multilingual-v1")
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{{
			IncomingPath: "/v1/rerank",
			UpstreamPath: "/rerank",
			Converter:    dto.AdvancedCustomConverterJinaRerankToTEINative,
			Auth:         &dto.AdvancedCustomRouteAuth{Type: dto.AdvancedCustomAuthTypeNone},
		}}},
	})

	request := &dto.RerankRequest{
		Model:     "reranker-gte-multilingual-v1",
		Query:     query,
		Documents: []any{first, second},
	}
	info := relaycommon.GenRelayInfoRerank(c, request)

	originalAcquire := acquireRerankGovernor
	t.Cleanup(func() {
		acquireRerankGovernor = originalAcquire
	})

	var captured embeddinggovernor.Request
	acquireRerankGovernor = func(ctx context.Context, req embeddinggovernor.Request) (*embeddinggovernor.Lease, *embeddinggovernor.Reject) {
		captured = req
		return nil, &embeddinggovernor.Reject{
			StatusCode: http.StatusTooManyRequests,
			Code:       "embedding_governor_queue_full",
			Message:    "synthetic governor reject",
			RetryAfter: 3 * time.Second,
		}
	}

	err := RerankHelper(c, info)

	require.NotNil(t, err)
	assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	assert.Equal(t, "3", recorder.Header().Get("Retry-After"))
	assert.Equal(t, "embedding_governor_queue_full", string(err.GetErrorCode()))
	assert.Equal(t, "reranker-gte-multilingual-v1", captured.Model)
	assert.Equal(t, 78, captured.ChannelID)
	assert.Equal(t, "Local TEI - GTE Reranker", captured.ChannelName)
	assert.Equal(t, "batch", captured.Workload)
	assert.Equal(t, 2, captured.InputCount)
	assert.Equal(t, len(query)+len(first)+len(second), captured.InputChars)
	assert.NotContains(t, err.Error(), query)
	assert.NotContains(t, err.Error(), first)
	assert.NotContains(t, err.Error(), second)
}

func TestRerankHelperRejectsGovernedDocumentCountAboveTEICap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	documents := make([]any, maxGovernedTEIRerankDocuments+1)
	for i := range documents {
		documents[i] = strings.Repeat("x", i+1)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", nil)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAdvancedCustom)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "reranker-gte-multilingual-v1")

	request := &dto.RerankRequest{
		Model:     "reranker-gte-multilingual-v1",
		Query:     "consulta",
		Documents: documents,
	}
	info := relaycommon.GenRelayInfoRerank(c, request)

	originalAcquire := acquireRerankGovernor
	t.Cleanup(func() {
		acquireRerankGovernor = originalAcquire
	})
	acquireCalled := false
	acquireRerankGovernor = func(ctx context.Context, req embeddinggovernor.Request) (*embeddinggovernor.Lease, *embeddinggovernor.Reject) {
		acquireCalled = true
		return nil, nil
	}

	err := RerankHelper(c, info)

	require.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode)
	assert.Equal(t, string(types.ErrorCodeInvalidRequest), string(err.GetErrorCode()))
	assert.Contains(t, err.Error(), "at most 20 documents")
	assert.False(t, acquireCalled)
	for _, document := range documents {
		assert.NotContains(t, err.Error(), document)
	}
}
