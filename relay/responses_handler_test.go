package relay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexResponsesUpstreamAuthErrorNormalizesTokenInvalidated(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body: io.NopCloser(strings.NewReader(
			`{"error":{"message":"token was invalidated","type":"invalid_request_error","code":"token_invalidated"}}`,
		)),
	}

	newAPIError := service.RelayErrorHandler(context.Background(), resp, false)
	normalized := service.NormalizeCodexUpstreamAuthError(newAPIError)

	require.NotNil(t, normalized)
	assert.Equal(t, http.StatusUnauthorized, normalized.StatusCode)
	assert.Equal(t, types.ErrorCodeCodexUpstreamTokenInvalidated, normalized.GetErrorCode())
	assert.Equal(t, "codex_upstream_auth_error", normalized.ToOpenAIError().Type)
}
