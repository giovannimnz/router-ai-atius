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
