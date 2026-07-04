package minimax

import (
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const defaultMiniMaxEmbeddingType = "query"

var validMiniMaxEmbeddingTypes = map[string]bool{
	"query": true,
	"db":    true,
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Texts []string `json:"texts"`
	Type  string   `json:"type"`
}

type embeddingBaseResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type embeddingResponse struct {
	Model       string                `json:"model"`
	Vectors     [][]float64           `json:"vectors"`
	TotalTokens int                   `json:"total_tokens"`
	BaseResp    embeddingBaseResponse `json:"base_resp"`
}

func openAIEmbeddingRequestToMiniMax(request dto.EmbeddingRequest) embeddingRequest {
	embeddingType := defaultMiniMaxEmbeddingType
	if validMiniMaxEmbeddingTypes[request.Type] {
		embeddingType = request.Type
	}
	return embeddingRequest{
		Model: request.Model,
		Texts: request.ParseInput(),
		Type:  embeddingType,
	}
}

func miniMaxEmbeddingResponseToOpenAI(requestModel string, response *embeddingResponse) *dto.OpenAIEmbeddingResponse {
	model := response.Model
	if model == "" {
		model = requestModel
	}
	openAIResponse := &dto.OpenAIEmbeddingResponse{
		Object: "list",
		Data:   make([]dto.OpenAIEmbeddingResponseItem, 0, len(response.Vectors)),
		Model:  model,
		Usage: dto.Usage{
			PromptTokens: response.TotalTokens,
			TotalTokens:  response.TotalTokens,
		},
	}
	for index, vector := range response.Vectors {
		openAIResponse.Data = append(openAIResponse.Data, dto.OpenAIEmbeddingResponseItem{
			Object:    "embedding",
			Index:     index,
			Embedding: vector,
		})
	}
	return openAIResponse
}

func miniMaxEmbeddingErrorStatus(code int) int {
	switch code {
	case 1002:
		return http.StatusTooManyRequests
	case 2013:
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

func miniMaxEmbeddingHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	service.CloseResponseBodyGracefully(resp)

	var miniMaxResponse embeddingResponse
	if err := common.Unmarshal(responseBody, &miniMaxResponse); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	if miniMaxResponse.BaseResp.StatusCode != 0 {
		message := miniMaxResponse.BaseResp.StatusMsg
		if message == "" {
			message = "MiniMax embeddings upstream error"
		}
		return nil, types.NewOpenAIError(
			fmt.Errorf("%s", message),
			types.ErrorCodeBadResponseBody,
			miniMaxEmbeddingErrorStatus(miniMaxResponse.BaseResp.StatusCode),
		)
	}

	openAIResponse := miniMaxEmbeddingResponseToOpenAI(info.UpstreamModelName, &miniMaxResponse)
	jsonResponse, err := common.Marshal(openAIResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("X-Embeddings-Adapter", "minimax-embo-01")
	c.Writer.WriteHeader(resp.StatusCode)
	service.IOCopyBytesGracefully(c, resp, jsonResponse)
	return &openAIResponse.Usage, nil
}
