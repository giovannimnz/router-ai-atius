package codex

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOpenAIResponsesRequestNormalizesCodexUpstreamContract(t *testing.T) {
	clientStream := false
	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, nil, dto.OpenAIResponsesRequest{
		Model:  "gpt-5.6-sol",
		Input:  []byte(`"Reply only OK"`),
		Stream: &clientStream,
	})
	require.NoError(t, err)

	request, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.NotNil(t, request.Stream)
	assert.True(t, *request.Stream)
	assert.JSONEq(t, `false`, string(request.Store))

	var input []map[string]any
	require.NoError(t, common.Unmarshal(request.Input, &input))
	require.Len(t, input, 1)
	assert.Equal(t, "message", input[0]["type"])
	assert.Equal(t, "user", input[0]["role"])
}
