package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRefreshCodexOAuthTokenPreservesUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"refresh_token_invalidated","error_description":"refresh token revoked"}`))
	}))
	defer server.Close()

	result, err := refreshCodexOAuthToken(context.Background(), server.Client(), server.URL, "client-id", "refresh-token")

	require.Nil(t, result)
	require.Error(t, err)
	var upstreamErr *CodexUpstreamAuthError
	require.True(t, errors.As(err, &upstreamErr))
	assert.Equal(t, http.StatusBadRequest, upstreamErr.Status)
	assert.Equal(t, "refresh_token_invalidated", upstreamErr.UpstreamError)
	assert.Equal(t, "refresh token revoked", upstreamErr.ErrorDescription)

	issue := ClassifyCodexCredentialIssue(err, 0)
	require.True(t, issue.IsAuth)
	assert.Equal(t, types.ErrorCodeCodexUpstreamRefreshInvalidated, issue.Code)
	assert.Equal(t, "refresh_token_invalidated", issue.Reason)
	assert.True(t, issue.RequiresRegeneration)
}

func TestCodexUpstreamAuthErrorDoesNotExposeUpstreamBody(t *testing.T) {
	err := newCodexUpstreamAuthError(
		"codex oauth refresh",
		http.StatusUnauthorized,
		[]byte(`{"error":"token_invalidated","error_description":"Bearer upstream-secret-value"}`),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error=token_invalidated")
	assert.NotContains(t, err.Error(), "upstream-secret-value")
}

func TestBuildCodexCredentialMetadataSanitizesTokensAndRequiresRegeneration(t *testing.T) {
	setting := dto.ChannelSettings{
		CodexCredentialHealth: &dto.CodexCredentialHealth{
			LastProbeAt:          "2026-07-10T10:00:00Z",
			LastProbeStatus:      "auth_failed",
			LastUpstreamStatus:   http.StatusUnauthorized,
			LastUpstreamAuthCode: "token_invalidated",
			RequiresRegeneration: true,
			RegenerationReason:   "token_invalidated",
		},
	}
	settingBytes, err := common.Marshal(setting)
	require.NoError(t, err)
	settingJSON := string(settingBytes)

	channel := &model.Channel{
		Id:      5,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     `{"access_token":"access-secret","id_token":"id-secret","account_id":"acct-test","email":"user@example.com","expired":"2026-07-17T11:04:04Z"}`,
		Setting: &settingJSON,
	}

	meta, err := BuildCodexCredentialMetadata(channel)
	require.NoError(t, err)

	assert.False(t, meta.Authenticated)
	assert.False(t, meta.HasRefreshToken)
	assert.True(t, meta.RequiresRegeneration)
	assert.Equal(t, "refresh_token_missing", meta.RegenerationReason)
	assert.Equal(t, "acct-test", meta.AccountID)
	assert.Equal(t, "user@example.com", meta.Email)
	assert.Equal(t, "token_invalidated", meta.LastUpstreamAuthError)

	encoded, err := common.Marshal(meta)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "access-secret")
	assert.NotContains(t, string(encoded), "id-secret")
	assert.NotContains(t, string(encoded), "refresh-secret")
}

func TestBuildCodexCredentialMetadataDoesNotTrustExpiredAccessToken(t *testing.T) {
	channel := &model.Channel{
		Id:   5,
		Type: constant.ChannelTypeCodex,
		Name: "OpenAI - Codex",
		Key:  `{"access_token":"access-secret","refresh_token":"refresh-secret","account_id":"acct-test","expired":"2020-01-01T00:00:00Z"}`,
	}

	meta, err := BuildCodexCredentialMetadata(channel)
	require.NoError(t, err)

	assert.False(t, meta.Authenticated)
	assert.True(t, meta.HasRefreshToken)
	assert.False(t, meta.RequiresRegeneration)
}

func TestBuildCodexCredentialMetadataKeepsProbeHistoryAfterAuthIssueClears(t *testing.T) {
	keyBytes, err := common.Marshal(CodexOAuthKey{
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		AccountID:    "acct-test",
		Expired:      time.Now().Add(time.Hour).Format(time.RFC3339),
	})
	require.NoError(t, err)
	setting := `{"codex_credential_health":{"last_probe_at":"2026-07-10T10:00:00Z","last_probe_status":"auth_failed"}}`
	channel := &model.Channel{
		Id:      5,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     string(keyBytes),
		Setting: &setting,
	}

	meta, err := BuildCodexCredentialMetadata(channel)
	require.NoError(t, err)
	assert.True(t, meta.Authenticated)
	assert.Equal(t, codexCredentialProbeStatusAuthFailed, meta.LastProbeStatus)
}

func TestRecordCodexCredentialIssueByChannelIDPersistsRelayAuthFailure(t *testing.T) {
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	channel := &model.Channel{
		Id:      5,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     `{"access_token":"access-secret","account_id":"acct-test"}`,
		Setting: common.GetPointer(`{"proxy":"http://proxy.example","system_prompt":"keep-me","codex_credential_health":{"last_probe_at":"2026-07-10T10:00:00Z","last_probe_status":"ok"}}`),
	}
	require.NoError(t, db.Create(channel).Error)

	issue := CodexCredentialIssue{
		IsAuth:               true,
		Code:                 types.ErrorCodeCodexUpstreamTokenInvalidated,
		Reason:               codexCredentialReasonTokenInvalid,
		Message:              "codex access_token was invalidated upstream; refresh or regenerate the credential",
		UpstreamError:        codexCredentialReasonTokenInvalid,
		UpstreamStatus:       http.StatusUnauthorized,
		RequiresRegeneration: true,
	}
	require.NoError(t, RecordCodexCredentialIssueByChannelID(5, issue))

	var persisted model.Channel
	require.NoError(t, db.First(&persisted, 5).Error)
	health := persisted.GetSetting().CodexCredentialHealth
	require.NotNil(t, health)
	assert.Equal(t, "http://proxy.example", persisted.GetSetting().Proxy)
	assert.Equal(t, "keep-me", persisted.GetSetting().SystemPrompt)
	assert.Equal(t, codexCredentialProbeStatusOK, health.LastProbeStatus)
	assert.Equal(t, "2026-07-10T10:00:00Z", health.LastProbeAt)
	assert.NotEmpty(t, health.LastUpstreamAuthAt)
	assert.Equal(t, http.StatusUnauthorized, health.LastUpstreamStatus)
	assert.Equal(t, codexCredentialReasonTokenInvalid, health.LastUpstreamAuthCode)
	assert.True(t, health.RequiresRegeneration)
}

func TestNormalizeCodexUpstreamAuthErrorUsesDistinctCode(t *testing.T) {
	upstream := types.WithOpenAIError(types.OpenAIError{
		Message: "The upstream token was invalidated",
		Type:    "invalid_request_error",
		Code:    "token_invalidated",
	}, http.StatusUnauthorized)

	normalized := NormalizeCodexUpstreamAuthError(upstream)

	require.NotNil(t, normalized)
	assert.Equal(t, http.StatusUnauthorized, normalized.StatusCode)
	assert.Equal(t, types.ErrorCodeCodexUpstreamTokenInvalidated, normalized.GetErrorCode())
	assert.Equal(t, "codex_upstream_auth_error", normalized.ToOpenAIError().Type)
	assert.Contains(t, normalized.Error(), "refresh or regenerate")
}

func TestUpdateCodexCredentialHealthMergesIntoLatestChannelSetting(t *testing.T) {
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	stale := &model.Channel{
		Id:      78,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     `{"access_token":"access","refresh_token":"refresh"}`,
		Setting: common.GetPointer(`{"proxy":"http://stale.example","system_prompt":"stale","codex_credential_health":{"last_probe_status":"ok"}}`),
	}
	require.NoError(t, db.Create(stale).Error)
	require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", stale.Id).Update(
		"setting",
		`{"proxy":"http://current.example","system_prompt":"keep-current","force_format":true,"codex_credential_health":{"last_probe_status":"ok"}}`,
	).Error)

	health := dto.CodexCredentialHealth{
		LastUpstreamStatus:   http.StatusUnauthorized,
		LastUpstreamAuthCode: codexCredentialReasonTokenInvalid,
		RequiresRegeneration: true,
		RegenerationReason:   codexCredentialReasonTokenInvalid,
	}
	require.NoError(t, UpdateCodexCredentialHealth(stale, health))

	var persisted model.Channel
	require.NoError(t, db.First(&persisted, stale.Id).Error)
	setting := persisted.GetSetting()
	assert.Equal(t, "http://current.example", setting.Proxy)
	assert.Equal(t, "keep-current", setting.SystemPrompt)
	assert.True(t, setting.ForceFormat)
	require.NotNil(t, setting.CodexCredentialHealth)
	assert.Equal(t, health, *setting.CodexCredentialHealth)
}

func TestRefreshCodexChannelCredentialCoalescesStaleAndFreshGenerationsOnSQLite(t *testing.T) {
	originalDB := model.DB
	originalSecret := common.CryptoSecret
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.CodexOAuthOperation{}))
	model.DB = db
	common.CryptoSecret = "codex-refresh-coalescing-test-secret"
	t.Cleanup(func() {
		model.DB = originalDB
		common.CryptoSecret = originalSecret
	})

	require.NoError(t, db.Create(&model.Channel{
		Id:      77,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     `{"access_token":"old-access","refresh_token":"old-refresh","account_id":"acct-test","type":"codex"}`,
		Setting: common.GetPointer(`{"codex_credential_health":{"last_upstream_status":401,"requires_regeneration":true}}`),
	}).Error)

	fixedNow := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh"}`)
	require.NoError(t, err)
	staleGeneration := codexCredentialGeneration(oldKey)
	calls := 0
	refresh := func(_ context.Context, refreshToken string, _ string) (*CodexOAuthTokenResult, error) {
		calls++
		require.Equal(t, "old-refresh", refreshToken)
		return &CodexOAuthTokenResult{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresAt:    fixedNow.Add(time.Hour),
		}, nil
	}

	first, err := refreshCodexChannelCredentialAfterObservation(
		context.Background(), 77, refresh, staleGeneration, fixedNow,
	)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "new-refresh", first.key.RefreshToken)
	assert.Equal(t, 1, calls)

	staleReplica, err := refreshCodexChannelCredentialAfterObservation(
		context.Background(), 77, refresh, staleGeneration, fixedNow.Add(time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, staleReplica)
	assert.Equal(t, "new-refresh", staleReplica.key.RefreshToken)
	assert.Equal(t, 1, calls, "stale replica must observe the generation written by the lock holder")

	freshGeneration := codexCredentialGeneration(staleReplica.key)
	freshReplica, err := refreshCodexChannelCredentialAfterObservation(
		context.Background(), 77, refresh, freshGeneration, fixedNow.Add(2*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, freshReplica)
	assert.Equal(t, "new-refresh", freshReplica.key.RefreshToken)
	assert.Equal(t, 1, calls, "a replica arriving just after commit must coalesce on freshness")

	var persisted model.Channel
	require.NoError(t, db.First(&persisted, 77).Error)
	persistedKey, err := parseCodexOAuthKey(persisted.Key)
	require.NoError(t, err)
	assert.Equal(t, "new-refresh", persistedKey.RefreshToken)
	assert.Equal(t, fixedNow.Format(time.RFC3339), persistedKey.LastRefresh)
	health := persisted.GetSetting().CodexCredentialHealth
	require.NotNil(t, health)
	assert.False(t, health.RequiresRegeneration)
	assert.Zero(t, health.LastUpstreamStatus)
}

func TestCodexCredentialRefreshRecoversDurableRotatedResultAfterProcessLoss(t *testing.T) {
	db := newCodexRefreshSQLTestDB(t)
	fixedNow := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh","account_id":"acct-test","type":"codex"}`)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Channel{
		Id: 88, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex",
		Key: `{"access_token":"old-access","refresh_token":"old-refresh","account_id":"acct-test","type":"codex"}`,
	}).Error)

	claim, current, err := claimCodexCredentialRefresh(
		context.Background(), 88, codexCredentialGeneration(oldKey), fixedNow,
	)
	require.NoError(t, err)
	require.Nil(t, current)
	require.NoError(t, markCodexCredentialRefreshStarted(context.Background(), claim, fixedNow))
	rotated := mergeCodexCredentialRefreshResult(oldKey, &CodexOAuthTokenResult{
		AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresAt: fixedNow.Add(time.Hour),
	}, fixedNow)
	payload, err := sealCodexOAuthPayload(rotated)
	require.NoError(t, err)
	require.NoError(t, persistCodexCredentialRefreshResult(context.Background(), claim, payload, fixedNow))

	var persistedOperation model.CodexOAuthOperation
	require.NoError(t, db.Where("operation_key = ?", claim.record.OperationKey).First(&persistedOperation).Error)
	assert.Equal(t, CodexDeviceAuthorizationStageExchanged, persistedOperation.Stage)
	assert.NotContains(t, persistedOperation.ProtectedPayload, "new-refresh")

	upstreamCalls := 0
	recovered, err := refreshCodexChannelCredentialAfterObservation(
		context.Background(),
		88,
		func(context.Context, string, string) (*CodexOAuthTokenResult, error) {
			upstreamCalls++
			return nil, errors.New("must not call upstream")
		},
		codexCredentialGeneration(oldKey),
		fixedNow.Add(codexCredentialRefreshLeaseTTL+time.Second),
	)
	require.NoError(t, err)
	assert.Equal(t, 0, upstreamCalls)
	assert.Equal(t, "new-refresh", recovered.key.RefreshToken)

	var channel model.Channel
	require.NoError(t, db.First(&channel, 88).Error)
	saved, err := parseCodexOAuthKey(channel.Key)
	require.NoError(t, err)
	assert.Equal(t, "new-refresh", saved.RefreshToken)
}

