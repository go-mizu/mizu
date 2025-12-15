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
	// Default: retry on 5xx responses or on non-nil error.
	RetryIf func(c *mizu.Ctx, err error, attempt int) bool

	// OnRetry is called before each retry (attempt starts at 1 for first retry).
	OnRetry func(c *mizu.Ctx, err error, attempt int)
}

// New creates retry middleware with default options.
func New() mizu.Middleware { return WithOptions(Options{}) }

// WithOptions creates retry middleware with custom options.
//
//nolint:cyclop // retry logic needs multiple checks
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
			orig := c.Writer()
			delay := opts.Delay

			var lastErr error
			var lastRW *retryResponseWriter

			for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
				if attempt > 0 {
					// Backoff before retry.
					time.Sleep(delay)

					delay = time.Duration(float64(delay) * opts.Multiplier)
					if delay > opts.MaxDelay {
						delay = opts.MaxDelay
					}

					if opts.OnRetry != nil {
						opts.OnRetry(c, lastErr, attempt)
					}
				}

				// Buffer this attempt's output.
				rw := newRetryResponseWriter(orig)
				lastRW = rw
				c.SetWriter(rw)

				lastErr = next(c)

				// Decide if we should retry.
				if !opts.RetryIf(c, lastErr, attempt) {
					// Commit response once.
					rw.flushTo(orig)
					c.SetWriter(orig)
					return lastErr
				}

				// If handler produced a successful (non-5xx) status, don't retry.
				// Note: status==0 means handler wrote nothing; treat as "unknown" and allow RetryIf.
				if rw.status > 0 && rw.status < 500 {
					rw.flushTo(orig)
					c.SetWriter(orig)
					return lastErr
				}

				// Retry: discard buffered output and continue.
				c.SetWriter(orig)
			}

			// Out of retries: commit the last attempt's buffered output (if any).
			if lastRW != nil {
				lastRW.flushTo(orig)
			}
			c.SetWriter(orig)
			return lastErr
		}
	}
}

// retryResponseWriter buffers headers/status/body for an attempt.
// Nothing is written to the underlying ResponseWriter until flushTo.
type retryResponseWriter struct {
	base   http.ResponseWriter
	header http.Header
	status int
	body   []byte
}

func newRetryResponseWriter(base http.ResponseWriter) *retryResponseWriter {
	return &retryResponseWriter{
		base:   base,
		header: make(http.Header),
	}
}

func (w *retryResponseWriter) Header() http.Header { return w.header }

func (w *retryResponseWriter) WriteHeader(code int) { w.status = code }

func (w *retryResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *retryResponseWriter) flushTo(dst http.ResponseWriter) {
	// Copy headers (overwrite keys in destination).
	for k, v := range w.header {
		dst.Header()[k] = append([]string(nil), v...)
	}

	if w.status != 0 {
		dst.WriteHeader(w.status)
	}
	if len(w.body) > 0 {
		_, _ = dst.Write(w.body)
	}
}

func defaultRetryIf(c *mizu.Ctx, err error, attempt int) bool {
	if err != nil {
		return true
	}

	// If writer is our buffered writer, retry on 5xx.
	if rw, ok := c.Writer().(*retryResponseWriter); ok {
		return rw.status >= 500
	}

	return false
}

// RetryOn creates a RetryIf function for specific status codes.
func RetryOn(codes ...int) func(*mizu.Ctx, error, int) bool {
	codeMap := make(map[int]bool, len(codes))
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

// RetryOnError retries only when err != nil.
func RetryOnError() func(*mizu.Ctx, error, int) bool {
	return func(c *mizu.Ctx, err error, attempt int) bool {
		return err != nil
	}
}

// NoRetry never retries.
func NoRetry() func(*mizu.Ctx, error, int) bool {
	return func(c *mizu.Ctx, err error, attempt int) bool {
		return false
	}
}
