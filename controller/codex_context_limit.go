package controller

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
)

const (
	codexStandardContextLimitTokens = 272000
	codexLongContextLimitTokens     = 1050000
	codexLongContextMaxOutputTokens = 128000
)

func codexContextLimits(model string) (inputLimit int, maxOutput int, ok bool) {
	switch model {
	case "gpt-5.5", "gpt-5.4":
		return codexStandardContextLimitTokens, 0, true
	case "gpt-5.5-1m", "gpt-5.4-1m":
		return codexLongContextLimitTokens, codexLongContextMaxOutputTokens, true
	default:
		return 0, 0, false
	}
}

func requestModelName(request dto.Request) string {
	switch r := request.(type) {
	case *dto.GeneralOpenAIRequest:
		return r.Model
	case *dto.OpenAIResponsesRequest:
		return r.Model
	case *dto.ClaudeRequest:
		return r.Model
	case *dto.EmbeddingRequest:
		return r.Model
	default:
		return ""
	}
}

func safeCountCodexTextTokens(text string, model string) (tokens int) {
	defer func() {
		if recover() != nil {
			tokens = 0
		}
	}()
	return service.CountTextToken(text, model)
}

func shouldValidateCodexContextWindow(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	switch info.RelayMode {
	case relayconstant.RelayModeChatCompletions, relayconstant.RelayModeCompletions, relayconstant.RelayModeResponses:
	default:
		return false
	}
	_, _, ok := codexContextLimits(info.OriginModelName)
	return ok
}

func codexPromptTokensForLimit(info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) int {
	if meta == nil || info == nil {
		return promptTokens
	}

	localTokens := 0
	if meta.TokenType == types.TokenTypeTextNumber {
		localTokens += utf8.RuneCountInString(meta.CombineText)
	} else {
		localTokens += safeCountCodexTextTokens(meta.CombineText, info.OriginModelName)
	}
	if info.RelayFormat == types.RelayFormatOpenAI {
		localTokens += meta.ToolsCount * 8
		localTokens += meta.MessagesCount * 3
		localTokens += meta.NameCount * 3
		localTokens += 3
	}
	wordBasedTokens := (len(strings.Fields(meta.CombineText))*5 + 3) / 4
	if wordBasedTokens > localTokens {
		localTokens = wordBasedTokens
	}
	if localTokens > promptTokens {
		return localTokens
	}
	return promptTokens
}

func validateCodexContextWindow(info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) *types.NewAPIError {
	if !shouldValidateCodexContextWindow(info) {
		return nil
	}

	inputLimit, maxOutput, ok := codexContextLimits(info.OriginModelName)
	if !ok {
		return nil
	}
	promptTokens = codexPromptTokensForLimit(info, promptTokens, meta)
	if promptTokens > inputLimit {
		err := fmt.Errorf("model %s context limit exceeded: prompt tokens %d exceed input limit %d", info.OriginModelName, promptTokens, inputLimit)
		return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	if maxOutput > 0 && meta != nil && meta.MaxTokens > maxOutput {
		err := fmt.Errorf("model %s max output limit exceeded: requested max tokens %d exceed output limit %d", info.OriginModelName, meta.MaxTokens, maxOutput)
		return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	return nil
}
