// Package fallback provides fallback response middleware for Mizu.
package fallback

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Options configures the fallback middleware.
type Options struct {
	// Handler is the fallback handler for errors.
	Handler func(c *mizu.Ctx, err error) error

	// NotFoundHandler handles 404 errors.
	NotFoundHandler func(c *mizu.Ctx) error

	// StatusCodes maps status codes to handlers.
	StatusCodes map[int]func(c *mizu.Ctx) error

	// CatchPanic catches panics and handles them.
	// Default: false.
	CatchPanic bool

	// DefaultMessage is shown for unhandled errors.
	// Default: "An error occurred".
	DefaultMessage string
}

// responseCapture captures the response status code.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *responseCapture) WriteHeader(code int) {
	if !r.written {
		r.statusCode = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	if !r.written {
		r.statusCode = http.StatusOK
		r.written = true
	}
	return r.ResponseWriter.Write(b)
}

// New creates fallback middleware with a default handler.
func New(handler func(c *mizu.Ctx, err error) error) mizu.Middleware {
	return WithOptions(Options{Handler: handler})
}

// WithOptions creates fallback middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.DefaultMessage == "" {
		opts.DefaultMessage = "An error occurred"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) (returnErr error) {
			// Catch panics if configured
			if opts.CatchPanic {
				defer func() {
					if r := recover(); r != nil {
						var err error
						if e, ok := r.(error); ok {
							err = e
						} else {
							err = &panicError{value: r}
						}
						if opts.Handler != nil {
							returnErr = opts.Handler(c, err)
						} else {
							returnErr = c.Text(http.StatusInternalServerError, opts.DefaultMessage)
						}
					}
				}()
			}

			// Execute handler
			err := next(c)

			// Handle errors
			if err != nil {
				if opts.Handler != nil {
					return opts.Handler(c, err)
				}
				return c.Text(http.StatusInternalServerError, opts.DefaultMessage)
			}

			return nil
		}
	}
}

type panicError struct {
	value any
}

func (e *panicError) Error() string {
	if err, ok := e.value.(error); ok {
		return err.Error()
	}
	return "panic occurred"
}

// NotFound creates middleware that handles 404 responses.
func NotFound(handler func(c *mizu.Ctx) error) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Capture response
			capture := &responseCapture{
				ResponseWriter: c.Writer(),
				statusCode:     http.StatusOK,
			}

			// Create new context with captured response
			originalWriter := c.Writer()
			c.SetWriter(capture)

			err := next(c)

			// Restore original writer
			c.SetWriter(originalWriter)

			// Check for 404
			if capture.statusCode == http.StatusNotFound || err != nil {
				return handler(c)
			}

			return nil
		}
	}
}

// ForStatus creates middleware that handles specific status codes.
func ForStatus(code int, handler func(c *mizu.Ctx) error) mizu.Middleware {
	return WithOptions(Options{
		StatusCodes: map[int]func(c *mizu.Ctx) error{
			code: handler,
		},
	})
}

// Default creates middleware with a simple text fallback.
func Default(message string) mizu.Middleware {
	return New(func(c *mizu.Ctx, _ error) error {
		return c.Text(http.StatusInternalServerError, message)
	})
}

// JSON creates middleware that returns JSON error responses.
func JSON() mizu.Middleware {
	return New(func(c *mizu.Ctx, err error) error {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	})
}

// Redirect creates middleware that redirects on errors.
func Redirect(url string, code int) mizu.Middleware {
	return New(func(c *mizu.Ctx, _ error) error {
		return c.Redirect(code, url)
	})
}

// Chain creates a fallback chain, trying each handler in order.
func Chain(handlers ...func(c *mizu.Ctx, err error) (handled bool, newErr error)) mizu.Middleware {
	return New(func(c *mizu.Ctx, err error) error {
		for _, h := range handlers {
			handled, newErr := h(c, err)
			if handled {
				return newErr
			}
		}
		// No handler handled the error
		return c.Text(http.StatusInternalServerError, "An error occurred")
	})
}
