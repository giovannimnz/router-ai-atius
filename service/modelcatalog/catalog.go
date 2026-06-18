package modelcatalog

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

var knownModelOrder = map[string]int{
	"MiniMax-M2.7":           0,
	"MiniMax-M2.5-highspeed": 1,
	"MiniMax-M2.5":           2,
	"deepseek-v4-pro":        3,
	"deepseek-v4-flash":      4,
	"gpt-5.5":                5,
	"gpt-5.4":                6,
	"gpt-5.4-mini":           7,
	"gpt-5.3-codex-spark":    8,
	"MiniMax-M3":             9,
	"embo-01":                10,
	"text-embedding-3-large": 11,
	"text-embedding-3-small": 12,
}

var versionTokenPattern = regexp.MustCompile(`\d+(?:\.\d+)*`)

func EndpointTypeLabel(endpointType constant.EndpointType) string {
	switch endpointType {
	case constant.EndpointTypeOpenAI:
		return "OpenAI-Compatible"
	case constant.EndpointTypeAnthropic:
		return "Anthropic-Compatible"
	case constant.EndpointTypeEmbeddings:
		return "Embeddings"
	case constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAIResponseCompact:
		return "OpenAI-Responses"
	default:
		return string(endpointType)
	}
}

func EndpointTypeLabels(endpointTypes []constant.EndpointType) []string {
	labels := make([]string, 0, len(endpointTypes))
	seen := make(map[string]struct{}, len(endpointTypes))
	for _, endpointType := range endpointTypes {
		label := EndpointTypeLabel(endpointType)
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	return labels
}

func EndpointRoutes(endpointTypes []constant.EndpointType) map[string]string {
	routes := make(map[string]string, len(endpointTypes))
	for _, endpointType := range endpointTypes {
		if info, ok := common.GetDefaultEndpointInfo(endpointType); ok {
			routes[string(endpointType)] = info.Path
		}
	}
	if len(routes) == 0 {
		return nil
	}
	return routes
}

func HasEndpointType(endpointTypes []constant.EndpointType, wanted constant.EndpointType) bool {
	for _, endpointType := range endpointTypes {
		if endpointType == wanted {
			return true
		}
	}
	return false
}

func IsAnthropicCapable(entry dto.ModelCatalogEntry) bool {
	return HasEndpointType(entry.SupportedEndpointTypes, constant.EndpointTypeAnthropic)
}

func ChannelOwnerName(channelType int) string {
	apiType, success := common.ChannelType2APIType(channelType)
	if !success {
		return strings.ToLower(constant.GetChannelTypeName(channelType))
	}
	adaptor := relay.GetAdaptor(apiType)
	if adaptor == nil {
		return strings.ToLower(constant.GetChannelTypeName(channelType))
	}
	adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelType: channelType,
	}})
	if name := strings.TrimSpace(adaptor.GetChannelName()); name != "" {
		return name
	}
	return strings.ToLower(constant.GetChannelTypeName(channelType))
}

func PreferredOwnerNames(modelNames []string, groups []string) (map[string]string, error) {
	channelTypes, err := model.GetPreferredModelOwnerChannelTypes(modelNames, groups)
	if err != nil {
		return nil, err
	}
	ownerByChannelType := make(map[int]string)
	owners := make(map[string]string, len(channelTypes))
	for modelName, channelType := range channelTypes {
		owner, ok := ownerByChannelType[channelType]
		if !ok {
			owner = ChannelOwnerName(channelType)
			ownerByChannelType[channelType] = owner
		}
		if owner != "" {
			owners[modelName] = owner
		}
	}
	return owners, nil
}

func pricingProvenance(pricing model.Pricing) (string, bool) {
	if pricing.QuotaType == 1 {
		if _, ok := ratio_setting.GetModelPrice(pricing.ModelName, false); ok {
			return "model_price", false
		}
	}
	if _, ok, _ := ratio_setting.GetModelRatio(pricing.ModelName); ok {
		return "model_ratio", false
	}
	if strings.TrimSpace(pricing.BillingMode) == "tiered_expr" && strings.TrimSpace(pricing.BillingExpr) != "" {
		return "billing_expr", false
	}
	return "missing", true
}

