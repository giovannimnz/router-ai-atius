package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPatchDollarCostPricesPreservesUnrelatedModelsAndIsIdempotent(t *testing.T) {
	originalDB := DB
	originalDatabaseType := common.MainDatabaseType()
	originalInputs, originalOutputs := ratio_setting.GetInputOutputPriceMaps()
	originalCacheRatios := ratio_setting.GetCacheRatioCopy()
	originalCreateCacheRatios := ratio_setting.GetCreateCacheRatioCopy()
	originalInputJSON, err := common.Marshal(originalInputs)
	require.NoError(t, err)
	originalOutputJSON, err := common.Marshal(originalOutputs)
	require.NoError(t, err)
	originalCacheJSON, err := common.Marshal(originalCacheRatios)
	require.NoError(t, err)
	originalCreateCacheJSON, err := common.Marshal(originalCreateCacheRatios)
	require.NoError(t, err)
	t.Cleanup(func() {
		DB = originalDB
		common.SetMainDatabaseType(originalDatabaseType)
		require.NoError(t, ratio_setting.UpdateInputOutputPricesByJSONStrings(string(originalInputJSON), string(originalOutputJSON)))
		require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(string(originalCacheJSON)))
		require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(string(originalCreateCacheJSON)))
	})

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&Option{}))
	require.NoError(t, db.Create(&[]Option{
		{Key: inputPriceOptionKey, Value: `{"unrelated":0.25,"gpt-5.6-sol":99}`},
		{Key: outputPriceOptionKey, Value: `{"unrelated":1.5,"gpt-5.6-sol":999}`},
		{Key: cacheRatioOptionKey, Value: `{"unrelated":0.2,"gpt-5.6-sol":0.5}`},
		{Key: createCacheRatioOptionKey, Value: `{"unrelated":2,"gpt-5.6-sol":3}`},
	}).Error)

	cacheReadRatio := 0.1
	cacheWriteRatio := 1.25
	patches := map[string]*DollarCostPrice{
		"gpt-5.6-sol": {
			Input:           5,
			Output:          30,
			CacheReadRatio:  &cacheReadRatio,
			CacheWriteRatio: &cacheWriteRatio,
			SyncCacheRead:   true,
			SyncCacheWrite:  true,
		},
		"free-model": {Input: 0, Output: 0},
	}
	changed, err := PatchDollarCostPrices(patches, nil)
	require.NoError(t, err)
	require.Equal(t, 2, changed)

	var inputOption Option
	require.NoError(t, db.First(&inputOption, "key = ?", inputPriceOptionKey).Error)
	var outputOption Option
	require.NoError(t, db.First(&outputOption, "key = ?", outputPriceOptionKey).Error)
	inputs, err := decodeDollarCostPriceMap(inputOption.Value)
	require.NoError(t, err)
	outputs, err := decodeDollarCostPriceMap(outputOption.Value)
	require.NoError(t, err)
	require.Equal(t, 0.25, inputs["unrelated"])
	require.Equal(t, 1.5, outputs["unrelated"])
	require.Equal(t, 5.0, inputs["gpt-5.6-sol"])
	require.Equal(t, 30.0, outputs["gpt-5.6-sol"])
	require.Zero(t, inputs["free-model"])
	require.Zero(t, outputs["free-model"])
	var cacheOption Option
	require.NoError(t, db.First(&cacheOption, "key = ?", cacheRatioOptionKey).Error)
	cacheRatios, err := decodeDollarCostPriceMap(cacheOption.Value)
	require.NoError(t, err)
	require.Equal(t, 0.2, cacheRatios["unrelated"])
	require.Equal(t, 0.1, cacheRatios["gpt-5.6-sol"])
	var createCacheOption Option
	require.NoError(t, db.First(&createCacheOption, "key = ?", createCacheRatioOptionKey).Error)
	createCacheRatios, err := decodeDollarCostPriceMap(createCacheOption.Value)
	require.NoError(t, err)
	require.Equal(t, 2.0, createCacheRatios["unrelated"])
	require.Equal(t, 1.25, createCacheRatios["gpt-5.6-sol"])

	changed, err = PatchDollarCostPrices(patches, map[string]string{
		cacheRatioOptionKey:       `{"unrelated":0.3,"gpt-5.6-sol":9}`,
		createCacheRatioOptionKey: `{"unrelated":2.5,"gpt-5.6-sol":9}`,
	})
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	require.NoError(t, db.First(&cacheOption, "key = ?", cacheRatioOptionKey).Error)
	cacheRatios, err = decodeDollarCostPriceMap(cacheOption.Value)
	require.NoError(t, err)
	require.Equal(t, 0.3, cacheRatios["unrelated"])
	require.Equal(t, 0.1, cacheRatios["gpt-5.6-sol"])
	require.NoError(t, db.First(&createCacheOption, "key = ?", createCacheRatioOptionKey).Error)
	createCacheRatios, err = decodeDollarCostPriceMap(createCacheOption.Value)
	require.NoError(t, err)
	require.Equal(t, 2.5, createCacheRatios["unrelated"])
	require.Equal(t, 1.25, createCacheRatios["gpt-5.6-sol"])

	changed, err = PatchDollarCostPrices(patches, nil)
	require.NoError(t, err)
	require.Zero(t, changed)

	changed, err = PatchDollarCostPrices(map[string]*DollarCostPrice{"gpt-5.6-sol": nil}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	_, _, configured := ratio_setting.GetInputOutputPrice("gpt-5.6-sol")
	require.False(t, configured)
	cacheRatio, configured := ratio_setting.GetCacheRatio("gpt-5.6-sol")
	require.True(t, configured)
	require.Equal(t, 0.1, cacheRatio)
	createCacheRatio, configured := ratio_setting.GetCreateCacheRatio("gpt-5.6-sol")
	require.True(t, configured)
	require.Equal(t, 1.25, createCacheRatio)
}
