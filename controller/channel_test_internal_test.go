package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettleTestQuotaUsesTieredBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:   "tiered_expr",
			ExprString:    `param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`,
			ExprHash:      billingexpr.ExprHashString(`param("stream") == true ? tier("stream", p * 3) : tier("base", p * 2)`),
			GroupRatio:    1,
			EstimatedTier: "stream",
			QuotaPerUnit:  common.QuotaPerUnit,
			ExprVersion:   1,
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"stream":true}`),
		},
	}

	quota, result := settleTestQuota(info, types.PriceData{
		ModelRatio:      1,
		CompletionRatio: 2,
	}, &dto.Usage{
		PromptTokens: 1000,
	})

	require.Equal(t, 1500, quota)
	require.NotNil(t, result)
	require.Equal(t, "stream", result.MatchedTier)
}

func TestBuildTestLogOtherInjectsTieredInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "tiered_expr",
			ExprString:  `tier("base", p * 2)`,
		},
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	priceData := types.PriceData{
		GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1},
	}
	usage := &dto.Usage{
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 12,
		},
	}

	other := buildTestLogOther(ctx, info, priceData, usage, &billingexpr.TieredResult{
		MatchedTier: "base",
	})

	require.Equal(t, "tiered_expr", other["billing_mode"])
	require.Equal(t, "base", other["matched_tier"])
	require.NotEmpty(t, other["expr_b64"])
}

func TestResolveChannelTestUserIDUsesRequestUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("id", 2)

	userID, err := resolveChannelTestUserID(ctx)

	require.NoError(t, err)
	require.Equal(t, 2, userID)
}

func TestShouldUseStreamForChannelTest(t *testing.T) {
	codexChannel := &model.Channel{Type: constant.ChannelTypeCodex}
	openAIChannel := &model.Channel{Type: constant.ChannelTypeOpenAI}

	tests := []struct {
		name         string
		channel      *model.Channel
		model        string
		endpointType string
		want         bool
	}{
		{
			name:    "codex default test uses responses stream",
			channel: codexChannel,
			model:   "gpt-5.5",
			want:    true,
		},
		{
			name:    "codex gpt-5.4 test uses responses stream",
			channel: codexChannel,
			model:   "gpt-5.4",
			want:    true,
		},
		{
			name:    "codex gpt-5.4-mini test uses responses stream",
			channel: codexChannel,
			model:   "gpt-5.4-mini",
			want:    true,
		},
		{
			name:    "codex gpt-5.3-codex-spark test uses responses stream",
			channel: codexChannel,
			model:   "gpt-5.3-codex-spark",
			want:    true,
		},
		{
			name:         "codex responses endpoint uses stream",
			channel:      codexChannel,
			model:        "gpt-5.5",
			endpointType: string(constant.EndpointTypeOpenAIResponse),
			want:         true,
		},
		{
			name:         "codex embeddings endpoint does not use stream",
			channel:      codexChannel,
			model:        "text-embedding-3-small",
			endpointType: string(constant.EndpointTypeEmbeddings),
			want:         false,
		},
		{
			name:         "codex compact endpoint does not use stream",
			channel:      codexChannel,
			model:        "gpt-5.5" + ratio_setting.CompactModelSuffix,
			endpointType: string(constant.EndpointTypeOpenAIResponseCompact),
			want:         false,
		},
		{
			name:    "non codex channel keeps non stream default",
			channel: openAIChannel,
			model:   "gpt-4o-mini",
			want:    false,
		},
		{
			name:  "nil channel keeps non stream default",
			model: "gpt-5.5",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUseStreamForChannelTest(tt.channel, tt.model, tt.endpointType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveChannelTestStream(t *testing.T) {
	codexChannel := &model.Channel{Type: constant.ChannelTypeCodex}
	openAIChannel := &model.Channel{Type: constant.ChannelTypeOpenAI}

	tests := []struct {
		name         string
		channel      *model.Channel
		model        string
		endpointType string
		requested    bool
		want         bool
	}{
		{
			name:      "codex responses test forces stream even when unchecked",
			channel:   codexChannel,
			model:     "gpt-5.4",
			requested: false,
			want:      true,
		},
		{
			name:         "codex explicit responses endpoint forces stream",
			channel:      codexChannel,
			model:        "gpt-5.6-terra",
			endpointType: string(constant.EndpointTypeOpenAIResponse),
			requested:    false,
			want:         true,
		},
		{
			name:         "codex compact endpoint keeps stream disabled",
			channel:      codexChannel,
			model:        "gpt-5.5" + ratio_setting.CompactModelSuffix,
			endpointType: string(constant.EndpointTypeOpenAIResponseCompact),
			requested:    false,
			want:         false,
		},
		{
			name:      "non codex preserves unchecked state",
			channel:   openAIChannel,
			model:     "gpt-4o-mini",
			requested: false,
			want:      false,
		},
		{
			name:      "explicit stream stays enabled",
			channel:   openAIChannel,
			model:     "gpt-4o-mini",
			requested: true,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveChannelTestStream(tt.channel, tt.model, tt.endpointType, tt.requested)
			assert.Equal(t, tt.want, got)
		})
	}
}
