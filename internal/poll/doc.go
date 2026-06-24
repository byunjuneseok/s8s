// Package poll provides a central, broker-neutral polling scheduler for s8s.
//
// The Toss securities API exposes no streaming/websocket interface, so screens
// in the TUI must poll for fresh data on intervals (for example prices every
// 2s, orderbook every 1s, holdings every 5s). Scheduler centralises this so a
// single component owns all timers, can pause polling for screens that are not
// visible, and can back off when the upstream API rate-limits us.
//
// # Concurrency
//
// Scheduler is safe for concurrent use. All registration and lifecycle methods
// (Register, Unregister, Pause, Resume) may be called from multiple goroutines,
// including from inside a job callback, while the scheduler is running.
//
// # Rate limiting
//
// Scheduler is deliberately broker-neutral: it does not import any broker,
// domain or UI package and depends only on the standard library. To stay aware
// of upstream rate limits without coupling to a specific transport, callbacks
// signal throttling by returning an error that wraps the sentinel ErrThrottled.
// Adapters are expected to map an HTTP 429 response (or a dangerously low
// remaining-limit header) to ErrThrottled, for example:
//
//	if resp.StatusCode == http.StatusTooManyRequests {
//		return fmt.Errorf("toss prices: %w", poll.ErrThrottled)
//	}
//
// When a callback returns ErrThrottled the scheduler grows that job's effective
// interval exponentially (base, base*factor, base*factor^2, ...) up to a
// configurable cap. The first successful (non-throttled) call resets the job
// back to its base interval.
package poll
