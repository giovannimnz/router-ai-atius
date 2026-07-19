package modelcatalog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gormName := strings.ReplaceAll(t.Name(), "/", "_")
	ginelessDSN := fmt.Sprintf("file:%s?mode=memory&cache=shared", gormName)

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	ratio_setting.InitRatioSettings()

	db, err := gorm.Open(sqlite.Open(ginelessDSN), &gorm.Config{})
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

func withCatalogBillingConfig(t *testing.T, modes map[string]string, exprs map[string]string) {
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

func pricingByName(pricings []model.Pricing) map[string]model.Pricing {
	out := make(map[string]model.Pricing, len(pricings))
	for _, pricing := range pricings {
		out[pricing.ModelName] = pricing
	}
	return out
}

func TestModelCatalogBuildEntries(t *testing.T) {
	withCatalogBillingConfig(t, map[string]string{
		"zz-tiered-catalog-model": "tiered_expr",
	}, map[string]string{
		"zz-tiered-catalog-model": `tier("base", p * 1 + c * 2)`,
	})

	db := setupModelCatalogTestDB(t)

	require.NoError(t, db.Create(&[]model.Channel{
		{Id: 11, Type: constant.ChannelTypeOpenAI, Name: "catalog-openai", Status: common.ChannelStatusEnabled},
		{Id: 12, Type: constant.ChannelTypeAnthropic, Name: "catalog-anthropic", Status: common.ChannelStatusEnabled},
	}).Error)

	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "gpt-5", ChannelId: 11, Enabled: true},
		{Group: "default", Model: "zz-missing-model", ChannelId: 11, Enabled: true},
		{Group: "default", Model: "zz-tiered-catalog-model", ChannelId: 11, Enabled: true},
		{Group: "default", Model: "zz-embed-custom", ChannelId: 12, Enabled: true},
	}).Error)

	require.NoError(t, db.Create(&model.Model{
		ModelName: "zz-embed-custom",
		NameRule:  model.NameRuleExact,
		Status:    1,
		Endpoints: `{"embeddings":{"path":"/v1/embeddings","method":"POST"}}`,
	}).Error)

	model.RefreshPricing()
	pricingMap := pricingByName(model.GetPricing())

	entries := BuildCatalogEntries([]model.Pricing{
		pricingMap["gpt-5"],
		pricingMap["zz-missing-model"],
		pricingMap["zz-tiered-catalog-model"],
		pricingMap["zz-embed-custom"],
	}, map[string]string{
		"gpt-5":           "openai",
		"zz-embed-custom": "anthropic",
	})

	byName := make(map[string]struct {
		Source    string
		Estimated bool
		OwnedBy   string
		Labels    []string
		Expr      string
		HasPrice  bool
	})
	for _, entry := range entries {
		byName[entry.ModelName] = struct {
			Source    string
			Estimated bool
			OwnedBy   string
			Labels    []string
			Expr      string
			HasPrice  bool
		}{
			Source:    entry.PricingSource,
			Estimated: entry.PricingEstimated,
			OwnedBy:   entry.OwnedBy,
			Labels:    entry.SupportedEndpointTypeLabels,
			Expr:      entry.BillingExpr,
			HasPrice:  entry.Pricing != nil,
		}
	}

	require.Equal(t, "model_ratio", byName["gpt-5"].Source)
	require.False(t, byName["gpt-5"].Estimated)
	require.Equal(t, "openai", byName["gpt-5"].OwnedBy)
	require.Contains(t, byName["gpt-5"].Labels, "OpenAI-Compatible")
	require.True(t, byName["gpt-5"].HasPrice)

	require.Equal(t, "missing", byName["zz-missing-model"].Source)
	require.True(t, byName["zz-missing-model"].Estimated)
	require.False(t, byName["zz-missing-model"].HasPrice)

	require.Equal(t, "billing_expr", byName["zz-tiered-catalog-model"].Source)
	require.False(t, byName["zz-tiered-catalog-model"].Estimated)
	require.NotEmpty(t, byName["zz-tiered-catalog-model"].Expr)
	require.False(t, byName["zz-tiered-catalog-model"].HasPrice)

	require.Contains(t, byName["zz-embed-custom"].Labels, "Embeddings")
	require.Equal(t, "anthropic", byName["zz-embed-custom"].OwnedBy)
}

func TestModelCatalogEndpointTypeLabelsDeduplicateResponseModes(t *testing.T) {
	labels := EndpointTypeLabels([]constant.EndpointType{
		constant.EndpointTypeOpenAIResponse,
		constant.EndpointTypeOpenAIResponseCompact,
		constant.EndpointTypeOpenAI,
	})

	require.Equal(t, []string{"OpenAI-Responses", "OpenAI-Compatible"}, labels)
}

func TestModelCatalogPricingPublishesExplicitUnit(t *testing.T) {
	ratio_setting.InitRatioSettings()
	entry := BuildCatalogEntry(model.Pricing{
		ModelName:       "gpt-5",
		ModelRatio:      1.25,
		CompletionRatio: 4,
	}, map[string]string{"gpt-5": "openai"})
	require.NotNil(t, entry.Pricing)
	require.Equal(t, "usd_per_1m_tokens", entry.Pricing.Unit)

	payload, err := common.Marshal(entry.Pricing)
	require.NoError(t, err)
	require.JSONEq(t, `{"input":2.5,"output":10,"unit":"usd_per_1m_tokens"}`, string(payload))
}

func TestModelCatalogPricingPrefersCanonicalDollarCost(t *testing.T) {
	cacheRatio := 0.1
	createCacheRatio := 1.25
	entry := BuildCatalogEntry(model.Pricing{
		ModelName:        "gpt-5.6-sol",
		ModelRatio:       37.5,
		CompletionRatio:  8,
		InputPrice:       5,
		OutputPrice:      30,
		UseDollarCost:    true,
		CacheRatio:       &cacheRatio,
		CreateCacheRatio: &createCacheRatio,
	}, map[string]string{"gpt-5.6-sol": "codex"})

	require.Equal(t, "input_output_price", entry.PricingSource)
	require.False(t, entry.PricingEstimated)
	require.Equal(t, "dollar_cost", entry.BillingMode)
	require.Equal(t, 5.0, entry.InputPrice)
	require.Equal(t, 30.0, entry.OutputPrice)
	require.NotNil(t, entry.Pricing)
	require.Equal(t, 5.0, entry.Pricing.Input)
	require.Equal(t, 30.0, entry.Pricing.Output)
	require.NotNil(t, entry.Pricing.CachedInput)
	require.Equal(t, 0.5, *entry.Pricing.CachedInput)
	require.NotNil(t, entry.Pricing.CacheWrite)
	require.Equal(t, 6.25, *entry.Pricing.CacheWrite)
}
