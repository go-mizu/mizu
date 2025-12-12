package circuitbreaker

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	failCount := 0
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		failCount++
		if failCount <= 5 {
			return errors.New("service error")
		}
		return c.Text(http.StatusOK, "ok")
	})

	// First 5 requests fail, circuit should open
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Next request should be blocked by open circuit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected circuit to be open, got status %d", rec.Code)
	}
}

func TestWithOptions_Threshold(t *testing.T) {
	failCount := 0
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Threshold: 3,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		failCount++
		return errors.New("error")
	})

	// 3 failures should open circuit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected circuit open after 3 failures")
	}
}

func TestWithOptions_Timeout(t *testing.T) {
	failCount := 0
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Threshold: 2,
		Timeout:   100 * time.Millisecond,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		failCount++
		if failCount <= 2 {
			return errors.New("error")
		}
		return c.Text(http.StatusOK, "ok")
	})

	// 2 failures open circuit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Circuit is open
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Error("expected circuit open")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should be half-open, allow one request
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected success after timeout, got %d", rec.Code)
	}
}

func TestWithOptions_OnStateChange(t *testing.T) {
	var transitions []string
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Threshold: 2,
		Timeout:   50 * time.Millisecond,
		OnStateChange: func(from, to State) {
			transitions = append(transitions, from.String()+"->"+to.String())
		},
	}))

	failCount := 0
	app.Get("/test", func(c *mizu.Ctx) error {
		failCount++
		if failCount <= 2 {
			return errors.New("error")
		}
		return c.Text(http.StatusOK, "ok")
	})

	// 2 failures -> open
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	if len(transitions) != 1 || transitions[0] != "closed->open" {
		t.Errorf("expected closed->open, got %v", transitions)
	}

	// Wait and trigger half-open
	time.Sleep(60 * time.Millisecond)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(transitions) < 2 {
		t.Error("expected transition to half-open")
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Threshold: 1,
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": "circuit breaker open",
			})
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return errors.New("error")
	})

	// Trigger open
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Check custom error handler
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestWithOptions_IsFailure(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Threshold: 2,
		IsFailure: func(err error) bool {
			// Only count specific errors
			return err != nil && err.Error() == "critical"
		},
	}))

	count := 0
	app.Get("/test", func(c *mizu.Ctx) error {
		count++
		if count <= 3 {
			return errors.New("non-critical")
		}
		return errors.New("critical")
	})

	// 3 non-critical errors - circuit stays closed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// 2 critical errors - circuit opens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Error("expected circuit open after critical errors")
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, tt.state.String(), tt.expected)
		}
	}
}
