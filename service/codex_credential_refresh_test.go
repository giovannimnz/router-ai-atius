package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
