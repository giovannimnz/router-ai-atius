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

var versionTokenPattern = regexp.MustCompile(`\d+(?:\.\d+)*`)

const (
	gteEmbeddingCreated = 0
	gteRerankerCreated  = 0
	gteContextLength    = 8192
	gteEmbeddingHFModel = "Alibaba-NLP/gte-multilingual-base"
	gteRerankerHFModel  = "Alibaba-NLP/gte-multilingual-reranker-base"
)

func EndpointTypeLabel(endpointType constant.EndpointType) string {
	switch endpointType {
	case constant.EndpointTypeOpenAI:
		return "OpenAI-Compatible"
	case constant.EndpointTypeAnthropic:
		return "Anthropic-Compatible"
	case constant.EndpointTypeEmbeddings:
		return "Embeddings"
	case constant.EndpointTypeJinaRerank:
		return "Rerank"
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
	if channelType == constant.ChannelTypeAtiusLocalEmbeddings {
		return "atius"
	}
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
	if pricing.UseDollarCost {
		return "input_output_price", false
	}
	if pricing.QuotaType == 1 {
		if pricing.ModelPrice > 0 {
			return "model_price", false
		}
		if _, ok := ratio_setting.GetModelPrice(pricing.ModelName, false); ok {
			return "model_price", false
		}
	}
	if info := ratio_setting.GetModelRatioInfo(pricing.ModelName); info.Explicit {
		return "model_ratio", false
	}
	if strings.TrimSpace(pricing.BillingMode) == "tiered_expr" && strings.TrimSpace(pricing.BillingExpr) != "" {
		return "billing_expr", false
	}
	return "missing", true
}

func PublicTokenPrices(pricing model.Pricing) (float64, float64) {
	if pricing.UseDollarCost {
		return pricing.InputPrice, pricing.OutputPrice
	}
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

func usdPerTokenString(perMillion float64) string {
	if perMillion <= 0 {
		return "0"
	}
	return strconv.FormatFloat(perMillion/1_000_000, 'f', -1, 64)
}

func syncOpenRouterPricing(pricing *dto.ModelCatalogPricing, endpointTypes []constant.EndpointType) {
	if pricing == nil {
		return
	}
	outputPrice := pricing.Output
	if HasEndpointType(endpointTypes, constant.EndpointTypeEmbeddings) || HasEndpointType(endpointTypes, constant.EndpointTypeJinaRerank) {
		outputPrice = 0
		pricing.Output = 0
	}
	pricing.Prompt = usdPerTokenString(pricing.Input)
	pricing.Completion = usdPerTokenString(outputPrice)
	if pricing.Request == "" {
		pricing.Request = "0"
	}
	pricing.Image = "0"
	pricing.CompatibilityUnit = "usd_per_token"
	if pricing.CachedInput != nil {
		pricing.InputCacheRead = usdPerTokenString(*pricing.CachedInput)
	}
	if pricing.CacheWrite != nil {
		pricing.InputCacheWrite = usdPerTokenString(*pricing.CacheWrite)
	}
}

func stringPointer(value string) *string {
	return &value
}

func localGTEProfile(entry *dto.ModelCatalogEntry) bool {
	if entry == nil {
		return false
	}

	var created int
	var canonicalSlug string
	var huggingFaceID string
	var name string
	var description string
	var architecture dto.ModelArchitecture
	switch entry.ModelName {
	case constant.AtiusLocalEmbeddingModel:
		created = gteEmbeddingCreated
		canonicalSlug = "alibaba-nlp/gte-multilingual-base"
		huggingFaceID = gteEmbeddingHFModel
		name = "Atius: GTE Multilingual Embeddings"
		description = "Self-hosted multilingual GTE embedding model with 768-dimensional vectors and an 8,192-token context window."
		architecture = dto.ModelArchitecture{
			Modality:         "text->embeddings",
			InputModalities:  []string{"text"},
			OutputModalities: []string{"embeddings"},
			Tokenizer:        "XLM-R",
		}
		entry.SupportedEndpointTypes = []constant.EndpointType{constant.EndpointTypeEmbeddings}
	case constant.AtiusLocalRerankerModel:
		created = gteRerankerCreated
		canonicalSlug = "alibaba-nlp/gte-multilingual-reranker-base"
		huggingFaceID = gteRerankerHFModel
		name = "Atius: GTE Multilingual Reranker"
		description = "Self-hosted multilingual GTE cross-encoder reranker with an 8,192-token context window."
		architecture = dto.ModelArchitecture{
			Modality:         "text->rerank",
			InputModalities:  []string{"text"},
			OutputModalities: []string{"rerank"},
			Tokenizer:        "XLM-R",
		}
		entry.SupportedEndpointTypes = []constant.EndpointType{constant.EndpointTypeJinaRerank}
	default:
		return false
	}

	entry.Created = created
	entry.CanonicalSlug = canonicalSlug
	entry.HuggingFaceID = stringPointer(huggingFaceID)
	entry.Name = name
	entry.Description = description
	entry.Provider = "Atius"
	entry.OwnedBy = "atius"
	entry.ContextWindow = &dto.ModelContextWindow{ContextLength: gteContextLength}
	entry.Architecture = &architecture
	entry.SupportedEndpointTypeLabels = EndpointTypeLabels(entry.SupportedEndpointTypes)
	entry.EndpointRoutes = EndpointRoutes(entry.SupportedEndpointTypes)
	entry.SupportedParameters = []string{}
	return true
}

func providerSlug(provider string, ownedBy string) string {
	lookup := strings.ToLower(strings.TrimSpace(provider + " " + ownedBy))
	switch {
	case strings.Contains(lookup, "openai") || strings.Contains(lookup, "codex"):
		return "openai"
	case strings.Contains(lookup, "minimax"):
		return "minimax"
	case strings.Contains(lookup, "deepseek"):
		return "deepseek"
	}
	slug := strings.ToLower(strings.TrimSpace(ownedBy))
	slug = strings.NewReplacer(" ", "-", "_", "-").Replace(slug)
	return strings.Trim(slug, "-/")
}

func genericArchitecture(entry dto.ModelCatalogEntry) *dto.ModelArchitecture {
	if HasEndpointType(entry.SupportedEndpointTypes, constant.EndpointTypeEmbeddings) {
		return &dto.ModelArchitecture{Modality: "text->embeddings", InputModalities: []string{"text"}, OutputModalities: []string{"embeddings"}, Tokenizer: "Other"}
	}
	if HasEndpointType(entry.SupportedEndpointTypes, constant.EndpointTypeJinaRerank) {
		return &dto.ModelArchitecture{Modality: "text->rerank", InputModalities: []string{"text"}, OutputModalities: []string{"rerank"}, Tokenizer: "Other"}
	}
	return nil
}

func defaultSupportedParameters(entry dto.ModelCatalogEntry) []string {
	if HasEndpointType(entry.SupportedEndpointTypes, constant.EndpointTypeEmbeddings) || HasEndpointType(entry.SupportedEndpointTypes, constant.EndpointTypeJinaRerank) {
		return []string{}
	}
	if strings.EqualFold(entry.Provider, "OpenAI Codex") {
		return []string{"include_reasoning", "max_completion_tokens", "max_tokens", "reasoning", "reasoning_effort", "response_format", "structured_outputs", "tool_choice", "tools"}
	}
	return []string{}
}

func EnrichOpenRouterEntry(entry *dto.ModelCatalogEntry) {
	if entry == nil {
		return
	}
	if localGTEProfile(entry) {
		if entry.Pricing == nil && entry.ModelRatio > 0 {
			entry.InputPrice = entry.ModelRatio * 2
			entry.OutputPrice = 0
			entry.Pricing = &dto.ModelCatalogPricing{Input: entry.InputPrice, Output: 0, Unit: "usd_per_1m_tokens"}
		}
		syncOpenRouterPricing(entry.Pricing, entry.SupportedEndpointTypes)
		return
	}
	if entry.CanonicalSlug == "" {
		slug := providerSlug(entry.Provider, entry.OwnedBy)
		if slug == "" {
			entry.CanonicalSlug = entry.ModelName
		} else {
			entry.CanonicalSlug = slug + "/" + entry.ModelName
		}
	}
	if entry.Description == "" {
		entry.Description = entry.Name
	}
	if entry.Architecture == nil {
		entry.Architecture = genericArchitecture(*entry)
	}
	if entry.SupportedParameters == nil {
		entry.SupportedParameters = defaultSupportedParameters(*entry)
	}
	syncOpenRouterPricing(entry.Pricing, entry.SupportedEndpointTypes)
}

func providerName(modelName string, ownedBy string) string {
	lookup := strings.ToLower(modelName + " " + ownedBy)
	switch {
	case modelName == constant.AtiusLocalEmbeddingModel || modelName == constant.AtiusLocalRerankerModel:
		return "Atius"
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
	inputPrice := 0.0
	outputPrice := 0.0
	var publicPricing *dto.ModelCatalogPricing
	if source == "input_output_price" || source == "model_ratio" || source == "model_price" {
		inputPrice, outputPrice = PublicTokenPrices(pricing)
		publicPricing = &dto.ModelCatalogPricing{
			Input:  inputPrice,
			Output: outputPrice,
			Unit:   "usd_per_1m_tokens",
		}
		if pricing.QuotaType == 1 {
			inputPrice = 0
			outputPrice = 0
			publicPricing.Input = 0
			publicPricing.Output = 0
			publicPricing.Request = strconv.FormatFloat(pricing.ModelPrice, 'f', -1, 64)
		}
		if pricing.UseDollarCost && pricing.CacheRatio != nil {
			cachedInput := inputPrice * *pricing.CacheRatio
			publicPricing.CachedInput = &cachedInput
		}
		if pricing.UseDollarCost && pricing.CreateCacheRatio != nil {
			cacheWrite := inputPrice * *pricing.CreateCacheRatio
			publicPricing.CacheWrite = &cacheWrite
		}
	}
	billingMode := pricing.BillingMode
	if pricing.UseDollarCost {
		billingMode = "dollar_cost"
	}
	provider := providerName(pricing.ModelName, ownedBy)
	entry := dto.ModelCatalogEntry{
		ModelName:                   pricing.ModelName,
		Created:                     0,
		Name:                        modelNameFromPricing(pricing),
		Description:                 strings.TrimSpace(pricing.Description),
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
		Pricing:                     publicPricing,
		BillingMode:                 billingMode,
		BillingExpr:                 pricing.BillingExpr,
		PricingSource:               source,
		PricingEstimated:            estimated,
		PricingVersion:              pricing.PricingVersion,
	}
	EnrichOpenRouterEntry(&entry)
	return entry
}

func BuildCatalogEntryForModel(modelName string, owner string, endpoints []constant.EndpointType) dto.ModelCatalogEntry {
	provider := providerName(modelName, owner)
	entry := dto.ModelCatalogEntry{
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
		Pricing:                     nil,
	}
	EnrichOpenRouterEntry(&entry)
	return entry
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
