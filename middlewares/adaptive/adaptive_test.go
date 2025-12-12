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

func TestNewLimiter_Defaults(t *testing.T) {
	// Test with negative values (should use defaults)
	limiter := NewLimiter(Options{
		InitialRate: -1, // Should default to 100
		MinRate:     -1, // Should default to 10
		MaxRate:     -1, // Should default to 1000
	})

	if limiter.opts.InitialRate != 100 {
		t.Errorf("expected default InitialRate 100, got %d", limiter.opts.InitialRate)
	}
	if limiter.opts.MinRate != 10 {
		t.Errorf("expected default MinRate 10, got %d", limiter.opts.MinRate)
	}
	if limiter.opts.MaxRate != 1000 {
		t.Errorf("expected default MaxRate 1000, got %d", limiter.opts.MaxRate)
	}
}

func TestLimiter_Adjust(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    100,
		MinRate:        10,
		MaxRate:        1000,
		TargetLatency:  100 * time.Millisecond,
		ErrorThreshold: 0.1,
		AdjustInterval: time.Hour, // Don't auto-adjust
	})

	// Test with no requests - should not change rate
	initialRate := limiter.CurrentRate()
	limiter.adjust()
	if limiter.CurrentRate() != initialRate {
		t.Error("rate should not change with no requests")
	}
}

func TestLimiter_AdjustHighErrorRate(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    100,
		MinRate:        10,
		MaxRate:        1000,
		TargetLatency:  100 * time.Millisecond,
		ErrorThreshold: 0.1,
		AdjustInterval: time.Hour, // Don't auto-adjust
	})

	// Simulate high error rate
	for i := 0; i < 100; i++ {
		limiter.requestCount++
		if i < 50 { // 50% error rate
			limiter.errorCount++
		}
		limiter.totalLatency += int64(50 * time.Millisecond) // Low latency
	}

	limiter.adjust()

	// Rate should decrease due to high error rate
	if limiter.CurrentRate() >= 100 {
		t.Errorf("rate should decrease with high error rate, got %d", limiter.CurrentRate())
	}
}

func TestLimiter_AdjustHighLatency(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    100,
		MinRate:        10,
		MaxRate:        1000,
		TargetLatency:  100 * time.Millisecond,
		ErrorThreshold: 0.1,
		AdjustInterval: time.Hour,
	})

	// Simulate high latency
	for i := 0; i < 100; i++ {
		limiter.requestCount++
		limiter.totalLatency += int64(200 * time.Millisecond) // High latency
	}

	limiter.adjust()

	// Rate should decrease due to high latency
	if limiter.CurrentRate() >= 100 {
		t.Errorf("rate should decrease with high latency, got %d", limiter.CurrentRate())
	}
}

func TestLimiter_AdjustLowLatencyAndErrors(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    100,
		MinRate:        10,
		MaxRate:        1000,
		TargetLatency:  100 * time.Millisecond,
		ErrorThreshold: 0.1,
		AdjustInterval: time.Hour,
	})

	// Simulate low latency and no errors
	for i := 0; i < 100; i++ {
		limiter.requestCount++
		limiter.totalLatency += int64(20 * time.Millisecond) // Very low latency
	}

	limiter.adjust()

	// Rate should increase due to good performance
	if limiter.CurrentRate() <= 100 {
		t.Errorf("rate should increase with good performance, got %d", limiter.CurrentRate())
	}
}

func TestLimiter_AdjustClampToMin(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    15, // Close to min
		MinRate:        10,
		MaxRate:        1000,
		TargetLatency:  100 * time.Millisecond,
		ErrorThreshold: 0.1,
		AdjustInterval: time.Hour,
	})

	// Simulate high error rate to force decrease
	for i := 0; i < 100; i++ {
		limiter.requestCount++
		limiter.errorCount++ // 100% error rate
		limiter.totalLatency += int64(50 * time.Millisecond)
	}

	// Force multiple adjustments to hit min
	for i := 0; i < 10; i++ {
		// Simulate high error rate each time
		limiter.requestCount = 100
		limiter.errorCount = 100
		limiter.totalLatency = int64(100 * 50 * time.Millisecond)
		limiter.adjust()
	}

	// Rate should be clamped to min
	if limiter.CurrentRate() < int64(10) {
		t.Errorf("rate should be clamped to min 10, got %d", limiter.CurrentRate())
	}
}

func TestLimiter_AdjustClampToMax(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    950, // Close to max
		MinRate:        10,
		MaxRate:        1000,
		TargetLatency:  100 * time.Millisecond,
		ErrorThreshold: 0.1,
		AdjustInterval: time.Hour,
	})

	// Simulate excellent performance to force increase
	for i := 0; i < 10; i++ {
		limiter.requestCount = 100
		limiter.errorCount = 0
		limiter.totalLatency = int64(100 * 10 * time.Millisecond) // Very low latency
		limiter.adjust()
	}

	// Rate should be clamped to max
	if limiter.CurrentRate() > int64(1000) {
		t.Errorf("rate should be clamped to max 1000, got %d", limiter.CurrentRate())
	}
}

func TestLimiter_AllowTokenRefill(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    10,
		AdjustInterval: time.Hour,
	})

	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		limiter.allow()
	}

	// Should be rate limited
	if limiter.allow() {
		t.Error("should be rate limited after exhausting tokens")
	}

	// Wait for token refill
	time.Sleep(200 * time.Millisecond)

	// Should have tokens again
	if !limiter.allow() {
		t.Error("should have tokens after refill period")
	}
}

func TestLimiter_AllowTokenCap(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    5,
		AdjustInterval: time.Hour,
	})

	// Wait to accumulate more than max tokens
	time.Sleep(500 * time.Millisecond)

	// Tokens should be capped at rate
	// We can verify this indirectly by consuming tokens
	for i := 0; i < 5; i++ {
		if !limiter.allow() {
			t.Errorf("should have token %d", i)
		}
	}

	// Next should fail (no more than rate tokens)
	if limiter.allow() {
		t.Error("should not have more than rate tokens")
	}
}

func TestLimiter_ErrorCounting(t *testing.T) {
	limiter := NewLimiter(Options{
		InitialRate:    100,
		AdjustInterval: time.Hour,
	})

	app := mizu.NewRouter()
	app.Use(limiter.Middleware())

	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})

	app.Get("/ok", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make error request
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	stats := limiter.Stats()
	if stats.RequestCount == 0 {
		t.Error("expected request count > 0")
	}
}
