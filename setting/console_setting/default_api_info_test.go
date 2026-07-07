package console_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultAPIInfoIsValidAndIncludesResponses(t *testing.T) {
	require.NoError(t, ValidateConsoleSettings(DefaultAPIInfo, "ApiInfo"))

	items := getJSONList(DefaultAPIInfo)
	require.NotEmpty(t, items)

	foundResponses := false
	for _, item := range items {
		route, _ := item["route"].(string)
		url, _ := item["url"].(string)
		if route == "Responses" {
			foundResponses = true
			assert.Equal(t, "https://router.atius.com.br/v1/responses", url)
		}
	}

	assert.True(t, foundResponses)
}

func TestNormalizeAPIInfoListInjectsResponsesAfterOpenAICompatible(t *testing.T) {
	input := []map[string]interface{}{
		{
			"url":         "https://router.atius.com.br/v1/chat/completions",
			"route":       "OpenAI Compatible",
			"description": "Chat Completions API",
			"color":       "blue",
		},
		{
			"url":         "https://router.atius.com.br/v1/messages",
			"route":       "Anthropic Compatible",
			"description": "Messages API",
			"color":       "orange",
		},
	}

	normalized := normalizeAPIInfoList(input)

	require.Len(t, normalized, 3)
	assert.Equal(t, "OpenAI Compatible", normalized[0]["route"])
	assert.Equal(t, "Responses", normalized[1]["route"])
	assert.Equal(t, "https://router.atius.com.br/v1/responses", normalized[1]["url"])
}

func TestNormalizeAPIInfoListDoesNotDuplicateResponses(t *testing.T) {
	input := []map[string]interface{}{
		{
			"url":         "https://router.atius.com.br/v1/chat/completions",
			"route":       "OpenAI Compatible",
			"description": "Chat Completions API",
			"color":       "blue",
		},
		{
			"url":         "https://router.atius.com.br/v1/responses",
			"route":       "Responses",
			"description": "Responses API",
			"color":       "indigo",
		},
	}

	normalized := normalizeAPIInfoList(input)

	require.Len(t, normalized, 2)
	assert.Equal(t, "Responses", normalized[1]["route"])
}