func TestCodexCredentialRefreshRejectsStaleFenceInChannelWriteTransaction(t *testing.T) {
	db := newCodexRefreshSQLTestDB(t)
	fixedNow := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Channel{
		Id: 89, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex",
		Key: `{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`,
	}).Error)
	first, _, err := claimCodexCredentialRefresh(context.Background(), 89, codexCredentialGeneration(oldKey), fixedNow)
	require.NoError(t, err)
	require.NoError(t, markCodexCredentialRefreshStarted(context.Background(), first, fixedNow))
	rotated := mergeCodexCredentialRefreshResult(oldKey, &CodexOAuthTokenResult{
		AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresAt: fixedNow.Add(time.Hour),
	}, fixedNow)
	payload, err := sealCodexOAuthPayload(rotated)
	require.NoError(t, err)
	require.NoError(t, persistCodexCredentialRefreshResult(context.Background(), first, payload, fixedNow))

	takeoverNow := fixedNow.Add(codexCredentialRefreshLeaseTTL + time.Second)
	second, _, err := claimCodexCredentialRefresh(context.Background(), 89, codexCredentialGeneration(oldKey), takeoverNow)
	require.NoError(t, err)
	require.Greater(t, second.record.Fence, first.record.Fence)

	_, err = commitCodexCredentialRefresh(context.Background(), first, rotated, takeoverNow)
	assert.ErrorIs(t, err, ErrCodexDeviceAuthorizationLeaseLost)
	var unchanged model.Channel
	require.NoError(t, db.First(&unchanged, 89).Error)
	assert.Contains(t, unchanged.Key, "old-refresh")

	recovered := &CodexOAuthKey{}
	require.NoError(t, openCodexOAuthPayload(second.record.ProtectedPayload, recovered))
	_, err = commitCodexCredentialRefresh(context.Background(), second, recovered, takeoverNow)
	require.NoError(t, err)
	var saved model.Channel
	require.NoError(t, db.First(&saved, 89).Error)
	assert.Contains(t, saved.Key, "new-refresh")
}

