package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestStartCodexDeviceAuthorizationReturnsUserFacingCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var request map[string]string
		require.NoError(t, common.DecodeJson(r.Body, &request))
		assert.Equal(t, "client-id", request["client_id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_auth_id":"internal-device-id","usercode":"ABCD-EFGH","interval":"7"}`))
	}))
	defer server.Close()

	result, err := startCodexDeviceAuthorization(context.Background(), server.Client(), server.URL, "client-id")

	require.NoError(t, err)
	assert.Equal(t, "internal-device-id", result.DeviceAuthID)
	assert.Equal(t, "ABCD-EFGH", result.UserCode)
	assert.Equal(t, codexDeviceVerifyURL, result.VerificationURL)
	assert.Equal(t, 7*time.Second, result.Interval)
	assert.WithinDuration(t, time.Now().Add(codexDeviceFlowTTL), result.ExpiresAt, time.Second)
}

func TestPollCodexDeviceAuthorizationDistinguishesPendingAndCompletion(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		polls++
		var request map[string]string
		require.NoError(t, common.DecodeJson(r.Body, &request))
		assert.Equal(t, "internal-device-id", request["device_auth_id"])
		assert.Equal(t, "ABCD-EFGH", request["user_code"])
		if polls == 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorization_code":"authorization-code","code_verifier":"code-verifier"}`))
	}))
	defer server.Close()

	pending, err := pollCodexDeviceAuthorization(context.Background(), server.Client(), server.URL, "internal-device-id", "ABCD-EFGH")
	require.NoError(t, err)
	assert.True(t, pending.Pending)

	completed, err := pollCodexDeviceAuthorization(context.Background(), server.Client(), server.URL, "internal-device-id", "ABCD-EFGH")
	require.NoError(t, err)
	assert.False(t, completed.Pending)
	assert.Equal(t, "authorization-code", completed.AuthorizationCode)
	assert.Equal(t, "code-verifier", completed.CodeVerifier)
}

func TestCodexDeviceAuthorizationStateIsScopedByUserChannelAndDeviceID(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	states := []*CodexDeviceAuthorizationState{
		{UserID: 17, ChannelID: 5, DeviceAuthID: "shared-id", UserCode: "USER-A", Status: CodexDeviceAuthorizationPending, ExpiresAt: now.Add(time.Minute).UnixMilli()},
		{UserID: 18, ChannelID: 5, DeviceAuthID: "shared-id", UserCode: "USER-B", Status: CodexDeviceAuthorizationPending, ExpiresAt: now.Add(time.Minute).UnixMilli()},
		{UserID: 17, ChannelID: 6, DeviceAuthID: "shared-id", UserCode: "USER-C", Status: CodexDeviceAuthorizationPending, ExpiresAt: now.Add(time.Minute).UnixMilli()},
	}
	for _, state := range states {
		key := codexDeviceAuthorizationKey(state.UserID, state.ChannelID, state.DeviceAuthID)
		require.NoError(t, store.Create(context.Background(), key, state, time.Minute))
	}

	for _, state := range states {
		key := codexDeviceAuthorizationKey(state.UserID, state.ChannelID, state.DeviceAuthID)
		loaded, err := store.Get(context.Background(), key)
		require.NoError(t, err)
		assert.Equal(t, state.UserCode, loaded.UserCode)
		assert.Equal(t, CodexDeviceAuthorizationPending, loaded.Status)
	}
	_, err := store.Get(context.Background(), codexDeviceAuthorizationKey(19, 5, "shared-id"))
	assert.ErrorIs(t, err, ErrCodexDeviceAuthorizationNotFound)
}

