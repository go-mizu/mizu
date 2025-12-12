package bulkhead

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
	app.Use(New(Options{
		MaxConcurrent: 2,
		MaxWait:       1,
	}))

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

	// Some should succeed, some should be rejected
	okCount := 0
	rejectCount := 0
	for _, code := range results {
		switch code {
		case http.StatusOK:
			okCount++
		case http.StatusServiceUnavailable:
			rejectCount++
		}
	}

	if okCount == 0 {
		t.Error("expected some successful requests")
	}
	if rejectCount == 0 {
		t.Error("expected some rejected requests")
	}
}

func TestBulkhead_Stats(t *testing.T) {
	b := NewBulkhead(Options{
		Name:          "test",
		MaxConcurrent: 5,
		MaxWait:       3,
	})

	stats := b.Stats()
	if stats.Name != "test" {
		t.Errorf("expected name 'test', got %q", stats.Name)
	}
	if stats.MaxActive != 5 {
		t.Errorf("expected max active 5, got %d", stats.MaxActive)
	}
	if stats.Available != 5 {
		t.Errorf("expected 5 available, got %d", stats.Available)
	}
}

func TestNewBulkhead_ErrorHandler(t *testing.T) {
	b := NewBulkhead(Options{
		MaxConcurrent: 1, // Only 1 slot
		MaxWait:       1, // Only 1 can wait
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "bulkhead full",
			})
		},
	})

	app := mizu.NewRouter()
	app.Use(b.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(100 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	// Fill the slot and the wait queue
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}()
	}

	// Give time for requests to acquire slot and wait queue
	time.Sleep(30 * time.Millisecond)

	// This request should be rejected (slot full, wait queue full)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected custom error code, got %d", rec.Code)
	}

	wg.Wait()
}

func TestManager(t *testing.T) {
	manager := NewManager()

	b1 := manager.Get("api", 10, 5)
	b2 := manager.Get("api", 10, 5) // Should return same bulkhead

	if b1 != b2 {
		t.Error("expected same bulkhead instance")
	}

	b3 := manager.Get("other", 5, 2)
	if b1 == b3 {
		t.Error("expected different bulkhead instance")
	}
}

func TestManager_Stats(t *testing.T) {
	manager := NewManager()
	manager.Get("api", 10, 5)
	manager.Get("db", 5, 2)

	stats := manager.Stats()
	if len(stats) != 2 {
		t.Errorf("expected 2 bulkheads, got %d", len(stats))
	}
	if _, ok := stats["api"]; !ok {
		t.Error("expected 'api' bulkhead")
	}
	if _, ok := stats["db"]; !ok {
		t.Error("expected 'db' bulkhead")
	}
}

func TestForPath(t *testing.T) {
	manager := NewManager()

	app := mizu.NewRouter()
	app.Use(ForPath(manager, 2, 1))

	app.Get("/api", func(c *mizu.Ctx) error {
		time.Sleep(20 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/db", func(c *mizu.Ctx) error {
		time.Sleep(20 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	// Make requests to different paths
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()

	// Check that separate bulkheads were created
	stats := manager.Stats()
	if _, ok := stats["/api"]; !ok {
		t.Error("expected /api bulkhead")
	}
}

func TestDefaults(t *testing.T) {
	b := NewBulkhead(Options{})

	stats := b.Stats()
	if stats.MaxActive != 10 {
		t.Errorf("expected default max active 10, got %d", stats.MaxActive)
	}
	if stats.MaxWaiting != 10 {
		t.Errorf("expected default max waiting 10, got %d", stats.MaxWaiting)
	}
}