func TestCodexCredentialRefreshRejectsLeaseCrossedWhileAcquiringCommitLocks(t *testing.T) {
	db := newCodexRefreshSQLTestDB(t)
	start := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	nowMillis := atomic.Int64{}
	nowMillis.Store(start.UnixMilli())
	codexCredentialRefreshCommitNow = func(time.Time) time.Time {
		return time.UnixMilli(nowMillis.Load())
	}
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Channel{
		Id: 92, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex",
		Key: `{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`,
	}).Error)
	claim, current, err := claimCodexCredentialRefresh(
		context.Background(), 92, codexCredentialGeneration(oldKey), start,
	)
	require.NoError(t, err)
	require.Nil(t, current)
	require.NoError(t, markCodexCredentialRefreshStarted(context.Background(), claim, start))
	rotated := mergeCodexCredentialRefreshResult(oldKey, &CodexOAuthTokenResult{
		AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresAt: start.Add(time.Hour),
	}, start)
	payload, err := sealCodexOAuthPayload(rotated)
	require.NoError(t, err)
	require.NoError(t, persistCodexCredentialRefreshResult(context.Background(), claim, payload, start))
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(
		"test:advance-refresh-clock-after-channel-lock",
		func(tx *gorm.DB) {
			if tx.Statement.Table == "channels" {
				nowMillis.Store(start.Add(codexCredentialRefreshLeaseTTL + time.Millisecond).UnixMilli())
			}
		},
	))

	_, err = commitCodexCredentialRefresh(context.Background(), claim, rotated, start)
	assert.ErrorIs(t, err, ErrCodexDeviceAuthorizationLeaseLost)
	var channel model.Channel
	require.NoError(t, db.First(&channel, 92).Error)
	assert.Contains(t, channel.Key, "old-refresh")
}

