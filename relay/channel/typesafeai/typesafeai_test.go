package typesafeai_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/typesafeai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeSafeAIAdaptorMetadata(t *testing.T) {
	adaptor := &typesafeai.Adaptor{}
	assert.Equal(t, "TypeSafe", adaptor.GetChannelName())
	models := adaptor.GetModelList()
	require.NotEmpty(t, models)
	assert.Contains(t, models, "jev-latest")
	assert.Contains(t, models, "jev-1.13.0")
	assert.Contains(t, models, "jev-preview")
}

func TestTypeSafeAIGetRequestURL(t *testing.T) {
	adaptor := &typesafeai.Adaptor{}

	// Default base URL
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeSystemOne,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "",
		},
	}
	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://api.typesafe.ai/v1/systemone", url)

	// Custom base URL with trailing slash
	info.ChannelMeta.ChannelBaseUrl = "https://custom.typesafe.ai/api/"
	url, err = adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://custom.typesafe.ai/api/v1/systemone", url)
}

func TestTypeSafeAISetupRequestHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	adaptor := &typesafeai.Adaptor{}
	reqHeader := http.Header{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "test-typesafe-key-123",
		},
	}

	err := adaptor.SetupRequestHeader(c, &reqHeader, info)
	require.NoError(t, err)
	assert.Equal(t, "application/json", reqHeader.Get("Content-Type"))
	assert.Equal(t, "Bearer test-typesafe-key-123", reqHeader.Get("Authorization"))
}

func TestTypeSafeAIConvertSystemOneRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adaptor := &typesafeai.Adaptor{}
	req := &dto.SystemOneRequest{
		Model: "jev-latest",
		State: "Context of the test decision.",
		Questions: []dto.SystemOneQuestion{
			{
				ID:       "q1",
				Question: "Should action A proceed?",
				Type:     "noul",
			},
		},
	}

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "jev-1.13.0",
		},
	}

	converted, err := adaptor.ConvertSystemOneRequest(c, info, req)
	require.NoError(t, err)
	require.NotNil(t, converted)

	convertedReq, ok := converted.(*dto.SystemOneRequest)
	require.True(t, ok)
	assert.Equal(t, "jev-1.13.0", convertedReq.Model)
	assert.Equal(t, "Context of the test decision.", convertedReq.State)
	require.Len(t, convertedReq.Questions, 1)
	assert.Equal(t, "q1", convertedReq.Questions[0].ID)
}

func TestTypeSafeAIDoResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adaptor := &typesafeai.Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeSystemOne,
	}

	upstreamJSON := `{"model":"jev-1.13.0","usage":{"prompt_tokens":120,"completion_tokens":0,"total_tokens":120},"answers":[{"id":"q1","type":"noul","noul":0.87}]}`
	httpResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(upstreamJSON)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}

	usageAny, apiErr := adaptor.DoResponse(c, httpResp, info)
	require.Nil(t, apiErr)
	require.NotNil(t, usageAny)

	usage, ok := usageAny.(*dto.Usage)
	require.True(t, ok)
	assert.Equal(t, 120, usage.PromptTokens)
	assert.Equal(t, 0, usage.CompletionTokens)
	assert.Equal(t, 120, usage.TotalTokens)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, upstreamJSON, w.Body.String())
}

func TestTypeSafeAIMapQuestionsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := `{
		"state": "Help! My payouts have been failing for 3 days.",
		"model": "jev-latest",
		"questions": {
			"is_urgent": {
				"type": "noul",
				"instructions": "Does this convey urgency?"
			}
		}
	}`

	var req dto.SystemOneRequest
	err := common.Unmarshal([]byte(payload), &req)
	require.NoError(t, err)
	assert.Equal(t, "jev-latest", req.Model)
	assert.Equal(t, "Help! My payouts have been failing for 3 days.", req.State)
	require.Len(t, req.Questions, 1)
	assert.Equal(t, "is_urgent", req.Questions[0].ID)
	assert.Equal(t, "noul", req.Questions[0].Type)
	assert.Equal(t, "Does this convey urgency?", req.Questions[0].Instructions)

	// Re-marshal should preserve questions map
	marshaled, err := common.Marshal(&req)
	require.NoError(t, err)
	assert.JSONEq(t, payload, string(marshaled))

	adaptor := &typesafeai.Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "jev-1.13.0",
		},
	}
	converted, err := adaptor.ConvertSystemOneRequest(c, info, &req)
	require.NoError(t, err)
	convertedReq, ok := converted.(*dto.SystemOneRequest)
	require.True(t, ok)
	assert.Equal(t, "jev-1.13.0", convertedReq.Model)
}
