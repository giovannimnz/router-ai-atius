package embeddinggovernor

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireNoopsForNonGovernedModel(t *testing.T) {
	g := New(testConfig())

	lease, reject := g.Acquire(context.Background(), Request{Model: "gpt-5.4"})

	require.Nil(t, reject)
	assert.Nil(t, lease)
	assert.Equal(t, 0, g.Snapshot().Running)
}

func TestLoadConfigUsesAdaptiveAutoscalingDefaults(t *testing.T) {
	t.Setenv("EMBEDDING_GOVERNOR_AUTO_WORKLOAD", "")
	t.Setenv("EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_MIN_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_MAX_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_CONCURRENCY", "")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_TIMEOUT", "")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_SLOW_REQUEST_DURATION", "")

	cfg := LoadConfigFromEnv()

	assert.Equal(t, 1, cfg.MinConcurrency)
	assert.Equal(t, 2, cfg.InitialConcurrency)
	assert.Equal(t, 0, cfg.MaxConcurrency)
	assert.Equal(t, 0, cfg.BatchConcurrency)
	assert.Equal(t, 10*time.Minute, cfg.BatchTimeout)
	assert.Equal(t, 10*time.Minute, cfg.BatchSlowRequestDuration)
	assert.True(t, cfg.AutoWorkload)
	assert.Equal(t, 2, cfg.BatchInputCountThreshold)
	assert.True(t, cfg.Models["embedding-gte-v1"])
	assert.False(t, cfg.Models["embedding-gte-v1-batch"])
	assert.Empty(t, cfg.BatchModels)
}

func TestUnboundedMaxConcurrencyScalesPastLegacyCapWhenHealthy(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 1
		cfg.MaxConcurrency = 0
		cfg.ScaleUpMinInterval = 0
		cfg.LatencyTarget = 0
		cfg.SuccessWindow = 100
	}))

	leases := make([]*Lease, 0, 5)
	for i := 0; i < 5; i++ {
		lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
		require.Nil(t, reject)
		require.NotNil(t, lease)
		leases = append(leases, lease)
	}

	snapshot := g.Snapshot()
	assert.Equal(t, 5, snapshot.CurrentConcurrency)
	assert.Equal(t, 5, snapshot.Running)
	assert.Equal(t, 0, snapshot.MaxConcurrency)

	for _, lease := range leases {
		lease.Finish(true, http.StatusOK, time.Millisecond)
	}
}

func TestUnboundedBatchConcurrencyUsesTotalAdaptivePool(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 2
		cfg.MaxConcurrency = 0
		cfg.BatchConcurrency = 0
	}))

	first, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1", Workload: "batch"})
	require.Nil(t, reject)
	require.NotNil(t, first)

	second, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1", Workload: "batch"})
	require.Nil(t, reject)
	require.NotNil(t, second)

	snapshot := g.Snapshot()
	assert.Equal(t, 2, snapshot.RunningBatch)
	assert.Equal(t, 0, snapshot.BatchConcurrency)

	first.Finish(true, http.StatusOK, time.Millisecond)
	second.Finish(true, http.StatusOK, time.Millisecond)
}

func TestWorkloadMetadataClassifiesUnlabeledLargeInputsAsBatch(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.BatchInputCountThreshold = 2
		cfg.BatchInputCharsThreshold = 12000
	}))

	tests := []struct {
		name string
		req  Request
		want bool
	}{
		{
			name: "small unlabeled interactive request stays interactive",
			req: Request{
				Model:      "embedding-gte-v1",
				InputCount: 1,
				InputChars: 640,
			},
			want: false,
		},
		{
			name: "large unlabeled request by count becomes batch",
			req: Request{
				Model:      "embedding-gte-v1",
				InputCount: 2,
				InputChars: 2000,
			},
			want: true,
		},
		{
			name: "large unlabeled request by chars becomes batch",
			req: Request{
				Model:      "embedding-gte-v1",
				InputCount: 1,
				InputChars: 12000,
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, map[bool]string{true: "batch", false: "interactive"}[tc.want], g.ClassifyWorkload(tc.req))
			assert.Equal(t, tc.want, g.isBatch(tc.req))
		})
	}
}

