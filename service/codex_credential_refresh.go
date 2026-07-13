package service

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
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const codexCredentialRefreshFreshnessWindow = 30 * time.Second

const (
	codexCredentialRefreshLeaseTTL     = 30 * time.Second
	codexCredentialRefreshOperationTTL = 24 * time.Hour
	codexCredentialRefreshStageStarted = "upstream_started"
)

var (
	errCodexCredentialRefreshInProgress = errors.New("codex credential refresh already in progress")
	errCodexCredentialRefreshUncertain  = errors.New("codex credential refresh outcome is unknown; regenerate the credential")
)

var codexCredentialRefreshGroup singleflight.Group

var codexCredentialRefreshCommitNow = func(_ time.Time) time.Time {
	return time.Now()
}

type CodexCredentialRefreshOptions struct {
	ResetCaches bool
}

type codexChannelOAuthRefreshFunc func(context.Context, string, string) (*CodexOAuthTokenResult, error)

type codexCredentialRefreshResult struct {
	key     *CodexOAuthKey
	channel *model.Channel
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
	LastUpstreamAuthAt    string `json:"last_upstream_auth_at,omitempty"`
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
	codexCredentialReasonRefreshUnknown  = "refresh_outcome_unknown"
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
	expiresAt, parseErr := time.Parse(time.RFC3339, meta.ExpiresAt)
	if parseErr != nil || !expiresAt.After(time.Now()) {
		meta.Authenticated = false
	}
	if health != nil {
		meta.LastProbeAt = strings.TrimSpace(health.LastProbeAt)
		meta.LastProbeStatus = strings.TrimSpace(health.LastProbeStatus)
		meta.LastUpstreamAuthAt = strings.TrimSpace(health.LastUpstreamAuthAt)
		meta.LastUpstreamStatus = health.LastUpstreamStatus
		meta.LastUpstreamAuthError = strings.TrimSpace(health.LastUpstreamAuthCode)
		meta.RequiresRegeneration = health.RequiresRegeneration
		meta.RegenerationReason = strings.TrimSpace(health.RegenerationReason)
		if health.RequiresRegeneration || health.LastUpstreamAuthCode != "" {
			meta.Authenticated = false
		}
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
			health.LastUpstreamAuthAt = time.Now().Format(time.RFC3339)
			health.LastUpstreamAuthCode = issue.UpstreamError
			health.LastUpstreamStatus = issue.UpstreamStatus
			health.RequiresRegeneration = issue.RequiresRegeneration
			health.RegenerationReason = issue.Reason
		}
		if healthErr := UpdateCodexCredentialHealth(ch, health); healthErr != nil {
			return nil, fmt.Errorf("failed to persist Codex credential probe failure: %w", healthErr)
		}
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
	return updateCodexCredentialHealth(ch, func(current *dto.CodexCredentialHealth) {
		*current = health
	})
}

func ClearCodexCredentialAuthIssue(ch *model.Channel) error {
	if ch == nil {
		return errors.New("channel not found")
	}
	return updateCodexCredentialHealth(ch, clearCodexCredentialAuthIssue)
}

func RecordCodexCredentialIssue(ch *model.Channel, issue CodexCredentialIssue) error {
	if ch == nil || !issue.IsAuth {
		return nil
	}
	return mergeCodexCredentialIssue(ch, "", issue)
}

func mergeCodexCredentialIssue(ch *model.Channel, expectedGenerationHash string, issue CodexCredentialIssue) error {
	return updateCodexCredentialHealthForGeneration(ch, expectedGenerationHash, func(health *dto.CodexCredentialHealth) {
		health.LastUpstreamAuthAt = time.Now().Format(time.RFC3339)
		health.LastUpstreamStatus = issue.UpstreamStatus
		health.LastUpstreamAuthCode = issue.UpstreamError
		health.RequiresRegeneration = issue.RequiresRegeneration
		health.RegenerationReason = issue.Reason
	})
}

