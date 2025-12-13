// Package session provides cookie-based session management middleware for Mizu applications.
//
// # Overview
//
// The session middleware enables server-side session management with secure cookie-based tracking.
// Sessions persist user data across multiple HTTP requests using configurable storage backends.
//
// # Features
//
//   - Secure session ID generation using crypto/rand
//   - Thread-safe session operations with sync.RWMutex
//   - Flexible storage backend via Store interface
//   - Built-in memory store with automatic cleanup
//   - Configurable cookie settings (secure, HTTPOnly, SameSite)
//   - Session lifecycle management (creation, persistence, cleanup)
//   - Deep copying of session data to prevent mutations
//
// # Basic Usage
//
//	app := mizu.New()
//
//	// Enable sessions with default options
//	app.Use(session.New(session.Options{}))
//
//	app.Get("/login", func(c *mizu.Ctx) error {
//	    sess := session.Get(c)
//	    sess.Set("user_id", "123")
//	    return c.Text(200, "Logged in")
//	})
//
//	app.Get("/profile", func(c *mizu.Ctx) error {
//	    sess := session.Get(c)
//	    userID := sess.Get("user_id").(string)
//	    return c.Text(200, "User: "+userID)
//	})
//
// # Configuration
//
// The Options struct allows customization of session behavior:
//
//	app.Use(session.New(session.Options{
//	    CookieName:     "my_session",      // Custom cookie name
//	    CookieSecure:   true,              // HTTPS only
//	    CookieHTTPOnly: true,              // Prevent JavaScript access
//	    SameSite:       http.SameSiteStrictMode,
//	    Lifetime:       7 * 24 * time.Hour, // 1 week
//	}))
//
// # Custom Storage
//
// Implement the Store interface for custom backends (Redis, database, etc.):
//
//	type Store interface {
//	    Get(id string) (*SessionData, error)
//	    Save(id string, data *SessionData, lifetime time.Duration) error
//	    Delete(id string) error
//	}
//
//	customStore := NewRedisStore()
//	app.Use(session.WithStore(customStore, session.Options{}))
//
// # Session Operations
//
// The Session type provides methods for managing session data:
//
//	sess := session.Get(c)
//
//	// Store values
//	sess.Set("key", "value")
//	sess.Set("count", 42)
//
//	// Retrieve values
//	value := sess.Get("key").(string)
//
//	// Delete specific value
//	sess.Delete("key")
//
//	// Clear all values
//	sess.Clear()
//
//	// Destroy session
//	sess.Destroy()
//
// # Thread Safety
//
// All session operations are thread-safe using sync.RWMutex:
//   - Read operations (Get) use read locks for concurrent access
//   - Write operations (Set, Delete, Clear) use write locks for exclusive access
//   - The internal changed flag optimizes store operations
//
// # Session Lifecycle
//
// 1. Request arrives without session cookie → New session created
// 2. Session ID generated using crypto/rand (32 bytes → 64 hex chars)
// 3. Session stored in request context
// 4. Cookie set before handler executes
// 5. Handler modifies session data
// 6. Session saved to store if changed
// 7. Response sent with session cookie
//
// # Memory Store
//
// The built-in MemoryStore provides:
//   - In-memory session storage (not suitable for distributed systems)
//   - Automatic cleanup every 10 minutes
//   - Deep copying to prevent external mutations
//   - Thread-safe operations
//
// Note: For production distributed systems, implement a custom Store backed by Redis or a database.
//
// # Security Considerations
//
//   - Always use CookieSecure: true in production (HTTPS only)
//   - Keep CookieHTTPOnly: true to prevent XSS attacks
//   - Use SameSiteStrictMode for sensitive applications
//   - Set appropriate session lifetimes (avoid infinite sessions)
//   - Regenerate session IDs after authentication state changes
//   - Store minimal sensitive data in sessions
//
// # Best Practices
//
//   - Use secure cookie settings in production environments
//   - Store only necessary data in sessions
//   - Implement cleanup mechanisms for expired sessions
//   - Consider Redis or database stores for distributed systems
//   - Monitor session store memory usage
//   - Set appropriate cookie domains for multi-subdomain apps
//
// # Context Integration
//
// Sessions are stored in the request context using a private contextKey{} struct,
// preventing collisions with other middleware or application code.
//
// Both Get() and FromContext() retrieve the same session instance from context.
package session
