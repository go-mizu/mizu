// Package responselog provides response body logging middleware for Mizu.
package responselog

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the responselog middleware.
type Options struct {
	// Logger is the slog logger to use.
	Logger *slog.Logger

	// LogBody logs response body.
	// Default: true.
	LogBody bool

	// LogHeaders logs response headers.
	// Default: false.
	LogHeaders bool

	// MaxBodySize is the max body size to log.
	// Default: 4KB.
	MaxBodySize int64

	// SkipPaths are paths to skip logging.
	SkipPaths []string

	// SkipStatuses are status codes to skip logging.
	SkipStatuses []int
}

// responseCapture captures response for logging.
type responseCapture struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	maxSize    int64
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	// Capture up to maxSize
	if r.body.Len() < int(r.maxSize) {
		remaining := int(r.maxSize) - r.body.Len()
		if len(b) > remaining {
			r.body.Write(b[:remaining])
		} else {
			r.body.Write(b)
		}
	}
	return r.ResponseWriter.Write(b)
}

// New creates responselog middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates responselog middleware with custom options.
//
//nolint:cyclop // Response logging requires multiple field extraction checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if opts.MaxBodySize == 0 {
		opts.MaxBodySize = 4 * 1024
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	skipStatuses := make(map[int]bool)
	for _, s := range opts.SkipStatuses {
		skipStatuses[s] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			start := time.Now()

			// Capture response
			capture := &responseCapture{
				ResponseWriter: c.Writer(),
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
				maxSize:        opts.MaxBodySize,
			}
			c.SetWriter(capture)

			err := next(c)

			// Restore writer
			c.SetWriter(capture.ResponseWriter)

			// Skip configured statuses
			if skipStatuses[capture.statusCode] {
				return err
			}

			duration := time.Since(start)

			attrs := []any{
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", capture.statusCode,
				"duration", duration.String(),
				"size", capture.body.Len(),
			}

			// Log headers
			if opts.LogHeaders {
				headers := make(map[string]string)
				for name, values := range c.Header() {
					if len(values) > 0 {
						headers[name] = values[0]
					}
				}
				attrs = append(attrs, "headers", headers)
			}

			// Log body
			if opts.LogBody && capture.body.Len() > 0 {
				attrs = append(attrs, "body", capture.body.String())
			}

			if capture.statusCode >= 400 {
				opts.Logger.Error("response", attrs...)
			} else {
				opts.Logger.Info("response", attrs...)
			}

			return err
		}
	}
}

// WithLogger creates middleware with a specific logger.
func WithLogger(logger *slog.Logger) mizu.Middleware {
	return WithOptions(Options{Logger: logger})
}

// Full creates middleware that logs everything.
func Full() mizu.Middleware {
	return WithOptions(Options{
		LogBody:    true,
		LogHeaders: true,
	})
}

// ErrorsOnly creates middleware that only logs error responses.
func ErrorsOnly() mizu.Middleware {
	return WithOptions(Options{
		LogBody:      true,
		SkipStatuses: []int{200, 201, 204, 301, 302, 304},
	})
}
