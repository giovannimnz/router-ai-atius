package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useAtiusLocalMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := DB
	originalDatabaseType := common.MainDatabaseType()
	t.Cleanup(func() {
		DB = originalDB
		common.SetMainDatabaseType(originalDatabaseType)
	})

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&Channel{}))
	return db
}

func TestMigrateAtiusLocalChannelName(t *testing.T) {
	db := useAtiusLocalMigrationTestDB(t)

	channel := Channel{
		Name:   "Atius Local Embeddings",
		Type:   constant.ChannelTypeAtiusLocalEmbeddings,
		Status: common.ChannelStatusEnabled,
		Models: constant.AtiusLocalEmbeddingModel + "," + constant.AtiusLocalRerankerModel,
	}
	require.NoError(t, db.Create(&channel).Error)

	updated, err := MigrateAtiusLocalChannelName()
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)
	var got struct {
		Name   string
		Type   int
		Models string
	}
	require.NoError(t, db.Model(&Channel{}).Select("name", "type", "models").First(&got, channel.Id).Error)
	require.Equal(t, "Atius Local", got.Name)
	require.Equal(t, constant.ChannelTypeAtiusLocalEmbeddings, got.Type)
	require.Equal(t, "embedding-gte-v1,reranker-gte-v1", got.Models)

	updated, err = MigrateAtiusLocalChannelName()
	require.NoError(t, err)
	require.Zero(t, updated)
}

func TestMigrateAtiusLocalEmbeddingEndpoint(t *testing.T) {
	db := useAtiusLocalMigrationTestDB(t)
	legacyBaseURL := "http://10.21.1.21:3115"
	legacySettings := `{"advanced_custom":{"advanced_routes":[{"incoming_path":"/v1/embeddings","upstream_path":"http://10.21.1.21:3115/v1/embeddings","converter":"none","auth":{"type":"none"}},{"incoming_path":"/v1/rerank","upstream_path":"http://10.21.1.21:31216/rerank","converter":"jina_rerank_to_tei_native","auth":{"type":"none"}}]}}`
	channel := Channel{
		Name:          constant.AtiusLocalChannelName,
		Type:          constant.ChannelTypeAtiusLocalEmbeddings,
		Status:        common.ChannelStatusEnabled,
		BaseURL:       &legacyBaseURL,
		Models:        constant.AtiusLocalEmbeddingModel + "," + constant.AtiusLocalRerankerModel,
		OtherSettings: legacySettings,
	}
	require.NoError(t, db.Create(&channel).Error)

	updated, err := MigrateAtiusLocalEmbeddingEndpoint()
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)

	var got struct {
		BaseURL       *string
		OtherSettings string `gorm:"column:settings"`
	}
	require.NoError(t, db.Model(&Channel{}).Select("base_url", "settings").First(&got, channel.Id).Error)
	require.NotNil(t, got.BaseURL)
	require.Equal(t, "http://10.21.1.21:31115", *got.BaseURL)
	require.Contains(t, got.OtherSettings, "http://10.21.1.21:31115/v1/embeddings")
	require.NotContains(t, got.OtherSettings, "http://10.21.1.21:3115/v1/embeddings")
	require.Contains(t, got.OtherSettings, "http://10.21.1.21:31216/rerank")

	updated, err = MigrateAtiusLocalEmbeddingEndpoint()
	require.NoError(t, err)
	require.Zero(t, updated)
}
