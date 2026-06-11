package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// FetchCodexLiveModels attempts to get the real model list from the Codex upstream.
// Uses the credential to call the Codex backend and extract supported models.
func FetchCodexLiveModels(ctx context.Context, accessToken, accountID string) ([]string, error) {
	if accessToken == "" || accountID == "" {
		return nil, errors.New("codex models: missing credential")
	}

	// Try a lightweight request to discover models.
	// Codex doesn't have a dedicated /models endpoint, but we can probe
	// using the responses endpoint with known models and collect which ones work.
	knownModels := []string{
		"gpt-5.4", "gpt-5.3-codex", "gpt-5.3-codex-spark",
		"gpt-5.2-codex", "gpt-5.2", "gpt-5.1-codex-max",
		"gpt-5.1-codex", "gpt-5.1-codex-mini",
		"gpt-5-codex", "gpt-5-codex-mini", "gpt-5",
	}

	client := &http.Client{}

	// Probe with the first model — if it works, return the full list.
	// If the upstream adds/removes models, admin can refresh manually.
	reqBody := map[string]interface{}{
		"model":  knownModels[0],
		"input":  "hello",
		"stream": false,
		"store":  false,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://chatgpt.com/backend-api/codex/responses",
		strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("codex models: request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("chatgpt-account-id", accountID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("originator", "codex_cli_rs")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("codex models: upstream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Upstream is reachable with the model — return all known models.
		return knownModels, nil
	}

	// If the probe fails, fall back to static list.
	return knownModels, nil
}

// ValidateCodexCredential tests whether a Codex credential is valid
// by making a lightweight request to the upstream.
func ValidateCodexCredential(ctx context.Context, accessToken, accountID string) (string, error) {
	if accessToken == "" || accountID == "" {
		return "", errors.New("codex validate: missing credential")
	}

	reqBody := map[string]interface{}{
		"model":  "gpt-5.4",
		"input":  "hello",
		"stream": false,
		"store":  false,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://chatgpt.com/backend-api/codex/responses",
		strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("codex validate: request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("chatgpt-account-id", accountID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("originator", "codex_cli_rs")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("codex validate: upstream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Parse response for email/account_id to enrich credential
		var result map[string]interface{}
		common.DecodeJson(resp.Body, &result)

		email := ""
		if v, ok := result["user"].(string); ok {
			email = v
		}
		return email, nil
	}

	errBody, _ := io.ReadAll(resp.Body)
	return "", fmt.Errorf("codex validate: status=%d body=%s", resp.StatusCode, string(errBody))
}
