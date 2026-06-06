package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// SidecarProxyTimeout is the max duration for a Codex SDK request through the sidecar.
// SDK calls can take minutes for complex prompts — timeout is generous.
const SidecarProxyTimeout = 300 * time.Second

// ProxyCodexSDKRequest proxies a one-shot request to the Codex sidecar at POST /v1/codex/run.
//
// sidecarBaseURL is the sidecar address, e.g. "http://codex-sidecar:1456".
// requestBody is the JSON body with {model, prompt, stream: false}.
// Returns the HTTP status code, response body, and any error.
func ProxyCodexSDKRequest(ctx context.Context, client *http.Client, sidecarBaseURL string, requestBody []byte) (statusCode int, body []byte, err error) {
	return proxyCodexSDK(ctx, client, sidecarBaseURL, "/v1/codex/run", requestBody)
}

// ProxyCodexSDKThread proxies a stateful thread request to the Codex sidecar at POST /v1/codex/thread.
//
// requestBody may include optional thread_id for continuing an existing thread.
func ProxyCodexSDKThread(ctx context.Context, client *http.Client, sidecarBaseURL string, requestBody []byte) (statusCode int, body []byte, err error) {
	return proxyCodexSDK(ctx, client, sidecarBaseURL, "/v1/codex/thread", requestBody)
}

// ProxyCodexSDKStream proxies a streaming request to the sidecar.
//
// The caller is responsible for reading and forwarding the SSE response body.
func ProxyCodexSDKStream(ctx context.Context, client *http.Client, sidecarBaseURL string, requestBody []byte) (*http.Response, error) {
	if client == nil {
		return nil, fmt.Errorf("nil http client")
	}
	bu := strings.TrimRight(strings.TrimSpace(sidecarBaseURL), "/")
	if bu == "" {
		return nil, fmt.Errorf("empty sidecarBaseURL")
	}

	// Wrap the body to add stream:true
	var wrapper map[string]interface{}
	if err := common.Unmarshal(requestBody, &wrapper); err != nil {
		return nil, fmt.Errorf("codex sdk stream: invalid request body: %w", err)
	}
	wrapper["stream"] = true
	streamBody, err := common.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("codex sdk stream: marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bu+"/v1/codex/run", bytes.NewReader(streamBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	return client.Do(req)
}

// proxyCodexSDK is the internal helper for non-streaming proxy calls.
func proxyCodexSDK(ctx context.Context, client *http.Client, sidecarBaseURL, path string, requestBody []byte) (int, []byte, error) {
	if client == nil {
		return 0, nil, fmt.Errorf("nil http client")
	}
	bu := strings.TrimRight(strings.TrimSpace(sidecarBaseURL), "/")
	if bu == "" {
		return 0, nil, fmt.Errorf("empty sidecarBaseURL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bu+path, bytes.NewReader(requestBody))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}
