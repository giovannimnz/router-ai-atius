package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexPublishedPricingRatios(t *testing.T) {
	InitRatioSettings()

	tests := []struct {
		model           string
		wantModelRatio  float64
		wantOutputRatio float64
	}{
		{model: "gpt-5.5", wantModelRatio: 2.5, wantOutputRatio: 6},
		{model: "gpt-5.4", wantModelRatio: 2.5, wantOutputRatio: 4.5},
		{model: "gpt-5.4-mini", wantModelRatio: 0.375, wantOutputRatio: 6},
		{model: "gpt-5.3-codex-spark", wantModelRatio: 0.875, wantOutputRatio: 8},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			modelRatio, ok, matched := GetModelRatio(tt.model)
			require.True(t, ok)
			assert.Equal(t, tt.model, matched)
			assert.Equal(t, tt.wantModelRatio, modelRatio)
			assert.Equal(t, tt.wantOutputRatio, GetCompletionRatio(tt.model))
		})
	}
}

func TestCodexPricingFallsBackToCodeDefaultsWhenStoredRatiosAreStale(t *testing.T) {
	InitRatioSettings()
	t.Cleanup(InitRatioSettings)

	require.NoError(t, UpdateModelRatioByJSONString(`{"legacy-model":1}`))

	modelRatio, ok, matched := GetModelRatio("gpt-5.5")
	require.True(t, ok)
	assert.Equal(t, "gpt-5.5", matched)
	assert.Equal(t, 2.5, modelRatio)
	assert.Equal(t, 6.0, GetCompletionRatio("gpt-5.5"))
}
