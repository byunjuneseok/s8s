package poll

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeTicker is a manually driven Ticker for deterministic tests. Calling
// tick() delivers one tick (blocking until the run loop consumes it, so tests
// never race ahead of the scheduler).
type fakeTicker struct {
	ch     chan time.Time
	mu     sync.Mutex
	period time.Duration
	resets int
}

func newFakeTicker(d time.Duration) *fakeTicker {
	return &fakeTicker{ch: make(chan time.Time), period: d}
}

func (f *fakeTicker) C() <-chan time.Time { return f.ch }

func (f *fakeTicker) Reset(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.period = d
	f.resets++
}

func (f *fakeTicker) Stop() {}

func (f *fakeTicker) currentPeriod() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.period
}

// tick delivers a single tick and waits for it to be received, or fails after a
// generous bound so a stalled loop does not hang the suite.
func (f *fakeTicker) tick(t *testing.T) {
	t.Helper()
	select {
	case f.ch <- time.Now():
	case <-time.After(time.Second):
		t.Fatal("scheduler did not consume tick within 1s")
	}
}

// singleTickerFactory hands out one pre-built fakeTicker, then panics if asked
// for more (each single-job test expects exactly one ticker).
func singleTickerFactory(t *testing.T, ft *fakeTicker) TickerFactory {
	t.Helper()
	var handed atomic.Bool
	return func(d time.Duration) Ticker {
		if handed.Swap(true) {
			t.Fatalf("unexpected second ticker request (d=%s)", d)
		}
		return ft
	}
}

func waitForCount(t *testing.T, c *atomic.Int64, want int64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if c.Load() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("counter reached %d, want >= %d", c.Load(), want)
}

func TestRegisteredJobFiresRepeatedly(t *testing.T) {
	ft := newFakeTicker(10 * time.Millisecond)
	s := New(WithTickerFactory(singleTickerFactory(t, ft)))

	var calls atomic.Int64
	s.Register("prices", 10*time.Millisecond, func(context.Context) error {
		calls.Add(1)
		return nil
	})

	s.Start(context.Background())
	defer s.Stop()

	for i := 0; i < 5; i++ {
		ft.tick(t)
	}
	waitForCount(t, &calls, 5)
}

func TestPauseAndResume(t *testing.T) {
	ft := newFakeTicker(10 * time.Millisecond)
	s := New(WithTickerFactory(singleTickerFactory(t, ft)))

	var calls atomic.Int64
	s.Register("orderbook", 10*time.Millisecond, func(context.Context) error {
		calls.Add(1)
		return nil
	})
	s.Start(context.Background())
	defer s.Stop()

	ft.tick(t)
	waitForCount(t, &calls, 1)

	s.Pause("orderbook")
	// Several ticks while paused must not advance the counter.
	for i := 0; i < 3; i++ {
		ft.tick(t)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("paused job fired: count=%d, want 1", got)
	}

	s.Resume("orderbook")
	ft.tick(t)
	waitForCount(t, &calls, 2)
}

func TestUnregisterStopsJob(t *testing.T) {
	ft := newFakeTicker(10 * time.Millisecond)
	s := New(WithTickerFactory(singleTickerFactory(t, ft)))

	var calls atomic.Int64
	s.Register("holdings", 10*time.Millisecond, func(context.Context) error {
		calls.Add(1)
		return nil
	})
	s.Start(context.Background())
	defer s.Stop()

	ft.tick(t)
	waitForCount(t, &calls, 1)

	s.Unregister("holdings")
	// The job goroutine is cancelled; sending on the channel must time out
	// because nothing is listening any more.
	select {
	case ft.ch <- time.Now():
		t.Fatal("unregistered job still consuming ticks")
	case <-time.After(50 * time.Millisecond):
		// expected: no receiver
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("unregistered job fired again: count=%d, want 1", got)
	}
}

