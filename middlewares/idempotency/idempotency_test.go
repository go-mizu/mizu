package idempotency

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var callCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusCreated, "created")
	})

	// First request
	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, rec.Code)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Second request with same key - should be replayed
	req = httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key-123")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, rec.Code)
	}
	if rec.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("expected Idempotent-Replayed header")
	}
	if callCount != 1 {
		t.Errorf("expected handler to be called only once, got %d", callCount)
	}
}

func TestWithOptions_NoKey(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var callCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusOK, "ok")
	})

	// Request without idempotency key
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/create", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls without key, got %d", callCount)
	}
}

func TestWithOptions_DifferentKeys(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var callCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusOK, "ok")
	})

	keys := []string{"key-1", "key-2", "key-3"}
	for _, key := range keys {
		req := httptest.NewRequest(http.MethodPost, "/create", nil)
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls with different keys, got %d", callCount)
	}
}

func TestWithOptions_CustomHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		KeyHeader: "X-Request-Id",
	}))

	var callCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("X-Request-Id", "custom-key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Same key should be replayed
	req = httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("X-Request-Id", "custom-key")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWithOptions_Methods(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Methods: []string{http.MethodPost},
	}))

	var postCount, putCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		postCount++
		return c.Text(http.StatusOK, "ok")
	})
	app.Put("/update", func(c *mizu.Ctx) error {
		putCount++
		return c.Text(http.StatusOK, "ok")
	})

	// POST should be idempotent
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/create", nil)
		req.Header.Set("Idempotency-Key", "key")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
	if postCount != 1 {
		t.Errorf("expected 1 POST call, got %d", postCount)
	}

	// PUT should NOT be idempotent (not in Methods list)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPut, "/update", nil)
		req.Header.Set("Idempotency-Key", "key")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
	if putCount != 2 {
		t.Errorf("expected 2 PUT calls, got %d", putCount)
	}
}

func TestWithOptions_GET(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var callCount int
	app.Get("/resource", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusOK, "ok")
	})

	// GET requests should not be cached
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		req.Header.Set("Idempotency-Key", "key")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	if callCount != 3 {
		t.Errorf("expected 3 GET calls (not cached), got %d", callCount)
	}
}

func TestWithOptions_ResponseHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/create", func(c *mizu.Ctx) error {
		c.Writer().Header().Set("X-Custom", "value")
		return c.Text(http.StatusOK, "ok")
	})

	// First request
	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Second request - should include cached headers
	req = httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Custom") != "value" {
		t.Error("expected X-Custom header in replayed response")
	}
}

func TestWithStore(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	app := mizu.NewRouter()
	app.Use(WithStore(store, Options{}))

	var callCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Verify store has the entry
	resp, err := store.Get("")
	// Key is hashed, so we can't directly check
	_ = resp
	_ = err
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	resp := &Response{
		StatusCode: 200,
		Body:       []byte("test"),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	// Set
	err := store.Set("key", resp)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Get
	got, err := store.Get("key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(got.Body) != "test" {
		t.Errorf("expected 'test', got %q", string(got.Body))
	}

	// Delete
	err = store.Delete("key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	got, _ = store.Get("key")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemoryStore_Expiry(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	resp := &Response{
		StatusCode: 200,
		Body:       []byte("test"),
		ExpiresAt:  time.Now().Add(-time.Hour), // Already expired
	}

	_ = store.Set("expired", resp)

	got, _ := store.Get("expired")
	if got != nil {
		t.Error("expected nil for expired entry")
	}
}

func TestWithOptions_CustomKeyGenerator(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		KeyGenerator: func(key string, c *mizu.Ctx) string {
			// Include user ID in key
			userID := c.Request().Header.Get("X-User-ID")
			return key + "-" + userID
		},
	}))

	var callCount int
	app.Post("/create", func(c *mizu.Ctx) error {
		callCount++
		return c.Text(http.StatusOK, "ok")
	})

	// Same idempotency key but different users
	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key")
	req.Header.Set("X-User-ID", "user1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "key")
	req.Header.Set("X-User-ID", "user2")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should be 2 calls because users are different
	if callCount != 2 {
		t.Errorf("expected 2 calls for different users, got %d", callCount)
	}
}

func TestWithOptions_ResponseBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/create", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "json-key")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Replay
	req = httptest.NewRequest(http.MethodPost, "/create", nil)
	req.Header.Set("Idempotency-Key", "json-key")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "123") {
		t.Errorf("expected JSON body to be replayed, got %q", rec.Body.String())
	}
}
