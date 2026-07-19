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
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
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
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Ability{},
		&model.Model{},
		&model.Vendor{},
		&model.CodexCatalogSnapshot{},
		&model.CodexCatalogCandidate{},
	))

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
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
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

func withSelfUseModeEnabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = true
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func decodeListModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]struct{} {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var raw map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &raw))
	require.Contains(t, raw, "data")
	require.NotContains(t, raw, "success")
	require.NotContains(t, raw, "object")

	var payload listModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))

	ids := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		ids[item.Id] = struct{}{}
	}
	return ids
}

func pricingByModelName(pricings []model.Pricing) map[string]model.Pricing {
	byName := make(map[string]model.Pricing, len(pricings))
	for _, pricing := range pricings {
		byName[pricing.ModelName] = pricing
	}
	return byName
}

func TestApplyCodexMetadataUsesOAuthContextAndBillingContract(t *testing.T) {
	entries := []dto.ModelCatalogEntry{{
		ModelName:   "gpt-5.6-sol",
		InputPrice:  5,
		OutputPrice: 30,
		Pricing: &dto.ModelCatalogPricing{
			Input:  5,
			Output: 30,
			Unit:   "usd_per_1m_tokens",
		},
	}}

	applyCodexMetadataToCatalogEntries(entries, map[string]service.CodexCatalogMetadata{
		"gpt-5.6-sol": {
			Provider:            "OpenAI Codex",
			OwnedBy:             "codex",
			ContextWindowTokens: 272000,
			BillingMode:         service.CodexCatalogBillingMode,
		},
	})

	require.Equal(t, "OpenAI Codex", entries[0].Provider)
	require.Equal(t, "codex", entries[0].OwnedBy)
	require.NotNil(t, entries[0].ContextWindow)
	require.Equal(t, 272000, entries[0].ContextWindow.ContextLength)
	require.Equal(t, service.CodexCatalogBillingMode, entries[0].BillingMode)
	require.Equal(t, 5.0, entries[0].InputPrice)
	require.Equal(t, 30.0, entries[0].OutputPrice)
	require.NotNil(t, entries[0].Pricing)
	require.Equal(t, 5.0, entries[0].Pricing.Input)
	require.Equal(t, 30.0, entries[0].Pricing.Output)
	require.Nil(t, entries[0].Pricing.CachedInput)
	require.Nil(t, entries[0].Pricing.CacheWrite)
	require.Equal(t, "usd_per_1m_tokens", entries[0].Pricing.Unit)
	require.Equal(t, 0.000005, entries[0].Pricing.Prompt)
	require.Equal(t, 0.00003, entries[0].Pricing.Completion)
	require.Equal(t, "usd_per_token", entries[0].Pricing.CompatibilityUnit)
	require.Equal(t, service.CodexOpenAIReferencePricingScope, entries[0].Pricing.Scope)

	payload, err := common.Marshal(buildOpenAIModelFromCatalog(entries[0]))
	require.NoError(t, err)
	var public map[string]any
	require.NoError(t, common.Unmarshal(payload, &public))
	pricing, ok := public["pricing"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 5, pricing["input"])
	require.EqualValues(t, 30, pricing["output"])
	require.NotContains(t, pricing, "cached_input")
	require.NotContains(t, pricing, "cache_write")
	require.Equal(t, "usd_per_1m_tokens", pricing["unit"])
	require.EqualValues(t, 0.000005, pricing["prompt"])
	require.EqualValues(t, 0.00003, pricing["completion"])
	require.Equal(t, "usd_per_token", pricing["compatibility_unit"])
	require.Equal(t, service.CodexOpenAIReferencePricingScope, pricing["scope"])
	require.Equal(t, service.CodexCatalogBillingMode, public["billing_mode"])
	contextWindow, ok := public["context_window"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 272000, contextWindow["context_length"])
	require.NotContains(t, contextWindow, "max_tokens")
	require.EqualValues(t, 128000, contextWindow["max_completion_tokens"])
}

func TestApplyCodexMetadataMapsLegacyMaxTokensToContextLength(t *testing.T) {
	entries := []dto.ModelCatalogEntry{{ModelName: "gpt-legacy-codex"}}

	applyCodexMetadataToCatalogEntries(entries, map[string]service.CodexCatalogMetadata{
		"gpt-legacy-codex": {MaxTokens: 128000},
	})

	require.NotNil(t, entries[0].ContextWindow)
	require.Equal(t, 128000, entries[0].ContextWindow.ContextLength)
	payload, err := common.Marshal(entries[0].ContextWindow)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "max_tokens")
}

func TestRetrieveModelUsesPromotedCodexMetadataAndOfficialOutputFallback(t *testing.T) {
	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1002,
		Username: "model-detail-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group: "default", Model: "gpt-5.6-sol", ChannelId: 5, Enabled: true,
	}).Error)
	discovery, err := common.Marshal(map[string]any{
		"slug":           "gpt-5.6-sol",
		"visibility":     "list",
		"context_window": 272000,
	})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.CodexCatalogCandidate{
		ChannelID:           5,
		ModelName:           "gpt-5.6-sol",
		Promoted:            true,
		DisplayName:         "OpenAI Codex GPT-5.6 Sol",
		Provider:            "OpenAI Codex",
		OwnedBy:             "codex",
		DiscoveryMetadata:   string(discovery),
		ContextWindowTokens: 272000,
		MaxTokens:           272000,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models/gpt-5.6-sol", nil)
	ctx.Params = gin.Params{{Key: "model", Value: "gpt-5.6-sol"}}
	ctx.Set("id", 1002)

	RetrieveModel(ctx, constant.ChannelTypeOpenAI)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload dto.OpenAIModels
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "gpt-5.6-sol", payload.Id)
	require.Equal(t, "codex", payload.OwnedBy)
	require.NotNil(t, payload.ContextWindow)
	require.Equal(t, 272000, payload.ContextWindow.ContextLength)
	require.Equal(t, 128000, payload.ContextWindow.MaxCompletionTokens)
}

func TestRetrieveModelRejectsModelOutsideTokenLimit(t *testing.T) {
	withSelfUseModeEnabled(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1003,
		Username: "model-detail-limited-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group: "default", Model: "gpt-5.6-sol", ChannelId: 5, Enabled: true,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models/gpt-5.6-sol", nil)
	ctx.Params = gin.Params{{Key: "model", Value: "gpt-5.6-sol"}}
	ctx.Set("id", 1003)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"gpt-5.6-terra": true,
	})

	RetrieveModel(ctx, constant.ChannelTypeOpenAI)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Error types.OpenAIError `json:"error"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "model_not_found", payload.Error.Code)
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
	_ = setupModelListControllerTestDB(t)

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
