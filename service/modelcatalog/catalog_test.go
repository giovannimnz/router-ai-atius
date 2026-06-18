package modelcatalog

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelCatalogEntryKeepsPricingProvenanceInternal(t *testing.T) {
	entry := BuildCatalogEntry(model.Pricing{
		ModelName: "zz-unpriced-model",
		SupportedEndpointTypes: []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeAnthropic,
			constant.EndpointTypeOpenAI,
		},
	}, map[string]string{"zz-unpriced-model": "MiniMax"})

	require.Equal(t, "missing", entry.PricingSource)
	require.True(t, entry.PricingEstimated)
	assert.Equal(t, []string{"OpenAI-Compatible", "Anthropic-Compatible"}, entry.SupportedEndpointTypeLabels)

	payload, err := common.Marshal(entry)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, common.Unmarshal(payload, &raw))
	assert.NotContains(t, raw, "pricing_source")
	assert.NotContains(t, raw, "pricing_estimated")
	assert.Contains(t, raw, "supported_endpoint_type_labels")
}
