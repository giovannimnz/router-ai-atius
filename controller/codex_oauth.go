package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

type codexOAuthCompleteRequest struct {
	Input string `json:"input"`
}

const codexDeviceAuthorizationTTL = 15 * time.Minute

func codexOAuthSessionKey(channelID int, field string) string {
	return fmt.Sprintf("codex_oauth_%s_%d", field, channelID)
}

func parseCodexAuthorizationInput(input string) (code string, state string, err error) {
	v := strings.TrimSpace(input)
	if v == "" {
		return "", "", errors.New("empty input")
	}
	if strings.Contains(v, "#") {
		parts := strings.SplitN(v, "#", 2)
		code = strings.TrimSpace(parts[0])
		state = strings.TrimSpace(parts[1])
		return code, state, nil
	}
	if strings.Contains(v, "code=") {
		u, parseErr := url.Parse(v)
		if parseErr == nil {
			q := u.Query()
			code = strings.TrimSpace(q.Get("code"))
			state = strings.TrimSpace(q.Get("state"))
			return code, state, nil
		}
		q, parseErr := url.ParseQuery(v)
		if parseErr == nil {
			code = strings.TrimSpace(q.Get("code"))
			state = strings.TrimSpace(q.Get("state"))
			return code, state, nil
		}
	}

	code = v
	return code, "", nil
}

func StartCodexOAuth(c *gin.Context) {
	startCodexOAuthWithChannelID(c, 0)
}

func StartCodexOAuthForChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	startCodexOAuthWithChannelID(c, channelID)
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
	flow, err := service.StartCodexDeviceAuthorization(ctx, channelProxy)
	if err != nil {
		common.SysError("failed to start codex device authorization: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "failed to start device authorization"})
		return
	}

	session := sessions.Default(c)
	session.Set(codexOAuthSessionKey(channelID, "device_auth_id"), flow.DeviceAuthID)
	session.Set(codexOAuthSessionKey(channelID, "device_user_code"), flow.UserCode)
	session.Set(codexOAuthSessionKey(channelID, "device_created_at"), time.Now().Unix())
	if err := session.Save(); err != nil {
		common.ApiError(c, err)
		return
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
	userCode, _ := session.Get(codexOAuthSessionKey(channelID, "device_user_code")).(string)
	createdAt := codexOAuthSessionUnix(session.Get(codexOAuthSessionKey(channelID, "device_created_at")))
	if strings.TrimSpace(deviceAuthID) == "" || strings.TrimSpace(userCode) == "" || createdAt == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "device authorization not started or session expired"})
		return
	}
	if time.Since(time.Unix(createdAt, 0)) > codexDeviceAuthorizationTTL {
		clearCodexDeviceOAuthSession(session, channelID)
		_ = session.Save()
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "device authorization expired; start again"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	poll, err := service.PollCodexDeviceAuthorization(ctx, deviceAuthID, userCode, channelProxy)
	if err != nil {
		common.SysError("failed to poll codex device authorization: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "device authorization polling failed"})
		return
	}
	if poll.Pending {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "pending", "data": gin.H{"status": "pending"}})
		return
	}
	tokenRes, err := service.ExchangeCodexDeviceAuthorizationCode(ctx, poll.AuthorizationCode, poll.CodeVerifier, channelProxy)
	if err != nil {
		common.SysError("failed to exchange codex device authorization code: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "device authorization exchange failed; start again"})
		return
	}
	clearCodexDeviceOAuthSession(session, channelID)
	_ = session.Save()
	saveCodexOAuthTokenResult(c, channelID, tokenRes)
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

func codexOAuthSessionUnix(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func clearCodexDeviceOAuthSession(session sessions.Session, channelID int) {
	session.Delete(codexOAuthSessionKey(channelID, "device_auth_id"))
	session.Delete(codexOAuthSessionKey(channelID, "device_user_code"))
	session.Delete(codexOAuthSessionKey(channelID, "device_created_at"))
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

func startCodexOAuthWithChannelID(c *gin.Context, channelID int) {
	if channelID > 0 {
		ch, err := model.GetChannelById(channelID, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if ch == nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel not found"})
			return
		}
		if ch.Type != constant.ChannelTypeCodex {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel type is not Codex"})
			return
		}
	}

	flow, err := service.CreateCodexOAuthAuthorizationFlow()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	session := sessions.Default(c)
	session.Set(codexOAuthSessionKey(channelID, "state"), flow.State)
	session.Set(codexOAuthSessionKey(channelID, "verifier"), flow.Verifier)
	session.Set(codexOAuthSessionKey(channelID, "created_at"), time.Now().Unix())
	_ = session.Save()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"authorize_url": flow.AuthorizeURL,
		},
	})
}

