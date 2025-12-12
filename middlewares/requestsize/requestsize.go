// Package requestsize provides request size tracking middleware for Mizu.
package requestsize

import (
	"context"
	"io"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Info contains request size information.
type Info struct {
	ContentLength int64
	BytesRead     int64
}

// Options configures the requestsize middleware.
type Options struct {
	// OnSize is called after request processing.
	OnSize func(c *mizu.Ctx, info *Info)
}

// trackingBody tracks bytes read from request body.
type trackingBody struct {
	io.ReadCloser
	bytesRead int64
}

func (t *trackingBody) Read(p []byte) (int, error) {
	n, err := t.ReadCloser.Read(p)
	t.bytesRead += int64(n)
	return n, err
}

// New creates requestsize middleware.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates requestsize middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			info := &Info{
				ContentLength: c.Request().ContentLength,
			}

			// Wrap body to track reads
			if c.Request().Body != nil {
				tb := &trackingBody{ReadCloser: c.Request().Body}
				c.Request().Body = tb

				defer func() {
					info.BytesRead = tb.bytesRead
					if opts.OnSize != nil {
						opts.OnSize(c, info)
					}
				}()
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// Get retrieves size info from context.
func Get(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(contextKey{}).(*Info); ok {
		return info
	}
	return &Info{}
}

// ContentLength returns the Content-Length from context.
func ContentLength(c *mizu.Ctx) int64 {
	return Get(c).ContentLength
}

// BytesRead returns the actual bytes read.
func BytesRead(c *mizu.Ctx) int64 {
	return Get(c).BytesRead
}

// WithCallback creates middleware with a callback.
func WithCallback(fn func(c *mizu.Ctx, info *Info)) mizu.Middleware {
	return WithOptions(Options{OnSize: fn})
}
