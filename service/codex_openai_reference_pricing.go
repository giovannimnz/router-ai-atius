package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	CodexOpenAIReferencePricingSourceURL = "https://developers.openai.com/api/docs/pricing.md"
	CodexOpenAIReferencePricingScope     = "openai_api_standard_reference"

	codexOpenAIReferencePricingOptionKey = "CodexOpenAIReferencePricing"
	codexOpenAIReferencePricingMaxBytes  = 1 << 20
)

var (
	codexOpenAIReferencePricingContextSuffix = regexp.MustCompile(`\s+\(<[^)]* context length\)$`)
	codexOpenAIReferenceMaxOutputPattern     = regexp.MustCompile(`(?i)([0-9][0-9,]*)\s*(?:<!--\s*-->)?\s*max output tokens`)
	codexOpenAIReferencePricingModels        = []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5"}
	codexOpenAIReferencePricingLoadOnce      sync.Once
	codexOpenAIReferencePricingMutex         sync.RWMutex
	codexOpenAIReferencePricingRefreshMutex  sync.Mutex
	codexOpenAIReferencePricingActive        = defaultCodexOpenAIReferencePricingSnapshot()
)

type CodexOpenAIReferencePrice struct {
	InputPerMillion       float64  `json:"input_per_million"`
	CachedInputPerMillion float64  `json:"cached_input_per_million"`
	CacheWritePerMillion  *float64 `json:"cache_write_per_million,omitempty"`
	OutputPerMillion      float64  `json:"output_per_million"`
	MaxCompletionTokens   int      `json:"max_completion_tokens"`
}

type codexOpenAIReferencePricingSnapshot struct {
	SourceURL    string                               `json:"source_url"`
	ETag         string                               `json:"etag,omitempty"`
	LastModified string                               `json:"last_modified,omitempty"`
	FetchedAt    time.Time                            `json:"fetched_at"`
	Prices       map[string]CodexOpenAIReferencePrice `json:"prices"`
}

type CodexOpenAIReferencePricingRefreshResult struct {
	NotModified      bool
	MetadataChanged  bool
	PriceChanged     bool
	UpdatedModels    int
	RegisteredModels int
}

type codexOpenAIReferencePricingHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func float64Pointer(value float64) *float64 {
	return &value
}

func defaultCodexOpenAIReferencePricingSnapshot() codexOpenAIReferencePricingSnapshot {
	return codexOpenAIReferencePricingSnapshot{
		SourceURL: CodexOpenAIReferencePricingSourceURL,
		FetchedAt: time.Date(2026, time.July, 18, 21, 40, 48, 0, time.UTC),
		Prices: map[string]CodexOpenAIReferencePrice{
			"gpt-5.6-sol": {
				InputPerMillion:       5,
				CachedInputPerMillion: 0.5,
				CacheWritePerMillion:  float64Pointer(6.25),
				OutputPerMillion:      30,
				MaxCompletionTokens:   128000,
			},
			"gpt-5.6-terra": {
				InputPerMillion:       2.5,
				CachedInputPerMillion: 0.25,
				CacheWritePerMillion:  float64Pointer(3.125),
				OutputPerMillion:      15,
				MaxCompletionTokens:   128000,
			},
			"gpt-5.6-luna": {
				InputPerMillion:       1,
				CachedInputPerMillion: 0.1,
				CacheWritePerMillion:  float64Pointer(1.25),
				OutputPerMillion:      6,
				MaxCompletionTokens:   128000,
			},
			"gpt-5.5": {
				InputPerMillion:       5,
				CachedInputPerMillion: 0.5,
				OutputPerMillion:      30,
				MaxCompletionTokens:   128000,
			},
		},
	}
}

func cloneCodexOpenAIReferencePrice(price CodexOpenAIReferencePrice) CodexOpenAIReferencePrice {
	if price.CacheWritePerMillion != nil {
		price.CacheWritePerMillion = float64Pointer(*price.CacheWritePerMillion)
	}
	return price
}

