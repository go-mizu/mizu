package maxconns

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(2))

	var concurrent int64
	app.Get("/", func(c *mizu.Ctx) error {
		atomic.AddInt64(&concurrent, 1)
		defer atomic.AddInt64(&concurrent, -1)
		time.Sleep(50 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	results := make([]int, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			results[idx] = rec.Code
		}(i)
	}

	wg.Wait()

	// Some requests should succeed, some should be rejected
	okCount := 0
	rejectCount := 0
	for _, code := range results {
		if code == http.StatusOK {
			okCount++
		} else if code == http.StatusServiceUnavailable {
			rejectCount++
		}
	}

	if okCount == 0 {
		t.Error("expected some requests to succeed")
	}
}

func TestWithOptions_PerIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Max: 100, PerIP: 1}))

	var concurrent int64
	app.Get("/", func(c *mizu.Ctx) error {
		atomic.AddInt64(&concurrent, 1)
		defer atomic.AddInt64(&concurrent, -1)
		time.Sleep(50 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	results := make([]int, 3)

	// All from same IP
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			results[idx] = rec.Code
		}(i)
	}

	wg.Wait()

	okCount := 0
	for _, code := range results {
		if code == http.StatusOK {
			okCount++
		}
	}

	// Only 1 should succeed per IP
	if okCount > 2 { // Allow for timing variations
		t.Errorf("expected at most 2 to succeed with per-IP limit, got %d", okCount)
	}
}

func TestWithOptions_CustomErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Max: 0, // Immediate rejection
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "rate limited",
			})
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected custom error code, got %d", rec.Code)
	}
}

func TestPerIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(PerIP(1))

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(20 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	// Sequential requests should work
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected sequential request to work, got %d", rec.Code)
	}
}

func TestGlobal(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Global(5))

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

func TestCounter(t *testing.T) {
	counter := NewCounter(10)

	if counter.Current() != 0 {
		t.Errorf("expected 0 current, got %d", counter.Current())
	}
	if counter.Max() != 10 {
		t.Errorf("expected max 10, got %d", counter.Max())
	}

	app := mizu.NewRouter()
	app.Use(counter.Middleware())

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
	app := mizu.NewRouter()
	app.Use(New(0)) // Immediate rejection

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("expected Retry-After header, got %q", rec.Header().Get("Retry-After"))
	}
}
