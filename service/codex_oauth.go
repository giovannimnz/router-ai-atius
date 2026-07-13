package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	codexOAuthClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	codexOAuthTokenURL     = "https://auth.openai.com/oauth/token"
	codexOAuthRedirectURI  = "http://localhost:1455/auth/callback"
	codexDeviceUserCodeURL = "https://auth.openai.com/api/accounts/deviceauth/usercode"
	codexDeviceTokenURL    = "https://auth.openai.com/api/accounts/deviceauth/token"
	codexDeviceVerifyURL   = "https://auth.openai.com/codex/device"
	codexDeviceRedirectURI = "https://auth.openai.com/deviceauth/callback"
	codexOAuthScope        = "openid profile email offline_access"
	codexJWTClaimPath      = "https://api.openai.com/auth"
	defaultHTTPTimeout     = 20 * time.Second
	codexDeviceFlowTTL     = 15 * time.Minute
	codexDeviceLeaseTTL    = 30 * time.Second
	codexOAuthStoreTimeout = 3 * time.Second
)

const (
	CodexDeviceAuthorizationPending    = "pending"
	CodexDeviceAuthorizationExchanging = "exchanging"
	CodexDeviceAuthorizationCompleted  = "completed"
	CodexDeviceAuthorizationCancelled  = "cancelled"
	CodexDeviceAuthorizationUncertain  = "uncertain_requires_regeneration"

	CodexDeviceAuthorizationStagePending         = "pending"
	CodexDeviceAuthorizationStagePolled          = "polled"
	CodexDeviceAuthorizationStageExchangeStarted = "exchange_started"
	CodexDeviceAuthorizationStageExchanged       = "exchanged"
	CodexDeviceAuthorizationStageSaved           = "saved"
)

var (
	ErrCodexDeviceAuthorizationNotFound  = errors.New("codex device authorization not found")
	ErrCodexDeviceAuthorizationExpired   = errors.New("codex device authorization expired")
	ErrCodexDeviceAuthorizationExists    = errors.New("codex device authorization already exists")
	ErrCodexDeviceAuthorizationCancelled = errors.New("codex device authorization cancelled")
	ErrCodexDeviceAuthorizationLeaseLost = errors.New("codex device authorization lease lost")
	ErrCodexOAuthStoreUnavailable        = errors.New("shared SQL OAuth operation store unavailable")
)

type CodexOAuthTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type CodexOAuthAuthorizationFlow struct {
	State        string
	Verifier     string
	Challenge    string
	AuthorizeURL string
}

type CodexDeviceAuthorizationStart struct {
	DeviceAuthID    string
	UserCode        string
	VerificationURL string
	Interval        time.Duration
	ExpiresAt       time.Time
}

type CodexDeviceAuthorizationPoll struct {
	Pending           bool
	AuthorizationCode string
	CodeVerifier      string
}

