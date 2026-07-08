package embeddinggovernor

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultModels             = "embedding-pt-v1,embedding-pt-v1-batch"
	defaultBatchModels        = "embedding-pt-v1-batch"
	defaultInitialConcurrency = 1
	defaultMinConcurrency     = 1
	defaultMaxConcurrency     = 4
	defaultBatchConcurrency   = 1
	defaultQueueLimit         = 128
	defaultBatchQueueLimit    = 512
	defaultSuccessWindow      = 8
)

// Request carries only routing metadata. It must never include embedding input text.
type Request struct {
	Model       string
	ChannelID   int
	ChannelName string
	Workload    string
}

type Reject struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter time.Duration
}

type Config struct {
	Enabled             bool
	Models              map[string]bool
	BatchModels         map[string]bool
	InitialConcurrency  int
	MinConcurrency      int
	MaxConcurrency      int
	BatchConcurrency    int
	QueueLimit          int
	BatchQueueLimit     int
	InteractiveTimeout  time.Duration
	BatchTimeout        time.Duration
	Cooldown            time.Duration
	SlowRequestDuration time.Duration
	LatencyTarget       time.Duration
	ScaleUpMinInterval  time.Duration
	ScaleDownIdle       time.Duration
	SuccessWindow       int
}

type Snapshot struct {
	Enabled              bool      `json:"enabled"`
	CurrentConcurrency   int       `json:"current_concurrency"`
	MinConcurrency       int       `json:"min_concurrency"`
	MaxConcurrency       int       `json:"max_concurrency"`
	BatchConcurrency     int       `json:"batch_concurrency"`
	Running              int       `json:"running"`
	RunningBatch         int       `json:"running_batch"`
	WaitingInteractive   int       `json:"waiting_interactive"`
	WaitingBatch         int       `json:"waiting_batch"`
	CooldownUntil        time.Time `json:"cooldown_until,omitempty"`
	ConsecutiveSuccesses int       `json:"consecutive_successes"`
	Completed            uint64    `json:"completed"`
	Failed               uint64    `json:"failed"`
	Slow                 uint64    `json:"slow"`
	AverageLatencyMs     int64     `json:"average_latency_ms"`
	LastSuccessAt        time.Time `json:"last_success_at,omitempty"`
	LastFailureAt        time.Time `json:"last_failure_at,omitempty"`
	LastScaleAt          time.Time `json:"last_scale_at,omitempty"`
	PeakRunning          int       `json:"peak_running"`
	PeakWaiting          int       `json:"peak_waiting"`
}

type Governor struct {
	cfg Config

	mu    sync.Mutex
	cond  *sync.Cond
	clock func() time.Time

	currentConcurrency int
	running            int
	runningBatch       int
	waitingInteractive int
	waitingBatch       int
	successes          int
	cooldownUntil      time.Time
	latencyEWMA        time.Duration
	lastSuccessAt      time.Time
	lastFailureAt      time.Time
	lastScaleAt        time.Time
	idleSince          time.Time
	completed          uint64
	failed             uint64
	slow               uint64
	peakRunning        int
	peakWaiting        int
}

type Lease struct {
	g     *Governor
	batch bool
	once  sync.Once
}

var global = New(LoadConfigFromEnv())

