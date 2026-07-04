package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

const (
	defaultOpenAIEmbeddingsBaseURL = "https://api.openai.com/v1"
	sharedCodexKeyPrefix           = "shared:codex"
	defaultSharedCodexChannelID    = 5
)

func (a *Adaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, errors.New("codex channel: endpoint not supported")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("codex channel: /v1/messages endpoint not supported")
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("codex channel: endpoint not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("codex channel: endpoint not supported")
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return nil, errors.New("codex channel: /v1/chat/completions endpoint not supported")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("codex channel: /v1/rerank endpoint not supported")
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	isCompact := info != nil && info.RelayMode == relayconstant.RelayModeResponsesCompact

	if info != nil && info.ChannelSetting.SystemPrompt != "" {
		systemPrompt := info.ChannelSetting.SystemPrompt

		if len(request.Instructions) == 0 {
			if b, err := common.Marshal(systemPrompt); err == nil {
				request.Instructions = b
			} else {
				return nil, err
			}
		} else if info.ChannelSetting.SystemPromptOverride {
			var existing string
			if err := common.Unmarshal(request.Instructions, &existing); err == nil {
				existing = strings.TrimSpace(existing)
				if existing == "" {
					if b, err := common.Marshal(systemPrompt); err == nil {
						request.Instructions = b
					} else {
						return nil, err
					}
				} else {
					if b, err := common.Marshal(systemPrompt + "\n" + existing); err == nil {
						request.Instructions = b
					} else {
						return nil, err
					}
				}
			} else {
				if b, err := common.Marshal(systemPrompt); err == nil {
					request.Instructions = b
				} else {
					return nil, err
				}
			}
		}
	}
	// Codex backend requires the `instructions` field to be present.
	// Keep it consistent with Codex CLI behavior by defaulting to an empty string.
	if len(request.Instructions) == 0 {
		request.Instructions = json.RawMessage(`""`)
	}

	if isCompact {
		return request, nil
	}
	// codex: store must be false
	request.Store = json.RawMessage("false")
	if !isCodexPublicOpenAIAPI(info) {
		// chatgpt.com/backend-api/codex/responses rejects these OpenAI public API
		// parameters. Keep them only when a Codex channel is explicitly pointed at
		// the public OpenAI API.
		request.MaxOutputTokens = nil
		request.Temperature = nil
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.RelayMode == relayconstant.RelayModeEmbeddings {
		return (&openai.Adaptor{ChannelType: constant.ChannelTypeOpenAI}).DoResponse(c, resp, info)
	}

	if info.RelayMode != relayconstant.RelayModeResponses && info.RelayMode != relayconstant.RelayModeResponsesCompact {
		return nil, types.NewError(errors.New("codex channel: endpoint not supported"), types.ErrorCodeInvalidRequest)
	}

	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		return openai.OaiResponsesCompactionHandler(c, resp)
	}

	if info.IsStream {
		return openai.OaiResponsesStreamHandler(c, info, resp)
	}
	return openai.OaiResponsesHandler(c, info, resp)
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if info.RelayMode == relayconstant.RelayModeEmbeddings {
		baseURL := strings.TrimRight(strings.TrimSpace(info.ChannelBaseUrl), "/")
		if baseURL == "" || strings.Contains(baseURL, "chatgpt.com") {
			baseURL = defaultOpenAIEmbeddingsBaseURL
		}
		if strings.HasSuffix(baseURL, "/v1") {
			return baseURL + "/embeddings", nil
		}
		return relaycommon.GetFullRequestURL(baseURL, "/v1/embeddings", constant.ChannelTypeOpenAI), nil
	}

	if info.RelayMode != relayconstant.RelayModeResponses && info.RelayMode != relayconstant.RelayModeResponsesCompact {
		return "", errors.New("codex channel: only /v1/responses, /v1/responses/compact and /v1/embeddings are supported")
	}
	if isCodexPublicOpenAIAPI(info) {
		if info.RelayMode == relayconstant.RelayModeResponsesCompact {
			return "", errors.New("codex public OpenAI API mode: /v1/responses/compact is not supported")
		}
		return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, "/v1/responses", constant.ChannelTypeOpenAI), nil
	}
	path := "/backend-api/codex/responses"
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		path = "/backend-api/codex/responses/compact"
	}
	return relaycommon.GetFullRequestURL(info.ChannelBaseUrl, path, info.ChannelType), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)

	key, err := resolveCodexOAuthKey(info.ApiKey)
	if err != nil {
		return err
	}
	if info.RelayMode != relayconstant.RelayModeEmbeddings && isCodexPublicOpenAIAPI(info) {
		if strings.HasPrefix(key, "{") {
			return errors.New("codex public OpenAI API mode: API key is required, OAuth JSON is not supported for /v1/responses")
		}
		if key == "" {
			return errors.New("codex public OpenAI API mode: API key is required")
		}
		req.Set("Authorization", "Bearer "+key)
		req.Set("Content-Type", "application/json")
		if info.IsStream {
			req.Set("Accept", "text/event-stream")
		} else if req.Get("Accept") == "" {
			req.Set("Accept", "application/json")
		}
		return nil
	}
	if !strings.HasPrefix(key, "{") {
		return errors.New("codex channel: key must be a JSON object")
	}

	oauthKey, err := ParseOAuthKey(key)
	if err != nil {
		return err
	}

	accessToken := strings.TrimSpace(oauthKey.AccessToken)
	if accessToken == "" {
		return errors.New("codex channel: access_token is required")
	}

	req.Set("Authorization", "Bearer "+accessToken)
	req.Set("Content-Type", "application/json")

	if info.RelayMode == relayconstant.RelayModeEmbeddings {
		if req.Get("Accept") == "" {
			req.Set("Accept", "application/json")
		}
		return nil
	}

	accountID := strings.TrimSpace(oauthKey.AccountID)
	if accountID == "" {
		return errors.New("codex channel: account_id is required")
	}

	req.Set("chatgpt-account-id", accountID)

	if req.Get("OpenAI-Beta") == "" {
		req.Set("OpenAI-Beta", "responses=experimental")
	}
	if req.Get("originator") == "" {
		req.Set("originator", "codex_cli_rs")
	}

	// chatgpt.com/backend-api/codex/responses is strict about Content-Type.
	// Clients may omit it or include parameters like `application/json; charset=utf-8`,
	// which can be rejected by the upstream. Force the exact media type.
	req.Set("Content-Type", "application/json")
	if info.IsStream {
		req.Set("Accept", "text/event-stream")
	} else if req.Get("Accept") == "" {
		req.Set("Accept", "application/json")
	}

	return nil
}