func TestCodexDeviceAuthorizationRunnerPersistsPendingThenCompletion(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID:       17,
		ChannelID:    5,
		DeviceAuthID: "device-id",
		UserCode:     "ABCD-EFGH",
		Status:       CodexDeviceAuthorizationPending,
		ExpiresAt:    now.Add(codexDeviceFlowTTL).UnixMilli(),
	}, codexDeviceFlowTTL))

	pollCount := 0
	runner := codexDeviceAuthorizationRunner{
		store: store,
		poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
			pollCount++
			if pollCount == 1 {
				return &CodexDeviceAuthorizationPoll{Pending: true}, nil
			}
			return &CodexDeviceAuthorizationPoll{AuthorizationCode: "code", CodeVerifier: "verifier"}, nil
		},
		exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
			return &CodexOAuthTokenResult{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour)}, nil
		},
		now:          func() time.Time { return now },
		waitInterval: time.Millisecond,
		leaseTTL:     time.Second,
	}
	saveCount := 0
	save := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		saveCount++
		return &CodexDeviceAuthorizationResult{ChannelID: 5, AccountID: "account"}, "encoded", nil
	}

	pending, err := runner.Run(context.Background(), 17, 5, "device-id", "", save)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationPending, pending.Status)
	assert.Equal(t, 0, saveCount)

	completed, err := runner.Run(context.Background(), 17, 5, "device-id", "", save)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationCompleted, completed.Status)
	require.NotNil(t, completed.Result)
	assert.Equal(t, "account", completed.Result.AccountID)
	assert.Equal(t, 1, saveCount)
}

func TestCodexDeviceAuthorizationStateExpiresAtAbsoluteTTL(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID:       17,
		ChannelID:    5,
		DeviceAuthID: "device-id",
		UserCode:     "ABCD-EFGH",
		Status:       CodexDeviceAuthorizationPending,
		ExpiresAt:    now.Add(time.Minute).UnixMilli(),
	}, time.Minute))

	now = now.Add(time.Minute + time.Millisecond)
	_, err := store.Get(context.Background(), key)
	assert.ErrorIs(t, err, ErrCodexDeviceAuthorizationExpired)
}

func TestCodexDeviceAuthorizationPrepareFailureIsTerminalAndIdempotent(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID:       17,
		ChannelID:    5,
		DeviceAuthID: "device-id",
		UserCode:     "ABCD-EFGH",
		Status:       CodexDeviceAuthorizationPending,
		ExpiresAt:    now.Add(time.Minute).UnixMilli(),
	}, time.Minute))

	var pollCount atomic.Int32
	var exchangeCount atomic.Int32
	var saveCount atomic.Int32
	runner := codexDeviceAuthorizationRunner{
		store: store,
		poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
			pollCount.Add(1)
			return &CodexDeviceAuthorizationPoll{AuthorizationCode: "code", CodeVerifier: "verifier"}, nil
		},
		exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
			exchangeCount.Add(1)
			return &CodexOAuthTokenResult{AccessToken: "access"}, nil
		},
		now:          func() time.Time { return now },
		waitInterval: time.Millisecond,
		leaseTTL:     time.Second,
	}
	save := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		saveCount.Add(1)
		return nil, "", errors.New("decrypt oauth operation payload: invalid prepared credential")
	}

	first, err := runner.Run(context.Background(), 17, 5, "device-id", "", save)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationCompleted, first.Status)
	assert.Contains(t, first.Error, "invalid prepared credential")

	second, err := runner.Run(context.Background(), 17, 5, "device-id", "", save)
	require.NoError(t, err)
	assert.Equal(t, first.Error, second.Error)
	assert.Equal(t, int32(1), pollCount.Load())
	assert.Equal(t, int32(1), exchangeCount.Load())
	assert.Equal(t, int32(1), saveCount.Load())
}