func cloneCodexOpenAIReferencePricingSnapshot(snapshot codexOpenAIReferencePricingSnapshot) codexOpenAIReferencePricingSnapshot {
	cloned := snapshot
	cloned.Prices = make(map[string]CodexOpenAIReferencePrice, len(snapshot.Prices))
	for modelName, price := range snapshot.Prices {
		cloned.Prices[modelName] = cloneCodexOpenAIReferencePrice(price)
	}
	return cloned
}

func validCodexOpenAIReferencePricingValue(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validateCodexOpenAIReferencePricingTable(snapshot codexOpenAIReferencePricingSnapshot) error {
	if snapshot.SourceURL != CodexOpenAIReferencePricingSourceURL {
		return fmt.Errorf("unexpected source URL %q", snapshot.SourceURL)
	}
	if snapshot.FetchedAt.IsZero() {
		return errors.New("missing fetch timestamp")
	}
	if len(snapshot.Prices) < len(codexOpenAIReferencePricingModels) {
		return fmt.Errorf("expected at least %d model prices, got %d", len(codexOpenAIReferencePricingModels), len(snapshot.Prices))
	}
	for modelName, price := range snapshot.Prices {
		if strings.TrimSpace(modelName) == "" {
			return errors.New("empty model name in pricing snapshot")
		}
		if !validCodexOpenAIReferencePricingValue(price.InputPerMillion) ||
			!validCodexOpenAIReferencePricingValue(price.OutputPerMillion) {
			return fmt.Errorf("invalid price for model %s", modelName)
		}
		if price.CachedInputPerMillion != 0 && !validCodexOpenAIReferencePricingValue(price.CachedInputPerMillion) {
			return fmt.Errorf("invalid cached input price for model %s", modelName)
		}
		if price.CacheWritePerMillion != nil && !validCodexOpenAIReferencePricingValue(*price.CacheWritePerMillion) {
			return fmt.Errorf("invalid cache write price for model %s", modelName)
		}
	}
	for _, modelName := range codexOpenAIReferencePricingModels {
		price, ok := snapshot.Prices[modelName]
		if !ok {
			return fmt.Errorf("missing required model %s", modelName)
		}
		if !validCodexOpenAIReferencePricingValue(price.CachedInputPerMillion) {
			return fmt.Errorf("missing cached input price for required model %s", modelName)
		}
	}
	return nil
}

func validateCodexOpenAIReferencePricingSnapshot(snapshot codexOpenAIReferencePricingSnapshot) error {
	if err := validateCodexOpenAIReferencePricingTable(snapshot); err != nil {
		return err
	}
	for _, modelName := range codexOpenAIReferencePricingModels {
		if snapshot.Prices[modelName].MaxCompletionTokens <= 0 {
			return fmt.Errorf("missing max completion tokens for required model %s", modelName)
		}
	}
	return nil
}

func normalizeCodexOpenAIReferencePricingModel(label string) string {
	return strings.TrimSpace(codexOpenAIReferencePricingContextSuffix.ReplaceAllString(strings.TrimSpace(label), ""))
}

func codexOpenAIReferencePricingNumber(value any) (float64, bool) {
	number, ok := value.(float64)
	return number, ok && validCodexOpenAIReferencePricingValue(number)
}

func parseCodexOpenAIReferencePricingRow(line string) (string, CodexOpenAIReferencePrice, bool, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[\"") || !strings.HasSuffix(line, "],") {
		return "", CodexOpenAIReferencePrice{}, false, nil
	}
	line = strings.TrimSuffix(line, ",")
	var values []any
	if err := common.Unmarshal([]byte(line), &values); err != nil {
		return "", CodexOpenAIReferencePrice{}, false, fmt.Errorf("parse pricing row: %w", err)
	}
	if len(values) != 4 && len(values) != 5 {
		return "", CodexOpenAIReferencePrice{}, false, nil
	}
	label, ok := values[0].(string)
	if !ok {
		return "", CodexOpenAIReferencePrice{}, false, nil
	}
	modelName := normalizeCodexOpenAIReferencePricingModel(label)
	input, inputOK := codexOpenAIReferencePricingNumber(values[1])
	outputIndex := len(values) - 1
	output, outputOK := codexOpenAIReferencePricingNumber(values[outputIndex])
	if modelName == "" || !inputOK || !outputOK {
		return "", CodexOpenAIReferencePrice{}, false, nil
	}

	price := CodexOpenAIReferencePrice{InputPerMillion: input, OutputPerMillion: output}
	if cachedInput, cachedOK := codexOpenAIReferencePricingNumber(values[2]); cachedOK {
		price.CachedInputPerMillion = cachedInput
	}
	if len(values) == 5 {
		if cacheWrite, cacheWriteOK := codexOpenAIReferencePricingNumber(values[3]); cacheWriteOK {
			price.CacheWritePerMillion = float64Pointer(cacheWrite)
		}
	}
	return modelName, price, true, nil
}

func parseCodexOpenAIStandardPricing(source []byte, fetchedAt time.Time, etag string, lastModified string) (codexOpenAIReferencePricingSnapshot, error) {
	const (
		standardMarker = `<div data-content-switcher-pane data-value="standard">`
		batchMarker    = `<div data-content-switcher-pane data-value="batch" hidden>`
	)

	content := string(source)
	standardStart := strings.Index(content, standardMarker)
	if standardStart < 0 {
		return codexOpenAIReferencePricingSnapshot{}, errors.New("standard pricing block not found")
	}
	standardBlock := content[standardStart+len(standardMarker):]
	standardEnd := strings.Index(standardBlock, batchMarker)
	if standardEnd < 0 {
		return codexOpenAIReferencePricingSnapshot{}, errors.New("standard pricing block terminator not found")
	}
	standardBlock = standardBlock[:standardEnd]

	prices := make(map[string]CodexOpenAIReferencePrice)
	for _, line := range strings.Split(standardBlock, "\n") {
		modelName, price, parsed, err := parseCodexOpenAIReferencePricingRow(line)
		if err != nil {
			return codexOpenAIReferencePricingSnapshot{}, err
		}
		if !parsed {
			continue
		}
		if _, duplicate := prices[modelName]; duplicate {
			return codexOpenAIReferencePricingSnapshot{}, fmt.Errorf("duplicate standard price for model %s", modelName)
		}
		prices[modelName] = price
	}

	snapshot := codexOpenAIReferencePricingSnapshot{
		SourceURL:    CodexOpenAIReferencePricingSourceURL,
		ETag:         strings.TrimSpace(etag),
		LastModified: strings.TrimSpace(lastModified),
		FetchedAt:    fetchedAt.UTC(),
		Prices:       prices,
	}
	if err := validateCodexOpenAIReferencePricingTable(snapshot); err != nil {
		return codexOpenAIReferencePricingSnapshot{}, err
	}
	return snapshot, nil
}

func codexOpenAIReferenceModelURL(modelName string) string {
	return "https://developers.openai.com/api/docs/models/" + strings.TrimSpace(modelName)
}

func parseCodexOpenAIReferenceMaxCompletionTokens(body []byte) (int, error) {
	match := codexOpenAIReferenceMaxOutputPattern.FindStringSubmatch(html.UnescapeString(string(body)))
	if len(match) != 2 {
		return 0, errors.New("max output tokens not found")
	}
	value, err := strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid max output tokens %q", match[1])
	}
	return value, nil
}

