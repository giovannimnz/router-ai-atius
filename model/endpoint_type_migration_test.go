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

func TestMigrateLegacyRerankerEndpointTypePreservesConfiguration(t *testing.T) {
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
	require.NoError(t, db.AutoMigrate(&Model{}, &PrefillGroup{}))

	legacyModel := Model{
		ModelName: "reranker-gte-v1",
		Endpoints: `{"jina-rerank":{"path":"/v1/rerank","method":"POST","timeout":30}}`,
		Status:    1,
	}
	require.NoError(t, db.Create(&legacyModel).Error)
	canonicalModel := Model{
		ModelName: "reranker-canonical",
		Endpoints: `{"jina-rerank":{"path":"/legacy"},"reranker":{"path":"/canonical","method":"POST"}}`,
		Status:    1,
	}
	require.NoError(t, db.Create(&canonicalModel).Error)
	group := PrefillGroup{
		Name:  "rerank-endpoints",
		Type:  "endpoint",
		Items: JSONValue(`{"jina-rerank":{"path":"/custom-rerank","method":"POST"}}`),
	}
	require.NoError(t, db.Create(&group).Error)

	updated, err := MigrateLegacyRerankerEndpointType()
	require.NoError(t, err)
	require.Equal(t, 3, updated)

	require.NoError(t, db.First(&legacyModel, legacyModel.Id).Error)
	require.JSONEq(t, `{"reranker":{"path":"/v1/rerank","method":"POST","timeout":30}}`, legacyModel.Endpoints)
	require.NoError(t, db.First(&canonicalModel, canonicalModel.Id).Error)
	require.JSONEq(t, `{"reranker":{"path":"/canonical","method":"POST"}}`, canonicalModel.Endpoints)
	require.NoError(t, db.First(&group, group.Id).Error)
	require.JSONEq(t, `{"reranker":{"path":"/custom-rerank","method":"POST"}}`, string(group.Items))

	updated, err = MigrateLegacyRerankerEndpointType()
	require.NoError(t, err)
	require.Zero(t, updated)
}

func TestNormalizeEndpointMetadataLeavesMalformedJSONUntouched(t *testing.T) {
	t.Parallel()

	raw := `{"jina-rerank":`
	normalized, changed := NormalizeEndpointMetadata(raw)
	require.False(t, changed)
	require.Equal(t, raw, normalized)
}
