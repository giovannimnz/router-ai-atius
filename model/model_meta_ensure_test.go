package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnsureExactModelMetadataCreatesOnlyMissingModels(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&Vendor{}, &Model{}))

	existingVendor := Vendor{Name: "Existing", Icon: "Other", Status: 1}
	require.NoError(t, db.Create(&existingVendor).Error)
	existingModel := Model{
		ModelName:    "gpt-5.6-sol",
		VendorID:     existingVendor.Id,
		Status:       0,
		SyncOfficial: 1,
		Description:  "managed by administrator",
	}
	require.NoError(t, db.Create(&existingModel).Error)
	require.NoError(t, db.Model(&Model{}).Where("id = ?", existingModel.Id).Update("status", 0).Error)

	created, err := EnsureExactModelMetadata(
		[]string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-terra"},
		"OpenAI Codex",
		"OpenAI",
	)
	require.NoError(t, err)
	require.Equal(t, 1, created)

	var preserved Model
	require.NoError(t, db.Where("model_name = ?", "gpt-5.6-sol").First(&preserved).Error)
	require.Equal(t, existingVendor.Id, preserved.VendorID)
	require.Zero(t, preserved.Status)
	require.Equal(t, "managed by administrator", preserved.Description)

	var vendor Vendor
	require.NoError(t, db.Where("name = ?", "OpenAI Codex").First(&vendor).Error)
	var createdModel Model
	require.NoError(t, db.Where("model_name = ?", "gpt-5.6-terra").First(&createdModel).Error)
	require.Equal(t, vendor.Id, createdModel.VendorID)
	require.Equal(t, 1, createdModel.Status)
	require.Zero(t, createdModel.SyncOfficial)
	require.Equal(t, NameRuleExact, createdModel.NameRule)

	created, err = EnsureExactModelMetadata([]string{"gpt-5.6-terra"}, "OpenAI Codex", "OpenAI")
	require.NoError(t, err)
	require.Zero(t, created)
}

func TestEnsureAtiusLocalEmbeddingsMetadataCreatesMissingRerankerOnly(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&Vendor{}, &Model{}))

	existingVendor := Vendor{Name: "Atius", Icon: "AtlasCloud", Status: 1}
	require.NoError(t, db.Create(&existingVendor).Error)
	existingEmbedding := Model{
		ModelName:    "embedding-gte-v1",
		VendorID:     existingVendor.Id,
		Status:       1,
		SyncOfficial: 1,
		Description:  "managed by administrator",
		Tags:         "custom",
		Endpoints:    `{"embeddings":{"path":"/custom","method":"POST"}}`,
	}
	require.NoError(t, db.Create(&existingEmbedding).Error)

	created, err := EnsureAtiusLocalEmbeddingsMetadata()
	require.NoError(t, err)
	require.Equal(t, 1, created)

	var preserved Model
	require.NoError(t, db.Where("model_name = ?", "embedding-gte-v1").First(&preserved).Error)
	require.Equal(t, "managed by administrator", preserved.Description)
	require.Equal(t, "custom", preserved.Tags)
	require.Equal(t, `{"embeddings":{"path":"/custom","method":"POST"}}`, preserved.Endpoints)

	var reranker Model
	require.NoError(t, db.Where("model_name = ?", "reranker-gte-v1").First(&reranker).Error)
	require.Equal(t, existingVendor.Id, reranker.VendorID)
	require.Equal(t, "TEI GTE Reranker", reranker.Description)
	require.Equal(t, "AtlasCloud", reranker.Icon)
	require.Equal(t, "Reranker,Local TEI,Governor", reranker.Tags)
	require.Equal(t, `{"jina-rerank":{"method":"POST","path":"/v1/rerank"}}`, reranker.Endpoints)
	require.Equal(t, 1, reranker.Status)
	require.Equal(t, 1, reranker.SyncOfficial)
	require.Equal(t, NameRuleExact, reranker.NameRule)

	created, err = EnsureAtiusLocalEmbeddingsMetadata()
	require.NoError(t, err)
	require.Zero(t, created)
}
