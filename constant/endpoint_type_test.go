package constant

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEndpointTypeCanonicalizesLegacyRerank(t *testing.T) {
	t.Parallel()

	require.Equal(t, EndpointTypeReranker, NormalizeEndpointType(EndpointTypeReranker))
	require.Equal(t, EndpointTypeReranker, NormalizeEndpointType(EndpointTypeJinaRerank))
	require.Equal(t, EndpointTypeOpenAI, NormalizeEndpointType(EndpointTypeOpenAI))
}