func TestCodexDeviceAuthorizationSerializesAcrossReplicaRunnersAndNotifiesWaiters(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID:       17,
		ChannelID:    5,
		DeviceAuthID: "device-id",
		UserCode:     "ABCD-EFGH",
		Status:       CodexDeviceAuthorizationPending,
		ExpiresAt:    now.Add(time.Minute).UnixMilli(),
	}, time.Minute))

	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	var pollCount atomic.Int32
	var exchangeCount atomic.Int32
	var saveCount atomic.Int32
	newRunner := func() codexDeviceAuthorizationRunner {
		return codexDeviceAuthorizationRunner{
			store: store,
			poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
				pollCount.Add(1)
				startOnce.Do(func() { close(started) })
				<-release
				return &CodexDeviceAuthorizationPoll{AuthorizationCode: "code", CodeVerifier: "verifier"}, nil
			},
			exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
				exchangeCount.Add(1)
				return &CodexOAuthTokenResult{AccessToken: "access"}, nil
			},
			now:          func() time.Time { return now },
			waitInterval: time.Millisecond,
			leaseTTL:     time.Second,
		}
	}
	save := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		saveCount.Add(1)
		return &CodexDeviceAuthorizationResult{ChannelID: 5, AccountID: "account"}, "encoded", nil
	}

	results := make(chan *CodexDeviceAuthorizationState, 2)
	errs := make(chan error, 2)
	for range 2 {
		runner := newRunner()
		go func() {
			state, err := runner.Run(context.Background(), 17, 5, "device-id", "", save)
			results <- state
			errs <- err
		}()
		<-started
	}
	close(release)

	for range 2 {
		require.NoError(t, <-errs)
		state := <-results
		require.NotNil(t, state)
		assert.Equal(t, CodexDeviceAuthorizationCompleted, state.Status)
		require.NotNil(t, state.Result)
		assert.Equal(t, "account", state.Result.AccountID)
	}
	assert.Equal(t, int32(1), pollCount.Load())
	assert.Equal(t, int32(1), exchangeCount.Load())
	assert.Equal(t, int32(1), saveCount.Load())
}

func TestCodexDeviceAuthorizationFailsClosedWithoutSharedSQLStore(t *testing.T) {
	originalDB := model.DB
	model.DB = nil
	t.Cleanup(func() { model.DB = originalDB })

	err := RegisterCodexDeviceAuthorization(context.Background(), 17, 5, &CodexDeviceAuthorizationStart{
		DeviceAuthID: "device-id",
		UserCode:     "ABCD-EFGH",
		ExpiresAt:    time.Now().Add(time.Minute),
	})

	assert.ErrorIs(t, err, ErrCodexOAuthStoreUnavailable)
}

func TestCodexDeviceAuthorizationFailsClosedWhenStartupMigrationIsMissing(t *testing.T) {
	originalDB := model.DB
	originalSecret := common.CryptoSecret
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	model.DB = db
	common.CryptoSecret = "stable-test-secret"
	t.Cleanup(func() {
		model.DB = originalDB
		common.CryptoSecret = originalSecret
	})

	err = EnsureCodexOAuthOperationStore(context.Background())
	assert.ErrorIs(t, err, ErrCodexOAuthStoreUnavailable)
	assert.Contains(t, err.Error(), "migration is missing")
}

func TestCodexDeviceAuthorizationCancelFencesEverySensitiveStage(t *testing.T) {
	for _, stage := range []string{"poll", "exchange", "write"} {
		t.Run(stage, func(t *testing.T) {
			now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
			store, db := newCodexDeviceSQLTestStore(t, func() time.Time { return now })
			key := seedCodexDeviceSQLTest(t, store, db, now)
			started := make(chan struct{})
			release := make(chan struct{})
			var once sync.Once
			block := func() {
				once.Do(func() { close(started) })
				<-release
			}
			runner := codexDeviceAuthorizationRunner{
				store: store,
				poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
					if stage == "poll" {
						block()
					}
					return &CodexDeviceAuthorizationPoll{AuthorizationCode: "code", CodeVerifier: "verifier"}, nil
				},
				exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
					if stage == "exchange" {
						block()
					}
					return &CodexOAuthTokenResult{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour)}, nil
				},
				now: func() time.Time { return now }, waitInterval: time.Millisecond, leaseTTL: time.Second,
			}
			prepare := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
				if stage == "write" {
					block()
				}
				return &CodexDeviceAuthorizationResult{ChannelID: 5}, "new-credential", nil
			}
			done := make(chan error, 1)
			go func() {
				_, err := runner.Run(context.Background(), 17, 5, "device-id", "", prepare)
				done <- err
			}()
			<-started
			cancelled, err := store.Cancel(context.Background(), key, now)
			require.NoError(t, err)
			assert.Equal(t, CodexDeviceAuthorizationCancelled, cancelled.Status)
			close(release)
			require.Error(t, <-done)

			var channel model.Channel
			require.NoError(t, db.First(&channel, 5).Error)
			assert.Equal(t, "old-credential", channel.Key)
		})
	}
}

