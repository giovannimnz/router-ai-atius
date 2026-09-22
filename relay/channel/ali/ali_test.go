package ali_test

import (
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/ali"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliMetadata(t *testing.T) {
	adaptor := &ali.Adaptor{}
	assert.Equal(t, "ali", adaptor.GetChannelName())
	models := adaptor.GetModelList()
	require.NotEmpty(t, models)
	assert.Contains(t, models, "qwen3.8-flash")
	assert.Contains(t, models, "qwen-turbo")
	assert.Contains(t, models, "qwen-plus")
	assert.Contains(t, models, "qwen-max")
	assert.Contains(t, models, "qwq-32b")
}

func TestAliGetRequestURL(t *testing.T) {
	adaptor := &ali.Adaptor{}

	// Default base URL for chat completions
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeChatCompletions,
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "",
		},
	}
	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions", url)

	// Bruno collection URL with trailing slash
	info.ChannelMeta.ChannelBaseUrl = "https://token-plan.ap-southeast-1.maas.aliyuncs.com/"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://token-plan.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1/chat/completions", url)

	// URL already containing /compatible-mode/v1
	info.ChannelMeta.ChannelBaseUrl = "https://token-plan.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://token-plan.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1/chat/completions", url)

	// Anthropic compatible format with qwen3.8-flash
	info.RelayFormat = types.RelayFormatClaude
	info.ChannelMeta.UpstreamModelName = "qwen3.8-flash"
	info.ChannelMeta.ChannelBaseUrl = "https://token-plan.ap-southeast-1.maas.aliyuncs.com/"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://token-plan.ap-southeast-1.maas.aliyuncs.com/apps/anthropic/v1/messages", url)

	// Embeddings URL
	info.RelayFormat = types.RelayFormatEmbedding
	info.RelayMode = relayconstant.RelayModeEmbeddings
	info.ChannelMeta.ChannelBaseUrl = "https://dashscope.aliyuncs.com"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings", url)

	// QwenCloud Token Plan OpenAI endpoint
	info.RelayFormat = types.RelayFormatOpenAI
	info.RelayMode = relayconstant.RelayModeChatCompletions
	info.ChannelMeta.ChannelBaseUrl = "https://token-plan.maas.qwencloudapi.com/compatible-mode/v1"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://token-plan.maas.qwencloudapi.com/compatible-mode/v1/chat/completions", url)

	// QwenCloud Token Plan Anthropic endpoint
	info.RelayFormat = types.RelayFormatClaude
	info.ChannelMeta.UpstreamModelName = "deepseek-r1"
	info.ChannelMeta.ChannelBaseUrl = "https://token-plan.maas.qwencloudapi.com/apps/anthropic"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://token-plan.maas.qwencloudapi.com/apps/anthropic/v1/messages", url)

	// QwenCloud Pay-as-you-go endpoint
	info.RelayFormat = types.RelayFormatOpenAI
	info.ChannelMeta.ChannelBaseUrl = "https://maas.qwencloudapi.com"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://maas.qwencloudapi.com/compatible-mode/v1/chat/completions", url)
}
