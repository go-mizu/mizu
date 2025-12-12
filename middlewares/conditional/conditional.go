// Package conditional provides combined conditional request handling middleware for Mizu.
package conditional

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the conditional middleware.
type Options struct {
	// ETag enables ETag generation.
	// Default: true.
	ETag bool

	// WeakETag uses weak ETag.
	// Default: false.
	WeakETag bool

	// LastModified enables Last-Modified handling.
	// Default: true.
	LastModified bool

	// ModTimeFunc returns last modified time.
	ModTimeFunc func(c *mizu.Ctx) time.Time

	// ETagFunc returns custom ETag.
	ETagFunc func(body []byte) string
}

// responseCapture captures response for ETag generation.
type responseCapture struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
	// Don't write header yet
}

func (r *responseCapture) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// New creates conditional middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates conditional middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	// Default ETag to true if not explicitly set
	if !opts.ETag && !opts.LastModified && opts.ETagFunc == nil && opts.ModTimeFunc == nil {
		opts.ETag = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Only handle GET and HEAD
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			// Capture response
			capture := &responseCapture{
				ResponseWriter: c.Writer(),
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			originalWriter := c.Writer()
			c.SetWriter(capture)

			// Execute handler
			if err := next(c); err != nil {
				c.SetWriter(originalWriter)
				return err
			}

			// Restore writer
			c.SetWriter(originalWriter)

			// Only process successful responses
			if capture.statusCode < 200 || capture.statusCode >= 300 {
				c.Writer().WriteHeader(capture.statusCode)
				_, err := c.Writer().Write(capture.body.Bytes())
				return err
			}

			body := capture.body.Bytes()

			// Generate ETag
			var etag string
			if opts.ETag {
				if opts.ETagFunc != nil {
					etag = opts.ETagFunc(body)
				} else {
					hash := md5.Sum(body)
					etag = hex.EncodeToString(hash[:])
				}
				if opts.WeakETag {
					etag = "W/" + `"` + etag + `"`
				} else {
					etag = `"` + etag + `"`
				}
				c.Header().Set("ETag", etag)
			}

			// Set Last-Modified
			var lastMod time.Time
			if opts.LastModified && opts.ModTimeFunc != nil {
				lastMod = opts.ModTimeFunc(c)
				if !lastMod.IsZero() {
					c.Header().Set("Last-Modified", lastMod.UTC().Format(http.TimeFormat))
				}
			}

			// Check If-None-Match
			if etag != "" {
				ifNoneMatch := c.Request().Header.Get("If-None-Match")
				if ifNoneMatch == etag || ifNoneMatch == "*" {
					c.Writer().WriteHeader(http.StatusNotModified)
					return nil
				}
			}

			// Check If-Modified-Since
			if !lastMod.IsZero() {
				ifModSince := c.Request().Header.Get("If-Modified-Since")
				if ifModSince != "" {
					t, err := http.ParseTime(ifModSince)
					if err == nil && !lastMod.Truncate(time.Second).After(t) {
						c.Writer().WriteHeader(http.StatusNotModified)
						return nil
					}
				}
			}

			// Write actual response
			c.Writer().WriteHeader(capture.statusCode)
			_, err := c.Writer().Write(body)
			return err
		}
	}
}

// ETagOnly creates middleware that only handles ETags.
func ETagOnly() mizu.Middleware {
	return WithOptions(Options{
		ETag:         true,
		LastModified: false,
	})
}

// LastModifiedOnly creates middleware that only handles Last-Modified.
func LastModifiedOnly(modTimeFunc func(c *mizu.Ctx) time.Time) mizu.Middleware {
	return WithOptions(Options{
		ETag:         false,
		LastModified: true,
		ModTimeFunc:  modTimeFunc,
	})
}

// WithModTime creates middleware with a modification time function.
func WithModTime(modTimeFunc func(c *mizu.Ctx) time.Time) mizu.Middleware {
	return WithOptions(Options{
		ETag:         true,
		LastModified: true,
		ModTimeFunc:  modTimeFunc,
	})
}
