// Package recover provides panic recovery middleware for Mizu.
package recover

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-mizu/mizu"
)

// Options configures the recovery middleware.
type Options struct {
	// StackSize is the buffer size for capturing stack traces.
	// Default: 4096 bytes.
	StackSize int

	// DisableStackAll limits stack trace to current goroutine only.
	// Note: debug.Stack() only captures current goroutine.
	DisableStackAll bool

	// DisablePrintStack disables logging of stack traces.
	DisablePrintStack bool

	// ErrorHandler is called when a panic is recovered.
	// If nil, a 500 Internal Server Error is returned.
	ErrorHandler func(c *mizu.Ctx, err any, stack []byte) error

	// Logger is used for logging panics.
	// If nil, c.Logger() is used.
	Logger *slog.Logger
}

// New creates a recovery middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates a recovery middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.StackSize <= 0 {
		opts.StackSize = 4096
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					var stack []byte
					if !opts.DisablePrintStack {
						stack = debug.Stack()
						if len(stack) > opts.StackSize {
							stack = stack[:opts.StackSize]
						}
					}

					log := opts.Logger
					if log == nil {
						log = c.Logger()
					}

					if !opts.DisablePrintStack && log != nil {
						log.Error("panic recovered",
							slog.Any("error", r),
							slog.String("stack", string(stack)),
						)
					}

					if opts.ErrorHandler != nil {
						err = opts.ErrorHandler(c, r, stack)
						return
					}

					err = c.Text(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
				}
			}()

			return next(c)
		}
	}
}
