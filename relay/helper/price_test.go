package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperDollarCostUsesUSDPerMillion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalInputs, originalOutputs := ratio_setting.GetInputOutputPriceMaps()
	originalInputJSON, err := common.Marshal(originalInputs)
	require.NoError(t, err)
	originalOutputJSON, err := common.Marshal(originalOutputs)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateInputOutputPricesByJSONStrings(string(originalInputJSON), string(originalOutputJSON)))
	})

	inputs := make(map[string]float64, len(originalInputs)+1)
	outputs := make(map[string]float64, len(originalOutputs)+1)
	for name, price := range originalInputs {
		inputs[name] = price
	}
	for name, price := range originalOutputs {
		outputs[name] = price
	}
	inputs["dollar-cost-test-model"] = 5
	outputs["dollar-cost-test-model"] = 30
	inputJSON, err := common.Marshal(inputs)
	require.NoError(t, err)
	outputJSON, err := common.Marshal(outputs)
	require.NoError(t, err)
	require.NoError(t, ratio_setting.UpdateInputOutputPricesByJSONStrings(string(inputJSON), string(outputJSON)))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ctx.Set("group", "default")
	info := &relaycommon.RelayInfo{
		OriginModelName: "dollar-cost-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	priceData, err := ModelPriceHelper(ctx, info, 1_000_000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.True(t, priceData.UseDollarCost)
	require.Equal(t, 5.0, priceData.InputPrice)
	require.Equal(t, 30.0, priceData.OutputPrice)
	require.Equal(t, int(5*common.QuotaPerUnit), priceData.QuotaToPreConsume)
}

func TestModelPriceHelperTieredUsesPreloadedRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/channel/test/1", nil)
	req.Body = nil
	req.ContentLength = 0
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "tiered-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"stream":true}`),
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.Equal(t, 1500, priceData.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "stream", info.TieredBillingSnapshot.EstimatedTier)
	require.Equal(t, billing_setting.BillingModeTieredExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, common.QuotaPerUnit, info.TieredBillingSnapshot.QuotaPerUnit)
}
