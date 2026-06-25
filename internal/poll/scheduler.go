package poll

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrThrottled is the sentinel error a job callback returns (directly or
// wrapped) to signal that the upstream API rate-limited the request. The
// scheduler reacts by backing off that job's effective interval exponentially.
//
// Adapters should map transport-level throttling (HTTP 429, or a dangerously
// low remaining-limit) onto this sentinel using fmt.Errorf("...: %w", ...) so
// that errors.Is(err, ErrThrottled) reports true.
var ErrThrottled = errors.New("poll: request throttled by upstream")

// Defaults applied when the corresponding option is not supplied.
const (
	// DefaultBackoffFactor is the multiplier applied to a job's interval for
	// each consecutive throttled call.
	DefaultBackoffFactor = 2.0
	// DefaultMaxBackoff caps how far a single job's interval may grow while
	// it is being throttled.
	DefaultMaxBackoff = 30 * time.Second
)

// Ticker is the minimal subset of time.Ticker the scheduler relies on. Tests
// supply a fake implementation to drive time deterministically.
type Ticker interface {
	// C returns the channel on which ticks are delivered.
	C() <-chan time.Time
	// Reset changes the ticker to fire with the given period. It mirrors
	// time.Ticker.Reset and must only be called with d > 0.
	Reset(d time.Duration)
	// Stop releases the ticker's resources. After Stop no further ticks are
	// delivered.
	Stop()
}

// TickerFactory creates a Ticker that fires with the given period. The default
// factory wraps time.NewTicker; tests inject a factory that yields fakes.
type TickerFactory func(d time.Duration) Ticker

// realTicker adapts *time.Ticker to the Ticker interface.
type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time   { return r.t.C }
func (r realTicker) Reset(d time.Duration) { r.t.Reset(d) }
func (r realTicker) Stop()                 { r.t.Stop() }

func newRealTicker(d time.Duration) Ticker { return realTicker{t: time.NewTicker(d)} }

// Option configures a Scheduler at construction time.
type Option func(*Scheduler)

// WithBackoffFactor sets the exponential backoff multiplier applied per
// consecutive throttled call. Values <= 1 are ignored in favour of the
// default, since a factor of 1 or less would never back off.
func WithBackoffFactor(factor float64) Option {
	return func(s *Scheduler) {
		if factor > 1 {
			s.backoffFactor = factor
		}
	}
}

// WithMaxBackoff caps how far any single job's interval may grow while it is
// being throttled. Non-positive values are ignored in favour of the default.
func WithMaxBackoff(d time.Duration) Option {
	return func(s *Scheduler) {
		if d > 0 {
			s.maxBackoff = d
		}
	}
}

// WithTickerFactory overrides how the scheduler obtains tickers. It exists for
// deterministic testing; production code should rely on the default.
func WithTickerFactory(f TickerFactory) Option {
	return func(s *Scheduler) {
		if f != nil {
			s.newTicker = f
		}
	}
}

// job holds the registration and mutable runtime state for a single poller.
type job struct {
	name     string
	base     time.Duration
	fn       func(context.Context) error
	paused   bool
	throttle int // consecutive throttled calls; 0 when healthy

	cancel context.CancelFunc // stops this job's goroutine
}

// Scheduler runs registered jobs on their own intervals. The zero value is not
// usable; construct one with New. Scheduler is safe for concurrent use.
type Scheduler struct {
	backoffFactor float64
	maxBackoff    time.Duration
	newTicker     TickerFactory

	mu       sync.Mutex
	jobs     map[string]*job
	ctx      context.Context    // root context once Start has been called
	cancel   context.CancelFunc // cancels the root context (and all jobs)
	started  bool
	stopped  bool      // rejects further registration (Stop or ctx cancellation)
	stopOnce sync.Once // ensures the drain in Stop runs exactly once
	wg       sync.WaitGroup
}

// New constructs a Scheduler. Jobs may be registered before or after Start.
func New(opts ...Option) *Scheduler {
	s := &Scheduler{
		backoffFactor: DefaultBackoffFactor,
		maxBackoff:    DefaultMaxBackoff,
		newTicker:     newRealTicker,
		jobs:          make(map[string]*job),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Register adds (or replaces) a job that invokes fn every interval. The job
// starts active. Registering a name that already exists replaces the previous
// job, stopping its goroutine first. interval must be > 0; a non-positive
// interval is a no-op. If the scheduler is already running the new job begins
// firing immediately; if it has been stopped, registration is a no-op.
func (s *Scheduler) Register(name string, interval time.Duration, fn func(context.Context) error) {
	if interval <= 0 || fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	if existing, ok := s.jobs[name]; ok && existing.cancel != nil {
		existing.cancel()
	}
	j := &job{name: name, base: interval, fn: fn}
	s.jobs[name] = j
	if s.started {
		s.launchLocked(j)
	}
}

// Unregister removes a job and stops its goroutine. Unknown names are ignored.
func (s *Scheduler) Unregister(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[name]; ok {
		if j.cancel != nil {
			j.cancel()
		}
		delete(s.jobs, name)
	}
}

// Pause marks a job inactive so its callback stops firing. The timer keeps
// running but ticks are skipped, so Resume takes effect within one interval.
// Unknown names are ignored.
func (s *Scheduler) Pause(name string) { s.setPaused(name, true) }

// Resume marks a previously paused job active again. Unknown names are ignored.
func (s *Scheduler) Resume(name string) { s.setPaused(name, false) }

func (s *Scheduler) setPaused(name string, paused bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[name]; ok {
		j.paused = paused
	}
}

// Start begins running all registered jobs and any registered later, until ctx
// is cancelled or Stop is called. Calling Start more than once is a no-op.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || s.stopped {
		return
	}
	rootCtx, cancel := context.WithCancel(ctx)
	s.ctx = rootCtx
	s.cancel = cancel
	s.started = true

	// When the caller's context is cancelled, flip the scheduler into the
	// stopped state so that late registrations are rejected. Job goroutines
	// observe rootCtx.Done() directly and drain on their own; this watchdog
	// is not tracked by s.wg so Stop() never waits on it.
	go func() {
		<-rootCtx.Done()
		s.mu.Lock()
		s.stopped = true
		s.mu.Unlock()
	}()

	for _, j := range s.jobs {
		s.launchLocked(j)
	}
}

// Stop cancels every running job and blocks until all goroutines have drained.
// It is safe to call Stop more than once and safe to call without Start.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		// Cancelling the root context cancels every job (their contexts derive
		// from it) and releases the watchdog goroutine started in Start.
		if s.cancel != nil {
			s.cancel()
		}
		for _, j := range s.jobs {
			if j.cancel != nil {
				j.cancel()
			}
		}
		s.mu.Unlock()
		s.wg.Wait()
	})
}

