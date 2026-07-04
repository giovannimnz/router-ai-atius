package deepseek

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestURLUsesOpenAIPathForOpenAIFormat(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeChatCompletions,
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.deepseek.com",
		},
	}

	got, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.deepseek.com/v1/chat/completions", got)
}

func TestGetRequestURLUsesOpenAIPathWithBaseURLV1(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeChatCompletions,
		RelayFormat: types.RelayFormatOpenAI,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.deepseek.com/v1",
		},
	}

	got, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.deepseek.com/v1/chat/completions", got)
}

func TestGetRequestURLUsesAnthropicPathForClaudeFormat(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeUnknown,
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.deepseek.com",
		},
	}

	got, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.deepseek.com/anthropic/v1/messages", got)
}

func TestGetRequestURLUsesAnthropicPathWithBaseURLV1(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeUnknown,
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.deepseek.com/v1",
		},
	}

	got, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.deepseek.com/anthropic/v1/messages", got)
}
