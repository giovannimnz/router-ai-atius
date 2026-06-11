package openaicompat

import "github.com/QuantumNous/new-api/setting/model_setting"

// ShouldChatCompletionsUseResponsesPolicy checks whether a Chat Completions request
// should be translated to Responses API format before being sent upstream.
//
// Codex channels (type 57) always use Responses API — it's the only format Codex supports.
func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	// Codex channel — always use Responses API (Codex only speaks /v1/responses)
	if channelType == 57 {
		return true
	}
	if !policy.IsChannelEnabled(channelID, channelType) {
		return false
	}
	return matchAnyRegex(policy.ModelPatterns, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesPolicy(
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}
