package adaptive

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{InitialRate: 10}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLimiter_RateLimited(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    1,
		AdjustInterval: time.Hour, // Don't adjust during test
	})

	app := mizu.NewRouter()
	app.Use(limiter.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// First request should succeed
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected first request to succeed, got %d", rec.Code)
	}

	// Immediate second request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected rate limit, got %d", rec.Code)
	}
}

func TestLimiter_CurrentRate(t *testing.T) {
	limiter := NewLimiter(Options{InitialRate: 50})

	if limiter.CurrentRate() != 50 {
		t.Errorf("expected initial rate 50, got %d", limiter.CurrentRate())
	}
}

func TestLimiter_Stats(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate: 100,
		MinRate:     10,
		MaxRate:     1000,
	})

	stats := limiter.Stats()

	if stats.CurrentRate != 100 {
		t.Errorf("expected current rate 100, got %d", stats.CurrentRate)
	}
	if stats.MinRate != 10 {
		t.Errorf("expected min rate 10, got %d", stats.MinRate)
	}
	if stats.MaxRate != 1000 {
		t.Errorf("expected max rate 1000, got %d", stats.MaxRate)
	}
}

func TestSimple(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Simple())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHighThroughput(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(HighThroughput())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Should handle many concurrent requests
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount == 0 {
		t.Error("expected some requests to succeed")
	}
}

func TestConservative(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Conservative())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRetryAfterHeader(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    0, // Immediately limited
		AdjustInterval: time.Hour,
	})

	app := mizu.NewRouter()
	app.Use(limiter.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After header, got %q", rec.Header().Get("Retry-After"))
	}
}

func TestCustomErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		InitialRate:    0,
		AdjustInterval: time.Hour,
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "custom rate limit",
			})
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Error("expected JSON response from custom handler")
	}
}