func CompleteCodexOAuth(c *gin.Context) {
	completeCodexOAuthWithChannelID(c, 0)
}

func CompleteCodexOAuthForChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}
	completeCodexOAuthWithChannelID(c, channelID)
}

func completeCodexOAuthWithChannelID(c *gin.Context, channelID int) {
	req := codexOAuthCompleteRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	code, state, err := parseCodexAuthorizationInput(req.Input)
	if err != nil {
		common.SysError("failed to parse codex authorization input: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解析授权信息失败，请检查输入格式"})
		return
	}
	if strings.TrimSpace(code) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "missing authorization code"})
		return
	}
	if strings.TrimSpace(state) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "missing state in input"})
		return
	}

	channelProxy := ""
	if channelID > 0 {
		ch, err := model.GetChannelById(channelID, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if ch == nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel not found"})
			return
		}
		if ch.Type != constant.ChannelTypeCodex {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel type is not Codex"})
			return
		}
		channelProxy = ch.GetSetting().Proxy
	}

	session := sessions.Default(c)
	expectedState, _ := session.Get(codexOAuthSessionKey(channelID, "state")).(string)
	verifier, _ := session.Get(codexOAuthSessionKey(channelID, "verifier")).(string)
	if strings.TrimSpace(expectedState) == "" || strings.TrimSpace(verifier) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "oauth flow not started or session expired"})
		return
	}
	if state != expectedState {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "state mismatch"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	tokenRes, err := service.ExchangeCodexAuthorizationCodeWithProxy(ctx, code, verifier, channelProxy)
	if err != nil {
		common.SysError("failed to exchange codex authorization code: " + err.Error())
		resp := gin.H{"success": false, "message": "authorization code exchange failed; retry regeneration"}
		if issue := service.ClassifyCodexCredentialIssue(err, 0); issue.IsAuth {
			resp["message"] = issue.Message
			resp["code"] = issue.Code
			resp["requires_regeneration"] = issue.RequiresRegeneration
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	saveCodexOAuthTokenResult(c, channelID, tokenRes)
}

func saveCodexOAuthTokenResult(c *gin.Context, channelID int, tokenRes *service.CodexOAuthTokenResult) {
	accountID, ok := service.ExtractCodexAccountIDFromJWT(tokenRes.AccessToken)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "failed to extract account_id from access_token"})
		return
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
		common.ApiError(c, err)
		return
	}

	session := sessions.Default(c)
	session.Delete(codexOAuthSessionKey(channelID, "state"))
	session.Delete(codexOAuthSessionKey(channelID, "verifier"))
	session.Delete(codexOAuthSessionKey(channelID, "created_at"))
	_ = session.Save()

	if channelID > 0 {
		if err := model.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("key", string(encoded)).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		if refreshedCh, getErr := model.GetChannelById(channelID, true); getErr == nil && refreshedCh != nil {
			if clearErr := service.ClearCodexCredentialAuthIssue(refreshedCh); clearErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "credential saved but failed to clear prior Codex auth health; retry the operation",
				})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "credential saved but failed to reload the Codex channel; retry the operation",
			})
			return
		}
		model.InitChannelCache()
		service.ResetProxyClientCache()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "saved",
			"data": gin.H{
				"channel_id":   channelID,
				"account_id":   accountID,
				"email":        email,
				"expires_at":   key.Expired,
				"last_refresh": key.LastRefresh,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "generated",
		"data": gin.H{
			"key":          string(encoded),
			"account_id":   accountID,
			"email":        email,
			"expires_at":   key.Expired,
			"last_refresh": key.LastRefresh,
		},
	})
}