type CodexDeviceAuthorizationResult struct {
	ChannelID   int    `json:"channel_id"`
	AccountID   string `json:"account_id,omitempty"`
	Email       string `json:"email,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	LastRefresh string `json:"last_refresh,omitempty"`
}

type CodexDeviceAuthorizationState struct {
	UserID           int                             `json:"user_id"`
	ChannelID        int                             `json:"channel_id"`
	DeviceAuthID     string                          `json:"device_auth_id"`
	UserCode         string                          `json:"user_code"`
	Status           string                          `json:"status"`
	Stage            string                          `json:"stage"`
	Owner            string                          `json:"owner,omitempty"`
	Fence            uint64                          `json:"fence"`
	LeaseUntil       int64                           `json:"lease_until,omitempty"`
	ExpiresAt        int64                           `json:"expires_at"`
	NextAttemptAt    int64                           `json:"next_attempt_at,omitempty"`
	RetryCount       int                             `json:"retry_count,omitempty"`
	Error            string                          `json:"error,omitempty"`
	Result           *CodexDeviceAuthorizationResult `json:"result,omitempty"`
	ProtectedPayload string                          `json:"-"`
}

type codexDeviceAuthorizationStore interface {
	Create(context.Context, string, *CodexDeviceAuthorizationState, time.Duration) error
	Get(context.Context, string) (*CodexDeviceAuthorizationState, error)
	Claim(context.Context, string, string, time.Time, time.Time) (*CodexDeviceAuthorizationState, bool, error)
	Renew(context.Context, string, string, uint64, time.Time, time.Time) error
	Advance(context.Context, string, string, uint64, string, string, time.Time) (*CodexDeviceAuthorizationState, error)
	ReleasePending(context.Context, string, string, uint64, time.Time, bool) (*CodexDeviceAuthorizationState, error)
	Complete(context.Context, string, string, uint64, *CodexDeviceAuthorizationResult, string, time.Time) (*CodexDeviceAuthorizationState, error)
	MarkUncertain(context.Context, string, string, uint64, string, time.Time) (*CodexDeviceAuthorizationState, error)
	Cancel(context.Context, string, time.Time) (*CodexDeviceAuthorizationState, error)
	Delete(context.Context, string) error
}

type codexDeviceAuthorizationRunner struct {
	store        codexDeviceAuthorizationStore
	poll         func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error)
	exchange     func(context.Context, string, string, string) (*CodexOAuthTokenResult, error)
	now          func() time.Time
	waitInterval time.Duration
	leaseTTL     time.Duration
}

type codexDeviceAuthorizationMemoryStore struct {
	mu     sync.Mutex
	states map[string]*CodexDeviceAuthorizationState
	now    func() time.Time
}

type codexDeviceAuthorizationSQLStore struct {
	db  *gorm.DB
	now func() time.Time
}

var codexOAuthEphemeralCryptoSecret = common.CryptoSecret

type CodexUpstreamAuthError struct {
	Operation        string
	Status           int
	UpstreamError    string
	ErrorDescription string
}

func (e *CodexUpstreamAuthError) Error() string {
	if e == nil {
		return ""
	}
	operation := strings.TrimSpace(e.Operation)
	if operation == "" {
		operation = "codex upstream auth"
	}
	parts := []string{fmt.Sprintf("%s failed: status=%d", operation, e.Status)}
	if strings.TrimSpace(e.UpstreamError) != "" {
		parts = append(parts, "error="+strings.TrimSpace(e.UpstreamError))
	}
	return strings.Join(parts, ", ")
}

func RefreshCodexOAuthToken(ctx context.Context, refreshToken string) (*CodexOAuthTokenResult, error) {
	return RefreshCodexOAuthTokenWithProxy(ctx, refreshToken, "")
}

func RefreshCodexOAuthTokenWithProxy(ctx context.Context, refreshToken string, proxyURL string) (*CodexOAuthTokenResult, error) {
	client, err := getCodexOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return refreshCodexOAuthToken(ctx, client, codexOAuthTokenURL, codexOAuthClientID, refreshToken)
}

func ExchangeCodexAuthorizationCode(ctx context.Context, code string, verifier string) (*CodexOAuthTokenResult, error) {
	return ExchangeCodexAuthorizationCodeWithProxy(ctx, code, verifier, "")
}

func ExchangeCodexAuthorizationCodeWithProxy(ctx context.Context, code string, verifier string, proxyURL string) (*CodexOAuthTokenResult, error) {
	client, err := getCodexOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return exchangeCodexAuthorizationCode(ctx, client, codexOAuthTokenURL, codexOAuthClientID, code, verifier, codexOAuthRedirectURI)
}

func StartCodexDeviceAuthorization(ctx context.Context, proxyURL string) (*CodexDeviceAuthorizationStart, error) {
	client, err := getCodexOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return startCodexDeviceAuthorization(ctx, client, codexDeviceUserCodeURL, codexOAuthClientID)
}

func PollCodexDeviceAuthorization(ctx context.Context, deviceAuthID string, userCode string, proxyURL string) (*CodexDeviceAuthorizationPoll, error) {
	client, err := getCodexOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return pollCodexDeviceAuthorization(ctx, client, codexDeviceTokenURL, deviceAuthID, userCode)
}

func ExchangeCodexDeviceAuthorizationCode(ctx context.Context, code string, verifier string, proxyURL string) (*CodexOAuthTokenResult, error) {
	client, err := getCodexOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return exchangeCodexAuthorizationCode(ctx, client, codexOAuthTokenURL, codexOAuthClientID, code, verifier, codexDeviceRedirectURI)
}

func RegisterCodexDeviceAuthorization(ctx context.Context, userID int, channelID int, flow *CodexDeviceAuthorizationStart) error {
	if userID <= 0 || channelID <= 0 || flow == nil || strings.TrimSpace(flow.DeviceAuthID) == "" || strings.TrimSpace(flow.UserCode) == "" {
		return errors.New("invalid codex device authorization state")
	}
	now := time.Now()
	ttl := flow.ExpiresAt.Sub(now)
	if ttl <= 0 {
		return ErrCodexDeviceAuthorizationExpired
	}
	state := &CodexDeviceAuthorizationState{
		UserID:       userID,
		ChannelID:    channelID,
		DeviceAuthID: strings.TrimSpace(flow.DeviceAuthID),
		UserCode:     strings.TrimSpace(flow.UserCode),
		Status:       CodexDeviceAuthorizationPending,
		Stage:        CodexDeviceAuthorizationStagePending,
		ExpiresAt:    flow.ExpiresAt.UnixMilli(),
	}
	store, err := codexDeviceAuthorizationStateStore(ctx)
	if err != nil {
		return err
	}
	return store.Create(ctx, codexDeviceAuthorizationKey(userID, channelID, flow.DeviceAuthID), state, ttl)
}

func DeleteCodexDeviceAuthorization(ctx context.Context, userID int, channelID int, deviceAuthID string) error {
	if strings.TrimSpace(deviceAuthID) == "" {
		return nil
	}
	store, err := codexDeviceAuthorizationStateStore(ctx)
	if err != nil {
		return err
	}
	return store.Delete(ctx, codexDeviceAuthorizationKey(userID, channelID, deviceAuthID))
}

func CancelCodexDeviceAuthorization(ctx context.Context, userID int, channelID int, deviceAuthID string) (*CodexDeviceAuthorizationState, error) {
	if userID <= 0 || channelID <= 0 || strings.TrimSpace(deviceAuthID) == "" {
		return nil, errors.New("invalid codex device authorization identity")
	}
	store, err := codexDeviceAuthorizationStateStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.Cancel(ctx, codexDeviceAuthorizationKey(userID, channelID, deviceAuthID), time.Now())
}

func ContinueCodexDeviceAuthorization(
	ctx context.Context,
	userID int,
	channelID int,
	deviceAuthID string,
	proxyURL string,
	prepare func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error),
) (*CodexDeviceAuthorizationState, error) {
	store, err := codexDeviceAuthorizationStateStore(ctx)
	if err != nil {
		return nil, err
	}
	runner := codexDeviceAuthorizationRunner{
		store:        store,
		poll:         PollCodexDeviceAuthorization,
		exchange:     ExchangeCodexDeviceAuthorizationCode,
		now:          time.Now,
		waitInterval: 25 * time.Millisecond,
		leaseTTL:     codexDeviceLeaseTTL,
	}
	return runner.Run(ctx, userID, channelID, deviceAuthID, proxyURL, prepare)
}

func (runner codexDeviceAuthorizationRunner) Run(
	ctx context.Context,
	userID int,
	channelID int,
	deviceAuthID string,
	proxyURL string,
	prepare func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error),
) (*CodexDeviceAuthorizationState, error) {
	if runner.store == nil || runner.poll == nil || runner.exchange == nil || runner.now == nil || prepare == nil {
		return nil, errors.New("incomplete codex device authorization runner")
	}
	deviceAuthID = strings.TrimSpace(deviceAuthID)
	if userID <= 0 || channelID <= 0 || deviceAuthID == "" {
		return nil, errors.New("invalid codex device authorization identity")
	}
	owner, err := createStateHex(16)
	if err != nil {
		return nil, err
	}
	key := codexDeviceAuthorizationKey(userID, channelID, deviceAuthID)
	for {
		now := runner.now()
		state, claimed, err := runner.store.Claim(ctx, key, owner, now, now.Add(runner.leaseTTL))
		if err != nil {
			return nil, err
		}
		if !claimed {
			if state.Status == CodexDeviceAuthorizationCompleted || state.Status == CodexDeviceAuthorizationCancelled ||
				state.Status == CodexDeviceAuthorizationUncertain || state.Status == CodexDeviceAuthorizationPending {
				return state, nil
			}
			state, err = runner.waitForConclusion(ctx, key)
			if err != nil {
				return nil, err
			}
			if state.Status == CodexDeviceAuthorizationExchanging && state.LeaseUntil <= runner.now().UnixMilli() {
				continue
			}
			return state, nil
		}

		for {
			switch state.Stage {
			case "", CodexDeviceAuthorizationStagePending:
				var poll *CodexDeviceAuthorizationPoll
				err := runner.withRenewableLease(ctx, key, owner, state.Fence, func(callCtx context.Context) error {
					var callErr error
					poll, callErr = runner.poll(callCtx, state.DeviceAuthID, state.UserCode, proxyURL)
					return callErr
				})
				if err != nil {
					return runner.handleError(ctx, key, owner, state.Fence, err)
				}
				if poll == nil {
					return runner.handleError(ctx, key, owner, state.Fence, errors.New("device authorization poll returned no result"))
				}
				if poll.Pending {
					stateCtx, cancel := codexDeviceAuthorizationStateWriteContext(ctx)
					defer cancel()
					return runner.store.ReleasePending(stateCtx, key, owner, state.Fence, runner.now(), false)
				}
				payload, err := sealCodexOAuthPayload(&codexDeviceAuthorizationPollPayload{
					AuthorizationCode: poll.AuthorizationCode,
					CodeVerifier:      poll.CodeVerifier,
				})
				if err != nil {
					return runner.handleError(ctx, key, owner, state.Fence, err)
				}
				state, err = runner.advanceDetached(ctx, key, owner, state.Fence, CodexDeviceAuthorizationStagePolled, payload)
				if err != nil {
					return nil, err
				}
			case CodexDeviceAuthorizationStagePolled:
				var polled codexDeviceAuthorizationPollPayload
				if err := openCodexOAuthPayload(state.ProtectedPayload, &polled); err != nil {
					return runner.completeWithError(ctx, key, owner, state.Fence, err)
				}
				state, err = runner.advanceDetached(ctx, key, owner, state.Fence, CodexDeviceAuthorizationStageExchangeStarted, state.ProtectedPayload)
				if err != nil {
					return nil, err
				}
				var token *CodexOAuthTokenResult
				err := runner.withRenewableLease(ctx, key, owner, state.Fence, func(callCtx context.Context) error {
					var callErr error
					token, callErr = runner.exchange(callCtx, polled.AuthorizationCode, polled.CodeVerifier, proxyURL)
					return callErr
				})
				if err != nil {
					if errors.Is(err, ErrCodexDeviceAuthorizationCancelled) ||
						errors.Is(err, ErrCodexDeviceAuthorizationExpired) ||
						errors.Is(err, ErrCodexDeviceAuthorizationLeaseLost) {
						return nil, err
					}
					return runner.markUncertain(ctx, key, owner, state.Fence, err)
				}
				if token == nil {
					return runner.markUncertain(ctx, key, owner, state.Fence, errors.New("device authorization exchange returned no token result"))
				}
				payload, err := sealCodexOAuthPayload(token)
				if err != nil {
					return runner.markUncertain(ctx, key, owner, state.Fence, err)
				}
				// This recovery write is not atomic with the upstream exchange. A
				// crash before it commits leaves exchange_started, which is terminal
				// and therefore never replays the one-time authorization code.
				fence := state.Fence
				state, err = runner.advanceDetached(ctx, key, owner, fence, CodexDeviceAuthorizationStageExchanged, payload)
				if err != nil {
					if errors.Is(err, ErrCodexDeviceAuthorizationCancelled) ||
						errors.Is(err, ErrCodexDeviceAuthorizationExpired) ||
						errors.Is(err, ErrCodexDeviceAuthorizationLeaseLost) {
						return nil, err
					}
					_, uncertainErr := runner.markUncertain(ctx, key, owner, fence, err)
					return nil, errors.Join(fmt.Errorf("codex exchange succeeded but durable recovery write failed: %w", err), uncertainErr)
				}
			case CodexDeviceAuthorizationStageExchangeStarted:
				return runner.markUncertain(ctx, key, owner, state.Fence, errors.New("device authorization exchange outcome is uncertain; regenerate the credential"))
			case CodexDeviceAuthorizationStageExchanged:
				var token CodexOAuthTokenResult
				if err := openCodexOAuthPayload(state.ProtectedPayload, &token); err != nil {
					return runner.completeWithError(ctx, key, owner, state.Fence, err)
				}
				result, encoded, err := prepare(&token)
				if err != nil {
					return runner.completeWithError(ctx, key, owner, state.Fence, err)
				}
				sqlStore, ok := runner.store.(*codexDeviceAuthorizationSQLStore)
				if !ok {
					stateCtx, cancel := codexDeviceAuthorizationStateWriteContext(ctx)
					defer cancel()
					return runner.store.Complete(stateCtx, key, owner, state.Fence, result, "", runner.now())
				}
				state, err = sqlStore.commitCredential(ctx, key, owner, state.Fence, encoded, result, runner.now())
				if err != nil {
					// A concurrent cancel, expiry, or takeover already fenced this
					// owner. Do not attempt a second state transition with a lost lease.
					if errors.Is(err, ErrCodexDeviceAuthorizationCancelled) ||
						errors.Is(err, ErrCodexDeviceAuthorizationExpired) ||
						errors.Is(err, ErrCodexDeviceAuthorizationLeaseLost) {
						return nil, err
					}
					return runner.handleError(ctx, key, owner, state.Fence, err)
				}
				model.InitChannelCache()
				ResetProxyClientCache()
				return state, nil
			case CodexDeviceAuthorizationStageSaved:
				return state, nil
			default:
				return runner.completeWithError(ctx, key, owner, state.Fence, errors.New("invalid codex device authorization stage"))
			}
		}
	}
}

func (runner codexDeviceAuthorizationRunner) waitForConclusion(ctx context.Context, key string) (*CodexDeviceAuthorizationState, error) {
	interval := runner.waitInterval
	if interval <= 0 {
		interval = 25 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			state, err := runner.store.Get(ctx, key)
			if err != nil {
				return nil, err
			}
			if state.Status != CodexDeviceAuthorizationExchanging || state.LeaseUntil <= runner.now().UnixMilli() {
				return state, nil
			}
		}
	}
}

func (runner codexDeviceAuthorizationRunner) completeWithError(ctx context.Context, key string, owner string, fence uint64, cause error) (*CodexDeviceAuthorizationState, error) {
	message := "codex device authorization failed"
	if cause != nil {
		message = common.MaskSensitiveInfo(cause.Error())
	}
	stateCtx, cancel := codexDeviceAuthorizationStateWriteContext(ctx)
	defer cancel()
	return runner.store.Complete(stateCtx, key, owner, fence, nil, message, runner.now())
}

func (runner codexDeviceAuthorizationRunner) markUncertain(ctx context.Context, key string, owner string, fence uint64, cause error) (*CodexDeviceAuthorizationState, error) {
	message := "device authorization exchange outcome is uncertain; regenerate the credential"
	if cause != nil {
		message += ": " + common.MaskSensitiveInfo(cause.Error())
	}
	stateCtx, cancel := codexDeviceAuthorizationStateWriteContext(ctx)
	defer cancel()
	return runner.store.MarkUncertain(stateCtx, key, owner, fence, message, runner.now())
}

func codexDeviceAuthorizationStateWriteContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
}

func codexDeviceAuthorizationStateStore(ctx context.Context) (codexDeviceAuthorizationStore, error) {
	if model.DB == nil {
		return nil, ErrCodexOAuthStoreUnavailable
	}
	if strings.TrimSpace(common.CryptoSecret) == "" || common.CryptoSecret == codexOAuthEphemeralCryptoSecret {
		return nil, fmt.Errorf("%w: configure stable CRYPTO_SECRET or SESSION_SECRET", ErrCodexOAuthStoreUnavailable)
	}
	store := &codexDeviceAuthorizationSQLStore{db: model.DB, now: time.Now}
	if !store.db.WithContext(ctx).Migrator().HasTable(&model.CodexOAuthOperation{}) {
		return nil, fmt.Errorf("%w: codex_oauth_operations migration is missing", ErrCodexOAuthStoreUnavailable)
	}
	return store, nil
}

func EnsureCodexOAuthOperationStore(ctx context.Context) error {
	_, err := codexDeviceAuthorizationStateStore(ctx)
	return err
}

func codexDeviceAuthorizationKey(userID int, channelID int, deviceAuthID string) string {
	return fmt.Sprintf("codex:device-auth:v1:%d:%d:%s", userID, channelID, strings.TrimSpace(deviceAuthID))
}

func newCodexDeviceAuthorizationMemoryStore(now func() time.Time) *codexDeviceAuthorizationMemoryStore {
	if now == nil {
		now = time.Now
	}
	return &codexDeviceAuthorizationMemoryStore{states: make(map[string]*CodexDeviceAuthorizationState), now: now}
}

type codexDeviceAuthorizationPollPayload struct {
	AuthorizationCode string
	CodeVerifier      string
}

type codexDeviceAuthorizationPendingPayload struct {
	UserCode string
}

func cloneCodexDeviceAuthorizationState(state *CodexDeviceAuthorizationState) *CodexDeviceAuthorizationState {
	if state == nil {
		return nil
	}
	cloned := *state
	if state.Result != nil {
		result := *state.Result
		cloned.Result = &result
	}
	return &cloned
}

func (store *codexDeviceAuthorizationMemoryStore) Create(_ context.Context, key string, state *CodexDeviceAuthorizationState, _ time.Duration) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing := store.states[key]; existing != nil && existing.ExpiresAt > store.now().UnixMilli() {
		if existing.Status == CodexDeviceAuthorizationPending || existing.Status == CodexDeviceAuthorizationExchanging {
			return ErrCodexDeviceAuthorizationExists
		}
	}
	store.states[key] = cloneCodexDeviceAuthorizationState(state)
	return nil
}

func (store *codexDeviceAuthorizationMemoryStore) Get(_ context.Context, key string) (*CodexDeviceAuthorizationState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if state.ExpiresAt <= store.now().UnixMilli() {
		delete(store.states, key)
		return nil, ErrCodexDeviceAuthorizationExpired
	}
	return cloneCodexDeviceAuthorizationState(state), nil
}

func (store *codexDeviceAuthorizationMemoryStore) Claim(_ context.Context, key string, owner string, now time.Time, leaseUntil time.Time) (*CodexDeviceAuthorizationState, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, false, ErrCodexDeviceAuthorizationNotFound
	}
	if state.Status == CodexDeviceAuthorizationExchanging && state.Stage == CodexDeviceAuthorizationStageExchangeStarted && state.LeaseUntil <= now.UnixMilli() {
		state.Status = CodexDeviceAuthorizationUncertain
		state.Owner = ""
		state.LeaseUntil = 0
		state.UserCode = ""
		state.ProtectedPayload = ""
		state.Error = "device authorization exchange outcome is uncertain; regenerate the credential"
		return cloneCodexDeviceAuthorizationState(state), false, nil
	}
	if state.ExpiresAt <= now.UnixMilli() {
		delete(store.states, key)
		return nil, false, ErrCodexDeviceAuthorizationExpired
	}
	claimable := state.NextAttemptAt <= now.UnixMilli() && (state.Status == CodexDeviceAuthorizationPending ||
		(state.Status == CodexDeviceAuthorizationExchanging && state.LeaseUntil <= now.UnixMilli()))
	if claimable {
		state.Status = CodexDeviceAuthorizationExchanging
		state.Owner = owner
		state.Fence++
		state.LeaseUntil = leaseUntil.UnixMilli()
	}
	return cloneCodexDeviceAuthorizationState(state), claimable, nil
}

func (store *codexDeviceAuthorizationMemoryStore) Renew(_ context.Context, key string, owner string, fence uint64, now time.Time, leaseUntil time.Time) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return ErrCodexDeviceAuthorizationNotFound
	}
	if state.ExpiresAt <= now.UnixMilli() {
		return ErrCodexDeviceAuthorizationExpired
	}
	if state.Status != CodexDeviceAuthorizationExchanging || state.Owner != owner || state.Fence != fence {
		return ErrCodexDeviceAuthorizationLeaseLost
	}
	state.LeaseUntil = leaseUntil.UnixMilli()
	return nil
}

func (store *codexDeviceAuthorizationMemoryStore) Advance(_ context.Context, key string, owner string, fence uint64, stage string, payload string, now time.Time) (*CodexDeviceAuthorizationState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if state.ExpiresAt <= now.UnixMilli() {
		return nil, ErrCodexDeviceAuthorizationExpired
	}
	if state.Status != CodexDeviceAuthorizationExchanging || state.Owner != owner || state.Fence != fence {
		return cloneCodexDeviceAuthorizationState(state), ErrCodexDeviceAuthorizationLeaseLost
	}
	state.Stage = stage
	state.ProtectedPayload = payload
	state.RetryCount = 0
	state.NextAttemptAt = 0
	return cloneCodexDeviceAuthorizationState(state), nil
}

func (store *codexDeviceAuthorizationMemoryStore) ReleasePending(_ context.Context, key string, owner string, fence uint64, now time.Time, backoff bool) (*CodexDeviceAuthorizationState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if state.ExpiresAt <= now.UnixMilli() {
		return nil, ErrCodexDeviceAuthorizationExpired
	}
	if state.Status == CodexDeviceAuthorizationExchanging && state.Owner == owner && state.Fence == fence {
		state.Status = CodexDeviceAuthorizationPending
		state.Owner = ""
		state.LeaseUntil = 0
		if backoff {
			state.RetryCount++
			state.NextAttemptAt = now.Add(codexDeviceRetryBackoff(state.RetryCount)).UnixMilli()
		} else {
			state.RetryCount = 0
			state.NextAttemptAt = 0
		}
	}
	return cloneCodexDeviceAuthorizationState(state), nil
}

func (store *codexDeviceAuthorizationMemoryStore) Complete(_ context.Context, key string, owner string, fence uint64, result *CodexDeviceAuthorizationResult, message string, now time.Time) (*CodexDeviceAuthorizationState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if state.ExpiresAt <= now.UnixMilli() {
		return nil, ErrCodexDeviceAuthorizationExpired
	}
	if state.Status == CodexDeviceAuthorizationCancelled {
		return cloneCodexDeviceAuthorizationState(state), ErrCodexDeviceAuthorizationCancelled
	}
	if state.Status == CodexDeviceAuthorizationCompleted {
		return cloneCodexDeviceAuthorizationState(state), nil
	}
	if state.Status != CodexDeviceAuthorizationExchanging || state.Owner != owner || state.Fence != fence {
		return cloneCodexDeviceAuthorizationState(state), ErrCodexDeviceAuthorizationLeaseLost
	}
	state.Status = CodexDeviceAuthorizationCompleted
	if result != nil {
		state.Stage = CodexDeviceAuthorizationStageSaved
	}
	state.Owner = ""
	state.LeaseUntil = 0
	state.UserCode = ""
	state.ProtectedPayload = ""
	state.Error = strings.TrimSpace(message)
	state.Result = result
	return cloneCodexDeviceAuthorizationState(state), nil
}

func (store *codexDeviceAuthorizationMemoryStore) MarkUncertain(_ context.Context, key string, owner string, fence uint64, message string, _ time.Time) (*CodexDeviceAuthorizationState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if state.Status != CodexDeviceAuthorizationExchanging || state.Owner != owner || state.Fence != fence {
		return cloneCodexDeviceAuthorizationState(state), ErrCodexDeviceAuthorizationLeaseLost
	}
	state.Status = CodexDeviceAuthorizationUncertain
	state.Owner = ""
	state.LeaseUntil = 0
	state.UserCode = ""
	state.ProtectedPayload = ""
	state.Error = strings.TrimSpace(message)
	return cloneCodexDeviceAuthorizationState(state), nil
}

func (store *codexDeviceAuthorizationMemoryStore) Cancel(_ context.Context, key string, now time.Time) (*CodexDeviceAuthorizationState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state := store.states[key]
	if state == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if state.ExpiresAt <= now.UnixMilli() {
		return nil, ErrCodexDeviceAuthorizationExpired
	}
	if state.Status != CodexDeviceAuthorizationCompleted && state.Status != CodexDeviceAuthorizationUncertain {
		state.Status = CodexDeviceAuthorizationCancelled
		state.Owner = ""
		state.Fence++
		state.LeaseUntil = 0
		state.UserCode = ""
		state.ProtectedPayload = ""
	}
	return cloneCodexDeviceAuthorizationState(state), nil
}

func (store *codexDeviceAuthorizationMemoryStore) Delete(_ context.Context, key string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.states, key)
	return nil
}

func (store *codexDeviceAuthorizationSQLStore) Create(ctx context.Context, key string, state *CodexDeviceAuthorizationState, _ time.Duration) error {
	protected, err := sealCodexOAuthPayload(&codexDeviceAuthorizationPendingPayload{UserCode: state.UserCode})
	if err != nil {
		return err
	}
	if err := store.db.WithContext(ctx).Where("kind = ? AND expires_at <= ?", "device", store.now().UnixMilli()).
		Delete(&model.CodexOAuthOperation{}).Error; err != nil {
		return err
	}
	record := model.CodexOAuthOperation{
		OperationKey: key, Kind: "device", UserID: state.UserID, ChannelID: state.ChannelID,
		DeviceAuthID: state.DeviceAuthID, Status: state.Status, Stage: state.Stage,
		ExpiresAt: state.ExpiresAt, ProtectedPayload: protected,
	}
	err = store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.CodexOAuthOperation
		err := tx.Where("operation_key = ?", key).First(&existing).Error
		if err == nil {
			if existing.ExpiresAt > store.now().UnixMilli() &&
				(existing.Status == CodexDeviceAuthorizationPending || existing.Status == CodexDeviceAuthorizationExchanging) {
				return ErrCodexDeviceAuthorizationExists
			}
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&record).Error
	})
	return err
}

func (store *codexDeviceAuthorizationSQLStore) Get(ctx context.Context, key string) (*CodexDeviceAuthorizationState, error) {
	var record model.CodexOAuthOperation
	if err := store.db.WithContext(ctx).Where("operation_key = ?", key).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCodexDeviceAuthorizationNotFound
		}
		return nil, err
	}
	return codexOAuthRecordState(&record, store.now())
}

func (store *codexDeviceAuthorizationSQLStore) Claim(ctx context.Context, key string, owner string, now time.Time, leaseUntil time.Time) (*CodexDeviceAuthorizationState, bool, error) {
	var record model.CodexOAuthOperation
	claimed := false
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockCodexOAuthOperation(tx, key, &record); err != nil {
			return err
		}
		if record.Status == CodexDeviceAuthorizationExchanging && record.Stage == CodexDeviceAuthorizationStageExchangeStarted && record.LeaseUntil <= now.UnixMilli() {
			record.Status = CodexDeviceAuthorizationUncertain
			record.Owner = ""
			record.LeaseUntil = 0
			record.UserCode = ""
			record.ProtectedPayload = ""
			record.ErrorMessage = "device authorization exchange outcome is uncertain; regenerate the credential"
			return tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", key).Updates(map[string]any{
				"status": record.Status, "owner": "", "lease_until": 0, "user_code": "",
				"protected_payload": "", "error_message": record.ErrorMessage,
			}).Error
		}
		if record.ExpiresAt <= now.UnixMilli() {
			return ErrCodexDeviceAuthorizationExpired
		}
		claimable := record.NextAttemptAt <= now.UnixMilli() && (record.Status == CodexDeviceAuthorizationPending ||
			(record.Status == CodexDeviceAuthorizationExchanging && record.LeaseUntil <= now.UnixMilli()))
		if !claimable {
			return nil
		}
		record.Status = CodexDeviceAuthorizationExchanging
		record.Owner = owner
		record.Fence++
		record.LeaseUntil = leaseUntil.UnixMilli()
		claimed = true
		return tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", key).Updates(map[string]any{
			"status": record.Status, "owner": record.Owner, "fence": record.Fence,
			"lease_until": record.LeaseUntil,
		}).Error
	})
	if err != nil {
		return nil, false, err
	}
	state, err := codexOAuthRecordState(&record, now)
	return state, claimed, err
}

func (store *codexDeviceAuthorizationSQLStore) Renew(ctx context.Context, key string, owner string, fence uint64, now time.Time, leaseUntil time.Time) error {
	result := store.db.WithContext(ctx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND status = ? AND owner = ? AND fence = ? AND expires_at > ?",
			key, CodexDeviceAuthorizationExchanging, owner, fence, now.UnixMilli()).
		Update("lease_until", leaseUntil.UnixMilli())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCodexDeviceAuthorizationLeaseLost
	}
	return nil
}

func (store *codexDeviceAuthorizationSQLStore) Advance(ctx context.Context, key string, owner string, fence uint64, stage string, payload string, now time.Time) (*CodexDeviceAuthorizationState, error) {
	result := store.db.WithContext(ctx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND status = ? AND owner = ? AND fence = ? AND expires_at > ?",
			key, CodexDeviceAuthorizationExchanging, owner, fence, now.UnixMilli()).
		Updates(map[string]any{
			"stage": stage, "protected_payload": payload, "retry_count": 0, "next_attempt_at": 0,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrCodexDeviceAuthorizationLeaseLost
	}
	return store.Get(ctx, key)
}

func (store *codexDeviceAuthorizationSQLStore) ReleasePending(ctx context.Context, key string, owner string, fence uint64, now time.Time, backoff bool) (*CodexDeviceAuthorizationState, error) {
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record model.CodexOAuthOperation
		if err := lockCodexOAuthOperation(tx, key, &record); err != nil {
			return err
		}
		if record.Status != CodexDeviceAuthorizationExchanging || record.Owner != owner ||
			record.Fence != fence || record.ExpiresAt <= now.UnixMilli() {
			return ErrCodexDeviceAuthorizationLeaseLost
		}
		if backoff {
			record.RetryCount++
			record.NextAttemptAt = now.Add(codexDeviceRetryBackoff(record.RetryCount)).UnixMilli()
		} else {
			record.RetryCount = 0
			record.NextAttemptAt = 0
		}
		return tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", key).Updates(map[string]any{
			"status": CodexDeviceAuthorizationPending, "owner": "", "lease_until": 0,
			"retry_count": record.RetryCount, "next_attempt_at": record.NextAttemptAt,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return store.Get(ctx, key)
}

func (store *codexDeviceAuthorizationSQLStore) Complete(ctx context.Context, key string, owner string, fence uint64, result *CodexDeviceAuthorizationResult, message string, now time.Time) (*CodexDeviceAuthorizationState, error) {
	resultJSON := ""
	if result != nil {
		encoded, err := common.Marshal(result)
		if err != nil {
			return nil, err
		}
		resultJSON = string(encoded)
	}
	updates := map[string]any{
		"status": CodexDeviceAuthorizationCompleted, "owner": "", "lease_until": 0,
		"user_code": "", "protected_payload": "", "result": resultJSON,
		"error_message": strings.TrimSpace(message),
	}
	if result != nil {
		updates["stage"] = CodexDeviceAuthorizationStageSaved
	}
	dbResult := store.db.WithContext(ctx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND status = ? AND owner = ? AND fence = ? AND expires_at > ?",
			key, CodexDeviceAuthorizationExchanging, owner, fence, now.UnixMilli()).
		Updates(updates)
	if dbResult.Error != nil {
		return nil, dbResult.Error
	}
	if dbResult.RowsAffected != 1 {
		return nil, ErrCodexDeviceAuthorizationLeaseLost
	}
	return store.Get(ctx, key)
}

func (store *codexDeviceAuthorizationSQLStore) MarkUncertain(ctx context.Context, key string, owner string, fence uint64, message string, _ time.Time) (*CodexDeviceAuthorizationState, error) {
	result := store.db.WithContext(ctx).Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ? AND status = ? AND owner = ? AND fence = ?",
			key, CodexDeviceAuthorizationExchanging, owner, fence).
		Updates(map[string]any{
			"status": CodexDeviceAuthorizationUncertain, "owner": "", "lease_until": 0,
			"user_code": "", "protected_payload": "", "error_message": strings.TrimSpace(message),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrCodexDeviceAuthorizationLeaseLost
	}
	return store.Get(ctx, key)
}

func (store *codexDeviceAuthorizationSQLStore) Cancel(ctx context.Context, key string, now time.Time) (*CodexDeviceAuthorizationState, error) {
	var record model.CodexOAuthOperation
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockCodexOAuthOperation(tx, key, &record); err != nil {
			return err
		}
		if record.ExpiresAt <= now.UnixMilli() {
			return ErrCodexDeviceAuthorizationExpired
		}
		if record.Status == CodexDeviceAuthorizationCompleted || record.Status == CodexDeviceAuthorizationUncertain {
			return nil
		}
		record.Status = CodexDeviceAuthorizationCancelled
		record.Fence++
		record.Owner = ""
		record.LeaseUntil = 0
		record.UserCode = ""
		record.ProtectedPayload = ""
		return tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", key).Updates(map[string]any{
			"status": record.Status, "fence": record.Fence, "owner": "", "lease_until": 0,
			"user_code": "", "protected_payload": "",
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return codexOAuthRecordState(&record, now)
}

func (store *codexDeviceAuthorizationSQLStore) Delete(ctx context.Context, key string) error {
	return store.db.WithContext(ctx).Where("operation_key = ?", key).Delete(&model.CodexOAuthOperation{}).Error
}

func lockCodexOAuthOperation(tx *gorm.DB, key string, record *model.CodexOAuthOperation) error {
	result := tx.Model(&model.CodexOAuthOperation{}).Where("operation_key = ?", key).
		UpdateColumn("fence", gorm.Expr("fence"))
	if result.Error != nil {
		return result.Error
	}
	err := tx.Where("operation_key = ?", key).First(record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCodexDeviceAuthorizationNotFound
	}
	return err
}

func codexOAuthRecordState(record *model.CodexOAuthOperation, now time.Time) (*CodexDeviceAuthorizationState, error) {
	if record == nil {
		return nil, ErrCodexDeviceAuthorizationNotFound
	}
	if record.ExpiresAt <= now.UnixMilli() && record.Status != CodexDeviceAuthorizationUncertain {
		return nil, ErrCodexDeviceAuthorizationExpired
	}
	state := &CodexDeviceAuthorizationState{
		UserID: record.UserID, ChannelID: record.ChannelID, DeviceAuthID: record.DeviceAuthID,
		UserCode: record.UserCode, Status: record.Status, Stage: record.Stage, Owner: record.Owner,
		Fence: record.Fence, LeaseUntil: record.LeaseUntil, ExpiresAt: record.ExpiresAt,
		NextAttemptAt: record.NextAttemptAt, Error: record.ErrorMessage,
		RetryCount:       record.RetryCount,
		ProtectedPayload: record.ProtectedPayload,
	}
	if record.Stage == CodexDeviceAuthorizationStagePending && state.UserCode == "" && record.ProtectedPayload != "" {
		var pending codexDeviceAuthorizationPendingPayload
		if err := openCodexOAuthPayload(record.ProtectedPayload, &pending); err != nil {
			return nil, err
		}
		state.UserCode = pending.UserCode
	}
	if strings.TrimSpace(record.Result) != "" {
		var result CodexDeviceAuthorizationResult
		if err := common.Unmarshal([]byte(record.Result), &result); err != nil {
			return nil, err
		}
		state.Result = &result
	}
	return state, nil
}

func (runner codexDeviceAuthorizationRunner) withRenewableLease(
	ctx context.Context,
	key string,
	owner string,
	fence uint64,
	call func(context.Context) error,
) error {
	leaseTTL := runner.leaseTTL
	if leaseTTL <= 0 {
		leaseTTL = codexDeviceLeaseTTL
	}
	callCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	renewed := make(chan error, 1)
	go func() {
		interval := leaseTTL / 3
		if interval <= 0 {
			interval = time.Millisecond
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-callCtx.Done():
				renewed <- nil
				return
			case <-ticker.C:
				now := runner.now()
				if err := runner.store.Renew(callCtx, key, owner, fence, now, now.Add(leaseTTL)); err != nil {
					cancel()
					renewed <- err
					return
				}
			}
		}
	}()
	callErr := call(callCtx)
	cancel()
	renewErr := <-renewed
	if callErr != nil {
		return callErr
	}
	return renewErr
}

func (runner codexDeviceAuthorizationRunner) advanceDetached(
	ctx context.Context,
	key string,
	owner string,
	fence uint64,
	stage string,
	payload string,
) (*CodexDeviceAuthorizationState, error) {
	stateCtx, cancel := codexDeviceAuthorizationStateWriteContext(ctx)
	defer cancel()
	return runner.store.Advance(stateCtx, key, owner, fence, stage, payload, runner.now())
}

func (runner codexDeviceAuthorizationRunner) handleError(
	ctx context.Context,
	key string,
	owner string,
	fence uint64,
	cause error,
) (*CodexDeviceAuthorizationState, error) {
	if isCodexDeviceAuthorizationTerminalError(cause) {
		return runner.completeWithError(ctx, key, owner, fence, cause)
	}
	stateCtx, cancel := codexDeviceAuthorizationStateWriteContext(ctx)
	defer cancel()
	state, err := runner.store.ReleasePending(stateCtx, key, owner, fence, runner.now(), true)
	if err != nil {
		return nil, errors.Join(cause, err)
	}
	return state, nil
}

func codexDeviceRetryBackoff(retryCount int) time.Duration {
	if retryCount < 1 {
		retryCount = 1
	}
	if retryCount > 6 {
		retryCount = 6
	}
	backoff := time.Second * time.Duration(1<<(retryCount-1))
	if backoff > 30*time.Second {
		return 30 * time.Second
	}
	return backoff
}

func isCodexDeviceAuthorizationTerminalError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrCodexDeviceAuthorizationExpired) ||
		errors.Is(err, ErrCodexDeviceAuthorizationCancelled) {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var upstream *CodexUpstreamAuthError
	if errors.As(err, &upstream) {
		if upstream.Status == http.StatusTooManyRequests || upstream.Status >= http.StatusInternalServerError {
			return false
		}
		return upstream.Status >= http.StatusBadRequest && upstream.Status < http.StatusInternalServerError
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "invalid_grant") ||
		strings.Contains(text, "access_denied") ||
		strings.Contains(text, "authorization denied") ||
		strings.Contains(text, "token response missing fields") ||
		strings.Contains(text, "invalid codex device authorization stage") ||
		strings.Contains(text, "decrypt oauth operation payload")
}

func sealCodexOAuthPayload(value any) (string, error) {
	if strings.TrimSpace(common.CryptoSecret) == "" {
		return "", errors.New("CRYPTO_SECRET is required for durable OAuth recovery")
	}
	plain, err := common.Marshal(value)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(common.CryptoSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, plain, []byte("codex-oauth-operation-v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func openCodexOAuthPayload(encoded string, target any) error {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return errors.New("decrypt oauth operation payload: invalid encoding")
	}
	key := sha256.Sum256([]byte(common.CryptoSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return errors.New("decrypt oauth operation payload: cipher unavailable")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(raw) < aead.NonceSize() {
		return errors.New("decrypt oauth operation payload: invalid ciphertext")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("codex-oauth-operation-v1"))
	if err != nil {
		return errors.New("decrypt oauth operation payload: authentication failed")
	}
	if err := common.Unmarshal(plain, target); err != nil {
		return errors.New("decrypt oauth operation payload: invalid plaintext")
	}
	return nil
}

func (store *codexDeviceAuthorizationSQLStore) commitCredential(
	ctx context.Context,
	key string,
	owner string,
	fence uint64,
	encodedCredential string,
	result *CodexDeviceAuthorizationResult,
	_ time.Time,
) (*CodexDeviceAuthorizationState, error) {
	commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), codexOAuthStoreTimeout)
	defer cancel()
	var committed model.CodexOAuthOperation
	var commitNow time.Time
	err := store.db.WithContext(commitCtx).Transaction(func(tx *gorm.DB) error {
		if err := lockCodexOAuthOperation(tx, key, &committed); err != nil {
			return err
		}
		ch, err := lockCodexChannel(tx, committed.ChannelID)
		if err != nil {
			return err
		}
		commitNow = store.now()
		if committed.ExpiresAt <= commitNow.UnixMilli() {
			return ErrCodexDeviceAuthorizationExpired
		}
		if committed.Status == CodexDeviceAuthorizationCancelled {
			return ErrCodexDeviceAuthorizationCancelled
		}
		if committed.Status != CodexDeviceAuthorizationExchanging ||
			committed.Stage != CodexDeviceAuthorizationStageExchanged ||
			committed.Owner != owner || committed.Fence != fence ||
			committed.LeaseUntil <= commitNow.UnixMilli() {
			return ErrCodexDeviceAuthorizationLeaseLost
		}
		if ch.Type != constant.ChannelTypeCodex {
			return errors.New("channel type is not Codex")
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
			"key": encodedCredential, "setting": *ch.Setting,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("operation_key = ?", codexCredentialRefreshOperationKey(ch.Id)).
			Delete(&model.CodexOAuthOperation{}).Error; err != nil {
			return err
		}
		resultJSON, err := common.Marshal(result)
		if err != nil {
			return err
		}
		committed.Status = CodexDeviceAuthorizationCompleted
		committed.Stage = CodexDeviceAuthorizationStageSaved
		committed.Owner = ""
		committed.LeaseUntil = 0
		committed.UserCode = ""
		committed.ProtectedPayload = ""
		committed.Result = string(resultJSON)
		update := tx.Model(&model.CodexOAuthOperation{}).
			Where("operation_key = ? AND status = ? AND stage = ? AND owner = ? AND fence = ? AND expires_at > ? AND lease_until > ?",
				key, CodexDeviceAuthorizationExchanging, CodexDeviceAuthorizationStageExchanged,
				owner, fence, commitNow.UnixMilli(), commitNow.UnixMilli()).
			Updates(map[string]any{
				"status": committed.Status, "stage": committed.Stage, "owner": "",
				"lease_until": 0, "user_code": "", "protected_payload": "",
				"result": committed.Result, "error_message": "",
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrCodexDeviceAuthorizationLeaseLost
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return codexOAuthRecordState(&committed, commitNow)
}

func CreateCodexOAuthAuthorizationFlow() (*CodexOAuthAuthorizationFlow, error) {
	state, err := createStateHex(16)
	if err != nil {
		return nil, err
	}
	verifier, challenge, err := generatePKCEPair()
	if err != nil {
		return nil, err
	}
	u, err := buildCodexAuthorizeURL(state, challenge)
	if err != nil {
		return nil, err
	}
	return &CodexOAuthAuthorizationFlow{
		State:        state,
		Verifier:     verifier,
		Challenge:    challenge,
		AuthorizeURL: u,
	}, nil
}

func startCodexDeviceAuthorization(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	clientID string,
) (*CodexDeviceAuthorizationStart, error) {
	body, err := common.Marshal(map[string]string{"client_id": clientID})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newCodexUpstreamAuthError("codex device authorization start", resp.StatusCode, responseBody)
	}
	var payload struct {
		DeviceAuthID string `json:"device_auth_id"`
		UserCode     string `json:"user_code"`
		UserCodeAlt  string `json:"usercode"`
		Interval     string `json:"interval"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return nil, err
	}
	userCode := strings.TrimSpace(payload.UserCode)
	if userCode == "" {
		userCode = strings.TrimSpace(payload.UserCodeAlt)
	}
	intervalSeconds, err := strconv.Atoi(strings.TrimSpace(payload.Interval))
	if err != nil || intervalSeconds <= 0 {
		intervalSeconds = 5
	}
	if strings.TrimSpace(payload.DeviceAuthID) == "" || userCode == "" {
		return nil, errors.New("codex device authorization response missing fields")
	}
	return &CodexDeviceAuthorizationStart{
		DeviceAuthID:    strings.TrimSpace(payload.DeviceAuthID),
		UserCode:        userCode,
		VerificationURL: codexDeviceVerifyURL,
		Interval:        time.Duration(intervalSeconds) * time.Second,
		ExpiresAt:       time.Now().Add(codexDeviceFlowTTL),
	}, nil
}

