package advancedcustom

import (
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type teiRerankRequest struct {
	Query               string   `json:"query"`
	Texts               []string `json:"texts"`
	RawScores           bool     `json:"raw_scores"`
	ReturnText          bool     `json:"return_text"`
	Truncate            bool     `json:"truncate"`
	TruncationDirection string   `json:"truncation_direction"`
}

type teiRerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
	Text  *string `json:"text,omitempty"`
}

func newTEIRerankRequest(request dto.RerankRequest) teiRerankRequest {
	texts := make([]string, len(request.Documents))
	for i, document := range request.Documents {
		texts[i] = rerankDocumentText(document)
	}
	return teiRerankRequest{
		Query:               request.Query,
		Texts:               texts,
		RawScores:           false,
		ReturnText:          false,
		Truncate:            true,
		TruncationDirection: "right",
	}
}

func rerankDocumentText(document any) string {
	switch value := document.(type) {
	case string:
		return value
	case dto.RerankDocument:
		return fmt.Sprint(value.Text)
	case map[string]any:
		if text, ok := value["text"]; ok {
			return fmt.Sprint(text)
		}
	}
	return fmt.Sprint(document)
}

func normalizedRerankTopN(topN *int, documentCount int) int {
	if topN == nil || *topN <= 0 || *topN > documentCount {
		return documentCount
	}
	return *topN
}

func doTEIRerankResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, topN int) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)

	var teiResults []teiRerankResult
	if err := common.Unmarshal(responseBody, &teiResults); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	sort.SliceStable(teiResults, func(i, j int) bool {
		return teiResults[i].Score > teiResults[j].Score
	})
	if topN > 0 && topN < len(teiResults) {
		teiResults = teiResults[:topN]
	}

	results := make([]dto.RerankResponseResult, 0, len(teiResults))
	for _, result := range teiResults {
		if result.Index < 0 || info == nil || result.Index >= len(info.Documents) {
			return nil, types.NewOpenAIError(fmt.Errorf("TEI rerank result index out of range: %d", result.Index), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		converted := dto.RerankResponseResult{
			Index:          result.Index,
			RelevanceScore: result.Score,
		}
		if info.ReturnDocuments {
			converted.Document = info.Documents[result.Index]
		}
		results = append(results, converted)
	}

	tokens := 0
	if info != nil {
		tokens = info.GetEstimatePromptTokens()
	}
	usage := dto.Usage{PromptTokens: tokens, TotalTokens: tokens}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.JSON(http.StatusOK, dto.RerankResponse{Results: results, Usage: usage})
	return &usage, nil
}
