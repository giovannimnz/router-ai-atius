package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedCodexFetchModels = []string{
	"gpt-5.6-sol",
	"gpt-5.6-terra",
	"gpt-5.6-luna",
	"gpt-5.5",
	"gpt-5.3-codex-spark",
}

var expectedCodexFallbackModels = append([]string(nil), expectedCodexFetchModels...)

func TestFetchCodexModelIDsReturnsCanonicalNativeModels(t *testing.T) {
	models := fetchCodexModelIDs()

	require.Equal(t, expectedCodexFetchModels, models)
	assert.NotContains(t, models, "text-embedding-3-small")
	assert.NotContains(t, models, "gpt-5")
	assert.NotContains(t, models, "gpt-5.4")
	assert.NotContains(t, models, "gpt-5.4-mini")
}

func TestFetchChannelUpstreamModelIDsUsesCanonicalCodexModels(t *testing.T) {
	channel := &model.Channel{
		Type:   constant.ChannelTypeCodex,
		Key:    "not-json-and-not-used",
		Models: "gpt-5.5,text-embedding-3-small,deepseek-v4-pro",
	}

	models, err := fetchChannelUpstreamModelIDs(channel)

	require.NoError(t, err)
	require.Equal(t, expectedCodexFallbackModels, models)
}

func TestFetchModelsPostUsesCanonicalCodexModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/channel/fetch_models",
		strings.NewReader(`{"type":57,"base_url":"","key":"not-json-and-not-used"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	FetchModels(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
		Message string   `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Empty(t, response.Message)
	require.Equal(t, expectedCodexFallbackModels, response.Data)
}

func TestFetchDynamicCodexModelIDsUsesAccountAwareDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/backend-api/codex/models", r.URL.Path)
		require.Equal(t, "acct-test", r.Header.Get("chatgpt-account-id"))
		require.Equal(t, "Bearer access-test", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"models":[{"slug":"gpt-5.5"},{"slug":"gpt-5.4"},{"slug":"gpt-5.4","visibility":"hide"},{"slug":"gpt-5.5"}]}`))
	}))
	defer server.Close()

	channel := &model.Channel{
		Type: constant.ChannelTypeCodex,
		Key:  `{"access_token":"access-test","account_id":"acct-test"}`,
	}
	channel.BaseURL = &server.URL

	models, err := fetchDynamicCodexModelIDs(channel)

	require.NoError(t, err)
	require.Equal(t, []string{"gpt-5.5"}, models)
}
