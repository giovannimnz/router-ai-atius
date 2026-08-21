package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestGetEndpointTypesByChannelTypeMiniMaxTextIsMultiProtocol(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeOpenAI, constant.EndpointTypeAnthropic},
		GetEndpointTypesByChannelType(constant.ChannelTypeMiniMax, "MiniMax-M3"),
	)
}

func TestGetEndpointTypesByChannelTypeMiniMaxEmbeddingIsEmbeddingOnly(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeEmbeddings},
		GetEndpointTypesByChannelType(constant.ChannelTypeMiniMax, "embo-01"),
	)
}

func TestGetEndpointTypesByChannelTypeDeepSeekIsMultiProtocol(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeAnthropic,
		},
		GetEndpointTypesByChannelType(constant.ChannelTypeDeepSeek, "deepseek-v4-pro"),
	)
}

func TestGetEndpointTypesByChannelTypeCodexTextPrefersResponses(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{
			constant.EndpointTypeOpenAIResponse,
			constant.EndpointTypeOpenAI,
		},
		GetEndpointTypesByChannelType(constant.ChannelTypeCodex, "gpt-5.4"),
	)
}

func TestGetEndpointTypesByChannelTypeCodexEmbeddingIsEmbeddingOnly(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeEmbeddings},
		GetEndpointTypesByChannelType(constant.ChannelTypeCodex, "text-embedding-3-small"),
	)
}

func TestGetEndpointTypesByChannelTypeJinaUsesCanonicalReranker(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeReranker},
		GetEndpointTypesByChannelType(constant.ChannelTypeJina, "jina-reranker-v2-base-multilingual"),
	)
}

func TestGetDefaultEndpointInfoAcceptsLegacyRerankAlias(t *testing.T) {
	t.Parallel()

	canonical, canonicalOK := GetDefaultEndpointInfo(constant.EndpointTypeReranker)
	legacy, legacyOK := GetDefaultEndpointInfo(constant.EndpointTypeJinaRerank)

	assert.True(t, canonicalOK)
	assert.True(t, legacyOK)
	assert.Equal(t, EndpointInfo{Path: "/v1/rerank", Method: "POST"}, canonical)
	assert.Equal(t, canonical, legacy)
}
