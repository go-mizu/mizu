// Package timeout provides request timeout middleware for Mizu.
package timeout

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the timeout middleware.
type Options struct {
	// Timeout is the maximum duration for request processing.
	Timeout time.Duration

	// ErrorHandler is called when a timeout occurs.
	// It receives the ResponseWriter directly (not Ctx) because the Ctx
	// may have been partially written to by the timed-out handler.
	// If nil, returns 503 Service Unavailable with ErrorMessage.
	ErrorHandler func(w http.ResponseWriter, r *http.Request)

	// ErrorMessage is the message returned on timeout.
	// Default: "Service Unavailable".
	ErrorMessage string
}

// New creates a timeout middleware with the specified duration.
func New(timeout time.Duration) mizu.Middleware {
	return WithOptions(Options{Timeout: timeout})
}

// WithOptions creates a timeout middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.ErrorMessage == "" {
		opts.ErrorMessage = "Service Unavailable"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			ctx, cancel := context.WithTimeout(c.Context(), opts.Timeout)
			defer cancel()

			// Create new request with timeout context
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Track which path wins
			var responded int32

			// Create a response recorder for the handler
			rec := &timeoutRecorder{
				headers:    make(http.Header),
				body:       &bytes.Buffer{},
				statusCode: http.StatusOK,
				mu:         &sync.Mutex{},
			}

			// Save original writer
			origWriter := c.Writer()

			// Set the recorder BEFORE spawning the goroutine
			c.SetWriter(rec)

			// Channel to receive handler result
			done := make(chan error, 1)

			go func() {
				err := next(c)
				done <- err
			}()

			select {
			case err := <-done:
				// Handler finished first - copy response if we win
				if atomic.CompareAndSwapInt32(&responded, 0, 1) {
					rec.mu.Lock()
					for k, v := range rec.headers {
						for _, vv := range v {
							origWriter.Header().Add(k, vv)
						}
					}
					origWriter.WriteHeader(rec.statusCode)
					_, _ = origWriter.Write(rec.body.Bytes())
					rec.mu.Unlock()
				}
				return err

			case <-ctx.Done():
				// Wait for handler goroutine to complete to prevent races
				<-done

				// Write timeout response
				if atomic.CompareAndSwapInt32(&responded, 0, 1) {
					if opts.ErrorHandler != nil {
						// Call the error handler with the original writer directly
						opts.ErrorHandler(origWriter, c.Request())
					} else {
						// Default: write directly to original writer
						origWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
						origWriter.WriteHeader(http.StatusServiceUnavailable)
						_, _ = origWriter.Write([]byte(opts.ErrorMessage))
					}
				}

				// Return nil since we've already written the response
				return nil
			}
		}
	}
}

// timeoutRecorder captures response for the handler goroutine.
type timeoutRecorder struct {
	headers    http.Header
	body       *bytes.Buffer
	statusCode int
	mu         *sync.Mutex
}

func (r *timeoutRecorder) Header() http.Header {
	return r.headers
}

func (r *timeoutRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.Write(b)
}

func (r *timeoutRecorder) WriteHeader(code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusCode = code
}

// freshResponseCtx wraps a ResponseWriter with fresh wroteHeader state.
// This allows the ErrorHandler to properly write headers even though
// the handler goroutine may have already written to a different writer.
type freshResponseCtx struct {
	w           http.ResponseWriter
	status      int
	wroteHeader bool
}

func (f *freshResponseCtx) Header() http.Header {
	return f.w.Header()
}

func (f *freshResponseCtx) Write(b []byte) (int, error) {
	if !f.wroteHeader {
		f.w.WriteHeader(f.status)
		f.wroteHeader = true
	}
	return f.w.Write(b)
}

func (f *freshResponseCtx) WriteHeader(code int) {
	if !f.wroteHeader {
		f.status = code
		f.w.WriteHeader(code)
		f.wroteHeader = true
	}
}