// launchLocked starts the goroutine for j. The caller must hold s.mu.
func (s *Scheduler) launchLocked(j *job) {
	jobCtx, cancel := context.WithCancel(s.ctx)
	j.cancel = cancel
	s.wg.Add(1)
	go s.run(jobCtx, j)
}

// run drives a single job's ticker loop until its context is cancelled.
func (s *Scheduler) run(ctx context.Context, j *job) {
	defer s.wg.Done()

	ticker := s.newTicker(j.base)
	defer ticker.Stop()
	current := j.base

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			if !s.shouldFire(j) {
				continue
			}
			err := j.fn(ctx)
			// Recompute the interval from the (possibly mutated) throttle
			// state and reset the ticker only when it actually changed.
			next := s.applyResult(j, err)
			if next != current {
				ticker.Reset(next)
				current = next
			}
		}
	}
}

// shouldFire reports whether j is currently active (not paused).
func (s *Scheduler) shouldFire(j *job) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !j.paused
}

// applyResult updates j's throttle counter based on err and returns the job's
// new effective interval.
func (s *Scheduler) applyResult(j *job, err error) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if errors.Is(err, ErrThrottled) {
		j.throttle++
	} else {
		j.throttle = 0
	}
	return nextInterval(j.base, s.maxBackoff, s.backoffFactor, j.throttle)
}

// nextInterval computes a job's effective interval given its base interval, the
// backoff cap, the per-step factor, and the number of consecutive throttled
// calls. With consecutiveThrottles == 0 it returns base; otherwise it returns
// base * factor^consecutiveThrottles, clamped to max. It is a pure function so
// that backoff policy can be unit-tested in isolation.
func nextInterval(base, maxInterval time.Duration, factor float64, consecutiveThrottles int) time.Duration {
	if consecutiveThrottles <= 0 {
		return base
	}
	d := float64(base)
	for i := 0; i < consecutiveThrottles; i++ {
		d *= factor
		if d >= float64(maxInterval) {
			return maxInterval
		}
	}
	if d >= float64(maxInterval) {
		return maxInterval
	}
	return time.Duration(d)
}
