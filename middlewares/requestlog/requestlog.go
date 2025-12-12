// Package requestlog provides detailed request logging middleware for Mizu.
package requestlog

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the requestlog middleware.
type Options struct {
	// Logger is the slog logger to use.
	Logger *slog.Logger

	// LogHeaders logs request headers.
	// Default: false.
	LogHeaders bool

	// LogBody logs request body.
	// Default: false.
	LogBody bool

	// MaxBodySize is the max body size to log.
	// Default: 4KB.
	MaxBodySize int64

	// SkipPaths are paths to skip logging.
	SkipPaths []string

	// SkipMethods are methods to skip logging.
	SkipMethods []string

	// Sensitive headers to redact.
	SensitiveHeaders []string
}

// New creates requestlog middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates requestlog middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if opts.MaxBodySize == 0 {
		opts.MaxBodySize = 4 * 1024
	}
	if len(opts.SensitiveHeaders) == 0 {
		opts.SensitiveHeaders = []string{"Authorization", "Cookie", "X-API-Key"}
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	skipMethods := make(map[string]bool)
	for _, m := range opts.SkipMethods {
		skipMethods[m] = true
	}

	sensitiveHeaders := make(map[string]bool)
	for _, h := range opts.SensitiveHeaders {
		sensitiveHeaders[h] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured paths/methods
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}
			if skipMethods[c.Request().Method] {
				return next(c)
			}

			start := time.Now()
			attrs := []any{
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"remote_addr", c.Request().RemoteAddr,
			}

			// Log query params
			if query := c.Request().URL.RawQuery; query != "" {
				attrs = append(attrs, "query", query)
			}

			// Log headers
			if opts.LogHeaders {
				headers := make(map[string]string)
				for name, values := range c.Request().Header {
					if sensitiveHeaders[name] {
						headers[name] = "[REDACTED]"
					} else if len(values) > 0 {
						headers[name] = values[0]
					}
				}
				attrs = append(attrs, "headers", headers)
			}

			// Log body
			if opts.LogBody && c.Request().Body != nil {
				body, _ := io.ReadAll(io.LimitReader(c.Request().Body, opts.MaxBodySize))
				c.Request().Body = io.NopCloser(bytes.NewReader(body))
				if len(body) > 0 {
					attrs = append(attrs, "body", string(body))
				}
			}

			opts.Logger.Info("request",
				attrs...,
			)

			err := next(c)

			duration := time.Since(start)
			attrs = append(attrs, "duration", duration.String())

			if err != nil {
				opts.Logger.Error("request error",
					append(attrs, "error", err.Error())...,
				)
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
		LogHeaders: true,
		LogBody:    true,
	})
}

// HeadersOnly creates middleware that logs headers.
func HeadersOnly() mizu.Middleware {
	return WithOptions(Options{LogHeaders: true})
}

// BodyOnly creates middleware that logs body.
func BodyOnly() mizu.Middleware {
	return WithOptions(Options{LogBody: true})
}
