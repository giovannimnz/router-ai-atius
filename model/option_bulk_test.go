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

func TestUpdateOptionsBulkAppliesDollarCostPairAtomically(t *testing.T) {
	previousDB := DB
	previousInputs, previousOutputs := ratio_setting.GetInputOutputPriceMaps()
	previousInputJSON, err := common.Marshal(previousInputs)
	require.NoError(t, err)
	previousOutputJSON, err := common.Marshal(previousOutputs)
	require.NoError(t, err)
	common.OptionMapRWMutex.Lock()
	previousOptionMap := make(map[string]string, len(common.OptionMap))
	for key, value := range common.OptionMap {
		previousOptionMap[key] = value
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		DB = previousDB
		require.NoError(t, ratio_setting.UpdateInputOutputPricesByJSONStrings(string(previousInputJSON), string(previousOutputJSON)))
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	require.NoError(t, DB.AutoMigrate(&Option{}))

	inputJSON := `{"gpt-5.6-sol":5,"unrelated-model":0.25}`
	outputJSON := `{"gpt-5.6-sol":30,"unrelated-model":1.5}`
	require.NoError(t, UpdateOptionsBulk(map[string]string{
		"InputPrice":  inputJSON,
		"OutputPrice": outputJSON,
	}))

	input, output, useDollarCost := ratio_setting.GetInputOutputPrice("gpt-5.6-sol")
	require.True(t, useDollarCost)
	require.Equal(t, 5.0, input)
	require.Equal(t, 30.0, output)
	var inputOption Option
	require.NoError(t, DB.First(&inputOption, "key = ?", "InputPrice").Error)
	require.Equal(t, inputJSON, inputOption.Value)
	var outputOption Option
	require.NoError(t, DB.First(&outputOption, "key = ?", "OutputPrice").Error)
	require.Equal(t, outputJSON, outputOption.Value)

	err = UpdateOptionsBulk(map[string]string{
		"InputPrice":  `{"gpt-5.6-sol":6}`,
		"OutputPrice": `not-json`,
	})
	require.Error(t, err)
	require.NoError(t, DB.First(&inputOption, "key = ?", "InputPrice").Error)
	require.Equal(t, inputJSON, inputOption.Value)
	input, output, useDollarCost = ratio_setting.GetInputOutputPrice("gpt-5.6-sol")
	require.True(t, useDollarCost)
	require.Equal(t, 5.0, input)
	require.Equal(t, 30.0, output)

	err = UpdateOptionsBulk(map[string]string{"InputPrice": `{"gpt-5.6-sol":6}`})
	require.Error(t, err)
	require.NoError(t, DB.First(&inputOption, "key = ?", "InputPrice").Error)
	require.Equal(t, inputJSON, inputOption.Value)
}
