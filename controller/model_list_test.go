package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type listModelsResponse struct {
	Data []dto.OpenAIModels `json:"data"`
}

func setupModelListControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	initModelListColumnNames(t)

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func initModelListColumnNames(t *testing.T) {
	t.Helper()

	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.UsingSQLite = false
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

func withTieredBillingConfig(t *testing.T, modes map[string]string, exprs map[string]string) {
	t.Helper()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "billing_setting.") {
			saved[key] = value
		}
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		model.InvalidatePricingCache()
	})

	modeBytes, err := common.Marshal(modes)
	require.NoError(t, err)
	exprBytes, err := common.Marshal(exprs)
	require.NoError(t, err)

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": string(modeBytes),
		"billing_setting.billing_expr": string(exprBytes),
	}))
	model.InvalidatePricingCache()
}

func withSelfUseModeDisabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = false
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func decodeListModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]struct{} {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload listModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))

	ids := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		ids[item.Id] = struct{}{}
	}
	return ids
}

func decodeListModelsPayload(t *testing.T, recorder *httptest.ResponseRecorder) ([]dto.OpenAIModels, map[string]interface{}) {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var raw map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &raw))
	require.ElementsMatch(t, []string{"data"}, mapKeys(raw))

	var payload listModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload.Data, raw
}

