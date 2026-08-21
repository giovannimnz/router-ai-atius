package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestChannelTestAcceptsCanonicalAndLegacyRerankerEndpointTypes(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeAdvancedCustom}
	for _, endpointType := range []string{
		string(constant.EndpointTypeReranker),
		string(constant.EndpointTypeJinaRerank),
	} {
		t.Run(endpointType, func(t *testing.T) {
			normalized := normalizeChannelTestEndpoint(channel, "reranker-gte-v1", endpointType)
			require.Equal(t, string(constant.EndpointTypeReranker), normalized)

			request, ok := buildTestRequest("reranker-gte-v1", normalized, channel, false).(*dto.RerankRequest)
			require.True(t, ok)
			require.Equal(t, "reranker-gte-v1", request.Model)
			require.Len(t, request.Documents, 2)
		})
	}
}
