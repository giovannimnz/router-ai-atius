package relay

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/embeddinggovernor"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

var acquireEmbeddingGovernor = embeddinggovernor.Acquire
var executeEmbeddingChunkRequest = executeEmbeddingChunkRequestImpl
var postEmbeddingConsumeQuota = service.PostTextConsumeQuota

const maxGovernedTEIInputCount = 4

func EmbeddingHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	embeddingReq, ok := info.Request.(*dto.EmbeddingRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected *dto.EmbeddingRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	publicModelName := embeddingReq.Model
	inputStats := embeddingReq.GetInputStats()

	request, err := common.DeepCopy(embeddingReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to EmbeddingRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	if shouldSplitGovernedEmbeddingRequest(publicModelName, inputStats.InputCount) {
		return relayGovernedEmbeddingInChunks(c, info, adaptor, request, publicModelName)
	}

	convertedRequest, err := adaptor.ConvertEmbeddingRequest(c, info, *request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return newAPIErrorFromParamOverride(err)
		}
	}

	logger.LogDebug(c, "converted embedding request size: %d bytes", len(jsonData))
	body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer closer.Close()
	jsonData = nil
	info.UpstreamRequestBodySize = size
	var requestBody io.Reader = body
	statusCodeMappingStr := c.GetString("status_code_mapping")

	lease, reject := acquireEmbeddingGovernor(c.Request.Context(), embeddinggovernor.Request{
		Model:       publicModelName,
		ChannelID:   c.GetInt("channel_id"),
		ChannelName: c.GetString("channel_name"),
		Workload:    c.GetHeader("X-Embedding-Workload"),
		InputCount:  inputStats.InputCount,
		InputChars:  inputStats.InputChars,
	})
	if reject != nil {
		if reject.RetryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(int(reject.RetryAfter.Seconds())))
		}
		return types.NewErrorWithStatusCode(fmt.Errorf("%s", reject.Message), types.ErrorCode(reject.Code), reject.StatusCode, types.ErrOptionWithSkipRetry())
	}
	governorStartedAt := time.Now()
	finishGovernor := func(success bool, statusCode int) {
		if lease == nil {
			return
		}
		lease.Finish(success, statusCode, time.Since(governorStartedAt))
		lease = nil
	}

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		finishGovernor(false, http.StatusInternalServerError)
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			finishGovernor(false, httpResp.StatusCode)
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		statusCode := http.StatusInternalServerError
		if httpResp != nil {
			statusCode = httpResp.StatusCode
		}
		finishGovernor(false, statusCode)
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}
	finishGovernor(true, http.StatusOK)
	postEmbeddingConsumeQuota(c, info, usage.(*dto.Usage), nil)
	return nil
}

func shouldSplitGovernedEmbeddingRequest(publicModelName string, inputCount int) bool {
	return embeddinggovernor.IsGovernedModel(publicModelName) && inputCount > maxGovernedTEIInputCount
}

func relayGovernedEmbeddingInChunks(c *gin.Context, info *relaycommon.RelayInfo, adaptor relaychannel.Adaptor, request *dto.EmbeddingRequest, publicModelName string) *types.NewAPIError {
	inputs := request.ParseInput()
	chunks := chunkEmbeddingInputs(inputs, maxGovernedTEIInputCount)
	responses := make([]*dto.OpenAIEmbeddingResponse, 0, len(chunks))

	for _, chunk := range chunks {
		chunkRequest := *request
		chunkRequest.Input = chunk

		response, _, err := executeEmbeddingChunkRequest(c, info, adaptor, &chunkRequest, publicModelName)
		if err != nil {
			return err
		}
		responses = append(responses, response)
	}

	merged := mergeOpenAIEmbeddingResponses(responses, request.Model)
	c.JSON(http.StatusOK, merged)
	postEmbeddingConsumeQuota(c, info, &merged.Usage, nil)
	return nil
}

