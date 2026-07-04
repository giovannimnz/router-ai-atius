package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetCodexChannelCredentialMetadata(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.Channel{
		Id:     5,
		Type:   constant.ChannelTypeCodex,
		Name:   "OpenAI - Codex",
		Status: common.ChannelStatusEnabled,
		Key:    `{"access_token":"token","refresh_token":"refresh","account_id":"acct_123","expired":"2026-07-04T22:00:00Z"}`,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "5"}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/5/codex/credential", nil)

	GetCodexChannelCredentialMetadata(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			OAuth         bool   `json:"oauth"`
			Authenticated bool   `json:"authenticated"`
			ExpiresAt     string `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.True(t, response.Data.OAuth)
	require.True(t, response.Data.Authenticated)
	require.Equal(t, "2026-07-04T22:00:00Z", response.Data.ExpiresAt)
}
