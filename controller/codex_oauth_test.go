package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCodexChannelCredentialReturnsSanitizedMetadata(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	settingJSON := `{"codex_credential_health":{"last_probe_status":"auth_failed","last_upstream_auth_error":"token_invalidated","requires_regeneration":true,"regeneration_reason":"token_invalidated"}}`
	require.NoError(t, db.Create(&model.Channel{
		Id:      5,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     `{"access_token":"access-secret","refresh_token":"refresh-secret","id_token":"id-secret","account_id":"acct-test","email":"user@example.com","expired":"2026-07-17T11:04:04Z"}`,
		Setting: &settingJSON,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/5/codex/credential", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	GetCodexChannelCredential(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "access-secret")
	assert.NotContains(t, recorder.Body.String(), "refresh-secret")
	assert.NotContains(t, recorder.Body.String(), "id-secret")

	var response struct {
		Success bool                            `json:"success"`
		Data    service.CodexCredentialMetadata `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.True(t, response.Data.Authenticated)
	assert.True(t, response.Data.HasRefreshToken)
	assert.Equal(t, "acct-test", response.Data.AccountID)
	assert.Equal(t, "token_invalidated", response.Data.LastUpstreamAuthError)
}
