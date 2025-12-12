// Package idempotency provides idempotency key middleware for Mizu.
package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Response represents a cached response.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ExpiresAt  time.Time
}

// Store is the idempotency store interface.
type Store interface {
	Get(key string) (*Response, error)
	Set(key string, resp *Response) error
	Delete(key string) error
}

// Options configures the idempotency middleware.
type Options struct {
	// Store is the backing store.
	// Default: in-memory store.
	Store Store

	// KeyHeader is the header containing the idempotency key.
	// Default: "Idempotency-Key".
	KeyHeader string

	// KeyLookup specifies additional places to look for the key.
	// Supported: "header:name", "query:name".
	// Default: "header:Idempotency-Key".
	KeyLookup string

	// Lifetime is how long to cache responses.
	// Default: 24h.
	Lifetime time.Duration

	// Methods are HTTP methods to apply idempotency to.
	// Default: POST, PUT, PATCH.
	Methods []string

	// KeyGenerator generates cache key from idempotency key.
	// Default: uses raw key.
	KeyGenerator func(key string, c *mizu.Ctx) string
}

// New creates idempotency middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithStore creates idempotency middleware with custom store.
func WithStore(store Store, opts Options) mizu.Middleware {
	opts.Store = store
	return WithOptions(opts)
}

// WithOptions creates idempotency middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Store == nil {
		opts.Store = NewMemoryStore()
	}
	if opts.KeyHeader == "" {
		opts.KeyHeader = "Idempotency-Key"
	}
	if opts.Lifetime == 0 {
		opts.Lifetime = 24 * time.Hour
	}
	if len(opts.Methods) == 0 {
		opts.Methods = []string{http.MethodPost, http.MethodPut, http.MethodPatch}
	}
	if opts.KeyGenerator == nil {
		opts.KeyGenerator = func(key string, c *mizu.Ctx) string {
			// Hash with method and path for safety
			h := sha256.New()
			h.Write([]byte(key))
			h.Write([]byte(c.Request().Method))
			h.Write([]byte(c.Request().URL.Path))
			return hex.EncodeToString(h.Sum(nil))
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check if method should be handled
			method := c.Request().Method
			var shouldHandle bool
			for _, m := range opts.Methods {
				if m == method {
					shouldHandle = true
					break
				}
			}
			if !shouldHandle {
				return next(c)
			}

			// Get idempotency key
			idempotencyKey := c.Request().Header.Get(opts.KeyHeader)
			if idempotencyKey == "" {
				return next(c)
			}

			// Generate cache key
			cacheKey := opts.KeyGenerator(idempotencyKey, c)

			// Check cache
			if cached, err := opts.Store.Get(cacheKey); err == nil && cached != nil {
				// Return cached response
				for k, v := range cached.Headers {
					for _, val := range v {
						c.Writer().Header().Add(k, val)
					}
				}
				c.Writer().Header().Set("Idempotent-Replayed", "true")
				c.Writer().WriteHeader(cached.StatusCode)
				_, _ = c.Writer().Write(cached.Body)
				return nil
			}

			// Capture response
			rw := &responseCapture{
				ResponseWriter: c.Writer(),
				body:           &bytes.Buffer{},
				status:         http.StatusOK,
			}
			c.SetWriter(rw)

			err := next(c)

			// Cache response
			resp := &Response{
				StatusCode: rw.status,
				Headers:    c.Writer().Header().Clone(),
				Body:       rw.body.Bytes(),
				ExpiresAt:  time.Now().Add(opts.Lifetime),
			}
			_ = opts.Store.Set(cacheKey, resp)

			return err
		}
	}
}

type responseCapture struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseCapture) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// MemoryStore is an in-memory idempotency store.
type MemoryStore struct {
	mu       sync.RWMutex
	data     map[string]*Response
	stopChan chan struct{}
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		data:     make(map[string]*Response),
		stopChan: make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *MemoryStore) Get(key string) (*Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if resp, ok := s.data[key]; ok {
		if time.Now().Before(resp.ExpiresAt) {
			return resp, nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) Set(key string, resp *Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = resp
	return nil
}

func (s *MemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, resp := range s.data {
				if now.After(resp.ExpiresAt) {
					delete(s.data, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopChan:
			return
		}
	}
}

// Close stops the cleanup goroutine.
func (s *MemoryStore) Close() {
	close(s.stopChan)
}
