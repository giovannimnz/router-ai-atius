package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestAtiusLocalModelsAdvertiseTheirActualEndpoints(t *testing.T) {
	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeEmbeddings},
		GetEndpointTypesByChannelType(constant.ChannelTypeAtiusLocalEmbeddings, constant.AtiusLocalEmbeddingModel),
	)
	assert.Equal(t,
		[]constant.EndpointType{constant.EndpointTypeJinaRerank},
		GetEndpointTypesByChannelType(constant.ChannelTypeAtiusLocalEmbeddings, constant.AtiusLocalRerankerModel),
	)
}
