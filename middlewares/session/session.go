// Package session provides cookie-based session management middleware for Mizu.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the session middleware.
type Options struct {
	// CookieName is the session cookie name.
	// Default: "session_id".
	CookieName string

	// CookiePath is the cookie path.
	// Default: "/".
	CookiePath string

	// CookieDomain is the cookie domain.
	CookieDomain string

	// CookieMaxAge is the max age in seconds.
	// Default: 86400 (24 hours).
	CookieMaxAge int

	// CookieSecure sets the Secure flag.
	CookieSecure bool

	// CookieHTTPOnly sets the HTTPOnly flag.
	// Default: true.
	CookieHTTPOnly bool

	// SameSite sets the SameSite attribute.
	// Default: Lax.
	SameSite http.SameSite

	// IdleTimeout is the session idle timeout.
	IdleTimeout time.Duration

	// Lifetime is the absolute session lifetime.
	Lifetime time.Duration

	// KeyGenerator generates session IDs.
	KeyGenerator func() string
}

// Store is the session storage interface.
type Store interface {
	Get(id string) (*SessionData, error)
	Save(id string, data *SessionData, lifetime time.Duration) error
	Delete(id string) error
}

// SessionData holds session data.
type SessionData struct {
	Values    map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Session represents an active session.
type Session struct {
	ID      string
	data    *SessionData
	store   Store
	opts    Options
	mu      sync.RWMutex
	changed bool
}

// New creates session middleware with in-memory store.
func New(opts Options) mizu.Middleware {
	return WithStore(NewMemoryStore(), opts)
}

// WithStore creates session middleware with custom store.
func WithStore(store Store, opts Options) mizu.Middleware {
	if opts.CookieName == "" {
		opts.CookieName = "session_id"
	}
	if opts.CookiePath == "" {
		opts.CookiePath = "/"
	}
	if opts.CookieMaxAge == 0 {
		opts.CookieMaxAge = 86400
	}
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	if !opts.CookieHTTPOnly {
		opts.CookieHTTPOnly = true
	}
	if opts.KeyGenerator == nil {
		opts.KeyGenerator = generateSessionID
	}
	if opts.Lifetime == 0 {
		opts.Lifetime = 24 * time.Hour
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var sessionID string
			if cookie, err := c.Cookie(opts.CookieName); err == nil {
				sessionID = cookie.Value
			}

			var sess *Session
			if sessionID != "" {
				if data, err := store.Get(sessionID); err == nil && data != nil {
					sess = &Session{
						ID:    sessionID,
						data:  data,
						store: store,
						opts:  opts,
					}
				}
			}

			isNew := false
			if sess == nil {
				isNew = true
				sessionID = opts.KeyGenerator()
				sess = &Session{
					ID: sessionID,
					data: &SessionData{
						Values:    make(map[string]any),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					store:   store,
					opts:    opts,
					changed: true,
				}
			}

			// Store session in context
			ctx := context.WithValue(c.Context(), contextKey{}, sess)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Set cookie for new sessions BEFORE handler runs (headers must be set before body)
			if isNew {
				c.SetCookie(&http.Cookie{
					Name:     opts.CookieName,
					Value:    sess.ID,
					Path:     opts.CookiePath,
					Domain:   opts.CookieDomain,
					MaxAge:   opts.CookieMaxAge,
					Secure:   opts.CookieSecure,
					HttpOnly: opts.CookieHTTPOnly,
					SameSite: opts.SameSite,
				})
			}

			err := next(c)

			// Save session data if changed
			if sess.changed {
				sess.data.UpdatedAt = time.Now()
				_ = store.Save(sess.ID, sess.data, opts.Lifetime)
			}

			return err
		}
	}
}

// Get retrieves session from context.
func Get(c *mizu.Ctx) *Session {
	if sess, ok := c.Context().Value(contextKey{}).(*Session); ok {
		return sess
	}
	return nil
}

// FromContext is an alias for Get.
func FromContext(c *mizu.Ctx) *Session {
	return Get(c)
}

// Get retrieves a value from the session.
func (s *Session) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Values[key]
}

// Set sets a value in the session.
func (s *Session) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Values[key] = value
	s.changed = true
}

// Delete removes a value from the session.
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Values, key)
	s.changed = true
}

// Clear removes all values from the session.
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Values = make(map[string]any)
	s.changed = true
}

// Destroy deletes the session.
func (s *Session) Destroy() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store.Delete(s.ID)
}

// MemoryStore is an in-memory session store.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		sessions: make(map[string]*SessionData),
	}
	go store.cleanup()
	return store
}

func (s *MemoryStore) Get(id string) (*SessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if data, ok := s.sessions[id]; ok {
		// Deep copy
		copy := &SessionData{
			Values:    make(map[string]any),
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		}
		for k, v := range data.Values {
			copy.Values[k] = v
		}
		return copy, nil
	}
	return nil, nil
}

func (s *MemoryStore) Save(id string, data *SessionData, lifetime time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = data
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *MemoryStore) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, data := range s.sessions {
			if now.Sub(data.UpdatedAt) > 24*time.Hour {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