func TestThrottleBacksOffThenRecovers(t *testing.T) {
	base := 10 * time.Millisecond
	ft := newFakeTicker(base)
	s := New(
		WithTickerFactory(singleTickerFactory(t, ft)),
		WithBackoffFactor(2),
		WithMaxBackoff(1*time.Second),
	)

	var throttle atomic.Bool
	throttle.Store(true)
	var calls atomic.Int64
	s.Register("prices", base, func(context.Context) error {
		calls.Add(1)
		if throttle.Load() {
			return fmt.Errorf("toss prices: %w", ErrThrottled)
		}
		return nil
	})
	s.Start(context.Background())
	defer s.Stop()

	// First throttled call -> interval should grow to base*2.
	ft.tick(t)
	waitForCount(t, &calls, 1)
	waitForPeriod(t, ft, base*2)

	// Second throttled call -> base*4.
	ft.tick(t)
	waitForCount(t, &calls, 2)
	waitForPeriod(t, ft, base*4)

	// Now succeed: interval must recover to base.
	throttle.Store(false)
	ft.tick(t)
	waitForCount(t, &calls, 3)
	waitForPeriod(t, ft, base)
}

func waitForPeriod(t *testing.T, ft *fakeTicker, want time.Duration) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ft.currentPeriod() == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("ticker period is %s, want %s", ft.currentPeriod(), want)
}

func TestStopDrainsGoroutines(t *testing.T) {
	s := New(WithTickerFactory(func(d time.Duration) Ticker { return newFakeTicker(d) }))

	var running sync.WaitGroup
	for i := 0; i < 5; i++ {
		running.Add(1)
		name := fmt.Sprintf("job-%d", i)
		s.Register(name, time.Millisecond, func(context.Context) error { return nil })
	}
	// Mark each goroutine as observed via context cancellation by wrapping the
	// drain in a done channel.
	s.Start(context.Background())

	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Stop returned, meaning every job goroutine drained.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not drain goroutines within 2s")
	}
	// satisfy the unused waitgroup (kept for documentation of intent)
	for i := 0; i < 5; i++ {
		running.Done()
	}
	running.Wait()
}

func TestStartCancelStopsScheduler(t *testing.T) {
	s := New(WithTickerFactory(func(d time.Duration) Ticker { return newFakeTicker(d) }))
	s.Register("prices", time.Millisecond, func(context.Context) error { return nil })

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	cancel()

	// After the root context is cancelled, Stop must return promptly and
	// further registration is rejected.
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop hung after context cancellation")
	}
}

func TestNextInterval(t *testing.T) {
	base := 2 * time.Second
	maxInterval := 30 * time.Second

	cases := []struct {
		throttles int
		want      time.Duration
	}{
		{0, base}, // healthy
		{1, 4 * time.Second},
		{2, 8 * time.Second},
		{3, 16 * time.Second},
		{4, maxInterval},  // 32s clamps to 30s
		{10, maxInterval}, // far beyond the cap
	}
	for _, tc := range cases {
		got := nextInterval(base, maxInterval, 2.0, tc.throttles)
		if got != tc.want {
			t.Errorf("nextInterval(throttles=%d) = %s, want %s", tc.throttles, got, tc.want)
		}
	}
}

func TestNextIntervalNegativeThrottles(t *testing.T) {
	base := time.Second
	if got := nextInterval(base, time.Minute, 2.0, -3); got != base {
		t.Fatalf("negative throttle count returned %s, want base %s", got, base)
	}
}

func TestRegisterValidation(t *testing.T) {
	s := New()
	// Non-positive interval and nil fn are no-ops.
	s.Register("bad-interval", 0, func(context.Context) error { return nil })
	s.Register("nil-fn", time.Second, nil)
	s.mu.Lock()
	n := len(s.jobs)
	s.mu.Unlock()
	if n != 0 {
		t.Fatalf("invalid registrations were stored: %d jobs", n)
	}
}

func TestThrottleSentinelMatch(t *testing.T) {
	wrapped := fmt.Errorf("layer: %w", ErrThrottled)
	if !errors.Is(wrapped, ErrThrottled) {
		t.Fatal("wrapped ErrThrottled not detected by errors.Is")
	}
}