func TestWorkloadHeaderOverridesMetadataClassification(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.BatchInputCountThreshold = 2
		cfg.BatchInputCharsThreshold = 12000
	}))

	tests := []struct {
		name string
		req  Request
		want bool
	}{
		{
			name: "batch header forces batch below thresholds",
			req: Request{
				Model:      "embedding-gte-v1",
				Workload:   "batch",
				InputCount: 1,
				InputChars: 320,
			},
			want: true,
		},
		{
			name: "bulk header forces batch below thresholds",
			req: Request{
				Model:      "embedding-gte-v1",
				Workload:   "bulk",
				InputCount: 1,
				InputChars: 320,
			},
			want: true,
		},
		{
			name: "interactive header wins over count threshold",
			req: Request{
				Model:      "embedding-gte-v1",
				Workload:   "interactive",
				InputCount: 8,
				InputChars: 24000,
			},
			want: false,
		},
		{
			name: "realtime header wins over chars threshold",
			req: Request{
				Model:      "embedding-gte-v1",
				Workload:   "realtime",
				InputCount: 2,
				InputChars: 24000,
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, map[bool]string{true: "batch", false: "interactive"}[tc.want], g.ClassifyWorkload(tc.req))
			assert.Equal(t, tc.want, g.isBatch(tc.req))
		})
	}
}

func TestLoadConfigNormalizesWorkloadMetadataThresholds(t *testing.T) {
	t.Setenv("EMBEDDING_GOVERNOR_AUTO_WORKLOAD", "false")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_INPUT_COUNT_THRESHOLD", "-1")
	t.Setenv("EMBEDDING_GOVERNOR_BATCH_INPUT_CHARS_THRESHOLD", "not-a-number")

	cfg := LoadConfigFromEnv()

	assert.False(t, cfg.AutoWorkload)
	assert.Equal(t, defaultBatchInputCountThreshold, cfg.BatchInputCountThreshold)
	assert.Equal(t, defaultBatchInputCharsThreshold, cfg.BatchInputCharsThreshold)
}

func TestAutoWorkloadDisabledFallsBackToHeadersAndBatchModels(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.AutoWorkload = false
		cfg.BatchModels = map[string]bool{}
		cfg.BatchInputCountThreshold = 2
		cfg.BatchInputCharsThreshold = 12000
	}))

	assert.Equal(t, "interactive", g.ClassifyWorkload(Request{
		Model:      "embedding-gte-v1",
		InputCount: 2,
		InputChars: 20000,
	}))
	assert.Equal(t, "batch", g.ClassifyWorkload(Request{
		Model:      "embedding-gte-v1",
		Workload:   "batch",
		InputCount: 1,
		InputChars: 100,
	}))
	assert.Equal(t, "interactive", g.ClassifyWorkload(Request{
		Model:      "embedding-gte-v1",
		Workload:   "interactive",
		InputCount: 2,
		InputChars: 20000,
	}))

	gWithBatchModel := New(testConfigWith(func(cfg *Config) {
		cfg.AutoWorkload = false
		cfg.BatchModels = map[string]bool{
			"embedding-gte-v1": true,
		}
	}))
	assert.Equal(t, "batch", gWithBatchModel.ClassifyWorkload(Request{
		Model: "embedding-gte-v1",
	}))
}

func TestIsGovernedModelMatchesDefaultScope(t *testing.T) {
	restore := ResetForTest(testConfig())
	defer restore()

	assert.True(t, IsGovernedModel("embedding-gte-v1"))
	assert.False(t, IsGovernedModel("embedding-gte-v1-batch"))
	assert.False(t, IsGovernedModel("gpt-5.4"))
}

