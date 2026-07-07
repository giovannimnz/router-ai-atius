package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
)

func TestShouldChatCompletionsUseResponsesPolicyAlwaysEnablesCodex(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       false,
		AllChannels:   false,
		ModelPatterns: nil,
	}

	assert.True(t, ShouldChatCompletionsUseResponsesPolicy(policy, 5, constant.ChannelTypeCodex, "gpt-5.4"))
}

func TestShouldChatCompletionsUseResponsesPolicyForAllActiveCodexModels(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{}
	models := []string{
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.3-codex-spark",
	}

	for _, model := range models {
		assert.True(t, ShouldChatCompletionsUseResponsesPolicy(policy, 5, constant.ChannelTypeCodex, model), model)
	}
}

func TestShouldChatCompletionsUseResponsesPolicyStillHonorsGlobalPolicyForOtherChannels(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   true,
		ModelPatterns: []string{`^gpt-5\.`},
	}

	assert.True(t, ShouldChatCompletionsUseResponsesPolicy(policy, 1, constant.ChannelTypeOpenAI, "gpt-5.4"))
	assert.False(t, ShouldChatCompletionsUseResponsesPolicy(policy, 1, constant.ChannelTypeOpenAI, "MiniMax-M3"))
}
