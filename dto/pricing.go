package dto

import "github.com/QuantumNous/new-api/constant"

// 这里不好动就不动了，本来想独立出来的（
type OpenAIModels struct {
	Id                          string                  `json:"id"`
	Object                      string                  `json:"object"`
	Created                     int                     `json:"created"`
	OwnedBy                     string                  `json:"owned_by"`
	CanonicalSlug               string                  `json:"canonical_slug,omitempty"`
	HuggingFaceID               *string                 `json:"hugging_face_id,omitempty"`
	Name                        string                  `json:"name,omitempty"`
	Description                 string                  `json:"description,omitempty"`
	Provider                    string                  `json:"provider,omitempty"`
	Links                       *ModelLinks             `json:"links,omitempty"`
	ContextLength               *int                    `json:"context_length,omitempty"`
	Architecture                *ModelArchitecture      `json:"architecture,omitempty"`
	TopProvider                 *ModelTopProvider       `json:"top_provider,omitempty"`
	PerRequestLimits            any                     `json:"per_request_limits,omitempty"`
	SupportedParameters         []string                `json:"supported_parameters,omitempty"`
	DefaultParameters           map[string]any          `json:"default_parameters,omitempty"`
	SupportedVoices             any                     `json:"supported_voices,omitempty"`
	KnowledgeCutoff             *string                 `json:"knowledge_cutoff,omitempty"`
	ExpirationDate              *string                 `json:"expiration_date,omitempty"`
	ContextWindow               *ModelContextWindow     `json:"context_window,omitempty"`
	SupportedEndpointTypes      []constant.EndpointType `json:"supported_endpoint_types"`
	SupportedEndpointTypeLabels []string                `json:"-"`
	EndpointRoutes              map[string]string       `json:"endpoint_routes,omitempty"`
	Pricing                     *ModelCatalogPricing    `json:"pricing,omitempty"`
	InputPrice                  float64                 `json:"-"`
	OutputPrice                 float64                 `json:"-"`
	QuotaType                   int                     `json:"-"`
	BillingMode                 string                  `json:"billing_mode,omitempty"`
	BillingExpr                 string                  `json:"billing_expr,omitempty"`
	PricingVersion              string                  `json:"pricing_version,omitempty"`
	EnableGroups                []string                `json:"-"`
}

// AnthropicModelsListResponse is the response shape for GET /v1/claude/models
type AnthropicModelsListResponse struct {
	Data    []AnthropicModel `json:"data"`
	HasMore bool             `json:"has_more"`
	FirstID string           `json:"first_id,omitempty"`
	LastID  string           `json:"last_id,omitempty"`
}

type AnthropicModel struct {
	ID            string                  `json:"id"`
	CreatedAt     string                  `json:"created_at"`
	DisplayName   string                  `json:"display_name"`
	Type          string                  `json:"type"`
	APIFormat     string                  `json:"api_format,omitempty"`
	ContextLength int                     `json:"context_length,omitempty"`
	InputPrice    float64                 `json:"input_price"`
	OutputPrice   float64                 `json:"output_price"`
	EndpointTypes []constant.EndpointType `json:"supported_endpoint_types,omitempty"`
	EndOfLifeAt   string                  `json:"end_of_life_at,omitempty"`
	Information   *AnthropicModelInfo     `json:"information,omitempty"`
}

type AnthropicModelInfo struct {
	Version string `json:"version,omitempty"`
	Status  string `json:"status,omitempty"`
	Tier    string `json:"tier,omitempty"`
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
	Created                     int                     `json:"created,omitempty"`
	Name                        string                  `json:"name,omitempty"`
	Description                 string                  `json:"description,omitempty"`
	Provider                    string                  `json:"provider,omitempty"`
	OwnedBy                     string                  `json:"owned_by"`
	CanonicalSlug               string                  `json:"canonical_slug,omitempty"`
	HuggingFaceID               *string                 `json:"hugging_face_id,omitempty"`
	ContextWindow               *ModelContextWindow     `json:"context_window,omitempty"`
	Architecture                *ModelArchitecture      `json:"architecture,omitempty"`
	SupportedParameters         []string                `json:"supported_parameters,omitempty"`
	DefaultParameters           map[string]any          `json:"default_parameters,omitempty"`
	EnableGroups                []string                `json:"enable_groups,omitempty"`
	SupportedEndpointTypes      []constant.EndpointType `json:"supported_endpoint_types,omitempty"`
	SupportedEndpointTypeLabels []string                `json:"supported_endpoint_type_labels,omitempty"`
	EndpointRoutes              map[string]string       `json:"endpoint_routes,omitempty"`
	QuotaType                   int                     `json:"quota_type"`
	ModelRatio                  float64                 `json:"model_ratio,omitempty"`
	ModelPrice                  float64                 `json:"model_price,omitempty"`
	CompletionRatio             float64                 `json:"completion_ratio,omitempty"`
	InputPrice                  float64                 `json:"input_price"`
	OutputPrice                 float64                 `json:"output_price"`
	Pricing                     *ModelCatalogPricing    `json:"pricing,omitempty"`
	BillingMode                 string                  `json:"billing_mode,omitempty"`
	BillingExpr                 string                  `json:"billing_expr,omitempty"`
	PricingSource               string                  `json:"-"`
	PricingEstimated            bool                    `json:"-"`
	PricingVersion              string                  `json:"pricing_version,omitempty"`
}

type ModelCatalogPricing struct {
	Prompt            string   `json:"prompt"`
	Completion        string   `json:"completion"`
	Request           string   `json:"request,omitempty"`
	Image             string   `json:"image,omitempty"`
	InputCacheRead    string   `json:"input_cache_read,omitempty"`
	InputCacheWrite   string   `json:"input_cache_write,omitempty"`
	Input             float64  `json:"input"`
	Output            float64  `json:"output"`
	CachedInput       *float64 `json:"cached_input,omitempty"`
	CacheWrite        *float64 `json:"cache_write,omitempty"`
	Unit              string   `json:"unit,omitempty"`
	CompatibilityUnit string   `json:"compatibility_unit,omitempty"`
	Scope             string   `json:"scope,omitempty"`
}

type ModelContextWindow struct {
	ContextLength       int `json:"context_length,omitempty"`
	MaxCompletionTokens int `json:"max_completion_tokens,omitempty"`
}

type ModelArchitecture struct {
	Modality         string   `json:"modality"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
	InstructType     *string  `json:"instruct_type"`
}

type ModelTopProvider struct {
	ContextLength       int  `json:"context_length"`
	MaxCompletionTokens *int `json:"max_completion_tokens"`
	IsModerated         bool `json:"is_moderated"`
}

type ModelLinks struct {
	Details string `json:"details"`
}
