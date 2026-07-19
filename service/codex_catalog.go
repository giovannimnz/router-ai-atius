package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"gorm.io/gorm"
)

const (
	codexCatalogDefaultChannelID           = 5
	codexCatalogDefaultClientVersion       = "0.111.0"
	codexCatalogDefaultReply               = "Ok"
	codexCatalogValidationContractVersion  = "3"
	codexCatalogDefaultDiscoveryTimeout    = 20 * time.Second
	codexCatalogDefaultValidationTimeout   = 30 * time.Second
	codexCatalogDefaultModelOptionKey      = "CodexCatalogDefaultModel"
	codexCatalogDenylistOptionKey          = "CodexCatalogDenylist"
	codexCatalogOverridesOptionKey         = "CodexCatalogMetadataOverrides"
	CodexCatalogBillingMode                = "dollar_cost"
	codexCatalogDiscoveryClientVersionEnv  = "CODEX_DISCOVERY_CLIENT_VERSION"
	codexCatalogDiscoveryClientVersionHint = "codex --version"
)

var (
	codexCatalogVersionPattern = regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)
	codexCatalogMutex          sync.Mutex
	knownRetiredCodexModels    = map[string]struct{}{
		"gpt-5.4":      {},
		"gpt-5.4-mini": {},
	}
)

type CodexCatalogMetadata struct {
	DisplayName               string                  `json:"display_name,omitempty"`
	Provider                  string                  `json:"provider,omitempty"`
	OwnedBy                   string                  `json:"owned_by,omitempty"`
	EndpointPreference        constant.EndpointType   `json:"endpoint_preference,omitempty"`
	SupportedEndpoints        []constant.EndpointType `json:"supported_endpoints,omitempty"`
	ContextWindowTokens       int                     `json:"context_window_tokens,omitempty"`
	MaxTokens                 int                     `json:"max_tokens,omitempty"`
	MaxCompletionTokens       int                     `json:"max_completion_tokens,omitempty"`
	SupportedReasoningEfforts []string                `json:"supported_reasoning_efforts,omitempty"`
	Capabilities              []string                `json:"capabilities,omitempty"`
	BillingMode               string                  `json:"billing_mode,omitempty"`
}

type codexCatalogPolicy struct {
	DefaultModel string                          `json:"default_model,omitempty"`
	Denylist     []string                        `json:"denylist,omitempty"`
	Overrides    map[string]CodexCatalogMetadata `json:"overrides,omitempty"`
}

type codexDiscoveryItem struct {
	Slug                string `json:"slug"`
	Visibility          string `json:"visibility,omitempty"`
	ContextWindow       int    `json:"context_window,omitempty"`
	MaxContextWindow    int    `json:"max_context_window,omitempty"`
	MaxOutputTokens     int    `json:"max_output_tokens,omitempty"`
	MaxCompletionTokens int    `json:"max_completion_tokens,omitempty"`
}

type codexDiscoveryResult struct {
	Models []string
	Hidden []string
	Items  map[string]codexDiscoveryItem
}

