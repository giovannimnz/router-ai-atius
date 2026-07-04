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
		[]constant.EndpointType{constant.EndpointTypeOpenAI, constant.EndpointTypeAnthropic},
		GetEndpointTypesByChannelType(constant.ChannelTypeDeepSeek, "deepseek-v4-pro"),
	)
}

func TestGetEndpointTypesByChannelTypeCodexTextIsOpenAICompatible(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeOpenAI},
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
