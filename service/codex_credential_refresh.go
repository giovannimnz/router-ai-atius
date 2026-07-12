package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

type CodexCredentialRefreshOptions struct {
	ResetCaches bool
}

type CodexOAuthKey struct {
	IDToken      string `json:"id_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`

	AccountID   string `json:"account_id,omitempty"`
	LastRefresh string `json:"last_refresh,omitempty"`
	Email       string `json:"email,omitempty"`
	Type        string `json:"type,omitempty"`
	Expired     string `json:"expired,omitempty"`
}

type CodexCredentialMetadata struct {
	ChannelID             int    `json:"channel_id"`
	ChannelType           int    `json:"channel_type"`
	ChannelName           string `json:"channel_name"`
	Authenticated         bool   `json:"authenticated"`
	HasRefreshToken       bool   `json:"has_refresh_token"`
	RequiresRegeneration  bool   `json:"requires_regeneration"`
	RegenerationReason    string `json:"regeneration_reason,omitempty"`
	ExpiresAt             string `json:"expires_at,omitempty"`
	LastRefresh           string `json:"last_refresh,omitempty"`
	AccountID             string `json:"account_id,omitempty"`
	Email                 string `json:"email,omitempty"`
	LastProbeAt           string `json:"last_probe_at,omitempty"`
	LastProbeStatus       string `json:"last_probe_status,omitempty"`
	LastUpstreamStatus    int    `json:"last_upstream_status,omitempty"`
	LastUpstreamAuthError string `json:"last_upstream_auth_error,omitempty"`
}

type CodexCredentialIssue struct {
	IsAuth               bool
	Code                 types.ErrorCode
	Reason               string
	Message              string
	UpstreamError        string
	UpstreamStatus       int
	RequiresRegeneration bool
}

const (
	codexCredentialProbeStatusOK         = "ok"
	codexCredentialProbeStatusFailed     = "failed"
	codexCredentialProbeStatusAuthFailed = "auth_failed"
	codexCredentialReasonMissingRefresh  = "refresh_token_missing"
	codexCredentialReasonRefreshInvalid  = "refresh_token_invalidated"
	codexCredentialReasonTokenInvalid    = "token_invalidated"
	codexCredentialReasonInvalidAPIKey   = "invalid_api_key"
	codexCredentialReasonAuthStatus      = "upstream_auth_status"
	codexCredentialReasonIncomplete      = "credential_incomplete"
)

func parseCodexOAuthKey(raw string) (*CodexOAuthKey, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("codex channel: empty oauth key")
	}
	var key CodexOAuthKey
	if err := common.Unmarshal([]byte(raw), &key); err != nil {
		return nil, errors.New("codex channel: invalid oauth key json")
	}
	return &key, nil
}

func BuildCodexCredentialMetadata(ch *model.Channel) (*CodexCredentialMetadata, error) {
	if ch == nil {
		return nil, errors.New("channel not found")
	}
	if ch.Type != constant.ChannelTypeCodex {
		return nil, fmt.Errorf("channel type is not Codex")
	}
	oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(ch.Key))
	if err != nil {
		return nil, err
	}
	health := ch.GetSetting().CodexCredentialHealth
	hasAccessIdentity := strings.TrimSpace(oauthKey.AccessToken) != "" && strings.TrimSpace(oauthKey.AccountID) != ""
	meta := &CodexCredentialMetadata{
		ChannelID:       ch.Id,
		ChannelType:     ch.Type,
		ChannelName:     ch.Name,
		Authenticated:   hasAccessIdentity,
		HasRefreshToken: strings.TrimSpace(oauthKey.RefreshToken) != "",
		ExpiresAt:       strings.TrimSpace(oauthKey.Expired),
		LastRefresh:     strings.TrimSpace(oauthKey.LastRefresh),
		AccountID:       strings.TrimSpace(oauthKey.AccountID),
		Email:           strings.TrimSpace(oauthKey.Email),
	}
	if expiresAt, parseErr := time.Parse(time.RFC3339, meta.ExpiresAt); parseErr == nil && !expiresAt.After(time.Now()) {
		meta.Authenticated = false
	}
	if health != nil {
		meta.LastProbeAt = strings.TrimSpace(health.LastProbeAt)
		meta.LastProbeStatus = strings.TrimSpace(health.LastProbeStatus)
		meta.LastUpstreamStatus = health.LastUpstreamStatus
		meta.LastUpstreamAuthError = strings.TrimSpace(health.LastUpstreamAuthCode)
		meta.RequiresRegeneration = health.RequiresRegeneration
		meta.RegenerationReason = strings.TrimSpace(health.RegenerationReason)
	}
	if !hasAccessIdentity {
		meta.RequiresRegeneration = true
		meta.RegenerationReason = codexCredentialReasonIncomplete
	}
	if !meta.HasRefreshToken {
		meta.RequiresRegeneration = true
		meta.RegenerationReason = codexCredentialReasonMissingRefresh
	}
	return meta, nil
}

