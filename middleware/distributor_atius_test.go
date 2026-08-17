package middleware

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
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
