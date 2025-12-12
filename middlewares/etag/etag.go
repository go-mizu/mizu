// Package etag provides ETag generation middleware for Mizu.
package etag

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"
)

// Options configures the ETag middleware.
type Options struct {
	// Weak generates weak ETags (W/"...").
	Weak bool

	// HashFunc is a custom hash function.
	// Default: CRC32.
	HashFunc func([]byte) string
}

// New creates ETag middleware with default settings.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates ETag middleware with options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.HashFunc == nil {
		opts.HashFunc = func(b []byte) string {
			return strconv.FormatUint(uint64(crc32.ChecksumIEEE(b)), 16)
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip if not GET/HEAD
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			// Buffer the response
			buf := &bytes.Buffer{}
			bw := &bufferedWriter{
				ResponseWriter: c.Writer(),
				buf:            buf,
			}
			c.SetWriter(bw)

			err := next(c)
			if err != nil {
				return err
			}

			// Determine status (default to 200 if not set)
			status := bw.status
			if status == 0 {
				status = http.StatusOK
			}

			// Only generate ETag for successful responses
			if status >= 200 && status < 300 && buf.Len() > 0 {
				// Generate ETag
				hash := opts.HashFunc(buf.Bytes())
				var etag string
				if opts.Weak {
					etag = fmt.Sprintf(`W/"%s"`, hash)
				} else {
					etag = fmt.Sprintf(`"%s"`, hash)
				}

				// Check If-None-Match
				ifNoneMatch := c.Request().Header.Get("If-None-Match")
				if ifNoneMatch == etag || ifNoneMatch == "*" {
					bw.ResponseWriter.Header().Set("ETag", etag)
					bw.ResponseWriter.WriteHeader(http.StatusNotModified)
					return nil
				}

				// Set ETag header and write response
				bw.ResponseWriter.Header().Set("ETag", etag)
			}

			// Write buffered content to original writer
			if !bw.headerWritten {
				bw.ResponseWriter.WriteHeader(status)
			}
			_, writeErr := bw.ResponseWriter.Write(buf.Bytes())
			return writeErr
		}
	}
}

// Weak creates middleware generating weak ETags.
func Weak() mizu.Middleware {
	return WithOptions(Options{Weak: true})
}

type bufferedWriter struct {
	http.ResponseWriter
	buf           *bytes.Buffer
	status        int
	headerWritten bool
}

func (w *bufferedWriter) WriteHeader(code int) {
	w.status = code
}

func (w *bufferedWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *bufferedWriter) Flush() {
	// Buffering, don't flush yet
}
