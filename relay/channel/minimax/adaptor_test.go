package minimax

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLForImageGeneration(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.minimax.chat",
		},
	}

	got, err := GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL returned error: %v", err)
	}

	want := "https://api.minimax.chat/v1/image_generation"
	if got != want {
		t.Fatalf("GetRequestURL() = %q, want %q", got, want)
	}
}

func TestGetRequestURLForEmbeddings(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeEmbeddings,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.minimax.io",
		},
	}

	got, err := GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimax.io/v1/embeddings", got)
}

func TestGetRequestURLForEmbeddingsNormalizesBaseURLWithV1(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeEmbeddings,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.minimax.io/v1",
		},
	}

	got, err := GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimax.io/v1/embeddings", got)
}

func TestGetRequestURLForClaudeFormat(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.minimax.io",
		},
	}

	got, err := GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimax.io/anthropic/v1/messages", got)
}

func TestGetRequestURLForClaudeFormatNormalizesBaseURLWithV1(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.minimax.io/v1",
		},
	}

	got, err := GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimax.io/anthropic/v1/messages", got)
}

func TestConvertEmbeddingRequestUsesMiniMaxNativeShape(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	request := dto.EmbeddingRequest{
		Model: "embo-01",
		Input: []string{"hello", "world"},
		Type:  "db",
	}

	got, err := adaptor.ConvertEmbeddingRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)

	payload, ok := got.(embeddingRequest)
	require.True(t, ok)
	assert.Equal(t, "embo-01", payload.Model)
	assert.Equal(t, []string{"hello", "world"}, payload.Texts)
	assert.Equal(t, "db", payload.Type)
}

func TestConvertEmbeddingRequestDefaultsInvalidMiniMaxTypeToQuery(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	request := dto.EmbeddingRequest{
		Model: "embo-01",
		Input: "hello",
		Type:  "invalid",
	}

	got, err := adaptor.ConvertEmbeddingRequest(nil, &relaycommon.RelayInfo{}, request)
	require.NoError(t, err)

	payload, ok := got.(embeddingRequest)
	require.True(t, ok)
	assert.Equal(t, []string{"hello"}, payload.Texts)
	assert.Equal(t, "query", payload.Type)
}

func TestConvertImageRequest(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		OriginModelName: "image-01",
	}
	request := dto.ImageRequest{
		Model:          "image-01",
		Prompt:         "a red fox in snowfall",
		Size:           "1536x1024",
		ResponseFormat: "url",
		N:              uintPtr(2),
	}

	got, err := adaptor.ConvertImageRequest(gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()), info, request)
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}

	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if payload["model"] != "image-01" {
		t.Fatalf("model = %#v, want %q", payload["model"], "image-01")
	}
	if payload["prompt"] != request.Prompt {
		t.Fatalf("prompt = %#v, want %q", payload["prompt"], request.Prompt)
	}
	if payload["n"] != float64(2) {
		t.Fatalf("n = %#v, want 2", payload["n"])
	}
	if payload["aspect_ratio"] != "3:2" {
		t.Fatalf("aspect_ratio = %#v, want %q", payload["aspect_ratio"], "3:2")
	}
	if payload["response_format"] != "url" {
		t.Fatalf("response_format = %#v, want %q", payload["response_format"], "url")
	}
}

func TestDoResponseForEmbedding(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeEmbeddings,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "embo-01",
		},
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       ioNopCloser(`{"vectors":[[0.1,0.2],[0.3,0.4]],"total_tokens":7,"base_resp":{"status_code":0}}`),
	}

	usage, err := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, err)
	require.NotNil(t, usage)

	body := recorder.Body.String()
	assert.Contains(t, body, `"object":"list"`)
	assert.Contains(t, body, `"model":"embo-01"`)
	assert.Contains(t, body, `"embedding":[0.1,0.2]`)
	assert.Contains(t, body, `"prompt_tokens":7`)
	assert.Equal(t, "minimax-embo-01", recorder.Header().Get("X-Embeddings-Adapter"))
}

func TestDoResponseForEmbeddingMapsMiniMaxRateLimitError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeEmbeddings}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       ioNopCloser(`{"base_resp":{"status_code":1002,"status_msg":"rate limit exceeded"}}`),
	}

	usage, err := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, usage)
	require.NotNil(t, err)
	assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestDoResponseForImageGeneration(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		StartTime: time.Unix(1700000000, 0),
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       httptest.NewRecorder().Result().Body,
	}
	resp.Body = ioNopCloser(`{"data":{"image_urls":["https://example.com/minimax.png"]}}`)

	adaptor := &Adaptor{}
	usage, err := adaptor.DoResponse(c, resp, info)
	if err != nil {
		t.Fatalf("DoResponse returned error: %v", err)
	}
	if usage == nil {
		t.Fatalf("DoResponse returned nil usage")
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"url":"https://example.com/minimax.png"`) {
		t.Fatalf("response body = %s, want OpenAI image response with image URL", body)
	}
	if strings.Contains(body, `"image_urls"`) {
		t.Fatalf("response body = %s, should not expose raw MiniMax image_urls payload", body)
	}
}

type nopReadCloser struct {
	*strings.Reader
}

func (n nopReadCloser) Close() error {
	return nil
}

func ioNopCloser(body string) nopReadCloser {
	return nopReadCloser{Reader: strings.NewReader(body)}
}

func uintPtr(v uint) *uint {
	return &v
}
