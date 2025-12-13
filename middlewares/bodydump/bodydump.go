// Package bodydump provides middleware for dumping request/response bodies.
package bodydump

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Options configures the bodydump middleware.
type Options struct {
	// Request dumps request body.
	// Default: true.
	Request bool

	// Response dumps response body.
	// Default: true.
	Response bool

	// MaxSize is the maximum bytes to dump.
	// Default: 64KB.
	MaxSize int64

	// Handler receives the dumped bodies.
	Handler func(c *mizu.Ctx, reqBody, respBody []byte)

	// SkipPaths are paths to skip.
	SkipPaths []string

	// SkipContentTypes are content types to skip.
	SkipContentTypes []string
}

// responseCapture captures the response body.
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

// New creates bodydump middleware with a handler.
func New(handler func(c *mizu.Ctx, reqBody, respBody []byte)) mizu.Middleware {
	return WithOptions(Options{
		Request:  true,
		Response: true,
		Handler:  handler,
	})
}

// WithOptions creates bodydump middleware with custom options.
//
//nolint:cyclop // Body dump requires multiple option and content type checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.MaxSize == 0 {
		opts.MaxSize = 64 * 1024 // 64KB
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	skipTypes := make(map[string]bool)
	for _, t := range opts.SkipContentTypes {
		skipTypes[t] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			var reqBody []byte
			var respBody []byte

			// Dump request body
			if opts.Request && c.Request().Body != nil {
				// Check content type
				ct := c.Request().Header.Get("Content-Type")
				if !skipTypes[ct] {
					body, err := io.ReadAll(io.LimitReader(c.Request().Body, opts.MaxSize))
					if err == nil {
						reqBody = body
						// Restore body for handler
						c.Request().Body = io.NopCloser(bytes.NewReader(body))
					}
				}
			}

			// Capture response body
			var capture *responseCapture
			if opts.Response {
				capture = &responseCapture{
					ResponseWriter: c.Writer(),
					body:           &bytes.Buffer{},
					statusCode:     http.StatusOK,
					maxSize:        opts.MaxSize,
				}
				c.SetWriter(capture)
			}

			// Execute handler
			err := next(c)

			// Get response body
			if opts.Response && capture != nil {
				respBody = capture.body.Bytes()
				// Restore original writer
				c.SetWriter(capture.ResponseWriter)
			}

			// Call dump handler
			if opts.Handler != nil {
				opts.Handler(c, reqBody, respBody)
			}

			return err
		}
	}
}

// RequestOnly dumps only request bodies.
func RequestOnly(handler func(c *mizu.Ctx, body []byte)) mizu.Middleware {
	return WithOptions(Options{
		Request:  true,
		Response: false,
		Handler: func(c *mizu.Ctx, reqBody, _ []byte) {
			handler(c, reqBody)
		},
	})
}

// ResponseOnly dumps only response bodies.
func ResponseOnly(handler func(c *mizu.Ctx, body []byte)) mizu.Middleware {
	return WithOptions(Options{
		Request:  false,
		Response: true,
		Handler: func(c *mizu.Ctx, _, respBody []byte) {
			handler(c, respBody)
		},
	})
}