func TestHealthProbeDisabledByDefault(t *testing.T) {
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED", "")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_URL", "")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_TIMEOUT", "")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_INTERVAL", "")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_BAD_WINDOW_THRESHOLD", "")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_SLOW_DURATION", "")

	cfg := LoadConfigFromEnv()

	assert.False(t, cfg.HealthProbeEnabled)
	assert.Empty(t, cfg.HealthProbeURL)
	assert.Equal(t, 30*time.Second, cfg.HealthProbeTimeout)
	assert.Equal(t, 30*time.Second, cfg.HealthProbeInterval)
	assert.Equal(t, 3, cfg.HealthBadWindowThreshold)
	assert.Equal(t, 10*time.Second, cfg.HealthSlowDuration)

	g := newGovernor(cfg, true)
	snapshot := g.Snapshot()
	assert.False(t, snapshot.HealthProbeEnabled)
	assert.Equal(t, 0, snapshot.HealthBadWindows)
	assert.Empty(t, snapshot.LastHealthStatus)
	assert.Equal(t, int64(0), snapshot.LastHealthLatencyMs)
	assert.True(t, snapshot.LastHealthAt.IsZero())
	assert.Nil(t, g.healthProbeStop)
}

func TestHealthHysteresisIgnoresSingleBadSample(t *testing.T) {
	now := time.Unix(200, 0)
	g := newGovernor(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 2
		cfg.MaxConcurrency = 3
		cfg.HealthProbeEnabled = true
		cfg.HealthProbeURL = "http://tei.local/health"
		cfg.HealthProbeTimeout = 30 * time.Second
		cfg.HealthProbeInterval = 30 * time.Second
		cfg.HealthBadWindowThreshold = 3
		cfg.HealthSlowDuration = 10 * time.Second
	}), false)
	g.clock = func() time.Time {
		return now
	}

	g.observeHealthSample(0, 30*time.Second, context.DeadlineExceeded)

	snapshot := g.Snapshot()
	assert.True(t, snapshot.HealthProbeEnabled)
	assert.Equal(t, 2, snapshot.CurrentConcurrency)
	assert.True(t, snapshot.CooldownUntil.IsZero())
	assert.True(t, snapshot.LastFailureAt.IsZero())
	assert.Equal(t, 1, snapshot.HealthBadWindows)
	assert.Equal(t, "timeout", snapshot.LastHealthStatus)
	assert.Equal(t, int64((30 * time.Second).Milliseconds()), snapshot.LastHealthLatencyMs)
	assert.Equal(t, now, snapshot.LastHealthAt)
	assert.True(t, healthCanIncrease(g, false))
}

func TestHealthHysteresisReducesAfterConsecutiveBadWindows(t *testing.T) {
	now := time.Unix(300, 0)
	g := newGovernor(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 3
		cfg.MaxConcurrency = 3
		cfg.MinConcurrency = 1
		cfg.HealthProbeEnabled = true
		cfg.HealthProbeURL = "http://tei.local/health"
		cfg.HealthProbeTimeout = 30 * time.Second
		cfg.HealthProbeInterval = 30 * time.Second
		cfg.HealthBadWindowThreshold = 3
		cfg.HealthSlowDuration = 10 * time.Second
	}), false)
	g.clock = func() time.Time {
		return now
	}

	g.observeHealthSample(0, 30*time.Second, context.DeadlineExceeded)
	now = now.Add(time.Minute)
	g.observeHealthSample(0, 30*time.Second, context.DeadlineExceeded)

	beforeThreshold := g.Snapshot()
	assert.Equal(t, 3, beforeThreshold.CurrentConcurrency)
	assert.Equal(t, 2, beforeThreshold.HealthBadWindows)

	now = now.Add(time.Minute)
	g.observeHealthSample(0, 30*time.Second, context.DeadlineExceeded)

	atThreshold := g.Snapshot()
	assert.Equal(t, 2, atThreshold.CurrentConcurrency)
	assert.Equal(t, 3, atThreshold.HealthBadWindows)
	assert.True(t, atThreshold.CooldownUntil.IsZero())
	assert.Equal(t, "timeout", atThreshold.LastHealthStatus)
	assert.False(t, healthCanIncrease(g, false))

	now = now.Add(time.Minute)
	g.observeHealthSample(http.StatusServiceUnavailable, 500*time.Millisecond, nil)

	afterThreshold := g.Snapshot()
	assert.Equal(t, 1, afterThreshold.CurrentConcurrency)
	assert.Equal(t, 4, afterThreshold.HealthBadWindows)
	assert.Equal(t, "http_503", afterThreshold.LastHealthStatus)
}