func TestCodexCredentialRefreshDoesNotRetryAfterAmbiguousTransportFailure(t *testing.T) {
	db := newCodexRefreshSQLTestDB(t)
	fixedNow := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Channel{
		Id: 90, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex",
		Key: `{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`,
	}).Error)
	upstreamCalls := 0
	refresh := func(context.Context, string, string) (*CodexOAuthTokenResult, error) {
		upstreamCalls++
		return nil, context.DeadlineExceeded
	}

	_, err = refreshCodexChannelCredentialAfterObservation(
		context.Background(), 90, refresh, codexCredentialGeneration(oldKey), fixedNow,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outcome is unknown")

	var operation model.CodexOAuthOperation
	require.NoError(t, db.Where("operation_key = ?", codexCredentialRefreshOperationKey(90)).First(&operation).Error)
	assert.Equal(t, codexCredentialRefreshStageStarted, operation.Stage)
	assert.Equal(t, CodexDeviceAuthorizationUncertain, operation.Status)

	_, err = refreshCodexChannelCredentialAfterObservation(
		context.Background(),
		90,
		refresh,
		codexCredentialGeneration(oldKey),
		fixedNow.Add(codexCredentialRefreshLeaseTTL+time.Second),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outcome is unknown")
	assert.Equal(t, 1, upstreamCalls)

	var channel model.Channel
	require.NoError(t, db.First(&channel, 90).Error)
	assert.Contains(t, channel.Key, "old-refresh")
}

func TestCodexCredentialRefreshFailureImmediatelyAfterUpstreamReturnIsTerminal(t *testing.T) {
	db := newCodexRefreshSQLTestDB(t)
	fixedNow := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Channel{
		Id: 91, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex",
		Key: `{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`,
	}).Error)

	upstreamCalls := 0
	refresh := func(context.Context, string, string) (*CodexOAuthTokenResult, error) {
		upstreamCalls++
		return &CodexOAuthTokenResult{
			AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresAt: fixedNow.Add(time.Hour),
		}, nil
	}
	_, err = refreshCodexChannelCredentialAfterObservation(
		context.Background(), 91, refresh, codexCredentialGeneration(oldKey), fixedNow,
		func() error { return errors.New("simulated process loss after upstream return") },
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outcome is unknown")

	var operation model.CodexOAuthOperation
	require.NoError(t, db.Where("operation_key = ?", codexCredentialRefreshOperationKey(91)).First(&operation).Error)
	assert.Equal(t, codexCredentialRefreshStageStarted, operation.Stage)
	assert.Equal(t, CodexDeviceAuthorizationUncertain, operation.Status)
	assert.NotContains(t, operation.ProtectedPayload, "new-refresh")

	_, err = refreshCodexChannelCredentialAfterObservation(
		context.Background(), 91, refresh, codexCredentialGeneration(oldKey), fixedNow.Add(time.Second),
	)
	require.Error(t, err)
	assert.Equal(t, 1, upstreamCalls, "the old refresh token must never be reused after the ambiguous boundary")

	var channel model.Channel
	require.NoError(t, db.First(&channel, 91).Error)
	assert.Contains(t, channel.Key, "old-refresh")
}