func PublicTokenPrices(pricing model.Pricing) (float64, float64) {
	if pricing.QuotaType == 1 {
		return pricing.ModelPrice, pricing.ModelPrice
	}
	inputPrice := pricing.ModelRatio * 2
	outputPrice := inputPrice
	if pricing.CompletionRatio != 0 {
		outputPrice = inputPrice * pricing.CompletionRatio
	}
	return inputPrice, outputPrice
}

func providerName(modelName string, ownedBy string) string {
	lookup := strings.ToLower(modelName + " " + ownedBy)
	switch {
	case strings.Contains(lookup, "minimax"):
		return "MiniMax"
	case strings.Contains(lookup, "deepseek"):
		return "DeepSeek"
	case strings.Contains(lookup, "codex"):
		return "OpenAI Codex"
	case strings.Contains(lookup, "openai") || strings.HasPrefix(strings.ToLower(modelName), "gpt-") || strings.HasPrefix(strings.ToLower(modelName), "text-embedding-"):
		return "OpenAI"
	default:
		return ownedBy
	}
}

func modelNameFromPricing(pricing model.Pricing) string {
	if strings.TrimSpace(pricing.Description) != "" {
		return pricing.Description
	}
	return pricing.ModelName
}

func BuildCatalogEntry(pricing model.Pricing, ownerByModel map[string]string) dto.ModelCatalogEntry {
	ownedBy := ownerByModel[pricing.ModelName]
	if strings.TrimSpace(ownedBy) == "" {
		ownedBy = pricing.OwnerBy
	}
	source, estimated := pricingProvenance(pricing)
	inputPrice, outputPrice := PublicTokenPrices(pricing)
	provider := providerName(pricing.ModelName, ownedBy)
	return dto.ModelCatalogEntry{
		ModelName:                   pricing.ModelName,
		Name:                        modelNameFromPricing(pricing),
		Provider:                    provider,
		OwnedBy:                     ownedBy,
		EnableGroups:                pricing.EnableGroup,
		SupportedEndpointTypes:      pricing.SupportedEndpointTypes,
		SupportedEndpointTypeLabels: EndpointTypeLabels(pricing.SupportedEndpointTypes),
		EndpointRoutes:              EndpointRoutes(pricing.SupportedEndpointTypes),
		QuotaType:                   pricing.QuotaType,
		ModelRatio:                  pricing.ModelRatio,
		ModelPrice:                  pricing.ModelPrice,
		CompletionRatio:             pricing.CompletionRatio,
		InputPrice:                  inputPrice,
		OutputPrice:                 outputPrice,
		Pricing: &dto.ModelCatalogPricing{
			Input:  inputPrice,
			Output: outputPrice,
			Unit:   "usd_per_1m_tokens",
		},
		BillingMode:      pricing.BillingMode,
		BillingExpr:      pricing.BillingExpr,
		PricingSource:    source,
		PricingEstimated: estimated,
		PricingVersion:   pricing.PricingVersion,
	}
}

func BuildCatalogEntryForModel(modelName string, owner string, endpoints []constant.EndpointType) dto.ModelCatalogEntry {
	provider := providerName(modelName, owner)
	return dto.ModelCatalogEntry{
		ModelName:                   modelName,
		Name:                        modelName,
		Provider:                    provider,
		OwnedBy:                     owner,
		SupportedEndpointTypes:      endpoints,
		SupportedEndpointTypeLabels: EndpointTypeLabels(endpoints),
		EndpointRoutes:              EndpointRoutes(endpoints),
		PricingSource:               "missing",
		PricingEstimated:            true,
		InputPrice:                  0,
		OutputPrice:                 0,
		Pricing: &dto.ModelCatalogPricing{
			Input:  0,
			Output: 0,
			Unit:   "usd_per_1m_tokens",
		},
	}
}

