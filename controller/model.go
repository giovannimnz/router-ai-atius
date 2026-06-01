package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/channel/ai360"
	"github.com/QuantumNous/new-api/relay/channel/lingyiwanwu"
	"github.com/QuantumNous/new-api/relay/channel/minimax"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// https://platform.openai.com/docs/api-reference/models/list

var openAIModels []dto.OpenAIModels
var openAIModelsMap map[string]dto.OpenAIModels
var channelId2Models map[int][]string

func init() {
	// https://platform.openai.com/docs/models/model-endpoint-compatibility
	for i := 0; i < constant.APITypeDummy; i++ {
		if i == constant.APITypeAIProxyLibrary {
			continue
		}
		adaptor := relay.GetAdaptor(i)
		channelName := adaptor.GetChannelName()
		modelNames := adaptor.GetModelList()
		for _, modelName := range modelNames {
			openAIModels = append(openAIModels, dto.OpenAIModels{
				Id:      modelName,
				Object:  "model",
				Created: 1626777600,
				OwnedBy: channelName,
			})
		}
	}
	for _, modelName := range ai360.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: ai360.ChannelName,
		})
	}
	for _, modelName := range moonshot.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: moonshot.ChannelName,
		})
	}
	for _, modelName := range lingyiwanwu.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: lingyiwanwu.ChannelName,
		})
	}
	for _, modelName := range minimax.ModelList {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: minimax.ChannelName,
		})
	}
	for modelName, _ := range constant.MidjourneyModel2Action {
		openAIModels = append(openAIModels, dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: "midjourney",
		})
	}
	openAIModelsMap = make(map[string]dto.OpenAIModels)
	for _, aiModel := range openAIModels {
		openAIModelsMap[aiModel.Id] = aiModel
	}
	channelId2Models = make(map[int][]string)
	for i := 1; i <= constant.ChannelTypeDummy; i++ {
		apiType, success := common.ChannelType2APIType(i)
		if !success || apiType == constant.APITypeAIProxyLibrary {
			continue
		}
		meta := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: i,
		}}
		adaptor := relay.GetAdaptor(apiType)
		adaptor.Init(meta)
		channelId2Models[i] = adaptor.GetModelList()
	}
	openAIModels = lo.UniqBy(openAIModels, func(m dto.OpenAIModels) string {
		return m.Id
	})
}