func pollCodexDeviceAuthorization(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	deviceAuthID string,
	userCode string,
) (*CodexDeviceAuthorizationPoll, error) {
	payloadBody, err := common.Marshal(map[string]string{
		"device_auth_id": strings.TrimSpace(deviceAuthID),
		"user_code":      strings.TrimSpace(userCode),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payloadBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return &CodexDeviceAuthorizationPoll{Pending: true}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newCodexUpstreamAuthError("codex device authorization poll", resp.StatusCode, responseBody)
	}
	var payload struct {
		AuthorizationCode string `json:"authorization_code"`
		CodeVerifier      string `json:"code_verifier"`
	}
	if err := common.Unmarshal(responseBody, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.AuthorizationCode) == "" || strings.TrimSpace(payload.CodeVerifier) == "" {
		return nil, errors.New("codex device authorization token response missing fields")
	}
	return &CodexDeviceAuthorizationPoll{
		AuthorizationCode: strings.TrimSpace(payload.AuthorizationCode),
		CodeVerifier:      strings.TrimSpace(payload.CodeVerifier),
	}, nil
}

func refreshCodexOAuthToken(
	ctx context.Context,
	client *http.Client,
	tokenURL string,
	clientID string,
	refreshToken string,
) (*CodexOAuthTokenResult, error) {
	rt := strings.TrimSpace(refreshToken)
	if rt == "" {
		return nil, errors.New("empty refresh_token")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", rt)
	form.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	decodeErr := common.Unmarshal(body, &payload)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newCodexUpstreamAuthError("codex oauth refresh", resp.StatusCode, body)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}

	if strings.TrimSpace(payload.AccessToken) == "" || strings.TrimSpace(payload.RefreshToken) == "" || payload.ExpiresIn <= 0 {
		return nil, errors.New("codex oauth refresh response missing fields")
	}

	return &CodexOAuthTokenResult{
		AccessToken:  strings.TrimSpace(payload.AccessToken),
		RefreshToken: strings.TrimSpace(payload.RefreshToken),
		ExpiresAt:    time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func exchangeCodexAuthorizationCode(
	ctx context.Context,
	client *http.Client,
	tokenURL string,
	clientID string,
	code string,
	verifier string,
	redirectURI string,
) (*CodexOAuthTokenResult, error) {
	c := strings.TrimSpace(code)
	v := strings.TrimSpace(verifier)
	if c == "" {
		return nil, errors.New("empty authorization code")
	}
	if v == "" {
		return nil, errors.New("empty code_verifier")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", c)
	form.Set("code_verifier", v)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	decodeErr := common.Unmarshal(body, &payload)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newCodexUpstreamAuthError("codex oauth code exchange", resp.StatusCode, body)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	if strings.TrimSpace(payload.AccessToken) == "" || strings.TrimSpace(payload.RefreshToken) == "" || payload.ExpiresIn <= 0 {
		return nil, errors.New("codex oauth token response missing fields")
	}
	return &CodexOAuthTokenResult{
		AccessToken:  strings.TrimSpace(payload.AccessToken),
		RefreshToken: strings.TrimSpace(payload.RefreshToken),
		ExpiresAt:    time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func getCodexOAuthHTTPClient(proxyURL string) (*http.Client, error) {
	baseClient, err := GetHttpClientWithProxy(strings.TrimSpace(proxyURL))
	if err != nil {
		return nil, err
	}
	if baseClient == nil {
		return &http.Client{Timeout: defaultHTTPTimeout}, nil
	}
	clientCopy := *baseClient
	clientCopy.Timeout = defaultHTTPTimeout
	return &clientCopy, nil
}

func buildCodexAuthorizeURL(state string, challenge string) (string, error) {
	u, err := url.Parse(codexOAuthAuthorizeURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", codexOAuthClientID)
	q.Set("redirect_uri", codexOAuthRedirectURI)
	q.Set("scope", codexOAuthScope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("originator", "codex_cli_rs")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func createStateHex(nBytes int) (string, error) {
	if nBytes <= 0 {
		return "", errors.New("invalid state bytes length")
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func generatePKCEPair() (verifier string, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func ExtractCodexAccountIDFromJWT(token string) (string, bool) {
	claims, ok := decodeJWTClaims(token)
	if !ok {
		return "", false
	}
	raw, ok := claims[codexJWTClaimPath]
	if !ok {
		return "", false
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return "", false
	}
	v, ok := obj["chatgpt_account_id"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func ExtractEmailFromJWT(token string) (string, bool) {
	claims, ok := decodeJWTClaims(token)
	if !ok {
		return "", false
	}
	v, ok := claims["email"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func decodeJWTClaims(token string) (map[string]any, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, false
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]any
	if err := common.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, false
	}
	return claims, true
}

func newCodexUpstreamAuthError(operation string, status int, body []byte) error {
	payload := struct {
		Error            any    `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
		Detail           string `json:"detail"`
	}{}
	_ = common.Unmarshal(body, &payload)
	upstreamError := sanitizeCodexOAuthErrorCode(codexOAuthErrorString(payload.Error))
	description := strings.TrimSpace(payload.ErrorDescription)
	if description == "" {
		description = strings.TrimSpace(payload.Message)
	}
	if description == "" {
		description = strings.TrimSpace(payload.Detail)
	}
	return &CodexUpstreamAuthError{
		Operation:        operation,
		Status:           status,
		UpstreamError:    upstreamError,
		ErrorDescription: common.MaskSensitiveInfo(description),
	}
}

func sanitizeCodexOAuthErrorCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return ""
	}
	return value
}

func codexOAuthErrorString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		for _, key := range []string{"code", "type", "message"} {
			if raw, ok := v[key]; ok {
				if s := strings.TrimSpace(fmt.Sprintf("%v", raw)); s != "" {
					return s
				}
			}
		}
	}
	return ""
}
