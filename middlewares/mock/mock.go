// Package mock provides request mocking middleware for Mizu.
package mock

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"
)

// Response represents a mock response.
type Response struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

// Options configures the mock middleware.
type Options struct {
	// Mocks maps paths to responses.
	Mocks map[string]*Response

	// DefaultResponse is returned for unmatched paths.
	DefaultResponse *Response

	// Passthrough passes unmatched requests to next handler.
	// Default: true.
	Passthrough bool
}

// Mock holds registered mock responses.
type Mock struct {
	mu      sync.RWMutex
	mocks   map[string]*Response
	methods map[string]map[string]*Response // method -> path -> response
	opts    Options
}

// NewMock creates a new mock instance.
func NewMock() *Mock {
	return &Mock{
		mocks:   make(map[string]*Response),
		methods: make(map[string]map[string]*Response),
		opts:    Options{Passthrough: true},
	}
}

// Register registers a mock response for a path.
func (m *Mock) Register(path string, resp *Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mocks[path] = resp
}

// RegisterMethod registers a mock for a specific method and path.
func (m *Mock) RegisterMethod(method, path string, resp *Response) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.methods[method] == nil {
		m.methods[method] = make(map[string]*Response)
	}
	m.methods[method][path] = resp
}

// Clear removes all mocks.
func (m *Mock) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mocks = make(map[string]*Response)
	m.methods = make(map[string]map[string]*Response)
}

// Middleware returns the mock middleware.
func (m *Mock) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			method := c.Request().Method

			m.mu.RLock()
			// Check method-specific mock first
			if methodMocks, ok := m.methods[method]; ok {
				if resp, ok := methodMocks[path]; ok {
					m.mu.RUnlock()
					return sendResponse(c, resp)
				}
			}
			// Check path-only mock
			if resp, ok := m.mocks[path]; ok {
				m.mu.RUnlock()
				return sendResponse(c, resp)
			}
			m.mu.RUnlock()

			// Check default response
			if m.opts.DefaultResponse != nil {
				return sendResponse(c, m.opts.DefaultResponse)
			}

			// Passthrough or 404
			if m.opts.Passthrough {
				return next(c)
			}

			return c.Text(http.StatusNotFound, "Mock not found")
		}
	}
}

func sendResponse(c *mizu.Ctx, resp *Response) error {
	for k, v := range resp.Headers {
		c.Header().Set(k, v)
	}
	c.Writer().WriteHeader(resp.Status)
	if len(resp.Body) > 0 {
		_, err := c.Writer().Write(resp.Body)
		return err
	}
	return nil
}

// New creates mock middleware with predefined responses.
func New(mocks map[string]*Response) mizu.Middleware {
	m := NewMock()
	for path, resp := range mocks {
		m.Register(path, resp)
	}
	return m.Middleware()
}

// WithOptions creates mock middleware with options.
func WithOptions(opts Options) mizu.Middleware {
	m := NewMock()
	m.opts = opts
	for path, resp := range opts.Mocks {
		m.Register(path, resp)
	}
	return m.Middleware()
}

// JSON creates a JSON mock response.
func JSON(status int, data any) *Response {
	body, _ := json.Marshal(data)
	return &Response{
		Status: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
}

// Text creates a text mock response.
func Text(status int, text string) *Response {
	return &Response{
		Status: status,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: []byte(text),
	}
}

// HTML creates an HTML mock response.
func HTML(status int, html string) *Response {
	return &Response{
		Status: status,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body: []byte(html),
	}
}

// File creates a file-like mock response.
func File(contentType string, data []byte) *Response {
	return &Response{
		Status: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		Body: data,
	}
}

// Redirect creates a redirect mock response.
func Redirect(url string, code int) *Response {
	return &Response{
		Status: code,
		Headers: map[string]string{
			"Location": url,
		},
	}
}

// Error creates an error mock response.
func Error(status int, message string) *Response {
	return JSON(status, map[string]string{"error": message})
}

// Prefix creates middleware that mocks all paths with a prefix.
func Prefix(prefix string, resp *Response) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if strings.HasPrefix(c.Request().URL.Path, prefix) {
				return sendResponse(c, resp)
			}
			return next(c)
		}
	}
}
