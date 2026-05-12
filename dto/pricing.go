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

// AnthropicModelsListResponse is the response shape for GET /v1/claude/models
type AnthropicModelsListResponse struct {
	Data     []AnthropicModel `json:"data"`
	HasMore  bool             `json:"has_more"`
	FirstID  string           `json:"first_id,omitempty"`
	LastID   string           `json:"last_id,omitempty"`
}

type AnthropicModel struct {
	ID          string                   `json:"id"`
	CreatedAt   string                   `json:"created_at"`
	DisplayName string                   `json:"display_name"`
	Type        string                   `json:"type"`
	EndOfLifeAt string                   `json:"end_of_life_at,omitempty"`
	Information *AnthropicModelInfo      `json:"information,omitempty"`
}

type AnthropicModelInfo struct {
	Version   string `json:"version,omitempty"`
	Status    string `json:"status,omitempty"`
	Tier      string `json:"tier,omitempty"`
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
