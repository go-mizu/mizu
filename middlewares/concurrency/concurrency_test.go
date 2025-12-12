package concurrency

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
	var maxConcurrent int64

	app.Get("/", func(c *mizu.Ctx) error {
		cur := atomic.AddInt64(&concurrent, 1)
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt64(&concurrent, -1)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()

	// Max concurrent should not exceed limit
	if maxConcurrent > 2 {
		t.Errorf("expected max concurrent <= 2, got %d", maxConcurrent)
	}
}

func TestNew_RejectsOverCapacity(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(1))

	var concurrent int64
	app.Get("/", func(c *mizu.Ctx) error {
		atomic.AddInt64(&concurrent, 1)
		defer atomic.AddInt64(&concurrent, -1)
		time.Sleep(100 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	results := make([]int, 3)

	for i := 0; i < 3; i++ {
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

	// Some requests should be rejected
	rejectedCount := 0
	for _, code := range results {
		if code == http.StatusServiceUnavailable {
			rejectedCount++
		}
	}

	if rejectedCount == 0 {
		t.Error("expected some requests to be rejected")
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Max: 0, // Immediately reject
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "at capacity",
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

func TestBlocking(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Blocking(2))

	var concurrent int64
	var maxConcurrent int64

	app.Get("/", func(c *mizu.Ctx) error {
		cur := atomic.AddInt64(&concurrent, 1)
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt64(&concurrent, -1)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			// All should eventually succeed with blocking
			if rec.Code != http.StatusOK {
				t.Errorf("expected OK with blocking, got %d", rec.Code)
			}
		}()
	}

	wg.Wait()

	// Max concurrent should not exceed limit
	if maxConcurrent > 2 {
		t.Errorf("expected max concurrent <= 2, got %d", maxConcurrent)
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

	if rec.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After header, got %q", rec.Header().Get("Retry-After"))
	}
}
