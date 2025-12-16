package timeout

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(50 * time.Millisecond))

	app.Get("/slow", func(c *mizu.Ctx) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return c.Text(http.StatusOK, "completed")
		case <-c.Context().Done():
			return c.Context().Err()
		}
	})

	app.Get("/fast", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "fast")
	})

	t.Run("times out slow request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})

	t.Run("allows fast request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fast", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "fast" {
			t.Errorf("expected body 'fast', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Timeout: 50 * time.Millisecond,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusGatewayTimeout)
			_, _ = w.Write([]byte(`{"error":"timeout"}`))
		},
	}))

	app.Get("/slow", func(c *mizu.Ctx) error {
		time.Sleep(200 * time.Millisecond)
		return c.Text(http.StatusOK, "completed")
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status %d, got %d", http.StatusGatewayTimeout, rec.Code)
	}
}

func TestWithOptions_CustomMessage(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Timeout:      50 * time.Millisecond,
		ErrorMessage: "Request took too long",
	}))

	app.Get("/slow", func(c *mizu.Ctx) error {
		time.Sleep(200 * time.Millisecond)
		return c.Text(http.StatusOK, "completed")
	})

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Request took too long" {
		t.Errorf("expected 'Request took too long', got %q", rec.Body.String())
	}
}

func TestWithOptions_DefaultTimeout(t *testing.T) {
	mw := WithOptions(Options{})
	if mw == nil {
		t.Error("expected middleware to be created with defaults")
	}
}

func TestTimeout_ContextCancellation(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(50 * time.Millisecond))

	var contextCancelled int32
	app.Get("/check", func(c *mizu.Ctx) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return c.Text(http.StatusOK, "not cancelled")
		case <-c.Context().Done():
			atomic.StoreInt32(&contextCancelled, 1)
			return c.Context().Err()
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Give some time for goroutine to notice cancellation
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&contextCancelled) != 1 {
		t.Error("expected context to be cancelled")
	}
}
