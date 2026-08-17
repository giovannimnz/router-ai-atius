package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func dollarCostOther(source string, inputPrice, outputPrice float64) string {
	metadata := map[string]any{
		"use_dollar_cost": true,
		"input_price":     inputPrice,
		"output_price":    outputPrice,
		"group_ratio":     1,
	}
	if source != "" {
		metadata["billing_source"] = source
	}
	encoded, err := common.Marshal(metadata)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func TestClassifyAtiusDollarCostLog(t *testing.T) {
	metadata := atiusDollarCostLogMetadata{
		UseDollarCost: true,
		InputPrice:    5,
		OutputPrice:   30,
		GroupRatio:    1,
	}
	legacy := Log{Id: 1, PromptTokens: 544_641, CompletionTokens: 3_240, Quota: 1_410_202_500_000}

	correction, ok, err := classifyAtiusDollarCostLog(legacy, metadata)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(1_410_203), correction.NewQuota)

	alreadyCorrect := legacy
	alreadyCorrect.Quota = int(correction.NewQuota)
	_, ok, err = classifyAtiusDollarCostLog(alreadyCorrect, metadata)
	require.NoError(t, err)
	assert.False(t, ok)

	ambiguous := legacy
	ambiguous.Quota = 123
	_, ok, err = classifyAtiusDollarCostLog(ambiguous, metadata)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAtiusDollarCostQuotaHandlesCacheSemantics(t *testing.T) {
	t.Run("OpenAI cache fields are excluded from base prompt", func(t *testing.T) {
		log := Log{Id: 1, PromptTokens: 1_000, CompletionTokens: 100}
		metadata := atiusDollarCostLogMetadata{
			UseDollarCost:       true,
			InputPrice:          5,
			OutputPrice:         30,
			GroupRatio:          1,
			CacheTokens:         100,
			CacheRatio:          0.1,
			CacheCreationTokens: 100,
			CacheCreationRatio:  1.25,
		}
		quota, err := atiusDollarCostQuota(log, metadata, true)
		require.NoError(t, err)
		assert.Equal(t, int64(3838), quota)
	})

	t.Run("Anthropic split cache creation uses frozen ratios", func(t *testing.T) {
		log := Log{Id: 2, PromptTokens: 1_000, CompletionTokens: 0}
		metadata := atiusDollarCostLogMetadata{
			UseDollarCost:         true,
			InputPrice:            2,
			GroupRatio:            1,
			UsageSemantic:         "anthropic",
			CacheTokens:           100,
			CacheRatio:            0.1,
			CacheCreationTokens:   50,
			CacheCreationRatio:    1,
			CacheCreationTokens5m: 10,
			CacheCreationRatio5m:  1.25,
			CacheCreationTokens1h: 20,
			CacheCreationRatio1h:  2,
		}
		quota, err := atiusDollarCostQuota(log, metadata, true)
		require.NoError(t, err)
		assert.Equal(t, int64(1083), quota)
	})

	t.Run("legacy split cache fields preserve Claude semantics", func(t *testing.T) {
		log := Log{Id: 3, PromptTokens: 1_000}
		metadata := atiusDollarCostLogMetadata{
			UseDollarCost:         true,
			InputPrice:            2,
			GroupRatio:            1,
			CacheTokens:           100,
			CacheRatio:            0.1,
			CacheCreationTokens:   50,
			CacheCreationRatio:    1,
			CacheCreationTokens5m: 10,
			CacheCreationRatio5m:  1.25,
			CacheCreationTokens1h: 20,
			CacheCreationRatio1h:  2,
		}
		quota, err := atiusDollarCostQuota(log, metadata, true)
		require.NoError(t, err)
		assert.Equal(t, int64(1083), quota)
	})
}

func TestClassifyAtiusDollarCostLogExcludesTieredAndSurcharges(t *testing.T) {
	log := Log{Id: 1, PromptTokens: 1_000, Quota: 37_500_000_000}
	for name, metadata := range map[string]atiusDollarCostLogMetadata{
		"tiered":           {UseDollarCost: true, BillingMode: "tiered_expr", InputPrice: 75, GroupRatio: 1},
		"web search":       {UseDollarCost: true, WebSearchCallCount: 1, InputPrice: 75, GroupRatio: 1},
		"file search":      {UseDollarCost: true, FileSearchCallCount: 1, InputPrice: 75, GroupRatio: 1},
		"audio surcharge":  {UseDollarCost: true, AudioInputTokenCount: 1, InputPrice: 75, GroupRatio: 1},
		"image generation": {UseDollarCost: true, ImageGenerationCall: true, InputPrice: 75, GroupRatio: 1},
	} {
		t.Run(name, func(t *testing.T) {
			_, ok, err := classifyAtiusDollarCostLog(log, metadata)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	}
}

func newAtiusDollarCostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousLogDB := LOG_DB
	previousType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	DB = db
	LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
		common.SetLogDatabaseType(previousLogType)
		initCol()
	})
	require.NoError(t, db.AutoMigrate(&Log{}, &User{}, &Token{}, &Channel{}, &Option{}, &QuotaData{}))
	return db
}

