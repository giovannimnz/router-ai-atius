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
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/embeddinggovernor"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

var acquireRerankGovernor = embeddinggovernor.Acquire

const maxGovernedTEIRerankDocuments = 20

func RerankHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	rerankReq, ok := info.Request.(*dto.RerankRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.RerankRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	publicModelName := rerankReq.Model
	if embeddinggovernor.IsGovernedModel(publicModelName) && len(rerankReq.Documents) > maxGovernedTEIRerankDocuments {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("%s supports at most %d documents per request", publicModelName, maxGovernedTEIRerankDocuments),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}
	inputChars := len(rerankReq.Query)
	for _, document := range rerankReq.Documents {
		inputChars += len(fmt.Sprint(document))
	}

	request, err := common.DeepCopy(rerankReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
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

	var requestBody io.Reader
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		requestBody = common.ReaderOnly(storage)
	} else {
		convertedRequest, err := adaptor.ConvertRerankRequest(c, info.RelayMode, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// apply param override
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return newAPIErrorFromParamOverride(err)
			}
		}

		logger.LogDebug(c, "Rerank request body: %s", jsonData)
		body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		defer closer.Close()
		jsonData = nil
		info.UpstreamRequestBodySize = size
		requestBody = body
	}
	lease, reject := acquireRerankGovernor(c.Request.Context(), embeddinggovernor.Request{
		Model:       publicModelName,
		ChannelID:   c.GetInt("channel_id"),
		ChannelName: c.GetString("channel_name"),
		Workload:    c.GetHeader("X-Rerank-Workload"),
		InputCount:  len(rerankReq.Documents),
		InputChars:  inputChars,
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

	statusCodeMappingStr := c.GetString("status_code_mapping")
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
	service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), nil)
	return nil
}
