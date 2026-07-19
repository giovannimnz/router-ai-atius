package controller

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

// Codex fetch-models is intentionally static: the upstream ChatGPT/Codex
// backend does not expose an OpenAI-compatible /v1/models list for this OAuth
// channel, and the admin UI must not show generic OpenAI or embedding models.
var codexFetchModelIDs = []string{
	"gpt-5.6-sol",
	"gpt-5.6-terra",
	"gpt-5.6-luna",
	"gpt-5.5",
	"gpt-5.3-codex-spark",
}

func fetchCodexModelIDs() []string {
	return append([]string(nil), codexFetchModelIDs...)
}

func fetchDynamicCodexModelIDs(channel *model.Channel) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return service.FetchCodexModelIDsForAdmin(ctx, channel)
}