func TestReconcileAtiusDollarCostV1(t *testing.T) {
	db := newAtiusDollarCostTestDB(t)

	const oldWalletDollarQuota = int64(37_500_000_000)
	const walletAlreadyCorrect = int64(500)
	const walletRatioQuota = int64(37_500)
	const walletAmbiguousQuota = int64(123_456)
	const userQuotaBefore = int64(9_000_000)
	const userUsedBefore = int64(37_500_100_000)
	const tokenRemainBefore = int64(3_000_000)
	const tokenUsedBefore = int64(37_500_200_000)
	const channelUsedBefore = int64(37_500_300_000)

	require.NoError(t, db.Create(&User{Id: 2, Username: "billing", Quota: int(userQuotaBefore), UsedQuota: int(userUsedBefore), RequestCount: 77}).Error)
	require.NoError(t, db.Create(&Token{Id: 1, UserId: 2, Key: "test-key", Name: "billing", RemainQuota: int(tokenRemainBefore), UsedQuota: int(tokenUsedBefore)}).Error)
	require.NoError(t, db.Create(&Channel{Id: 5, Name: "billing", UsedQuota: channelUsedBefore}).Error)

	logs := []Log{
		{Id: 1, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, ModelName: "dollar-bad", Quota: int(oldWalletDollarQuota), ChannelId: 5, TokenId: 1, Other: dollarCostOther("wallet", 75, 0)},
		{Id: 2, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, ModelName: "dollar-correct", Quota: int(walletAlreadyCorrect), ChannelId: 5, TokenId: 1, Other: dollarCostOther("wallet", 1, 0)},
		{Id: 3, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, ModelName: "ratio-legacy", Quota: int(walletRatioQuota), ChannelId: 5, TokenId: 1, Other: `{"billing_source":"wallet","model_ratio":37.5,"group_ratio":1}`},
		{Id: 4, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, ModelName: "dollar-missing-source", Quota: 1_000_000_000, ChannelId: 5, TokenId: 1, Other: dollarCostOther("", 2, 0)},
		{Id: 5, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, ModelName: "dollar-missing-source-2", Quota: 1_500_000_000, ChannelId: 5, TokenId: 1, Other: dollarCostOther("", 3, 0)},
		{Id: 6, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, ModelName: "dollar-ambiguous", Quota: int(walletAmbiguousQuota), ChannelId: 5, TokenId: 1, Other: dollarCostOther("wallet", 4, 0)},
	}
	require.NoError(t, db.Create(&logs).Error)
	require.NoError(t, db.Create(&QuotaData{
		UserID: 2, Username: "", ModelName: "dollar-bad", CreatedAt: 7_200,
		TokenID: 1, ChannelID: 5, Count: 1, Quota: int(oldWalletDollarQuota), TokenUsed: 1_000,
	}).Error)

	result, err := ReconcileAtiusDollarCostV1(db)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.CorrectedLogs)
	assert.Equal(t, 6, result.MaxLogID)
	assert.Equal(t, int64(1), result.WalletCorrectedLogs)
	assert.Equal(t, oldWalletDollarQuota-37_500, result.WalletAdjustmentQuota)
	assert.Equal(t, int64(2), result.MissingSourceLogs)
	assert.Equal(t, int64(2_499_997_500), result.LogOnlyAdjustmentQuota)
	assert.Equal(t, int64(10_800), result.HistoricalLogCutoff)

	var migrated []Log
	require.NoError(t, db.Order("id").Find(&migrated).Error)
	assert.Equal(t, 37_500, migrated[0].Quota)
	assert.Equal(t, 500, migrated[1].Quota)
	assert.Equal(t, int(walletRatioQuota), migrated[2].Quota)
	assert.Equal(t, 1_000, migrated[3].Quota)
	assert.Equal(t, 1_500, migrated[4].Quota)
	assert.Equal(t, int(walletAmbiguousQuota), migrated[5].Quota)

	walletAdjustment := oldWalletDollarQuota - 37_500
	var user User
	require.NoError(t, db.First(&user, 2).Error)
	assert.Equal(t, int(userUsedBefore-walletAdjustment), user.UsedQuota)
	assert.Equal(t, int(userQuotaBefore+walletAdjustment), user.Quota)
	assert.Equal(t, 77, user.RequestCount)

	var token Token
	require.NoError(t, db.First(&token, 1).Error)
	assert.Equal(t, int(tokenUsedBefore-walletAdjustment), token.UsedQuota)
	assert.Equal(t, int(tokenRemainBefore+walletAdjustment), token.RemainQuota)

	var channel Channel
	require.NoError(t, db.First(&channel, 5).Error)
	assert.Equal(t, channelUsedBefore-walletAdjustment, channel.UsedQuota)

	var quotaData []QuotaData
	require.NoError(t, db.Order("model_name").Find(&quotaData).Error)
	require.Len(t, quotaData, 1)
	assert.Equal(t, int(oldWalletDollarQuota), quotaData[0].Quota, "legacy aggregate is intentionally not rewritten")

	second, err := ReconcileAtiusDollarCostV1(db)
	require.NoError(t, err)
	assert.True(t, second.AlreadyApplied)
	assert.Equal(t, result.CorrectedLogs, second.CorrectedLogs)
}

