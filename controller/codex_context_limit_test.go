package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCodexContextWindow(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		promptTokens int
		maxTokens    int
		wantError    bool
	}{
		{name: "base at standard limit", model: "gpt-5.5", promptTokens: codexStandardContextLimitTokens},
		{name: "base above standard limit", model: "gpt-5.5", promptTokens: codexStandardContextLimitTokens + 1, wantError: true},
		{name: "long context at input limit", model: "gpt-5.5-1m", promptTokens: codexLongContextLimitTokens, maxTokens: codexLongContextMaxOutputTokens},
		{name: "long context above input limit", model: "gpt-5.5-1m", promptTokens: codexLongContextLimitTokens + 1, wantError: true},
		{name: "long context above output limit", model: "gpt-5.4-1m", promptTokens: 1000, maxTokens: codexLongContextMaxOutputTokens + 1, wantError: true},
		{name: "unrelated model ignored", model: "MiniMax-M3", promptTokens: codexLongContextLimitTokens + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: tt.model,
				RelayMode:       relayconstant.RelayModeChatCompletions,
			}
			meta := &types.TokenCountMeta{MaxTokens: tt.maxTokens}

			err := validateCodexContextWindow(info, tt.promptTokens, meta)
			if tt.wantError {
				require.NotNil(t, err)
				assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
				return
			}
			require.Nil(t, err)
		})
	}
}

func TestRequestModelNameFallsBackToOpenAIRequestModel(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "gpt-5.4"}

	require.Equal(t, "gpt-5.4", requestModelName(request))
}

func TestValidateCodexContextWindowCountsMetaWhenPromptTokensMissing(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.4",
		RelayMode:       relayconstant.RelayModeChatCompletions,
	}
	meta := &types.TokenCountMeta{
		TokenType:     types.TokenTypeTextNumber,
		CombineText:   strings.Repeat("a", codexStandardContextLimitTokens+1),
		MessagesCount: 1,
	}

	err := validateCodexContextWindow(info, 0, meta)
	require.NotNil(t, err)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
}

func TestValidateCodexContextWindowUsesConservativeWordEstimate(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.4",
		RelayFormat:     types.RelayFormatOpenAI,
		RelayMode:       relayconstant.RelayModeChatCompletions,
	}
	meta := &types.TokenCountMeta{
		TokenType:   types.TokenTypeTokenizer,
		CombineText: strings.Repeat("alpha ", 218000),
	}

	err := validateCodexContextWindow(info, 0, meta)
	require.NotNil(t, err)
	assert.Equal(t, types.ErrorCodeInvalidRequest, err.GetErrorCode())
}
