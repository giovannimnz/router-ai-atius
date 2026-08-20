package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newQuotaDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousLogDB := LOG_DB
	previousType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
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
	require.NoError(t, db.AutoMigrate(&Log{}, &QuotaData{}, &Option{}))
	return db
}

func TestQuotaDashboardUsesReconciledLogsOnlyBeforeCutoff(t *testing.T) {
	db := newQuotaDashboardTestDB(t)
	require.NoError(t, db.Create(&Log{Id: 1, Type: LogTypeConsume, ModelName: "historical", CreatedAt: 3_600, Quota: 100, PromptTokens: 10}).Error)
	require.NoError(t, db.Create(&Log{Id: 2, Type: LogTypeConsume, ModelName: "future-log", CreatedAt: 7_200, Quota: 777, PromptTokens: 70}).Error)
	require.NoError(t, db.Create(&QuotaData{ModelName: "historical", CreatedAt: 3_600, Quota: 999, Count: 1, TokenUsed: 99}).Error)
	require.NoError(t, db.Create(&QuotaData{ModelName: "future-aggregate", CreatedAt: 7_200, Quota: 200, Count: 1, TokenUsed: 20}).Error)
	marker, err := common.Marshal(AtiusDollarCostReconciliationResult{Version: 1, HistoricalLogCutoff: 7_200})
	require.NoError(t, err)
	require.NoError(t, db.Create(&Option{Key: atiusDollarCostReconciliationV1Option, Value: string(marker)}).Error)

	rows, err := GetAllQuotaDates(0, 10_000, "")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	byModel := make(map[string]*QuotaData, len(rows))
	for _, row := range rows {
		byModel[row.ModelName] = row
	}
	assert.Equal(t, 100, byModel["historical"].Quota)
	assert.Equal(t, 10, byModel["historical"].TokenUsed)
	assert.Equal(t, 200, byModel["future-aggregate"].Quota)
	assert.NotContains(t, byModel, "future-log")
}

func TestQuotaDashboardWithoutMarkerKeepsDurableAggregates(t *testing.T) {
	db := newQuotaDashboardTestDB(t)
	require.NoError(t, db.Create(&Log{Id: 1, Type: LogTypeConsume, ModelName: "log-only", CreatedAt: 3_600, Quota: 100}).Error)
	require.NoError(t, db.Create(&QuotaData{ModelName: "aggregate", CreatedAt: 3_600, Quota: 200, Count: 1}).Error)

	rows, err := GetAllQuotaDates(0, 10_000, "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "aggregate", rows[0].ModelName)
	assert.Equal(t, 200, rows[0].Quota)
}
