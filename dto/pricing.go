package dto

import "github.com/QuantumNous/new-api/constant"

// 这里不好动就不动了，本来想独立出来的（
type OpenAIModels struct {
	Id                     string                  `json:"id"`
	Object                 string                  `json:"object"`
	Created                int                     `json:"created"`
	OwnedBy                string                  `json:"owned_by"`
	SupportedEndpointTypes []constant.EndpointType `json:"supported_endpoint_types"`
}

type AnthropicModel struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

type GeminiModel struct {
	Name                       interface{}   `json:"name"`
	BaseModelId                interface{}   `json:"baseModelId"`
	Version                    interface{}   `json:"version"`
	DisplayName                interface{}   `json:"displayName"`
	Description                interface{}   `json:"description"`
	InputTokenLimit            interface{}   `json:"inputTokenLimit"`
	OutputTokenLimit           interface{}   `json:"outputTokenLimit"`
	SupportedGenerationMethods []interface{} `json:"supportedGenerationMethods"`
	Thinking                   interface{}   `json:"thinking"`
	Temperature                interface{}   `json:"temperature"`
	MaxTemperature             interface{}   `json:"maxTemperature"`
	TopP                       interface{}   `json:"topP"`
	TopK                       interface{}   `json:"topK"`
}

type ModelCatalogEntry struct {
	ModelName                   string                  `json:"model_name"`
	OwnedBy                     string                  `json:"owned_by"`
	EnableGroups                []string                `json:"enable_groups,omitempty"`
	SupportedEndpointTypes      []constant.EndpointType `json:"supported_endpoint_types,omitempty"`
	SupportedEndpointTypeLabels []string                `json:"supported_endpoint_type_labels,omitempty"`
	QuotaType                   int                     `json:"quota_type"`
	ModelRatio                  float64                 `json:"model_ratio,omitempty"`
	ModelPrice                  float64                 `json:"model_price,omitempty"`
	CompletionRatio             float64                 `json:"completion_ratio,omitempty"`
	BillingMode                 string                  `json:"billing_mode,omitempty"`
	BillingExpr                 string                  `json:"billing_expr,omitempty"`
	PricingSource               string                  `json:"-"`
	PricingEstimated            bool                    `json:"-"`
	PricingVersion              string                  `json:"pricing_version,omitempty"`
}
