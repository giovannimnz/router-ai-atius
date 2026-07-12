package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchCodexChannelWhamDataClassifiesAuthWithoutLeakingBody(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	setting := `{"codex_credential_health":{"last_probe_at":"2026-07-10T10:00:00Z","last_probe_status":"ok"}}`
	require.NoError(t, db.Create(&model.Channel{
		Id:      5,
		Type:    constant.ChannelTypeCodex,
		Name:    "OpenAI - Codex",
		Key:     `{"access_token":"access-secret","account_id":"acct-test"}`,
		Setting: &setting,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/5/codex/usage", nil)
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}

	fetchCodexChannelWhamData(ctx, func(context.Context, *http.Client, string, string, string) (int, []byte, error) {
		return http.StatusUnauthorized, []byte(`{"error":"token_invalidated","secret":"must-not-leak"}`), nil
	}, "test usage failure", "usage failed")

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "must-not-leak")
	var response struct {
		Success bool            `json:"success"`
		Code    types.ErrorCode `json:"code"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Equal(t, types.ErrorCodeCodexUpstreamTokenInvalidated, response.Code)

	var persisted model.Channel
	require.NoError(t, db.First(&persisted, 5).Error)
	health := persisted.GetSetting().CodexCredentialHealth
	require.NotNil(t, health)
	assert.Equal(t, "ok", health.LastProbeStatus)
	assert.NotEmpty(t, health.LastUpstreamAuthAt)
}
