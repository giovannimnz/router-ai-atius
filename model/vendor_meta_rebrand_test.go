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

func TestMigrateOpenAICodexVendorMovesModelsAndRetiresLegacyVendor(t *testing.T) {
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

	canonical := Vendor{Name: "OpenAI", Icon: "Other", Status: 0}
	legacy := Vendor{Name: "OpenAI Codex", Icon: "OpenAI", Status: 1}
	require.NoError(t, db.Create(&canonical).Error)
	require.NoError(t, db.Model(&Vendor{}).Where("id = ?", canonical.Id).Update("status", 0).Error)
	require.NoError(t, db.Create(&legacy).Error)
	item := Model{ModelName: "gpt-5.6-sol", VendorID: legacy.Id, Status: 1}
	require.NoError(t, db.Create(&item).Error)

	moved, err := MigrateOpenAICodexVendor()
	require.NoError(t, err)
	require.EqualValues(t, 1, moved)

	require.NoError(t, db.First(&item, item.Id).Error)
	require.Equal(t, canonical.Id, item.VendorID)
	require.NoError(t, db.First(&canonical, canonical.Id).Error)
	require.Equal(t, "OpenAI", canonical.Name)
	require.Equal(t, "OpenAI", canonical.Icon)
	require.Equal(t, 1, canonical.Status)
	var legacyCount int64
	require.NoError(t, db.Model(&Vendor{}).Where("name = ?", "OpenAI Codex").Count(&legacyCount).Error)
	require.Zero(t, legacyCount)

	moved, err = MigrateOpenAICodexVendor()
	require.NoError(t, err)
	require.Zero(t, moved)
}