func GetCodexCredentialMetadata(channelID int) (*CodexCredentialMetadata, *model.Channel, error) {
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, nil, err
	}
	meta, err := BuildCodexCredentialMetadata(ch)
	return meta, ch, err
}

func ProbeCodexChannelCredential(ctx context.Context, channelID int) (*CodexCredentialMetadata, error) {
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, errors.New("channel not found")
	}
	if ch.Type != constant.ChannelTypeCodex {
		return nil, fmt.Errorf("channel type is not Codex")
	}
	if _, err := doCodexDiscoveryRequest(ctx, ch, resolveCodexDiscoveryClientVersion()); err != nil {
		issue := ClassifyCodexCredentialIssue(err, 0)
		health := dto.CodexCredentialHealth{
			LastProbeAt:     time.Now().Format(time.RFC3339),
			LastProbeStatus: codexCredentialProbeStatusFailed,
		}
		if issue.IsAuth {
			health.LastProbeStatus = codexCredentialProbeStatusAuthFailed
			health.LastUpstreamAuthCode = issue.UpstreamError
			health.LastUpstreamStatus = issue.UpstreamStatus
			health.RequiresRegeneration = issue.RequiresRegeneration
			health.RegenerationReason = issue.Reason
		}
		_ = UpdateCodexCredentialHealth(ch, health)
		meta, metaErr := BuildCodexCredentialMetadata(ch)
		if metaErr != nil {
			return nil, err
		}
		return meta, err
	}
	health := dto.CodexCredentialHealth{
		LastProbeAt:          time.Now().Format(time.RFC3339),
		LastProbeStatus:      codexCredentialProbeStatusOK,
		RequiresRegeneration: false,
	}
	if err := UpdateCodexCredentialHealth(ch, health); err != nil {
		return nil, err
	}
	return BuildCodexCredentialMetadata(ch)
}

func UpdateCodexCredentialHealth(ch *model.Channel, health dto.CodexCredentialHealth) error {
	if ch == nil {
		return errors.New("channel not found")
	}
	setting := ch.GetSetting()
	setting.CodexCredentialHealth = &health
	ch.SetSetting(setting)
	if ch.Setting == nil {
		return errors.New("codex credential health setting is empty")
	}
	return model.DB.Model(&model.Channel{}).Where("id = ?", ch.Id).Update("setting", *ch.Setting).Error
}

func ClearCodexCredentialAuthIssue(ch *model.Channel) error {
	if ch == nil {
		return errors.New("channel not found")
	}
	setting := ch.GetSetting()
	health := dto.CodexCredentialHealth{}
	if setting.CodexCredentialHealth != nil {
		health = *setting.CodexCredentialHealth
	}
	health.LastUpstreamStatus = 0
	health.LastUpstreamAuthCode = ""
	health.RequiresRegeneration = false
	health.RegenerationReason = ""
	return UpdateCodexCredentialHealth(ch, health)
}

func RecordCodexCredentialIssue(ch *model.Channel, issue CodexCredentialIssue) error {
	if ch == nil || !issue.IsAuth {
		return nil
	}
	health := dto.CodexCredentialHealth{
		LastProbeAt:          time.Now().Format(time.RFC3339),
		LastProbeStatus:      codexCredentialProbeStatusAuthFailed,
		LastUpstreamStatus:   issue.UpstreamStatus,
		LastUpstreamAuthCode: issue.UpstreamError,
		RequiresRegeneration: issue.RequiresRegeneration,
		RegenerationReason:   issue.Reason,
	}
	return UpdateCodexCredentialHealth(ch, health)
}

func RecordCodexCredentialIssueByChannelID(channelID int, issue CodexCredentialIssue) error {
	if channelID <= 0 || !issue.IsAuth {
		return nil
	}
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return err
	}
	return RecordCodexCredentialIssue(ch, issue)
}

