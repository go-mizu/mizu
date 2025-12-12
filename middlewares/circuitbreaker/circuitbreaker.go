// Package circuitbreaker provides circuit breaker pattern middleware for Mizu.
package circuitbreaker

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed allows requests through.
	StateClosed State = iota
	// StateOpen blocks all requests.
	StateOpen
	// StateHalfOpen allows limited requests for testing.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Options configures the circuit breaker middleware.
type Options struct {
	// Threshold is the number of failures before opening.
	// Default: 5.
	Threshold int

	// Timeout is the time before transitioning to half-open.
	// Default: 30s.
	Timeout time.Duration

	// MaxRequests is the number of requests allowed in half-open.
	// Default: 1.
	MaxRequests int

	// OnStateChange is called when state changes.
	OnStateChange func(from, to State)

	// IsFailure determines if an error should count as failure.
	// Default: all errors are failures.
	IsFailure func(err error) bool

	// ErrorHandler handles requests when circuit is open.
	ErrorHandler func(c *mizu.Ctx) error
}

type circuitBreaker struct {
	mu          sync.Mutex
	state       State
	failures    int
	successes   int
	lastFailure time.Time
	opts        Options
}

// New creates a circuit breaker with default settings.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates a circuit breaker with custom settings.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Threshold <= 0 {
		opts.Threshold = 5
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MaxRequests <= 0 {
		opts.MaxRequests = 1
	}
	if opts.IsFailure == nil {
		opts.IsFailure = func(err error) bool { return err != nil }
	}

	cb := &circuitBreaker{
		state: StateClosed,
		opts:  opts,
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if !cb.allowRequest() {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				return c.Text(http.StatusServiceUnavailable, "Service Unavailable")
			}

			err := next(c)
			cb.recordResult(err)
			return err
		}
	}
}

func (cb *circuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailure) > cb.opts.Timeout {
			cb.setState(StateHalfOpen)
			cb.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successes < cb.opts.MaxRequests
	}
	return false
}

func (cb *circuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.opts.IsFailure(err) {
		cb.failures++
		cb.lastFailure = time.Now()

		switch cb.state {
		case StateClosed:
			if cb.failures >= cb.opts.Threshold {
				cb.setState(StateOpen)
			}
		case StateHalfOpen:
			cb.setState(StateOpen)
		}
	} else {
		switch cb.state {
		case StateClosed:
			cb.failures = 0
		case StateHalfOpen:
			cb.successes++
			if cb.successes >= cb.opts.MaxRequests {
				cb.failures = 0
				cb.setState(StateClosed)
			}
		}
	}
}

func (cb *circuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}
	old := cb.state
	cb.state = state
	if cb.opts.OnStateChange != nil {
		cb.opts.OnStateChange(old, state)
	}
}

// GetState returns the current state (for testing/monitoring).
func (cb *circuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
