package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrioritizeCodexDefaultModel(t *testing.T) {
	reordered := prioritizeCodexDefaultModel([]string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini"}, "gpt-5.4")
	require.Equal(t, []string{"gpt-5.4", "gpt-5.5", "gpt-5.4-mini"}, reordered)
}

func TestNextCodexCatalogSyncDelay(t *testing.T) {
	location := time.FixedZone("BRT", -3*60*60)

	delay := nextCodexCatalogSyncDelay(time.Date(2026, 7, 7, 3, 30, 0, 0, location))
	assert.Equal(t, 30*time.Minute, delay)

	delay = nextCodexCatalogSyncDelay(time.Date(2026, 7, 7, 4, 30, 0, 0, location))
	assert.Equal(t, 23*time.Hour+30*time.Minute, delay)
}

func TestMergeCodexCatalogMetadataPrefersOverrideAndKeepsContextWindowGroup(t *testing.T) {
	meta := mergeCodexCatalogMetadata(
		"gpt-5.4",
		CodexCatalogMetadata{
			DisplayName: "Source Name",
			OwnedBy:     "source-owner",
		},
		CodexCatalogMetadata{
			DisplayName:         "Override Name",
			EndpointPreference:  constant.EndpointTypeOpenAIResponse,
			SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
			ContextWindowTokens: 1050000,
			MaxTokens:           1050000,
			MaxCompletionTokens: 128000,
		},
	)

	assert.Equal(t, "Override Name", meta.DisplayName)
	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI}, meta.SupportedEndpoints)
	assert.Equal(t, 1050000, meta.ContextWindowTokens)
	assert.Equal(t, 1050000, meta.MaxTokens)
	assert.Equal(t, 128000, meta.MaxCompletionTokens)
}
