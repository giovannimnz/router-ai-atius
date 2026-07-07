package console_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type defaultAPIInfoEntry struct {
	URL         string `json:"url"`
	Route       string `json:"route"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

var defaultAPIInfoEntries = []defaultAPIInfoEntry{
	{
		URL:         "https://router.atius.com.br/v1/chat/completions",
		Route:       "OpenAI Compatible",
		Description: "Chat Completions API - OpenAI-compatible endpoint for chat-style text generation",
		Color:       "blue",
	},
	{
		URL:         "https://router.atius.com.br/v1/responses",
		Route:       "Responses",
		Description: "Responses API - OpenAI-compatible endpoint for stateful and tool-ready responses",
		Color:       "indigo",
	},
	{
		URL:         "https://router.atius.com.br/v1/messages",
		Route:       "Anthropic Compatible",
		Description: "Messages API - Anthropic-compatible endpoint for Claude-format requests",
		Color:       "orange",
	},
	{
		URL:         "https://router.atius.com.br/v1/completions",
		Route:       "Completions",
		Description: "Completions API - Legacy prompt-completion endpoint",
		Color:       "green",
	},
	{
		URL:         "https://router.atius.com.br/v1/embeddings",
		Route:       "Embeddings",
		Description: "Embeddings API - Text embedding generation endpoint",
		Color:       "purple",
	},
	{
		URL:         "https://router.atius.com.br/v1/audio/speech",
		Route:       "Text-to-Speech",
		Description: "TTS API - Text-to-Speech synthesis endpoint",
		Color:       "pink",
	},
	{
		URL:         "https://router.atius.com.br/v1/models",
		Route:       "Models",
		Description: "Models API - List all available models",
		Color:       "cyan",
	},
}

var DefaultAPIInfo = mustMarshalDefaultAPIInfo()

func mustMarshalDefaultAPIInfo() string {
	data, err := common.Marshal(defaultAPIInfoEntries)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func normalizeAPIInfoList(items []map[string]interface{}) []map[string]interface{} {
	if len(items) == 0 {
		return getJSONList(DefaultAPIInfo)
	}

	hasResponses := false
	for _, item := range items {
		route, _ := item["route"].(string)
		url, _ := item["url"].(string)
		if strings.TrimSpace(route) == "Responses" || strings.HasSuffix(strings.TrimRight(strings.TrimSpace(url), "/"), "/v1/responses") {
			hasResponses = true
			break
		}
	}
	if hasResponses {
		return items
	}

	responseEntry := map[string]interface{}{
		"url":         defaultAPIInfoEntries[1].URL,
		"route":       defaultAPIInfoEntries[1].Route,
		"description": defaultAPIInfoEntries[1].Description,
		"color":       defaultAPIInfoEntries[1].Color,
	}

	out := make([]map[string]interface{}, 0, len(items)+1)
	inserted := false
	for _, item := range items {
		out = append(out, item)
		route, _ := item["route"].(string)
		if !inserted && strings.TrimSpace(route) == "OpenAI Compatible" {
			out = append(out, responseEntry)
			inserted = true
		}
	}
	if !inserted {
		out = append(out, responseEntry)
	}
	return out
}
