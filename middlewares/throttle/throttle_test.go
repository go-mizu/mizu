package throttle

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

	var concurrent int32
	var maxConcurrent int32
	var mu sync.Mutex

	app.Get("/", func(c *mizu.Ctx) error {
		current := atomic.AddInt32(&concurrent, 1)
		mu.Lock()
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&concurrent, -1)
		return c.Text(http.StatusOK, "ok")
	})

	// Send concurrent requests
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

	if maxConcurrent > 2 {
		t.Errorf("expected max concurrent <= 2, got %d", maxConcurrent)
	}
}

func TestWithOptions_Backlog(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Limit:   1,
		Backlog: 1,
		Timeout: time.Second,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(100 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	var results []int
	var resultsMu sync.Mutex

	// Send 3 concurrent requests (1 processing, 1 backlog, 1 rejected)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			resultsMu.Lock()
			results = append(results, rec.Code)
			resultsMu.Unlock()
		}()
	}
	wg.Wait()

	// At least one should be service unavailable
	var unavailableCount int
	for _, code := range results {
		if code == http.StatusServiceUnavailable {
			unavailableCount++
		}
	}

	if unavailableCount == 0 {
		t.Error("expected at least one service unavailable response")
	}
}

func TestWithOptions_Timeout(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Limit:   1,
		Backlog: 10,
		Timeout: 50 * time.Millisecond,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(200 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	var timedOutCount int32

	// First request takes the slot
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}()

	// Wait for first request to start
	time.Sleep(10 * time.Millisecond)

	// Second request should timeout waiting
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code == http.StatusServiceUnavailable {
			atomic.AddInt32(&timedOutCount, 1)
		}
	}()

	wg.Wait()

	if timedOutCount == 0 {
		t.Error("expected second request to timeout")
	}
}

func TestWithOptions_OnThrottle(t *testing.T) {
	var throttled int32

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Limit:      1,
		Backlog:    0, // No backlog, immediate rejection
		BacklogSet: true,
		OnThrottle: func(c *mizu.Ctx) {
			atomic.AddInt32(&throttled, 1)
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(50 * time.Millisecond)
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup

	// First request
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}()

	time.Sleep(10 * time.Millisecond)

	// Second request should be throttled
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}()

	wg.Wait()

	if throttled == 0 {
		t.Error("expected OnThrottle to be called")
	}
}

func TestConcurrency(t *testing.T) {
	// Concurrency is an alias for New
	app := mizu.NewRouter()
	app.Use(Concurrency(5))

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
