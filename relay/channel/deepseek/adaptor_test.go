package deepseek

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLResponsesNormalizesDeepSeekBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "provider root", baseURL: "https://api.deepseek.com"},
		{name: "provider root trailing slash", baseURL: "https://api.deepseek.com/"},
		{name: "v1 base", baseURL: "https://api.deepseek.com/v1"},
		{name: "v1 base trailing slash", baseURL: "https://api.deepseek.com/v1/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := &relaycommon.RelayInfo{
				RelayMode: relayconstant.RelayModeResponses,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelBaseUrl: tt.baseURL,
				},
			}

			got, err := (&Adaptor{}).GetRequestURL(info)
			require.NoError(t, err)
			assert.Equal(t, "https://api.deepseek.com/responses", got)
		})
	}
}

func TestGetRequestURLNormalizesDeepSeekBaseURLForEveryProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseURL     string
		relayMode   int
		relayFormat types.RelayFormat
		want        string
	}{
		{
			name:      "chat provider root",
			baseURL:   "https://api.deepseek.com",
			relayMode: relayconstant.RelayModeChatCompletions,
			want:      "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name:      "chat v1 base",
			baseURL:   "https://api.deepseek.com/v1/",
			relayMode: relayconstant.RelayModeChatCompletions,
			want:      "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name:        "anthropic v1 base",
			baseURL:     "https://api.deepseek.com/v1",
			relayMode:   relayconstant.RelayModeChatCompletions,
			relayFormat: types.RelayFormatClaude,
			want:        "https://api.deepseek.com/anthropic/v1/messages",
		},
		{
			name:      "fim v1 base",
			baseURL:   "https://api.deepseek.com/v1",
			relayMode: relayconstant.RelayModeCompletions,
			want:      "https://api.deepseek.com/beta/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := &relaycommon.RelayInfo{
				RelayMode:   tt.relayMode,
				RelayFormat: tt.relayFormat,
				ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: tt.baseURL},
			}
			got, err := (&Adaptor{}).GetRequestURL(info)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertOpenAIResponsesRequestPreservesDTOAndAppliesMaxAlias(t *testing.T) {
	t.Parallel()

	stream := false
	maxOutputTokens := uint(0)
	temperature := float64(0)
	request := dto.OpenAIResponsesRequest{
		Model:           "router-model",
		Input:           []byte(`"Reply only OK"`),
		Instructions:    []byte(`"Be concise"`),
		MaxOutputTokens: &maxOutputTokens,
		Metadata:        []byte(`{"trace":"deepseek"}`),
		Reasoning:       &dto.Reasoning{Effort: "high", Summary: "detailed"},
		Stream:          &stream,
		Temperature:     &temperature,
		Tools:           []byte(`[{"type":"function","name":"lookup"}]`),
	}
	originalReasoning := request.Reasoning
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "deepseek-v4-pro-max",
		},
	}

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "deepseek-v4-pro", got.Model)
	require.NotNil(t, got.Reasoning)
	assert.Equal(t, "max", got.Reasoning.Effort)
	assert.Equal(t, "detailed", got.Reasoning.Summary)
	assert.NotSame(t, originalReasoning, got.Reasoning)
	assert.Equal(t, "high", originalReasoning.Effort)
	assert.Equal(t, request.Input, got.Input)
	assert.Equal(t, request.Instructions, got.Instructions)
	assert.Equal(t, request.MaxOutputTokens, got.MaxOutputTokens)
	assert.Equal(t, request.Metadata, got.Metadata)
	assert.Equal(t, request.Stream, got.Stream)
	assert.Equal(t, request.Temperature, got.Temperature)
	assert.Equal(t, request.Tools, got.Tools)
	assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
	assert.Equal(t, "max", info.ReasoningEffort)
}

func TestConvertOpenAIResponsesRequestAppliesNoneAlias(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "deepseek-v4-flash-none",
		},
	}
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, dto.OpenAIResponsesRequest{
		Model: "router-model",
		Input: []byte(`"hello"`),
	})
	require.NoError(t, err)
	got, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)

	assert.Equal(t, "deepseek-v4-flash", got.Model)
	require.NotNil(t, got.Reasoning)
	assert.Equal(t, "none", got.Reasoning.Effort)
	assert.Equal(t, "deepseek-v4-flash", info.UpstreamModelName)
	assert.Equal(t, "none", info.ReasoningEffort)
}

func TestDoResponseRoutesNativeResponsesNonStream(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	body := `{"id":"resp_deepseek","object":"response","model":"deepseek-v4-pro","status":"completed","output":[],"usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatClaude,
	}

	usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.IsType(t, &dto.Usage{}, usage)
	assert.Equal(t, 3, usage.(*dto.Usage).TotalTokens)
	assert.JSONEq(t, body, recorder.Body.String())
}

func TestDoResponseRoutesNativeResponsesStreamWithoutDoneSentinel(t *testing.T) {
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 300
	t.Cleanup(func() { constant.StreamingTimeout = oldStreamingTimeout })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	body := strings.Join([]string{
		`data: {"type":"response.output_text.delta","delta":"OK"}`,
		``,
		`data: {"type":"response.completed","response":{"id":"resp_deepseek","object":"response","model":"deepseek-v4-pro","status":"completed","output":[],"usage":{"input_tokens":2,"output_tokens":1,"total_tokens":3}}}`,
		``,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "deepseek-v4-pro"},
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatClaude,
		IsStream:    true,
	}

	usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.IsType(t, &dto.Usage{}, usage)
	assert.Equal(t, 3, usage.(*dto.Usage).TotalTokens)
	assert.Contains(t, recorder.Body.String(), `event: response.output_text.delta`)
	assert.Contains(t, recorder.Body.String(), `event: response.completed`)
	assert.NotContains(t, recorder.Body.String(), "[DONE]")
}