func channelOwnerName(channelType int) string {
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

func getPreferredModelOwners(modelNames []string, groups []string) map[string]string {
	channelTypes, err := model.GetPreferredModelOwnerChannelTypes(modelNames, groups)
	if err != nil {
		common.SysLog(fmt.Sprintf("GetPreferredModelOwnerChannelTypes error: %v", err))
		return map[string]string{}
	}

	ownerByChannelType := make(map[int]string)
	owners := make(map[string]string, len(channelTypes))
	for modelName, channelType := range channelTypes {
		owner, ok := ownerByChannelType[channelType]
		if !ok {
			owner = channelOwnerName(channelType)
			ownerByChannelType[channelType] = owner
		}
		if owner != "" {
			owners[modelName] = owner
		}
	}
	return owners
}

func buildOpenAIModel(modelName string, ownerByModel map[string]string) dto.OpenAIModels {
	var oaiModel dto.OpenAIModels
	if staticModel, ok := openAIModelsMap[modelName]; ok {
		oaiModel = staticModel
	} else {
		oaiModel = dto.OpenAIModels{
			Id:      modelName,
			Object:  "model",
			Created: 1626777600,
			OwnedBy: "custom",
		}
	}
	if owner, ok := ownerByModel[modelName]; ok && owner != "" {
		oaiModel.OwnedBy = owner
	}
	oaiModel.SupportedEndpointTypes = model.GetModelSupportEndpointTypes(modelName)
	// Enrich with context_length / max_completion_tokens / pricing from DB
	if m, err := model.GetModelByName(modelName); err == nil && m != nil {
		if m.ContextLength > 0 {
			oaiModel.ContextLength = m.ContextLength
		}
		if m.MaxCompletionTokens > 0 {
			oaiModel.MaxCompletionTokens = m.MaxCompletionTokens
		}
		if m.PricingPrompt > 0 {
			oaiModel.PricingPrompt = m.PricingPrompt
		}
		if m.PricingCompletion > 0 {
			oaiModel.PricingCompletion = m.PricingCompletion
		}
	}
	return oaiModel
}

type modelListGroups struct {
	userGroup   string
	tokenGroup  string
	ownerGroups []string
}

func getModelListGroups(c *gin.Context) (modelListGroups, error) {
	tokenGroup := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup == "" && (tokenGroup == "" || tokenGroup == "auto") {
		var err error
		userGroup, err = model.GetUserGroup(c.GetInt("id"), false)
		if err != nil {
			return modelListGroups{}, err
		}
	}

	if tokenGroup == "auto" {
		return modelListGroups{
			userGroup:   userGroup,
			tokenGroup:  tokenGroup,
			ownerGroups: service.GetUserAutoGroup(userGroup),
		}, nil
	}

	group := userGroup
	if tokenGroup != "" {
		group = tokenGroup
	}
	return modelListGroups{
		userGroup:   userGroup,
		tokenGroup:  tokenGroup,
		ownerGroups: []string{group},
	}, nil
}

func ListModels(c *gin.Context, modelType int) {
	acceptUnsetRatioModel := operation_setting.SelfUseModeEnabled
	if !acceptUnsetRatioModel {
		userId := c.GetInt("id")
		if userId > 0 {
			userSettings, _ := model.GetUserSetting(userId, false)
			if userSettings.AcceptUnsetRatioModel {
				acceptUnsetRatioModel = true
			}
		}
	}

	userModelNames := make([]string, 0)
	groups, err := getModelListGroups(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "get user group failed",
		})
		return
	}
	ownerGroups := groups.ownerGroups
	modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
	if modelLimitEnable {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		var tokenModelLimit map[string]bool
		if ok {
			tokenModelLimit = s.(map[string]bool)
		} else {
			tokenModelLimit = map[string]bool{}
		}
		for allowModel, _ := range tokenModelLimit {
			if !acceptUnsetRatioModel {
				if !helper.HasModelBillingConfig(allowModel) {
					continue
				}
			}
			userModelNames = append(userModelNames, allowModel)
		}
	} else {
		var models []string
		if groups.tokenGroup == "auto" {
			for _, autoGroup := range ownerGroups {
				groupModels := model.GetGroupEnabledModels(autoGroup)
				for _, g := range groupModels {
					if !common.StringsContains(models, g) {
						models = append(models, g)
					}
				}
			}
		} else {
			models = model.GetGroupEnabledModels(ownerGroups[0])
		}
		for _, modelName := range models {
			if !acceptUnsetRatioModel {
				if !helper.HasModelBillingConfig(modelName) {
					continue
				}
			}
			userModelNames = append(userModelNames, modelName)
		}
	}

	// Filter out -hs aliases: they exist in abilities for routing (model_mapping
	// rewrites them upstream to -highspeed), but they should not appear as
	// separate entries in /v1/models. The /v1/chat/completions endpoint still
	// accepts -hs and redirects transparently.
	filtered := make([]string, 0, len(userModelNames))
	for _, n := range userModelNames {
		if strings.HasSuffix(n, "-hs") {
			continue
		}
		filtered = append(filtered, n)
	}
	userModelNames = filtered

	ownerByModel := map[string]string{}
	if len(ownerGroups) > 0 {
		ownerByModel = getPreferredModelOwners(userModelNames, ownerGroups)
	}
	// Sort: higher version first, -highspeed/-pro before standard/-flash
	sortModelsByPriority(userModelNames)
	userOpenAiModels := make([]dto.OpenAIModels, 0, len(userModelNames))
	for _, modelName := range userModelNames {
		userOpenAiModels = append(userOpenAiModels, buildOpenAIModel(modelName, ownerByModel))
	}

	switch modelType {
	case constant.ChannelTypeAnthropic:
		useranthropicModels := make([]dto.AnthropicModel, len(userOpenAiModels))
		for i, model := range userOpenAiModels {
			useranthropicModels[i] = dto.AnthropicModel{
				ID:          model.Id,
				CreatedAt:   time.Unix(int64(model.Created), 0).UTC().Format(time.RFC3339),
				DisplayName: model.Id,
				Type:        "model",
			}
		}
		c.JSON(200, gin.H{
			"data":     useranthropicModels,
			"first_id": useranthropicModels[0].ID,
			"has_more": false,
			"last_id":  useranthropicModels[len(useranthropicModels)-1].ID,
		})
	case constant.ChannelTypeGemini:
		userGeminiModels := make([]dto.GeminiModel, len(userOpenAiModels))
		for i, model := range userOpenAiModels {
			userGeminiModels[i] = dto.GeminiModel{
				Name:        model.Id,
				DisplayName: model.Id,
			}
		}
		c.JSON(200, gin.H{
			"models":        userGeminiModels,
			"nextPageToken": nil,
		})
	default:
		c.JSON(200, gin.H{
			"success": true,
			"data":    userOpenAiModels,
			"object":  "list",
		})
	}
}

func ChannelListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    openAIModels,
	})
}

func DashboardListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    channelId2Models,
	})
}

func EnabledListModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"data":    model.GetEnabledModels(),
	})
}

func RetrieveModel(c *gin.Context, modelType int) {
	modelId := c.Param("model")
	if aiModel, ok := openAIModelsMap[modelId]; ok {
		// Mirror buildOpenAIModel: enrich with supported endpoint types + context/pricing from DB
		aiModel.SupportedEndpointTypes = model.GetModelSupportEndpointTypes(modelId)
		if m, err := model.GetModelByName(modelId); err == nil && m != nil {
			if m.ContextLength > 0 {
				aiModel.ContextLength = m.ContextLength
			}
			if m.MaxCompletionTokens > 0 {
				aiModel.MaxCompletionTokens = m.MaxCompletionTokens
			}
			if m.PricingPrompt > 0 {
				aiModel.PricingPrompt = m.PricingPrompt
			}
			if m.PricingCompletion > 0 {
				aiModel.PricingCompletion = m.PricingCompletion
			}
		}
		switch modelType {
		case constant.ChannelTypeAnthropic:
			c.JSON(200, dto.AnthropicModel{
				ID:          aiModel.Id,
				CreatedAt:   time.Unix(int64(aiModel.Created), 0).UTC().Format(time.RFC3339),
				DisplayName: aiModel.Id,
				Type:        "model",
			})
		default:
			c.JSON(200, aiModel)
		}
	} else {
		openAIError := types.OpenAIError{
			Message: fmt.Sprintf("The model '%s' does not exist", modelId),
			Type:    "invalid_request_error",
			Param:   "model",
			Code:    "model_not_found",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
	}
}

// ListClaudeModels returns all Anthropic-compatible models (channels with type=14)
// GET /v1/claude/models
func ListClaudeModels(c *gin.Context) {
	userId := c.GetInt("id")
	userGroup := ""
	groupSet := make(map[string]bool)

	// Get user group from token
	tokenGroup := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	if tokenGroup != "" && tokenGroup != "auto" {
		groupSet[tokenGroup] = true
	} else {
		// Get user's default group
		ug, err := model.GetUserGroup(userId, false)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "get user group failed"})
			return
		}
		groupSet[ug] = true
	}

	// Handle auto group
	if tokenGroup == "auto" {
		autoGroups := service.GetUserAutoGroup(userGroup)
		groupSet = make(map[string]bool)
		for _, g := range autoGroups {
			groupSet[g] = true
		}
	}

	// Collect all groups
	var groups []string
	for g := range groupSet {
		groups = append(groups, g)
	}
	if len(groups) == 0 {
		groups = []string{"default"}
	}

	// Query all abilities for channels with type=14 (Anthropic)
	// Use a subquery to avoid referencing "type" column (reserved keyword) in WHERE clause
	var abilities []model.AbilityWithChannel
	subQuery := model.DB.Table("channels").Select("id").Where("status = ? AND type = ?", common.ChannelStatusEnabled, constant.ChannelTypeAnthropic)

	err := model.DB.Table("abilities").
		Select("abilities.*, channels.type as channel_type").
		Joins("LEFT JOIN channels ON abilities.channel_id = channels.id").
		Where("abilities.channel_id IN (?) AND abilities.enabled = ?", subQuery, true).
		Where("abilities.group IN ?", groups).
		Find(&abilities).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "query abilities failed"})
		return
	}

	// Collect unique model names
	modelSet := make(map[string]bool)
	for _, a := range abilities {
		modelSet[a.Model] = true
	}
	modelLimitEnabled := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)

	if modelLimitEnabled {
		s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
		if ok {
			limits := s.(map[string]bool)
			filtered := make(map[string]bool)
			for m := range modelSet {
				if limits[m] {
					filtered[m] = true
				}
			}
			modelSet = filtered
		}
	}


	// Build Anthropic format response
	var models []dto.AnthropicModel
	for modelName := range modelSet {
		models = append(models, dto.AnthropicModel{
			ID:          modelName,
			CreatedAt:   time.Unix(1626777600, 0).UTC().Format(time.RFC3339),
			DisplayName: modelName,
			Type:        "model",
		})
	}


	hasMore := false
	firstID := ""
	lastID := ""
	if len(models) > 0 {
		firstID = models[0].ID
		lastID = models[len(models)-1].ID
	}

	// Add deprecation headers
	c.Header("Deprecation", "true")
	c.Header("Sunset", "Sat, 01 Jan 2027 00:00:00 GMT")
	c.Header("Link", `</v1/models?api_format=anthropic>; rel="successor-version"`)

	c.JSON(http.StatusOK, dto.AnthropicModelsListResponse{
		Data:    models,
		HasMore: hasMore,
		FirstID: firstID,
		LastID:  lastID,
	})
}