func TestHealthHysteresisHealthySampleResetsBadWindows(t *testing.T) {
	now := time.Unix(400, 0)
	g := newGovernor(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 3
		cfg.MaxConcurrency = 3
		cfg.MinConcurrency = 1
		cfg.HealthProbeEnabled = true
		cfg.HealthProbeURL = "http://tei.local/health"
		cfg.HealthProbeTimeout = 30 * time.Second
		cfg.HealthProbeInterval = 30 * time.Second
		cfg.HealthBadWindowThreshold = 3
		cfg.HealthSlowDuration = 10 * time.Second
	}), false)
	g.clock = func() time.Time {
		return now
	}

	g.observeHealthSample(0, 30*time.Second, context.DeadlineExceeded)
	now = now.Add(time.Minute)
	g.observeHealthSample(http.StatusServiceUnavailable, 500*time.Millisecond, nil)

	badSnapshot := g.Snapshot()
	assert.Equal(t, 2, badSnapshot.HealthBadWindows)
	assert.Equal(t, 3, badSnapshot.CurrentConcurrency)

	now = now.Add(time.Minute)
	g.observeHealthSample(http.StatusOK, 1500*time.Millisecond, nil)

	resetSnapshot := g.Snapshot()
	assert.Equal(t, 0, resetSnapshot.HealthBadWindows)
	assert.Equal(t, 3, resetSnapshot.CurrentConcurrency)
	assert.Equal(t, "ok", resetSnapshot.LastHealthStatus)

	now = now.Add(time.Minute)
	g.observeHealthSample(0, 30*time.Second, context.DeadlineExceeded)

	afterReset := g.Snapshot()
	assert.Equal(t, 1, afterReset.HealthBadWindows)
	assert.Equal(t, 3, afterReset.CurrentConcurrency)
}

func TestLoadConfigNormalizesHealthProbeSettings(t *testing.T) {
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_ENABLED", "true")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_URL", "http://127.0.0.1:9999/health")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_TIMEOUT", "5s")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_PROBE_INTERVAL", "0s")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_BAD_WINDOW_THRESHOLD", "1")
	t.Setenv("EMBEDDING_GOVERNOR_HEALTH_SLOW_DURATION", "-1s")

	cfg := LoadConfigFromEnv()

	assert.True(t, cfg.HealthProbeEnabled)
	assert.Equal(t, "http://127.0.0.1:9999/health", cfg.HealthProbeURL)
	assert.Equal(t, 30*time.Second, cfg.HealthProbeTimeout)
	assert.Equal(t, 30*time.Second, cfg.HealthProbeInterval)
	assert.Equal(t, 3, cfg.HealthBadWindowThreshold)
	assert.Equal(t, 10*time.Second, cfg.HealthSlowDuration)
}

func TestSplitLatencyMetricsTrackInteractiveAndBatchSeparately(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.BatchSlowRequestDuration = 10 * time.Minute
	}))

	interactive, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, interactive)
	interactive.Finish(true, http.StatusOK, 2*time.Second)

	batch, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1", Workload: "batch"})
	require.Nil(t, reject)
	require.NotNil(t, batch)
	batch.Finish(true, http.StatusOK, 5*time.Minute)

	snapshot := g.Snapshot()
	assert.Equal(t, int64((2 * time.Second).Milliseconds()), snapshot.InteractiveAverageLatencyMs)
	assert.Equal(t, int64((5 * time.Minute).Milliseconds()), snapshot.BatchAverageLatencyMs)
	assert.Equal(t, uint64(1), snapshot.InteractiveCompleted)
	assert.Equal(t, uint64(1), snapshot.BatchCompleted)
	assert.NotEqual(t, snapshot.InteractiveAverageLatencyMs, snapshot.BatchAverageLatencyMs)
}

func TestBatchLatencyDoesNotBlockInteractiveScaleUp(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 1
		cfg.MaxConcurrency = 3
		cfg.SuccessWindow = 100
		cfg.ScaleUpMinInterval = 0
		cfg.LatencyTarget = 90 * time.Second
		cfg.BatchSlowRequestDuration = 10 * time.Minute
	}))

	batch, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1", Workload: "batch"})
	require.Nil(t, reject)
	require.NotNil(t, batch)
	batch.Finish(true, http.StatusOK, 5*time.Minute)

	firstInteractive, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, firstInteractive)

	secondInteractive, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, secondInteractive)
	assert.Equal(t, 2, g.Snapshot().CurrentConcurrency)

	firstInteractive.Finish(true, http.StatusOK, time.Second)
	secondInteractive.Finish(true, http.StatusOK, time.Second)
}

