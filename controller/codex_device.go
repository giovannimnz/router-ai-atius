package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/codex"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// StartDeviceAuth starts a device auth flow via Codex CLI.
// POST /api/channel/codex/oauth/device/start
func StartDeviceAuth(c *gin.Context) {
	output, handle, err := service.StartDeviceAuth(c.Request.Context())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"session_id":       handle.ID,
			"verification_url": output.VerificationURL,
			"user_code":        output.UserCode,
			"expires_in":       output.ExpiresIn,
		},
	})
}

// PollDeviceAuth checks if a device auth session has completed.
// POST /api/channel/codex/oauth/device/poll
func PollDeviceAuth(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "missing session_id"})
		return
	}

	result, err := service.PollDeviceAuth(sessionID)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if result == nil {
		// Still pending
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{"status": "pending"},
		})
		return
	}

	// Auth complete — build credential key
	key := codex.OAuthKey{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		AccountID:    result.AccountID,
		LastRefresh:  result.LastRefresh,
		Expired:      result.ExpiresAt,
		Email:        result.Email,
		Type:         "codex",
	}

	encoded, err := common.Marshal(key)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":          "complete",
			"key":             string(encoded),
			"email":           result.Email,
			"account_id":      result.AccountID,
			"access_token":    result.AccessToken,
			"expires_at":      result.ExpiresAt,
			"last_refresh":    result.LastRefresh,
		},
	})
}

// UploadDeviceAuthJSON accepts a pasted or uploaded auth.json from Codex CLI.
// POST /api/channel/codex/oauth/device/upload
func UploadDeviceAuthJSON(c *gin.Context) {
	type uploadReq struct {
		JSON string `json:"json"` // raw JSON content of auth.json
	}

	var req uploadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if strings.TrimSpace(req.JSON) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "missing json field"})
		return
	}

	result, err := service.ParseAuthJSON(req.JSON)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	key := codex.OAuthKey{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		AccountID:    result.AccountID,
		LastRefresh:  time.Now().Format(time.RFC3339),
		Expired:      result.ExpiresAt,
		Email:        result.Email,
		Type:         "codex",
	}

	encoded, err := common.Marshal(key)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"key":          string(encoded),
			"email":        result.Email,
			"account_id":   result.AccountID,
			"expires_at":   result.ExpiresAt,
			"last_refresh": time.Now().Format(time.RFC3339),
		},
	})
}