func TestReconcileAtiusDollarCostV1RejectsSubscriptionWithoutLedger(t *testing.T) {
	db := newAtiusDollarCostTestDB(t)
	require.NoError(t, db.Create(&Log{
		Id: 1, UserId: 2, Type: LogTypeConsume, PromptTokens: 1_000,
		Quota: 1_500_000_000, ChannelId: 5, TokenId: 1,
		Other: dollarCostOther("subscription", 3, 0),
	}).Error)

	_, err := ReconcileAtiusDollarCostV1(db)
	require.ErrorContains(t, err, `unsupported billing source "subscription"`)
	var log Log
	require.NoError(t, db.First(&log, 1).Error)
	assert.Equal(t, 1_500_000_000, log.Quota)
}

func TestReconcileAtiusDollarCostV1RejectsWalletWithoutRequiredIdentifiers(t *testing.T) {
	for name, mutate := range map[string]func(*Log){
		"user":    func(log *Log) { log.UserId = 0 },
		"token":   func(log *Log) { log.TokenId = 0 },
		"channel": func(log *Log) { log.ChannelId = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			db := newAtiusDollarCostTestDB(t)
			log := Log{
				Id: 1, UserId: 2, Type: LogTypeConsume, PromptTokens: 1_000,
				Quota: 1_500_000_000, ChannelId: 5, TokenId: 1,
				Other: dollarCostOther("wallet", 3, 0),
			}
			mutate(&log)
			require.NoError(t, db.Create(&log).Error)

			_, err := ReconcileAtiusDollarCostV1(db)
			require.ErrorContains(t, err, "requires positive user, token, and channel identifiers")
			var unchanged Log
			require.NoError(t, db.First(&unchanged, 1).Error)
			assert.Equal(t, 1_500_000_000, unchanged.Quota)
		})
	}
}

func TestReconcileAtiusDollarCostV1RejectsTokenOwnedByAnotherUser(t *testing.T) {
	db := newAtiusDollarCostTestDB(t)
	const oldQuota = int64(1_500_000_000)
	require.NoError(t, db.Create(&User{Id: 2, Username: "billing", Quota: 100, UsedQuota: int(oldQuota)}).Error)
	require.NoError(t, db.Create(&Token{Id: 1, UserId: 99, Key: "wrong-owner", RemainQuota: 100, UsedQuota: int(oldQuota)}).Error)
	require.NoError(t, db.Create(&Channel{Id: 5, Name: "billing", UsedQuota: oldQuota}).Error)
	require.NoError(t, db.Create(&Log{
		Id: 1, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000,
		Quota: int(oldQuota), ChannelId: 5, TokenId: 1, Other: dollarCostOther("wallet", 3, 0),
	}).Error)

	_, err := ReconcileAtiusDollarCostV1(db)
	require.ErrorContains(t, err, "token 1 for user 2")
	var unchanged Log
	require.NoError(t, db.First(&unchanged, 1).Error)
	assert.Equal(t, int(oldQuota), unchanged.Quota)
}

func TestApplyAtiusDollarCostCorrectionsGuardsOldQuota(t *testing.T) {
	db := newAtiusDollarCostTestDB(t)
	require.NoError(t, db.Create(&Log{Id: 1, Type: LogTypeConsume, Quota: 999}).Error)

	err := applyAtiusDollarCostCorrections(db, []atiusDollarCostCorrection{{LogID: 1, OldQuota: 1_000, NewQuota: 1}})
	require.ErrorContains(t, err, "guard matched 0 of 1 rows")
	var unchanged Log
	require.NoError(t, db.First(&unchanged, 1).Error)
	assert.Equal(t, 999, unchanged.Quota)
}

