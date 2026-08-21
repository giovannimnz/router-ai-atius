package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAtiusLocalEmbeddingsChannelSupportsOnlyConfiguredPaths(t *testing.T) {
	channel := &model.Channel{Type: constant.ChannelTypeAtiusLocalEmbeddings}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{
				{IncomingPath: "/v1/embeddings", UpstreamPath: "http://embeddings.internal/v1/embeddings"},
				{IncomingPath: "/v1/rerank", UpstreamPath: "http://reranker.internal/rerank"},
			},
		},
	})

	assert.True(t, channelSupportsRequestPath(channel, "/v1/embeddings"))
	assert.True(t, channelSupportsRequestPath(channel, "/v1/rerank"))
	assert.False(t, channelSupportsRequestPath(channel, "/v1/chat/completions"))
}

func TestGetModelRequestNormalizesLegacyRerankerAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := `{"model":"reranker-gte-multilingual-v1","query":"ola","documents":["a","b"]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/rerank", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	modelRequest, shouldSelectChannel, err := getModelRequest(c)

	assert.NoError(t, err)
	assert.True(t, shouldSelectChannel)
	assert.Equal(t, "reranker-gte-v1", modelRequest.Model)
}
