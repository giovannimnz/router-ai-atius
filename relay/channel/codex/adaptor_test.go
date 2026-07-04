package codex

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

const testCodexOAuthKey = `{"access_token":"access-token","account_id":"account-id","refresh_token":"refresh-token"}`
const testCodexPublicAPIKey = "sk-REDACTED-test"

func newCodexEmbeddingRelayInfo(baseURL string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:      relayconstant.RelayModeEmbeddings,
		RequestURLPath: "/v1/embeddings",
		RelayFormat:    types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeCodex,
			ChannelId:      9,
			ChannelBaseUrl: baseURL,
			ApiType:        constant.APITypeCodex,
			ApiKey:         testCodexOAuthKey,
		},
	}
}

func newCodexResponsesRelayInfo(baseURL string, apiKey string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode:      relayconstant.RelayModeResponses,
		RequestURLPath: "/v1/responses",
		RelayFormat:    types.RelayFormatOpenAIResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeCodex,
			ChannelId:      5,
			ChannelBaseUrl: baseURL,
			ApiType:        constant.APITypeCodex,
			ApiKey:         apiKey,
		},
	}
}

func TestCodexEmbeddingRequestUsesOpenAIEndpoint(t *testing.T) {
	adaptor := &Adaptor{}
	info := newCodexEmbeddingRelayInfo("https://api.openai.com/v1")
	request := dto.EmbeddingRequest{
		Model: "text-embedding-3-small",
		Input: "ping",
	}

	converted, err := adaptor.ConvertEmbeddingRequest(nil, info, request)
	require.NoError(t, err)
	assert.Equal(t, request, converted)

	requestURL, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1/embeddings", requestURL)
}

func TestCodexEmbeddingRequestFallsBackFromChatGPTBaseToOpenAIEndpoint(t *testing.T) {
	adaptor := &Adaptor{}
	info := newCodexEmbeddingRelayInfo("https://chatgpt.com")

	requestURL, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1/embeddings", requestURL)
}

func TestCodexEmbeddingHeaderUsesOAuthAccessTokenWithoutChatGPTHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &Adaptor{}
	info := newCodexEmbeddingRelayInfo("https://api.openai.com/v1")
	header := http.Header{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)

	err := adaptor.SetupRequestHeader(c, &header, info)
	require.NoError(t, err)

	assert.Equal(t, "Bearer access-token", header.Get("Authorization"))
	assert.Equal(t, "application/json", header.Get("Content-Type"))
	assert.Equal(t, "application/json", header.Get("Accept"))
	assert.Empty(t, header.Get("chatgpt-account-id"))
	assert.Empty(t, header.Get("OpenAI-Beta"))
	assert.Empty(t, header.Get("originator"))
}

func TestCodexEmbeddingDoResponseDelegatesToOpenAIHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &Adaptor{}
	info := newCodexEmbeddingRelayInfo("https://api.openai.com/v1")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"object":"list",
			"data":[{"object":"embedding","index":0,"embedding":[0.1,0.2]}],
			"model":"text-embedding-3-small",
			"usage":{"prompt_tokens":3,"total_tokens":3}
		}`)),
	}

	usage, apiErr := adaptor.DoResponse(c, resp, info)
	require.Nil(t, apiErr)

	gotUsage := usage.(*dto.Usage)
	assert.Equal(t, 3, gotUsage.PromptTokens)
	assert.Equal(t, 3, gotUsage.TotalTokens)
	assert.Contains(t, recorder.Body.String(), `"model":"text-embedding-3-small"`)
}

func TestCodexResponsesUsesChatGPTBackendByDefault(t *testing.T) {
	adaptor := &Adaptor{}
	info := newCodexResponsesRelayInfo("https://chatgpt.com", testCodexOAuthKey)

	requestURL, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://chatgpt.com/backend-api/codex/responses", requestURL)
}

func TestCodexResponsesUsesPublicOpenAIEndpointWhenBaseURLIsOpenAI(t *testing.T) {
	adaptor := &Adaptor{}

	for _, baseURL := range []string{"https://api.openai.com", "https://api.openai.com/v1"} {
		t.Run(baseURL, func(t *testing.T) {
			info := newCodexResponsesRelayInfo(baseURL, testCodexPublicAPIKey)

			requestURL, err := adaptor.GetRequestURL(info)
			require.NoError(t, err)
			assert.Equal(t, "https://api.openai.com/v1/responses", requestURL)
		})
	}
}

func TestCodexPublicOpenAIResponsesHeadersUseAPIKeyWithoutChatGPTHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &Adaptor{}
	info := newCodexResponsesRelayInfo("https://api.openai.com/v1", testCodexPublicAPIKey)
	header := http.Header{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	err := adaptor.SetupRequestHeader(c, &header, info)
	require.NoError(t, err)

	assert.Equal(t, "Bearer "+testCodexPublicAPIKey, header.Get("Authorization"))
	assert.Equal(t, "application/json", header.Get("Content-Type"))
	assert.Equal(t, "application/json", header.Get("Accept"))
	assert.Empty(t, header.Get("chatgpt-account-id"))
	assert.Empty(t, header.Get("OpenAI-Beta"))
	assert.Empty(t, header.Get("originator"))
}

func TestCodexPublicOpenAIResponsesRejectsOAuthJSONKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adaptor := &Adaptor{}
	info := newCodexResponsesRelayInfo("https://api.openai.com/v1", testCodexOAuthKey)
	header := http.Header{}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	err := adaptor.SetupRequestHeader(c, &header, info)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestCodexPublicOpenAIResponsesPreservesMaxOutputTokens(t *testing.T) {
	adaptor := &Adaptor{}
	info := newCodexResponsesRelayInfo("https://api.openai.com/v1", testCodexPublicAPIKey)
	maxOutputTokens := uint(128000)
	temperature := 0.2
	request := dto.OpenAIResponsesRequest{
		Model:           "gpt-5.5",
		MaxOutputTokens: &maxOutputTokens,
		Temperature:     &temperature,
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)

	convertedRequest := converted.(dto.OpenAIResponsesRequest)
	require.NotNil(t, convertedRequest.MaxOutputTokens)
	assert.Equal(t, maxOutputTokens, *convertedRequest.MaxOutputTokens)
	require.NotNil(t, convertedRequest.Temperature)
	assert.Equal(t, temperature, *convertedRequest.Temperature)
}

func TestCodexChatGPTResponsesStripsPublicOpenAIOnlyParameters(t *testing.T) {
	adaptor := &Adaptor{}
	info := newCodexResponsesRelayInfo("https://chatgpt.com", testCodexOAuthKey)
	maxOutputTokens := uint(128000)
	temperature := 0.2
	request := dto.OpenAIResponsesRequest{
		Model:           "gpt-5.5",
		MaxOutputTokens: &maxOutputTokens,
		Temperature:     &temperature,
	}

	converted, err := adaptor.ConvertOpenAIResponsesRequest(nil, info, request)
	require.NoError(t, err)

	convertedRequest := converted.(dto.OpenAIResponsesRequest)
	assert.Nil(t, convertedRequest.MaxOutputTokens)
	assert.Nil(t, convertedRequest.Temperature)
}

func TestParseSharedCodexChannelID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      int
		wantError bool
	}{
		{name: "default", input: "shared:codex", want: defaultSharedCodexChannelID},
		{name: "explicit", input: " shared:codex:42 ", want: 42},
		{name: "invalid empty", input: "shared:codex:", wantError: true},
		{name: "invalid text", input: "shared:codex:abc", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSharedCodexChannelID(tt.input)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
