package modelcatalog

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

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
	if info := ratio_setting.GetModelRatioInfo(pricing.ModelName); info.Explicit {
		return "model_ratio", false
	}
	if strings.TrimSpace(pricing.BillingMode) == "tiered_expr" && strings.TrimSpace(pricing.BillingExpr) != "" {
		return "billing_expr", false
	}
	return "missing", true
}

func BuildCatalogEntry(pricing model.Pricing, ownerByModel map[string]string) dto.ModelCatalogEntry {
	ownedBy := ownerByModel[pricing.ModelName]
	if strings.TrimSpace(ownedBy) == "" {
		ownedBy = pricing.OwnerBy
	}
	source, estimated := pricingProvenance(pricing)
	return dto.ModelCatalogEntry{
		ModelName:                   pricing.ModelName,
		OwnedBy:                     ownedBy,
		EnableGroups:                pricing.EnableGroup,
		SupportedEndpointTypes:      pricing.SupportedEndpointTypes,
		SupportedEndpointTypeLabels: EndpointTypeLabels(pricing.SupportedEndpointTypes),
		QuotaType:                   pricing.QuotaType,
		ModelRatio:                  pricing.ModelRatio,
		ModelPrice:                  pricing.ModelPrice,
		CompletionRatio:             pricing.CompletionRatio,
		BillingMode:                 pricing.BillingMode,
		BillingExpr:                 pricing.BillingExpr,
		PricingSource:               source,
		PricingEstimated:            estimated,
		PricingVersion:              pricing.PricingVersion,
	}
}

func BuildCatalogEntries(pricings []model.Pricing, ownerByModel map[string]string) []dto.ModelCatalogEntry {
	entries := make([]dto.ModelCatalogEntry, 0, len(pricings))
	for _, pricing := range pricings {
		entries = append(entries, BuildCatalogEntry(pricing, ownerByModel))
	}
	return entries
}