func updateCodexCredentialHealth(ch *model.Channel, merge func(*dto.CodexCredentialHealth)) error {
	return updateCodexCredentialHealthForGeneration(ch, "", merge)
}

func updateCodexCredentialHealthForGeneration(ch *model.Channel, expectedGenerationHash string, merge func(*dto.CodexCredentialHealth)) error {
	var persistedSetting *string
	updated := false
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		current, err := lockCodexChannel(tx, ch.Id)
		if err != nil {
			return err
		}
		if expectedGenerationHash != "" {
			currentKey, err := parseCodexOAuthKey(current.Key)
			if err != nil {
				return err
			}
			if codexCredentialRefreshGenerationHash(currentKey) != expectedGenerationHash {
				return nil
			}
		}
		setting := current.GetSetting()
		health := dto.CodexCredentialHealth{}
		if setting.CodexCredentialHealth != nil {
			health = *setting.CodexCredentialHealth
		}
		merge(&health)
		setting.CodexCredentialHealth = &health
		current.SetSetting(setting)
		if current.Setting == nil {
			return errors.New("codex credential health setting is empty")
		}
		if err := tx.Model(&model.Channel{}).Where("id = ?", current.Id).Update("setting", *current.Setting).Error; err != nil {
			return err
		}
		persistedSetting = current.Setting
		updated = true
		return nil
	})
	if err == nil && updated {
		ch.Setting = persistedSetting
	}
	return err
}

func clearCodexCredentialAuthIssue(health *dto.CodexCredentialHealth) {
	health.LastUpstreamStatus = 0
	health.LastUpstreamAuthCode = ""
	health.LastUpstreamAuthAt = ""
	health.RequiresRegeneration = false
	health.RegenerationReason = ""
}