func fetchCodexOpenAIReferenceMaxCompletionTokens(ctx context.Context, client codexOpenAIReferencePricingHTTPClient, modelName string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, codexOpenAIReferenceModelURL(modelName), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "router-ai-atius-codex-pricing/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("model reference returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, codexOpenAIReferencePricingMaxBytes+1))
	if err != nil {
		return 0, err
	}
	if len(body) > codexOpenAIReferencePricingMaxBytes {
		return 0, errors.New("model reference exceeded maximum response size")
	}
	return parseCodexOpenAIReferenceMaxCompletionTokens(body)
}

func enrichCodexOpenAIReferenceModelLimits(ctx context.Context, client codexOpenAIReferencePricingHTTPClient, snapshot *codexOpenAIReferencePricingSnapshot) (bool, error) {
	changed := false
	for _, modelName := range codexOpenAIReferencePricingModels {
		maxCompletionTokens, err := fetchCodexOpenAIReferenceMaxCompletionTokens(ctx, client, modelName)
		if err != nil {
			return false, fmt.Errorf("fetch %s max output: %w", modelName, err)
		}
		price, ok := snapshot.Prices[modelName]
		if !ok {
			return false, fmt.Errorf("missing required model %s", modelName)
		}
		if price.MaxCompletionTokens != maxCompletionTokens {
			price.MaxCompletionTokens = maxCompletionTokens
			snapshot.Prices[modelName] = price
			changed = true
		}
	}
	return changed, nil
}