func TestReconcileAtiusDollarCostV1FailsClosedOnUnexpectedScope(t *testing.T) {
	db := newAtiusDollarCostTestDB(t)
	const oldQuota = int64(37_500_000_000)
	const expectedQuota = int64(37_500)
	const adjustment = oldQuota - expectedQuota
	require.NoError(t, db.Create(&User{Id: 2, Username: "billing", Quota: 100, UsedQuota: int(oldQuota)}).Error)
	require.NoError(t, db.Create(&Token{Id: 1, UserId: 2, Key: "test-key", Name: "billing", RemainQuota: 100, UsedQuota: int(oldQuota)}).Error)
	require.NoError(t, db.Create(&Channel{Id: 5, Name: "billing", UsedQuota: oldQuota}).Error)
	require.NoError(t, db.Create(&Log{Id: 1, UserId: 2, CreatedAt: 7_200, Type: LogTypeConsume, PromptTokens: 1_000, Quota: int(oldQuota), ChannelId: 5, TokenId: 1, Other: dollarCostOther("wallet", 75, 0)}).Error)

	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS", "2")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_DELTA", fmt.Sprint(adjustment))
	_, err := ReconcileAtiusDollarCostV1(db)
	require.ErrorContains(t, err, "does not match expected")
	var unchanged Log
	require.NoError(t, db.First(&unchanged, 1).Error)
	assert.Equal(t, int(oldQuota), unchanged.Quota)

	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS", "1")
	result, err := ReconcileAtiusDollarCostV1(db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.CorrectedLogs)
	assert.Equal(t, adjustment, result.WalletAdjustmentQuota)
}

func TestReconcileAtiusDollarCostV1RollsBackMalformedBillingMetadata(t *testing.T) {
	db := newAtiusDollarCostTestDB(t)
	require.NoError(t, db.Create(&User{Id: 2, Username: "billing", Quota: 100, UsedQuota: 100}).Error)
	require.NoError(t, db.Create(&Token{Id: 1, UserId: 2, Key: "test-key", Name: "billing", RemainQuota: 100, UsedQuota: 100}).Error)
	require.NoError(t, db.Create(&Channel{Id: 5, Name: "billing", UsedQuota: 100}).Error)
	require.NoError(t, db.Create(&Log{Id: 1, UserId: 2, Type: LogTypeConsume, PromptTokens: 1_000, Quota: 37_500_000_000, ChannelId: 5, TokenId: 1, Other: dollarCostOther("wallet", 75, 0)}).Error)
	require.NoError(t, db.Create(&Log{Id: 2, UserId: 2, Type: LogTypeConsume, Quota: 100, ChannelId: 5, TokenId: 1, Other: `{"billing_source":"wallet"`}).Error)
	require.NoError(t, db.Create(&Log{Id: 3, UserId: 2, Type: LogTypeConsume, Quota: 100, ChannelId: 5, TokenId: 1, Other: `{"use_dollar_cost":`}).Error)

	_, err := ReconcileAtiusDollarCostV1(db)
	require.ErrorContains(t, err, "decode billing metadata")

	var log Log
	require.NoError(t, db.First(&log, 1).Error)
	assert.Equal(t, 37_500_000_000, log.Quota)
	var markerCount int64
	require.NoError(t, db.Model(&Option{}).Where(commonKeyCol+" = ?", atiusDollarCostReconciliationV1Option).Count(&markerCount).Error)
	assert.Zero(t, markerCount)
}

func TestMigrateAtiusDollarCostV1RequiresAuditedExpectations(t *testing.T) {
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_ENABLED", "true")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS", "")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_DELTA", "")

	err := MigrateAtiusDollarCostV1()
	require.ErrorContains(t, err, "EXPECTED_LOGS is required")
}

func TestMigrateAtiusDollarCostV1FailsClosedWithSeparateLogDatabase(t *testing.T) {
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_ENABLED", "true")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS", "0")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_DELTA", "0")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_LOGS", "0")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_DELTA", "0")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_LOGS", "0")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_DELTA", "0")
	t.Setenv("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_CUTOFF", "0")
	t.Setenv("LOG_SQL_DSN", "postgres://separate-log-db")

	err := MigrateAtiusDollarCostV1()
	require.ErrorContains(t, err, "cannot run with LOG_SQL_DSN")
}
