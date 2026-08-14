package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreConsumeBilling_WalletOnlyAllowsNegativeWalletQuota(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 501
	const tokenID = 601
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "wallet-overdraft-token", 1000000)

	relayInfo := &relaycommon.RelayInfo{
		UserId:         userID,
		TokenId:        tokenID,
		TokenKey:       "wallet-overdraft-token",
		TokenUnlimited: false,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}
	ctx.Set("token_quota", 1000000)

	apiErr := PreConsumeBilling(ctx, 250, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	assert.Equal(t, BillingSourceWallet, relayInfo.BillingSource)
	assert.Equal(t, -150, getUserQuota(t, userID))
	assert.Equal(t, 999750, getTokenRemainQuota(t, tokenID))
}

func TestPreConsumeBilling_SubscriptionOverflowFallsBackToNegativeWallet(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 502
	const tokenID = 602
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "subscription-overdraft-token", 1000000)
	seedSubscription(t, 701, userID, 50, 50)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "subscription-overdraft-token",
		TokenUnlimited:  false,
		OriginModelName: "gpt-5.4",
		RequestId:       "req-wallet-overdraft-fallback",
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_first",
		},
	}
	ctx.Set("token_quota", 1000000)

	apiErr := PreConsumeBilling(ctx, 250, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	assert.Equal(t, BillingSourceWallet, relayInfo.BillingSource)
	assert.Equal(t, -150, getUserQuota(t, userID))
	assert.Equal(t, int64(50), getSubscriptionUsed(t, 701))
	assert.Equal(t, 999750, getTokenRemainQuota(t, tokenID))
}

func TestPreConsumeBilling_SubscriptionOverflowFallsBackDespitePlanFlag(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 503
	const tokenID = 603
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "subscription-strict-token", 1000000)
	seedSubscription(t, 702, userID, 50, 50)
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("id = ?", 702).Update("allow_wallet_overflow", false).Error)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "subscription-strict-token",
		TokenUnlimited:  false,
		OriginModelName: "gpt-5.4",
		RequestId:       "req-wallet-overdraft-strict",
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_first",
		},
	}
	ctx.Set("token_quota", 1000000)

	apiErr := PreConsumeBilling(ctx, 250, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	assert.Equal(t, BillingSourceWallet, relayInfo.BillingSource)
	assert.Equal(t, -150, getUserQuota(t, userID))
	assert.Equal(t, int64(50), getSubscriptionUsed(t, 702))
	assert.Equal(t, 999750, getTokenRemainQuota(t, tokenID))
}

func TestPreConsumeBilling_SubscriptionOnlyExhaustionFallsBackDespitePlanFlag(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 505
	const tokenID = 605
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "subscription-only-overdraft-token", 1000000)
	seedSubscription(t, 703, userID, 50, 50)
	require.NoError(t, model.DB.Model(&model.UserSubscription{}).Where("id = ?", 703).Update("allow_wallet_overflow", false).Error)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "subscription-only-overdraft-token",
		TokenUnlimited:  false,
		OriginModelName: "gpt-5.4",
		RequestId:       "req-subscription-only-overdraft",
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_only",
		},
	}
	ctx.Set("token_quota", 1000000)

	apiErr := PreConsumeBilling(ctx, 250, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	assert.Equal(t, BillingSourceWallet, relayInfo.BillingSource)
	assert.Equal(t, -150, getUserQuota(t, userID))
	assert.Equal(t, int64(50), getSubscriptionUsed(t, 703))
	assert.Equal(t, 999750, getTokenRemainQuota(t, tokenID))
}

func TestPreConsumeBilling_SubscriptionOnlyWithoutSubscriptionFallsBackToWallet(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 506
	const tokenID = 606
	seedUser(t, userID, -100)
	seedToken(t, tokenID, userID, "subscription-only-no-plan-token", 1000000)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "subscription-only-no-plan-token",
		TokenUnlimited:  false,
		OriginModelName: "gpt-5.4",
		RequestId:       "req-subscription-only-no-plan",
		UserSetting: dto.UserSetting{
			BillingPreference: "subscription_only",
		},
	}
	ctx.Set("token_quota", 1000000)

	apiErr := PreConsumeBilling(ctx, 250, relayInfo)
	require.Nil(t, apiErr)
	require.NotNil(t, relayInfo.Billing)
	assert.Equal(t, BillingSourceWallet, relayInfo.BillingSource)
	assert.Equal(t, -350, getUserQuota(t, userID))
	assert.Equal(t, 999750, getTokenRemainQuota(t, tokenID))
}

func TestPreConsumeBilling_WalletOnlyStillBlocksWhenTokenQuotaInsufficient(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 504
	const tokenID = 604
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, "token-strict-wallet", 120)

	relayInfo := &relaycommon.RelayInfo{
		UserId:         userID,
		TokenId:        tokenID,
		TokenKey:       "token-strict-wallet",
		TokenUnlimited: false,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}
	ctx.Set("token_quota", 120)

	apiErr := PreConsumeBilling(ctx, 250, relayInfo)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.Equal(t, types.ErrorCodePreConsumeTokenQuotaFailed, apiErr.GetErrorCode())
	assert.Equal(t, 100, getUserQuota(t, userID))
	assert.Equal(t, 120, getTokenRemainQuota(t, tokenID))
}

func TestPreWssConsumeQuota_NegativeWalletWithUnlimitedTokenDoesNotBlock(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	const userID = 507
	const tokenID = 607
	seedUser(t, userID, -100)
	seedToken(t, tokenID, userID, "wss-unlimited-token", 1000000)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Update("unlimited_quota", true).Error)

	relayInfo := &relaycommon.RelayInfo{
		UserId:          userID,
		TokenId:         tokenID,
		TokenKey:        "sk-wss-unlimited-token",
		TokenUnlimited:  true,
		OriginModelName: "gpt-4",
		UsingGroup:      "default",
		UserGroup:       "default",
	}
	usage := &dto.RealtimeUsage{
		InputTokens: 10,
		InputTokenDetails: dto.InputTokenDetails{
			TextTokens: 10,
		},
	}

	require.NoError(t, PreWssConsumeQuota(ctx, relayInfo, usage))
	assert.Less(t, getUserQuota(t, userID), -100)
}
