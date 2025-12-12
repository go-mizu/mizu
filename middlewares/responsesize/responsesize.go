// Package responsesize provides response size tracking middleware for Mizu.
package responsesize

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Info contains response size information.
type Info struct {
	bytesWritten int64
}

// BytesWritten returns the number of bytes written.
func (i *Info) BytesWritten() int64 {
	return atomic.LoadInt64(&i.bytesWritten)
}

// Options configures the responsesize middleware.
type Options struct {
	// OnSize is called after response is written.
	OnSize func(c *mizu.Ctx, size int64)
}

// trackingWriter tracks bytes written to response.
type trackingWriter struct {
	http.ResponseWriter
	info *Info
}

func (t *trackingWriter) Write(b []byte) (int, error) {
	n, err := t.ResponseWriter.Write(b)
	atomic.AddInt64(&t.info.bytesWritten, int64(n))
	return n, err
}

// New creates responsesize middleware.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates responsesize middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			info := &Info{}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Wrap response writer
			tw := &trackingWriter{
				ResponseWriter: c.Writer(),
				info:           info,
			}
			c.SetWriter(tw)

			err := next(c)

			// Call callback
			if opts.OnSize != nil {
				opts.OnSize(c, info.BytesWritten())
			}

			// Restore original writer
			c.SetWriter(tw.ResponseWriter)

			return err
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

// BytesWritten returns the bytes written from context.
func BytesWritten(c *mizu.Ctx) int64 {
	return Get(c).BytesWritten()
}

// WithCallback creates middleware with a callback.
func WithCallback(fn func(c *mizu.Ctx, size int64)) mizu.Middleware {
	return WithOptions(Options{OnSize: fn})
}
