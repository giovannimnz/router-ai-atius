package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/codex"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func codexOAuthSessionKey(channelID int, field string) string {
	return fmt.Sprintf("codex_oauth_%s_%d", field, channelID)
}

func StartCodexOAuth(c *gin.Context) {
	failClosedLegacyCodexPKCE(c)
}

func StartCodexOAuthForChannel(c *gin.Context) {
	failClosedLegacyCodexPKCE(c)
}

func failClosedLegacyCodexPKCE(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"success": false,
		"message": "legacy Codex PKCE flow is disabled; use device authorization",
		"code":    "codex_pkce_disabled",
	})
}

func StartCodexDeviceOAuthForChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	channelProxy, ok := getCodexOAuthChannelProxy(c, channelID)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if err := service.EnsureCodexOAuthOperationStore(ctx); err != nil {
		common.SysError("codex OAuth operation store unavailable: " + common.MaskSensitiveInfo(err.Error()))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "shared OAuth operation store unavailable"})
		return
	}
	flow, err := service.StartCodexDeviceAuthorization(ctx, channelProxy)
	if err != nil {
		common.SysError("failed to start codex device authorization: " + common.MaskSensitiveInfo(err.Error()))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "failed to start device authorization"})
		return
	}

	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiError(c, errors.New("authenticated user id is required"))
		return
	}
	if err := service.RegisterCodexDeviceAuthorization(ctx, userID, channelID, flow); err != nil {
		common.SysError("failed to register codex device authorization: " + common.MaskSensitiveInfo(err.Error()))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "failed to persist device authorization"})
		return
	}

	session := sessions.Default(c)
	previousDeviceAuthID, _ := session.Get(codexOAuthSessionKey(channelID, "device_auth_id")).(string)
	if err := saveCodexDeviceOAuthSession(session, channelID, flow.DeviceAuthID); err != nil {
		_ = service.DeleteCodexDeviceAuthorization(ctx, userID, channelID, flow.DeviceAuthID)
		common.ApiError(c, err)
		return
	}
	if previousDeviceAuthID != "" && previousDeviceAuthID != flow.DeviceAuthID {
		_ = service.DeleteCodexDeviceAuthorization(ctx, userID, channelID, previousDeviceAuthID)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"flow":             "device_code",
			"verification_url": flow.VerificationURL,
			"user_code":        flow.UserCode,
			"interval_seconds": int(flow.Interval.Seconds()),
			"expires_at":       flow.ExpiresAt.Format(time.RFC3339),
		},
	})
}

func PollCodexDeviceOAuthForChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	channelProxy, ok := getCodexOAuthChannelProxy(c, channelID)
	if !ok {
		return
	}
	session := sessions.Default(c)
	deviceAuthID, _ := session.Get(codexOAuthSessionKey(channelID, "device_auth_id")).(string)
	if strings.TrimSpace(deviceAuthID) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "device authorization not started or session expired"})
		return
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiError(c, errors.New("authenticated user id is required"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	state, err := service.ContinueCodexDeviceAuthorization(ctx, userID, channelID, deviceAuthID, channelProxy, func(token *service.CodexOAuthTokenResult) (*service.CodexDeviceAuthorizationResult, string, error) {
		return prepareCodexOAuthTokenResult(channelID, token)
	})
	if err != nil {
		if errors.Is(err, service.ErrCodexDeviceAuthorizationNotFound) || errors.Is(err, service.ErrCodexDeviceAuthorizationExpired) {
			if saveErr := clearCodexDeviceOAuthSession(session, channelID); saveErr != nil {
				common.ApiError(c, saveErr)
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": false, "message": "device authorization expired; start again",
				"terminal": true, "data": gin.H{"status": "expired"},
			})
			return
		}
		common.SysError("failed to continue codex device authorization: " + common.MaskSensitiveInfo(err.Error()))
		c.JSON(http.StatusOK, gin.H{
			"success": false, "message": "device authorization state is temporarily unavailable",
			"retryable": true, "retry_after": 2, "data": gin.H{"status": "pending"},
		})
		return
	}
	if state.Status == service.CodexDeviceAuthorizationPending || state.Status == service.CodexDeviceAuthorizationExchanging {
		retryAfter := 1
		if state.NextAttemptAt > time.Now().UnixMilli() {
			retryAfter = int(time.Until(time.UnixMilli(state.NextAttemptAt)).Seconds()) + 1
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true, "message": "pending", "retryable": true,
			"retry_after": retryAfter, "data": gin.H{"status": "pending"},
		})
		return
	}
	if err := clearCodexDeviceOAuthSession(session, channelID); err != nil {
		common.ApiError(c, err)
		return
	}
	if state.Status == service.CodexDeviceAuthorizationCancelled {
		c.JSON(http.StatusOK, gin.H{
			"success": false, "message": "device authorization cancelled",
			"terminal": true, "data": gin.H{"status": "cancelled"},
		})
		return
	}
	if state.Status == service.CodexDeviceAuthorizationUncertain {
		c.JSON(http.StatusOK, gin.H{
			"success": false, "message": state.Error, "requires_regeneration": true,
			"terminal": true, "data": gin.H{"status": service.CodexDeviceAuthorizationUncertain},
		})
		return
	}
	if state.Error != "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false, "message": state.Error,
			"terminal": true, "data": gin.H{"status": "terminal"},
		})
		return
	}
	if state.Result == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "device authorization completed without a saved credential"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "saved",
		"data":    state.Result,
	})
}

func CancelCodexDeviceOAuthForChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	if _, ok := getCodexOAuthChannelProxy(c, channelID); !ok {
		return
	}
	session := sessions.Default(c)
	deviceAuthID, _ := session.Get(codexOAuthSessionKey(channelID, "device_auth_id")).(string)
	if strings.TrimSpace(deviceAuthID) == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "cancelled"})
		return
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiError(c, errors.New("authenticated user id is required"))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	state, err := service.CancelCodexDeviceAuthorization(ctx, userID, channelID, deviceAuthID)
	if err != nil &&
		!errors.Is(err, service.ErrCodexDeviceAuthorizationNotFound) &&
		!errors.Is(err, service.ErrCodexDeviceAuthorizationExpired) {
		common.SysError("failed to cancel codex device authorization: " + common.MaskSensitiveInfo(err.Error()))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "failed to cancel device authorization"})
		return
	}
	if err := clearCodexDeviceOAuthSession(session, channelID); err != nil {
		common.ApiError(c, err)
		return
	}
	if state != nil && state.Status == service.CodexDeviceAuthorizationCompleted && state.Result != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "credential was already saved before cancellation"})
		return
	}
	if state != nil && state.Status == service.CodexDeviceAuthorizationUncertain {
		c.JSON(http.StatusOK, gin.H{
			"success": false, "message": state.Error, "requires_regeneration": true,
			"terminal": true, "data": gin.H{"status": service.CodexDeviceAuthorizationUncertain},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "cancelled"})
}

func getCodexOAuthChannelProxy(c *gin.Context, channelID int) (string, bool) {
	ch, err := model.GetChannelById(channelID, false)
	if err != nil {
		common.ApiError(c, err)
		return "", false
	}
	if ch == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel not found"})
		return "", false
	}
	if ch.Type != constant.ChannelTypeCodex {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel type is not Codex"})
		return "", false
	}
	return ch.GetSetting().Proxy, true
}

func saveCodexDeviceOAuthSession(session sessions.Session, channelID int, deviceAuthID string) error {
	session.Set(codexOAuthSessionKey(channelID, "device_auth_id"), strings.TrimSpace(deviceAuthID))
	session.Delete(codexOAuthSessionKey(channelID, "device_user_code"))
	session.Delete(codexOAuthSessionKey(channelID, "device_created_at"))
	return session.Save()
}

func clearCodexDeviceOAuthSession(session sessions.Session, channelID int) error {
	session.Delete(codexOAuthSessionKey(channelID, "device_auth_id"))
	session.Delete(codexOAuthSessionKey(channelID, "device_user_code"))
	session.Delete(codexOAuthSessionKey(channelID, "device_created_at"))
	return session.Save()
}

func GetCodexChannelCredential(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	meta, _, err := service.GetCodexCredentialMetadata(channelID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    meta,
	})
}

func ProbeCodexChannelCredential(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	meta, err := service.ProbeCodexChannelCredential(ctx, channelID)
	if err != nil {
		issue := service.ClassifyCodexCredentialIssue(err, 0)
		resp := gin.H{
			"success": false,
			"message": common.MaskSensitiveInfo(err.Error()),
			"data":    meta,
		}
		if issue.IsAuth {
			resp["message"] = issue.Message
			resp["code"] = issue.Code
			resp["requires_regeneration"] = issue.RequiresRegeneration
			resp["upstream_status"] = issue.UpstreamStatus
		}
		c.JSON(http.StatusOK, resp)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ok",
		"data":    meta,
	})
}

func CompleteCodexOAuth(c *gin.Context) {
	failClosedLegacyCodexPKCE(c)
}

func CompleteCodexOAuthForChannel(c *gin.Context) {
	failClosedLegacyCodexPKCE(c)
}

func prepareCodexOAuthTokenResult(channelID int, tokenRes *service.CodexOAuthTokenResult) (*service.CodexDeviceAuthorizationResult, string, error) {
	accountID, ok := service.ExtractCodexAccountIDFromJWT(tokenRes.AccessToken)
	if !ok {
		return nil, "", errors.New("failed to extract account_id from access_token")
	}
	email, _ := service.ExtractEmailFromJWT(tokenRes.AccessToken)

	key := codex.OAuthKey{
		AccessToken:  tokenRes.AccessToken,
		RefreshToken: tokenRes.RefreshToken,
		AccountID:    accountID,
		LastRefresh:  time.Now().Format(time.RFC3339),
		Expired:      tokenRes.ExpiresAt.Format(time.RFC3339),
		Email:        email,
		Type:         "codex",
	}
	encoded, err := common.Marshal(key)
	if err != nil {
		return nil, "", err
	}
	result := &service.CodexDeviceAuthorizationResult{
		ChannelID:   channelID,
		AccountID:   accountID,
		Email:       email,
		ExpiresAt:   key.Expired,
		LastRefresh: key.LastRefresh,
	}

	return result, string(encoded), nil
}