func executeEmbeddingChunkRequestImpl(c *gin.Context, info *relaycommon.RelayInfo, adaptor relaychannel.Adaptor, request *dto.EmbeddingRequest, publicModelName string) (*dto.OpenAIEmbeddingResponse, *dto.Usage, *types.NewAPIError) {
	convertedRequest, err := adaptor.ConvertEmbeddingRequest(c, info, *request)
	if err != nil {
		return nil, nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return nil, nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return nil, nil, newAPIErrorFromParamOverride(err)
		}
	}

	logger.LogDebug(c, "converted embedding request size: %d bytes", len(jsonData))
	body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return nil, nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer closer.Close()
	info.UpstreamRequestBodySize = size

	inputStats := request.GetInputStats()
	lease, reject := acquireEmbeddingGovernor(c.Request.Context(), embeddinggovernor.Request{
		Model:       publicModelName,
		ChannelID:   c.GetInt("channel_id"),
		ChannelName: c.GetString("channel_name"),
		Workload:    c.GetHeader("X-Embedding-Workload"),
		InputCount:  inputStats.InputCount,
		InputChars:  inputStats.InputChars,
	})
	if reject != nil {
		if reject.RetryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(int(reject.RetryAfter.Seconds())))
		}
		return nil, nil, types.NewErrorWithStatusCode(fmt.Errorf("%s", reject.Message), types.ErrorCode(reject.Code), reject.StatusCode, types.ErrOptionWithSkipRetry())
	}
	governorStartedAt := time.Now()
	finishGovernor := func(success bool, statusCode int) {
		if lease == nil {
			return
		}
		lease.Finish(success, statusCode, time.Since(governorStartedAt))
		lease = nil
	}

	resp, err := adaptor.DoRequest(c, info, body)
	if err != nil {
		finishGovernor(false, http.StatusInternalServerError)
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")
	httpResp, _ := resp.(*http.Response)
	if httpResp == nil {
		finishGovernor(false, http.StatusInternalServerError)
		return nil, nil, types.NewOpenAIError(fmt.Errorf("invalid embedding response type %T", resp), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		finishGovernor(false, httpResp.StatusCode)
		newAPIError := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return nil, nil, newAPIError
	}

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		finishGovernor(false, http.StatusInternalServerError)
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var embeddingResponse dto.OpenAIEmbeddingResponse
	if err := common.Unmarshal(responseBody, &embeddingResponse); err != nil {
		finishGovernor(false, http.StatusInternalServerError)
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	finishGovernor(true, http.StatusOK)
	usage := embeddingResponse.Usage
	return &embeddingResponse, &usage, nil
}

func chunkEmbeddingInputs(inputs []string, chunkSize int) [][]string {
	if len(inputs) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		return [][]string{append([]string(nil), inputs...)}
	}
	chunks := make([][]string, 0, (len(inputs)+chunkSize-1)/chunkSize)
	for start := 0; start < len(inputs); start += chunkSize {
		end := start + chunkSize
		if end > len(inputs) {
			end = len(inputs)
		}
		chunks = append(chunks, append([]string(nil), inputs[start:end]...))
	}
	return chunks
}

func mergeOpenAIEmbeddingResponses(chunks []*dto.OpenAIEmbeddingResponse, fallbackModel string) *dto.OpenAIEmbeddingResponse {
	merged := &dto.OpenAIEmbeddingResponse{
		Object: "list",
		Model:  fallbackModel,
		Data:   make([]dto.OpenAIEmbeddingResponseItem, 0),
	}
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		if chunk.Object != "" {
			merged.Object = chunk.Object
		}
		if chunk.Model != "" {
			merged.Model = chunk.Model
		}
		for _, item := range chunk.Data {
			item.Index = len(merged.Data)
			merged.Data = append(merged.Data, item)
		}
		merged.Usage.PromptTokens += chunk.Usage.PromptTokens
		merged.Usage.CompletionTokens += chunk.Usage.CompletionTokens
		merged.Usage.TotalTokens += chunk.Usage.TotalTokens
		merged.Usage.PromptCacheHitTokens += chunk.Usage.PromptCacheHitTokens
		merged.Usage.InputTokens += chunk.Usage.InputTokens
		merged.Usage.OutputTokens += chunk.Usage.OutputTokens
		merged.Usage.ClaudeCacheCreation5mTokens += chunk.Usage.ClaudeCacheCreation5mTokens
		merged.Usage.ClaudeCacheCreation1hTokens += chunk.Usage.ClaudeCacheCreation1hTokens
		merged.Usage.PromptTokensDetails.CachedTokens += chunk.Usage.PromptTokensDetails.CachedTokens
		merged.Usage.PromptTokensDetails.CachedCreationTokens += chunk.Usage.PromptTokensDetails.CachedCreationTokens
		merged.Usage.PromptTokensDetails.TextTokens += chunk.Usage.PromptTokensDetails.TextTokens
		merged.Usage.PromptTokensDetails.AudioTokens += chunk.Usage.PromptTokensDetails.AudioTokens
		merged.Usage.PromptTokensDetails.ImageTokens += chunk.Usage.PromptTokensDetails.ImageTokens
		merged.Usage.CompletionTokenDetails.TextTokens += chunk.Usage.CompletionTokenDetails.TextTokens
		merged.Usage.CompletionTokenDetails.AudioTokens += chunk.Usage.CompletionTokenDetails.AudioTokens
		merged.Usage.CompletionTokenDetails.ImageTokens += chunk.Usage.CompletionTokenDetails.ImageTokens
		merged.Usage.CompletionTokenDetails.ReasoningTokens += chunk.Usage.CompletionTokenDetails.ReasoningTokens
		if chunk.Usage.InputTokensDetails != nil {
			if merged.Usage.InputTokensDetails == nil {
				merged.Usage.InputTokensDetails = &dto.InputTokenDetails{}
			}
			merged.Usage.InputTokensDetails.CachedTokens += chunk.Usage.InputTokensDetails.CachedTokens
			merged.Usage.InputTokensDetails.CachedCreationTokens += chunk.Usage.InputTokensDetails.CachedCreationTokens
			merged.Usage.InputTokensDetails.TextTokens += chunk.Usage.InputTokensDetails.TextTokens
			merged.Usage.InputTokensDetails.AudioTokens += chunk.Usage.InputTokensDetails.AudioTokens
			merged.Usage.InputTokensDetails.ImageTokens += chunk.Usage.InputTokensDetails.ImageTokens
		}
		if merged.Usage.UsageSemantic == "" && chunk.Usage.UsageSemantic != "" {
			merged.Usage.UsageSemantic = chunk.Usage.UsageSemantic
		}
		if merged.Usage.UsageSource == "" && chunk.Usage.UsageSource != "" {
			merged.Usage.UsageSource = chunk.Usage.UsageSource
		}
	}
	return merged
}