func TestCodexDeviceAuthorizationExpiryImmediatelyBeforeWriteLeavesChannelUnchanged(t *testing.T) {
	nowMillis := atomic.Int64{}
	start := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	nowMillis.Store(start.UnixMilli())
	now := func() time.Time { return time.UnixMilli(nowMillis.Load()) }
	store, db := newCodexDeviceSQLTestStore(t, now)
	seedCodexDeviceSQLTest(t, store, db, start)
	started := make(chan struct{})
	release := make(chan struct{})
	runner := codexDeviceAuthorizationRunner{
		store: store,
		poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
			return &CodexDeviceAuthorizationPoll{AuthorizationCode: "code", CodeVerifier: "verifier"}, nil
		},
		exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
			return &CodexOAuthTokenResult{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: start.Add(time.Hour)}, nil
		},
		now: now, waitInterval: time.Millisecond, leaseTTL: time.Second,
	}
	prepare := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		close(started)
		<-release
		return &CodexDeviceAuthorizationResult{ChannelID: 5}, "new-credential", nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(context.Background(), 17, 5, "device-id", "", prepare)
		done <- err
	}()
	<-started
	nowMillis.Store(start.Add(time.Minute + time.Millisecond).UnixMilli())
	close(release)
	require.Error(t, <-done)

	var channel model.Channel
	require.NoError(t, db.First(&channel, 5).Error)
	assert.Equal(t, "old-credential", channel.Key)
}

func TestCodexDeviceAuthorizationRejectsExpiryCrossedWhileAcquiringCommitLocks(t *testing.T) {
	nowMillis := atomic.Int64{}
	start := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	nowMillis.Store(start.UnixMilli())
	store, db := newCodexDeviceSQLTestStore(t, func() time.Time {
		return time.UnixMilli(nowMillis.Load())
	})
	key := seedCodexDeviceSQLTest(t, store, db, start)
	owner := "commit-owner"
	require.NoError(t, db.Model(&model.CodexOAuthOperation{}).
		Where("operation_key = ?", key).
		Updates(map[string]any{
			"status": CodexDeviceAuthorizationExchanging,
			"stage":  CodexDeviceAuthorizationStageExchanged,
			"owner":  owner, "fence": 1,
			"lease_until": start.Add(time.Minute).UnixMilli(),
		}).Error)
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(
		"test:advance-device-clock-after-channel-lock",
		func(tx *gorm.DB) {
			if tx.Statement.Table == "channels" {
				nowMillis.Store(start.Add(time.Minute + time.Millisecond).UnixMilli())
			}
		},
	))

	_, err := store.commitCredential(
		context.Background(), key, owner, 1, "new-credential",
		&CodexDeviceAuthorizationResult{ChannelID: 5}, start,
	)
	assert.ErrorIs(t, err, ErrCodexDeviceAuthorizationExpired)

	var channel model.Channel
	require.NoError(t, db.First(&channel, 5).Error)
	assert.Equal(t, "old-credential", channel.Key)
}