func TestSnapshotContainsOnlyAggregateEmbeddingGovernorMetadata(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.BatchSlowRequestDuration = 10 * time.Minute
	}))

	interactive, reject := g.Acquire(context.Background(), Request{
		Model:      "embedding-gte-v1",
		InputCount: 3,
		InputChars: 4096,
		Workload:   "interactive",
	})
	require.Nil(t, reject)
	require.NotNil(t, interactive)
	interactive.Finish(true, http.StatusOK, 1500*time.Millisecond)

	batch, reject := g.Acquire(context.Background(), Request{
		Model:      "embedding-gte-v1",
		InputCount: 4,
		InputChars: 16000,
		Workload:   "batch",
	})
	require.Nil(t, reject)
	require.NotNil(t, batch)
	batch.Finish(true, http.StatusOK, 4*time.Minute)

	payload, err := common.Marshal(g.Snapshot())
	require.NoError(t, err)

	raw := strings.ToLower(string(payload))
	assert.Contains(t, raw, "\"interactive_average_latency_ms\"")
	assert.Contains(t, raw, "\"batch_average_latency_ms\"")
	assert.Contains(t, raw, "\"interactive_completed\"")
	assert.Contains(t, raw, "\"batch_completed\"")
	assert.Contains(t, raw, "\"interactive_slow\"")
	assert.Contains(t, raw, "\"batch_slow\"")
	assert.NotContains(t, raw, "input_count")
	assert.NotContains(t, raw, "input_chars")
	assert.NotContains(t, raw, "workload")
	assert.NotContains(t, raw, "channel_name")
	assert.NotContains(t, raw, "authorization")
	assert.NotContains(t, raw, "token")
	assert.NotContains(t, raw, "secret")
}

func TestStatusClassificationIgnoresClientErrors(t *testing.T) {
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusUnprocessableEntity,
	}

	for _, status := range statuses {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			g := New(testConfigWith(func(cfg *Config) {
				cfg.InitialConcurrency = 2
				cfg.MaxConcurrency = 3
				cfg.MinConcurrency = 1
				cfg.Cooldown = time.Minute
			}))

			lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
			require.Nil(t, reject)
			require.NotNil(t, lease)
			lease.Finish(false, status, 250*time.Millisecond)

			snapshot := g.Snapshot()
			assert.Equal(t, 2, snapshot.CurrentConcurrency)
			assert.True(t, snapshot.CooldownUntil.IsZero())
			assert.True(t, snapshot.LastFailureAt.IsZero())
		})
	}
}

func TestStatusClassificationReducesOnPressureFailures(t *testing.T) {
	statuses := []int{
		0,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
	}

	for _, status := range statuses {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			g := New(testConfigWith(func(cfg *Config) {
				cfg.InitialConcurrency = 2
				cfg.MaxConcurrency = 3
				cfg.MinConcurrency = 1
				cfg.Cooldown = time.Minute
			}))

			lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
			require.Nil(t, reject)
			require.NotNil(t, lease)
			lease.Finish(false, status, 250*time.Millisecond)

			snapshot := g.Snapshot()
			assert.Equal(t, 1, snapshot.CurrentConcurrency)
			assert.False(t, snapshot.CooldownUntil.IsZero())
			assert.False(t, snapshot.LastFailureAt.IsZero())
		})
	}
}