func LoadConfigFromEnv() Config {
	cfg := Config{
		Enabled:             envBool("EMBEDDING_GOVERNOR_ENABLED", true),
		Models:              parseCSVSet(envString("EMBEDDING_GOVERNOR_MODELS", defaultModels)),
		BatchModels:         parseCSVSet(envString("EMBEDDING_GOVERNOR_BATCH_MODELS", defaultBatchModels)),
		InitialConcurrency:  envInt("EMBEDDING_GOVERNOR_INITIAL_CONCURRENCY", defaultInitialConcurrency),
		MinConcurrency:      envInt("EMBEDDING_GOVERNOR_MIN_CONCURRENCY", defaultMinConcurrency),
		MaxConcurrency:      envInt("EMBEDDING_GOVERNOR_MAX_CONCURRENCY", defaultMaxConcurrency),
		BatchConcurrency:    envInt("EMBEDDING_GOVERNOR_BATCH_CONCURRENCY", defaultBatchConcurrency),
		QueueLimit:          envInt("EMBEDDING_GOVERNOR_QUEUE_LIMIT", defaultQueueLimit),
		BatchQueueLimit:     envInt("EMBEDDING_GOVERNOR_BATCH_QUEUE_LIMIT", defaultBatchQueueLimit),
		InteractiveTimeout:  envDuration("EMBEDDING_GOVERNOR_INTERACTIVE_TIMEOUT", 30*time.Second),
		BatchTimeout:        envDuration("EMBEDDING_GOVERNOR_BATCH_TIMEOUT", 5*time.Minute),
		Cooldown:            envDuration("EMBEDDING_GOVERNOR_COOLDOWN", 10*time.Minute),
		SlowRequestDuration: envDuration("EMBEDDING_GOVERNOR_SLOW_REQUEST_DURATION", 2*time.Minute),
		LatencyTarget:       envDuration("EMBEDDING_GOVERNOR_LATENCY_TARGET", 90*time.Second),
		ScaleUpMinInterval:  envDuration("EMBEDDING_GOVERNOR_SCALE_UP_MIN_INTERVAL", 30*time.Second),
		ScaleDownIdle:       envDuration("EMBEDDING_GOVERNOR_SCALE_DOWN_IDLE", 10*time.Minute),
		SuccessWindow:       envInt("EMBEDDING_GOVERNOR_SUCCESS_WINDOW", defaultSuccessWindow),
	}
	return normalizeConfig(cfg)
}

func New(cfg Config) *Governor {
	cfg = normalizeConfig(cfg)
	g := &Governor{
		cfg:                cfg,
		clock:              time.Now,
		currentConcurrency: cfg.InitialConcurrency,
	}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func Acquire(ctx context.Context, req Request) (*Lease, *Reject) {
	return global.Acquire(ctx, req)
}

func CurrentSnapshot() Snapshot {
	return global.Snapshot()
}

func ResetForTest(cfg Config) func() {
	previous := global
	global = New(cfg)
	return func() {
		global = previous
	}
}

func (g *Governor) Acquire(ctx context.Context, req Request) (*Lease, *Reject) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !g.applies(req.Model) {
		return nil, nil
	}

	batch := g.isBatch(req)
	timeout := g.cfg.InteractiveTimeout
	if batch {
		timeout = g.cfg.BatchTimeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	g.mu.Lock()
	if reject := g.queueCapacityRejectLocked(batch); reject != nil {
		g.mu.Unlock()
		return nil, reject
	}
	if batch {
		g.waitingBatch++
	} else {
		g.waitingInteractive++
	}
	g.observePeaksLocked()
	g.maybeScaleForDemandLocked(g.clock())
	g.mu.Unlock()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			g.mu.Lock()
			g.cond.Broadcast()
			g.mu.Unlock()
		case <-done:
		}
	}()
	defer close(done)

	g.mu.Lock()
	defer g.mu.Unlock()
	for {
		if ctx.Err() != nil {
			g.decrementWaiterLocked(batch)
			return nil, rejectFromContext(ctx.Err(), timeout)
		}
		if g.canStartLocked(batch) {
			g.decrementWaiterLocked(batch)
			g.running++
			if batch {
				g.runningBatch++
			}
			g.observePeaksLocked()
			return &Lease{g: g, batch: batch}, nil
		}
		g.cond.Wait()
	}
}

func (g *Governor) Snapshot() Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	return Snapshot{
		Enabled:              g.cfg.Enabled,
		CurrentConcurrency:   g.currentConcurrency,
		MinConcurrency:       g.cfg.MinConcurrency,
		MaxConcurrency:       g.cfg.MaxConcurrency,
		BatchConcurrency:     g.cfg.BatchConcurrency,
		Running:              g.running,
		RunningBatch:         g.runningBatch,
		WaitingInteractive:   g.waitingInteractive,
		WaitingBatch:         g.waitingBatch,
		CooldownUntil:        g.cooldownUntil,
		ConsecutiveSuccesses: g.successes,
		Completed:            g.completed,
		Failed:               g.failed,
		Slow:                 g.slow,
		AverageLatencyMs:     g.latencyEWMA.Milliseconds(),
		LastSuccessAt:        g.lastSuccessAt,
		LastFailureAt:        g.lastFailureAt,
		LastScaleAt:          g.lastScaleAt,
		PeakRunning:          g.peakRunning,
		PeakWaiting:          g.peakWaiting,
	}
}

