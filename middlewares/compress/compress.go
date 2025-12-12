// Package compress provides response compression middleware for Mizu.
package compress

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"
)

// Options configures the compression middleware.
type Options struct {
	// Level is the compression level (1-9).
	// Default: 6 (gzip.DefaultCompression).
	Level int

	// MinSize is the minimum response size to compress.
	// Default: 1024 bytes.
	MinSize int

	// ContentTypes are the MIME types to compress.
	// Default: text/*, application/json, application/javascript, application/xml.
	ContentTypes []string
}

var defaultContentTypes = []string{
	"text/html",
	"text/css",
	"text/plain",
	"text/javascript",
	"text/xml",
	"application/json",
	"application/javascript",
	"application/xml",
	"application/xhtml+xml",
	"application/rss+xml",
	"application/atom+xml",
	"image/svg+xml",
}

// Gzip creates gzip compression middleware with default compression level.
func Gzip() mizu.Middleware {
	return GzipLevel(gzip.DefaultCompression)
}

// GzipLevel creates gzip compression middleware with the specified level.
func GzipLevel(level int) mizu.Middleware {
	return newCompressor("gzip", level, Options{})
}

// Deflate creates deflate compression middleware with default compression level.
func Deflate() mizu.Middleware {
	return DeflateLevel(flate.DefaultCompression)
}

// DeflateLevel creates deflate compression middleware with the specified level.
func DeflateLevel(level int) mizu.Middleware {
	return newCompressor("deflate", level, Options{})
}

// New creates compression middleware supporting multiple algorithms.
func New(opts Options) mizu.Middleware {
	return newCompressor("", opts.Level, opts)
}

func newCompressor(encoding string, level int, opts Options) mizu.Middleware {
	if level == 0 {
		level = gzip.DefaultCompression
	}
	if opts.MinSize == 0 {
		opts.MinSize = 1024
	}
	if len(opts.ContentTypes) == 0 {
		opts.ContentTypes = defaultContentTypes
	}

	gzipPool := sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, level)
			return w
		},
	}

	flatePool := sync.Pool{
		New: func() any {
			w, _ := flate.NewWriter(io.Discard, level)
			return w
		},
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			acceptEncoding := c.Request().Header.Get("Accept-Encoding")

			// Determine which encoding to use
			var selectedEncoding string
			if encoding != "" {
				if strings.Contains(acceptEncoding, encoding) {
					selectedEncoding = encoding
				}
			} else {
				// Auto-select based on Accept-Encoding
				if strings.Contains(acceptEncoding, "gzip") {
					selectedEncoding = "gzip"
				} else if strings.Contains(acceptEncoding, "deflate") {
					selectedEncoding = "deflate"
				}
			}

			if selectedEncoding == "" {
				return next(c)
			}

			// Create compressed response writer
			cw := &compressWriter{
				ResponseWriter: c.Writer(),
				encoding:       selectedEncoding,
				minSize:        opts.MinSize,
				contentTypes:   opts.ContentTypes,
				gzipPool:       &gzipPool,
				flatePool:      &flatePool,
			}
			defer func() { _ = cw.Close() }()

			c.SetWriter(cw)
			c.Header().Set("Vary", "Accept-Encoding")

			return next(c)
		}
	}
}

type compressWriter struct {
	http.ResponseWriter
	encoding     string
	minSize      int
	contentTypes []string
	writer       io.WriteCloser
	gzipPool     *sync.Pool
	flatePool    *sync.Pool
	buf          []byte
	wroteHeader  bool
	compress     bool
	statusCode   int
}

func (w *compressWriter) shouldCompress() bool {
	ct := w.Header().Get("Content-Type")
	if ct == "" {
		return false
	}

	// Check if already encoded
	if w.Header().Get("Content-Encoding") != "" {
		return false
	}

	// Check content type
	for _, allowed := range w.contentTypes {
		if strings.HasPrefix(ct, allowed) || strings.Contains(ct, allowed) {
			return true
		}
	}
	return false
}

func (w *compressWriter) WriteHeader(code int) {
	if w.statusCode == 0 {
		w.statusCode = code
	}
}

func (w *compressWriter) flushHeader() {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	if w.shouldCompress() && len(w.buf) >= w.minSize {
		w.compress = true
		w.Header().Set("Content-Encoding", w.encoding)
		w.Header().Del("Content-Length")

		switch w.encoding {
		case "gzip":
			gw := w.gzipPool.Get().(*gzip.Writer)
			gw.Reset(w.ResponseWriter)
			w.writer = gw
		case "deflate":
			fw := w.flatePool.Get().(*flate.Writer)
			fw.Reset(w.ResponseWriter)
			w.writer = fw
		}
	}

	code := w.statusCode
	if code == 0 {
		code = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(code)

	// Write buffered content
	if len(w.buf) > 0 {
		if w.writer != nil {
			_, _ = w.writer.Write(w.buf)
		} else {
			_, _ = w.ResponseWriter.Write(w.buf)
		}
		w.buf = nil
	}
}

func (w *compressWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		// Buffer until we know if we should compress
		w.buf = append(w.buf, b...)
		if len(w.buf) >= w.minSize {
			w.flushHeader()
		}
		return len(b), nil
	}

	if w.writer != nil {
		return w.writer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *compressWriter) Close() error {
	// Flush any remaining buffered data
	if !w.wroteHeader && len(w.buf) > 0 {
		w.flushHeader()
	}

	if w.writer != nil {
		err := w.writer.Close()

		// Return writer to pool
		switch w.encoding {
		case "gzip":
			w.gzipPool.Put(w.writer)
		case "deflate":
			w.flatePool.Put(w.writer)
		}

		return err
	}
	return nil
}

func (w *compressWriter) Flush() {
	if w.writer != nil {
		if f, ok := w.writer.(interface{ Flush() error }); ok {
			_ = f.Flush()
		}
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
