package controller

// Codex fetch-models is intentionally static: the upstream ChatGPT/Codex
// backend does not expose an OpenAI-compatible /v1/models list for this OAuth
// channel, and the admin UI must not show generic OpenAI or embedding models.
var codexFetchModelIDs = []string{
	"gpt-5.5",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.3-codex-spark",
}

func fetchCodexModelIDs() []string {
	return append([]string(nil), codexFetchModelIDs...)
}