func ClassifyCodexCredentialIssue(err error, fallbackStatus int) CodexCredentialIssue {
	issue := CodexCredentialIssue{Code: types.ErrorCodeCodexUpstreamAuthFailed}
	if err == nil {
		return issue
	}
	status := fallbackStatus
	upstreamError := ""
	var upstreamAuthErr *CodexUpstreamAuthError
	if errors.As(err, &upstreamAuthErr) {
		status = upstreamAuthErr.Status
		upstreamError = strings.TrimSpace(upstreamAuthErr.UpstreamError)
	}
	var newAPIError *types.NewAPIError
	if errors.As(err, &newAPIError) {
		if status == 0 {
			status = newAPIError.StatusCode
		}
		upstreamError = strings.TrimSpace(string(newAPIError.GetErrorCode()))
	}
	text := strings.ToLower(err.Error())
	lowerUpstreamError := strings.ToLower(upstreamError)
	switch {
	case strings.Contains(text, "refresh_token is required") || strings.Contains(text, "empty refresh_token"):
		issue.IsAuth = true
		issue.Code = types.ErrorCodeCodexUpstreamRefreshInvalidated
		issue.Reason = codexCredentialReasonMissingRefresh
		issue.UpstreamError = codexCredentialReasonMissingRefresh
	case lowerUpstreamError == codexCredentialReasonRefreshInvalid || strings.Contains(text, codexCredentialReasonRefreshInvalid):
		issue.IsAuth = true
		issue.Code = types.ErrorCodeCodexUpstreamRefreshInvalidated
		issue.Reason = codexCredentialReasonRefreshInvalid
		issue.UpstreamError = codexCredentialReasonRefreshInvalid
	case lowerUpstreamError == codexCredentialReasonTokenInvalid || strings.Contains(text, codexCredentialReasonTokenInvalid):
		issue.IsAuth = true
		issue.Code = types.ErrorCodeCodexUpstreamTokenInvalidated
		issue.Reason = codexCredentialReasonTokenInvalid
		issue.UpstreamError = codexCredentialReasonTokenInvalid
	case lowerUpstreamError == codexCredentialReasonInvalidAPIKey || strings.Contains(text, codexCredentialReasonInvalidAPIKey):
		issue.IsAuth = true
		issue.Code = types.ErrorCodeCodexUpstreamAuthFailed
		issue.Reason = codexCredentialReasonInvalidAPIKey
		issue.UpstreamError = codexCredentialReasonInvalidAPIKey
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		issue.IsAuth = true
		issue.Code = types.ErrorCodeCodexUpstreamAuthFailed
		issue.Reason = codexCredentialReasonAuthStatus
		issue.UpstreamError = codexCredentialReasonAuthStatus
	}
	if !issue.IsAuth {
		return issue
	}
	issue.UpstreamStatus = status
	issue.RequiresRegeneration = true
	issue.Message = CodexCredentialIssueMessage(issue)
	return issue
}

func CodexCredentialIssueMessage(issue CodexCredentialIssue) string {
	switch issue.Reason {
	case codexCredentialReasonMissingRefresh:
		return "codex credential cannot be refreshed because refresh_token is missing; regenerate the credential"
	case codexCredentialReasonRefreshInvalid:
		return "codex refresh_token was invalidated upstream; regenerate the credential"
	case codexCredentialReasonTokenInvalid:
		return "codex access_token was invalidated upstream; refresh or regenerate the credential"
	case codexCredentialReasonInvalidAPIKey, codexCredentialReasonAuthStatus:
		return "codex upstream authentication failed; refresh or regenerate the credential"
	default:
		return "codex upstream authentication failed"
	}
}

func NormalizeCodexUpstreamAuthError(err *types.NewAPIError) *types.NewAPIError {
	if err == nil {
		return nil
	}
	issue := ClassifyCodexCredentialIssue(err, err.StatusCode)
	if !issue.IsAuth {
		return err
	}
	openAIError := types.OpenAIError{
		Message: issue.Message,
		Type:    "codex_upstream_auth_error",
		Code:    issue.Code,
	}
	return types.WithOpenAIError(openAIError, err.StatusCode, types.ErrOptionWithSkipRetry())
}

func RefreshCodexChannelCredential(ctx context.Context, channelID int, opts CodexCredentialRefreshOptions) (*CodexOAuthKey, *model.Channel, error) {
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return nil, nil, err
	}
	if ch == nil {
		return nil, nil, fmt.Errorf("channel not found")
	}
	if ch.Type != constant.ChannelTypeCodex {
		return nil, nil, fmt.Errorf("channel type is not Codex")
	}

	oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(ch.Key))
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(oauthKey.RefreshToken) == "" {
		return nil, nil, fmt.Errorf("codex channel: refresh_token is required to refresh credential")
	}

	refreshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res, err := RefreshCodexOAuthTokenWithProxy(refreshCtx, oauthKey.RefreshToken, ch.GetSetting().Proxy)
	if err != nil {
		_ = RecordCodexCredentialIssue(ch, ClassifyCodexCredentialIssue(err, 0))
		return nil, nil, err
	}

	oauthKey.AccessToken = res.AccessToken
	oauthKey.RefreshToken = res.RefreshToken
	oauthKey.LastRefresh = time.Now().Format(time.RFC3339)
	oauthKey.Expired = res.ExpiresAt.Format(time.RFC3339)
	if strings.TrimSpace(oauthKey.Type) == "" {
		oauthKey.Type = "codex"
	}

	if strings.TrimSpace(oauthKey.AccountID) == "" {
		if accountID, ok := ExtractCodexAccountIDFromJWT(oauthKey.AccessToken); ok {
			oauthKey.AccountID = accountID
		}
	}
	if strings.TrimSpace(oauthKey.Email) == "" {
		if email, ok := ExtractEmailFromJWT(oauthKey.AccessToken); ok {
			oauthKey.Email = email
		}
	}

	encoded, err := common.Marshal(oauthKey)
	if err != nil {
		return nil, nil, err
	}

	if err := model.DB.Model(&model.Channel{}).Where("id = ?", ch.Id).Update("key", string(encoded)).Error; err != nil {
		return nil, nil, err
	}
	_ = ClearCodexCredentialAuthIssue(ch)

	if opts.ResetCaches {
		model.InitChannelCache()
		ResetProxyClientCache()
	}

	return oauthKey, ch, nil
}