func TestCodexDeviceAuthorizationLeaseTakeoverResumesExchangedResultWithoutSecondExchange(t *testing.T) {
	nowMillis := atomic.Int64{}
	start := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	nowMillis.Store(start.UnixMilli())
	now := func() time.Time { return time.UnixMilli(nowMillis.Load()) }
	store, db := newCodexDeviceSQLTestStore(t, now)
	seedCodexDeviceSQLTest(t, store, db, start)
	started := make(chan struct{})
	release := make(chan struct{})
	var prepareCount atomic.Int32
	var exchangeCount atomic.Int32
	newRunner := func() codexDeviceAuthorizationRunner {
		return codexDeviceAuthorizationRunner{
			store: store,
			poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
				return &CodexDeviceAuthorizationPoll{AuthorizationCode: "code", CodeVerifier: "verifier"}, nil
			},
			exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
				exchangeCount.Add(1)
				return &CodexOAuthTokenResult{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: start.Add(time.Hour)}, nil
			},
			now: now, waitInterval: time.Millisecond, leaseTTL: time.Second,
		}
	}
	prepare := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		if prepareCount.Add(1) == 1 {
			close(started)
			<-release
		}
		return &CodexDeviceAuthorizationResult{ChannelID: 5, AccountID: "account"}, "new-credential", nil
	}
	firstDone := make(chan error, 1)
	first := newRunner()
	go func() {
		_, err := first.Run(context.Background(), 17, 5, "device-id", "", prepare)
		firstDone <- err
	}()
	<-started
	nowMillis.Store(start.Add(2 * time.Second).UnixMilli())
	second := newRunner()
	completed, err := second.Run(context.Background(), 17, 5, "device-id", "", prepare)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationCompleted, completed.Status)
	close(release)
	require.Error(t, <-firstDone)
	assert.Equal(t, int32(1), exchangeCount.Load())

	var channel model.Channel
	require.NoError(t, db.First(&channel, 5).Error)
	assert.Equal(t, "new-credential", channel.Key)
}

func TestCodexDeviceAuthorizationClassifiesTransientAndTerminalErrors(t *testing.T) {
	assert.False(t, isCodexDeviceAuthorizationTerminalError(context.DeadlineExceeded))
	assert.False(t, isCodexDeviceAuthorizationTerminalError(&CodexUpstreamAuthError{Status: http.StatusServiceUnavailable}))
	assert.False(t, isCodexDeviceAuthorizationTerminalError(&CodexUpstreamAuthError{Status: http.StatusTooManyRequests}))
	assert.True(t, isCodexDeviceAuthorizationTerminalError(&CodexUpstreamAuthError{Status: http.StatusBadRequest}))
	assert.True(t, isCodexDeviceAuthorizationTerminalError(errors.New("access_denied")))
	assert.Equal(t, 30*time.Second, codexDeviceRetryBackoff(20))
}

func TestCodexDeviceAuthorizationTransientFailureReturnsPendingWithBackoff(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID: 17, ChannelID: 5, DeviceAuthID: "device-id", UserCode: "ABCD-EFGH",
		Status: CodexDeviceAuthorizationPending, Stage: CodexDeviceAuthorizationStagePending,
		ExpiresAt: now.Add(time.Minute).UnixMilli(),
	}, time.Minute))
	runner := codexDeviceAuthorizationRunner{
		store: store,
		poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
			return nil, &CodexUpstreamAuthError{Status: http.StatusServiceUnavailable}
		},
		exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
			t.Fatal("exchange must not run")
			return nil, nil
		},
		now: func() time.Time { return now }, waitInterval: time.Millisecond, leaseTTL: time.Second,
	}
	state, err := runner.Run(
		context.Background(), 17, 5, "device-id", "",
		func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
			t.Fatal("prepare must not run")
			return nil, "", nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationPending, state.Status)
	assert.Equal(t, 1, state.RetryCount)
	assert.Equal(t, now.Add(time.Second).UnixMilli(), state.NextAttemptAt)
}

func TestCodexDeviceAuthorizationExchangeTimeoutIsUncertainAndNeverRetried(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID: 17, ChannelID: 5, DeviceAuthID: "device-id", UserCode: "ABCD-EFGH",
		Status: CodexDeviceAuthorizationPending, Stage: CodexDeviceAuthorizationStagePending,
		ExpiresAt: now.Add(time.Minute).UnixMilli(),
	}, time.Minute))
	var exchangeCount atomic.Int32
	runner := codexDeviceAuthorizationRunner{
		store: store,
		poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
			return &CodexDeviceAuthorizationPoll{AuthorizationCode: "one-time-code", CodeVerifier: "verifier"}, nil
		},
		exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
			exchangeCount.Add(1)
			return nil, context.DeadlineExceeded
		},
		now: func() time.Time { return now }, waitInterval: time.Millisecond, leaseTTL: time.Second,
	}
	prepare := func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		t.Fatal("uncertain exchange must not reach credential preparation")
		return nil, "", nil
	}

	first, err := runner.Run(context.Background(), 17, 5, "device-id", "", prepare)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationUncertain, first.Status)
	assert.Equal(t, CodexDeviceAuthorizationStageExchangeStarted, first.Stage)
	second, err := runner.Run(context.Background(), 17, 5, "device-id", "", prepare)
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationUncertain, second.Status)
	assert.Equal(t, int32(1), exchangeCount.Load())
}