func TestCodexCredentialRefreshFailureDoesNotInvalidateRegeneratedCredential(t *testing.T) {
	db := newCodexRefreshSQLTestDB(t)
	fixedNow := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	oldKey, err := parseCodexOAuthKey(`{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Channel{
		Id: 93, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex",
		Key: `{"access_token":"old-access","refresh_token":"old-refresh","type":"codex"}`,
	}).Error)

	regeneratedKey := `{"access_token":"device-access","refresh_token":"device-refresh","account_id":"device-account","type":"codex"}`
	regeneratedSetting := `{"codex_credential_health":{"last_probe_at":"2026-07-13T12:00:00Z","last_probe_status":"ok"}}`
	_, err = refreshCodexChannelCredentialAfterObservation(
		context.Background(),
		93,
		func(context.Context, string, string) (*CodexOAuthTokenResult, error) {
			return &CodexOAuthTokenResult{
				AccessToken: "refresh-access", RefreshToken: "refresh-rotated", ExpiresAt: fixedNow.Add(time.Hour),
			}, nil
		},
		codexCredentialGeneration(oldKey),
		fixedNow,
		func() error {
			require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", 93).Updates(map[string]any{
				"key": regeneratedKey, "setting": regeneratedSetting,
			}).Error)
			return errors.New("simulated old refresh persistence failure")
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outcome is unknown")

	var operation model.CodexOAuthOperation
	require.NoError(t, db.Where("operation_key = ?", codexCredentialRefreshOperationKey(93)).First(&operation).Error)
	assert.Equal(t, codexCredentialRefreshStageStarted, operation.Stage)
	assert.Equal(t, CodexDeviceAuthorizationUncertain, operation.Status)

	var channel model.Channel
	require.NoError(t, db.First(&channel, 93).Error)
	assert.Equal(t, regeneratedKey, channel.Key)
	health := channel.GetSetting().CodexCredentialHealth
	require.NotNil(t, health)
	assert.Equal(t, codexCredentialProbeStatusOK, health.LastProbeStatus)
	assert.False(t, health.RequiresRegeneration)
	assert.Zero(t, health.LastUpstreamStatus)
	assert.Empty(t, health.LastUpstreamAuthCode)
}

func newCodexRefreshSQLTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:codex-refresh-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.CodexOAuthOperation{}))
	originalDB := model.DB
	originalSecret := common.CryptoSecret
	originalCommitNow := codexCredentialRefreshCommitNow
	model.DB = db
	common.CryptoSecret = "codex-refresh-test-secret"
	codexCredentialRefreshCommitNow = func(fallback time.Time) time.Time { return fallback }
	t.Cleanup(func() {
		model.DB = originalDB
		common.CryptoSecret = originalSecret
		codexCredentialRefreshCommitNow = originalCommitNow
	})
	return db
}
