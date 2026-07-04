package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesStreamToChatHandlerAggregatesSSE(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"model":"gpt-5.5","created_at":1710000000}}`,
		``,
		`data: {"type":"response.output_text.delta","delta":"hello "}`,
		``,
		`data: {"type":"response.output_text.delta","delta":"world"}`,
		``,
		`data: {"type":"response.completed","response":{"model":"gpt-5.5","created_at":1710000001,"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(common.RequestIdKey, "test-request")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.5-1m",
		RelayFormat:     types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.5",
		},
	}

	usage, apiErr := OaiResponsesStreamToChatHandler(c, info, resp)
	require.Nil(t, apiErr)
	require.Equal(t, 3, usage.PromptTokens)
	require.Equal(t, 2, usage.CompletionTokens)
	require.Equal(t, 5, usage.TotalTokens)

	var chatResp dto.OpenAITextResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &chatResp))
	require.Equal(t, "chat.completion", chatResp.Object)
	require.Equal(t, "gpt-5.5-1m", chatResp.Model)
	require.Len(t, chatResp.Choices, 1)
	require.Equal(t, "assistant", chatResp.Choices[0].Message.Role)
	require.Equal(t, "hello world", chatResp.Choices[0].Message.StringContent())
	require.Equal(t, "stop", chatResp.Choices[0].FinishReason)
	require.Equal(t, 5, chatResp.Usage.TotalTokens)
}

func TestOaiResponsesStreamToChatHandlerPropagatesErrorEvent(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	body := strings.Join([]string{
		`data: {"type":"error","error":{"type":"invalid_request_error","code":"context_length_exceeded","message":"too long","param":"input"}}`,
		``,
		`data: {"type":"response.failed","response":{"status":"failed"},"error":{"code":"context_length_exceeded","message":"too long"}}`,
		``,
	}, "\n")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(common.RequestIdKey, "test-request")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.4-1m",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4",
		},
	}

	usage, apiErr := OaiResponsesStreamToChatHandler(c, info, resp)
	require.Nil(t, usage)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Contains(t, apiErr.Error(), "too long")
}