func isCodexPublicOpenAIAPI(info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	return isOpenAIPublicBaseURL(info.ChannelBaseUrl)
}

func isOpenAIPublicBaseURL(baseURL string) bool {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return false
	}
	parsed, err := url.Parse(baseURL)
	if err == nil && parsed.Hostname() != "" {
		return strings.EqualFold(parsed.Hostname(), "api.openai.com")
	}
	return strings.Contains(strings.ToLower(baseURL), "api.openai.com")
}

func resolveCodexOAuthKey(raw string) (string, error) {
	key := strings.TrimSpace(raw)
	if !isSharedCodexKey(key) {
		return key, nil
	}

	channelID, err := parseSharedCodexChannelID(key)
	if err != nil {
		return "", err
	}

	sharedChannel, err := model.CacheGetChannel(channelID)
	if err != nil {
		return "", fmt.Errorf("codex channel: shared codex channel %d not found: %w", channelID, err)
	}
	if sharedChannel.Type != constant.ChannelTypeCodex {
		return "", fmt.Errorf("codex channel: shared channel %d is not Codex", channelID)
	}
	if sharedChannel.Status != common.ChannelStatusEnabled {
		return "", fmt.Errorf("codex channel: shared channel %d is not enabled", channelID)
	}

	sharedKey := strings.TrimSpace(sharedChannel.Key)
	if sharedKey == "" {
		return "", fmt.Errorf("codex channel: shared channel %d has empty key", channelID)
	}
	if isSharedCodexKey(sharedKey) {
		return "", fmt.Errorf("codex channel: shared channel %d cannot reference another shared key", channelID)
	}
	return sharedKey, nil
}

func isSharedCodexKey(key string) bool {
	key = strings.TrimSpace(key)
	return key == sharedCodexKeyPrefix || strings.HasPrefix(key, sharedCodexKeyPrefix+":")
}

func parseSharedCodexChannelID(key string) (int, error) {
	key = strings.TrimSpace(key)
	if key == sharedCodexKeyPrefix {
		return defaultSharedCodexChannelID, nil
	}
	if !strings.HasPrefix(key, sharedCodexKeyPrefix+":") {
		return 0, fmt.Errorf("codex channel: invalid shared codex key reference")
	}
	rawID := strings.TrimSpace(strings.TrimPrefix(key, sharedCodexKeyPrefix+":"))
	channelID, err := strconv.Atoi(rawID)
	if err != nil || channelID <= 0 {
		return 0, fmt.Errorf("codex channel: invalid shared codex channel id")
	}
	return channelID, nil
}
