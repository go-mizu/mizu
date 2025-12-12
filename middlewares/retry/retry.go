// Package retry provides request retry middleware for Mizu.
package retry

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the retry middleware.
type Options struct {
	// MaxRetries is the maximum number of retries.
	// Default: 3.
	MaxRetries int

	// Delay is the initial delay between retries.
	// Default: 100ms.
	Delay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Default: 1s.
	MaxDelay time.Duration

	// Multiplier is the delay multiplier for exponential backoff.
	// Default: 2.0.
	Multiplier float64

	// RetryIf determines if a request should be retried.
	// Default: retry on 5xx errors.
	RetryIf func(c *mizu.Ctx, err error, attempt int) bool

	// OnRetry is called before each retry.
	OnRetry func(c *mizu.Ctx, err error, attempt int)
}

// New creates retry middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates retry middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.Delay == 0 {
		opts.Delay = 100 * time.Millisecond
	}
	if opts.MaxDelay == 0 {
		opts.MaxDelay = time.Second
	}
	if opts.Multiplier == 0 {
		opts.Multiplier = 2.0
	}
	if opts.RetryIf == nil {
		opts.RetryIf = defaultRetryIf
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var lastErr error
			delay := opts.Delay

			for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
				if attempt > 0 {
					// Wait before retry
					time.Sleep(delay)

					// Increase delay for next attempt
					delay = time.Duration(float64(delay) * opts.Multiplier)
					if delay > opts.MaxDelay {
						delay = opts.MaxDelay
					}

					// Call OnRetry callback
					if opts.OnRetry != nil {
						opts.OnRetry(c, lastErr, attempt)
					}
				}

				// Reset response writer for retry
				rw := &retryResponseWriter{
					ResponseWriter: c.Writer(),
					status:         0,
				}
				c.SetWriter(rw)

				lastErr = next(c)

				// Check if we should retry
				if !opts.RetryIf(c, lastErr, attempt) {
					return lastErr
				}

				// Don't retry if response was already sent with success
				if rw.status > 0 && rw.status < 500 {
					return lastErr
				}
			}

			return lastErr
		}
	}
}

type retryResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *retryResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *retryResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func defaultRetryIf(c *mizu.Ctx, err error, attempt int) bool {
	// Retry on errors
	if err != nil {
		return true
	}

	// Check response status from writer
	if rw, ok := c.Writer().(*retryResponseWriter); ok {
		return rw.status >= 500
	}

	return false
}

// RetryOn creates a RetryIf function for specific status codes.
func RetryOn(codes ...int) func(*mizu.Ctx, error, int) bool {
	codeMap := make(map[int]bool)
	for _, code := range codes {
		codeMap[code] = true
	}

	return func(c *mizu.Ctx, err error, attempt int) bool {
		if err != nil {
			return true
		}
		if rw, ok := c.Writer().(*retryResponseWriter); ok {
			return codeMap[rw.status]
		}
		return false
	}
}

// RetryOnError creates a RetryIf function that only retries on errors.
func RetryOnError() func(*mizu.Ctx, error, int) bool {
	return func(c *mizu.Ctx, err error, attempt int) bool {
		return err != nil
	}
}

// NoRetry creates a RetryIf function that never retries.
func NoRetry() func(*mizu.Ctx, error, int) bool {
	return func(c *mizu.Ctx, err error, attempt int) bool {
		return false
	}
}
