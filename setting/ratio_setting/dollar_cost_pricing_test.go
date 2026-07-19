package ratio_setting

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func encodeDollarCostPricingTestMap(t *testing.T, values map[string]float64) string {
	t.Helper()
	raw, err := common.Marshal(values)
	require.NoError(t, err)
	return string(raw)
}

func TestDollarCostPricingPublicationDoesNotExposeMixedSnapshots(t *testing.T) {
	originalInputs, originalOutputs := GetInputOutputPriceMaps()
	originalCache := GetCacheRatioCopy()
	originalCreateCache := GetCreateCacheRatioCopy()
	t.Cleanup(func() {
		inputJSON := encodeDollarCostPricingTestMap(t, originalInputs)
		outputJSON := encodeDollarCostPricingTestMap(t, originalOutputs)
		cacheJSON := encodeDollarCostPricingTestMap(t, originalCache)
		createCacheJSON := encodeDollarCostPricingTestMap(t, originalCreateCache)
		require.NoError(t, UpdateDollarCostPricingByJSONStrings(
			&inputJSON, &outputJSON, &cacheJSON, &createCacheJSON,
		))
	})

	modelName := "atomic-dollar-cost-test"
	states := [][4]float64{{1, 2, 0.1, 1.1}, {3, 4, 0.2, 1.2}}
	encoded := make([][4]string, len(states))
	for i, state := range states {
		for j, value := range state {
			encoded[i][j] = encodeDollarCostPricingTestMap(t, map[string]float64{modelName: value})
		}
	}
	require.NoError(t, UpdateDollarCostPricingByJSONStrings(
		&encoded[0][0], &encoded[0][1], &encoded[0][2], &encoded[0][3],
	))

	done := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		defer close(done)
		for i := 0; i < 2000; i++ {
			state := &encoded[i%len(encoded)]
			if err := UpdateDollarCostPricingByJSONStrings(&state[0], &state[1], &state[2], &state[3]); err != nil {
				errCh <- err
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			select {
			case err := <-errCh:
				require.NoError(t, err)
			default:
			}
			return
		default:
			pricing := GetDollarCostPricing(modelName)
			got := [4]float64{pricing.InputPrice, pricing.OutputPrice, pricing.CacheRatio, pricing.CreateCacheRatio}
			if got != states[0] && got != states[1] {
				require.FailNow(t, "mixed pricing snapshot", fmt.Sprintf("observed %v", got))
			}
		}
	}
}
