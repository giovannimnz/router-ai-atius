package advancedcustom

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func teiRerankRelayInfo(documents []any, returnDocuments bool) *relaycommon.RelayInfo {
	info := advancedCustomRelayInfo(&dto.AdvancedCustomConfig{Routes: []dto.AdvancedCustomRoute{{
		IncomingPath: "/v1/rerank",
		UpstreamPath: "/rerank",
		Converter:    dto.AdvancedCustomConverterJinaRerankToTEINative,
		Auth:         &dto.AdvancedCustomRouteAuth{Type: dto.AdvancedCustomAuthTypeNone},
	}}})
	info.RelayMode = relayconstant.RelayModeRerank
	info.RelayFormat = types.RelayFormatRerank
	info.RequestURLPath = "/v1/rerank"
	info.RerankerInfo = &relaycommon.RerankerInfo{Documents: documents, ReturnDocuments: returnDocuments}
	return info
}

func TestTEIRerankRequestConversion(t *testing.T) {
	documents := []any{"primeiro", map[string]any{"text": "segundo"}}
	info := teiRerankRelayInfo(documents, true)
	adaptor := &Adaptor{}
	adaptor.Init(info)
	topN := 1

	converted, err := adaptor.ConvertRerankRequest(advancedCustomGinContext("/v1/rerank"), relayconstant.RelayModeRerank, dto.RerankRequest{
		Query:     "consulta",
		Documents: documents,
		TopN:      &topN,
	})
	require.NoError(t, err)

	request, ok := converted.(teiRerankRequest)
	require.True(t, ok)
	assert.Equal(t, "consulta", request.Query)
	assert.Equal(t, []string{"primeiro", "segundo"}, request.Texts)
	assert.True(t, request.Truncate)
	assert.False(t, request.RawScores)
	assert.False(t, request.ReturnText)
	assert.Equal(t, 1, adaptor.rerankTopN)

	requestURL, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://fallback.example/rerank", requestURL)
}

func TestTEIRerankResponseConversionAppliesTopNAndDocuments(t *testing.T) {
	documents := []any{"zero", "um", "dois"}
	info := teiRerankRelayInfo(documents, true)
	adaptor := &Adaptor{}
	adaptor.Init(info)
	topN := 2
	_, err := adaptor.ConvertRerankRequest(advancedCustomGinContext("/v1/rerank"), relayconstant.RelayModeRerank, dto.RerankRequest{
		Query: "q", Documents: documents, TopN: &topN,
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := ginTestContext(recorder, "/v1/rerank")
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[
		{"index":0,"score":0.2},{"index":2,"score":0.9},{"index":1,"score":0.7}
	]`))}
	usage, apiErr := adaptor.DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)

	var output dto.RerankResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &output))
	require.Len(t, output.Results, 2)
	assert.Equal(t, 2, output.Results[0].Index)
	assert.Equal(t, "dois", output.Results[0].Document)
	assert.InDelta(t, 0.9, output.Results[0].RelevanceScore, 0.000001)
	assert.Equal(t, 1, output.Results[1].Index)
}

func ginTestContext(recorder *httptest.ResponseRecorder, path string) (*gin.Context, *httptest.ResponseRecorder) {
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	return c, recorder
}