type codexDiscoveryResponse struct {
	Models []codexDiscoveryItem `json:"models"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type CodexCatalogSyncResult struct {
	ChannelID      int
	Discovered     []string
	Promoted       []string
	Changed        bool
	ValidatedCount int
}

func normalizeCodexCatalogModelNames(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func filterKnownRetiredCodexModels(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, modelName := range normalizeCodexCatalogModelNames(values) {
		if _, retired := knownRetiredCodexModels[modelName]; retired {
			continue
		}
		filtered = append(filtered, modelName)
	}
	return filtered
}

func officialGPT56CodexMetadata(displayName string, reasoningEfforts []string) CodexCatalogMetadata {
	return CodexCatalogMetadata{
		DisplayName:               displayName,
		Provider:                  "OpenAI Codex",
		OwnedBy:                   "codex",
		EndpointPreference:        constant.EndpointTypeOpenAIResponse,
		SupportedEndpoints:        []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
		ContextWindowTokens:       272000,
		MaxTokens:                 272000,
		BillingMode:               CodexCatalogBillingMode,
		SupportedReasoningEfforts: append([]string(nil), reasoningEfforts...),
		Capabilities: []string{
			"text_input",
			"image_input",
			"text_output",
			"streaming",
			"function_calling",
			"structured_outputs",
			"web_search",
			"file_search",
			"image_generation",
			"code_interpreter",
			"hosted_shell",
			"apply_patch",
			"skills",
			"computer_use",
			"mcp",
			"tool_search",
		},
	}
}

func defaultCodexCatalogPolicy() codexCatalogPolicy {
	return codexCatalogPolicy{
		DefaultModel: "gpt-5.6-terra",
		Denylist: []string{
			"codex-auto-review",
			"gpt-5.4-1m",
			"gpt-5.5-1m",
		},
		Overrides: map[string]CodexCatalogMetadata{
			"gpt-5.6-sol": officialGPT56CodexMetadata(
				"OpenAI Codex GPT-5.6 Sol",
				[]string{"low", "medium", "high", "xhigh", "max", "ultra"},
			),
			"gpt-5.6-terra": officialGPT56CodexMetadata(
				"OpenAI Codex GPT-5.6 Terra",
				[]string{"low", "medium", "high", "xhigh", "max", "ultra"},
			),
			"gpt-5.6-luna": officialGPT56CodexMetadata(
				"OpenAI Codex GPT-5.6 Luna",
				[]string{"low", "medium", "high", "xhigh", "max"},
			),
			"gpt-5.5": {
				DisplayName:         "OpenAI Codex GPT-5.5",
				Provider:            "OpenAI Codex",
				OwnedBy:             "codex",
				EndpointPreference:  constant.EndpointTypeOpenAIResponse,
				SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
				ContextWindowTokens: 272000,
				MaxTokens:           272000,
				BillingMode:         CodexCatalogBillingMode,
			},
			"gpt-5.3-codex-spark": {
				DisplayName:         "OpenAI Codex GPT-5.3-codex-spark",
				Provider:            "OpenAI Codex",
				OwnedBy:             "codex",
				EndpointPreference:  constant.EndpointTypeOpenAIResponse,
				SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
				ContextWindowTokens: 128000,
				MaxTokens:           128000,
				BillingMode:         CodexCatalogBillingMode,
			},
		},
	}
}

func fallbackCodexModelIDs() []string {
	return []string{
		"gpt-5.6-sol",
		"gpt-5.6-terra",
		"gpt-5.6-luna",
		"gpt-5.5",
		"gpt-5.3-codex-spark",
	}
}

func codexCatalogCandidateModelIDs(discovered []string, hidden []string) []string {
	candidates := append([]string(nil), discovered...)
	candidates = append(candidates, fallbackCodexModelIDs()...)
	candidates = normalizeCodexCatalogModelNames(candidates)
	if len(hidden) == 0 {
		return candidates
	}

	hiddenSet := make(map[string]struct{}, len(hidden))
	for _, modelName := range normalizeCodexCatalogModelNames(hidden) {
		hiddenSet[modelName] = struct{}{}
	}
	visible := make([]string, 0, len(candidates))
	for _, modelName := range candidates {
		if _, isHidden := hiddenSet[modelName]; isHidden {
			continue
		}
		visible = append(visible, modelName)
	}
	return visible
}

func codexCatalogModelsAfterFailedPromotion(currentModels string, hidden []string) []string {
	hiddenSet := make(map[string]struct{}, len(hidden))
	for _, modelName := range normalizeCodexCatalogModelNames(hidden) {
		hiddenSet[modelName] = struct{}{}
	}

	visible := make([]string, 0)
	for _, modelName := range normalizeCodexCatalogModelNames(strings.Split(currentModels, ",")) {
		if _, isHidden := hiddenSet[modelName]; isHidden {
			continue
		}
		visible = append(visible, modelName)
	}
	return filterKnownRetiredCodexModels(visible)
}

func readOptionMapValue(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(common.OptionMap[key])
}

func loadCodexCatalogPolicy() codexCatalogPolicy {
	policy := defaultCodexCatalogPolicy()

	if value := readOptionMapValue(codexCatalogDefaultModelOptionKey); value != "" {
		policy.DefaultModel = value
	}

	if raw := readOptionMapValue(codexCatalogDenylistOptionKey); raw != "" {
		var denylist []string
		if err := common.UnmarshalJsonStr(raw, &denylist); err == nil && len(denylist) > 0 {
			policy.Denylist = normalizeCodexCatalogModelNames(denylist)
		}
	}

	if raw := readOptionMapValue(codexCatalogOverridesOptionKey); raw != "" {
		var overrides map[string]CodexCatalogMetadata
		if err := common.UnmarshalJsonStr(raw, &overrides); err == nil && len(overrides) > 0 {
			for modelName, meta := range overrides {
				policy.Overrides[strings.TrimSpace(modelName)] = meta
			}
		}
	}

	return policy
}

func isDeniedCodexModel(modelName string, policy codexCatalogPolicy) bool {
	for _, denied := range policy.Denylist {
		if denied == modelName {
			return true
		}
	}
	return false
}

func normalizeCodexDiscoveryError(statusCode int, payload codexDiscoveryResponse, bodyText string) error {
	if payload.Detail != "" {
		return fmt.Errorf("codex discovery failed: status=%d detail=%s", statusCode, payload.Detail)
	}
	if payload.Error != nil && strings.TrimSpace(payload.Error.Message) != "" {
		return fmt.Errorf("codex discovery failed: status=%d error=%s", statusCode, strings.TrimSpace(payload.Error.Message))
	}
	bodyText = strings.TrimSpace(bodyText)
	if bodyText == "" {
		bodyText = http.StatusText(statusCode)
	}
	return fmt.Errorf("codex discovery failed: status=%d body=%s", statusCode, bodyText)
}

func normalizeCodexDiscoveryResult(items []codexDiscoveryItem) codexDiscoveryResult {
	result := codexDiscoveryResult{
		Models: make([]string, 0, len(items)),
		Hidden: make([]string, 0),
		Items:  make(map[string]codexDiscoveryItem, len(items)),
	}
	visibleSeen := make(map[string]struct{}, len(items))
	hiddenSeen := make(map[string]struct{})
	for _, item := range items {
		slug := strings.TrimSpace(item.Slug)
		if slug == "" {
			continue
		}
		visibility := strings.ToLower(strings.TrimSpace(item.Visibility))
		if visibility != "" && visibility != "list" {
			if _, ok := hiddenSeen[slug]; !ok {
				hiddenSeen[slug] = struct{}{}
				result.Hidden = append(result.Hidden, slug)
			}
			continue
		}
		if _, ok := visibleSeen[slug]; ok {
			continue
		}
		visibleSeen[slug] = struct{}{}
		result.Models = append(result.Models, slug)
		item.Slug = slug
		item.Visibility = visibility
		result.Items[slug] = item
	}
	if len(result.Hidden) > 0 {
		hiddenSet := make(map[string]struct{}, len(result.Hidden))
		for _, slug := range result.Hidden {
			hiddenSet[slug] = struct{}{}
			delete(result.Items, slug)
		}
		visible := result.Models[:0]
		for _, slug := range result.Models {
			if _, hidden := hiddenSet[slug]; !hidden {
				visible = append(visible, slug)
			}
		}
		result.Models = visible
	}
	return result
}

func resolveCodexDiscoveryClientVersion() string {
	if explicit := strings.TrimSpace(common.GetEnvOrDefaultString(codexCatalogDiscoveryClientVersionEnv, "")); explicit != "" {
		return explicit
	}

	output, err := exec.Command("codex", "--version").CombinedOutput()
	if err == nil {
		if match := codexCatalogVersionPattern.FindString(string(output)); match != "" {
			return match
		}
	}

	common.SysLog(fmt.Sprintf("codex catalog: using fallback discovery client version %s because %s could not be resolved", codexCatalogDefaultClientVersion, codexCatalogDiscoveryClientVersionHint))
	return codexCatalogDefaultClientVersion
}

func resolveCodexDiscoveryBaseURL(channel *model.Channel) string {
	baseURL := strings.TrimRight(strings.TrimSpace(constant.ChannelBaseURLs[constant.ChannelTypeCodex]), "/")
	if channel != nil && strings.TrimSpace(channel.GetBaseURL()) != "" {
		baseURL = strings.TrimRight(strings.TrimSpace(channel.GetBaseURL()), "/")
	}
	if baseURL == "" {
		baseURL = "https://chatgpt.com"
	}
	if strings.HasSuffix(baseURL, "/backend-api/codex") {
		return baseURL
	}
	return baseURL + "/backend-api/codex"
}

func doCodexDiscoveryRequest(ctx context.Context, channel *model.Channel, clientVersion string) (codexDiscoveryResult, error) {
	if channel == nil {
		return codexDiscoveryResult{}, errors.New("codex discovery: nil channel")
	}
	if channel.Type != constant.ChannelTypeCodex {
		return codexDiscoveryResult{}, fmt.Errorf("codex discovery: invalid channel type %d", channel.Type)
	}

	oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(channel.Key))
	if err != nil {
		return codexDiscoveryResult{}, err
	}
	accessToken := strings.TrimSpace(oauthKey.AccessToken)
	accountID := strings.TrimSpace(oauthKey.AccountID)
	if accessToken == "" || accountID == "" {
		return codexDiscoveryResult{}, errors.New("codex discovery: access_token/account_id are required")
	}

	client, err := NewProxyHttpClient(channel.GetSetting().Proxy)
	if err != nil {
		return codexDiscoveryResult{}, err
	}

	baseURL := resolveCodexDiscoveryBaseURL(channel)
	requestURL := fmt.Sprintf("%s/models?client_version=%s", baseURL, clientVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return codexDiscoveryResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("chatgpt-account-id", accountID)
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return codexDiscoveryResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return codexDiscoveryResult{}, err
	}

	var payload codexDiscoveryResponse
	decodeErr := common.Unmarshal(body, &payload)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return codexDiscoveryResult{}, newCodexUpstreamAuthError("codex discovery", resp.StatusCode, body)
		}
		if decodeErr != nil {
			return codexDiscoveryResult{}, fmt.Errorf("codex discovery returned invalid JSON: %w", decodeErr)
		}
		return codexDiscoveryResult{}, normalizeCodexDiscoveryError(resp.StatusCode, payload, string(body))
	}
	if decodeErr != nil {
		return codexDiscoveryResult{}, fmt.Errorf("codex discovery returned invalid JSON: %w", decodeErr)
	}

	result := normalizeCodexDiscoveryResult(payload.Models)
	if len(result.Models) == 0 && len(result.Hidden) == 0 {
		return codexDiscoveryResult{}, errors.New("codex discovery returned an empty model list")
	}
	return result, nil
}

func discoverCodexModels(ctx context.Context, channel *model.Channel) (codexDiscoveryResult, string, error) {
	clientVersion := resolveCodexDiscoveryClientVersion()
	result, err := doCodexDiscoveryRequest(ctx, channel, clientVersion)
	if err == nil {
		return result, clientVersion, nil
	}

	if channel != nil && channel.Id > 0 {
		refreshCtx, cancel := context.WithTimeout(ctx, codexCatalogDefaultDiscoveryTimeout)
		defer cancel()
		if _, refreshedChannel, refreshErr := RefreshCodexChannelCredential(refreshCtx, channel.Id, CodexCredentialRefreshOptions{ResetCaches: false}); refreshErr == nil {
			result, retryErr := doCodexDiscoveryRequest(ctx, refreshedChannel, clientVersion)
			if retryErr == nil {
				return result, clientVersion, nil
			}
			err = retryErr
		} else if issue := ClassifyCodexCredentialIssue(refreshErr, 0); issue.IsAuth {
			if healthErr := RecordCodexCredentialIssue(channel, issue); healthErr != nil {
				err = errors.Join(refreshErr, fmt.Errorf("failed to persist Codex auth health: %w", healthErr))
			} else {
				err = refreshErr
			}
		}
	}
	return codexDiscoveryResult{}, clientVersion, err
}

func DiscoverCodexModelIDs(ctx context.Context, channel *model.Channel) ([]string, string, error) {
	result, clientVersion, err := discoverCodexModels(ctx, channel)
	return result.Models, clientVersion, err
}

func ListCachedCodexDiscoveredModelIDs(channelID int) []string {
	if !codexCatalogStorageReady() {
		return nil
	}

	snapshot, err := model.GetLatestCodexCatalogSnapshot(channelID)
	if err == nil && snapshot != nil && strings.TrimSpace(snapshot.Snapshot) != "" {
		var items []codexDiscoveryItem
		if common.UnmarshalJsonStr(snapshot.Snapshot, &items) == nil {
			models := filterKnownRetiredCodexModels(normalizeCodexDiscoveryResult(items).Models)
			if len(models) > 0 {
				return models
			}
		}
	}

	candidates, err := model.GetCodexCatalogCandidatesByChannel(channelID)
	if err != nil {
		return nil
	}
	models := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		modelName := strings.TrimSpace(candidate.ModelName)
		if modelName == "" {
			continue
		}
		models = append(models, modelName)
	}
	return filterKnownRetiredCodexModels(models)
}

func ListPromotedCodexModelIDs(channelID int) []string {
	if !codexCatalogStorageReady() {
		return nil
	}

	candidates, err := model.GetPromotedCodexCatalogCandidatesByChannel(channelID)
	if err != nil {
		return nil
	}
	models := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		modelName := strings.TrimSpace(candidate.ModelName)
		if modelName == "" {
			continue
		}
		models = append(models, modelName)
	}
	return normalizeCodexCatalogModelNames(models)
}

func FetchCodexModelIDsForAdmin(ctx context.Context, channel *model.Channel) ([]string, error) {
	models, _, err := DiscoverCodexModelIDs(ctx, channel)
	if err == nil {
		return models, nil
	}

	if channel != nil {
		if cached := ListCachedCodexDiscoveredModelIDs(channel.Id); len(cached) > 0 {
			common.SysLog(fmt.Sprintf("codex catalog: admin discovery fallback to cached snapshot for channel %d: %v", channel.Id, err))
			return cached, nil
		}
	}
	common.SysLog(fmt.Sprintf("codex catalog: admin discovery fallback to default model list: %v", err))
	return fallbackCodexModelIDs(), nil
}

func promotedCodexMetadataByModelName(modelNames []string) (map[string]CodexCatalogMetadata, error) {
	normalized := normalizeCodexCatalogModelNames(modelNames)
	if len(normalized) == 0 {
		return map[string]CodexCatalogMetadata{}, nil
	}
	if !codexCatalogStorageReady() {
		return map[string]CodexCatalogMetadata{}, nil
	}
	var candidates []*model.CodexCatalogCandidate
	if err := model.DB.Where("promoted = ? AND model_name IN ?", true, normalized).Find(&candidates).Error; err != nil {
		return nil, err
	}
	result := make(map[string]CodexCatalogMetadata, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		endpoints := parseCodexCatalogEndpoints(candidate.SupportedEndpoints)
		metadata := CodexCatalogMetadata{
			DisplayName:         strings.TrimSpace(candidate.DisplayName),
			Provider:            strings.TrimSpace(candidate.Provider),
			OwnedBy:             strings.TrimSpace(candidate.OwnedBy),
			EndpointPreference:  constant.EndpointType(strings.TrimSpace(candidate.EndpointPreference)),
			SupportedEndpoints:  endpoints,
			ContextWindowTokens: candidate.ContextWindowTokens,
			MaxTokens:           candidate.MaxTokens,
			MaxCompletionTokens: candidate.MaxCompletionTokens,
			BillingMode:         CodexCatalogBillingMode,
		}
		var discoveryItem codexDiscoveryItem
		if err := common.UnmarshalJsonStr(candidate.DiscoveryMetadata, &discoveryItem); err == nil {
			activeContext, activeOutput := codexDiscoveryLimits(discoveryItem)
			if activeContext > 0 {
				metadata.ContextWindowTokens = activeContext
				metadata.MaxTokens = activeContext
			}
			if activeOutput > 0 {
				metadata.MaxCompletionTokens = activeOutput
			} else if reference, ok := CodexOpenAIReferencePriceForModel(candidate.ModelName); ok {
				metadata.MaxCompletionTokens = reference.MaxCompletionTokens
			}
		}
		var sourceMetadata CodexCatalogMetadata
		if err := common.UnmarshalJsonStr(candidate.SourceMetadata, &sourceMetadata); err == nil {
			metadata.SupportedReasoningEfforts = append([]string(nil), sourceMetadata.SupportedReasoningEfforts...)
			metadata.Capabilities = append([]string(nil), sourceMetadata.Capabilities...)
		}
		var overrideMetadata CodexCatalogMetadata
		if err := common.UnmarshalJsonStr(candidate.OverrideMetadata, &overrideMetadata); err == nil {
			if len(overrideMetadata.SupportedReasoningEfforts) > 0 {
				metadata.SupportedReasoningEfforts = append([]string(nil), overrideMetadata.SupportedReasoningEfforts...)
			}
			if len(overrideMetadata.Capabilities) > 0 {
				metadata.Capabilities = append([]string(nil), overrideMetadata.Capabilities...)
			}
		}
		result[candidate.ModelName] = metadata
	}
	return result, nil
}

func parseCodexCatalogEndpoints(raw string) []constant.EndpointType {
	var values []string
	if err := common.UnmarshalJsonStr(raw, &values); err != nil {
		return nil
	}
	endpoints := make([]constant.EndpointType, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		endpoints = append(endpoints, constant.EndpointType(value))
	}
	return endpoints
}

func CodexPromotedMetadataByModelName(modelNames []string) (map[string]CodexCatalogMetadata, error) {
	return promotedCodexMetadataByModelName(modelNames)
}

func codexCatalogStorageReady() bool {
	if model.DB == nil {
		return false
	}
	sqlDB, err := model.DB.DB()
	if err != nil {
		return false
	}
	if err := sqlDB.Ping(); err != nil {
		return false
	}
	return model.DB.Migrator().HasTable(&model.CodexCatalogCandidate{}) &&
		model.DB.Migrator().HasTable(&model.CodexCatalogSnapshot{})
}

func codexCatalogSignature(models []string, policy codexCatalogPolicy, discoveryItems map[string]codexDiscoveryItem) (string, error) {
	normalized := normalizeCodexCatalogModelNames(models)
	sort.Strings(normalized)
	canonicalDiscovery := make([]codexDiscoveryItem, 0, len(normalized))
	for _, modelName := range normalized {
		item := discoveryItems[modelName]
		item.Slug = modelName
		item.Visibility = ""
		canonicalDiscovery = append(canonicalDiscovery, item)
	}
	discoveryPayload, err := common.Marshal(canonicalDiscovery)
	if err != nil {
		return "", err
	}
	policyPayload, err := common.Marshal(policy)
	if err != nil {
		return "", err
	}
	payload := strings.Join(normalized, "\n") +
		"\n--validation-contract--\n" + codexCatalogValidationContractVersion +
		"\n--oauth-discovery--\n" + string(discoveryPayload) +
		"\n--policy--\n" + string(policyPayload)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(payload))), nil
}

func codexSourceMetadataByModelName(modelNames []string) map[string]CodexCatalogMetadata {
	result := make(map[string]CodexCatalogMetadata, len(modelNames))

	var modelRows []model.Model
	if err := model.DB.Where("model_name IN ?", normalizeCodexCatalogModelNames(modelNames)).Find(&modelRows).Error; err == nil {
		for _, row := range modelRows {
			meta := result[row.ModelName]
			if desc := strings.TrimSpace(row.Description); desc != "" {
				meta.DisplayName = desc
			}
			endpoints := model.GetModelSupportEndpointTypes(row.ModelName)
			if len(endpoints) > 0 {
				meta.SupportedEndpoints = endpoints
			}
			result[row.ModelName] = meta
		}
	}

	for _, pricing := range model.GetPricing() {
		if _, ok := result[pricing.ModelName]; !ok {
			continue
		}
		meta := result[pricing.ModelName]
		if desc := strings.TrimSpace(pricing.Description); desc != "" {
			meta.DisplayName = desc
		}
		if owner := strings.TrimSpace(pricing.OwnerBy); owner != "" {
			meta.OwnedBy = owner
		}
		if len(pricing.SupportedEndpointTypes) > 0 {
			meta.SupportedEndpoints = pricing.SupportedEndpointTypes
		}
		result[pricing.ModelName] = meta
	}

	return result
}

func codexFallbackMetadataForModel(modelName string) CodexCatalogMetadata {
	meta := CodexCatalogMetadata{
		DisplayName:         modelName,
		Provider:            "OpenAI Codex",
		OwnedBy:             "codex",
		EndpointPreference:  constant.EndpointTypeOpenAIResponse,
		SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
		ContextWindowTokens: 272000,
		MaxTokens:           272000,
		BillingMode:         CodexCatalogBillingMode,
	}
	normalized := strings.ToLower(modelName)
	switch {
	case strings.Contains(normalized, "spark"):
		meta.ContextWindowTokens = 128000
		meta.MaxTokens = 128000
	}
	if reference, ok := CodexOpenAIReferencePriceForModel(modelName); ok {
		meta.MaxCompletionTokens = reference.MaxCompletionTokens
	}
	return meta
}

func codexDiscoveryLimits(item codexDiscoveryItem) (int, int) {
	contextLength := item.ContextWindow
	if contextLength <= 0 {
		contextLength = item.MaxContextWindow
	}
	maxCompletionTokens := item.MaxCompletionTokens
	if maxCompletionTokens <= 0 {
		maxCompletionTokens = item.MaxOutputTokens
	}
	return contextLength, maxCompletionTokens
}

func applyCodexDiscoveryLimits(meta *CodexCatalogMetadata, item codexDiscoveryItem) {
	contextLength, maxCompletionTokens := codexDiscoveryLimits(item)
	if contextLength > 0 {
		meta.ContextWindowTokens = contextLength
		meta.MaxTokens = contextLength
	}
	if maxCompletionTokens > 0 {
		meta.MaxCompletionTokens = maxCompletionTokens
	}
}

func mergeCodexCatalogMetadata(modelName string, source CodexCatalogMetadata, override CodexCatalogMetadata) CodexCatalogMetadata {
	meta := codexFallbackMetadataForModel(modelName)

	if source.DisplayName != "" {
		meta.DisplayName = source.DisplayName
	}
	if source.Provider != "" {
		meta.Provider = source.Provider
	}
	if source.OwnedBy != "" {
		meta.OwnedBy = source.OwnedBy
	}
	if len(source.SupportedEndpoints) > 0 {
		meta.SupportedEndpoints = append([]constant.EndpointType(nil), source.SupportedEndpoints...)
	}
	if source.EndpointPreference != "" {
		meta.EndpointPreference = source.EndpointPreference
	}
	if source.ContextWindowTokens > 0 {
		meta.ContextWindowTokens = source.ContextWindowTokens
	}
	if source.MaxTokens > 0 {
		meta.MaxTokens = source.MaxTokens
	}
	if source.MaxCompletionTokens > 0 {
		meta.MaxCompletionTokens = source.MaxCompletionTokens
	}
	if len(source.SupportedReasoningEfforts) > 0 {
		meta.SupportedReasoningEfforts = append([]string(nil), source.SupportedReasoningEfforts...)
	}
	if len(source.Capabilities) > 0 {
		meta.Capabilities = append([]string(nil), source.Capabilities...)
	}

	if override.DisplayName != "" {
		meta.DisplayName = override.DisplayName
	}
	if override.Provider != "" {
		meta.Provider = override.Provider
	}
	if override.OwnedBy != "" {
		meta.OwnedBy = override.OwnedBy
	}
	if len(override.SupportedEndpoints) > 0 {
		meta.SupportedEndpoints = append([]constant.EndpointType(nil), override.SupportedEndpoints...)
	}
	if override.EndpointPreference != "" {
		meta.EndpointPreference = override.EndpointPreference
	}
	if override.ContextWindowTokens > 0 {
		meta.ContextWindowTokens = override.ContextWindowTokens
	}
	if override.MaxTokens > 0 {
		meta.MaxTokens = override.MaxTokens
	}
	if override.MaxCompletionTokens > 0 {
		meta.MaxCompletionTokens = override.MaxCompletionTokens
	}
	if len(override.SupportedReasoningEfforts) > 0 {
		meta.SupportedReasoningEfforts = append([]string(nil), override.SupportedReasoningEfforts...)
	}
	if len(override.Capabilities) > 0 {
		meta.Capabilities = append([]string(nil), override.Capabilities...)
	}

	if meta.EndpointPreference == "" {
		meta.EndpointPreference = constant.EndpointTypeOpenAIResponse
	}
	if len(meta.SupportedEndpoints) == 0 {
		meta.SupportedEndpoints = []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI}
	}
	if meta.Provider == "" {
		meta.Provider = "OpenAI Codex"
	}
	if meta.OwnedBy == "" {
		meta.OwnedBy = "codex"
	}
	if meta.DisplayName == "" {
		meta.DisplayName = modelName
	}
	meta.BillingMode = CodexCatalogBillingMode
	if meta.MaxTokens <= 0 {
		meta.MaxTokens = meta.ContextWindowTokens
	}
	if meta.ContextWindowTokens <= 0 {
		meta.ContextWindowTokens = meta.MaxTokens
	}
	return meta
}

func validateCodexCandidate(ctx context.Context, channel *model.Channel, modelName string) (string, error) {
	if channel == nil {
		return "", errors.New("codex validation: nil channel")
	}
	oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(channel.Key))
	if err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(oauthKey.AccessToken)
	accountID := strings.TrimSpace(oauthKey.AccountID)
	if accessToken == "" || accountID == "" {
		return "", errors.New("codex validation: access_token/account_id are required")
	}

	client, err := NewProxyHttpClient(channel.GetSetting().Proxy)
	if err != nil {
		return "", err
	}

	requestURL := resolveCodexDiscoveryBaseURL(channel) + "/responses"
	payload := map[string]any{
		"model":        modelName,
		"instructions": "",
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "Reply only Ok."},
				},
			},
		},
		"store":  false,
		"stream": true,
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}

	doRequest := func(currentAccessToken string) (string, int, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(string(body)))
		if reqErr != nil {
			return "", 0, reqErr
		}
		req.Header.Set("Authorization", "Bearer "+currentAccessToken)
		req.Header.Set("chatgpt-account-id", accountID)
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", "codex_cli_rs")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, doErr := client.Do(req)
		if doErr != nil {
			return "", 0, doErr
		}
		defer resp.Body.Close()

		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", resp.StatusCode, readErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return strings.TrimSpace(string(respBody)), resp.StatusCode, fmt.Errorf("codex validation failed: status=%d", resp.StatusCode)
		}
		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(contentType, "text/event-stream") || looksLikeCodexValidationStream(respBody) {
			text, streamErr := extractCodexValidationStreamText(respBody)
			return text, resp.StatusCode, streamErr
		}

		var response dto.OpenAIResponsesResponse
		if err := common.Unmarshal(respBody, &response); err != nil {
			return "", resp.StatusCode, err
		}
		return strings.TrimSpace(openaicompat.ExtractOutputTextFromResponses(&response)), resp.StatusCode, nil
	}

	text, statusCode, err := doRequest(accessToken)
	if err == nil {
		return text, nil
	}
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return text, err
	}

	refreshCtx, cancel := context.WithTimeout(ctx, codexCatalogDefaultValidationTimeout)
	defer cancel()
	updatedKey, _, refreshErr := RefreshCodexChannelCredential(refreshCtx, channel.Id, CodexCredentialRefreshOptions{ResetCaches: false})
	if refreshErr != nil {
		if issue := ClassifyCodexCredentialIssue(refreshErr, 0); issue.IsAuth {
			if healthErr := RecordCodexCredentialIssue(channel, issue); healthErr != nil {
				return text, errors.Join(refreshErr, fmt.Errorf("failed to persist Codex auth health: %w", healthErr))
			}
			return text, refreshErr
		}
		return text, err
	}
	text, _, err = doRequest(updatedKey.AccessToken)
	return text, err
}

func looksLikeCodexValidationStream(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	return bytes.HasPrefix(trimmed, []byte("event:")) || bytes.HasPrefix(trimmed, []byte("data:"))
}

func extractCodexValidationStreamText(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	var textBuilder strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
			Text  string `json:"text"`
		}
		if err := common.Unmarshal([]byte(data), &event); err != nil {
			return "", fmt.Errorf("codex validation returned invalid SSE JSON: %w", err)
		}
		switch event.Type {
		case "response.output_text.delta":
			textBuilder.WriteString(event.Delta)
		case "response.output_text.done":
			if textBuilder.Len() == 0 {
				textBuilder.WriteString(event.Text)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		return "", errors.New("codex validation stream returned no output text")
	}
	return text, nil
}

func isExpectedCodexValidationOutput(output string) bool {
	normalized := strings.TrimSpace(output)
	normalized = strings.TrimSuffix(normalized, ".")
	return strings.EqualFold(strings.TrimSpace(normalized), codexCatalogDefaultReply)
}

func syncCodexChannelModels(channel *model.Channel, promotedModels []string, allowEmpty bool) error {
	if channel == nil {
		return errors.New("codex catalog sync: nil channel")
	}
	promotedModels = normalizeCodexCatalogModelNames(promotedModels)
	if len(promotedModels) == 0 && !allowEmpty {
		return errors.New("codex catalog sync: refusing to replace channel models with an empty promoted catalog")
	}
	newModels := strings.Join(promotedModels, ",")
	if strings.TrimSpace(channel.Models) == newModels {
		return nil
	}

	channel.Models = newModels
	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("models", channel.Models).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(promotedModels) == 0 {
		if err := tx.Where("channel_id = ?", channel.Id).Delete(&model.Ability{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if err := channel.UpdateAbilities(tx); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func prioritizeCodexDefaultModel(modelNames []string, defaultModel string) []string {
	modelNames = normalizeCodexCatalogModelNames(modelNames)
	defaultModel = strings.TrimSpace(defaultModel)
	if defaultModel == "" {
		return modelNames
	}
	for i, modelName := range modelNames {
		if modelName != defaultModel {
			continue
		}
		if i == 0 {
			return modelNames
		}
		reordered := []string{modelName}
		reordered = append(reordered, append(modelNames[:i], modelNames[i+1:]...)...)
		return reordered
	}
	return modelNames
}

func SyncCodexCatalog(ctx context.Context, channelID int) (*CodexCatalogSyncResult, error) {
	codexCatalogMutex.Lock()
	defer codexCatalogMutex.Unlock()

	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if channel.Type != constant.ChannelTypeCodex {
		return nil, fmt.Errorf("codex catalog sync: channel %d is not codex", channelID)
	}

	discovery, clientVersion, err := discoverCodexModels(ctx, channel)
	if err != nil {
		return nil, err
	}
	discoveredModels := codexCatalogCandidateModelIDs(discovery.Models, discovery.Hidden)
	hiddenModels := make(map[string]struct{}, len(discovery.Hidden))
	for _, modelName := range discovery.Hidden {
		hiddenModels[modelName] = struct{}{}
	}

	policy := loadCodexCatalogPolicy()
	signature, err := codexCatalogSignature(discoveredModels, policy, discovery.Items)
	if err != nil {
		return nil, err
	}
	latestSnapshot, _ := model.GetLatestCodexCatalogSnapshot(channelID)
	if latestSnapshot != nil && latestSnapshot.SnapshotHash == signature {
		existingCandidates, _ := model.GetCodexCatalogCandidatesByChannel(channelID)
		promotedModels := ListPromotedCodexModelIDs(channelID)
		if len(existingCandidates) == 0 || len(promotedModels) == 0 {
			latestSnapshot = nil
		} else {
			promotedModels = prioritizeCodexDefaultModel(promotedModels, policy.DefaultModel)
			if err := syncCodexChannelModels(channel, promotedModels, false); err != nil {
				return nil, err
			}
			return &CodexCatalogSyncResult{
				ChannelID:  channelID,
				Discovered: discoveredModels,
				Promoted:   promotedModels,
				Changed:    false,
			}, nil
		}
	}

	snapshotItems := make([]codexDiscoveryItem, 0, len(discoveredModels))
	for _, modelName := range discoveredModels {
		item := discovery.Items[modelName]
		item.Slug = modelName
		snapshotItems = append(snapshotItems, item)
	}
	snapshotPayload, err := common.Marshal(snapshotItems)
	if err != nil {
		return nil, err
	}
	snapshot := &model.CodexCatalogSnapshot{
		ChannelID:     channelID,
		SnapshotHash:  signature,
		ClientVersion: clientVersion,
		ModelCount:    len(discoveredModels),
		Snapshot:      string(snapshotPayload),
	}
	sourceMetadata := codexSourceMetadataByModelName(discoveredModels)
	now := common.GetTimestamp()
	seen := make(map[string]struct{}, len(discoveredModels))
	validatedCount := 0
	promotedModels := make([]string, 0, len(discoveredModels))

	for _, modelName := range discoveredModels {
		seen[modelName] = struct{}{}
		sourceMeta := sourceMetadata[modelName]
		overrideMeta := policy.Overrides[modelName]
		mergedMeta := mergeCodexCatalogMetadata(modelName, sourceMeta, overrideMeta)
		applyCodexDiscoveryLimits(&mergedMeta, discovery.Items[modelName])

		candidate, findErr := model.FindCodexCatalogCandidate(channelID, modelName)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, findErr
		}
		if candidate == nil {
			candidate = &model.CodexCatalogCandidate{
				ChannelID: channelID,
				ModelName: modelName,
			}
		}

		discoveryItem := discovery.Items[modelName]
		discoveryItem.Slug = modelName
		discoveryPayload, _ := common.Marshal(discoveryItem)
		sourcePayload, _ := common.Marshal(sourceMeta)
		overridePayload, _ := common.Marshal(overrideMeta)

		candidate.DiscoveryHash = signature
		candidate.DiscoveryMetadata = string(discoveryPayload)
		candidate.SourceMetadata = string(sourcePayload)
		candidate.OverrideMetadata = string(overridePayload)
		candidate.DisplayName = mergedMeta.DisplayName
		candidate.Provider = mergedMeta.Provider
		candidate.OwnedBy = mergedMeta.OwnedBy
		candidate.EndpointPreference = string(mergedMeta.EndpointPreference)
		supportedEndpoints := make([]string, 0, len(mergedMeta.SupportedEndpoints))
		for _, endpointType := range mergedMeta.SupportedEndpoints {
			supportedEndpoints = append(supportedEndpoints, string(endpointType))
		}
		endpointPayload, _ := common.Marshal(supportedEndpoints)
		candidate.SupportedEndpoints = string(endpointPayload)
		candidate.ContextWindowTokens = mergedMeta.ContextWindowTokens
		candidate.MaxTokens = mergedMeta.MaxTokens
		candidate.MaxCompletionTokens = mergedMeta.MaxCompletionTokens
		candidate.LastDiscoveredTime = now
		candidate.LastSeenTime = now
		candidate.Status = model.CodexCatalogStatusEnriched
		candidate.ValidationState = ""
		candidate.ValidationError = ""
		candidate.ValidationOutput = ""
		candidate.Promoted = false

		if isDeniedCodexModel(modelName, policy) {
			candidate.Status = model.CodexCatalogStatusRejected
			candidate.ValidationState = "denied"
			candidate.ValidationError = "model denied by local policy"
		} else {
			probeCtx, cancel := context.WithTimeout(ctx, codexCatalogDefaultValidationTimeout)
			output, validateErr := validateCodexCandidate(probeCtx, channel, modelName)
			cancel()

			candidate.LastValidatedTime = common.GetTimestamp()
			candidate.ValidationOutput = strings.TrimSpace(output)
			if validateErr != nil {
				candidate.Status = model.CodexCatalogStatusRejected
				candidate.ValidationState = "failed"
				candidate.ValidationError = validateErr.Error()
			} else if !isExpectedCodexValidationOutput(output) {
				candidate.Status = model.CodexCatalogStatusRejected
				candidate.ValidationState = "unexpected_output"
				candidate.ValidationError = fmt.Sprintf("expected %q, got %q", codexCatalogDefaultReply, output)
			} else {
				candidate.Status = model.CodexCatalogStatusPromoted
				candidate.ValidationState = "ok"
				candidate.Promoted = true
				candidate.LastPromotedTime = common.GetTimestamp()
				promotedModels = append(promotedModels, modelName)
				validatedCount++
			}
		}

		if err := candidate.Save(); err != nil {
			return nil, err
		}
	}

	existingCandidates, err := model.GetCodexCatalogCandidatesByChannel(channelID)
	if err != nil {
		return nil, err
	}
	for _, candidate := range existingCandidates {
		if candidate == nil {
			continue
		}
		if _, ok := seen[candidate.ModelName]; ok {
			continue
		}
		candidate.Promoted = false
		candidate.Status = model.CodexCatalogStatusRejected
		if _, hidden := hiddenModels[candidate.ModelName]; hidden {
			candidate.ValidationState = "hidden_upstream"
			candidate.ValidationError = "model hidden by upstream visibility policy"
		} else {
			candidate.ValidationState = "no_longer_discovered"
			candidate.ValidationError = "model disappeared from dynamic discovery"
		}
		if err := candidate.Save(); err != nil {
			return nil, err
		}
	}

	promotedModels = prioritizeCodexDefaultModel(promotedModels, policy.DefaultModel)
	modelsToSync := promotedModels
	allowEmptyCatalog := false
	if len(modelsToSync) == 0 && len(discovery.Hidden) > 0 {
		// Visibility is authoritative even when every visible/fallback candidate fails validation.
		modelsToSync = codexCatalogModelsAfterFailedPromotion(channel.Models, discovery.Hidden)
		allowEmptyCatalog = true
	}
	if err := syncCodexChannelModels(channel, modelsToSync, allowEmptyCatalog); err != nil {
		return nil, err
	}
	if err := snapshot.Save(); err != nil {
		return nil, err
	}

	model.InvalidatePricingCache()
	if common.MemoryCacheEnabled {
		model.InitChannelCache()
	}

	return &CodexCatalogSyncResult{
		ChannelID:      channelID,
		Discovered:     discoveredModels,
		Promoted:       promotedModels,
		Changed:        true,
		ValidatedCount: validatedCount,
	}, nil
}
