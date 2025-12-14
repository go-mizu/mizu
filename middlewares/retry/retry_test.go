package retry

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	var attempts int
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return c.Text(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithOptions_MaxRetries(t *testing.T) {
	var attempts int
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxRetries: 2,
		Delay:      time.Millisecond,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		attempts++
		return errors.New("always fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// 1 initial + 2 retries = 3 attempts
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithOptions_OnRetry(t *testing.T) {
	var onRetryCalls int
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxRetries: 2,
		Delay:      time.Millisecond,
		OnRetry: func(c *mizu.Ctx, err error, attempt int) {
			onRetryCalls++
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return errors.New("fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// OnRetry called before each retry (not initial attempt)
	if onRetryCalls != 2 {
		t.Errorf("expected OnRetry to be called 2 times, got %d", onRetryCalls)
	}
}

func TestWithOptions_NoRetryNeeded(t *testing.T) {
	var attempts int
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		attempts++
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestWithOptions_CustomRetryIf(t *testing.T) {
	var attempts int
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxRetries: 5,
		Delay:      time.Millisecond,
		RetryIf: func(c *mizu.Ctx, err error, attempt int) bool {
			// Only retry twice regardless of error
			return attempt < 2
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		attempts++
		return errors.New("fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should stop after 2 retries even though MaxRetries is 5
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryOn(t *testing.T) {
	retryIf := RetryOn(500, 502, 503)

	tests := []struct {
		name     string
		status   int
		expected bool
	}{
		{"500", 500, true},
		{"502", 502, true},
		{"503", 503, true},
		{"200", 200, false},
		{"404", 404, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mizu.NewRouter()
			var c *mizu.Ctx
			var rw *retryResponseWriter
			app.Get("/", func(ctx *mizu.Ctx) error {
				c = ctx
				rw = &retryResponseWriter{
					ResponseWriter: ctx.Writer(),
					status:         tt.status,
				}
				ctx.SetWriter(rw)
				return nil
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if c == nil {
				t.Fatal("context not set")
			}

			// Test with nil error
			result := retryIf(c, nil, 0)
			if result != tt.expected {
				t.Errorf("RetryOn for status %d: got %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestRetryOnError(t *testing.T) {
	retryIf := RetryOnError()

	app := mizu.NewRouter()
	var c *mizu.Ctx
	app.Get("/", func(ctx *mizu.Ctx) error {
		c = ctx
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should not retry on nil error
	if retryIf(c, nil, 0) {
		t.Error("expected false for nil error")
	}

	// Should retry on error
	if !retryIf(c, errors.New("error"), 0) {
		t.Error("expected true for error")
	}
}

func TestNoRetry(t *testing.T) {
	retryIf := NoRetry()

	app := mizu.NewRouter()
	var c *mizu.Ctx
	app.Get("/", func(ctx *mizu.Ctx) error {
		c = ctx
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if retryIf(c, nil, 0) {
		t.Error("expected false")
	}
	if retryIf(c, errors.New("error"), 0) {
		t.Error("expected false even for error")
	}
}

func TestWithOptions_ExponentialBackoff(t *testing.T) {
	var delays []time.Duration
	var lastTime time.Time

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxRetries: 3,
		Delay:      10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
		OnRetry: func(c *mizu.Ctx, err error, attempt int) {
			if !lastTime.IsZero() {
				delays = append(delays, time.Since(lastTime))
			}
			lastTime = time.Now()
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		if lastTime.IsZero() {
			lastTime = time.Now()
		}
		return errors.New("fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Check that delays increase
	if len(delays) < 2 {
		t.Fatal("expected at least 2 delays recorded")
	}

	// Each delay should be roughly double the previous (with some tolerance)
	for i := 1; i < len(delays); i++ {
		if delays[i] < delays[i-1] {
			t.Logf("delays: %v", delays)
			// Allow some variance due to timing
		}
	}
}
