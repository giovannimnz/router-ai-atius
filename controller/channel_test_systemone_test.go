package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/modelcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelTestSystemOneEndpoint(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeTypesafeAI}

	// 1. Normalization by channel type
	normalized := normalizeChannelTestEndpoint(channel, "jev-latest", "")
	assert.Equal(t, string(constant.EndpointTypeSystemOne), normalized)

	// 2. Normalization by model prefix
	channelGeneric := &model.Channel{Type: constant.ChannelTypeCustom}
	normalizedByModel := normalizeChannelTestEndpoint(channelGeneric, "jev-1.13.0", "")
	assert.Equal(t, string(constant.EndpointTypeSystemOne), normalizedByModel)

	// 3. Normalization by explicit endpoint
	normalizedExplicit := normalizeChannelTestEndpoint(channelGeneric, "custom-model", "systemone")
	assert.Equal(t, string(constant.EndpointTypeSystemOne), normalizedExplicit)

	// 4. Test request building
	reqAny := buildTestRequest("jev-latest", normalized, channel, false)
	req, ok := reqAny.(*dto.SystemOneRequest)
	require.True(t, ok)
	assert.Equal(t, "jev-latest", req.Model)
	assert.NotEmpty(t, req.State)
	require.NotEmpty(t, req.Questions)
	assert.Equal(t, "noul", req.Questions[0].Type)

	// 5. Default endpoint info
	info, exists := common.GetDefaultEndpointInfo(constant.EndpointTypeSystemOne)
	require.True(t, exists)
	assert.Equal(t, "/v1/systemone", info.Path)
	assert.Equal(t, "POST", info.Method)

	// 6. Model catalog label
	label := modelcatalog.EndpointTypeLabel(constant.EndpointTypeSystemOne)
	assert.Equal(t, "SystemOne", label)
}