func (l *Lease) Finish(success bool, statusCode int, latency time.Duration) {
	if l == nil || l.g == nil {
		return
	}
	l.once.Do(func() {
		l.g.finish(l.batch, success, statusCode, latency)
	})
}

func (g *Governor) finish(batch bool, success bool, statusCode int, latency time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running > 0 {
		g.running--
	}
	if batch && g.runningBatch > 0 {
		g.runningBatch--
	}

	if statusCode >= http.StatusInternalServerError {
		success = false
	}
	if g.cfg.SlowRequestDuration > 0 && latency >= g.cfg.SlowRequestDuration {
		success = false
		g.slow++
	}

	now := g.clock()
	g.latencyEWMA = blendDuration(g.latencyEWMA, latency)
	if !success {
		g.failed++
		g.currentConcurrency = g.cfg.MinConcurrency
		g.successes = 0
		g.cooldownUntil = now.Add(g.cfg.Cooldown)
		g.lastFailureAt = now
		g.lastScaleAt = now
		g.updateIdleLocked(now)
		g.cond.Broadcast()
		return
	}

	g.completed++
	g.lastSuccessAt = now
	if g.currentConcurrency < g.cfg.MaxConcurrency && g.canIncreaseLocked(now) {
		g.successes++
		if g.successes >= g.cfg.SuccessWindow {
			g.currentConcurrency++
			g.successes = 0
			g.lastScaleAt = now
		}
	}
	g.updateIdleLocked(now)
	g.cond.Broadcast()
}