func TestCodexDeviceAuthorizationCrashAfterExchangeStartedNeverReusesCode(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	store := newCodexDeviceAuthorizationMemoryStore(func() time.Time { return now })
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID: 17, ChannelID: 5, DeviceAuthID: "device-id", UserCode: "ABCD-EFGH",
		Status: CodexDeviceAuthorizationPending, Stage: CodexDeviceAuthorizationStagePending,
		ExpiresAt: now.Add(time.Minute).UnixMilli(),
	}, time.Minute))
	claimed, ok, err := store.Claim(context.Background(), key, "dead-owner", now, now.Add(time.Second))
	require.NoError(t, err)
	require.True(t, ok)
	payload, err := sealCodexOAuthPayload(&codexDeviceAuthorizationPollPayload{
		AuthorizationCode: "one-time-code", CodeVerifier: "verifier",
	})
	require.NoError(t, err)
	_, err = store.Advance(context.Background(), key, "dead-owner", claimed.Fence, CodexDeviceAuthorizationStageExchangeStarted, payload, now)
	require.NoError(t, err)

	now = now.Add(2 * time.Second)
	var exchangeCount atomic.Int32
	runner := codexDeviceAuthorizationRunner{
		store: store,
		poll: func(context.Context, string, string, string) (*CodexDeviceAuthorizationPoll, error) {
			t.Fatal("takeover must not poll again")
			return nil, nil
		},
		exchange: func(context.Context, string, string, string) (*CodexOAuthTokenResult, error) {
			exchangeCount.Add(1)
			return nil, nil
		},
		now: func() time.Time { return now }, waitInterval: time.Millisecond, leaseTTL: time.Second,
	}
	state, err := runner.Run(context.Background(), 17, 5, "device-id", "", func(*CodexOAuthTokenResult) (*CodexDeviceAuthorizationResult, string, error) {
		t.Fatal("takeover must not prepare a credential")
		return nil, "", nil
	})
	require.NoError(t, err)
	assert.Equal(t, CodexDeviceAuthorizationUncertain, state.Status)
	assert.Equal(t, int32(0), exchangeCount.Load())
}

func newCodexDeviceSQLTestStore(t *testing.T, now func() time.Time) (*codexDeviceAuthorizationSQLStore, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:codex-device-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.CodexOAuthOperation{}))
	originalDB := model.DB
	originalSecret := common.CryptoSecret
	model.DB = db
	common.CryptoSecret = "codex-device-test-secret"
	t.Cleanup(func() {
		model.DB = originalDB
		common.CryptoSecret = originalSecret
	})
	return &codexDeviceAuthorizationSQLStore{db: db, now: now}, db
}

func seedCodexDeviceSQLTest(t *testing.T, store *codexDeviceAuthorizationSQLStore, db *gorm.DB, now time.Time) string {
	t.Helper()
	require.NoError(t, db.Create(&model.Channel{
		Id: 5, Type: constant.ChannelTypeCodex, Name: "OpenAI - Codex", Key: "old-credential",
	}).Error)
	key := codexDeviceAuthorizationKey(17, 5, "device-id")
	require.NoError(t, store.Create(context.Background(), key, &CodexDeviceAuthorizationState{
		UserID: 17, ChannelID: 5, DeviceAuthID: "device-id", UserCode: "ABCD-EFGH",
		Status: CodexDeviceAuthorizationPending, Stage: CodexDeviceAuthorizationStagePending,
		ExpiresAt: now.Add(time.Minute).UnixMilli(),
	}, time.Minute))
	return key
}