func decodeAnthropicModelsPayload(t *testing.T, recorder *httptest.ResponseRecorder) ([]dto.AnthropicModel, map[string]interface{}) {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var raw map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &raw))
	require.ElementsMatch(t, []string{"data"}, mapKeys(raw))

	var payload struct {
		Data []dto.AnthropicModel `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	return payload.Data, raw
}

func mapKeys(raw map[string]interface{}) []string {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	return keys
}

func modelIDs(models []dto.OpenAIModels) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.Id)
	}
	return ids
}

func anthropicModelIDs(models []dto.AnthropicModel) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}

func withSelfUseModeEnabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func setupRepresentativeModelListFixture(t *testing.T) *gorm.DB {
	t.Helper()

	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&[]model.Channel{
		{Id: 1, Type: constant.ChannelTypeMiniMax, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "MiniMax - OpenAI-Compatible"},
		{Id: 2, Type: constant.ChannelTypeDeepSeek, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "DeepSeek - OpenAI-Compatible"},
		{Id: 3, Type: constant.ChannelTypeCodex, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "OpenAI Codex OAuth"},
		{Id: 4, Type: constant.ChannelTypeMiniMax, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "MiniMax - Embeddings"},
		{Id: 5, Type: constant.ChannelTypeOpenAI, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "OpenAI - Embeddings"},
		{Id: 6, Type: constant.ChannelTypeAnthropic, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "MiniMax - Anthropic-Compatible"},
		{Id: 7, Type: constant.ChannelTypeAnthropic, Key: "test-key", Status: common.ChannelStatusEnabled, Name: "DeepSeek - Anthropic-Compatible"},
	}).Error)

	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "text-embedding-3-small", ChannelId: 5, Enabled: true},
		{Group: "default", Model: "MiniMax-M2.5", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "gpt-5.4-mini", ChannelId: 3, Enabled: true},
		{Group: "default", Model: "deepseek-v4-flash", ChannelId: 2, Enabled: true},
		{Group: "default", Model: "MiniMax-M2.7", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "embo-01", ChannelId: 4, Enabled: true},
		{Group: "default", Model: "gpt-5.3-codex-spark", ChannelId: 3, Enabled: true},
		{Group: "default", Model: "MiniMax-M3", ChannelId: 6, Enabled: true},
		{Group: "default", Model: "deepseek-v4-pro", ChannelId: 2, Enabled: true},
		{Group: "default", Model: "text-embedding-3-large", ChannelId: 5, Enabled: true},
		{Group: "default", Model: "gpt-5.5", ChannelId: 3, Enabled: true},
		{Group: "default", Model: "MiniMax-M2.5-highspeed", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "gpt-5.4", ChannelId: 3, Enabled: true},
		{Group: "default", Model: "MiniMax-M2.7", ChannelId: 6, Enabled: true},
		{Group: "default", Model: "MiniMax-M2.5-highspeed", ChannelId: 6, Enabled: true},
		{Group: "default", Model: "MiniMax-M2.5", ChannelId: 6, Enabled: true},
		{Group: "default", Model: "deepseek-v4-pro", ChannelId: 7, Enabled: true},
		{Group: "default", Model: "deepseek-v4-flash", ChannelId: 7, Enabled: true},
	}).Error)

	require.NoError(t, db.Create(&[]model.Model{
		{ModelName: "embo-01", Endpoints: `{"embeddings":"/v1/embeddings"}`, Status: 1},
		{ModelName: "text-embedding-3-large", Endpoints: `{"embeddings":"/v1/embeddings"}`, Status: 1},
		{ModelName: "text-embedding-3-small", Endpoints: `{"embeddings":"/v1/embeddings"}`, Status: 1},
	}).Error)
	model.InvalidatePricingCache()
	return db
}

func requestListModels(t *testing.T, target string, modelType int) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")

	ListModels(ctx, modelType)
	return recorder
}

func pricingByModelName(pricings []model.Pricing) map[string]model.Pricing {
	byName := make(map[string]model.Pricing, len(pricings))
	for _, pricing := range pricings {
		byName[pricing.ModelName] = pricing
	}
	return byName
}

func TestListModelsIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-tiered-visible-model":      "tiered_expr",
		"zz-tiered-empty-expr-model":   "tiered_expr",
		"zz-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-tiered-empty-expr-model": "   ",
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1001,
		Username: "model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-tiered-visible-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-empty-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-missing-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-unpriced-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1001)

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-tiered-visible-model")
	require.NotContains(t, ids, "zz-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-unpriced-model")

	pricingByName := pricingByModelName(model.GetPricing())
	visiblePricing, ok := pricingByName["zz-tiered-visible-model"]
	require.True(t, ok)
	require.Equal(t, "tiered_expr", visiblePricing.BillingMode)
	require.NotEmpty(t, visiblePricing.BillingExpr)

	emptyExprPricing, ok := pricingByName["zz-tiered-empty-expr-model"]
	require.True(t, ok)
	require.Empty(t, emptyExprPricing.BillingMode)
	require.Empty(t, emptyExprPricing.BillingExpr)

	missingExprPricing, ok := pricingByName["zz-tiered-missing-expr-model"]
	require.True(t, ok)
	require.Empty(t, missingExprPricing.BillingMode)
	require.Empty(t, missingExprPricing.BillingExpr)
}

func TestListModelsTokenLimitIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-token-tiered-visible-model":      "tiered_expr",
		"zz-token-tiered-empty-expr-model":   "tiered_expr",
		"zz-token-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-token-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-token-tiered-empty-expr-model": "",
	})
	setupModelListControllerTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"zz-token-tiered-visible-model":      true,
		"zz-token-tiered-empty-expr-model":   true,
		"zz-token-tiered-missing-expr-model": true,
		"zz-token-unpriced-model":            true,
	})

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-token-tiered-visible-model")
	require.NotContains(t, ids, "zz-token-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-token-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-token-unpriced-model")
}

func TestListModelsPayloadShapeAndPublicFields(t *testing.T) {
	setupRepresentativeModelListFixture(t)

	recorder := requestListModels(t, "/v1/models", constant.ChannelTypeOpenAI)
	models, raw := decodeListModelsPayload(t, recorder)

	assert.NotContains(t, raw, "object")
	assert.NotContains(t, raw, "success")
	require.NotEmpty(t, models)
	for _, modelItem := range models {
		assert.Equal(t, "model", modelItem.Object)
		assert.NotEmpty(t, modelItem.Id)
		assert.NotZero(t, modelItem.Created)
		assert.NotEmpty(t, modelItem.OwnedBy)
		assert.NotNil(t, modelItem.SupportedEndpointTypes)
		assert.NotNil(t, modelItem.Pricing)
	}

	var itemMaps struct {
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &itemMaps))
	for _, modelItem := range itemMaps.Data {
		assert.NotContains(t, modelItem, "pricing_source")
		assert.NotContains(t, modelItem, "pricing_estimated")
	}
}

func TestListModelsRepresentativeOrder(t *testing.T) {
	setupRepresentativeModelListFixture(t)

	recorder := requestListModels(t, "/v1/models", constant.ChannelTypeOpenAI)
	models, _ := decodeListModelsPayload(t, recorder)

	require.Equal(t, []string{
		"MiniMax-M2.7",
		"MiniMax-M2.5-highspeed",
		"MiniMax-M2.5",
		"deepseek-v4-pro",
		"deepseek-v4-flash",
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.3-codex-spark",
		"MiniMax-M3",
		"embo-01",
		"text-embedding-3-large",
		"text-embedding-3-small",
	}, modelIDs(models))
}

func TestListModelsAnthropicPayloadAndOrder(t *testing.T) {
	setupRepresentativeModelListFixture(t)

	recorder := requestListModels(t, "/v1/models?api_format=anthropic", constant.ChannelTypeAnthropic)
	models, raw := decodeAnthropicModelsPayload(t, recorder)

	assert.NotContains(t, raw, "object")
	assert.NotContains(t, raw, "success")
	assert.NotContains(t, raw, "first_id")
	assert.NotContains(t, raw, "last_id")
	assert.NotContains(t, raw, "has_more")
	require.Equal(t, []string{
		"MiniMax-M2.7",
		"MiniMax-M2.5-highspeed",
		"MiniMax-M2.5",
		"deepseek-v4-pro",
		"deepseek-v4-flash",
		"MiniMax-M3",
	}, anthropicModelIDs(models))
	for _, modelItem := range models {
		assert.Equal(t, "model", modelItem.Type)
		assert.Equal(t, "anthropic", modelItem.APIFormat)
		assert.NotEmpty(t, modelItem.EndpointTypes)
	}
}

func TestListModelsAnthropicEmptyPayload(t *testing.T) {
	withSelfUseModeEnabled(t)
	setupModelListControllerTestDB(t)

	recorder := requestListModels(t, "/v1/models?api_format=anthropic", constant.ChannelTypeAnthropic)
	models, raw := decodeAnthropicModelsPayload(t, recorder)

	assert.NotContains(t, raw, "first_id")
	assert.NotContains(t, raw, "last_id")
	assert.NotContains(t, raw, "has_more")
	require.Empty(t, models)
}
