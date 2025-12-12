package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(3, time.Minute))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// Different IP should not be rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP: expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Headers(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Rate:     10,
		Interval: time.Minute,
		Headers:  true,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-RateLimit-Limit") != "10" {
		t.Error("expected X-RateLimit-Limit: 10")
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Rate:     1,
		Interval: time.Hour,
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded",
			})
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// First request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Second request - rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
}

func TestWithOptions_KeyFunc(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Rate:     2,
		Interval: time.Minute,
		KeyFunc: func(c *mizu.Ctx) string {
			return c.Query("api_key")
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// API key 1 - 2 requests allowed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test?api_key=key1", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("key1 request %d: expected %d, got %d", i+1, http.StatusOK, rec.Code)
		}
	}

	// API key 1 - 3rd request rate limited
	req := httptest.NewRequest(http.MethodGet, "/test?api_key=key1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("key1 request 3: expected %d, got %d", http.StatusTooManyRequests, rec.Code)
	}

	// API key 2 - should work
	req = httptest.NewRequest(http.MethodGet, "/test?api_key=key2", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("key2: expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Skip(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Rate:     1,
		Interval: time.Hour,
		Skip: func(c *mizu.Ctx) bool {
			return c.Query("skip") == "true"
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Normal request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Error("expected rate limiting")
	}

	// Skip rate limiting
	req = httptest.NewRequest(http.MethodGet, "/test?skip=true", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Error("expected skip to work")
	}
}

func TestWithOptions_RetryAfter(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(1, time.Minute))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// First request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Second request - rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestPerSecond(t *testing.T) {
	mw := PerSecond(100)
	if mw == nil {
		t.Error("expected middleware")
	}
}

func TestPerMinute(t *testing.T) {
	mw := PerMinute(100)
	if mw == nil {
		t.Error("expected middleware")
	}
}

func TestPerHour(t *testing.T) {
	mw := PerHour(1000)
	if mw == nil {
		t.Error("expected middleware")
	}
}

func TestMemoryStore_Allow(t *testing.T) {
	store := NewMemoryStore()

	// Allow up to 3 requests
	for i := 0; i < 3; i++ {
		allowed, info := store.Allow("test-key", 3, time.Minute, 3)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		if info.Limit != 3 {
			t.Errorf("expected limit 3, got %d", info.Limit)
		}
	}

	// 4th should be denied
	allowed, _ := store.Allow("test-key", 3, time.Minute, 3)
	if allowed {
		t.Error("4th request should be denied")
	}
}

func TestMemoryStore_TokenRefill(t *testing.T) {
	store := NewMemoryStore()

	// Use all tokens
	for i := 0; i < 10; i++ {
		store.Allow("refill-test", 10, 100*time.Millisecond, 10)
	}

	// Should be rate limited
	allowed, _ := store.Allow("refill-test", 10, 100*time.Millisecond, 10)
	if allowed {
		t.Error("should be rate limited")
	}

	// Wait for tokens to refill
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, _ = store.Allow("refill-test", 10, 100*time.Millisecond, 10)
	if !allowed {
		t.Error("should be allowed after refill")
	}
}

func TestMemoryStore_DifferentKeys(t *testing.T) {
	store := NewMemoryStore()

	// Exhaust key1
	for i := 0; i < 5; i++ {
		store.Allow("key1", 5, time.Hour, 5)
	}

	allowed, _ := store.Allow("key1", 5, time.Hour, 5)
	if allowed {
		t.Error("key1 should be exhausted")
	}

	// key2 should still work
	allowed, _ = store.Allow("key2", 5, time.Hour, 5)
	if !allowed {
		t.Error("key2 should be allowed")
	}
}