func lockCodexChannel(tx *gorm.DB, channelID int) (*model.Channel, error) {
	// A no-op UPDATE is the portable lock: row-level on MySQL/PostgreSQL and a
	// write lock on SQLite, which has no SELECT FOR UPDATE support.
	if err := tx.Model(&model.Channel{}).Where("id = ?", channelID).
		UpdateColumn("id", gorm.Expr("id")).Error; err != nil {
		return nil, err
	}
	var ch model.Channel
	if err := tx.First(&ch, channelID).Error; err != nil {
		return nil, err
	}
	return &ch, nil
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

func recordCodexCredentialIssueForGeneration(channelID int, generationHash string, issue CodexCredentialIssue) error {
	if channelID <= 0 || !issue.IsAuth {
		return nil
	}
	ch, err := model.GetChannelById(channelID, true)
	if err != nil {
		return err
	}
	return mergeCodexCredentialIssue(ch, generationHash, issue)
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
	case strings.Contains(text, "refresh outcome is unknown"):
		issue.IsAuth = true
		issue.Code = types.ErrorCodeCodexUpstreamRefreshInvalidated
		issue.Reason = codexCredentialReasonRefreshUnknown
		issue.UpstreamError = codexCredentialReasonRefreshUnknown
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

func ClassifyCodexUpstreamResponse(status int, body []byte) CodexCredentialIssue {
	return ClassifyCodexCredentialIssue(newCodexUpstreamAuthError("codex upstream request", status, body), status)
}

func CodexCredentialIssueMessage(issue CodexCredentialIssue) string {
	switch issue.Reason {
	case codexCredentialReasonMissingRefresh:
		return "codex credential cannot be refreshed because refresh_token is missing; regenerate the credential"
	case codexCredentialReasonRefreshInvalid:
		return "codex refresh_token was invalidated upstream; regenerate the credential"
	case codexCredentialReasonRefreshUnknown:
		return "codex refresh outcome is uncertain after process loss; regenerate the credential"
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
	return refreshCodexChannelCredential(ctx, channelID, opts, RefreshCodexOAuthTokenWithProxy)
}

func refreshCodexChannelCredential(ctx context.Context, channelID int, opts CodexCredentialRefreshOptions, refreshOAuth codexChannelOAuthRefreshFunc) (*CodexOAuthKey, *model.Channel, error) {
	var observed model.Channel
	if err := model.DB.WithContext(ctx).Select("id", "type", "key").First(&observed, channelID).Error; err != nil {
		return nil, nil, err
	}
	if observed.Type != constant.ChannelTypeCodex {
		return nil, nil, fmt.Errorf("channel type is not Codex")
	}
	observedKey, err := parseCodexOAuthKey(strings.TrimSpace(observed.Key))
	if err != nil {
		return nil, nil, err
	}
	observedGeneration := codexCredentialGeneration(observedKey)
	value, err, _ := codexCredentialRefreshGroup.Do(strconv.Itoa(channelID), func() (any, error) {
		return refreshCodexChannelCredentialAfterObservation(ctx, channelID, refreshOAuth, observedGeneration, time.Now())
	})
	if err != nil {
		return nil, nil, err
	}
	refreshed := value.(*codexCredentialRefreshResult)
	if opts.ResetCaches {
		model.InitChannelCache()
		ResetProxyClientCache()
	}
	return refreshed.key, refreshed.channel, nil
}

func refreshCodexChannelCredentialAfterObservation(
	ctx context.Context,
	channelID int,
	refreshOAuth codexChannelOAuthRefreshFunc,
	observedGeneration string,
	now time.Time,
	afterUpstream ...func() error,
) (*codexCredentialRefreshResult, error) {
	observedGenerationHash := codexCredentialRefreshGenerationValueHash(observedGeneration)
	claim, current, err := claimCodexCredentialRefresh(ctx, channelID, observedGeneration, now)
	if err != nil {
		return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, err)
	}
	if claim == nil {
		return current, nil
	}
	observedGenerationHash = codexCredentialRefreshGenerationHash(claim.currentKey)

	var refreshedKey *CodexOAuthKey
	if claim.record.Stage == CodexDeviceAuthorizationStageExchanged {
		refreshedKey = &CodexOAuthKey{}
		if err := openCodexOAuthPayload(claim.record.ProtectedPayload, refreshedKey); err != nil {
			return nil, err
		}
	} else {
		if err := markCodexCredentialRefreshStarted(ctx, claim, now); err != nil {
			return nil, err
		}
		refreshCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
		res, refreshErr := refreshOAuth(refreshCtx, claim.currentKey.RefreshToken, claim.proxyURL)
		cancel()
		if refreshErr != nil {
			var upstream *CodexUpstreamAuthError
			if errors.As(refreshErr, &upstream) {
				_ = releaseCodexCredentialRefresh(context.WithoutCancel(ctx), claim, now, refreshErr)
				return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, refreshErr)
			}
			// A timeout, connection loss, or malformed 2xx response can happen
			// after upstream rotated the token. Keep upstream_started fenced and
			// fail closed rather than retrying the old refresh token.
			uncertainErr := markCodexCredentialRefreshUncertain(ctx, claim, refreshErr)
			return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, errors.Join(errCodexCredentialRefreshUncertain, uncertainErr))
		}
		if len(afterUpstream) > 0 && afterUpstream[0] != nil {
			if hookErr := afterUpstream[0](); hookErr != nil {
				uncertainErr := markCodexCredentialRefreshUncertain(ctx, claim, hookErr)
				return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, errors.Join(errCodexCredentialRefreshUncertain, hookErr, uncertainErr))
			}
		}
		if res == nil {
			nilResultErr := errors.New("codex refresh returned no token result")
			uncertainErr := markCodexCredentialRefreshUncertain(ctx, claim, nilResultErr)
			return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, errors.Join(errCodexCredentialRefreshUncertain, nilResultErr, uncertainErr))
		}
		refreshedKey = mergeCodexCredentialRefreshResult(claim.currentKey, res, now)
		payload, sealErr := sealCodexOAuthPayload(refreshedKey)
		if sealErr != nil {
			uncertainErr := markCodexCredentialRefreshUncertain(ctx, claim, sealErr)
			return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, errors.Join(errCodexCredentialRefreshUncertain, sealErr, uncertainErr))
		}
		// Upstream token rotation cannot be atomic with SQL. This detached write
		// is intentionally the first operation after receiving the response; once
		// it commits, every retry recovers this encrypted result without reusing
		// the rotated refresh token.
		if err := persistCodexCredentialRefreshResult(ctx, claim, payload, now); err != nil {
			uncertainErr := markCodexCredentialRefreshUncertain(ctx, claim, err)
			return nil, recordCodexCredentialRefreshFailure(channelID, observedGenerationHash, errors.Join(
				errCodexCredentialRefreshUncertain,
				fmt.Errorf("codex refresh succeeded but durable recovery write failed: %w", err),
				uncertainErr,
			))
		}
	}

	refreshed, err := commitCodexCredentialRefresh(ctx, claim, refreshedKey, now)
	if err != nil {
		return nil, err
	}
	return refreshed, nil
}