// modelSortKey returns a tuple (version, tier) used to sort /v1/models output.
// Higher version first; -highspeed/-pro ahead of standard/-flash; tiebreak by name.
func modelSortKey(name string) (float64, int, string) {
	lower := strings.ToLower(name)
	// Tier: -highspeed/-hs = 0 (highest), pro=0, standard=1, flash=2
	tier := 1
	if strings.HasSuffix(lower, "-highspeed") || strings.HasSuffix(lower, "-hs") {
		tier = 0
	} else if strings.HasSuffix(lower, "-pro") {
		tier = 0
	} else if strings.HasSuffix(lower, "-flash") {
		tier = 2
	}
	// Version: extract numeric after the M-/v prefix, e.g. "MiniMax-M3" -> 3, "MiniMax-M2.7" -> 2.7,
	// "deepseek-v4-flash" -> 4, "deepseek-v4-pro" -> 4
	version := 0.0
	// Try MiniMax-M pattern
	if idx := strings.Index(lower, "minimax-m"); idx >= 0 {
		rest := lower[idx+len("minimax-m"):]
		rest = strings.TrimSuffix(rest, "-highspeed")
		rest = strings.TrimSuffix(rest, "-hs")
		if v, err := strconv.ParseFloat(rest, 64); err == nil {
			version = v
		}
	} else if idx := strings.Index(lower, "deepseek-v"); idx >= 0 {
		rest := lower[idx+len("deepseek-v"):]
		// extract leading digits
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end > 0 {
			if v, err := strconv.ParseFloat(rest[:end], 64); err == nil {
				version = v
			}
		}
	}
	// DESC version => negate for sort.Slice ascending
	return -version, tier, name
}

func sortModelsByPriority(names []string) {
	sort.SliceStable(names, func(i, j int) bool {
		vi, ti, ni := modelSortKey(names[i])
		vj, tj, nj := modelSortKey(names[j])
		if vi != vj {
			return vi < vj // vi is already negated, so smaller = higher version
		}
		if ti != tj {
			return ti < tj
		}
		return ni < nj
	})
}
