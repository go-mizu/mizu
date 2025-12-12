// Package bodyclose provides middleware to ensure request bodies are properly closed.
package bodyclose

import (
	"io"

	"github.com/go-mizu/mizu"
)

// Options configures the bodyclose middleware.
type Options struct {
	// DrainBody drains the body before closing.
	// Default: true.
	DrainBody bool

	// MaxDrain is the maximum bytes to drain.
	// Default: 8KB.
	MaxDrain int64
}

// New creates bodyclose middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates bodyclose middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.MaxDrain == 0 {
		opts.MaxDrain = 8 * 1024 // 8KB
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			body := c.Request().Body
			if body == nil {
				return next(c)
			}

			defer func() {
				if opts.DrainBody {
					// Drain remaining body to allow connection reuse
					_, _ = io.CopyN(io.Discard, body, opts.MaxDrain)
				}
				_ = body.Close()
			}()

			return next(c)
		}
	}
}

// Drain creates middleware that drains request bodies.
func Drain() mizu.Middleware {
	return WithOptions(Options{DrainBody: true})
}

// NoDrain creates middleware that closes without draining.
func NoDrain() mizu.Middleware {
	return WithOptions(Options{DrainBody: false})
}