func recordCodexCredentialRefreshFailure(channelID int, observedGenerationHash string, cause error) error {
	issue := ClassifyCodexCredentialIssue(cause, 0)
	if !issue.IsAuth {
		return cause
	}
	if err := recordCodexCredentialIssueForGeneration(channelID, observedGenerationHash, issue); err != nil {
		return errors.Join(cause, fmt.Errorf("failed to persist Codex auth health: %w", err))
	}
	return cause
}

type codexCredentialRefreshClaim struct {
	record     model.CodexOAuthOperation
	currentKey *CodexOAuthKey
	proxyURL   string
}

func codexCredentialRefreshOperationKey(channelID int) string {
	return fmt.Sprintf("codex:refresh:v2:%d", channelID)
}

func codexCredentialRefreshGenerationHash(key *CodexOAuthKey) string {
	return codexCredentialRefreshGenerationValueHash(codexCredentialGeneration(key))
}

func codexCredentialRefreshGenerationValueHash(generation string) string {
	return common.GenerateHMAC("codex-refresh-generation-v2:" + generation)
}

func claimCodexCredentialRefresh(
	ctx context.Context,
	channelID int,
	observedGeneration string,
	now time.Time,
) (*codexCredentialRefreshClaim, *codexCredentialRefreshResult, error) {
	if _, err := codexDeviceAuthorizationStateStore(ctx); err != nil {
		return nil, nil, err
	}
	owner, err := createStateHex(16)
	if err != nil {
		return nil, nil, err
	}
	operationKey := codexCredentialRefreshOperationKey(channelID)
	var claim *codexCredentialRefreshClaim
	var current *codexCredentialRefreshResult
	uncertainTakeover := false
	err = model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ch, err := lockCodexChannel(tx, channelID)
		if err != nil {
			return err
		}
		if ch.Type != constant.ChannelTypeCodex {
			return errors.New("channel type is not Codex")
		}
		oauthKey, err := parseCodexOAuthKey(strings.TrimSpace(ch.Key))
		if err != nil {
			return err
		}
		if strings.TrimSpace(oauthKey.RefreshToken) == "" {
			return errors.New("codex channel: refresh_token is required to refresh credential")
		}
		if codexCredentialGeneration(oauthKey) != observedGeneration || codexCredentialRecentlyRefreshed(oauthKey, now) {
			current = &codexCredentialRefreshResult{key: oauthKey, channel: ch}
			return nil
		}

		var record model.CodexOAuthOperation
		err = tx.Where("operation_key = ?", operationKey).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record = model.CodexOAuthOperation{
				OperationKey: operationKey, Kind: "refresh", ChannelID: channelID,
				Status: CodexDeviceAuthorizationExchanging, Stage: CodexDeviceAuthorizationStagePending,
				Owner: owner, Fence: 1, LeaseUntil: now.Add(codexCredentialRefreshLeaseTTL).UnixMilli(),
				ExpiresAt:      now.Add(codexCredentialRefreshOperationTTL).UnixMilli(),
				GenerationHash: codexCredentialRefreshGenerationHash(oauthKey),
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			claim = &codexCredentialRefreshClaim{record: record, currentKey: oauthKey, proxyURL: ch.GetSetting().Proxy}
			return nil
		}
		if err != nil {
			return err
		}
		if record.Status == CodexDeviceAuthorizationCompleted && record.ErrorMessage != "" {
			return errors.New("codex credential refresh terminal failure; regenerate the credential")
		}
		if record.Status == CodexDeviceAuthorizationUncertain {
			return errCodexCredentialRefreshUncertain
		}
		if record.ExpiresAt <= now.UnixMilli() || record.Status == CodexDeviceAuthorizationCompleted {
			if err := tx.Delete(&record).Error; err != nil {
				return err
			}
			record = model.CodexOAuthOperation{
				OperationKey: operationKey, Kind: "refresh", ChannelID: channelID,
				Status: CodexDeviceAuthorizationExchanging, Stage: CodexDeviceAuthorizationStagePending,
				Owner: owner, Fence: record.Fence + 1,
				LeaseUntil:     now.Add(codexCredentialRefreshLeaseTTL).UnixMilli(),
				ExpiresAt:      now.Add(codexCredentialRefreshOperationTTL).UnixMilli(),
				GenerationHash: codexCredentialRefreshGenerationHash(oauthKey),
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		} else {
			if record.LeaseUntil > now.UnixMilli() {
				return errCodexCredentialRefreshInProgress
			}
			if record.Stage == codexCredentialRefreshStageStarted {
				record.Status = CodexDeviceAuthorizationUncertain
				record.Owner = ""
				record.LeaseUntil = 0
				record.ErrorMessage = errCodexCredentialRefreshUncertain.Error()
				if err := tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", operationKey).Updates(map[string]any{
					"status": record.Status, "owner": "", "lease_until": 0,
					"error_message": record.ErrorMessage,
				}).Error; err != nil {
					return err
				}
				uncertainTakeover = true
				return nil
			}
			record.Owner = owner
			record.Fence++
			record.LeaseUntil = now.Add(codexCredentialRefreshLeaseTTL).UnixMilli()
			if err := tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", operationKey).Updates(map[string]any{
				"owner": record.Owner, "fence": record.Fence, "lease_until": record.LeaseUntil,
				"status": CodexDeviceAuthorizationExchanging,
			}).Error; err != nil {
				return err
			}
		}
		claim = &codexCredentialRefreshClaim{record: record, currentKey: oauthKey, proxyURL: ch.GetSetting().Proxy}
		return nil
	})
	if err == nil && uncertainTakeover {
		return nil, nil, errCodexCredentialRefreshUncertain
	}
	return claim, current, err
}

