package constant

type EndpointType string

const (
	EndpointTypeOpenAI                EndpointType = "openai"
	EndpointTypeOpenAIResponse        EndpointType = "openai-response"
	EndpointTypeOpenAIResponseCompact EndpointType = "openai-response-compact"
	EndpointTypeAnthropic             EndpointType = "anthropic"
	EndpointTypeGemini                EndpointType = "gemini"
	EndpointTypeReranker              EndpointType = "reranker"
	// EndpointTypeJinaRerank is retained only for legacy API and database input.
	EndpointTypeJinaRerank      EndpointType = "jina-rerank"
	EndpointTypeImageGeneration EndpointType = "image-generation"
	EndpointTypeEmbeddings      EndpointType = "embeddings"
	EndpointTypeOpenAIVideo     EndpointType = "openai-video"
	//EndpointTypeMidjourney     EndpointType = "midjourney-proxy"
	//EndpointTypeSuno           EndpointType = "suno-proxy"
	//EndpointTypeKling          EndpointType = "kling"
	//EndpointTypeJimeng         EndpointType = "jimeng"
)

func NormalizeEndpointType(endpointType EndpointType) EndpointType {
	if endpointType == EndpointTypeJinaRerank {
		return EndpointTypeReranker
	}
	return endpointType
}
