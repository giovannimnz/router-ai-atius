package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
)

func TestGetFullRequestURLNormalizesBaseURLWithV1(t *testing.T) {
	t.Parallel()

	got := GetFullRequestURL("https://api.example.com/v1", "/v1/chat/completions", constant.ChannelTypeOpenAI)

	assert.Equal(t, "https://api.example.com/v1/chat/completions", got)
}

func TestGetFullRequestURLTrimsWhitespaceAndTrailingSlash(t *testing.T) {
	t.Parallel()

	got := GetFullRequestURL(" https://api.example.com/v1/ ", "v1/embeddings", constant.ChannelTypeOpenAI)

	assert.Equal(t, "https://api.example.com/v1/embeddings", got)
}

func TestNormalizeProviderRootBaseURLRemovesTrailingV1(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://api.example.com", NormalizeProviderRootBaseURL("https://api.example.com/v1/"))
}