func markCodexCredentialRefreshStarted(ctx context.Context, claim *codexCredentialRefreshClaim, now time.Time) error {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), codexOAuthStoreTimeout)
	defer cancel()
	result := model.DB.WithContext(writeCtx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND owner = ? AND fence = ? AND expires_at > ?",
			claim.record.OperationKey, claim.record.Owner, claim.record.Fence, now.UnixMilli()).
		Updates(map[string]any{
			"stage":       codexCredentialRefreshStageStarted,
			"lease_until": now.Add(codexCredentialRefreshLeaseTTL).UnixMilli(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexDeviceAuthorizationLeaseLost
	}
	claim.record.Stage = codexCredentialRefreshStageStarted
	return nil
}

func markCodexCredentialRefreshUncertain(ctx context.Context, claim *codexCredentialRefreshClaim, cause error) error {
	if claim == nil {
		return errors.New("codex credential refresh claim is missing")
	}
	message := errCodexCredentialRefreshUncertain.Error()
	if cause != nil {
		message += ": " + common.MaskSensitiveInfo(cause.Error())
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), codexOAuthStoreTimeout)
	defer cancel()
	result := model.DB.WithContext(writeCtx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND owner = ? AND fence = ? AND stage = ?",
			claim.record.OperationKey, claim.record.Owner, claim.record.Fence, codexCredentialRefreshStageStarted).
		Updates(map[string]any{
			"status": CodexDeviceAuthorizationUncertain, "owner": "", "lease_until": 0,
			"error_message": message,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexDeviceAuthorizationLeaseLost
	}
	claim.record.Status = CodexDeviceAuthorizationUncertain
	claim.record.Owner = ""
	claim.record.LeaseUntil = 0
	claim.record.ErrorMessage = message
	return nil
}

func releaseCodexCredentialRefresh(ctx context.Context, claim *codexCredentialRefreshClaim, now time.Time, cause error) error {
	status := CodexDeviceAuthorizationPending
	message := ""
	if isCodexDeviceAuthorizationTerminalError(cause) {
		status = CodexDeviceAuthorizationCompleted
		message = common.MaskSensitiveInfo(cause.Error())
	}
	writeCtx, cancel := context.WithTimeout(ctx, codexOAuthStoreTimeout)
	defer cancel()
	return model.DB.WithContext(writeCtx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND owner = ? AND fence = ?",
			claim.record.OperationKey, claim.record.Owner, claim.record.Fence).
		Updates(map[string]any{
			"status": status, "stage": CodexDeviceAuthorizationStagePending,
			"owner": "", "lease_until": 0, "error_message": message,
		}).Error
}

func persistCodexCredentialRefreshResult(ctx context.Context, claim *codexCredentialRefreshClaim, payload string, now time.Time) error {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), codexOAuthStoreTimeout)
	defer cancel()
	result := model.DB.WithContext(writeCtx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND owner = ? AND fence = ? AND expires_at > ?",
			claim.record.OperationKey, claim.record.Owner, claim.record.Fence, now.UnixMilli()).
		Updates(map[string]any{
			"stage": CodexDeviceAuthorizationStageExchanged, "protected_payload": payload,
			"lease_until": now.Add(codexCredentialRefreshLeaseTTL).UnixMilli(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexDeviceAuthorizationLeaseLost
	}
	claim.record.Stage = CodexDeviceAuthorizationStageExchanged
	claim.record.ProtectedPayload = payload
	return nil
}

func mergeCodexCredentialRefreshResult(current *CodexOAuthKey, result *CodexOAuthTokenResult, now time.Time) *CodexOAuthKey {
	refreshed := *current
	refreshed.AccessToken = result.AccessToken
	refreshed.RefreshToken = result.RefreshToken
	refreshed.LastRefresh = now.Format(time.RFC3339)
	refreshed.Expired = result.ExpiresAt.Format(time.RFC3339)
	if strings.TrimSpace(refreshed.Type) == "" {
		refreshed.Type = "codex"
	}
	if strings.TrimSpace(refreshed.AccountID) == "" {
		refreshed.AccountID, _ = ExtractCodexAccountIDFromJWT(refreshed.AccessToken)
	}
	if strings.TrimSpace(refreshed.Email) == "" {
		refreshed.Email, _ = ExtractEmailFromJWT(refreshed.AccessToken)
	}
	return &refreshed
}

func commitCodexCredentialRefresh(
	ctx context.Context,
	claim *codexCredentialRefreshClaim,
	refreshedKey *CodexOAuthKey,
	now time.Time,
) (*codexCredentialRefreshResult, error) {
	encoded, err := common.Marshal(refreshedKey)
	if err != nil {
		return nil, err
	}
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), codexOAuthStoreTimeout)
	defer cancel()
	var refreshed codexCredentialRefreshResult
	err = model.DB.WithContext(commitCtx).Transaction(func(tx *gorm.DB) error {
		var record model.CodexOAuthOperation
		if err := lockCodexOAuthOperation(tx, claim.record.OperationKey, &record); err != nil {
			return err
		}
		ch, err := lockCodexChannel(tx, claim.record.ChannelID)
		if err != nil {
			return err
		}
		commitNow := codexCredentialRefreshCommitNow(now)
		if record.ExpiresAt <= commitNow.UnixMilli() {
			return ErrCodexDeviceAuthorizationExpired
		}
		if record.Owner != claim.record.Owner || record.Fence != claim.record.Fence ||
			record.Stage != CodexDeviceAuthorizationStageExchanged ||
			record.LeaseUntil <= commitNow.UnixMilli() {
			return ErrCodexDeviceAuthorizationLeaseLost
		}
		currentKey, err := parseCodexOAuthKey(ch.Key)
		if err != nil {
			return err
		}
		if codexCredentialRefreshGenerationHash(currentKey) != record.GenerationHash {
			update := tx.Model(&model.CodexOAuthOperation{}).
				Where("operation_key = ? AND fence = ?", record.OperationKey, record.Fence).
				Updates(map[string]any{
					"status": CodexDeviceAuthorizationCompleted, "stage": CodexDeviceAuthorizationStageSaved,
					"owner": "", "lease_until": 0, "protected_payload": "",
					"error_message": "superseded by a newer channel credential",
				})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return ErrCodexDeviceAuthorizationLeaseLost
			}
			refreshed = codexCredentialRefreshResult{key: currentKey, channel: ch}
			return nil
		}
		setting := ch.GetSetting()
		health := dto.CodexCredentialHealth{}
		if setting.CodexCredentialHealth != nil {
			health = *setting.CodexCredentialHealth
		}
		clearCodexCredentialAuthIssue(&health)
		setting.CodexCredentialHealth = &health
		ch.SetSetting(setting)
		if ch.Setting == nil {
			return errors.New("codex credential health setting is empty")
		}
		if err := tx.Model(&model.Channel{}).Where("id = ?", ch.Id).Updates(map[string]any{
			"key": string(encoded), "setting": *ch.Setting,
		}).Error; err != nil {
			return err
		}
		update := tx.Model(&model.CodexOAuthOperation{}).
			Where("operation_key = ? AND status = ? AND stage = ? AND owner = ? AND fence = ? AND expires_at > ? AND lease_until > ?",
				record.OperationKey, CodexDeviceAuthorizationExchanging, CodexDeviceAuthorizationStageExchanged,
				record.Owner, record.Fence, commitNow.UnixMilli(), commitNow.UnixMilli()).
			Updates(map[string]any{
				"status": CodexDeviceAuthorizationCompleted, "stage": CodexDeviceAuthorizationStageSaved,
				"owner": "", "lease_until": 0, "protected_payload": "", "error_message": "",
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrCodexDeviceAuthorizationLeaseLost
		}
		ch.Key = string(encoded)
		refreshed = codexCredentialRefreshResult{key: refreshedKey, channel: ch}
		return nil
	})
	return &refreshed, err
}

func codexCredentialGeneration(key *CodexOAuthKey) string {
	return key.AccessToken + "\x00" + key.RefreshToken + "\x00" + key.LastRefresh
}

func codexCredentialRecentlyRefreshed(key *CodexOAuthKey, now time.Time) bool {
	lastRefresh, err := time.Parse(time.RFC3339, strings.TrimSpace(key.LastRefresh))
	if err != nil {
		return false
	}
	age := now.Sub(lastRefresh)
	return age >= 0 && age < codexCredentialRefreshFreshnessWindow
}
