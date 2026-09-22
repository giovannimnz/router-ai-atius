package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtiusLocalEmbeddingsUsesAdvancedCustomAPIType(t *testing.T) {
	apiType, ok := ChannelType2APIType(constant.ChannelTypeAtiusLocalEmbeddings)

	require.True(t, ok)
	assert.Equal(t, constant.APITypeAdvancedCustom, apiType)
	assert.True(t, constant.IsAdvancedCustomChannelType(constant.ChannelTypeAtiusLocalEmbeddings))
}

func TestCodexChannelTypeUsesCanonicalProductionName(t *testing.T) {
	assert.Equal(t, "ChatGPT - Codex", constant.GetChannelTypeName(constant.ChannelTypeCodex))
}

func TestAntigravityChannelTypeUsesAntigravityAPIType(t *testing.T) {
	apiType, ok := ChannelType2APIType(constant.ChannelTypeAntigravity)

	require.True(t, ok)
	assert.Equal(t, constant.APITypeAntigravity, apiType)
}

func TestAntigravityChannelTypeUsesCanonicalProductionName(t *testing.T) {
	assert.Equal(t, "Antigravity", constant.GetChannelTypeName(constant.ChannelTypeAntigravity))
}