func BuildCatalogEntries(pricings []model.Pricing, ownerByModel map[string]string) []dto.ModelCatalogEntry {
	entries := make([]dto.ModelCatalogEntry, 0, len(pricings))
	for _, pricing := range pricings {
		entries = append(entries, BuildCatalogEntry(pricing, ownerByModel))
	}
	return entries
}

func SortEntries(entries []dto.ModelCatalogEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return compareModels(entries[i].ModelName, entries[i].Provider, entries[i].SupportedEndpointTypes, entries[j].ModelName, entries[j].Provider, entries[j].SupportedEndpointTypes) < 0
	})
}

func SortOpenAIModels(models []dto.OpenAIModels) {
	sort.SliceStable(models, func(i, j int) bool {
		return compareModels(models[i].Id, models[i].Provider, models[i].SupportedEndpointTypes, models[j].Id, models[j].Provider, models[j].SupportedEndpointTypes) < 0
	})
}

func compareModels(leftName string, leftProvider string, leftEndpoints []constant.EndpointType, rightName string, rightProvider string, rightEndpoints []constant.EndpointType) int {
	if leftRank, ok := knownModelOrder[leftName]; ok {
		if rightRank, ok := knownModelOrder[rightName]; ok {
			return leftRank - rightRank
		}
	}
	leftCategory := categoryRank(leftName, leftEndpoints)
	rightCategory := categoryRank(rightName, rightEndpoints)
	if leftCategory != rightCategory {
		return leftCategory - rightCategory
	}
	leftProviderRank := providerRank(leftName, leftProvider)
	rightProviderRank := providerRank(rightName, rightProvider)
	if leftProviderRank != rightProviderRank {
		return leftProviderRank - rightProviderRank
	}
	leftVersion := versionRank(leftName)
	rightVersion := versionRank(rightName)
	if leftVersion != rightVersion {
		if leftVersion > rightVersion {
			return -1
		}
		return 1
	}
	leftCapacity := capacityRank(leftName)
	rightCapacity := capacityRank(rightName)
	if leftCapacity != rightCapacity {
		return rightCapacity - leftCapacity
	}
	return strings.Compare(leftName, rightName)
}

func categoryRank(modelName string, endpointTypes []constant.EndpointType) int {
	lowerName := strings.ToLower(modelName)
	if HasEndpointType(endpointTypes, constant.EndpointTypeEmbeddings) || strings.Contains(lowerName, "embedding") || strings.HasPrefix(lowerName, "embo-") {
		return 1
	}
	return 0
}

func providerRank(modelName string, provider string) int {
	normalized := strings.ToLower(modelName + " " + provider)
	switch {
	case strings.Contains(normalized, "minimax"):
		return 0
	case strings.Contains(normalized, "deepseek"):
		return 1
	case strings.Contains(normalized, "openai") || strings.Contains(normalized, "codex") || strings.HasPrefix(strings.ToLower(modelName), "gpt-") || strings.HasPrefix(strings.ToLower(modelName), "text-embedding-"):
		return 2
	default:
		return 100
	}
}

func versionRank(modelName string) float64 {
	token := versionTokenPattern.FindString(modelName)
	if token == "" {
		return 0
	}
	parts := strings.Split(token, ".")
	scale := 1.0
	score := 0.0
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		score += float64(value) * scale
		scale /= 100
	}
	return score
}

func capacityRank(modelName string) int {
	normalized := strings.ToLower(modelName)
	score := 0
	switch {
	case strings.Contains(normalized, "large"):
		score += 300
	case strings.Contains(normalized, "small"):
		score += 100
	}
	switch {
	case strings.Contains(normalized, "highspeed"):
		score += 40
	case strings.Contains(normalized, "pro"):
		score += 30
	case strings.Contains(normalized, "flash"):
		score += 10
	case strings.Contains(normalized, "mini"):
		score -= 10
	}
	return score
}