func TestStatusClassificationKeepsSlowRequestsAsPressure(t *testing.T) {
	tests := []struct {
		name                    string
		req                     Request
		latency                 time.Duration
		expectedSlow            uint64
		expectedBatchSlow       uint64
		expectedInteractiveSlow uint64
	}{
		{
			name:                    "interactive slow request still applies pressure",
			req:                     Request{Model: "embedding-gte-v1"},
			latency:                 3 * time.Minute,
			expectedSlow:            1,
			expectedInteractiveSlow: 1,
		},
		{
			name:              "batch slow request still applies pressure",
			req:               Request{Model: "embedding-gte-v1", Workload: "batch"},
			latency:           11 * time.Minute,
			expectedSlow:      1,
			expectedBatchSlow: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := New(testConfigWith(func(cfg *Config) {
				cfg.InitialConcurrency = 2
				cfg.MaxConcurrency = 3
				cfg.MinConcurrency = 1
				cfg.Cooldown = time.Minute
				cfg.SlowRequestDuration = 2 * time.Minute
				cfg.BatchSlowRequestDuration = 10 * time.Minute
			}))

			lease, reject := g.Acquire(context.Background(), tc.req)
			require.Nil(t, reject)
			require.NotNil(t, lease)
			lease.Finish(true, http.StatusOK, tc.latency)

			snapshot := g.Snapshot()
			assert.Equal(t, 1, snapshot.CurrentConcurrency)
			assert.False(t, snapshot.CooldownUntil.IsZero())
			assert.Equal(t, tc.expectedSlow, snapshot.Slow)
			assert.Equal(t, tc.expectedBatchSlow, snapshot.BatchSlow)
			assert.Equal(t, tc.expectedInteractiveSlow, snapshot.InteractiveSlow)
		})
	}
}

func TestAcquireRejectsWhenQueueIsFull(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 1
		cfg.MaxConcurrency = 1
		cfg.QueueLimit = 1
	}))

	first, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, first)

	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	waiting := make(chan *Lease, 1)
	waitingReject := make(chan *Reject, 1)
	go func() {
		lease, reject := g.Acquire(waitCtx, Request{Model: "embedding-gte-v1"})
		waiting <- lease
		waitingReject <- reject
	}()
	require.Eventually(t, func() bool {
		return g.Snapshot().WaitingInteractive == 1
	}, time.Second, 10*time.Millisecond)

	third, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, third)
	require.NotNil(t, reject)
	assert.Equal(t, "embedding_governor_queue_full", reject.Code)
	assert.Equal(t, http.StatusTooManyRequests, reject.StatusCode)

	first.Finish(true, http.StatusOK, time.Millisecond)
	second := <-waiting
	secondReject := <-waitingReject
	require.Nil(t, secondReject)
	require.NotNil(t, second)
	second.Finish(true, http.StatusOK, time.Millisecond)
}

func TestInteractiveRequestCanPassWhileBatchIsQueued(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 2
		cfg.MaxConcurrency = 2
		cfg.BatchConcurrency = 1
	}))

	batch := Request{Model: "embedding-gte-v1", Workload: "batch"}
	firstBatch, reject := g.Acquire(context.Background(), batch)
	require.Nil(t, reject)
	require.NotNil(t, firstBatch)

	batchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	batchWaiting := make(chan *Lease, 1)
	batchReject := make(chan *Reject, 1)
	go func() {
		lease, reject := g.Acquire(batchCtx, batch)
		batchWaiting <- lease
		batchReject <- reject
	}()
	require.Eventually(t, func() bool {
		return g.Snapshot().WaitingBatch == 1
	}, time.Second, 10*time.Millisecond)

	interactive, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, interactive)

	select {
	case lease := <-batchWaiting:
		require.Nil(t, lease, "batch must not bypass its own concurrency limit")
	case <-time.After(50 * time.Millisecond):
	}

	firstBatch.Finish(true, http.StatusOK, time.Millisecond)
	secondBatch := <-batchWaiting
	secondBatchReject := <-batchReject
	require.Nil(t, secondBatchReject)
	require.NotNil(t, secondBatch)

	interactive.Finish(true, http.StatusOK, time.Millisecond)
	secondBatch.Finish(true, http.StatusOK, time.Millisecond)
}

func TestInteractiveDemandCanScaleFromOneWhenHealthy(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 1
		cfg.MaxConcurrency = 3
		cfg.SuccessWindow = 100
		cfg.ScaleUpMinInterval = 0
		cfg.LatencyTarget = 0
	}))

	first, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, first)

	second, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, second)
	assert.Equal(t, 2, g.Snapshot().CurrentConcurrency)

	first.Finish(true, http.StatusOK, time.Millisecond)
	second.Finish(true, http.StatusOK, time.Millisecond)
}