func (g *Governor) applies(model string) bool {
	if !g.cfg.Enabled {
		return false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	return g.cfg.Models[model]
}

func (g *Governor) isBatch(req Request) bool {
	workload := strings.ToLower(strings.TrimSpace(req.Workload))
	if workload == "batch" || workload == "bulk" {
		return true
	}
	if workload == "interactive" || workload == "realtime" {
		return false
	}
	return g.cfg.BatchModels[strings.TrimSpace(req.Model)]
}

func (g *Governor) queueCapacityRejectLocked(batch bool) *Reject {
	waitingTotal := g.waitingInteractive + g.waitingBatch
	if g.cfg.QueueLimit > 0 && waitingTotal >= g.cfg.QueueLimit {
		return &Reject{
			StatusCode: http.StatusTooManyRequests,
			Code:       "embedding_governor_queue_full",
			Message:    "embedding governor queue is full",
			RetryAfter: g.cfg.InteractiveTimeout,
		}
	}
	if batch && g.cfg.BatchQueueLimit > 0 && g.waitingBatch >= g.cfg.BatchQueueLimit {
		return &Reject{
			StatusCode: http.StatusTooManyRequests,
			Code:       "embedding_governor_batch_queue_full",
			Message:    "embedding governor batch queue is full",
			RetryAfter: g.cfg.BatchTimeout,
		}
	}
	return nil
}

func (g *Governor) canStartLocked(batch bool) bool {
	if g.running >= g.currentConcurrency {
		return false
	}
	if !batch {
		return true
	}
	if g.waitingInteractive > 0 {
		return false
	}
	return g.runningBatch < g.cfg.BatchConcurrency
}

func (g *Governor) maybeScaleForDemandLocked(now time.Time) {
	if g.waitingInteractive == 0 {
		return
	}
	if g.running < g.currentConcurrency {
		return
	}
	if !g.canIncreaseLocked(now) {
		return
	}
	g.currentConcurrency++
	g.successes = 0
	g.lastScaleAt = now
	g.cond.Broadcast()
}

func (g *Governor) canIncreaseLocked(now time.Time) bool {
	if g.currentConcurrency >= g.cfg.MaxConcurrency {
		return false
	}
	if now.Before(g.cooldownUntil) {
		return false
	}
	if !g.lastFailureAt.IsZero() && g.cfg.Cooldown > 0 && now.Sub(g.lastFailureAt) < g.cfg.Cooldown {
		return false
	}
	if g.cfg.ScaleUpMinInterval > 0 && !g.lastScaleAt.IsZero() && now.Sub(g.lastScaleAt) < g.cfg.ScaleUpMinInterval {
		return false
	}
	if g.cfg.LatencyTarget > 0 && g.latencyEWMA > g.cfg.LatencyTarget {
		return false
	}
	return true
}

func (g *Governor) updateIdleLocked(now time.Time) {
	if g.running > 0 || g.waitingInteractive > 0 || g.waitingBatch > 0 {
		g.idleSince = time.Time{}
		return
	}
	if g.idleSince.IsZero() {
		g.idleSince = now
		return
	}
	if g.currentConcurrency <= g.cfg.MinConcurrency || g.cfg.ScaleDownIdle <= 0 {
		return
	}
	if now.Sub(g.idleSince) >= g.cfg.ScaleDownIdle {
		g.currentConcurrency--
		g.lastScaleAt = now
		g.idleSince = now
	}
}

func (g *Governor) observePeaksLocked() {
	if g.running > g.peakRunning {
		g.peakRunning = g.running
	}
	waiting := g.waitingInteractive + g.waitingBatch
	if waiting > g.peakWaiting {
		g.peakWaiting = waiting
	}
}

func (g *Governor) decrementWaiterLocked(batch bool) {
	if batch {
		if g.waitingBatch > 0 {
			g.waitingBatch--
		}
		return
	}
	if g.waitingInteractive > 0 {
		g.waitingInteractive--
	}
}

func rejectFromContext(err error, timeout time.Duration) *Reject {
	if err == context.Canceled {
		return &Reject{
			StatusCode: http.StatusRequestTimeout,
			Code:       "embedding_governor_request_canceled",
			Message:    "embedding governor request was canceled before dispatch",
		}
	}
	return &Reject{
		StatusCode: http.StatusTooManyRequests,
		Code:       "embedding_governor_queue_timeout",
		Message:    "embedding governor queue timeout before dispatch",
		RetryAfter: timeout,
	}
}

func normalizeConfig(cfg Config) Config {
	if cfg.Models == nil {
		cfg.Models = parseCSVSet(defaultModels)
	}
	if cfg.BatchModels == nil {
		cfg.BatchModels = parseCSVSet(defaultBatchModels)
	}
	if cfg.MinConcurrency < 1 {
		cfg.MinConcurrency = defaultMinConcurrency
	}
	if cfg.MaxConcurrency < cfg.MinConcurrency {
		cfg.MaxConcurrency = cfg.MinConcurrency
	}
	if cfg.InitialConcurrency < cfg.MinConcurrency {
		cfg.InitialConcurrency = cfg.MinConcurrency
	}
	if cfg.InitialConcurrency > cfg.MaxConcurrency {
		cfg.InitialConcurrency = cfg.MaxConcurrency
	}
	if cfg.BatchConcurrency < 1 {
		cfg.BatchConcurrency = defaultBatchConcurrency
	}
	if cfg.BatchConcurrency > cfg.MaxConcurrency {
		cfg.BatchConcurrency = cfg.MaxConcurrency
	}
	if cfg.QueueLimit < 1 {
		cfg.QueueLimit = defaultQueueLimit
	}
	if cfg.BatchQueueLimit < 1 {
		cfg.BatchQueueLimit = defaultBatchQueueLimit
	}
	if cfg.InteractiveTimeout <= 0 {
		cfg.InteractiveTimeout = 30 * time.Second
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 5 * time.Minute
	}
	if cfg.Cooldown < 0 {
		cfg.Cooldown = 0
	}
	if cfg.SlowRequestDuration < 0 {
		cfg.SlowRequestDuration = 0
	}
	if cfg.LatencyTarget < 0 {
		cfg.LatencyTarget = 0
	}
	if cfg.ScaleUpMinInterval < 0 {
		cfg.ScaleUpMinInterval = 0
	}
	if cfg.ScaleDownIdle < 0 {
		cfg.ScaleDownIdle = 0
	}
	if cfg.SuccessWindow < 1 {
		cfg.SuccessWindow = defaultSuccessWindow
	}
	return cfg
}

func blendDuration(current time.Duration, sample time.Duration) time.Duration {
	if sample <= 0 {
		return current
	}
	if current <= 0 {
		return sample
	}
	return time.Duration((int64(current)*7)/10 + (int64(sample) * 3 / 10))
}

func parseCSVSet(value string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		set[item] = true
	}
	return set
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
