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
	codexCatalogDefaultDiscoveryTimeout    = 20 * time.Second
	codexCatalogDefaultValidationTimeout   = 30 * time.Second
	codexCatalogDefaultModelOptionKey      = "CodexCatalogDefaultModel"
	codexCatalogDenylistOptionKey          = "CodexCatalogDenylist"
	codexCatalogOverridesOptionKey         = "CodexCatalogMetadataOverrides"
	codexCatalogDiscoveryClientVersionEnv  = "CODEX_DISCOVERY_CLIENT_VERSION"
	codexCatalogDiscoveryClientVersionHint = "codex --version"
)

var (
	codexCatalogVersionPattern = regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)
	codexCatalogMutex          sync.Mutex
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
}

type codexCatalogPolicy struct {
	DefaultModel string                          `json:"default_model,omitempty"`
	Denylist     []string                        `json:"denylist,omitempty"`
	Overrides    map[string]CodexCatalogMetadata `json:"overrides,omitempty"`
}

type codexDiscoveryItem struct {
	Slug string `json:"slug"`
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

func officialGPT56CodexMetadata(displayName string, reasoningEfforts []string) CodexCatalogMetadata {
	return CodexCatalogMetadata{
		DisplayName:               displayName,
		Provider:                  "OpenAI Codex",
		OwnedBy:                   "codex",
		EndpointPreference:        constant.EndpointTypeOpenAIResponse,
		SupportedEndpoints:        []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
		ContextWindowTokens:       1050000,
		MaxTokens:                 1050000,
		MaxCompletionTokens:       128000,
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
		DefaultModel: "gpt-5.4",
		Denylist: []string{
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
				MaxCompletionTokens: 128000,
			},
			"gpt-5.4": {
				DisplayName:         "OpenAI Codex GPT-5.4",
				Provider:            "OpenAI Codex",
				OwnedBy:             "codex",
				EndpointPreference:  constant.EndpointTypeOpenAIResponse,
				SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
				ContextWindowTokens: 1050000,
				MaxTokens:           1050000,
				MaxCompletionTokens: 128000,
			},
			"gpt-5.4-mini": {
				DisplayName:         "OpenAI Codex GPT-5.4-mini",
				Provider:            "OpenAI Codex",
				OwnedBy:             "codex",
				EndpointPreference:  constant.EndpointTypeOpenAIResponse,
				SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
				ContextWindowTokens: 272000,
				MaxTokens:           272000,
				MaxCompletionTokens: 64000,
			},
			"gpt-5.3-codex-spark": {
				DisplayName:         "OpenAI Codex GPT-5.3-codex-spark",
				Provider:            "OpenAI Codex",
				OwnedBy:             "codex",
				EndpointPreference:  constant.EndpointTypeOpenAIResponse,
				SupportedEndpoints:  []constant.EndpointType{constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAI},
				ContextWindowTokens: 128000,
				MaxTokens:           128000,
				MaxCompletionTokens: 32000,
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
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.3-codex-spark",
	}
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

func normalizeCodexModelIDs(items []codexDiscoveryItem) []string {
	models := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		slug := strings.TrimSpace(item.Slug)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		models = append(models, slug)
	}
	return models
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

func doCodexDiscoveryRequest(ctx context.Context, channel *model.Channel, clientVersion string) ([]string, error) {
	if channel == nil {
		return nil, errors.New("codex discovery: nil channel")
	}
	if channel.Type != constant.ChannelTypeCodex {
		return nil, fmt.Errorf("codex discovery: invalid channel type %d", channel.Type)
	}

	oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(channel.Key))
	if err != nil {
		return nil, err
	}
	accessToken := strings.TrimSpace(oauthKey.AccessToken)
	accountID := strings.TrimSpace(oauthKey.AccountID)
	if accessToken == "" || accountID == "" {
		return nil, errors.New("codex discovery: access_token/account_id are required")
	}

	client, err := NewProxyHttpClient(channel.GetSetting().Proxy)
	if err != nil {
		return nil, err
	}

	baseURL := resolveCodexDiscoveryBaseURL(channel)
	requestURL := fmt.Sprintf("%s/models?client_version=%s", baseURL, clientVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("chatgpt-account-id", accountID)
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload codexDiscoveryResponse
	decodeErr := common.Unmarshal(body, &payload)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, newCodexUpstreamAuthError("codex discovery", resp.StatusCode, body)
		}
		if decodeErr != nil {
			return nil, fmt.Errorf("codex discovery returned invalid JSON: %w", decodeErr)
		}
		return nil, normalizeCodexDiscoveryError(resp.StatusCode, payload, string(body))
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("codex discovery returned invalid JSON: %w", decodeErr)
	}

	models := normalizeCodexModelIDs(payload.Models)
	if len(models) == 0 {
		return nil, errors.New("codex discovery returned an empty model list")
	}
	return models, nil
}

func DiscoverCodexModelIDs(ctx context.Context, channel *model.Channel) ([]string, string, error) {
	clientVersion := resolveCodexDiscoveryClientVersion()
	models, err := doCodexDiscoveryRequest(ctx, channel, clientVersion)
	if err == nil {
		return models, clientVersion, nil
	}

	if channel != nil && channel.Id > 0 {
		refreshCtx, cancel := context.WithTimeout(ctx, codexCatalogDefaultDiscoveryTimeout)
		defer cancel()
		if _, refreshedChannel, refreshErr := RefreshCodexChannelCredential(refreshCtx, channel.Id, CodexCredentialRefreshOptions{ResetCaches: false}); refreshErr == nil {
			models, retryErr := doCodexDiscoveryRequest(ctx, refreshedChannel, clientVersion)
			if retryErr == nil {
				return models, clientVersion, nil
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
	return nil, clientVersion, err
}

func ListCachedCodexDiscoveredModelIDs(channelID int) []string {
	if !codexCatalogStorageReady() {
		return nil
	}

	snapshot, err := model.GetLatestCodexCatalogSnapshot(channelID)
	if err == nil && snapshot != nil && strings.TrimSpace(snapshot.Snapshot) != "" {
		var items []codexDiscoveryItem
		if common.UnmarshalJsonStr(snapshot.Snapshot, &items) == nil {
			models := normalizeCodexModelIDs(items)
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
	return normalizeCodexCatalogModelNames(models)
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

func codexCatalogSignature(models []string, policy codexCatalogPolicy) (string, error) {
	normalized := normalizeCodexCatalogModelNames(models)
	sort.Strings(normalized)
	policyPayload, err := common.Marshal(policy)
	if err != nil {
		return "", err
	}
	payload := strings.Join(normalized, "\n") + "\n--policy--\n" + string(policyPayload)
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
		MaxCompletionTokens: 128000,
	}
	normalized := strings.ToLower(modelName)
	switch {
	case strings.Contains(normalized, "mini"):
		meta.MaxCompletionTokens = 64000
	case strings.Contains(normalized, "spark"):
		meta.ContextWindowTokens = 128000
		meta.MaxTokens = 128000
		meta.MaxCompletionTokens = 32000
	}
	return meta
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
	if meta.MaxTokens <= 0 {
		meta.MaxTokens = meta.ContextWindowTokens
	}
	if meta.ContextWindowTokens <= 0 {
		meta.ContextWindowTokens = meta.MaxTokens
	}
	if meta.MaxCompletionTokens <= 0 {
		meta.MaxCompletionTokens = 128000
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
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
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

func syncCodexChannelModels(channel *model.Channel, promotedModels []string) error {
	if channel == nil {
		return errors.New("codex catalog sync: nil channel")
	}
	promotedModels = normalizeCodexCatalogModelNames(promotedModels)
	if len(promotedModels) == 0 {
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
	if err := channel.UpdateAbilities(tx); err != nil {
		tx.Rollback()
		return err
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

	discoveredModels, clientVersion, err := DiscoverCodexModelIDs(ctx, channel)
	if err != nil {
		return nil, err
	}

	policy := loadCodexCatalogPolicy()
	signature, err := codexCatalogSignature(discoveredModels, policy)
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
		snapshotItems = append(snapshotItems, codexDiscoveryItem{Slug: modelName})
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
	if err := snapshot.Save(); err != nil {
		return nil, err
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

		discoveryPayload, _ := common.Marshal(map[string]any{
			"model":          modelName,
			"client_version": clientVersion,
		})
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
			} else if !strings.EqualFold(strings.TrimSpace(output), codexCatalogDefaultReply) {
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
		candidate.ValidationState = "no_longer_discovered"
		candidate.ValidationError = "model disappeared from dynamic discovery"
		if err := candidate.Save(); err != nil {
			return nil, err
		}
	}

	promotedModels = prioritizeCodexDefaultModel(promotedModels, policy.DefaultModel)
	if err := syncCodexChannelModels(channel, promotedModels); err != nil {
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