func TestGovernorHoldsRequestsDuringCooldownBeforeReopening(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 2
		cfg.MaxConcurrency = 3
		cfg.MinConcurrency = 1
		cfg.Cooldown = 30 * time.Millisecond
		cfg.InteractiveTimeout = 200 * time.Millisecond
	}))

	finishSequential(t, g, false, http.StatusInternalServerError)

	startedAt := time.Now()
	lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, lease)
	assert.GreaterOrEqual(t, time.Since(startedAt), 25*time.Millisecond)

	lease.Finish(true, http.StatusOK, time.Millisecond)
}

func TestGovernorReducesOnFailureAndReopensAfterCooldown(t *testing.T) {
	now := time.Unix(100, 0)
	g := New(testConfigWith(func(cfg *Config) {
		cfg.InitialConcurrency = 2
		cfg.MaxConcurrency = 3
		cfg.MinConcurrency = 1
		cfg.SuccessWindow = 2
		cfg.Cooldown = time.Minute
	}))
	g.clock = func() time.Time {
		return now
	}

	finishSequential(t, g, true, http.StatusOK)
	finishSequential(t, g, true, http.StatusOK)
	assert.Equal(t, 3, g.Snapshot().CurrentConcurrency)

	finishSequential(t, g, false, http.StatusInternalServerError)
	snapshot := g.Snapshot()
	assert.Equal(t, 1, snapshot.CurrentConcurrency)
	assert.True(t, snapshot.CooldownUntil.After(now))

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	lease, reject := g.Acquire(ctx, Request{Model: "embedding-gte-v1"})
	require.Nil(t, lease)
	require.NotNil(t, reject)
	assert.Equal(t, "embedding_governor_queue_timeout", reject.Code)

	now = now.Add(time.Minute + time.Nanosecond)
	finishSequential(t, g, true, http.StatusOK)
	finishSequential(t, g, true, http.StatusOK)
	assert.Equal(t, 2, g.Snapshot().CurrentConcurrency)
}

func TestBatchUsesBatchSlowRequestDuration(t *testing.T) {
	g := New(testConfigWith(func(cfg *Config) {
		cfg.SlowRequestDuration = 2 * time.Minute
		cfg.BatchSlowRequestDuration = 10 * time.Minute
	}))

	lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1", Workload: "batch"})
	require.Nil(t, reject)
	require.NotNil(t, lease)
	lease.Finish(true, http.StatusOK, 5*time.Minute)

	snapshot := g.Snapshot()
	assert.Equal(t, uint64(1), snapshot.Completed)
	assert.Equal(t, uint64(0), snapshot.Failed)
	assert.Equal(t, uint64(0), snapshot.Slow)
}

func finishSequential(t *testing.T, g *Governor, success bool, statusCode int) {
	t.Helper()
	lease, reject := g.Acquire(context.Background(), Request{Model: "embedding-gte-v1"})
	require.Nil(t, reject)
	require.NotNil(t, lease)
	lease.Finish(success, statusCode, time.Millisecond)
}

func healthCanIncrease(g *Governor, batch bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.canIncreaseLocked(g.clock(), batch)
}

func testConfig() Config {
	return Config{
		Enabled:                  true,
		Models:                   parseCSVSet(defaultModels),
		BatchModels:              parseCSVSet(defaultBatchModels),
		AutoWorkload:             true,
		InitialConcurrency:       2,
		MinConcurrency:           1,
		MaxConcurrency:           3,
		BatchConcurrency:         1,
		QueueLimit:               8,
		BatchQueueLimit:          8,
		InteractiveTimeout:       time.Second,
		BatchTimeout:             time.Second,
		Cooldown:                 time.Minute,
		SlowRequestDuration:      0,
		LatencyTarget:            0,
		ScaleUpMinInterval:       0,
		ScaleDownIdle:            0,
		SuccessWindow:            20,
		HealthProbeTimeout:       30 * time.Second,
		HealthProbeInterval:      30 * time.Second,
		HealthBadWindowThreshold: 3,
		HealthSlowDuration:       10 * time.Second,
	}
}

func testConfigWith(update func(*Config)) Config {
	cfg := testConfig()
	update(&cfg)
	return cfg
}
