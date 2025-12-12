// Package audit provides request/response audit logging middleware for Mizu.
package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Entry represents an audit log entry.
type Entry struct {
	Timestamp   time.Time         `json:"timestamp"`
	RequestID   string            `json:"request_id,omitempty"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Query       string            `json:"query,omitempty"`
	RemoteAddr  string            `json:"remote_addr"`
	UserAgent   string            `json:"user_agent,omitempty"`
	RequestBody string            `json:"request_body,omitempty"`
	Status      int               `json:"status"`
	Latency     time.Duration     `json:"latency"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Handler processes audit entries.
type Handler func(entry *Entry)

// Options configures the audit middleware.
type Options struct {
	// Handler processes each audit entry.
	// Default: writes JSON to stdout.
	Handler Handler

	// RequestIDHeader is the header containing request ID.
	// Default: "X-Request-ID".
	RequestIDHeader string

	// IncludeRequestBody includes request body in audit.
	// Default: false.
	IncludeRequestBody bool

	// MaxBodySize is the max request body size to capture.
	// Default: 1024.
	MaxBodySize int

	// Skip determines which requests to skip.
	Skip func(c *mizu.Ctx) bool

	// Metadata adds custom metadata to entries.
	Metadata func(c *mizu.Ctx) map[string]string
}

// New creates audit middleware with handler.
func New(handler Handler) mizu.Middleware {
	return WithOptions(Options{Handler: handler})
}

// WithOptions creates audit middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Handler == nil {
		opts.Handler = defaultHandler
	}
	if opts.RequestIDHeader == "" {
		opts.RequestIDHeader = "X-Request-ID"
	}
	if opts.MaxBodySize == 0 {
		opts.MaxBodySize = 1024
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if opts.Skip != nil && opts.Skip(c) {
				return next(c)
			}

			r := c.Request()
			start := time.Now()

			entry := &Entry{
				Timestamp:  start,
				RequestID:  r.Header.Get(opts.RequestIDHeader),
				Method:     r.Method,
				Path:       r.URL.Path,
				Query:      r.URL.RawQuery,
				RemoteAddr: r.RemoteAddr,
				UserAgent:  r.UserAgent(),
			}

			// Capture request body
			if opts.IncludeRequestBody && r.Body != nil && r.ContentLength > 0 {
				body, err := io.ReadAll(io.LimitReader(r.Body, int64(opts.MaxBodySize)))
				if err == nil {
					entry.RequestBody = string(body)
					r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), r.Body))
				}
			}

			// Capture response status
			rw := &auditResponseWriter{ResponseWriter: c.Writer()}
			c.SetWriter(rw)

			// Execute handler
			err := next(c)

			entry.Status = rw.status
			if entry.Status == 0 {
				entry.Status = http.StatusOK
			}
			entry.Latency = time.Since(start)

			if err != nil {
				entry.Error = err.Error()
			}

			// Add custom metadata
			if opts.Metadata != nil {
				entry.Metadata = opts.Metadata(c)
			}

			// Call handler
			opts.Handler(entry)

			return err
		}
	}
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *auditResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func defaultHandler(entry *Entry) {
	json.NewEncoder(io.Discard).Encode(entry)
}

// ChannelHandler creates a handler that sends entries to a channel.
func ChannelHandler(ch chan<- *Entry) Handler {
	return func(entry *Entry) {
		select {
		case ch <- entry:
		default:
			// Drop if channel is full
		}
	}
}

// BufferedHandler collects entries and flushes periodically.
type BufferedHandler struct {
	entries  []*Entry
	mu       sync.Mutex
	maxSize  int
	flush    func([]*Entry)
	stopChan chan struct{}
}

// NewBufferedHandler creates a buffered handler.
func NewBufferedHandler(maxSize int, flushInterval time.Duration, flush func([]*Entry)) *BufferedHandler {
	h := &BufferedHandler{
		entries:  make([]*Entry, 0, maxSize),
		maxSize:  maxSize,
		flush:    flush,
		stopChan: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.Flush()
			case <-h.stopChan:
				return
			}
		}
	}()

	return h
}

// Handle processes an audit entry.
func (h *BufferedHandler) Handle(entry *Entry) {
	h.mu.Lock()
	h.entries = append(h.entries, entry)
	shouldFlush := len(h.entries) >= h.maxSize
	h.mu.Unlock()

	if shouldFlush {
		h.Flush()
	}
}

// Flush sends buffered entries.
func (h *BufferedHandler) Flush() {
	h.mu.Lock()
	if len(h.entries) == 0 {
		h.mu.Unlock()
		return
	}
	entries := h.entries
	h.entries = make([]*Entry, 0, h.maxSize)
	h.mu.Unlock()

	h.flush(entries)
}

// Close stops the buffered handler.
func (h *BufferedHandler) Close() {
	close(h.stopChan)
	h.Flush()
}

// Handler returns the handler function.
func (h *BufferedHandler) Handler() Handler {
	return h.Handle
}
