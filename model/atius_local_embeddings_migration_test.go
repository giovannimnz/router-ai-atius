package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMigrateAtiusLocalRerankerAlias(t *testing.T) {
	previousDB := DB
	previousLogDB := LOG_DB
	previousType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
		common.SetLogDatabaseType(previousLogType)
		initCol()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}, &Token{}, &Model{}, &Option{}, &Log{}, &QuotaData{}))

	testModel := constant.LegacyAtiusLocalRerankerModel
	channel := Channel{
		Id:        11,
		Type:      constant.ChannelTypeAtiusLocalEmbeddings,
		Name:      constant.AtiusLocalEmbeddingsChannelName,
		Models:    constant.AtiusLocalEmbeddingModel + "," + constant.LegacyAtiusLocalRerankerModel,
		Group:     "default",
		Status:    common.ChannelStatusEnabled,
		TestModel: &testModel,
	}
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&Ability{Group: "default", Model: constant.LegacyAtiusLocalRerankerModel, ChannelId: 11, Enabled: true}).Error)
	require.NoError(t, db.Create(&Token{Id: 9, Name: "limited", ModelLimitsEnabled: true, ModelLimits: constant.AtiusLocalEmbeddingModel + "," + constant.LegacyAtiusLocalRerankerModel}).Error)
	require.NoError(t, db.Create(&Model{ModelName: constant.LegacyAtiusLocalRerankerModel}).Error)
	require.NoError(t, db.Create(&Option{Key: "ModelRatio", Value: `{"reranker-gte-multilingual-v1":0.01115,"embedding-gte-v1":0.035}`}).Error)
	require.NoError(t, db.Create(&Log{Id: 1, Type: LogTypeConsume, ModelName: constant.LegacyAtiusLocalRerankerModel, Quota: 10}).Error)
	require.NoError(t, db.Create(&QuotaData{Id: 1, ModelName: constant.LegacyAtiusLocalRerankerModel, Quota: 10}).Error)

	require.NoError(t, MigrateAtiusLocalRerankerAlias())
	require.NoError(t, MigrateAtiusLocalRerankerAlias(), "migration must be idempotent")
	require.NoError(t, MigrateAtiusLocalRerankerLogAlias(LOG_DB))
	require.NoError(t, MigrateAtiusLocalRerankerLogAlias(LOG_DB), "log migration must be idempotent")

	var migratedChannel Channel
	require.NoError(t, db.First(&migratedChannel, 11).Error)
	assert.Equal(t, "embedding-gte-v1,reranker-gte-v1", migratedChannel.Models)
	require.NotNil(t, migratedChannel.TestModel)
	assert.Equal(t, constant.AtiusLocalRerankerModel, *migratedChannel.TestModel)

	var abilities []Ability
	require.NoError(t, db.Where("channel_id = ?", 11).Order("model").Find(&abilities).Error)
	require.Len(t, abilities, 2)
	assert.Equal(t, constant.AtiusLocalEmbeddingModel, abilities[0].Model)
	assert.Equal(t, constant.AtiusLocalRerankerModel, abilities[1].Model)

	var token Token
	require.NoError(t, db.First(&token, 9).Error)
	assert.Equal(t, "embedding-gte-v1,reranker-gte-v1", token.ModelLimits)

	var modelMeta Model
	require.NoError(t, db.Where("model_name = ?", constant.AtiusLocalRerankerModel).First(&modelMeta).Error)

	var option Option
	require.NoError(t, db.Where(commonKeyCol+" = ?", "ModelRatio").First(&option).Error)
	assert.JSONEq(t, `{"reranker-gte-v1":0.01115,"embedding-gte-v1":0.035}`, option.Value)
	assert.NotContains(t, option.Value, constant.LegacyAtiusLocalRerankerModel)

	var legacyLogCount int64
	require.NoError(t, db.Model(&Log{}).Where("model_name = ?", constant.LegacyAtiusLocalRerankerModel).Count(&legacyLogCount).Error)
	assert.Zero(t, legacyLogCount)
	var migratedLog Log
	require.NoError(t, db.First(&migratedLog, 1).Error)
	assert.Equal(t, constant.AtiusLocalRerankerModel, migratedLog.ModelName)

	var quotaData QuotaData
	require.NoError(t, db.First(&quotaData, 1).Error)
	assert.Equal(t, constant.AtiusLocalRerankerModel, quotaData.ModelName)
}