func fetchCodexOpenAIReferencePricing(ctx context.Context, client codexOpenAIReferencePricingHTTPClient, etag string, lastModified string) (codexOpenAIReferencePricingSnapshot, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, CodexOpenAIReferencePricingSourceURL, nil)
	if err != nil {
		return codexOpenAIReferencePricingSnapshot{}, false, err
	}
	req.Header.Set("Accept", "text/markdown")
	req.Header.Set("User-Agent", "router-ai-atius-codex-pricing/1.0")
	if strings.TrimSpace(etag) != "" {
		req.Header.Set("If-None-Match", strings.TrimSpace(etag))
	}
	if strings.TrimSpace(lastModified) != "" {
		req.Header.Set("If-Modified-Since", strings.TrimSpace(lastModified))
	}

	resp, err := client.Do(req)
	if err != nil {
		return codexOpenAIReferencePricingSnapshot{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return codexOpenAIReferencePricingSnapshot{}, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return codexOpenAIReferencePricingSnapshot{}, false, fmt.Errorf("pricing source returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, codexOpenAIReferencePricingMaxBytes+1))
	if err != nil {
		return codexOpenAIReferencePricingSnapshot{}, false, err
	}
	if len(body) > codexOpenAIReferencePricingMaxBytes {
		return codexOpenAIReferencePricingSnapshot{}, false, errors.New("pricing source exceeded maximum response size")
	}

	snapshot, err := parseCodexOpenAIStandardPricing(
		body,
		time.Now().UTC(),
		resp.Header.Get("ETag"),
		resp.Header.Get("Last-Modified"),
	)
	if err != nil {
		return codexOpenAIReferencePricingSnapshot{}, false, err
	}
	return snapshot, false, nil
}

func loadPersistedCodexOpenAIReferencePricing() {
	raw := readOptionMapValue(codexOpenAIReferencePricingOptionKey)
	if raw == "" {
		return
	}
	var snapshot codexOpenAIReferencePricingSnapshot
	if err := common.UnmarshalJsonStr(raw, &snapshot); err != nil {
		common.SysLog(fmt.Sprintf("codex pricing: ignoring invalid persisted snapshot: %v", err))
		return
	}
	if err := validateCodexOpenAIReferencePricingSnapshot(snapshot); err != nil {
		common.SysLog(fmt.Sprintf("codex pricing: ignoring invalid persisted snapshot: %v", err))
		return
	}
	codexOpenAIReferencePricingMutex.Lock()
	codexOpenAIReferencePricingActive = cloneCodexOpenAIReferencePricingSnapshot(snapshot)
	codexOpenAIReferencePricingMutex.Unlock()
}

func currentCodexOpenAIReferencePricingSnapshot() codexOpenAIReferencePricingSnapshot {
	codexOpenAIReferencePricingMutex.RLock()
	defer codexOpenAIReferencePricingMutex.RUnlock()
	return cloneCodexOpenAIReferencePricingSnapshot(codexOpenAIReferencePricingActive)
}

func equalCodexOpenAIReferencePrices(left map[string]CodexOpenAIReferencePrice, right map[string]CodexOpenAIReferencePrice) bool {
	if len(left) != len(right) {
		return false
	}
	for modelName, leftPrice := range left {
		rightPrice, ok := right[modelName]
		if !ok || leftPrice.InputPerMillion != rightPrice.InputPerMillion ||
			leftPrice.CachedInputPerMillion != rightPrice.CachedInputPerMillion ||
			leftPrice.OutputPerMillion != rightPrice.OutputPerMillion ||
			leftPrice.MaxCompletionTokens != rightPrice.MaxCompletionTokens {
			return false
		}
		if (leftPrice.CacheWritePerMillion == nil) != (rightPrice.CacheWritePerMillion == nil) {
			return false
		}
		if leftPrice.CacheWritePerMillion != nil && *leftPrice.CacheWritePerMillion != *rightPrice.CacheWritePerMillion {
			return false
		}
	}
	return true
}

func codexOpenAIPricedModelIDs(snapshot codexOpenAIReferencePricingSnapshot) []string {
	models := normalizeCodexCatalogModelNames(ListPromotedCodexModelIDs(codexCatalogDefaultChannelID))
	result := make([]string, 0, len(models))
	for _, modelName := range models {
		if _, ok := snapshot.Prices[modelName]; ok {
			result = append(result, modelName)
		}
	}
	return result
}

func codexOpenAIReferencePricingPatches(snapshot codexOpenAIReferencePricingSnapshot) map[string]*model.DollarCostPrice {
	patches := make(map[string]*model.DollarCostPrice)
	for _, modelName := range codexOpenAIPricedModelIDs(snapshot) {
		price := snapshot.Prices[modelName]
		cacheReadRatio := price.CachedInputPerMillion / price.InputPerMillion
		patch := &model.DollarCostPrice{
			Input:          price.InputPerMillion,
			Output:         price.OutputPerMillion,
			CacheReadRatio: &cacheReadRatio,
			SyncCacheRead:  true,
			SyncCacheWrite: true,
		}
		if price.CacheWritePerMillion != nil {
			cacheWriteRatio := *price.CacheWritePerMillion / price.InputPerMillion
			patch.CacheWriteRatio = &cacheWriteRatio
		}
		patches[modelName] = patch
	}
	return patches
}

func reconcileCodexOpenAIReferencePricing(snapshot codexOpenAIReferencePricingSnapshot, extraOptions map[string]string) (int, int, error) {
	modelNames := codexOpenAIPricedModelIDs(snapshot)
	registeredModels, err := model.EnsureExactModelMetadata(modelNames, "OpenAI Codex", "OpenAI")
	if err != nil {
		return 0, 0, err
	}
	updatedModels, err := model.PatchDollarCostPrices(codexOpenAIReferencePricingPatches(snapshot), extraOptions)
	if err != nil {
		return 0, registeredModels, err
	}
	if registeredModels > 0 {
		model.RefreshPricing()
	}
	return updatedModels, registeredModels, nil
}

func RefreshCodexOpenAIReferencePricing(ctx context.Context) (CodexOpenAIReferencePricingRefreshResult, error) {
	codexOpenAIReferencePricingRefreshMutex.Lock()
	defer codexOpenAIReferencePricingRefreshMutex.Unlock()

	current := currentCodexOpenAIReferencePricingSnapshot()
	snapshot, notModified, err := fetchCodexOpenAIReferencePricing(ctx, GetHttpClient(), current.ETag, current.LastModified)
	if err != nil {
		updatedModels, registeredModels, reconcileErr := reconcileCodexOpenAIReferencePricing(current, nil)
		if reconcileErr != nil {
			return CodexOpenAIReferencePricingRefreshResult{}, fmt.Errorf("fetch pricing: %v; reconcile last known pricing: %w", err, reconcileErr)
		}
		if updatedModels > 0 {
			common.SysLog(fmt.Sprintf("codex pricing source unavailable; restored last known canonical prices for %d models", updatedModels))
		}
		return CodexOpenAIReferencePricingRefreshResult{
			PriceChanged:     updatedModels > 0,
			UpdatedModels:    updatedModels,
			RegisteredModels: registeredModels,
		}, err
	}
	if notModified {
		snapshot = cloneCodexOpenAIReferencePricingSnapshot(current)
	}
	limitsChanged, limitsErr := enrichCodexOpenAIReferenceModelLimits(ctx, GetHttpClient(), &snapshot)
	if limitsErr != nil {
		updatedModels, registeredModels, reconcileErr := reconcileCodexOpenAIReferencePricing(current, nil)
		if reconcileErr != nil {
			return CodexOpenAIReferencePricingRefreshResult{}, fmt.Errorf("refresh official model limits: %v; reconcile last known pricing: %w", limitsErr, reconcileErr)
		}
		return CodexOpenAIReferencePricingRefreshResult{
			PriceChanged:     updatedModels > 0,
			UpdatedModels:    updatedModels,
			RegisteredModels: registeredModels,
		}, limitsErr
	}
	if limitsChanged {
		snapshot.FetchedAt = time.Now().UTC()
	}
	if err := validateCodexOpenAIReferencePricingSnapshot(snapshot); err != nil {
		return CodexOpenAIReferencePricingRefreshResult{}, err
	}
	if notModified && !limitsChanged {
		updatedModels, registeredModels, reconcileErr := reconcileCodexOpenAIReferencePricing(current, nil)
		if reconcileErr != nil {
			return CodexOpenAIReferencePricingRefreshResult{}, fmt.Errorf("restore unchanged canonical pricing: %w", reconcileErr)
		}
		if updatedModels > 0 {
			common.SysLog(fmt.Sprintf("codex pricing refresh not modified; repaired canonical prices for %d models", updatedModels))
			return CodexOpenAIReferencePricingRefreshResult{
				NotModified:      true,
				PriceChanged:     true,
				UpdatedModels:    updatedModels,
				RegisteredModels: registeredModels,
			}, nil
		}
		common.SysLog(fmt.Sprintf("codex pricing refresh unchanged: source=%s models=%d", CodexOpenAIReferencePricingSourceURL, len(current.Prices)))
		return CodexOpenAIReferencePricingRefreshResult{NotModified: true, RegisteredModels: registeredModels}, nil
	}

	metadataChanged := readOptionMapValue(codexOpenAIReferencePricingOptionKey) == "" ||
		current.SourceURL != snapshot.SourceURL || current.ETag != snapshot.ETag ||
		current.LastModified != snapshot.LastModified || !equalCodexOpenAIReferencePrices(current.Prices, snapshot.Prices)
	extraOptions := make(map[string]string)
	if metadataChanged {
		raw, marshalErr := common.Marshal(snapshot)
		if marshalErr != nil {
			return CodexOpenAIReferencePricingRefreshResult{}, marshalErr
		}
		extraOptions[codexOpenAIReferencePricingOptionKey] = string(raw)
	}
	updatedModels, registeredModels, err := reconcileCodexOpenAIReferencePricing(snapshot, extraOptions)
	if err != nil {
		return CodexOpenAIReferencePricingRefreshResult{}, fmt.Errorf("persist canonical pricing options: %w", err)
	}
	if updatedModels == 0 && !metadataChanged {
		common.SysLog(fmt.Sprintf("codex pricing refresh unchanged after parse: source=%s models=%d", snapshot.SourceURL, len(snapshot.Prices)))
		return CodexOpenAIReferencePricingRefreshResult{}, nil
	}
	if metadataChanged {
		codexOpenAIReferencePricingMutex.Lock()
		codexOpenAIReferencePricingActive = cloneCodexOpenAIReferencePricingSnapshot(snapshot)
		codexOpenAIReferencePricingMutex.Unlock()
	}
	common.SysLog(fmt.Sprintf("codex pricing refresh complete: source=%s price_changed=%t updated_models=%d metadata_changed=%t fetched_at=%s", snapshot.SourceURL, updatedModels > 0, updatedModels, metadataChanged, snapshot.FetchedAt.Format(time.RFC3339)))
	return CodexOpenAIReferencePricingRefreshResult{
		MetadataChanged:  metadataChanged,
		PriceChanged:     updatedModels > 0,
		UpdatedModels:    updatedModels,
		RegisteredModels: registeredModels,
	}, nil
}

func CodexOpenAIReferencePriceForModel(modelName string) (CodexOpenAIReferencePrice, bool) {
	codexOpenAIReferencePricingMutex.RLock()
	defer codexOpenAIReferencePricingMutex.RUnlock()
	price, ok := codexOpenAIReferencePricingActive.Prices[strings.TrimSpace(modelName)]
	return cloneCodexOpenAIReferencePrice(price), ok
}
